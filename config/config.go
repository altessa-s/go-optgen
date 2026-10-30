// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the name of the optgen configuration file.
	ConfigFileName = ".optgen.yaml"
)

// ErrPluginsUnsupported reports a config that lists external plugins, which
// optgen no longer loads.
var ErrPluginsUnsupported = errors.New("defaults.plugins is not supported: external plugins were removed")

// File represents the optgen configuration file structure.
type File struct {
	// Defaults contains global default settings for all generations.
	Defaults Defaults `yaml:"defaults,omitempty"`

	// Packages contains per-package overrides keyed by relative path.
	// Example: "./pkg/config" or "internal/options"
	Packages map[string]PackageConfig `yaml:"packages,omitempty"`
}

// Defaults contains global default settings that apply to all generations.
type Defaults struct {
	// Type is the default struct type name to look for.
	Type string `yaml:"type,omitempty"`

	// Output is the default output file name.
	Output string `yaml:"output,omitempty"`

	// OptionType is the name of the Option type.
	OptionType string `yaml:"option-type,omitempty"`

	// OptionError generates Option as func(*T) error.
	OptionError bool `yaml:"option-error,omitempty"`

	// AllFields processes all struct fields, not just tagged ones.
	AllFields bool `yaml:"all-fields,omitempty"`

	// SkipOptionType skips generating the Option type definition.
	SkipOptionType bool `yaml:"skip-option-type,omitempty"`

	// SkipDefaultFunc skips generating the defaultOptions function.
	SkipDefaultFunc bool `yaml:"skip-default-func,omitempty"`

	// SkipNewFunc skips generating the newOptions function.
	SkipNewFunc bool `yaml:"skip-new-func,omitempty"`

	// DefaultFuncName is the custom name for the default function.
	DefaultFuncName string `yaml:"default-func-name,omitempty"`

	// NewFuncName is the custom name for the new function.
	NewFuncName string `yaml:"new-func-name,omitempty"`

	// OptionPrefix is the prefix for generated option function names.
	OptionPrefix string `yaml:"option-prefix,omitempty"`

	// DisablePlugins is a list of plugin names to disable.
	DisablePlugins []string `yaml:"disable-plugins,omitempty"`

	// Verbose enables verbose output.
	Verbose bool `yaml:"verbose,omitempty"`

	// Formatter is the command to format generated files.
	// Nil keeps the default "gofmt -w"; an empty string disables formatting.
	// Examples: "gofmt -w", "goimports -w", "gofumpt -w"
	Formatter *string `yaml:"formatter,omitempty"`

	// NoFormat disables formatting of generated files.
	NoFormat bool `yaml:"no-format,omitempty"`

	// KeepEmptyStrings assigns empty strings instead of skipping them.
	KeepEmptyStrings bool `yaml:"keep-empty-strings,omitempty"`

	// NoDefaultModifiers disables the modifiers plugins apply by default,
	// such as trimspaces for strings.
	NoDefaultModifiers bool `yaml:"no-default-modifiers,omitempty"`

	// FuncPrefix prefixes option function names; empty means "With".
	FuncPrefix string `yaml:"func-prefix,omitempty"`

	// InvertFuncPrefix prefixes inverted bool flag options; empty means "Without".
	InvertFuncPrefix string `yaml:"invert-func-prefix,omitempty"`

	// StringSetter selects the string option signature: "generic" or "plain".
	StringSetter string `yaml:"string-setter,omitempty"`

	// CheckFailure selects how options without an error result report a failed
	// check: "panic" or "skip".
	CheckFailure string `yaml:"check-failure,omitempty"`

	// TypeDefaults maps a field type, as written in the source, to its default
	// expression, such as "*slog.Logger": "slog.Default()".
	TypeDefaults map[string]string `yaml:"type-defaults,omitempty"`
}

// PackageConfig contains per-package configuration overrides.
// All fields are optional; unset fields inherit from Defaults.
type PackageConfig struct {
	// Type overrides the struct type name for this package.
	Type string `yaml:"type,omitempty"`

	// Output overrides the output file name for this package.
	Output string `yaml:"output,omitempty"`

	// OptionType overrides the Option type name.
	OptionType string `yaml:"option-type,omitempty"`

	// OptionError generates Option as func(*T) error.
	OptionError *bool `yaml:"option-error,omitempty"`

	// AllFields processes all struct fields.
	AllFields *bool `yaml:"all-fields,omitempty"`

	// SkipOptionType skips generating the Option type definition.
	SkipOptionType *bool `yaml:"skip-option-type,omitempty"`

	// SkipDefaultFunc skips generating the defaultOptions function.
	SkipDefaultFunc *bool `yaml:"skip-default-func,omitempty"`

	// SkipNewFunc skips generating the newOptions function.
	SkipNewFunc *bool `yaml:"skip-new-func,omitempty"`

	// DefaultFuncName is the custom name for the default function.
	DefaultFuncName string `yaml:"default-func-name,omitempty"`

	// NewFuncName is the custom name for the new function.
	NewFuncName string `yaml:"new-func-name,omitempty"`

	// OptionPrefix is the prefix for generated option function names.
	OptionPrefix string `yaml:"option-prefix,omitempty"`

	// DisablePlugins is a list of plugin names to disable.
	DisablePlugins []string `yaml:"disable-plugins,omitempty"`

	// Formatter overrides the formatter command for this package.
	Formatter *string `yaml:"formatter,omitempty"`

	// NoFormat disables formatting for this package.
	NoFormat *bool `yaml:"no-format,omitempty"`

	// KeepEmptyStrings overrides keep-empty-strings for this package.
	KeepEmptyStrings *bool `yaml:"keep-empty-strings,omitempty"`

	// NoDefaultModifiers overrides no-default-modifiers for this package.
	NoDefaultModifiers *bool `yaml:"no-default-modifiers,omitempty"`

	// FuncPrefix prefixes option function names; empty means "With".
	FuncPrefix string `yaml:"func-prefix,omitempty"`

	// InvertFuncPrefix prefixes inverted bool flag options; empty means "Without".
	InvertFuncPrefix string `yaml:"invert-func-prefix,omitempty"`

	// StringSetter selects the string option signature: "generic" or "plain".
	StringSetter string `yaml:"string-setter,omitempty"`

	// CheckFailure selects how options without an error result report a failed
	// check: "panic" or "skip".
	CheckFailure string `yaml:"check-failure,omitempty"`

	// TypeDefaults adds or overrides type defaults for this package.
	TypeDefaults map[string]string `yaml:"type-defaults,omitempty"`
}

// Load reads and parses an .optgen.yaml file at path.
// Returns the parsed [File] or an error if the file cannot be read or contains
// invalid YAML. A defaults.plugins list is rejected with [ErrPluginsUnsupported]:
// external plugins are not supported.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- Config file path comes from CLI flag or parent-directory discovery.
	if err != nil {
		return nil, err
	}

	var cfg File
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	var removed struct {
		Defaults struct {
			Plugins []string `yaml:"plugins"`
		} `yaml:"defaults"`
	}
	if err := yaml.Unmarshal(data, &removed); err != nil {
		return nil, err
	}
	if len(removed.Defaults.Plugins) > 0 {
		return nil, fmt.Errorf("%s: %w", path, ErrPluginsUnsupported)
	}
	return &cfg, nil
}

// FindConfigFile searches for .optgen.yaml starting from startDir and walking
// parent directories up to the project root (identified by go.mod or .git).
//
// It returns (configPath, projectRoot). If no config file is found but a
// project root exists, configPath is empty and projectRoot is set. If
// neither is found (e.g. filesystem root reached), both are empty.
func FindConfigFile(startDir string) (configPath, projectRoot string) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", ""
	}

	for {
		// Check for config file in current directory
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, dir
		}

		// Check if this is a project root (has go.mod or .git)
		if isProjectRoot(dir) {
			// File not found, but we found project root
			return "", dir
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", ""
		}
		dir = parent
	}
}

// isProjectRoot checks if the directory is a project root.
func isProjectRoot(dir string) bool {
	// Check for go.mod
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	// Check for .git
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return true
	}
	return false
}

// GetPackageConfig returns [Defaults] for pkgPath by copying the global defaults
// and merging any matching [PackageConfig] entry on top. The pkgPath is
// cleaned with [filepath.Clean] before lookup, so "./pkg" and "pkg" both match.
func (c *File) GetPackageConfig(pkgPath string) Defaults {
	result := c.Defaults

	// Normalize the package path for lookup
	pkgPath = filepath.Clean(pkgPath)

	// Try to find package-specific config
	if pkgCfg, ok := c.Packages[pkgPath]; ok {
		mergePackageConfig(&result, pkgCfg)
	}

	return result
}

// mergePackageConfig merges package-specific config into defaults.
func mergePackageConfig(defaults *Defaults, pkg PackageConfig) {
	if pkg.Type != "" {
		defaults.Type = pkg.Type
	}
	if pkg.Output != "" {
		defaults.Output = pkg.Output
	}
	if pkg.OptionType != "" {
		defaults.OptionType = pkg.OptionType
	}
	if pkg.OptionError != nil {
		defaults.OptionError = *pkg.OptionError
	}
	if pkg.AllFields != nil {
		defaults.AllFields = *pkg.AllFields
	}
	if pkg.SkipOptionType != nil {
		defaults.SkipOptionType = *pkg.SkipOptionType
	}
	if pkg.SkipDefaultFunc != nil {
		defaults.SkipDefaultFunc = *pkg.SkipDefaultFunc
	}
	if pkg.SkipNewFunc != nil {
		defaults.SkipNewFunc = *pkg.SkipNewFunc
	}
	if pkg.DefaultFuncName != "" {
		defaults.DefaultFuncName = pkg.DefaultFuncName
	}
	if pkg.NewFuncName != "" {
		defaults.NewFuncName = pkg.NewFuncName
	}
	if pkg.OptionPrefix != "" {
		defaults.OptionPrefix = pkg.OptionPrefix
	}
	if len(pkg.DisablePlugins) > 0 {
		defaults.DisablePlugins = pkg.DisablePlugins
	}
	if pkg.Formatter != nil {
		defaults.Formatter = pkg.Formatter
	}
	if pkg.NoFormat != nil {
		defaults.NoFormat = *pkg.NoFormat
	}
	if pkg.KeepEmptyStrings != nil {
		defaults.KeepEmptyStrings = *pkg.KeepEmptyStrings
	}
	if pkg.NoDefaultModifiers != nil {
		defaults.NoDefaultModifiers = *pkg.NoDefaultModifiers
	}
	for dst, src := range map[*string]string{
		&defaults.FuncPrefix:       pkg.FuncPrefix,
		&defaults.InvertFuncPrefix: pkg.InvertFuncPrefix,
		&defaults.StringSetter:     pkg.StringSetter,
		&defaults.CheckFailure:     pkg.CheckFailure,
	} {
		if src != "" {
			*dst = src
		}
	}
	if len(pkg.TypeDefaults) > 0 {
		merged := maps.Clone(defaults.TypeDefaults)
		if merged == nil {
			merged = make(map[string]string, len(pkg.TypeDefaults))
		}
		maps.Copy(merged, pkg.TypeDefaults)
		defaults.TypeDefaults = merged
	}
}
