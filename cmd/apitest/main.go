// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

// apitest is a cli tool (and Docker image) used for testing the Moov API.
// This tool is designed to operate against the production API and a local
// setup.
//
// With no arguments the contaier runs tests against the production API.
//
// apitest is not a stable tool. Please contact Moov developers if you intend to use this tool,
// otherwise we might change the tool (or remove it) without notice.
package main

import (
	"crypto/rand"
	"strings"
	"log"
	"encoding/hex"
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/moov-io/api/internal/version"

	moov "github.com/moov-io/go-client/client"
)

var (
	defaultApiAddress = "https://api.moov.io"

	flagApiAddress = flag.String("address", defaultApiAddress, "Moov API address")
)

func main() {
	flag.Parse()
	log.Printf("Starting apitest %s", version.Version)

	ctx := context.TODO()

	conf := moov.NewConfiguration()
	conf.BasePath = *flagApiAddress
	conf.UserAgent = fmt.Sprintf("apitest/%s", version.Version)
	conf.HTTPClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     1 * time.Minute,
		},
	}

	requestId := generateID()
	conf.AddDefaultHeader("X-Request-ID", requestId)
	log.Printf("Using X-Request-ID: %s", requestId)

	api := moov.NewAPIClient(conf)

	// Run tests
	if err := pingApps(ctx, api); err != nil {
		log.Fatal(err)
	}
}

// generateID creates a unique random string
func generateID() string {
	bs := make([]byte, 20)
	n, err := rand.Read(bs)
	if err != nil || n == 0 {
		return ""
	}
	return strings.ToLower(hex.EncodeToString(bs))
}

func pingApps(ctx context.Context, api *moov.APIClient) error {
	// ACH
	resp, err := api.MonitorApi.PingACH(ctx)
	if err != nil {
		return fmt.Errorf("ERROR: failed to ping ACH: %v", err)
	}
	resp.Body.Close()
	log.Println("ACH PONG")

	// auth
	resp, err = api.MonitorApi.PingAuth(ctx)
	if err != nil {
		return fmt.Errorf("ERROR: failed to ping auth: %v", err)
	}
	resp.Body.Close()
	log.Println("auth PONG")

	// paygate
	resp, err = api.MonitorApi.PingPaygate(ctx)
	if err != nil {
		return fmt.Errorf("ERROR: failed to ping paygate: %v", err)
	}
	resp.Body.Close()
	log.Println("paygaate PONG")
	return nil
}
