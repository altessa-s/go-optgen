// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const (
	oneofPriority = 40
	stringType    = "string"
)

// OneOfCheck generates validation code ensuring a value is one of the allowed values.
// Tag usage: optcheck:"oneof=[val1,val2,val3]".
type OneOfCheck struct {
	plugin.CheckBase
}

// Validate accepts string fields with at least one allowed value.
func (c *OneOfCheck) Validate(field model.OptField, rawValue string) error {
	items := parseOneOfList(rawValue)
	if len(items) == 0 {
		return fmt.Errorf("%w: oneof requires at least one value", plugin.ErrInvalidCheckValue)
	}

	seen := make(map[string]string, len(items))
	for _, item := range items {
		identity := field.CheckValueIdentity[item]
		if identity == "" {
			identity = strings.TrimSpace(item)
			if field.Type == stringType {
				identity = normalizeStringLiteral(item)
			}
		}

		if previous, exists := seen[identity]; exists {
			return fmt.Errorf("%w: duplicate oneof items %q and %q", plugin.ErrInvalidCheckValue, previous, item)
		}

		seen[identity] = item
	}

	if field.Type == stringType {
		return nil
	}

	if !oneOfApplicable(field) {
		return notApplicable(field, "comparable")
	}

	for _, item := range items {
		if err := validateOneOfItem(field, item); err != nil {
			return err
		}
	}

	return nil
}

func oneOfApplicable(field model.OptField) bool {
	if !field.TypeResolved && identifierExpression(field.Type) {
		return true
	}

	return field.IsComparable || field.Type == "bool" || plugin.IsNumericField(field) || strings.HasPrefix(field.Type, "*")
}

func validateOneOfItem(field model.OptField, item string) error {
	if valid, ok := field.CheckValueValid[item]; ok && !valid {
		return fmt.Errorf("%w: oneof item %q must be a compatible constant", plugin.ErrInvalidCheckValue, item)
	}

	if plugin.IsNumericField(field) {
		if evaluated, err := types.Eval(token.NewFileSet(), nil, token.NoPos, item); err == nil && evaluated.Value != nil {
			kind := evaluated.Value.Kind()
			if kind != constant.Int && kind != constant.Float {
				return fmt.Errorf("%w: oneof item %q is not numeric", plugin.ErrInvalidCheckValue, item)
			}

			if isIntegerType(field.Type) && kind != constant.Int {
				return fmt.Errorf("%w: oneof item %q is not an integer", plugin.ErrInvalidCheckValue, item)
			}

			return nil
		}

		if isGoLiteral(item) && (item[0] == '"' || item[0] == '\'' || item[0] == '`') {
			return fmt.Errorf("%w: oneof item %q is not numeric", plugin.ErrInvalidCheckValue, item)
		}
	}

	if isGoLiteral(item) {
		if _, err := parser.ParseExpr(item); err != nil {
			return fmt.Errorf("%w: oneof item %q is not a valid Go literal", plugin.ErrInvalidCheckValue, item)
		}

		return nil
	}

	if !identifierExpression(item) {
		return fmt.Errorf("%w: oneof item %q must be a Go identifier or qualified identifier", plugin.ErrInvalidCheckValue, item)
	}

	return nil
}

func isIntegerType(typeName string) bool {
	return strings.HasPrefix(typeName, "int") || strings.HasPrefix(typeName, "uint")
}

// Generate renders the validation code for valueVar.
func (c *OneOfCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, rawValue string) []string {
	items := parseOneOfList(rawValue)

	vals := make([]string, 0, len(items))
	for _, item := range items {
		if field.Type == stringType {
			vals = append(vals, normalizeStringLiteral(item))
		} else {
			vals = append(vals, item)
		}
	}

	return []string{
		"switch " + valueVar + " {",
		"case " + strings.Join(vals, ", ") + ":",
		"default:",
		"  " + buildFail(ctx, fmt.Sprintf("%s has invalid value (allowed: %s)", field.FieldName, strings.Join(vals, ", "))),
		"}",
	}
}

func parseOneOfList(raw string) []string {
	s := strings.TrimSpace(raw)
	if rest, ok := strings.CutPrefix(s, "["); ok {
		if rest, ok = strings.CutSuffix(rest, "]"); ok {
			s = rest
		}
	}

	if strings.TrimSpace(s) == "" {
		return nil
	}

	var (
		items []string
		cur   strings.Builder
		quote rune
	)

	escaped := false

	for _, r := range s {
		if quote != 0 {
			cur.WriteRune(r)

			if escaped {
				escaped = false
				continue
			}

			if r == '\\' {
				escaped = true
				continue
			}

			if r == quote {
				quote = 0
			}

			continue
		}

		switch r {
		case '"', '\'', '`':
			quote = r
			cur.WriteRune(r)
		case ',':
			if item := strings.TrimSpace(cur.String()); item != "" {
				items = append(items, item)
			}

			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}

	if item := strings.TrimSpace(cur.String()); item != "" {
		items = append(items, item)
	}

	return items
}

func isGoLiteral(s string) bool {
	return len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '`' && s[len(s)-1] == '`') ||
		(s[0] == '\'' && s[len(s)-1] == '\''))
}

func identifierExpression(s string) bool {
	expr, err := parser.ParseExpr(s)
	if err != nil {
		return false
	}

	var valid func(ast.Expr) bool

	valid = func(expr ast.Expr) bool {
		switch node := expr.(type) {
		case *ast.Ident:
			return token.IsIdentifier(node.Name)
		case *ast.SelectorExpr:
			return token.IsIdentifier(node.Sel.Name) && valid(node.X)
		default:
			return false
		}
	}

	return valid(expr)
}

func normalizeStringLiteral(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '`' && s[len(s)-1] == '`')) {
		return s
	}

	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
	}

	return strconv.Quote(s)
}
