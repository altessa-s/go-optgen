// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package plugin

// BuildEarlyReturn generates an if-block that returns early from the option function
// when condition is true. If ctx.OptionReturnsError is set the block returns nil
// (no error); otherwise it uses a bare return. The returned slice contains the opening
// "if", the return statement, and the closing brace -- without leading indentation.
func BuildEarlyReturn(ctx GenerationContext, condition string) []string {
	return []string{
		"if " + condition + " {",
		"  " + earlyReturn(ctx),
		"}",
	}
}

// BuildNilEarlyReturn generates code that returns early from the option
// function when valueVar holds nil, including an interface that wraps a nil
// pointer, map, slice, channel or function. It compiles for any type, so it
// suits fields whose type cannot be resolved syntactically.
func BuildNilEarlyReturn(ctx GenerationContext, valueVar string) []string {
	return BuildNilCheck(ctx, valueVar, earlyReturn(ctx))
}

// BuildNilCheck generates code that runs body when valueVar holds nil,
// including an interface that wraps a nil pointer, map, slice, channel or
// function. It imports reflect and compiles for any type.
func BuildNilCheck(ctx GenerationContext, valueVar string, body ...string) []string {
	ctx.AddImport("reflect")
	lines := make([]string, 0, 2*len(body)+6)
	lines = append(lines, "switch rv := reflect.ValueOf("+valueVar+"); rv.Kind() {", "case reflect.Invalid:")
	for _, line := range body {
		lines = append(lines, "  "+line)
	}
	lines = append(lines,
		"case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:",
		"  if rv.IsNil() {",
	)
	for _, line := range body {
		lines = append(lines, "    "+line)
	}
	return append(lines, "  }", "}")
}

func earlyReturn(ctx GenerationContext) string {
	if ctx.OptionReturnsError {
		return "return nil"
	}
	return "return"
}
