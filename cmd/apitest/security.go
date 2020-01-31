// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
)

type authChecker struct {
	apiAddress string

	origDepID    string
	originatorID string
	recDepID     string
	receiverID   string
	transferID   string

	requestID string
	userID    string
}

func (ac *authChecker) checkAll() error {
	if err := ac.canWeBypassAuth("depositories", ac.origDepID); err != nil {
		return fmt.Errorf("originator depository: %v", err)
	}
	if err := ac.canWeBypassAuth("originators", ac.originatorID); err != nil {
		return fmt.Errorf("originators: %v", err)
	}

	if err := ac.canWeBypassAuth("depositories", ac.recDepID); err != nil {
		return fmt.Errorf("receiver depository: %v", err)
	}
	if err := ac.canWeBypassAuth("receivers", ac.receiverID); err != nil {
		return fmt.Errorf("receivers: %v", err)
	}

	if err := ac.canWeBypassAuth("transfers", ac.transferID); err != nil {
		return fmt.Errorf("transfers: %v", err)
	}

	log.Println("INFO: unable to naively bypass auth")

	return nil
}

func (ac *authChecker) canWeBypassAuth(objPathSegments ...string) error {
	u, err := url.Parse(ac.apiAddress)
	if err != nil {
		return err
	}
	u.Path = path.Join(append([]string{"v1", "ach"}, objPathSegments...)...)

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-request-id", ac.requestID)
	req.Header.Set("x-user-id", ac.userID)
	req.Header.Set("Origin", "https://moov.io") // ask for CORS headers

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		bs, _ := ioutil.ReadAll(resp.Body)
		if len(bs) > 0 {
			fmt.Printf("response body:\n%s\n", string(bs))
		}
		return fmt.Errorf("got HTTP status %v back, expected %d", resp.StatusCode, http.StatusForbidden)

	case http.StatusForbidden:
		if *flagDebug {
			log.Printf("DEBUG: ")
		}
		return nil // We expect to be blocked
	}

	return fmt.Errorf("unexpected HTTP status %v", resp.Status)
}
