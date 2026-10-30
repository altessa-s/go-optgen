// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package optgen implements the optgen code generator CLI as a library: the
// optgen binary is a thin main around [Run], and programs that register their
// own plugins call [Run] with [WithPlugins].
//
// The package exposes three entry points:
//
//   - [New] returns a ready-to-execute [cobra.Command] tree (generate, list-plugins, and --version).
//   - [Run] is a convenience wrapper that accepts arguments and executes the root command.
//   - [NewGenerate] and [NewListPlugins] return individual subcommands.
//
// Configuration is loaded from .optgen.yaml (see [config.File]) unless --no-config is set.
// CLI flags always take precedence over config-file values.
//
// # Usage
//
//	cmd := optgen.New()
//	err := cmd.ExecuteContext(ctx)
package optgen
