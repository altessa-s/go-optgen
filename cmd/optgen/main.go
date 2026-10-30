// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package main

import (
	"os"

	optgen "github.com/altessa-s/go-optgen"
)

func main() {
	if err := optgen.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
