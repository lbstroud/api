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

type fiInfo struct {
	// Name is a human readable name for the financial institution
	Name string

	AccountNumber string
	RoutingNumber string
}

func createDepository(ctx context.Context, api *moov.APIClient, u *user, fi *fiInfo, requestId string) (moov.Depository, error) {
	req := moov.Depository{
		BankName:      fi.Name,
		AccountNumber: fi.AccountNumber,
		RoutingNumber: fi.RoutingNumber,
		Holder:        u.Name,
		HolderType:    "Individual",
		Type:          "Checking",
	}
	dep, resp, err := api.DepositoriesApi.AddDepository(ctx, req, &moov.AddDepositoryOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if err != nil {
		return dep, fmt.Errorf("problem creating depository (name: %s) for user (userId=%s): %v", fi.Name, u.ID, err)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// verify with (known, fixed values) micro-deposits
	if err := verifyDepository(ctx, api, dep, requestId); err != nil {
		return dep, fmt.Errorf("problem verifying depository (name: %s) for user (userId=%s): %v", fi.Name, u.ID, err)
	}

	return dep, nil
}

var (
	knownDepositAmounts = moov.Amounts{
		Amounts: []string{"USD 0.01", "USD 0.03"}, // from paygate, microDeposits.go
	}
)

func verifyDepository(ctx context.Context, api *moov.APIClient, dep moov.Depository, requestId string) error {
	// start micro deposits
	resp, err := api.DepositoriesApi.InitiateMicroDeposits(ctx, dep.Id, &moov.InitiateMicroDepositsOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if err != nil {
		return fmt.Errorf("problem starting micro deposits: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// confirm micro deposits
	resp, err = api.DepositoriesApi.ConfirmMicroDeposits(ctx, dep.Id, knownDepositAmounts, &moov.ConfirmMicroDepositsOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if err != nil {
		return fmt.Errorf("problem verifying micro deposits: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return nil
}

func createOriginator(ctx context.Context, api *moov.APIClient, depId, requestId string) (moov.Originator, error) {
	req := moov.Originator{
		DefaultDepository: depId,
		Identification:    "123-45-6789", // SSN
	}
	orig, resp, err := api.OriginatorsApi.AddOriginator(ctx, req, &moov.AddOriginatorOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if err != nil {
		return orig, fmt.Errorf("problem creating originator: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return orig, nil
}

// Create Customer Depository
// curl -s -o /tmp/paygate/custDep.json -XPOST -H "x-user-id: $userId" -H "x-request-id: $requestId" http://localhost:8082/depositories --data '{"bankName":"cust bank", "holder": "you", "holderType": "individual", "type": "Checking", "routingNumber": "231380104", "accountNumber": "451"}'

// Verify Customer Depository
// curl -s -o /tmp/paygate/custDep-initverify.json -XPOST -H "x-user-id: $userId" -H "x-request-id: $requestId" http://localhost:8082/depositories/"$custDepId"/micro-deposits --data '{"amounts": ["USD 0.01", "USD 0.03"]}'
// curl -s -o /tmp/paygate/custDep-verify.json -XPOST -H "x-user-id: $userId" -H "x-request-id: $requestId" http://localhost:8082/depositories/"$custDepId"/micro-deposits/confirm --data '{"amounts": ["USD 0.01", "USD 0.03"]}'

// Create Customer
// curl -s -o /tmp/paygate/cust.json -XPOST -H "x-user-id: $userId" -H "x-request-id: $requestId" http://localhost:8082/customers --data "{\"defaultDepository\": \"$custDepId\", \"email\": \"test@moov.io\"}"

// Create Transfer
// curl -s -o /tmp/paygate/transfer.json -XPOST -H "x-user-id: $userId" -H "x-request-id: $requestId" http://localhost:8082/transfers --data "{\"transferType\": \"push\", \"amount\": \"USD 78.54\", \"originator\": \"$orig\", \"originatorDepository\": \"$origDepId\", \"customer\": \"$cust\", \"customerDepository\": \"$custDepId\", \"description\": \"test payment\", \"standardEntryClassCode\": \"PPD\"}"
