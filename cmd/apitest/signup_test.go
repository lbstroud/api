// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSignup__findMoovCookie(t *testing.T) {
	var cookies []*http.Cookie
	cookies = append(cookies, &http.Cookie{
		Name:  "moov_auth",
		Value: "foobar",
	})

	// happy path
	cookie := findMoovCookie(cookies)
	if cookie == nil {
		t.Fatal("should have found moov_auth")
	}
	if cookie.Value != "foobar" {
		t.Fatalf("expected foobar got %s", cookie.Value)
	}

	// non happy path
	cookie = findMoovCookie(nil)
	if cookie != nil {
		t.Errorf("cookie shouldn't have been found, got %s", cookie.String())
	}
}

func TestSignup__email(t *testing.T) {
	v := email("Jane", "Doe")
	if !strings.HasPrefix(v, "jane.doe") || !strings.HasSuffix(v, "@example.com") {
		t.Errorf("got %s", v)
	}
}

func TestSignup__name(t *testing.T) {
	for i := 0; i < 1e5; i++ {
		first, last := name()
		if first == "" || last == "" {
			t.Errorf("first=%q last=%q", first, last)
		}
	}
}

func TestSignup__phone(t *testing.T) {
	for i := 0; i < 1e5; i++ {
		v := phone()
		if n := strings.Count(v, "."); n != 2 {
			t.Errorf("%s has missing/extra .'s", v)
		}
		n1, n2 := utf8.RuneCountInString(v), utf8.RuneCountInString("123.456.7890")
		if n1 != n2 {
			t.Errorf("%s has incorrect length: %d expected %d", v, n1, n2)
		}
	}
}
