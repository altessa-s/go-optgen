// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package server shows the strict end of optgen: options that return
// errors, optcheck rules re-checked by the constructor, and the built-in
// field plugins.
//
// # Generation
//
//	//go:generate optgen generate --option-error --validate --source-file $GOFILE
//
// The repository .optgen.yaml adds all-fields, so untagged fields such as
// logger and headers get options and type defaults too.
//
// # Features
//
//   - optgen:"mandatory" turns addr into a parameter of newOptions.
//   - optgen:"default=Const" starts fields at exported constants.
//   - optcheck rules (required, minlen, maxlen, oneof, nonzero) fail the
//     option and, with --validate, the final configuration.
//   - optgen:"parseip,err=ErrInvalidIP" parses []net.IP from strings.
//   - optgen:"append" with optval:"nonempty,dedup" accumulates clean tags.
//   - optval:"invert" and optval:"param" shape bool options.
package server
