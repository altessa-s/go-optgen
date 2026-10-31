// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optgen

import (
	"os"
	"path/filepath"

	optconfig "github.com/altessa-s/go-optgen/config"
)

// readConfigFile reads and parses the .optgen.yaml file at path.
func readConfigFile(path string) (*optconfig.File, error) {
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- Config file path comes from parent-directory discovery.
	if err != nil {
		return nil, err
	}
	return optconfig.Parse(data)
}

// findConfigFile searches for .optgen.yaml starting from startDir and walking
// parent directories up to the project root (identified by go.mod or .git).
//
// It returns (configPath, projectRoot). If no config file is found but a
// project root exists, configPath is empty and projectRoot is set. If
// neither is found (e.g. filesystem root reached), both are empty.
func findConfigFile(startDir string) (configPath, projectRoot string) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", ""
	}
	for {
		configPath := filepath.Join(dir, optconfig.ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, dir
		}
		if isProjectRoot(dir) {
			return "", dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}
}

// isProjectRoot reports whether dir contains go.mod or .git.
func isProjectRoot(dir string) bool {
	for _, marker := range []string{"go.mod", ".git"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}
