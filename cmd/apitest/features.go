package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type featureFlags struct {
	AccountsCallsDisabled  bool `json:"accountsCallsDisabled"`
	CustomersCallsDisabled bool `json:"customersCallsDisabled"`
}

func grabPaygateFeatures(flagLocal *bool, paygateAdminAddress string, httpClient *http.Client) (*featureFlags, error) {
	if !*flagLocal && !*flagLocalDev {
		return &featureFlags{
			AccountsCallsDisabled:  true,
			CustomersCallsDisabled: true,
		}, nil
	}

	u, err := url.Parse(paygateAdminAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", paygateAdminAddress, err)
	}
	u.Path = "/features"

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("problem creating request: %v", err)
	}
	req.Header.Set("Origin", "https://moov.io")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to load feature flags: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 200 {
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	if err := checkCORSHeaders(resp); err != nil {
		return nil, fmt.Errorf("get paygate features: %v", err)
	}

	var flags featureFlags
	if err := json.NewDecoder(resp.Body).Decode(&flags); err != nil {
		return nil, fmt.Errorf("failed to read feature flags: %v", err)
	}

	if *flagDebug {
		log.Printf("[DEBUG] feature flags: %#v", flags)
	}

	return &flags, nil
}
