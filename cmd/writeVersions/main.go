// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	templateFilepaths = func() []string {
		paths := append(
			readFilepaths(filepath.Join("site", "admin", "*", "index.html.tpl")),
			filepath.Join("openapi.yaml.tpl"),
		)
		paths = append(paths, readFilepaths(filepath.Join("site", "apps", "*", "index.html.tpl"))...)
		sort.Strings(paths)
		return paths
	}()

	versions = map[string]string{
		"accounts":        "v0.4.1",
		"ach":             "v1.3.1",
		"auth":            "v0.8.0",
		"customers":       "v0.3.0",
		"fed":             "v0.4.1",
		"imagecashletter": "v0.3.0",
		"paygate":         "v0.7.1",
		"watchman":        "v0.13.1",
		"wire":            "v0.3.0",
	}
)

func readFilepaths(pattern string) []string {
	infos, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("pattern=%s error=%v", pattern, err)
	}
	return infos
}

func replaceVersions(path string) error {
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	bs, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil
	}

	for app := range versions {
		needle := fmt.Sprintf("$%sVersion", app)
		bs = bytes.ReplaceAll(bs, []byte(needle), []byte(versions[app]))
	}

	path = strings.TrimSuffix(path, filepath.Ext(path))
	if err := ioutil.WriteFile(path, bs, 0644); err != nil {
		return err
	}

	log.Printf("wrote %s", path)
	return nil
}

func main() {
	for i := range templateFilepaths {
		if err := replaceVersions(templateFilepaths[i]); err != nil {
			log.Fatalf("path=%s error=%v", templateFilepaths[i], err)
		}
	}
}
