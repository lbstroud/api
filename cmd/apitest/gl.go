// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	"github.com/moov-io/base/http/bind"
	"github.com/moov-io/base/k8s"
	gl "github.com/moov-io/gl/client"

	"github.com/antihax/optional"
)

func setupGLClient() *gl.APIClient {
	conf := gl.NewConfiguration()
	// conf.BasePath = "https://api.moov.io/v1/qledger"
	conf.BasePath = "http://localhost" + bind.HTTP("gl")
	if k8s.Inside() {
		conf.BasePath = "http://gl.apps.svc.cluster.local:8080"
	}
	return gl.NewAPIClient(conf)
}

func createGLAccount(ctx context.Context, api *gl.APIClient, u *user, name, requestId string) (*gl.Account, error) {
	opts := &gl.CreateAccountOpts{
		XRequestId: optional.NewString(requestId),
	}
	account, resp, err := api.GLApi.CreateAccount(ctx, u.ID, u.ID, gl.CreateAccount{
		Name: name,
		Type: "Savings",
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("problem creating GL account %s: %v", name, err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return &account, nil
}
