// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

// setMoovAuthCookie adds authentication onto our Moov API client for all requests
func setMoovAuthCookie(conf *moov.Configuration, user *user) {
	if user.Cookie.Value != "" {
		conf.AddDefaultHeader("Cookie", fmt.Sprintf("moov_auth=%s", user.Cookie.Value))
	} else {
		log.Fatalf("no cookie found (userId: %v)", user.ID)
	}
	conf.AddDefaultHeader("X-User-Id", user.ID)
}

func removeMoovAuthCookie(conf *moov.Configuration) {
	delete(conf.DefaultHeader, "Cookie")
}

// verifyUserIsLoggedIn takes the given moov.APIClient and checks if it's is logged in. A non-nil error signals
// the client doens't have valid authentication.
func verifyUserIsLoggedIn(ctx context.Context, api *moov.APIClient, user *user, requestID string) error {
	resp, err := api.UserApi.CheckUserLogin(ctx, &moov.CheckUserLoginOpts{
		XRequestID: optional.NewString(requestID),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("problem checking user (id=%s) login: %v", user.ID, err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("on cookie check, got %s status", resp.Status)
	}
	return nil
}

// attemptFailedLogin will try with random data to ensure failed credentials don't authenticate a request.
func attemptFailedLogin(ctx context.Context, api *moov.APIClient, requestID string) error {
	email, password := name()                                                     // random noise
	login := moov.Login{Email: email + "@moov.io", Password: password + password} // email format, make sure it's long enough
	_, resp, err := api.UserApi.UserLogin(ctx, login, &moov.UserLoginOpts{
		XRequestID:      optional.NewString(requestID),
		XIdempotencyKey: optional.NewString(generateID()),
	})
	if resp != nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			return fmt.Errorf("got %s response code", resp.Status)
		}
	}
	if err == nil {
		bs, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%v: %v", string(bs), err)
		}
		return errors.New("expected error, but got nothing")
	}
	return nil
}
