// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/moov-io/api/cmd/apitest/local"
	"github.com/moov-io/api/internal/version"
)

var (
	flagHttpAddr = flag.String("http.addr", ":9000", "HTTP listen address")
	flagDebug    = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds | log.Lshortfile)
	log.Printf("Starting moov localproxy %s", version.Version)

	u, _ := url.Parse("http://localhost") // no port, local.Transport overrides that
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = &local.Transport{
		Underlying: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     1 * time.Minute,
		},
		Debug: *flagDebug,
	}

	server := &http.Server{
		Addr:         *flagHttpAddr,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	shutdownServer := func() {
		if err := server.Shutdown(context.TODO()); err != nil {
			log.Println(err)
		}
	}
	defer shutdownServer()

	// Start main HTTP server
	log.Printf("binding to %s for HTTP", *flagHttpAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Println(err)
	}
}
