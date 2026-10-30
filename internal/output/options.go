// Copyright 2021-2026 ALTESSA SOLUTIONS INC. All rights reserved.
// Use of this source code is governed by license that can be found in
// the LICENSE file.

package output

//go:generate optgen generate

type options struct {
	// allowedFormatters extends the built-in formatter allowlist with command
	// names resolved from PATH. Paths are rejected like any formatter path.
	allowedFormatters []string `optval:"nodup"`
}
