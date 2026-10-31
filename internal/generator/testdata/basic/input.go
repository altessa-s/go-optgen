// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package basic

// This file is parsed by generator golden tests (it is under testdata, so it is not compiled).

import (
	"log/slog"
	"net"
	"regexp"
	"time"
)

type options struct {
	timeout   time.Duration     `opt:"Timeout" optgen:"default=30*time.Second"`
	hosts     []string          `opt:"Hosts"`
	labels    []string          `opt:"Labels" optval:"trimspaces,lower,dedup" optcheck:"maxlen=50"`
	patterns  []*regexp.Regexp  `opt:"Patterns" optgen:"manual"`
	ips       []net.IP          `opt:"IPs" optgen:"parseip,err=errInvalidIP,append" optcheck:"maxlen=10"`
	ports     []int             `opt:"Ports"` // deduplicated straight from the variadic argument
	logger    *slog.Logger      `opt:"Logger"`
	headers   map[string]string `opt:"Headers"`
	maxItems  int               `opt:"MaxItems" optval:"positive"`
	threshold int               `opt:"Threshold" optval:"positive=allow_zero"`
	mandatory bool              `opt:"Mandatory" optval:"param" optgen:"default=true"`
	name      string            `opt:"Name" optcheck:"minlen=3"`
}

var errInvalidIP = func() error { return nil }()
