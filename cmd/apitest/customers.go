// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func attemptCustomerApproval(ctx context.Context, customersAdminAddress string, httpClient *http.Client, customerID, requestID string) error {
	if customersAdminAddress == "" {
		return nil
	}

	// 'OFAC' is the minimum status required for a Customer before Paygate will initiate a transfer
	body := strings.NewReader(`{"status": "OFAC", "comments": "approval from apitest"}`)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/customers/%s/status", customersAdminAddress, customerID), body)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return fmt.Errorf("problem updating customer=%s status=%v", customerID, resp.Status)
	}
	return nil
}
