// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
)

type wrapper struct {
	Tag string `json:"tag_name"`
}

func checkLatestVersion(app string) error {
	resp, err := httpClient.Get(fmt.Sprintf("https://api.github.com/repos/moov-io/%s/releases/latest", app))
	if err != nil {
		return fmt.Errorf("error getting %s version: %v", app, err)
	}
	defer resp.Body.Close()

	var wrapper wrapper
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return fmt.Errorf("error reading %s json: %v", app, err)
	}

	if v := versions[app]; strings.EqualFold(wrapper.Tag, v) {
		return nil // configured version is latest
	} else {
		log.Printf("WARN %s is configured for %s but %s is latest release", app, v, wrapper.Tag)
	}
	return nil
}
