// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package customplugin

// The directive runs the wrapping main of ./cmd/optgen instead of the
// optgen binary, so the collapse and hostport keys are known.
//go:generate go run ./cmd/optgen generate --option-error --validate --source-file $GOFILE

type options struct {
	// endpoint must be a host:port pair; --validate also rejects a client
	// created without one.
	endpoint string `optcheck:"required,hostport"`

	// userAgent has its inner white space collapsed.
	userAgent string `optval:"collapse" optcheck:"maxlen=64"`
}
