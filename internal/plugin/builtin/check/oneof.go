// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package check

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const oneofPriority = 40

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

	if field.Type == "string" {
		return nil
	}

	if !field.IsComparable && field.Type != "bool" && !plugin.IsNumericField(field) && !strings.HasPrefix(field.Type, "*") {
		return notApplicable(field, "comparable")
	}

	for _, item := range items {
		if isGoLiteral(item) {
			if _, err := parser.ParseExpr(item); err != nil {
				return fmt.Errorf("%w: oneof item %q is not a valid Go literal", plugin.ErrInvalidCheckValue, item)
			}

			continue
		}

		if !identifierExpression(item) {
			return fmt.Errorf("%w: oneof item %q must be a Go identifier or qualified identifier", plugin.ErrInvalidCheckValue, item)
		}
	}

	return nil
}

// Generate renders the validation code for valueVar.
func (c *OneOfCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, rawValue string) []string {
	items := parseOneOfList(rawValue)

	vals := make([]string, 0, len(items))
	for _, item := range items {
		if field.Type == "string" {
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
