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
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/moov-io/api/internal/version"
	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
)

var (
	defaultApiAddress = "https://api.moov.io"

	flagApiAddress = flag.String("address", defaultApiAddress, "Moov API address")
	flagLocal      = flag.Bool("local", false, "Use local HTTP addresses")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds | log.Lshortfile)
	log.Printf("Starting apitest %s", version.Version)

	ctx := context.TODO()

	conf := moov.NewConfiguration()
	if *flagLocal {
		// If '-local and -address <foo>' use <foo>
		if addr := *flagApiAddress; addr != defaultApiAddress {
			conf.BasePath = addr
		} else {
			conf.BasePath = "http://localhost"
		}
	} else {
		conf.BasePath = *flagApiAddress
	}
	log.Printf("Using %s as base API address", conf.BasePath)
	conf.UserAgent = fmt.Sprintf("moov apitest/%s", version.Version)

	// setup HTTP client
	conf.HTTPClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     1 * time.Minute,
		},
	}
	if *flagLocal {
		tr := conf.HTTPClient.Transport
		conf.HTTPClient.Transport = &localPathTransport{
			tr: tr,
		}
	}

	requestId := generateID()
	conf.AddDefaultHeader("X-Request-ID", requestId)
	log.Printf("Using X-Request-ID: %s", requestId)

	api := moov.NewAPIClient(conf)

	// Run tests
	if err := pingApps(ctx, api, requestId); err != nil {
		log.Fatal(err)
	}

	// Create our random user
	user, err := createUser(ctx, api, requestId)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created user %s (email: %s)", user.ID, user.Email)

	// Add moov_auth cookie on every request from now on
	setMoovCookie(conf, user.Cookie)

	// Verify Cookie works
	if err := verifyUserIsLoggedIn(ctx, api, user, requestId); err != nil {
		log.Fatal(err)
	}
	log.Printf("Cookie works for user %s", user.ID)

	oauthToken, err := createOAuthToken(ctx, api, user, requestId)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created OAuth access token, expires in %v", oauthToken.Expires())

	// Create Originator Depository
	origFI := &fiInfo{Name: "orig bank", AccountNumber: "132", RoutingNumber: "121042882"}
	origDep, err := createDepository(ctx, api, user, origFI, requestId)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created Originator Depository (id=%s) for user", origDep.Id)

	// Create Originator
	orig, err := createOriginator(ctx, api, origDep.Id, requestId)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created Originator (id=%s) for user", orig.Id)

	// Create Customer Depository
	custFI := &fiInfo{Name: "cust bank", AccountNumber: "5211", RoutingNumber: "231380104"}
	custDep, err := createDepository(ctx, api, user, custFI, requestId)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created Customer Depository (id=%s) for user", custDep.Id)

	// Create Customer
	cust, err := createCustomer(ctx, api, user, custDep.Id, requestId)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created Customer (id=%s) for user", cust.Id)

	// Create Transfer
	tx, err := createTransfer(ctx, api, cust, orig, amount(), requestId)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Created %s transfer (id=%s) for user", tx.Amount, tx.Id)
}

// amount returns a random amount in string form accepted by the Moov API
func amount() string {
	n := float64(randSource.Int63()%2500) / 10.2 // max out at $250
	return fmt.Sprintf("USD %.2f", n)
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

func pingApps(ctx context.Context, api *moov.APIClient, requestId string) error {
	// ACH
	resp, err := api.MonitorApi.PingACH(ctx, &moov.PingACHOpts{
		XRequestId: optional.NewString(requestId),
	})
	if err != nil {
		return fmt.Errorf("ERROR: failed to ping ACH: %v", err)
	}
	resp.Body.Close()
	log.Println("ACH PONG")

	// auth
	resp, err = api.MonitorApi.PingAuth(ctx, &moov.PingAuthOpts{
		XRequestId: optional.NewString(requestId),
	})
	if err != nil {
		return fmt.Errorf("ERROR: failed to ping auth: %v", err)
	}
	resp.Body.Close()
	log.Println("auth PONG")

	// paygate
	resp, err = api.MonitorApi.PingPaygate(ctx, &moov.PingPaygateOpts{
		XRequestId: optional.NewString(requestId),
	})
	if err != nil {
		return fmt.Errorf("ERROR: failed to ping paygate: %v", err)
	}
	resp.Body.Close()
	log.Println("paygate PONG")
	return nil
}
