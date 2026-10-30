// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pipeline

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Letter cases accepted by WithLetterCase.
const (
	caseLower = "lower"
	caseUpper = "upper"
)

// ErrRejected is returned by Check for text that breaks a rule.
var ErrRejected = errors.New("pipeline: text rejected")

// Check validates s and reports how long it may take to process it.
func Check(s string, opts ...CheckOption) (time.Duration, error) {
	o := newCheckOptions(opts...)
	if len(s) > o.maxLen {
		return 0, fmt.Errorf("%w: longer than %d bytes", ErrRejected, o.maxLen)
	}
	lower := strings.ToLower(s)
	for _, word := range o.forbidden {
		if strings.Contains(lower, word) {
			return 0, fmt.Errorf("%w: contains %q", ErrRejected, word)
		}
	}
	return o.timeout, nil
}

// Transform rewrites s.
func Transform(s string, opts ...TransformOption) string {
	o := newTransformOptions(opts...)
	switch o.letterCase {
	case caseLower:
		s = strings.ToLower(s)
	case caseUpper:
		s = strings.ToUpper(s)
	}
	return o.prefix + s
}
