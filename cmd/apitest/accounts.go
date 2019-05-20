// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

func createAccount(ctx context.Context, api *moov.APIClient, u *user, name, requestId string) (*moov.Account, error) {
	req := moov.CreateAccount{
		CustomerId: u.ID,
		Name:       name,
		Type:       "Savings",
		Balance:    1000 * 100, // $1,000
	}
	opts := &moov.CreateAccountOpts{
		XRequestId: optional.NewString(requestId),
	}
	account, resp, err := api.AccountsApi.CreateAccount(ctx, u.ID, req, opts)
	if *flagDebug && resp != nil {
		log.Printf("problem creating account request URL: %s (status=%s): %v\n", resp.Request.URL.String(), resp.Status, err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("problem creating account %q: %v", name, err)
	}
	return &account, nil
}

// Verify accountId and Transaction exist of a given amount (used to double check transfers).
func checkTransactions(ctx context.Context, api *moov.APIClient, accountId string, u *user, amount string, requestId string) error {
	opts := &moov.GetAccountTransactionsOpts{
		Limit:      optional.NewFloat32(25),
		XRequestId: optional.NewString(requestId),
	}
	transactions, resp, err := api.AccountsApi.GetAccountTransactions(ctx, accountId, u.ID, opts)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("accounts: GetAccountTransactions: %v", err)
	}
	for i := range transactions {
		for j := range transactions[i].Lines {
			line := transactions[i].Lines[j]
			switch {
			case strings.EqualFold(line.Purpose, "achdebit"):
				// We expect the amount to be negative then invert that
				amount = "USD -" + amount[len("USD "):]
			case strings.EqualFold(line.Purpose, "achcredit") && strings.Contains(amount, "USD -"):
				// Invert negative to positive
				amount = "USD " + amount[len("USD -"):]
			}
			// match transaction against posted ones on the account
			if v := fmt.Sprintf("USD %.2f", float32(line.Amount)/100.0); line.AccountId == accountId && v == amount {
				return nil // Matched Transaction
			}
		}
	}
	return fmt.Errorf("accounts: unable to find %q transaction for account=%s", amount, accountId)
}
