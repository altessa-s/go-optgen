// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package parser

import (
	"go/ast"
	"go/token"

	"github.com/altessa-s/go-optgen/model"
)

// TypeFacts are the properties of a resolved field type that plugins rely on.
type TypeFacts struct {
	Bool            bool // The type's underlying type is bool
	Interface       bool // The type is an interface
	Nilable         bool // The type can be nil: pointer, slice, map, channel, function or interface
	Numeric         bool // The underlying type is an integer or floating-point type
	Comparable      bool // Values of the type are comparable
	MapKeyType      string
	MapValueType    string
	MapValueNilable bool
}

// TypeResolver supplies facts about type expressions of the parsed files. It
// reports false when the type is unknown, for example a type parameter or a
// type that failed to type-check; the parser then infers from the syntax.
type TypeResolver interface {
	TypeFacts(expr ast.Expr) (TypeFacts, bool)
	ResolveSetterTarget(field ast.Expr, pos token.Pos, expression string, method bool, imports map[string]model.ImportInfo) (model.SetterTarget, error)
}

// CheckValueResolver reports whether a check expression is type-compatible with its field.
type CheckValueResolver interface {
	CheckValueValid(field ast.Expr, pos token.Pos, expression string, constant bool) bool
}
