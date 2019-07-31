// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"

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
			// match transaction against posted ones on the account
			line := transactions[i].Lines[j]
			if v := fmt.Sprintf("USD %.2f", float32(line.Amount)/100.0); line.AccountId == accountId && v == amount {
				return nil // Matched Transaction
			}
		}
	}
	return fmt.Errorf("accounts: unable to find %q transaction for account=%s", amount, accountId)
}

func getMicroDepositsTransactions(ctx context.Context, api *moov.APIClient, accountId string, u *user, requestId string) ([]*moov.Transaction, error) {
	opts := &moov.GetAccountTransactionsOpts{
		Limit:      optional.NewFloat32(25),
		XRequestId: optional.NewString(requestId),
	}
	transactions, resp, err := api.AccountsApi.GetAccountTransactions(ctx, accountId, u.ID, opts)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("accounts: getMicroDeposits: %v", err)
	}
	var txs []*moov.Transaction
	for i := range transactions {
		if len(transactions[i].Lines) != 2 {
			continue
		}
		for j := range transactions[i].Lines {
			if transactions[i].Lines[j].Amount > 100 {
				continue // skip transaction as micro-deposits are under $1.00
			}
		}
		txs = append(txs, &transactions[i])
	}
	if len(txs) == 0 {
		return nil, errors.New("unable to find micro-deposit transaction")
	}
	return txs, nil
}
