// Copyright 2020 The Moov Authors
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

func setMoovOAuthToken(conf *moov.Configuration, oauthToken *moov.OAuth2Token) {
	if oauthToken == nil || oauthToken.AccessToken == "" {
		log.Fatal("FAILURE: No OAuth token provided")
	} else {
		conf.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", oauthToken.AccessToken))
	}
}

func createOAuthToken(ctx context.Context, api *moov.APIClient, u *user) (*moov.OAuth2Token, error) {
	// Create OAuth client credentials
	clients, resp, err := api.OAuth2Api.CreateOAuth2Client(ctx, &moov.CreateOAuth2ClientOpts{
		XIdempotencyKey: optional.NewString(generateID()),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("problem creating user: %v", err)
	}

	if len(clients) == 0 {
		return nil, errors.New("no OAuth2 clients created")
	}
	client := clients[0]

	// Generate OAuth2 Token
	token, resp, err := api.OAuth2Api.CreateOAuth2Token(ctx, &moov.CreateOAuth2TokenOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		GrantType:       optional.NewString("client_credentials"),
		ClientId:        optional.NewString(client.ClientId),
		ClientSecret:    optional.NewString(client.ClientSecret),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("problem creating user: %v", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("no OAuth2 access token created")
	}

	// Verify OAuth access token works
	accessToken := fmt.Sprintf("Bearer %s", token.AccessToken)
	resp, err = api.OAuth2Api.CheckOAuthClientCredentials(ctx, accessToken, &moov.CheckOAuthClientCredentialsOpts{})
	if resp != nil {
		resp.Body.Close()
	}
	return &token, err
}

// attemptFailedOAuth2Login will try with a OAuth2 access token to ensure failed credentials don't authenticate a request.
func attemptFailedOAuth2Login(ctx context.Context, api *moov.APIClient) error {
	token, _ := name()

	resp, err := api.OAuth2Api.CheckOAuthClientCredentials(ctx, fmt.Sprintf("Bearer %s", token), &moov.CheckOAuthClientCredentialsOpts{})
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
