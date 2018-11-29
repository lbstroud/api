// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"net/http"
	"strings"
)

var (
	// localhostOverrides is a mapping of $app (from /v1/$app/* paths) to port and path.
	localhostOverrides = map[string]string{
		"ach":     ":8080",
		"auth":    ":8081",
		"paygate": ":8082",
		"users":   ":8081/users/",
		"x9":      ":8083",
	}
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

	if over, exists := localhostOverrides[parts[2]]; exists {
		r.URL.Scheme = "http"

		idx := strings.Index(over, "/")
		if idx > -1 {
			r.URL.Host = "localhost" + over[:idx]
			r.URL.Path = over[idx:] + strings.Join(parts[3:], "/")
		} else {
			r.URL.Host = "localhost" + over
			r.URL.Path = "/" + strings.Join(parts[3:], "/") // everything after $app
		}
	}

	return t.tr.RoundTrip(r)
}
