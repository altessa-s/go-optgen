// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package rootcmd_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-optgen/internal/rootcmd"
)

func TestRun_PanicInSubcommand_ReturnsErrorAndWritesToErr(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var errOut bytes.Buffer

	boom := &cobra.Command{
		Use: "boom",
		RunE: func(_ *cobra.Command, _ []string) error {
			panic("boom")
		},
	}

	err := rootcmd.Run("testtool", "test", []string{"boom"}, rootcmd.WithSubcommands(boom), rootcmd.WithOut(&out), rootcmd.WithErr(&errOut))

	require.Error(t, err)
	require.NotEmpty(t, err.Error())
	require.True(t, bytes.Contains(errOut.Bytes(), []byte("Panic recovered")), "expected panic output on stderr; got: %s", errOut.String())
}

func TestNew_UsesProvidedWriters(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var errOut bytes.Buffer

	cmd := rootcmd.New("testtool", "test", rootcmd.WithOut(&out), rootcmd.WithErr(&errOut), rootcmd.WithSubcommands(&cobra.Command{
		Use: "hello", RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "hello")
			return nil
		},
	}))

	cmd.SetArgs([]string{"hello"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	require.NotEmpty(t, out.String(), "expected stdout to be written")
	require.Empty(t, errOut.String(), "expected stderr empty")
}

func TestNew_HelpAndVersion(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	cmd := rootcmd.New("testtool", "short text", rootcmd.WithOut(&out), rootcmd.WithLong("long text"), rootcmd.WithExample("testtool run"))
	cmd.SetArgs([]string{"--help"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	require.Contains(t, out.String(), "long text")
	require.Equal(t, "testtool run", cmd.Example)

	out.Reset()
	cmd = rootcmd.New("testtool", "short text", rootcmd.WithOut(&out))
	cmd.SetArgs([]string{"--version"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	require.Contains(t, out.String(), "testtool version "+rootcmd.Version())
}

func TestNew_RunsPreAndPostHooks(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	cmd := rootcmd.New("testtool", "test", rootcmd.WithOut(&out), rootcmd.WithSubcommands(&cobra.Command{
		Use: "ok", Run: func(cmd *cobra.Command, _ []string) { _, _ = fmt.Fprint(cmd.OutOrStdout(), "ran") },
	}))
	cmd.SetArgs([]string{"ok"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	require.Equal(t, "ran", out.String())
}

func TestVersionDefaultsToBuildInfo(t *testing.T) {
	t.Parallel()
	require.NotEmpty(t, rootcmd.Version())
}
