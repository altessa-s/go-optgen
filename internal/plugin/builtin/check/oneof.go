// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package check

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/altessa-s/go-optgen/model"
	"github.com/altessa-s/go-optgen/plugin"
)

const oneofPriority = 40

// OneOfCheck generates validation code ensuring a string value is one of allowed values.
// Tag usage: optcheck:"oneof=[val1,val2,val3]" where values are string literals.
type OneOfCheck struct {
	plugin.CheckBase
}

// Validate accepts string fields with at least one allowed value.
func (c *OneOfCheck) Validate(field model.OptField, rawValue string) error {
	if plugin.ValueKindOf(field) != plugin.ValueString {
		return notApplicable(field, "string")
	}
	if len(parseStringList(rawValue)) == 0 {
		return fmt.Errorf("%w: oneof requires at least one value", plugin.ErrInvalidCheckValue)
	}
	return nil
}

// Generate renders the validation code for valueVar.
func (c *OneOfCheck) Generate(ctx plugin.GenerationContext, field model.OptField, valueVar, rawValue string) []string {
	vals := parseStringList(rawValue)
	return []string{
		"switch " + valueVar + " {",
		"case " + strings.Join(vals, ", ") + ":",
		"default:",
		"  " + buildFail(ctx, fmt.Sprintf("%s has invalid value (allowed: %s)", field.FieldName, strings.Join(vals, ", "))),
		"}",
	}
}

func parseStringList(raw string) []string {
	s := strings.TrimSpace(raw)
	if rest, ok := strings.CutPrefix(s, "["); ok {
		if rest, ok = strings.CutSuffix(rest, "]"); ok {
			s = rest
		}
	}
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var items []string
	var cur strings.Builder
	var quote rune
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
			if item := normalizeStringLiteral(cur.String()); item != "" {
				items = append(items, item)
			}
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if item := normalizeStringLiteral(cur.String()); item != "" {
		items = append(items, item)
	}
	return items
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
