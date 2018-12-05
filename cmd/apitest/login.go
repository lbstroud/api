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

// setMoovCookie adds authentication onto our Moov API client for all requests
func setMoovCookie(conf *moov.Configuration, cookie *http.Cookie) {
	if cookie.Value != "" {
		conf.AddDefaultHeader("Cookie", fmt.Sprintf("moov_auth=%s", cookie.Value))
	} else {
		log.Fatal("no cookie found")
	}
}

// verifyUserIsLoggedIn takes the given moov.APIClient and checks if it's is logged in. A non-nil error signals
// the client doens't have valid authentication.
func verifyUserIsLoggedIn(ctx context.Context, api *moov.APIClient, user *user, requestId string) error {
	resp, err := api.UserApi.CheckUserLogin(ctx, &moov.CheckUserLoginOpts{
		XRequestId: optional.NewString(requestId),
	})
	if err != nil {
		return fmt.Errorf("problem checking user (id=%s) login: %v", user.ID, err)
	}
	if resp != nil {
		if resp.StatusCode != 200 {
			return fmt.Errorf("on cookie check, got %s status", resp.Status)
		}
		return resp.Body.Close()
	}
	return nil
}

// attemptFailedLogin will try with random data to ensure failed credentials don't authenticate a request.
func attemptFailedLogin(ctx context.Context, api *moov.APIClient, requestId string) error {
	email, password := name()                                                     // random noise
	login := moov.Login{Email: email + "@moov.io", Password: password + password} // email format, make sure it's long enough
	_, resp, err := api.UserApi.UserLogin(ctx, login, &moov.UserLoginOpts{
		XRequestId:      optional.NewString(requestId),
		XIdempotencyKey: optional.NewString(generateID()),
	})
	if resp != nil {
		if resp.StatusCode != http.StatusForbidden {
			return fmt.Errorf("got %s response code", resp.Status)
		}
		defer resp.Body.Close()
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
