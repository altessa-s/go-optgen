// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package defaults

import (
	"go/parser"
	"strconv"
	"time"
)

// DurationDefaultPlugin rewrites time.Duration tag defaults such as "30s" or
// "1h30m", which are not Go expressions, into ones such as 90 * time.Minute.
// It provides no default of its own.
type DurationDefaultPlugin struct {
	typeDefaultBase
}

// CanProvideDefault reports false: durations have no implicit default.
func (p *DurationDefaultPlugin) CanProvideDefault(string) bool { return false }

// GetDefault is never called because CanProvideDefault reports false.
func (p *DurationDefaultPlugin) GetDefault(string) string { return "" }

// NormalizeDefault rewrites a duration literal of a time.Duration field.
// Defaults that already are Go expressions or no valid duration are left alone.
func (p *DurationDefaultPlugin) NormalizeDefault(typeStr, def string) (string, bool) {
	if typeStr != "time.Duration" {
		return "", false
	}
	if _, err := parser.ParseExpr(def); err == nil {
		return "", false
	}
	d, err := time.ParseDuration(def)
	if err != nil {
		return "", false
	}
	for _, unit := range []struct {
		d    time.Duration
		name string
	}{{time.Hour, "Hour"}, {time.Minute, "Minute"}, {time.Second, "Second"}, {time.Millisecond, "Millisecond"}, {time.Microsecond, "Microsecond"}} {
		if d%unit.d == 0 {
			return strconv.FormatInt(int64(d/unit.d), 10) + " * time." + unit.name, true
		}
	}
	return "time.Duration(" + strconv.FormatInt(int64(d), 10) + ")", true
}
