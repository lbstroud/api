// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moov-io/base/http/bind"
)

func TestLocalPathTransport(t *testing.T) {
	r := httptest.NewRequest("GET", "https://api.moov.io/v1/ach/files/fileId", nil)
	tr := &localPathTransport{
		tr: &http.Transport{},
	}

	svc := &http.Server{
		Addr: bind.HTTP("ach"),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		}),
	}
	go svc.ListenAndServe()
	defer svc.Close()

	resp, err := tr.RoundTrip(r)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Request.URL.Path != "/files/fileId" {
		t.Errorf("got %s", resp.Request.URL.Path)
	}
	if resp.Request.URL.Host != "localhost:8080" {
		t.Errorf("got %s", resp.Request.URL.Host)
	}
	if resp.Request.URL.Scheme != "http" {
		t.Errorf("got %s", resp.Request.URL.Scheme)
	}
}
