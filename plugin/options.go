// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin

//go:generate optgen generate --type checkOptions,transformOptions,registryOptions --no-config --all-fields --no-format

type checkOptions struct {
	requiresValue bool     `optval:"param"`
	aliases       []string `optval:"notrim,nodup,allownull"`
}

type transformOptions struct {
	applicableTypes []string `optval:"notrim,nodup,allownull"`
	priority        int      `optgen:"default=DefaultPriority"`
	defaultForTypes []string `optval:"notrim,nodup,allownull"`
	disabledBy      string   `optval:"notrim"`
}

type registryOptions struct {
	// warningSink receives lookup diagnostics, such as a plugin ignored for an
	// inapplicable type. It is called without registry locks held.
	warningSink func(message string)
}
