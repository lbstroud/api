// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/moov-io/ach"
)

func verifyDirIsEmpty(dir string) bool {
	s, err := os.Stat(dir)
	if err != nil || !s.IsDir() {
		if os.IsNotExist(err) {
			return true // dir doesn't exist
		}
		return false // dir doesnisn't a directory
	}
	fd, err := os.Open(dir)
	if err != nil {
		return false
	}
	names, err := fd.Readdirnames(1)
	if (err != nil && err != io.EOF) || len(names) > 0 {
		return false // found a file, so not empty
	}
	return true
}

// verifyTransfersWereMerged will take the incoming iterations (i.e. Transfers and related metadata) to
// verify all transfers exist in the merged ACH files in dir. This is done to help ensure paygate handles
// and uploads all the given transfers to the FED / receiving FI.
func verifyTransfersWereMerged(dir string, iterations []*iteration) error {
	iterationsBeforeMatching, mergedFilesProcessed := len(iterations), 0
	if len(iterations) == 0 {
		return fmt.Errorf("no iterations (transfers) found")
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if (err != nil && err != filepath.SkipDir) || info.IsDir() {
			return nil // Ignore SkipDir and directories
		}

		file, err := parseACHFilepath(path)
		if err != nil {
			return fmt.Errorf("error reading %s: %v", path, err)
		}
		i := -1
		mergedFilesProcessed++
		for {
			i++
			if i >= len(iterations) {
				break
			}
			if file.Header.ImmediateOrigin != iterations[i].originatorDepository.RoutingNumber {
				continue
			}
			if file.Header.ImmediateDestination != iterations[i].receiverDepository.RoutingNumber {
				continue
			}
			if *flagDebug {
				log.Printf("origin: %s vs %s destination: %s vs %s\n",
					file.Header.ImmediateOrigin, iterations[i].originatorDepository.RoutingNumber,
					file.Header.ImmediateDestination, iterations[i].receiverDepository.RoutingNumber)
			}
			// Check file's batches
			for j := range file.Batches {
				entries := file.Batches[j].GetEntries()
				for k := range entries {
					amount := fmt.Sprintf("USD %.2f", float64(entries[k].Amount)/100.0) // TODO(adam): use paygate's shared Amount type
					if *flagDebug {
						log.Printf("DEBUG: amounts %s vs %s\n", iterations[i].transfer.Amount, amount)
					}
					if iterations[i].transfer.Amount == amount {
						log.Printf("INFO: Matched transfer %s for %s", iterations[i].transfer.ID, iterations[i].transfer.Amount)
						// found a match // TODO(adam): compare more fields?
						iterations = append(iterations[:i], iterations[i+1:]...) // remove iteration
						i = -1                                                   // reset i so we reprocess entire iterations array
						goto next
					}
				}
			}
		next:
			// check the next iteration, iterations leftover are those that weren't found in a file
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(iterations) > 0 {
		var transferLine []string
		for i := range iterations {
			transferLine = append(transferLine, fmt.Sprintf("%s (amount: %s)", iterations[i].transfer.ID, iterations[i].transfer.Amount))
		}
		if iterationsBeforeMatching == len(iterations) || mergedFilesProcessed == 0 {
			log.Printf("0/%d transfers matched, did paygate create any merged files? (%d files processed)", iterationsBeforeMatching, mergedFilesProcessed)
		}
		return fmt.Errorf(fmt.Sprintf("transfers not matched!!\n%s", strings.Join(transferLine, "\n")))
	} else {
		log.Printf("SUCCESS: all transfers matched in merged file(s)")
	}
	return nil
}

func parseACHFilepath(path string) (*ach.File, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	file, err := ach.NewReader(fd).Read()
	if err != nil {
		return nil, err
	}
	return &file, nil
}
