// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

// localdevproxy is a cli tool (and Docker image) used for proxying requests to the Moov API.
//
// This tool is not used in production and is intended for local development work intended to act
// like our Production load balancing.
//
// localdevproxy will proxy requests to local endpoints (similar to local.Transport / apitest -local)
// so developers can 'go run' applications together.
//
// localdevproxy will proxy requests when inside a Kubernetes cluster (for tilt local dev) according
// to Kubernetes service DNS records.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/moov-io/api"
	"github.com/moov-io/api/cmd/apitest/local"
	"github.com/moov-io/base/k8s"
)

var (
	flagHttpAddr = flag.String("http.addr", ":9000", "HTTP listen address")
	flagDebug    = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds | log.Lshortfile)
	log.Printf("Starting moov localproxy %s", api.Version())

	proxy := createReverseProxy()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
			return
		}
		proxy.ServeHTTP(w, r)
	})
	server := &http.Server{
		Addr:         *flagHttpAddr,
		Handler:      handler,
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

func createReverseProxy() *httputil.ReverseProxy {
	if k8s.Inside() {
		log.Println("using Kubernetes reverse proxy")
		return kubernetesReverseProxy()
	}
	log.Println("using local reverse proxy")
	return localReverseProxy()
}

// kubernetesReverseProxy returns a httputil.ReverseProxy instance that rewrites incoming requests
// for Kubernetes dns names and ports. This is designed for drop-in support into a Kubernetes cluster.
func kubernetesReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			origURL := r.URL.String()

			r.URL.Scheme = "http"

			// Each route looks like /v1/$app/... so match based on that
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) < 3 { // parts splits into: "", v1, $app, (rest of url)
				return // unknown route, just proxy
			}

			// Routing logic, should match Ingress routes and ./cmd/apitest's logic
			switch strings.ToLower(parts[2]) {
			case "ach":
				switch strings.ToLower(parts[3]) {
				case "depositories", "originators", "receivers", "transfers":
					r.URL.Host = "paygate.apps.svc.cluster.local:8080"
				default:
					r.URL.Host = "ach.apps.svc.cluster.local:8080"
				}
			case "accounts", "customers":
				// Append the above path segments back onto the URL
				r.URL.Host = fmt.Sprintf("%s.apps.svc.cluster.local:8080", strings.ToLower(parts[2]))
				r.URL.Path = fmt.Sprintf("/%s", parts[2])
				if len(parts) > 3 {
					r.URL.Path += "/" + strings.Join(parts[3:], "/")
				}
				return // early exit since we already modify .Path
			case "oauth2", "users":
				// Append the above path segments back onto the URL
				r.URL.Host = "auth.apps.svc.cluster.local:8080"
				r.URL.Path = fmt.Sprintf("/%s", parts[2])
				if len(parts) > 3 {
					r.URL.Path += "/" + strings.Join(parts[3:], "/")
				}
				return // early exit since we already modify .Path
			default:
				r.URL.Host = fmt.Sprintf("%s.apps.svc.cluster.local:8080", strings.ToLower(parts[2]))
			}

			// Truncate off /v1/$name
			r.URL.Path = "/" + strings.Join(parts[3:], "/")

			if *flagDebug {
				log.Printf("%v %v request URL (Original: %v)", r.Method, r.URL.String(), origURL)
			}
		},
	}
}

// localReverseProxy returns a httputil.ReverseProxy instance that rewrites incoming requests
// according to the local.Transport struct. This allows proxing for services ran with 'go run'.
func localReverseProxy() *httputil.ReverseProxy {
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
	return httputil.NewSingleHostReverseProxy(u)
}
