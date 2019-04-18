// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"

	"github.com/moov-io/ach"
	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

type fiInfo struct {
	// Name is a human readable name for the financial institution
	Name string

	AccountNumber string
	RoutingNumber string
}

// TODO(adam): on -fake-data we need to randomize all data, but keep the existing values on single Transfer creations

func createDepository(ctx context.Context, api *moov.APIClient, u *user, account *moov.Account, requestId string) (moov.Depository, error) {
	req := moov.CreateDepository{
		BankName:      "Moov Bank",
		AccountNumber: account.AccountNumber,
		RoutingNumber: account.RoutingNumber,
		Holder:        u.Name,
		HolderType:    "Individual",
		Type:          account.Type,
	}
	dep, resp, err := api.DepositoriesApi.AddDepository(ctx, req, &moov.AddDepositoryOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return dep, fmt.Errorf("problem creating depository (name: %q) for user (userId=%s): %v", account.Name, u.ID, err)
	}

	// verify with (known, fixed values) micro-deposits
	if err := verifyDepository(ctx, api, dep, requestId); err != nil {
		return dep, fmt.Errorf("problem verifying depository (name: %q) for user (userId=%s): %v", account.Name, u.ID, err)
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
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("problem starting micro deposits: %v", err)
	}

	// confirm micro deposits
	resp, err = api.DepositoriesApi.ConfirmMicroDeposits(ctx, dep.Id, knownDepositAmounts, &moov.ConfirmMicroDepositsOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("problem verifying micro deposits: %v", err)
	}
	return nil
}

func createOriginator(ctx context.Context, api *moov.APIClient, depId, requestId string) (moov.Originator, error) {
	req := moov.CreateOriginator{
		DefaultDepository: depId,
		Identification:    "123456789", // SSN
		Metadata:          "Acme Corp",
	}
	orig, resp, err := api.OriginatorsApi.AddOriginator(ctx, req, &moov.AddOriginatorOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return orig, fmt.Errorf("problem creating originator: %v", err)
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
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return cust, fmt.Errorf("problem creating customer: %v", err)
	}
	return cust, nil
}

func createTransfer(ctx context.Context, api *moov.APIClient, cust moov.Customer, orig moov.Originator, amount string, requestId string) (moov.Transfer, error) {
	req := moov.CreateTransfer{
		TransferType:         "Push",
		Amount:               amount,
		Originator:           orig.Id,
		OriginatorDepository: orig.DefaultDepository,
		Customer:             cust.Id,
		CustomerDepository:   cust.DefaultDepository,
		Description:          "apitest transfer",
	}
	switch *flagACHType {
	case ach.IAT:
		req.StandardEntryClassCode = "IAT"
		req.IATDetail = createIATDetail(cust, orig)
	case ach.PPD:
		req.StandardEntryClassCode = "PPD"
	case ach.WEB:
		req.StandardEntryClassCode = "WEB"
		req.WEBDetail = createWEBDetail()

	}
	tx, resp, err := api.TransfersApi.AddTransfer(ctx, req, &moov.AddTransferOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestId:      optional.NewString(requestId),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return tx, fmt.Errorf("problem creating transfer: %v", err)
	}
	if !*flagFakeData {
		// Delete the transfer (and underlying file) since we're only making one Transfer
		resp, err = api.TransfersApi.DeleteTransferByID(ctx, tx.Id, &moov.DeleteTransferByIDOpts{
			XRequestId: optional.NewString(requestId),
		})
		if resp != nil {
			resp.Body.Close()
		}
		if err != nil {
			return tx, fmt.Errorf("problem deleting transfer: %v", err)
		}
	}
	return tx, nil
}

func createIATDetail(cust moov.Customer, orig moov.Originator) moov.IatDetail {
	return moov.IatDetail{
		OriginatorName:               orig.Metadata,
		OriginatorAddress:            "123 1st st",
		OriginatorCity:               "anytown",
		OriginatorState:              "PA",
		OriginatorPostalCode:         "12345",
		OriginatorCountryCode:        "US",
		ODFIName:                     "my bank",
		ODFIIDNumberQualifier:        "01",
		ODFIIdentification:           "2",
		ODFIBranchCurrencyCode:       "USD",
		ReceiverName:                 cust.Metadata,
		ReceiverAddress:              "321 2nd st",
		ReceiverCity:                 "othertown",
		ReceiverState:                "GB",
		ReceiverPostalCode:           "54321",
		ReceiverCountryCode:          "GB",
		RDFIName:                     "their bank",
		RDFIIDNumberQualifier:        "01",
		RDFIIdentification:           "4",
		RDFIBranchCurrencyCode:       "GBP",
		ForeignCorrespondentBankName: "their bank",
		ForeignCorrespondentBankIDNumberQualifier: "5",
		ForeignCorrespondentBankIDNumber:          "6",
		ForeignCorrespondentBankBranchCountryCode: "GB",
	}
}

func createWEBDetail() moov.WebDetail {
	return moov.WebDetail{
		PaymentInformation: "apitest payment",
		PaymentType:        "single",
	}
}
