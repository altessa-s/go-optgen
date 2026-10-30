// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package server_test

import (
	"errors"
	"fmt"

	"github.com/altessa-s/go-optgen/examples/server"
)

func Example() {
	s, err := server.New(":8080",
		server.WithName("  API  "), // trimmed, then lower-cased
		server.WithMode("https"),
		server.WithReadTimeout(0), // nonpositive: ignored, the default stays
		server.WithTrustedProxies("10.0.0.1", "::1"),
		server.WithTags("edge", "", "edge", "eu"), // empty and duplicate entries dropped
		server.WithTags("canary"),                 // appended
		server.WithoutKeepAlive(),
		server.WithDebug(true),
		server.WithLogger(nil), // nil: ignored, the discarding logger stays
		server.WithHeaders(map[string]string{"X-Env": "prod"}),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(s)
	fmt.Println(s.Logger() != nil)
	// Output:
	// api@:8080 https timeout=5s maxConns=1024 keepAlive=false debug=true proxies=[10.0.0.1 ::1] tags=[edge eu canary] headers=[X-Env=prod]
	// true
}

// An option rejects a bad argument immediately, and the constructor checks
// the optcheck rules once more on the final configuration, so a rule also
// catches a field that no option ever set.
func Example_validation() {
	_, err := server.New(":8080", server.WithName("api"), server.WithMode("ftp"))
	fmt.Println(err)

	_, err = server.New(":8080", server.WithName("x"))
	fmt.Println(err)

	_, err = server.New(":8080") // name is required, but no option set it
	fmt.Println(err)

	_, err = server.New(":8080", server.WithName("api"), server.WithTrustedProxies("10.0.0.1", "proxy.local"))
	fmt.Println(errors.Is(err, server.ErrInvalidIP))
	// Output:
	// server: mode has invalid value (allowed: "http", "https", "h2c")
	// server: name is too short (min 3)
	// server: name is required
	// true
}

// String options accept a *string too; a nil pointer leaves the field unset.
func Example_stringPointer() {
	var mode *string
	s, err := server.New(":80", server.WithName(new("web")), server.WithMode(mode))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(s)
	// Output:
	// web@:80 http timeout=5s maxConns=1024 keepAlive=true debug=false proxies=[] tags=[] headers=[]
}
