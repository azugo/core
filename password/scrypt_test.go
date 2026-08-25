// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
)

func fastScrypt(opts ...ScryptOption) Hasher {
	return NewScrypt(append([]ScryptOption{LogN(4)}, opts...)...)
}

func TestScryptRoundTrip(t *testing.T) {
	h := fastScrypt()

	encoded, err := h.Hash("secret")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(strings.HasPrefix(encoded, "$scrypt$ln=4,r=8,p=1$")))

	ok, err := h.Verify("secret", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	ok, err = h.Verify("wrong", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))

	ok, err = h.Verify("", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))
}

func TestScryptVerifyInvalid(t *testing.T) {
	h := fastScrypt()

	for _, encoded := range []string{
		"",
		"$scrypt$ln=4,r=8,p=1$abc",
		"$scrypt$ln=0,r=8,p=1$c2FsdA$a2V5",
		"$scrypt$ln=4,r=8,p=1$!!$a2V5",
		"$scrypt$ln=4,r=8,p=1$c2FsdA$!!",
		"$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5",
	} {
		if _, err := h.Verify("secret", encoded); err == nil {
			t.Errorf("Verify(%q): expected error", encoded)
		}
	}
}

func TestScryptNeedsRehash(t *testing.T) {
	h := fastScrypt()

	encoded, err := h.Hash("secret")
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.IsFalse(h.NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastScrypt(LogN(5)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastScrypt(BlockSize(4)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastScrypt(Parallelism(2)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastScrypt(SaltLength(32)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(fastScrypt(KeyLength(64)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(h.NeedsRehash("garbage")))
}

func TestNewScryptInvalidParams(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic on invalid scrypt parameters")
		}
	}()
	_ = NewScrypt(LogN(0))
}

func TestScryptVerifyEmpty(t *testing.T) {
	fastScrypt().VerifyEmpty("secret")
	fastScrypt().VerifyEmpty("")
}
