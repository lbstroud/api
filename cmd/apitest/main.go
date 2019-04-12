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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/moov-io/api/cmd/apitest/local"
	"github.com/moov-io/api/internal/version"
	gl "github.com/moov-io/gl/client"
	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
	"go4.org/syncutil"
)

var (
	defaultApiAddress = "https://api.moov.io"

	flagApiAddress = flag.String("address", defaultApiAddress, "Moov API address")
	flagDebug      = flag.Bool("debug", false, "Enable Debug logging.")
	flagLocal      = flag.Bool("local", false, "Use local HTTP addresses (e.g. 'go run')")
	flagLocalDev   = flag.Bool("dev", false, "Use tilt local HTTP address")

	// Business logic flags
	flagACHType = flag.String("ach.type", "PPD", "ACH Service Class Code (SEC) to use. Options: PPD, IAT")
	flagOAuth   = flag.Bool("oauth", false, "Use OAuth instead of cookie auth")

	flagFakeData       = flag.Bool("fake-data", false, "Generate fake data (instead of one transfer) across several routing numbers, customers, and originators")
	flagFakeIterations = flag.Int("fake-data.iterations", 1000, "How many users and transfers to create")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds | log.Lshortfile)
	log.Printf("Starting apitest %s", version.Version)

	ctx := context.TODO()

	// Run tests
	if err := pingApps(ctx); err != nil {
		log.Fatalf("FAILURE: %v", err)
	}

	// Run either one or many iterations
	if *flagFakeData {
		fmt.Println("") // add buffer space in output

		var wg sync.WaitGroup
		gate := syncutil.NewGate(10) // allow 10 concurrent iterations
		for i := 0; i < *flagFakeIterations; i++ {
			wg.Add(1)
			gate.Start()
			go func() {
				iterate(ctx)
				gate.Done()
				wg.Done()
			}()
		}
		wg.Wait()
	} else {
		iterate(ctx) // just one user and transfer
	}
}

var apiAddressOnce sync.Once

func makeConfiguration() *moov.Configuration {
	conf := moov.NewConfiguration()
	if *flagLocal {
		// If '-local and -address <foo>' use <foo>
		if addr := *flagApiAddress; addr != defaultApiAddress {
			conf.BasePath = addr
		} else {
			conf.BasePath = "http://localhost"
		}
	} else {
		if *flagLocalDev {
			conf.BasePath = "http://localhost:9000"
		} else {
			conf.BasePath = *flagApiAddress
		}
	}
	apiAddressOnce.Do(func() {
		log.Printf("Using %s as base API address", conf.BasePath)
	})
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
		conf.HTTPClient.Transport = &local.Transport{
			Underlying: tr,
			Debug:      *flagDebug,
		}
	}
	return conf
}

func pingApps(ctx context.Context) error {
	requestId := generateID()
	conf := makeConfiguration()
	conf.AddDefaultHeader("X-Request-ID", requestId)
	api := moov.NewAPIClient(conf)

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

type iteration struct {
	user       *user
	oauthToken OAuthToken

	originator           moov.Originator
	originatorAccount    *gl.Account
	originatorDepository moov.Depository

	customer           moov.Customer
	customerAccount    *gl.Account
	customerDepository moov.Depository

	transfer moov.Transfer
}

var logmu sync.Mutex // guards iterate(..) logging

func iterate(ctx context.Context) *iteration {
	var lines []string
	debugLogger := func(tpl string, args ...interface{}) {
		if *flagFakeData {
			lines = append(lines, fmt.Sprintf(tpl, args...))
		} else {
			log.Printf(tpl, args...)
		}
	}
	errLogger := func(tpl string, args ...interface{}) {
		if *flagFakeData {
			lines = append(lines, fmt.Sprintf(tpl, args...))
		} else {
			log.Fatalf(tpl, args...)
		}
	}
	defer func() { // after an iteration print all logs at once
		logmu.Lock()
		defer logmu.Unlock()
		for i := range lines {
			log.Println(lines[i])
		}
		fmt.Println("")
	}()

	requestId := generateID()
	conf := makeConfiguration()
	conf.AddDefaultHeader("X-Request-ID", requestId)
	debugLogger("Using X-Request-ID: %s", requestId)
	api := moov.NewAPIClient(conf)

	// Create our random user
	user, err := createUser(ctx, api, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: Created user %s (email: %s)", user.ID, user.Email)

	// Add auth cookie and userId on every request from now on
	setMoovAuthCookie(conf, user)

	// Verify Cookie works
	if err := verifyUserIsLoggedIn(ctx, api, user, requestId); err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: Cookie works for user %s", user.ID)

	oauthToken, err := createOAuthToken(ctx, api, user, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}
	if v := os.Getenv("TRAVIS_OS_NAME"); v != "" {
		debugLogger("SUCCESS: Created OAuth access token, expires in %v", oauthToken.Expires())
	}
	debugLogger("SUCCESS: Created OAuth access token (%s), expires in %v", oauthToken.Access(), oauthToken.Expires())

	if *flagOAuth {
		debugLogger("Using OAuth for all requests now.")

		removeMoovAuthCookie(conf) // we only want OAuth credentials on requests
		setMoovOAuthToken(conf, oauthToken)
	}

	glClient := setupGLClient(user)

	// Create Originator GL account
	// This is only needed because our GL setup (for apitest's default environment) doesn't have
	// accounts backing it. We create on in our GL service for each test run.
	origAcct, err := createGLAccount(ctx, glClient, user, "from account", requestId) // TODO(adam): need to add balance, paygate will check
	if err != nil {
		errLogger("FAILURE: %v", err)
	}

	// Create Originator Depository
	origDep, err := createDepository(ctx, api, user, origAcct, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: Created Originator Depository (id=%s) for user", origDep.Id)

	// Create Originator
	orig, err := createOriginator(ctx, api, origDep.Id, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: Created Originator (id=%s) for user", orig.Id)

	// Create Customer GL account
	custAcct, err := createGLAccount(ctx, glClient, user, "to account", requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}

	// Create Customer Depository
	custDep, err := createDepository(ctx, api, user, custAcct, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: Created Customer Depository (id=%s) for user", custDep.Id)

	// Create Customer
	cust, err := createCustomer(ctx, api, user, custDep.Id, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: Created Customer (id=%s) for user", cust.Id)

	// Create Transfer
	tx, err := createTransfer(ctx, api, cust, orig, amount(), requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: Created %s transfer (id=%s) for user", tx.Amount, tx.Id)

	// Attempt a Failed login
	if err := attemptFailedLogin(ctx, api, requestId); err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: invalid login credentials were rejected")

	// Attempt a Failed OAuth2 auth check
	if err := attemptFailedOAuth2Login(ctx, api, requestId); err != nil {
		errLogger("FAILURE: %v", err)
	}
	debugLogger("SUCCESS: invalid OAuth2 access token was rejected")

	return &iteration{
		user:                 user,
		oauthToken:           oauthToken,
		originator:           orig,
		originatorAccount:    origAcct,
		originatorDepository: origDep,
		customer:             cust,
		customerAccount:      custAcct,
		customerDepository:   custDep,
		transfer:             tx,
	}
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
