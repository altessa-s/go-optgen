// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package myplugins implements two optgen plugins: the optval:"collapse"
// transform and the optcheck:"hostport" check.
//
// Plugins are compiled into a program that wraps the optgen CLI; register
// them with optgen.WithPlugins:
//
//	optgen.Run(os.Args[1:], optgen.WithPlugins(myplugins.Register))
//
// # Transforms
//
// [Collapse] embeds plugin.TransformBase for its key and applicable types
// and only implements Apply, which wraps the value expression.
//
// # Checks
//
// [HostPort] embeds plugin.CheckBase. Validate rejects unsupported field
// types during generation; Generate emits the check and reports a failure
// the way the generation settings ask for.
package myplugins
