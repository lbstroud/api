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
	fmt.Printf("Starting apitest %s\n", version.Version)

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
	api := moov.NewAPIClient(conf)

	resp, err := api.MonitorApi.PingACH(ctx)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	fmt.Printf("PONG - %s\n", conf.BasePath)
}
