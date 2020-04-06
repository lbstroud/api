// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

var (
	originRoutingNumber      = defaultRoutingNumber
	destinationRoutingNumber = defaultRoutingNumber
)

// setupGateway will create a Gateway object in PayGate that's used to setup the FileHeader
// in all ACH files sent through your ODFI. These are typically values given to you by them.
func setupGateway(ctx context.Context, api *moov.APIClient, u *user) (moov.Gateway, error) {
	req := moov.CreateGateway{
		Origin:          originRoutingNumber,
		OriginName:      "My Bank",
		Destination:     destinationRoutingNumber,
		DestinationName: "Their Bank",
	}
	opts := &moov.AddGatewayOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestID:      optional.NewString(generateID()),
	}
	gateway, resp, err := api.GatewaysApi.AddGateway(ctx, u.ID, req, opts)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return gateway, fmt.Errorf("problem setting up Gateway: %v", err)
	}
	return gateway, nil
}
