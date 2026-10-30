// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package naming_test

import (
	"testing"

	"github.com/altessa-s/go-optgen/internal/naming"
)

func BenchmarkOptionName(b *testing.B) {
	for b.Loop() {
		_ = naming.OptionName("entityIdExtractor")
	}
}
