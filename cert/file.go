// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cert

import (
	"os"
)

// LoadPEMFromFile loads a PEM-encoded certificate and private key from
// the specified file.
func LoadPEMFromFile(path string, opt ...Option) ([]byte, []byte, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	return LoadPEMFromReader(r, opt...)
}
