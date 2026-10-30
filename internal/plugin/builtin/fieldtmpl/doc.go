// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

// Package fieldtmpl provides templates and option data shared by the built-in
// field plugins.
//
// # Usage
//
//	var setter = fieldtmpl.Must("setter", `...{{template "optionReturn" .}}`)
//
//	func (p *Plugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
//		return fieldtmpl.Execute(setter, fieldtmpl.NewData(ctx, field, values, "v"))
//	}
package fieldtmpl
