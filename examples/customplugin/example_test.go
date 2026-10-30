// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package customplugin_test

import (
	"errors"
	"fmt"
	"net"

	"github.com/altessa-s/go-optgen/examples/customplugin"
)

func Example() {
	c, err := customplugin.NewClient(
		customplugin.WithEndpoint("api.example.com:443"),
		customplugin.WithUserAgent("  my-app/1.0 \t (darwin;   arm64) "), // collapse
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(c)
	// Output:
	// endpoint=api.example.com:443 userAgent="my-app/1.0 (darwin; arm64)"
}

// The hostport check wraps the error of net.SplitHostPort, so callers can
// inspect it.
func Example_hostPort() {
	_, err := customplugin.NewClient(customplugin.WithEndpoint("api.example.com"))
	fmt.Println(err)
	if addrErr, ok := errors.AsType[*net.AddrError](err); ok {
		fmt.Println(addrErr.Addr)
	}

	_, err = customplugin.NewClient()
	fmt.Println(err)
	// Output:
	// client: endpoint: address api.example.com: missing port in address
	// api.example.com
	// client: endpoint is required
}
