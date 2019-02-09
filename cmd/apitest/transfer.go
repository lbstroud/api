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
	req := moov.CreateDepository{
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
	req := moov.CreateOriginator{
		DefaultDepository: depId,
		Identification:    "123456789", // SSN
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

func createCustomer(ctx context.Context, api *moov.APIClient, u *user, depId, requestId string) (moov.Customer, error) {
	req := moov.CreateCustomer{
		Email:             fmt.Sprintf("%s+apitest@moov.io", u.Name),
		DefaultDepository: depId,
		Metadata:          "Jane Doe",
	}
	cust, resp, err := api.CustomersApi.AddCustomers(ctx, req, &moov.AddCustomersOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if err != nil {
		return cust, fmt.Errorf("problem creating customer: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return cust, nil
}

func createTransfer(ctx context.Context, api *moov.APIClient, cust moov.Customer, orig moov.Originator, amount string, requestId string) (moov.Transfer, error) {
	req := moov.CreateTransfer{
		TransferType:           "Push",
		Amount:                 amount,
		Originator:             orig.Id,
		OriginatorDepository:   orig.DefaultDepository,
		Customer:               cust.Id,
		CustomerDepository:     cust.DefaultDepository,
		Description:            "apitest transfer",
		StandardEntryClassCode: "PPD",
	}
	tx, resp, err := api.TransfersApi.AddTransfer(ctx, req, &moov.AddTransferOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if err != nil {
		return tx, fmt.Errorf("problem creating transfer: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Delete the transfer (and underlying file)
	resp, err = api.TransfersApi.DeleteTransferByID(ctx, tx.Id, &moov.DeleteTransferByIDOpts{
		XRequestId: optional.NewString(requestId),
	})
	if err != nil {
		return tx, fmt.Errorf("problem deleting transfer: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}

	return tx, nil
}
