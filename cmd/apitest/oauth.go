// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
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

	// Verify OAuth access token works
	resp, err = api.OAuth2Api.CheckOAuthClientCredentials(ctx, fmt.Sprintf("Bearer %s", accessToken), &moov.CheckOAuthClientCredentialsOpts{
		XRequestId: optional.NewString(requestId),
	})
	if resp != nil {
		resp.Body.Close()
	}
	return token, err
}
