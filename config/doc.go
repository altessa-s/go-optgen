// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package config decodes and merges .optgen.yaml configuration documents.
//
// [Parse] decodes the YAML into a [File], and [File.GetPackageConfig] merges
// global [Defaults] with any [PackageConfig] overrides keyed by relative path.
// The package performs no I/O: the optgen command finds the file by walking
// from the target directory to the first .optgen.yaml or project boundary
// (go.mod / .git) and passes its contents to [Parse].
//
// CLI flags always take precedence over config-file values.
//
// # Configuration format
//
//	defaults:
//	  type: options
//	  formatter: "goimports -w"
//	packages:
//	  ./internal/auth:
//	    type: authOptions
//	    option-prefix: Auth
package config
