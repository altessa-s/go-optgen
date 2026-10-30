// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDependencyDirection(t *testing.T) {
	t.Parallel()
	const module = "github.com/altessa-s/go-optgen/"
	layers := map[string][]string{
		"model":                {},
		"plugin":               {"model"},
		"internal/application": {"model"},
		"internal/parser":      {"model", "internal/naming"},
		"internal/semantic":    {"model", "plugin"},
		"internal/generator":   {"model", "plugin"},
		"internal/source":      {"model", "internal/parser"},
		"internal/output":      {},
		"internal/naming":      {},
		"internal/plugin/builtin": {"model", "plugin",
			"internal/plugin/builtin/check", "internal/plugin/builtin/defaults", "internal/plugin/builtin/fields",
			"internal/plugin/builtin/fieldtmpl", "internal/plugin/builtin/guard", "internal/plugin/builtin/postprocess",
			"internal/plugin/builtin/transform"},
	}
	// Adapters may use the libraries that implement their port.
	externals := map[string][]string{"internal/source": {"golang.org/x/tools/go/packages"}}
	for layer, allowed := range layers {
		t.Run(layer, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join("..", layer)
			err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if d.Name() == "testdata" {
						return filepath.SkipDir
					}
					return nil
				}
				if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return nil
				}
				file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
				require.NoError(t, err)
				for _, imp := range file.Imports {
					name, err := strconv.Unquote(imp.Path.Value)
					require.NoError(t, err)
					if strings.HasPrefix(name, module) {
						require.Contains(t, allowed, strings.TrimPrefix(name, module), "forbidden outward dependency in %s", path)
					} else if !slices.Contains(externals[layer], name) {
						first, _, _ := strings.Cut(name, "/")
						require.NotContains(t, first, ".", "framework dependency in inner layer %s", path)
					}
				}
				return nil
			})
			require.NoError(t, err)
		})
	}
}
