// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen

import (
	"cmp"
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/altessa-s/go-optgen/internal/application"
	"github.com/altessa-s/go-optgen/model"

	optconfig "github.com/altessa-s/go-optgen/config"
)

const (
	defaultOutputFile = "options_gen.go"
	defaultTypeName   = "options"
	defaultOptionType = "Option"
)

// generateSettings holds the generate flags. Values come from the command
// line, then from .optgen.yaml for flags that were not set explicitly.
type generateSettings struct {
	typeName         string
	output           string
	optionType       string
	optionError      bool
	directory        string
	skipOptionType   bool
	skipDefaultFunc  bool
	skipNewFunc      bool
	dryRun           bool
	verbose          bool
	disablePlugins   string
	defaultFunc      string
	newFunc          string
	optionPrefix     string
	processAllFields bool
	noConfig         bool   // Skip loading .optgen.yaml config file
	formatter        string // Formatter command (default: "gofmt -w")
	noFormat         bool   // Disable formatting
	allowFormatters  []string
	funcPrefix       string
	invertFuncPrefix string
	stringSetter     string
	keepEmptyStrings bool
	checkFailure     string
	noDefaultMods    bool
	typeDefaults     map[string]string // Configured type defaults (config only)
	validate         bool
	withIsSet        bool
	strict           bool
	defaultsFrom     string
	sliceSetter      string
	exclude          []string
	sourceFile       string // Restrict the struct search to this file (CLI only)
}

// target is one struct to generate options for, with its resolved names.
type target struct {
	typeName, output, optionType, defaultFunc, newFunc string
}

// changedFlags reports whether a flag was set on the command line.
type changedFlags interface{ Changed(name string) bool }

// loadConfig finds .optgen.yaml for the target directory and applies the
// configuration of that package to flags not set on the command line.
func (s *generateSettings) loadConfig(flags changedFlags, out io.Writer) error {
	configPath, projectRoot := findConfigFile(s.directory)
	if configPath == "" {
		return nil
	}
	cfg, err := readConfigFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config %s: %w", configPath, err)
	}
	if s.verbose {
		_, _ = fmt.Fprintf(out, "Loaded config from: %s\n", configPath)
	}
	s.applyConfig(flags, cfg.GetPackageConfig(packagePath(projectRoot, s.directory)))
	return nil
}

// packagePath returns dir relative to projectRoot in the "./pkg" form used
// by the packages section of .optgen.yaml.
func packagePath(projectRoot, dir string) string {
	if projectRoot == "" {
		return dir
	}
	absDir, errDir := filepath.Abs(dir)
	absRoot, errRoot := filepath.Abs(projectRoot)
	if errDir != nil || errRoot != nil {
		return dir
	}
	rel, err := filepath.Rel(absRoot, absDir)
	if err != nil {
		return dir
	}
	return "./" + rel
}

// applyConfig applies config values to flags that weren't explicitly set.
func (s *generateSettings) applyConfig(flags changedFlags, cfg optconfig.Defaults) {
	defaultString(flags, "type", &s.typeName, cfg.Type)
	defaultString(flags, "output", &s.output, cfg.Output)
	defaultString(flags, "option-type", &s.optionType, cfg.OptionType)
	defaultString(flags, "default-func-name", &s.defaultFunc, cfg.DefaultFuncName)
	defaultString(flags, "new-func-name", &s.newFunc, cfg.NewFuncName)
	defaultString(flags, "option-prefix", &s.optionPrefix, cfg.OptionPrefix)
	if cfg.Formatter != nil && !flags.Changed("formatter") {
		s.formatter = *cfg.Formatter
	}
	if len(cfg.DisablePlugins) > 0 && !flags.Changed("disable-plugins") {
		s.disablePlugins = strings.Join(cfg.DisablePlugins, ",")
	}

	defaultBool(flags, "option-error", &s.optionError, cfg.OptionError)
	defaultBool(flags, "all-fields", &s.processAllFields, cfg.AllFields)
	defaultBool(flags, "skip-option-type", &s.skipOptionType, cfg.SkipOptionType)
	defaultBool(flags, "skip-default-func", &s.skipDefaultFunc, cfg.SkipDefaultFunc)
	defaultBool(flags, "skip-new-func", &s.skipNewFunc, cfg.SkipNewFunc)
	defaultBool(flags, "verbose", &s.verbose, cfg.Verbose)
	defaultBool(flags, "no-format", &s.noFormat, cfg.NoFormat)
	defaultString(flags, "func-prefix", &s.funcPrefix, cfg.FuncPrefix)
	defaultString(flags, "invert-func-prefix", &s.invertFuncPrefix, cfg.InvertFuncPrefix)
	defaultString(flags, "string-setter", &s.stringSetter, cfg.StringSetter)
	defaultString(flags, "check-failure", &s.checkFailure, cfg.CheckFailure)
	defaultBool(flags, "keep-empty-strings", &s.keepEmptyStrings, cfg.KeepEmptyStrings)
	defaultBool(flags, "no-default-modifiers", &s.noDefaultMods, cfg.NoDefaultModifiers)
	s.typeDefaults = cfg.TypeDefaults
	defaultBool(flags, "validate", &s.validate, cfg.Validate)
	defaultBool(flags, "with-isset", &s.withIsSet, cfg.WithIsSet)
	defaultBool(flags, "strict", &s.strict, cfg.Strict)
	defaultString(flags, "defaults-from", &s.defaultsFrom, cfg.DefaultsFrom)
	defaultString(flags, "slice-setter", &s.sliceSetter, cfg.SliceSetter)
	if len(cfg.Exclude) > 0 && !flags.Changed("exclude") {
		s.exclude = cfg.Exclude
	}
}

// defaultString sets *dst to a non-empty config value unless flag name was
// set on the command line.
func defaultString(flags changedFlags, name string, dst *string, value string) {
	if value != "" && !flags.Changed(name) {
		*dst = value
	}
}

// defaultBool sets *dst to true when the config enables it, unless flag name
// was set on the command line.
func defaultBool(flags changedFlags, name string, dst *bool, value bool) {
	if value && !flags.Changed(name) {
		*dst = true
	}
}

// disabledPluginNames returns the comma-separated --disable-plugins entries.
func (s *generateSettings) disabledPluginNames() []string {
	if s.disablePlugins == "" {
		return nil
	}
	return strings.Split(s.disablePlugins, ",")
}

// validateDirectory checks that the target directory exists and is a
// directory.
func (s *generateSettings) validateDirectory() error {
	info, err := os.Stat(s.directory)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", s.directory)
		}
		return fmt.Errorf("failed to access directory %s: %w", s.directory, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", s.directory)
	}
	return nil
}

// targets resolves the structs to generate. One --type keeps the configured
// names; several types derive each name from the type by repository
// convention (checkOptions -> CheckOption, check_options_gen.go), so the
// flags that name a single output must not be set explicitly.
func (s *generateSettings) targets(flags changedFlags) ([]target, error) {
	var types []string
	for name := range strings.SplitSeq(s.typeName, ",") {
		if name = strings.TrimSpace(name); name != "" {
			types = append(types, name)
		}
	}
	if len(types) == 0 {
		return nil, errors.New("type name cannot be empty")
	}
	if len(types) == 1 {
		t := target{typeName: types[0], output: cmp.Or(s.output, defaultOutputFile), optionType: s.optionType,
			defaultFunc: s.defaultFunc, newFunc: s.newFunc}
		t.defaultFunc = cmp.Or(t.defaultFunc, "default"+capitalizeFirst(t.typeName))
		t.newFunc = cmp.Or(t.newFunc, "new"+capitalizeFirst(t.typeName))
		return []target{t}, t.validate()
	}
	for _, name := range []string{"output", "option-type", "default-func-name", "new-func-name"} {
		if flags.Changed(name) {
			return nil, fmt.Errorf("--%s names a single output and cannot be used with several types", name)
		}
	}
	out := make([]target, 0, len(types))
	for _, name := range types {
		t := target{typeName: name, output: snakeCase(name) + "_gen.go", optionType: derivedOptionType(name),
			defaultFunc: "default" + capitalizeFirst(name), newFunc: "new" + capitalizeFirst(name)}
		if err := t.validate(); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// derivedOptionType names the option type of typeName: checkOptions ->
// CheckOption, options -> Option.
func derivedOptionType(typeName string) string {
	base := strings.TrimSuffix(strings.TrimSuffix(typeName, "Options"), "options")
	return capitalizeFirst(base) + "Option"
}

// snakeCase converts a Go identifier to snake case: checkOptions -> check_options.
func snakeCase(name string) string {
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// validate checks that every generated name of t is a valid Go identifier.
func (t target) validate() error {
	if t.optionType == "" {
		return errors.New("option type cannot be empty")
	}
	for _, name := range []struct{ what, value string }{
		{"type name", t.typeName},
		{"option type", t.optionType},
		{"default function name", t.defaultFunc},
		{"new function name", t.newFunc},
	} {
		if !token.IsIdentifier(name.value) {
			return fmt.Errorf("invalid %s: %q is not a valid Go identifier", name.what, name.value)
		}
	}
	if t.defaultFunc == t.newFunc {
		return fmt.Errorf("default and new function names must differ (both are %q)", t.defaultFunc)
	}
	return nil
}

// validateNames checks the settings shared by every target.
func (s *generateSettings) validateNames() error {
	if s.optionPrefix != "" && !token.IsIdentifier(s.optionPrefix) {
		return fmt.Errorf("invalid option prefix: %q is not a valid Go identifier", s.optionPrefix)
	}
	for _, name := range []struct{ what, value string }{{"function prefix", s.funcPrefix}, {"invert function prefix", s.invertFuncPrefix}} {
		if name.value != "" && !token.IsIdentifier(name.value+"X") {
			return fmt.Errorf("invalid %s: %q cannot start a Go identifier", name.what, name.value)
		}
	}
	for _, pattern := range s.exclude {
		if _, err := path.Match(pattern, ""); err != nil {
			return fmt.Errorf("invalid --exclude pattern %q: %w", pattern, err)
		}
	}
	for _, name := range s.allowFormatters {
		if strings.ContainsAny(name, `/\`) || strings.TrimSpace(name) == "" {
			return fmt.Errorf("invalid --allow-formatter %q: want a command name resolved from PATH", name)
		}
	}
	return nil
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// defaults splits --defaults-from into its source and name: tag (default),
// var[=name], func[=name] or none. [model.RenderSettings.Verify] validates them;
// var and func sources default to default<Type>Value and getDefault<Type>.
func (s *generateSettings) defaults() (model.DefaultsSource, string) {
	kind, name, _ := strings.Cut(s.defaultsFrom, "=")
	return model.DefaultsSource(kind), name
}

// request builds the use-case request for t from the resolved settings.
func (s *generateSettings) request(t target) application.Request {
	source, name := s.defaults()
	switch {
	case source == model.DefaultsFromVar && name == "":
		name = "default" + capitalizeFirst(t.typeName) + "Value"
	case source == model.DefaultsFromFunc && name == "":
		name = "getDefault" + capitalizeFirst(t.typeName)
	}
	formatter := s.formatter
	if s.noFormat {
		formatter = ""
	}
	return application.Request{
		Directory: s.directory,
		Output:    t.output,
		AllFields: s.processAllFields,
		DryRun:    s.dryRun,
		Formatter: formatter,
		Settings: model.RenderSettings{
			TypeName:            t.typeName,
			OptionType:          t.optionType,
			GenerateOptionType:  !s.skipOptionType,
			GenerateDefaultFunc: !s.skipDefaultFunc,
			GenerateNewFunc:     !s.skipNewFunc,
			OptionReturnsError:  s.optionError,
			DefaultFuncName:     t.defaultFunc,
			NewFuncName:         t.newFunc,
			FuncPrefix:          s.funcPrefix,
			InvertFuncPrefix:    s.invertFuncPrefix,
			StringSetter:        model.StringSetterMode(s.stringSetter),
			KeepEmptyStrings:    s.keepEmptyStrings,
			CheckFailure:        model.CheckFailureMode(s.checkFailure),
			SliceSetter:         model.SliceSetterMode(s.sliceSetter),
			DefaultsFrom:        source,
			DefaultsName:        name,
			Validate:            s.validate,
			WithIsSet:           s.withIsSet,
		},
	}
}
