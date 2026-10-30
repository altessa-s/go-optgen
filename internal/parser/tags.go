// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package parser

import (
	"reflect"
	"strings"
)

// ParseTag extracts the value of a specific tag from a struct tag string.
// Uses reflect.StructTag for safe and proper tag parsing that handles
// multiple tags, escaping, and edge cases correctly.
func ParseTag(tagStr, key string) string {
	tagStr = strings.Trim(tagStr, "`")
	tag := reflect.StructTag(tagStr)
	return tag.Get(key)
}

// LookupTag extracts the value of a specific tag and reports whether it exists.
// This distinguishes between a missing tag and an empty tag value.
func LookupTag(tagStr, key string) (value string, ok bool) {
	tagStr = strings.Trim(tagStr, "`")
	tag := reflect.StructTag(tagStr)
	return tag.Lookup(key)
}

// ParseOptName parses the opt tag value.
// Only two forms are supported:
//   - opt:"Name"  (generate WithName)
//   - opt:"-"     (skip field)
func ParseOptName(tag string) (name string, skip bool) {
	tag = strings.TrimSpace(tag)
	if tag == "-" {
		return "", true
	}
	return tag, false
}

// ParseOptModifiers parses modifiers (optval tag).
// Accepted forms:
//   - "lower"
//   - "lower,trimspaces"
//   - "[lower, trimspaces]"
func ParseOptModifiers(s string) []string {
	return ParseExprList(s)
}

// ParseExprList parses a comma-separated list of Go expressions.
// It accepts optional surrounding brackets:
//   - "a,b,c"
//   - "[a, b, c]"
//
// Whitespace is ignored and empty items are dropped.
func ParseExprList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if rest, ok := strings.CutPrefix(s, "["); ok {
		if rest, ok = strings.CutSuffix(rest, "]"); ok {
			s = rest
		}
	}
	parts := smartSplit(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseFlagKV parses comma-separated flag / key=value items.
// Bare flags are treated as true.
func parseFlagKV(parts []string) map[string]string {
	out := make(map[string]string)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.Index(part, "="); idx > 0 {
			key := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])
			if key != "" {
				out[key] = value
			}
			continue
		}
		out[part] = "true"
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseOptChecks parses comma-separated validation rules used by optcheck tag.
// It shares the same "flag / key=value" grammar as optgen metadata.
func parseOptChecks(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]string{}
	}
	parts := smartSplit(s)
	out := parseFlagKV(parts)
	if len(out) == 0 {
		return map[string]string{}
	}

	return out
}

// bareKeys reports which items were written without an explicit "=value"
// part. parseFlagKV coerces such items to "true", which is fine for flags but
// must not satisfy checks that require a value.
func bareKeys(parts []string) map[string]bool {
	bare := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && strings.Index(part, "=") <= 0 {
			bare[part] = true
		}
	}
	return bare
}

// ParseOptGen parses optgen tag (generation-related settings).
// Supported:
//   - default=Expr
//   - flags / key=value metadata (e.g. append, notnil, parseip, err=...)
func ParseOptGen(s string) (defaultVal string, metadata map[string]string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	parts := smartSplit(s)
	metaParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if val, ok := strings.CutPrefix(part, "default="); ok {
			defaultVal = val
			continue
		}
		metaParts = append(metaParts, part)
	}
	metadata = parseFlagKV(metaParts)
	return defaultVal, metadata
}

// smartSplit splits a string by delimiter but respects brackets [], braces {},
// parentheses (), and quoted strings.
// This is used to parse tag values that may contain complex expressions like:
//   - default=[value1, value2]
//   - default={key: value}
//   - default=time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
func smartSplit(s string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	var quote rune
	escaped := false

	for _, r := range s {
		if quote != 0 {
			current.WriteRune(r)
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
			current.WriteRune(r)
		case '[', '{', '(':
			depth++
			current.WriteRune(r)
		case ']', '}', ')':
			if depth > 0 {
				depth--
			}
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
