// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

// Package rootcmd constructs CLI roots with generated options and panic recovery.
//
// # Usage
//
//	cmd := rootcmd.New("optgen", "Generate options", rootcmd.WithSubcommands(generate))
//
// Run wraps command execution with panic recovery. New returns an ordinary
// Cobra command; direct Execute calls retain Cobra's panic behavior.
package rootcmd
