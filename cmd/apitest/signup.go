// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	moov "github.com/moov-io/go-client/client"

	"github.com/antihax/optional"
	"github.com/docker/docker/pkg/namesgenerator"
)

var (
	randSource = rand.NewSource(time.Now().Unix())

	numbers       = "0123456789"
	numbersLength = int64(len(numbers) - 1)

	flagPassword = flag.String("user.password", "password", "Password to set for user")
)

type user struct {
	ID    string
	Email string
	Name  string

	Cookie *http.Cookie
}

// createUser randomly generates a user (with profile data) and creates it against the given Moov API.
func createUser(ctx context.Context, api *moov.APIClient, requestId string) (*user, error) {
	first, last := name()
	req := moov.CreateUser{
		Email:     email(first, last),
		Password:  *flagPassword,
		FirstName: first,
		LastName:  last,
		Phone:     phone(),
	}
	_, resp, err := api.UserApi.CreateUser(ctx, req, &moov.CreateUserOpts{
		XRequestId:      optional.NewString(requestId),
		XIdempotencyKey: optional.NewString(generateID()),
	})
	if err != nil {
		if resp != nil {
			bs, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("Response:")
			fmt.Printf("  Status: %v\n", resp.Status)
			fmt.Printf("  Header: %v\n", resp.Header)
			fmt.Printf("  Body: %v\n", string(bs))
		}
		return nil, fmt.Errorf("problem creating user: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	// Now login
	login := moov.Login{Email: req.Email, Password: *flagPassword}
	u, resp, err := api.UserApi.UserLogin(ctx, login, &moov.UserLoginOpts{
		XRequestId:      optional.NewString(requestId),
		XIdempotencyKey: optional.NewString(generateID()),
	})
	if err != nil {
		return nil, fmt.Errorf("problem logging in for user: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	return &user{
		ID:     u.Id,
		Name:   fmt.Sprintf("%s %s", u.FirstName, u.LastName),
		Email:  u.Email,
		Cookie: findMoovCookie(resp.Cookies()),
	}, nil
}

// findMoovCookie pulls out the Moov API cookie. It's designed to be used
// directly from *http.Response.Cookies()
func findMoovCookie(cookies []*http.Cookie) *http.Cookie {
	for i := range cookies {
		if cookies[i].Name == "moov_auth" {
			return cookies[i]
		}
	}
	return nil
}

// email creates an email address of the form X.YD@example.com
// X - first name
// Y - last name
// D - random int between 0 and 50
//
// An email address returned is not guarenteed to be unique.
func email(first, last string) string {
	return fmt.Sprintf("%s.%s%d@example.com", strings.ToLower(first), strings.ToLower(last), randSource.Int63()%50)
}

// name generates a random first and last name
//
// The names come from a fixed list so overlaps are probable.
func name() (string, string) {
	parts := strings.Split(namesgenerator.GetRandomName(0), "_")
	if len(parts) != 2 {
		return "", ""
	}
	return strings.Title(parts[0]), strings.Title(parts[1])
}

// phone generates a random phone number accepted by the Moov API in the form XXX.YYY.ZZZZ
func phone() string {
	tpl, out := "XXX.XXX.XXXX", ""
	for _, c := range tpl {
		if c == '.' {
			out += "."
			continue
		}
		out += string(numbers[randSource.Int63()%numbersLength])
	}
	return out
}
