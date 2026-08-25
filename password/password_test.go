// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
)

func TestPackageLevel(t *testing.T) {
	// The default hasher is argon2id.
	encoded, err := Hash("secret")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(strings.HasPrefix(encoded, "$argon2id$")))

	ok, err := Verify("secret", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	qt.Check(t, qt.IsFalse(NeedsRehash(encoded)))

	// Verification dispatches to bcrypt by the hash prefix.
	bhash, err := NewBcrypt(Cost(4)).Hash("secret")
	qt.Assert(t, qt.IsNil(err))

	ok, err = Verify("secret", bhash)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	// A bcrypt hash always needs a rehash to the default algorithm.
	qt.Check(t, qt.IsTrue(NeedsRehash(bhash)))

	// Verification dispatches to scrypt by the hash prefix.
	shash, err := fastScrypt().Hash("secret")
	qt.Assert(t, qt.IsNil(err))

	ok, err = Verify("secret", shash)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))
	qt.Check(t, qt.IsTrue(NeedsRehash(shash)))

	VerifyEmpty("secret")
}

func TestVerifyErrors(t *testing.T) {
	if _, err := Verify("secret", "$md5$abc$def"); err == nil {
		t.Error("unknown algorithm: expected error")
	}

	for _, encoded := range []string{"", "garbage", "$", "$$"} {
		if _, err := Verify("secret", encoded); err == nil {
			t.Errorf("Verify(%q): expected error", encoded)
		}
	}
}

type fakeHasher struct{}

func (fakeHasher) Name() string                     { return "fake" }
func (fakeHasher) Hash(string) (string, error)      { return "$fake$x", nil }
func (fakeHasher) Verify(_, e string) (bool, error) { return e == "$fake$x", nil }
func (fakeHasher) VerifyEmpty(string)               {}
func (fakeHasher) NeedsRehash(e string) bool        { return e != "$fake$x" }

func TestRegisterAndSetDefault(t *testing.T) {
	Register(fakeHasher{})

	ok, err := Verify("anything", "$fake$x")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	prev := defaultHasher()
	defer SetDefault(prev)

	SetDefault(fakeHasher{})

	encoded, err := Hash("secret")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(encoded, "$fake$x"))
	qt.Check(t, qt.IsFalse(NeedsRehash("$fake$x")))
}
