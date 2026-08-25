// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
)

func fastArgon2id(opts ...Argon2idOption) Hasher {
	return NewArgon2id(append([]Argon2idOption{Memory(64), Time(1)}, opts...)...)
}

func TestArgon2idRoundTrip(t *testing.T) {
	h := fastArgon2id()

	encoded, err := h.Hash("secret")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(strings.HasPrefix(encoded, "$argon2id$v=19$m=64,t=1,p=1$")))

	ok, err := h.Verify("secret", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	ok, err = h.Verify("wrong", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))

	// Two hashes of the same secret differ by salt but both verify.
	other, err := h.Hash("secret")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Not(qt.Equals(other, encoded)))

	ok, err = h.Verify("secret", other)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))
}

func TestArgon2idEmptySecret(t *testing.T) {
	h := fastArgon2id()

	encoded, err := h.Hash("")
	qt.Assert(t, qt.IsNil(err))

	ok, err := h.Verify("", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	ok, err = h.Verify("x", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))
}

func TestArgon2idVerifyInvalid(t *testing.T) {
	h := fastArgon2id()

	for _, encoded := range []string{
		"",
		"secret",
		"$argon2id$v=19$m=64,t=1,p=1$abc",
		"$argon2id$v=18$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$a2V5",
		"$argon2id$v=19$m=64,t=1,p=1$!!$a2V5",
		"$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$!!",
		"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
	} {
		if _, err := h.Verify("secret", encoded); err == nil {
			t.Errorf("Verify(%q): expected error", encoded)
		}
	}
}

func TestArgon2idNeedsRehash(t *testing.T) {
	h := fastArgon2id()

	encoded, err := h.Hash("secret")
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.IsFalse(h.NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastArgon2id(Memory(128)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastArgon2id(Time(2)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastArgon2id(Parallelism(2)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastArgon2id(SaltLength(32)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastArgon2id(KeyLength(64)).NeedsRehash(encoded)))

	// Foreign algorithm and garbage always need a rehash.
	qt.Check(t, qt.IsTrue(h.NeedsRehash("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")))
	qt.Check(t, qt.IsTrue(h.NeedsRehash("garbage")))
}

func TestArgon2idVerifyEmpty(t *testing.T) {
	fastArgon2id().VerifyEmpty("secret")
	fastArgon2id().VerifyEmpty("")
}
