// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package customplugin shows generation with plugins of your own.
//
// The collapse transform and the hostport check live in [myplugins]. The
// optgen binary does not know them, so the package generates its options
// through ./cmd/optgen, a main that wraps optgen.Run with
// optgen.WithPlugins(myplugins.Register):
//
//	//go:generate go run ./cmd/optgen generate --option-error --validate --source-file $GOFILE
//
// [myplugins]: https://pkg.go.dev/github.com/altessa-s/go-optgen/examples/customplugin/myplugins
package customplugin
