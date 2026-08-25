// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// BcryptOption is an option for the bcrypt hasher.
type BcryptOption interface {
	applyBcrypt(h *bcryptHasher)
}

// Cost is the bcrypt cost.
type Cost int

func (c Cost) applyBcrypt(h *bcryptHasher) {
	h.cost = int(c)
}

// NewBcrypt returns a Hasher using bcrypt with the default cost 10.
func NewBcrypt(opts ...BcryptOption) Hasher {
	h := &bcryptHasher{cost: bcrypt.DefaultCost}

	for _, opt := range opts {
		opt.applyBcrypt(h)
	}

	secret := make([]byte, 16)
	_, _ = rand.Read(secret)

	dummy, err := bcrypt.GenerateFromPassword(secret, h.cost)
	if err != nil {
		panic(err)
	}

	h.dummy = dummy

	return h
}

type bcryptHasher struct {
	cost  int
	dummy []byte
}

func (h *bcryptHasher) Name() string {
	return "2a"
}

func (h *bcryptHasher) Hash(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), h.cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (h *bcryptHasher) Verify(secret, encoded string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(secret))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (h *bcryptHasher) VerifyEmpty(secret string) {
	_ = bcrypt.CompareHashAndPassword(h.dummy, []byte(secret))
}

func (h *bcryptHasher) NeedsRehash(encoded string) bool {
	cost, err := bcrypt.Cost([]byte(encoded))

	return err != nil || cost != h.cost
}
