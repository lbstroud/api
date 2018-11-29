// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
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

// createUser randomly generates a user (with profile data) and creates it against the given Moov API.
// Returned is the User's Id and Email address
func createUser(ctx context.Context, api *moov.APIClient, requestId string) (string, string, error) {
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
		log.Fatalf("problem creating user: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	// Now login
	login := moov.Login{Email: req.Email, Password: *flagPassword}
	user, resp, err := api.UserApi.UserLogin(ctx, login, &moov.UserLoginOpts{
		XRequestId:      optional.NewString(requestId),
		XIdempotencyKey: optional.NewString(generateID()),
	})
	if err != nil {
		log.Fatalf("problem logging in for user: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	return user.Id, user.Email, nil
}

func email(first, last string) string {
	return fmt.Sprintf("%s.%s%d@example.com", strings.ToLower(first), strings.ToLower(last), randSource.Int63()%50)
}

func name() (string, string) {
	parts := strings.Split(namesgenerator.GetRandomName(0), "_")
	if len(parts) != 2 {
		return "", ""
	}
	return strings.Title(parts[0]), strings.Title(parts[1])
}

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
