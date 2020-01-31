// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func attemptCustomerApproval(ctx context.Context, address string, customerID string) error {
	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", address, err)
	}
	u.Path += fmt.Sprintf("/customers/%s/status", customerID)

	// 'OFAC' is the minimum status required for a Customer before Paygate will initiate a transfer
	body := strings.NewReader(`{"status": "OFAC", "comments": "approval from apitest"}`)
	req, err := http.NewRequest("PUT", u.String(), body)
	if err != nil {
		return err
	}

	resp, err := adminHTTPClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		bs, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("problem updating customer=%s status=%v: %v", customerID, resp.Status, string(bs))
	}
	return nil
}
