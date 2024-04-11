// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"os"
)

// LoadRemoteSecret loads a remote secret from configuration provided in environment variable.
//
// Environment variable name is expected to be in the format:
//
//	<name>_FILE - path to the file containing the secret
func LoadRemoteSecret(name string) (string, error) {
	path := os.Getenv(name + "_FILE")
	if _, err := os.Stat(path); err == nil {
		if content, err := os.ReadFile(path); err != nil {
			return "", err
		} else if len(content) > 0 {
			return string(bytes.TrimSpace(content)), nil
		}
	}

	return "", nil
}
