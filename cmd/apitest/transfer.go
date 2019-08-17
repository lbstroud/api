// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/moov-io/ach"
	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

func createDepository(ctx context.Context, api *moov.APIClient, u *user, account *moov.Account, requestID string) (moov.Depository, error) {
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
		XRequestID:      optional.NewString(requestID),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return dep, fmt.Errorf("problem creating depository (name: %q) for user (userID=%s): %v", account.Name, u.ID, err)
	}

	// verify with (known, fixed values) micro-deposits
	if err := verifyDepository(ctx, api, account.ID, dep, u, requestID); err != nil {
		return dep, fmt.Errorf("problem verifying depository (name: %q) for user (userID=%s): %v", account.Name, u.ID, err)
	}

	return dep, nil
}

func verifyDepository(ctx context.Context, api *moov.APIClient, accountID string, dep moov.Depository, u *user, requestID string) error {
	// start micro deposits
	resp, err := api.DepositoriesApi.InitiateMicroDeposits(ctx, dep.ID, &moov.InitiateMicroDepositsOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestID:      optional.NewString(requestID),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("problem starting micro deposits: %v", err)
	}

	// Grab the micro-deposit transactions
	var microDepositTransactions []*moov.Transaction
	for i := 0; i < 3; i++ {
		microDepositTransactions, err = getMicroDepositsTransactions(ctx, api, accountID, u, requestID)
		if len(microDepositTransactions) > 0 {
			time.Sleep(250 * time.Millisecond)
			break
		}
	}
	if err != nil {
		return fmt.Errorf("problem getting micro-deposit transaction: %v", err)
	}
	var microDeposits moov.Amounts
	for i := range microDepositTransactions {
		microDeposits.Amounts = append(microDeposits.Amounts, fmt.Sprintf("USD %.2f", microDepositTransactions[i].Lines[0].Amount/100))
	}
	if *flagDebug {
		log.Printf("verifying Depository with micro-deposit amounts: %s", strings.Join(microDeposits.Amounts, ", "))
	}

	// confirm micro deposits
	resp, err = api.DepositoriesApi.ConfirmMicroDeposits(ctx, dep.ID, microDeposits, &moov.ConfirmMicroDepositsOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestID:      optional.NewString(requestID),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("problem verifying micro deposits: %v", err)
	}
	return nil
}

func createOriginator(ctx context.Context, api *moov.APIClient, depId, requestID string) (moov.Originator, error) {
	first, _ := name()
	req := moov.CreateOriginator{
		DefaultDepository: depId,
		Identification:    "123456789",
		Metadata:          fmt.Sprintf("%s Corp", first),
	}
	orig, resp, err := api.OriginatorsApi.AddOriginator(ctx, req, &moov.AddOriginatorOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestID:      optional.NewString(requestID),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return orig, fmt.Errorf("problem creating originator: %v", err)
	}
	return orig, nil
}

func createReceiver(ctx context.Context, api *moov.APIClient, u *user, depId, requestID string) (moov.Receiver, error) {
	req := moov.CreateReceiver{
		Email:             email(name()), // new random email address
		DefaultDepository: depId,
		Metadata:          u.Name,
	}
	receiver, resp, err := api.ReceiversApi.AddReceivers(ctx, req, &moov.AddReceiversOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestID:      optional.NewString(requestID),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return receiver, fmt.Errorf("problem creating receiver: %v", err)
	}
	return receiver, nil
}

func createTransfer(ctx context.Context, api *moov.APIClient, receiver moov.Receiver, orig moov.Originator, amount string, requestID string) (moov.Transfer, error) {
	req := moov.CreateTransfer{
		TransferType:         "Push",
		Amount:               amount,
		Originator:           orig.ID,
		OriginatorDepository: orig.DefaultDepository,
		Receiver:             receiver.ID,
		ReceiverDepository:   receiver.DefaultDepository,
		Description:          fmt.Sprintf("apitest transfer to %s", receiver.Metadata),
	}
	switch *flagACHType {
	case ach.IAT:
		req.StandardEntryClassCode = "IAT"
		req.IATDetail = createIATDetail(receiver, orig)
	case ach.PPD:
		req.StandardEntryClassCode = "PPD"
	case ach.WEB:
		req.StandardEntryClassCode = "WEB"
		req.WEBDetail = createWEBDetail()

	}
	tx, resp, err := api.TransfersApi.AddTransfer(ctx, req, &moov.AddTransferOpts{
		XIdempotencyKey: optional.NewString(generateID()),
		XRequestID:      optional.NewString(requestID),
	})
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return tx, fmt.Errorf("problem creating %s transfer: %v", amount, err)
	}
	if *flagCleanup {
		// Delete the transfer (and underlying file) since we're only making one Transfer
		resp, err = api.TransfersApi.DeleteTransferByID(ctx, tx.ID, &moov.DeleteTransferByIDOpts{
			XRequestID: optional.NewString(requestID),
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

func createIATDetail(receiver moov.Receiver, orig moov.Originator) moov.IatDetail {
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
		ReceiverName:                 receiver.Metadata,
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
