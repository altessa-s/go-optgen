// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package parser

import "go/ast"

// TypeFacts are the properties of a resolved field type that plugins rely on.
type TypeFacts struct {
	Interface  bool // The type is an interface
	Nilable    bool // The type can be nil: pointer, slice, map, channel, function or interface
	Numeric    bool // The underlying type is an integer or floating-point type
	Comparable bool // Values of the type are comparable
}

// TypeResolver supplies facts about type expressions of the parsed files. It
// reports false when the type is unknown, for example a type parameter or a
// type that failed to type-check; the parser then infers from the syntax.
type TypeResolver interface {
	TypeFacts(expr ast.Expr) (TypeFacts, bool)
}
