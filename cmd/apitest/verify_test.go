// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerify_verifyDirIsEmpty(t *testing.T) {
	dir, err := ioutil.TempDir("", "verifyDirIsEmpty")
	if err != nil {
		t.Fatal(err)
	}

	if !verifyDirIsEmpty(dir) {
		t.Error("empty dir should be empty")
	}

	// write a file and verify the dir is non-empty
	fd, err := os.Create(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(fd, strings.NewReader("hello, world")); err != nil {
		t.Fatal(err)
	}
	fd.Sync()
	fd.Close()

	// won't be empty
	if verifyDirIsEmpty(dir) {
		t.Error("dir should be empty")
	}

	// delete that file and retry
	if err := os.Remove(filepath.Join(dir, "file.txt")); err != nil {
		t.Fatal(err)
	}
	if !verifyDirIsEmpty(dir) {
		t.Error("empty dir should be empty")
	}
}
