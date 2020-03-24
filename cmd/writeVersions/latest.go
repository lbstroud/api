// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
)

func checkLatestVersion(w func(app string, current, latest, prerelease string), app string) error {
	latest, err := latestRelease(app)
	if err != nil {
		return fmt.Errorf("latest: %v", err)
	}
	prerelease, err := latestPreRelease(app)
	if err != nil {
		return fmt.Errorf("pre-release: %v", err)
	}
	current := versions[app]

	if latest == "" || prerelease == "" {
		return nil
	}
	if strings.EqualFold(latest, current) || strings.EqualFold(prerelease, current) {
		return nil
	}
	w(app, current, latest, prerelease) // write version differences
	return nil
}

func latestRelease(app string) (string, error) {
	return retrieveVersion(
		app,
		fmt.Sprintf("https://api.github.com/repos/moov-io/%s/releases/latest", app),
	)
}

func latestPreRelease(app string) (string, error) {
	return retrieveVersion(
		app,
		fmt.Sprintf("https://api.github.com/repos/moov-io/%s/releases", app),
	)
}

type body struct {
	Tag string `json:"tag_name"`
}

func retrieveVersion(app, url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("error getting %s version: %v", app, err)
	}
	defer resp.Body.Close()

	bs, _ := ioutil.ReadAll(resp.Body)

	var wrapper body
	if err := json.NewDecoder(bytes.NewReader(bs)).Decode(&wrapper); err != nil {
		var wrapper2 []body
		if err := json.NewDecoder(bytes.NewReader(bs)).Decode(&wrapper2); err != nil {
			return "", fmt.Errorf("error reading %s json: %v", app, err)
		}
		return wrapper2[0].Tag, nil
	}
	return wrapper.Tag, nil
}
