// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/altessa-s/go-optgen/plugin"
)

// NewListPlugins returns a cobra command that prints all registered plugins,
// type default providers, modifier plugins, and optcheck keys. The builtin
// plugins are registered explicitly for each command execution.
func NewListPlugins(opts ...Option) *cobra.Command {
	o := newOptions(opts...)
	var showAll bool
	cmd := &cobra.Command{
		Use:   "list-plugins",
		Short: "List known plugin and provider type names",
		Long: strings.TrimSpace(`
List registered field plugins and type default providers by type name.
These names can be used with --disable-plugins.
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			registry, err := newSessionRegistry(cmd.ErrOrStderr(), o, nil, nil)
			if err != nil {
				return err
			}
			printPlugins(cmd.OutOrStdout(), registry.Descriptors(), showAll)
			return nil
		},
	}

	cmd.Flags().BoolVar(&showAll, "show-empty", false, "Show headers even when no plugins/providers are registered")

	return cmd
}

// printPlugins writes descriptors grouped by plugin kind.
func printPlugins(w io.Writer, descriptors []plugin.Descriptor, showAll bool) {
	byKind := make(map[plugin.Kind][]plugin.Descriptor)
	for _, d := range descriptors {
		byKind[d.Kind] = append(byKind[d.Kind], d)
	}
	if len(byKind[plugin.KindField]) == 0 && len(byKind[plugin.KindTypeDefault]) == 0 && !showAll {
		_, _ = fmt.Fprintln(w, "No plugins/providers registered.")
		return
	}

	_, _ = fmt.Fprintln(w, "Field plugins:")
	for _, d := range byKind[plugin.KindField] {
		_, _ = fmt.Fprintf(w, "  - %s (priority=%d)\n", d.Name, d.Meta.Priority)
	}
	_, _ = fmt.Fprintln(w, "Type default providers:")
	for _, d := range byKind[plugin.KindTypeDefault] {
		_, _ = fmt.Fprintf(w, "  - %s\n", d.Name)
	}
	_, _ = fmt.Fprintln(w, "Transform plugins:")
	for _, d := range byKind[plugin.KindTransform] {
		_, _ = fmt.Fprintf(w, "  - %s (key=%q)\n", d.Name, d.Key)
	}
	_, _ = fmt.Fprintln(w, "Modifier plugins:")
	for _, d := range byKind[plugin.KindModifier] {
		_, _ = fmt.Fprintf(w, "  - %s (key=%q, phase=%s)\n", d.Name, d.Key, d.Phase)
	}
	_, _ = fmt.Fprintln(w, "Optcheck plugins:")
	for _, d := range byKind[plugin.KindOptCheck] {
		aliases := ""
		if len(d.Aliases) > 0 {
			aliases = ", aliases=" + strings.Join(d.Aliases, ",")
		}
		_, _ = fmt.Fprintf(w, "  - %s (key=%q%s)\n", d.Name, d.Key, aliases)
	}
}
