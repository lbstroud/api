// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/moov-io/base/http/bind"
	"github.com/moov-io/base/k8s"
	gl "github.com/moov-io/gl/client"

	"github.com/antihax/optional"
)

var glAddressOnce sync.Once

// setupGLClient returns an API client for our GL service
//
// TODO(adam): remove this extra client init and fold createGLAccount into our main moov.APIClient
// We can't do this right now because of an OpenAPITools issue (https://github.com/OpenAPITools/openapi-generator/issues/2008)
// which is preventing us from generating a client with remote $ref's
func setupGLClient(u *user) *gl.APIClient {
	conf := gl.NewConfiguration()

	// logic copied from login.go
	conf.AddDefaultHeader("Cookie", fmt.Sprintf("moov_auth=%s", u.Cookie.Value))
	// conf.AddDefaultHeader("X-User-Id", u.ID) // api.GLApi.CreateAccount adds this // TODO(adam): remove once we can consolidate OpenAPI docs / generated client

	glAddressOnce.Do(func() {
		log.Printf("Using %s as base GL address", conf.BasePath)
	})

	if *flagLocal {
		conf.BasePath = "http://localhost" + bind.HTTP("gl")
		return gl.NewAPIClient(conf)
	}
	if k8s.Inside() {
		conf.BasePath = "http://gl.apps.svc.cluster.local:8080"
		return gl.NewAPIClient(conf)
	}
	conf.BasePath = "https://api.moov.io/v1/gl"
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

	if *flagDebug {
		log.Printf("GL create account request URL: %s (status=%s): %v\n", resp.Request.URL.String(), resp.Status, err)
	}

	if err != nil {
		return nil, fmt.Errorf("problem creating GL account %s: %v", name, err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return &account, nil
}
