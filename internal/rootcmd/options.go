// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package rootcmd

import (
	"io"

	"github.com/spf13/cobra"

	// os is imported for the parser: the os.Stdout and os.Stderr defaults
	// below resolve their import from this file.
	_ "os"
)

//go:generate optgen generate --no-config --all-fields --no-format

type options struct {
	long        string           `optval:"notrim"`
	example     string           `optval:"notrim"`
	subcommands []*cobra.Command `optval:"nodup,allownull"`
	out         io.Writer        `optgen:"default=os.Stdout,allownull"`
	err         io.Writer        `optgen:"default=os.Stderr,allownull"`
}
