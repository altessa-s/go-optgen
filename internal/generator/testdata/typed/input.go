// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package typed

// This file is parsed by generator golden tests (it is under testdata, so it is not compiled).
// It imports only the standard library so the generated output can be type-checked.

type options struct {
	id      [16]byte               `opt:"ID"`
	hook    func(int) error        `opt:"Hook"`
	feature struct{ Enabled bool } `opt:"Feature"`
	tagged  struct {
		A int `json:"a"`
	} `opt:"Tagged"`
	closer interface{ Close() error } `opt:"Closer"`
	name   string                     `opt:"Name" optcheck:"minlen=3"`
}
