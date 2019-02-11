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
	"os"
	"path/filepath"
	"time"

	"github.com/moov-io/api/pkg/moovoauth"
	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

var (
	OAuthTokenStorageFilepath string = moovoauth.TokenFilepath()
)

// OAuthToken token holds various values from the OAuth2 Token generation.
// These values include: access token, expires, refresh token, etc..
type OAuthToken map[string]interface{}

// Get returns a specific OAuth2 token or value by their given name.
func (o OAuthToken) Get(name string) string {
	v, ok := o[name].(string)
	if !ok {
		return ""
	}
	return v
}

// Access returns the OAuth access token value.
//
// Note: In order to use in a header you must prefix 'Bearer ' with this value.
func (o OAuthToken) Access() string {
	return o.Get("access_token")
}

// Expires returns a Duration for when the access/refresh tokens expire.
func (o OAuthToken) Expires() time.Duration {
	v, ok := o["expires_in"].(float64)
	if !ok {
		return time.Duration(0 * time.Second)
	}
	dur, err := time.ParseDuration(fmt.Sprintf("%fs", v))
	if err != nil {
		return time.Duration(0 * time.Second)
	}
	return dur
}

func setMoovOAuthToken(conf *moov.Configuration, oauthToken OAuthToken) {
	if v := oauthToken.Access(); v != "" {
		conf.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", v))
	} else {
		log.Fatal("FAILURE: No OAuth token provided")
	}
}

func createOAuthToken(ctx context.Context, api *moov.APIClient, u *user, requestId string) (OAuthToken, error) {
	// Create OAuth client credentials
	clients, resp, err := api.OAuth2Api.CreateOAuth2Client(ctx, &moov.CreateOAuth2ClientOpts{
		XRequestId:      optional.NewString(requestId),
		XIdempotencyKey: optional.NewString(generateID()),
	})
	if err != nil {
		return nil, fmt.Errorf("problem creating user: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	if len(clients) == 0 {
		return nil, errors.New("no OAuth2 clients created")
	}
	client := clients[0]

	// Generate OAuth2 Token
	tk, resp, err := api.OAuth2Api.CreateOAuth2Token(ctx, &moov.CreateOAuth2TokenOpts{
		XRequestId:      optional.NewString(requestId),
		XIdempotencyKey: optional.NewString(generateID()),
		GrantType:       optional.NewString("client_credentials"),
		ClientId:        optional.NewString(client.ClientId),
		ClientSecret:    optional.NewString(client.ClientSecret),
	})
	if err != nil {
		return nil, fmt.Errorf("problem creating user: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	if len(tk) == 0 {
		return nil, errors.New("no OAuth2 token created")
	}
	token := OAuthToken(tk)

	accessToken := token.Access()
	if accessToken == "" {
		return nil, errors.New("no OAuth2 access token created")
	}

	// Write OAuth access token to disk
	if OAuthTokenStorageFilepath != "" {
		if err := writeOAuthToken(accessToken, OAuthTokenStorageFilepath); err != nil {
			return nil, err
		}
	}

	// Verify OAuth access token works
	resp, err = api.OAuth2Api.CheckOAuthClientCredentials(ctx, fmt.Sprintf("Bearer %s", accessToken), &moov.CheckOAuthClientCredentialsOpts{
		XRequestId: optional.NewString(requestId),
	})
	if resp != nil {
		resp.Body.Close()
	}
	return token, err
}

// attemptFailedOAuth2Login will try with a OAuth2 access token to ensure failed credentials don't authenticate a request.
func attemptFailedOAuth2Login(ctx context.Context, api *moov.APIClient, requestId string) error {
	token, _ := name()

	resp, err := api.OAuth2Api.CheckOAuthClientCredentials(ctx, fmt.Sprintf("Bearer %s", token), &moov.CheckOAuthClientCredentialsOpts{
		XRequestId: optional.NewString(requestId),
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

func writeOAuthToken(accessToken string, path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("writeOAuthToken: %v", err)
	}
	// Remove file if it already exists
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}

	// Create the parent dir if it doesn't exist
	parent, _ := filepath.Split(path)
	if _, err := os.Stat(parent); err != nil && os.IsNotExist(err) {
		fmt.Println(parent)

		if err := os.MkdirAll(parent, 0777); err != nil {
			return fmt.Errorf("writeOAuthToken: mkdir %s got %v", parent, err)
		}
	}

	return ioutil.WriteFile(path, []byte(accessToken), 0600)
}
