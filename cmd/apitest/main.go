// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/moov-io/api/internal/version"
)

var (
	defaultApiAddress = "https://api.moov.io/v1/"

	flagApiAddress = flag.String("address", defaultApiAddress, "Moov API address")
)

func main() {
	flag.Parse()

	fmt.Printf("Starting apitest %s\n", version.Version)

	os.Exit(0)
}
