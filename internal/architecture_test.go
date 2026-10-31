// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package internal_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const module = "github.com/altessa-s/go-optgen"

func TestDependencyDirection(t *testing.T) {
	t.Parallel()
	const builtin = "internal/plugin/builtin"
	// Each entry lists the module packages its directory may import; "" is the
	// root package. Subdirectories are separate entries, so every package that
	// is not listed here is a test failure.
	layers := map[string][]string{
		"model":                  {},
		"plugin":                 {"model"},
		"config":                 {},
		"internal/application":   {"model"},
		"internal/parser":        {"model", "internal/naming"},
		"internal/semantic":      {"model", "plugin"},
		"internal/generator":     {"model", "plugin"},
		"internal/source":        {"model", "internal/parser"},
		"internal/output":        {},
		"internal/naming":        {},
		"internal/rootcmd":       {},
		"cmd/optgen":             {""},
		builtin:                  {"plugin", builtin + "/check", builtin + "/defaults", builtin + "/fields", builtin + "/guard", builtin + "/postprocess", builtin + "/transform"},
		builtin + "/check":       {"model", "plugin"},
		builtin + "/defaults":    {"model", "plugin"},
		builtin + "/fields":      {"model", "plugin", builtin + "/fieldtmpl"},
		builtin + "/fieldtmpl":   {"model", "plugin"},
		builtin + "/guard":       {"model", "plugin"},
		builtin + "/postprocess": {"model", "plugin"},
		builtin + "/transform":   {"model", "plugin"},
	}
	// Adapters may use the libraries that implement their port.
	externals := map[string][]string{
		"internal/source":  {"golang.org/x/tools/go/packages"},
		"internal/rootcmd": {"github.com/spf13/cobra", "github.com/MakeNowJust/heredoc"},
		"config":           {"gopkg.in/yaml.v3"},
	}
	t.Run("every package is listed", func(t *testing.T) {
		t.Parallel()
		for _, dir := range []string{"internal", "cmd"} {
			err := filepath.WalkDir(filepath.Join("..", dir), func(path string, d fs.DirEntry, err error) error {
				if err != nil || !d.IsDir() {
					return err
				}
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				rel, err := filepath.Rel("..", path)
				require.NoError(t, err)
				if len(sourceFiles(t, path)) > 0 && rel != "internal" {
					require.Contains(t, layers, filepath.ToSlash(rel), "package %s has no dependency rule", rel)
				}
				return nil
			})
			require.NoError(t, err)
		}
	})
	for layer, allowed := range layers {
		t.Run(layer, func(t *testing.T) {
			t.Parallel()
			for _, path := range sourceFiles(t, filepath.Join("..", layer)) {
				file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
				require.NoError(t, err)
				for _, imp := range file.Imports {
					name, err := strconv.Unquote(imp.Path.Value)
					require.NoError(t, err)
					if after, ok := strings.CutPrefix(name, module); ok && (after == "" || after[0] == '/') {
						require.Contains(t, allowed, strings.TrimPrefix(after, "/"), "forbidden outward dependency in %s", path)
					} else if !slices.Contains(externals[layer], name) {
						first, _, _ := strings.Cut(name, "/")
						require.NotContains(t, first, ".", "framework dependency in inner layer %s", path)
					}
				}
			}
		})
	}
}

// TestNoInitOrDynamicLoading pins two AGENTS.md rules repo-wide: no init()
// registration and no loading of shared objects through the standard library
// plugin package.
func TestNoInitOrDynamicLoading(t *testing.T) {
	t.Parallel()
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// devtools is a separate module that only pins tools.
			if path != ".." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "devtools") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		require.NoError(t, err)
		for _, imp := range file.Imports {
			require.NotEqual(t, `"plugin"`, imp.Path.Value, "dynamic plugin loading in %s", path)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			require.False(t, ok && fn.Recv == nil && fn.Name.Name == "init", "init function in %s", path)
		}
		return nil
	})
	require.NoError(t, err)
}

// sourceFiles lists the non-test Go files directly in dir.
func sourceFiles(tb testing.TB, dir string) []string {
	tb.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(tb, err)
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files
}
