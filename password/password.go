// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package password provides secret hashing and verification with hashes
// stored as a single PHC-formatted string.
package password

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

var errInvalid = errors.New("password: invalid hash")

// Hasher hashes and verifies secrets in a PHC-formatted string.
type Hasher interface {
	// Name is the PHC algorithm identifier.
	Name() string
	// Hash returns the PHC encoded hash of the secret using the configured
	// parameters and a fresh random salt.
	Hash(secret string) (string, error)
	// Verify reports in constant time whether secret matches the encoded
	// hash.
	Verify(secret, encoded string) (bool, error)
	// VerifyEmpty performs the same amount of work as Verify against a hash
	// that does not exist.
	VerifyEmpty(secret string)
	// NeedsRehash reports whether encoded was produced with an algorithm or
	// parameters different from this hasher's configuration.
	NeedsRehash(encoded string) bool
}

var (
	mu       sync.RWMutex
	def      Hasher
	registry = map[string]Hasher{}
)

func init() {
	a := NewArgon2id()
	b := NewBcrypt()
	s := NewScrypt()

	def = a
	registry[a.Name()] = a
	registry[s.Name()] = s

	// Go bcrypt verifies all common bcrypt version prefixes.
	for _, name := range []string{"2a", "2b", "2y"} {
		registry[name] = b
	}
}

// SetDefault replaces the process-wide default Hasher used by Hash,
// VerifyEmpty and NeedsRehash, and registers it for verification.
func SetDefault(h Hasher) {
	mu.Lock()
	defer mu.Unlock()

	def = h
	registry[h.Name()] = h
}

// Register adds an algorithm to the verification registry so Verify can
// dispatch on its encoded name prefix.
func Register(h Hasher) {
	mu.Lock()
	defer mu.Unlock()

	registry[h.Name()] = h
}

// Hash returns the PHC encoded hash of the secret using the default Hasher.
func Hash(secret string) (string, error) {
	return defaultHasher().Hash(secret)
}

// Verify reports whether secret matches the encoded hash, dispatching on
// the hash algorithm name prefix.
func Verify(secret, encoded string) (bool, error) {
	name := algorithm(encoded)
	if name == "" {
		return false, errInvalid
	}

	mu.RLock()

	h, ok := registry[name]

	mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("password: unknown algorithm %q", name)
	}

	return h.Verify(secret, encoded)
}

// VerifyEmpty performs the same amount of work as Verify against a hash
// that does not exist using the default Hasher, always failing.
func VerifyEmpty(secret string) {
	defaultHasher().VerifyEmpty(secret)
}

// NeedsRehash reports whether encoded was produced with an algorithm or
// parameters different from the default Hasher's configuration.
func NeedsRehash(encoded string) bool {
	return defaultHasher().NeedsRehash(encoded)
}

func defaultHasher() Hasher {
	mu.RLock()
	defer mu.RUnlock()

	return def
}

// parseParam parses a PHC prefixed unsigned parameter, e.g. "m=19456".
func parseParam(s, prefix string, bits int) (uint64, bool) {
	s, ok := strings.CutPrefix(s, prefix)
	if !ok {
		return 0, false
	}

	v, err := strconv.ParseUint(s, 10, bits)

	return v, err == nil
}

// algorithm returns the PHC algorithm name prefix of the encoded hash.
func algorithm(encoded string) string {
	rest, ok := strings.CutPrefix(encoded, "$")
	if !ok {
		return ""
	}

	name, _, found := strings.Cut(rest, "$")
	if !found {
		return ""
	}

	return name
}
