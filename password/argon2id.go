// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idOption is an option for the argon2id hasher.
type Argon2idOption interface {
	applyArgon2id(h *argon2id)
}

// Memory is the argon2id memory cost in KiB.
type Memory uint32

func (m Memory) applyArgon2id(h *argon2id) {
	h.memory = uint32(m)
}

// Time is the argon2id number of passes.
type Time uint32

func (t Time) applyArgon2id(h *argon2id) {
	h.time = uint32(t)
}

// Parallelism is the argon2id number of threads.
type Parallelism uint8

func (p Parallelism) applyArgon2id(h *argon2id) {
	h.threads = uint8(p)
}

// SaltLength is the salt length in bytes.
type SaltLength uint32

func (s SaltLength) applyArgon2id(h *argon2id) {
	h.saltLen = uint32(s)
}

// KeyLength is the derived key length in bytes.
type KeyLength uint32

func (k KeyLength) applyArgon2id(h *argon2id) {
	h.keyLen = uint32(k)
}

// NewArgon2id returns a Hasher using argon2id with OWASP recommended
// defaults: 19 MiB memory, 2 passes, 1 thread, 16 byte salt, 32 byte key.
func NewArgon2id(opts ...Argon2idOption) Hasher {
	h := &argon2id{
		memory:  19456,
		time:    2,
		threads: 1,
		saltLen: 16,
		keyLen:  32,
	}

	for _, opt := range opts {
		opt.applyArgon2id(h)
	}

	return h
}

type argon2id struct {
	memory  uint32
	time    uint32
	threads uint8
	saltLen uint32
	keyLen  uint32
}

func (h *argon2id) Name() string {
	return "argon2id"
}

func (h *argon2id) Hash(secret string) (string, error) {
	salt := make([]byte, h.saltLen)
	_, _ = rand.Read(salt)

	key := argon2.IDKey([]byte(secret), salt, h.time, h.memory, h.threads, h.keyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memory, h.time, h.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (h *argon2id) Verify(secret, encoded string) (bool, error) {
	memory, time, threads, salt, key, err := parseArgon2id(encoded)
	if err != nil {
		return false, err
	}

	derived := argon2.IDKey([]byte(secret), salt, time, memory, threads, uint32(len(key))) //nolint:gosec

	return subtle.ConstantTimeCompare(key, derived) == 1, nil
}

func (h *argon2id) VerifyEmpty(secret string) {
	derived := argon2.IDKey([]byte(secret), make([]byte, h.saltLen), h.time, h.memory, h.threads, h.keyLen)
	_ = subtle.ConstantTimeCompare(derived, make([]byte, h.keyLen))
}

func (h *argon2id) NeedsRehash(encoded string) bool {
	memory, time, threads, salt, key, err := parseArgon2id(encoded)
	if err != nil {
		return true
	}

	return memory != h.memory || time != h.time || threads != h.threads ||
		len(salt) != int(h.saltLen) || len(key) != int(h.keyLen)
}

// parseArgon2id decodes the PHC encoded argon2id hash parameters.
func parseArgon2id(encoded string) (uint32, uint32, uint8, []byte, []byte, error) {
	rest, ok := strings.CutPrefix(encoded, "$argon2id$")
	if !ok {
		return 0, 0, 0, nil, nil, errInvalid
	}

	verPart, rest, ok := strings.Cut(rest, "$")
	if !ok {
		return 0, 0, 0, nil, nil, errInvalid
	}

	if version, okv := parseParam(verPart, "v=", 32); !okv || version != uint64(argon2.Version) {
		return 0, 0, 0, nil, nil, errInvalid
	}

	params, rest, ok := strings.Cut(rest, "$")
	if !ok {
		return 0, 0, 0, nil, nil, errInvalid
	}

	mPart, params, okm := strings.Cut(params, ",")
	tPart, pPart, okt := strings.Cut(params, ",")

	memory, ok1 := parseParam(mPart, "m=", 32)
	time, ok2 := parseParam(tPart, "t=", 32)
	threads, ok3 := parseParam(pPart, "p=", 8)

	if !okm || !okt || !ok1 || !ok2 || !ok3 {
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

	return uint32(memory), uint32(time), uint8(threads), salt, key, nil //nolint:gosec // Bounded by ParseUint bit sizes.
}
