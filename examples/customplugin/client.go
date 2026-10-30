// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package customplugin

import "fmt"

// Client is a configured client of a remote endpoint.
type Client struct {
	opts *options
}

// NewClient returns a Client. WithEndpoint is required.
func NewClient(opts ...Option) (*Client, error) {
	o, err := newOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}
	return &Client{opts: o}, nil
}

// String describes the effective configuration.
func (c *Client) String() string {
	return fmt.Sprintf("endpoint=%s userAgent=%q", c.opts.endpoint, c.opts.userAgent)
}
