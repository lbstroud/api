// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var (
	versionFileTemplate = `// Copyright %s The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package api

func Version() string {
	return "%s"
}
`
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatalf("missing version")
	}

	bs, err := format.Source([]byte(fmt.Sprintf(versionFileTemplate, time.Now().Format("2006"), os.Args[1])))
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("version.go", bs, 0644); err != nil {
		log.Fatal(err)
	}

	log.Printf("moov-io/api version %s\n", os.Args[1])
}
