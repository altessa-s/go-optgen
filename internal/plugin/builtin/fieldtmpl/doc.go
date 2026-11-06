// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package fieldtmpl provides templates and option data shared by the built-in
// field plugins.
//
// # Usage
//
//	var setter = fieldtmpl.Must("setter", `...{{template "optionReturn" .}}`)
//
//	func (p *Plugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
//		data := fieldtmpl.NewData(ctx, field, values, "v")
//		return fieldtmpl.Execute(setter, &data)
//	}
package fieldtmpl
