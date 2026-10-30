// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pipeline_test

import (
	"fmt"

	"github.com/altessa-s/go-optgen/examples/pipeline"
)

func Example() {
	timeout, err := pipeline.Check("Hello, World")
	fmt.Println(timeout, err)

	fmt.Printf("%q\n", pipeline.Transform("Hello, World"))
	fmt.Printf("%q\n", pipeline.Transform("Hello, World", pipeline.WithLetterCase(" UPPER "), pipeline.WithPrefix("  ")))
	// Output:
	// 30s <nil>
	// "> hello, world"
	// "  HELLO, WORLD"
}

// With check-failure: skip an argument that fails its optcheck rule is
// dropped and the previous value stays.
func Example_checkFailureSkip() {
	fmt.Printf("%q\n", pipeline.Transform("Hello", pipeline.WithLetterCase("title")))

	_, err := pipeline.Check("a long sentence", pipeline.WithMaxLen(0), pipeline.WithMaxLen(5))
	fmt.Println(err)
	// Output:
	// "> hello"
	// pipeline: text rejected: longer than 5 bytes
}

func Example_sliceSetter() {
	_, err := pipeline.Check("No SPAM here", pipeline.WithForbidden([]string{" Spam ", "SPAM", "ads"}))
	fmt.Println(err)
	// Output:
	// pipeline: text rejected: contains "spam"
}
