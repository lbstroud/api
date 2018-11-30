// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

func verifyUserIsLoggedIn(ctx context.Context, api *moov.APIClient, user *User, requestId string) error {
	resp, err := api.UserApi.CheckUserLogin(ctx, &moov.CheckUserLoginOpts{
		XRequestId: optional.NewString(requestId),
	})
	if err != nil {
		return fmt.Errorf("problem checking user (id=%s) login: %v", user.ID, err)
	}
	if resp != nil {
		return resp.Body.Close()
	}
	return nil
}
