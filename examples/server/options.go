// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package server

import (
	"errors"
	"log/slog"
	"net"
	"time"
)

//go:generate optgen generate --option-error --validate --source-file $GOFILE

// Defaults referenced from optgen:"default=..." tags.
const (
	DefaultReadTimeout = 5 * time.Second
	DefaultMaxConns    = 1024
	DefaultMode        = "http"
)

// ErrInvalidIP is returned by WithTrustedProxies for an address that does
// not parse as an IP.
var ErrInvalidIP = errors.New("server: invalid trusted proxy address")

type options struct {
	// addr is mandatory: it becomes the first parameter of newOptions.
	addr string `optgen:"mandatory"`

	// name is trimmed and lower-cased before the checks run; the constructor
	// rejects a missing name because of --validate.
	name string `optval:"lower" optcheck:"required,minlen=3,maxlen=32"`

	// mode accepts one value of a closed set.
	mode string `optgen:"default=DefaultMode" optcheck:"oneof=[http,https,h2c]"`

	// readTimeout must stay positive; a nonpositive value is ignored.
	readTimeout time.Duration `optgen:"default=DefaultReadTimeout" optval:"positive"`

	// maxConns is rejected when zero.
	maxConns int `optgen:"default=DefaultMaxConns" optcheck:"nonzero"`

	// trustedProxies are parsed from strings; a bad address fails the option
	// with ErrInvalidIP.
	trustedProxies []net.IP `optgen:"parseip,err=ErrInvalidIP"`

	// tags accumulate across calls; each call drops empty and duplicate entries.
	tags []string `optgen:"append" optval:"nonempty,dedup"`

	// disableKeepAlive is a negative flag: optval:"invert" names its option
	// WithoutKeepAlive, which sets the field to true.
	disableKeepAlive bool `opt:"KeepAlive" optval:"invert"`

	// debug takes its value as a parameter: WithDebug(true).
	debug bool `optval:"param"`

	// logger defaults to a discarding logger; a nil logger is ignored.
	logger *slog.Logger

	// headers start as an empty map, so callers can add to them safely.
	headers map[string]string

	// id is derived by New: no option is generated for it.
	id string `opt:"-"`
}
