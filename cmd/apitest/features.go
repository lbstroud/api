package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type featureFlags struct {
	AccountsCallsDisabled  bool `json:"accountsCallsDisabled"`
	CustomersCallsDisabled bool `json:"customersCallsDisabled"`
}

func grabPaygateFeatures(paygateAdminAddress string, httpClient *http.Client) (*featureFlags, error) {
	u, err := url.Parse(paygateAdminAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", paygateAdminAddress, err)
	}
	u.Path = "/features"

	resp, err := httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to load feature flags: %v", err)
	}
	if resp.StatusCode > 200 {
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	var flags featureFlags
	if err := json.NewDecoder(resp.Body).Decode(&flags); err != nil {
		return nil, fmt.Errorf("failed to read feature flags: %v", err)
	}
	return &flags, nil
}
