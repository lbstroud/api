// Copyright 2018 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"time"
)

func Version() string {
	return fmt.Sprintf("v%s.1", time.Now().Format("2006-01-02"))
}
