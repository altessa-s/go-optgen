// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package server

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
)

// Server is a configured, not yet started, server.
type Server struct {
	opts *options
}

// New returns a Server listening on addr. It fails when an option rejects
// its argument or when the final configuration breaks an optcheck rule.
func New(addr string, opts ...Option) (*Server, error) {
	o, err := newOptions(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}
	o.id = o.name + "@" + o.addr
	return &Server{opts: o}, nil
}

// Logger returns the logger the server writes to.
func (s *Server) Logger() *slog.Logger { return s.opts.logger }

// String describes the effective configuration.
func (s *Server) String() string {
	o := s.opts
	proxies := make([]string, len(o.trustedProxies))
	for i, ip := range o.trustedProxies {
		proxies[i] = ip.String()
	}
	headers := make([]string, 0, len(o.headers))
	for _, k := range slices.Sorted(maps.Keys(o.headers)) {
		headers = append(headers, k+"="+o.headers[k])
	}
	return fmt.Sprintf("%s %s timeout=%s maxConns=%d keepAlive=%t debug=%t proxies=[%s] tags=[%s] headers=[%s]",
		o.id, o.mode, o.readTimeout, o.maxConns, !o.disableKeepAlive, o.debug,
		strings.Join(proxies, " "), strings.Join(o.tags, " "), strings.Join(headers, " "))
}
