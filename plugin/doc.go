// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

// Package plugin defines extension contracts and per-session plugin execution.
//
// # Registration
//
// Application assembly creates a registry and registers fresh implementations:
//
//	r := plugin.NewRegistry(plugin.WithWarningSink(warn))
//	if err := r.Register(&MyFieldPlugin{}); err != nil {
//		return err
//	}
//
// Register rejects values that implement no contract or several, and keys or
// names already owned by another plugin. No init-time registration or
// process-wide catalog is used.
//
// # Pipeline
//
// Executor.BuildPipeline runs all phases, propagates OutputVar, and honors
// Final in every phase. Conditional runtime guards must leave Final false.
// Registry configuration precedes generation; plugin implementations must be
// safe for concurrent reads when a registry is shared.
package plugin
