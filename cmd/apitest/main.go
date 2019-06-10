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

	flagFakeData       = flag.Bool("fake-data", false, "Generate fake data (instead of one transfer) across several routing numbers, receivers, and originators")
	flagFakeIterations = flag.Int("fake-data.iterations", 1000, "How many users and transfers to create")

	flagVerifyTransfers    = flag.String("verify-transfers.dir", "", "Verify the created transfers exist in the given directory of ACH files")
	flagVerifyInitialSleep = flag.Duration("verify-transfers.initial-sleep", 1*time.Minute, "Duration to sleep so paygate can process and merge all transfers")

	flagVerifyAccounts = flag.Bool("verify.accounts", true, "Verify account balances and posted transactions, see ACCOUNTS_CALLS_DISABLED in paygate")
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

	// If we're going to verify we need the directory to be empty beforehand
	if *flagVerifyTransfers != "" && !verifyDirIsEmpty(*flagVerifyTransfers) {
		log.Fatalf("FAILURE: verify directory %s is not empty", *flagVerifyTransfers)
	}

	var mu sync.Mutex
	var iterations []*iteration

	// Run either one or many iterations
	if *flagFakeData {
		fmt.Println("") // add buffer space in output

		var wg sync.WaitGroup
		gate := syncutil.NewGate(10) // allow 10 concurrent iterations
		for i := 0; i < *flagFakeIterations; i++ {
			wg.Add(1)
			gate.Start()
			go func() {
				if iter := iterate(ctx); iter != nil {
					mu.Lock()
					iterations = append(iterations, iter)
					mu.Unlock()
				}
				gate.Done()
				wg.Done()
			}()
		}
		wg.Wait()
	} else {
		if iter := iterate(ctx); iter != nil {
			iterations = append(iterations, iter) // just one user and transfer
		}
	}

	// Verify every transfer we made exists
	if *flagVerifyTransfers != "" {
		if len(iterations) == 0 {
			log.Fatalf("FAILURE: unable to create any transfers, see above output logs for errors")
		}
		log.Printf("Sleeping for %v to let paygate collect and merge %d transfers", flagVerifyInitialSleep, len(iterations))
		time.Sleep(*flagVerifyInitialSleep)
		if err := verifyTransfersWereMerged(*flagVerifyTransfers, iterations); err != nil {
			log.Fatalf("FAILURE: %v", err)
		}
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
	oauthToken moov.OAuth2Token

	originator           moov.Originator
	originatorAccount    *moov.Account
	originatorDepository moov.Depository

	receiver           moov.Receiver
	receiverAccount    *moov.Account
	receiverDepository moov.Depository

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
		return nil
	}
	debugLogger("SUCCESS: Created user %s (email: %s)", user.ID, user.Email)

	// Add auth cookie and userId on every request from now on
	setMoovAuthCookie(conf, user)

	// Verify Cookie works
	if err := verifyUserIsLoggedIn(ctx, api, user, requestId); err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: Cookie works for user %s", user.ID)

	oauthToken, err := createOAuthToken(ctx, api, user, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	expiresIn, _ := time.ParseDuration(fmt.Sprintf("%ds", oauthToken.ExpiresIn))
	if v := os.Getenv("TRAVIS_OS_NAME"); v != "" {
		// Hide our OAuth2 access_token from TravisCI logs...
		debugLogger("SUCCESS: Created OAuth access token, expires in %v", expiresIn)
	}
	debugLogger("SUCCESS: Created OAuth access token (%s), expires in %v", oauthToken.AccessToken, expiresIn)

	if *flagOAuth {
		debugLogger("Using OAuth for all requests now.")

		removeMoovAuthCookie(conf) // we only want OAuth credentials on requests
		setMoovOAuthToken(conf, oauthToken)
	}

	// Create Originator Account
	// We create these accounts because they won't exist in the Accounts service already. (We're using fake data/accounts.)
	origAcct, err := createAccount(ctx, api, user, "from account", requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}

	// Create Originator Depository
	origDep, err := createDepository(ctx, api, user, origAcct, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: Created Originator Depository (id=%s) for user", origDep.Id)

	// Create Originator
	orig, err := createOriginator(ctx, api, origDep.Id, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: Created Originator (id=%s) for user", orig.Id)

	// Create Receiver Account
	receiverAcct, err := createAccount(ctx, api, user, "to account", requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}

	// Create Receiver Depository
	receiverDep, err := createDepository(ctx, api, user, receiverAcct, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: Created Receiver Depository (id=%s) for user", receiverDep.Id)

	// Create Receiver
	receiver, err := createReceiver(ctx, api, user, receiverDep.Id, requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: Created Receiver (id=%s) for user", receiver.Id)

	// Create Transfer
	tx, err := createTransfer(ctx, api, receiver, orig, amount(), requestId)
	if err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: Created %s transfer (id=%s) for user", tx.Amount, tx.Id)

	// Verify the Transaction was posted
	if *flagVerifyAccounts {
		if err := checkTransactions(ctx, api, origAcct.Id, user, tx.Amount, requestId); err != nil {
			errLogger("FAILURE: %v", err)
			return nil
		}
		if err := checkTransactions(ctx, api, receiverAcct.Id, user, tx.Amount, requestId); err != nil {
			errLogger("FAILURE: %v", err)
			return nil
		}
		debugLogger("SUCCESS: Matched transactions on accounts")
	}

	// Attempt a Failed login
	if err := attemptFailedLogin(ctx, api, requestId); err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: invalid login credentials were rejected")

	// Attempt a Failed OAuth2 auth check
	if err := attemptFailedOAuth2Login(ctx, api, requestId); err != nil {
		errLogger("FAILURE: %v", err)
		return nil
	}
	debugLogger("SUCCESS: invalid OAuth2 access token was rejected")

	return &iteration{
		user:                 user,
		oauthToken:           *oauthToken,
		originator:           orig,
		originatorAccount:    origAcct,
		originatorDepository: origDep,
		receiver:             receiver,
		receiverAccount:      receiverAcct,
		receiverDepository:   receiverDep,
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
