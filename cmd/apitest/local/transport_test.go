// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package local

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTransport(t *testing.T) {
	cases := []struct {
		incoming string // URL client would send to Moov's LB
		proxied  string // URL our Transport creates to proxy request
	}{
		{
			"https://api.moov.io/v1/ach/files", // ACH
			"http://localhost:8080/files",
		},
		{
			"https://api.moov.io/v1/users/create", // auth
			"http://localhost:8081/users/create",
		},
		{
			"https://api.moov.io/v1/oauth2/clients", // auth
			"http://localhost:8081/oauth2/clients",
		},
		{
			"https://api.moov.io/v1/ach/customers/foo", // paygate
			"http://localhost:8082/customers/foo",
		},
		{
			"https://api.moov.io/v1/ach/depositories/foo", // paygate
			"http://localhost:8082/depositories/foo",
		},
		{
			"https://api.moov.io/v1/ach/originators/foo", // paygate
			"http://localhost:8082/originators/foo",
		},
		{
			"https://api.moov.io/v1/ach/transfers/foo", // paygate
			"http://localhost:8082/transfers/foo",
		},
		{
			"https://api.moov.io/v1/ofac/downloads", // OFAC
			"http://localhost:8084/downloads",
		},
		{
			"https://api.moov.io/v1/fed/test", // fed
			"http://localhost:8086/fed/test",
		},
	}
	for i := range cases {
		r := httptest.NewRequest("GET", cases[i].incoming, nil)
		u, err := url.Parse(cases[i].proxied)
		if err != nil {
			t.Fatal(err) // problem with test example
		}

		// Proxy request
		tr := &Transport{}
		resp, err := tr.RoundTrip(r) // ignore proxy error
		if resp == nil && strings.Contains(err.Error(), "nil underlying Transport") {
			// svc.Close()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("bogus HTTP status: %s for URL %s", resp.Status, resp.Request.URL)
		}
		if resp.Request.URL.Scheme != u.Scheme {
			t.Errorf("got %s", resp.Request.URL.Scheme)
		}
		if resp.Request.URL.Host != u.Host {
			t.Errorf("got %s", resp.Request.URL.Host)
		}
		if resp.Request.URL.Path != u.Path {
			t.Errorf("got %s", resp.Request.URL.Path)
		}
	}
}
