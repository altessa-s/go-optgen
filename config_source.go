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

// findConfigFile searches upward from startDir for the nearest .optgen.yaml.
// The search includes and stops at the nearest .git directory or file, crossing
// nested go.mod boundaries. Without a .git ancestor, it stops at the nearest
// go.mod instead, or at the filesystem root if neither marker exists.
//
// It returns (configPath, projectRoot). When a config is found, projectRoot is
// its directory. Otherwise, projectRoot is the nearest go.mod directory within
// the repository, falling back to the .git directory or empty if neither exists.
func findConfigFile(startDir string) (configPath, projectRoot string) {
	start, err := filepath.Abs(startDir)
	if err != nil {
		return "", ""
	}

	// Locate the repository boundary before deciding whether to cross go.mod.
	boundary := ""
	for dir := start; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && projectRoot == "" {
			projectRoot = dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			boundary = dir
			if projectRoot == "" {
				projectRoot = dir
			}
			break
		}
		if filepath.Dir(dir) == dir {
			boundary = projectRoot
			break
		}
	}

	for dir := start; ; dir = filepath.Dir(dir) {
		configPath := filepath.Join(dir, optconfig.ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, dir
		}
		if dir == boundary || filepath.Dir(dir) == dir {
			return "", projectRoot
		}
	}
}
