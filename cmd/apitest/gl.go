// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"

	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

func createGLAccount(ctx context.Context, api *moov.APIClient, u *user, name, requestId string) (*moov.Account, error) {
	req := moov.CreateAccount{
		Name: name,
		Type: "Savings",
	}
	opts := &moov.CreateAccountOpts{
		XRequestId: optional.NewString(requestId),
	}
	account, resp, err := api.GLApi.CreateAccount(ctx, u.ID, u.ID, req, opts)
	if *flagDebug && resp != nil {
		log.Printf("GL create account request URL: %s (status=%s): %v\n", resp.Request.URL.String(), resp.Status, err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("problem creating GL account %s: %v", name, err)
	}
	return &account, nil
}
