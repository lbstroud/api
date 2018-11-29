// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/moov-io/base/http/bind"
)

type localPathTransport struct {
	tr http.RoundTripper
}

// RoundTrip modifies the incoming request to reshape a Moov API production URL to a local dev URL.
// The GoDoc for http.RoundTripper state that the request SHOULD not be modified, not MUST. If this
// ends up causing problems we'll have to figure out another solution.
//
// This means:
//  - Dropping /v1/$app routing prefix
//  - Changing the local port used (each app runs on its own port now)
//    - Adjusting the scheme if needed.
func (t *localPathTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Each route looks like /v1/$app/... so we need to trim off the v1 and $app segments
	// while looking up $app's port mapping.
	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 { // parts splits into: "", v1, $app, (rest of url)
		// Pass through whatever this request is.
		return t.tr.RoundTrip(r)
	}

	port := bind.HTTP(parts[2])
	if port == "" {
		return nil, fmt.Errorf("unknown HTTP port for %s", parts[2])
	}

	r.URL.Host = "localhost" + port
	r.URL.Scheme = "http"

	// fixup our path now
	r.URL.Path = "/" + strings.Join(parts[3:], "/") // everything after $app

	return t.tr.RoundTrip(r)
}
