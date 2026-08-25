// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// ScryptOption is an option for the scrypt hasher.
type ScryptOption interface {
	applyScrypt(h *scryptHasher)
}

// LogN is the base 2 logarithm of the scrypt CPU/memory cost parameter N.
type LogN int

func (n LogN) applyScrypt(h *scryptHasher) {
	h.logN = int(n)
}

// BlockSize is the scrypt block size parameter r.
type BlockSize int

func (r BlockSize) applyScrypt(h *scryptHasher) {
	h.blockSize = int(r)
}

func (p Parallelism) applyScrypt(h *scryptHasher) {
	h.parallel = int(p)
}

func (s SaltLength) applyScrypt(h *scryptHasher) {
	h.saltLen = int(s)
}

func (k KeyLength) applyScrypt(h *scryptHasher) {
	h.keyLen = int(k)
}

// NewScrypt returns a Hasher using scrypt with OWASP recommended defaults:
// N=2^17, r=8, p=1, 16 byte salt, 32 byte key.
func NewScrypt(opts ...ScryptOption) Hasher {
	h := &scryptHasher{
		logN:      17,
		blockSize: 8,
		parallel:  1,
		saltLen:   16,
		keyLen:    32,
	}

	for _, opt := range opts {
		opt.applyScrypt(h)
	}

	if h.logN < 1 || h.logN > 31 || h.blockSize < 1 || h.parallel < 1 {
		panic(errors.New("password: invalid scrypt parameters"))
	}

	return h
}

type scryptHasher struct {
	logN      int
	blockSize int
	parallel  int
	saltLen   int
	keyLen    int
}

func (h *scryptHasher) Name() string {
	return "scrypt"
}

func (h *scryptHasher) Hash(secret string) (string, error) {
	salt := make([]byte, h.saltLen)
	_, _ = rand.Read(salt)

	key, err := scrypt.Key([]byte(secret), salt, 1<<h.logN, h.blockSize, h.parallel, h.keyLen)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("$scrypt$ln=%d,r=%d,p=%d$%s$%s",
		h.logN, h.blockSize, h.parallel,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (h *scryptHasher) Verify(secret, encoded string) (bool, error) {
	logN, blockSize, parallel, salt, key, err := parseScrypt(encoded)
	if err != nil {
		return false, err
	}

	derived, err := scrypt.Key([]byte(secret), salt, 1<<logN, blockSize, parallel, len(key))
	if err != nil {
		return false, err
	}

	return subtle.ConstantTimeCompare(key, derived) == 1, nil
}

func (h *scryptHasher) VerifyEmpty(secret string) {
	derived, _ := scrypt.Key([]byte(secret), make([]byte, h.saltLen), 1<<h.logN, h.blockSize, h.parallel, h.keyLen)
	_ = subtle.ConstantTimeCompare(derived, make([]byte, h.keyLen))
}

func (h *scryptHasher) NeedsRehash(encoded string) bool {
	logN, blockSize, parallel, salt, key, err := parseScrypt(encoded)
	if err != nil {
		return true
	}

	return logN != h.logN || blockSize != h.blockSize || parallel != h.parallel ||
		len(salt) != h.saltLen || len(key) != h.keyLen
}

// parseScrypt decodes the PHC encoded scrypt hash parameters.
func parseScrypt(encoded string) (int, int, int, []byte, []byte, error) {
	rest, ok := strings.CutPrefix(encoded, "$scrypt$")
	if !ok {
		return 0, 0, 0, nil, nil, errInvalid
	}

	params, rest, ok := strings.Cut(rest, "$")
	if !ok {
		return 0, 0, 0, nil, nil, errInvalid
	}

	lnPart, params, okn := strings.Cut(params, ",")
	rPart, pPart, okr := strings.Cut(params, ",")

	logN, ok1 := parseParam(lnPart, "ln=", 8)
	blockSize, ok2 := parseParam(rPart, "r=", 32)
	parallel, ok3 := parseParam(pPart, "p=", 32)

	if !okn || !okr || !ok1 || !ok2 || !ok3 || logN < 1 || logN > 31 {
		return 0, 0, 0, nil, nil, errInvalid
	}

	saltPart, keyPart, ok := strings.Cut(rest, "$")
	if !ok || strings.ContainsRune(keyPart, '$') {
		return 0, 0, 0, nil, nil, errInvalid
	}

	salt, err := base64.RawStdEncoding.DecodeString(saltPart)
	if err != nil {
		return 0, 0, 0, nil, nil, errInvalid
	}

	key, err := base64.RawStdEncoding.DecodeString(keyPart)
	if err != nil {
		return 0, 0, 0, nil, nil, errInvalid
	}

	return int(logN), int(blockSize), int(parallel), salt, key, nil //nolint:gosec // Bounded by ParseUint bit sizes.
}
