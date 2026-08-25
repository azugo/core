// Copyright 2026 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
)

func TestBcryptRoundTrip(t *testing.T) {
	h := NewBcrypt(Cost(4))

	encoded, err := h.Hash("secret")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(strings.HasPrefix(encoded, "$2a$04$")))

	ok, err := h.Verify("secret", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	ok, err = h.Verify("wrong", encoded)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))

	if _, err := h.Verify("secret", "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5"); err == nil {
		t.Error("Verify of foreign hash: expected error")
	}
}

func TestBcryptNeedsRehash(t *testing.T) {
	h := NewBcrypt(Cost(4))

	encoded, err := h.Hash("secret")
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.IsFalse(h.NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(NewBcrypt(Cost(5)).NeedsRehash(encoded)))
	qt.Check(t, qt.IsTrue(h.NeedsRehash("$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5")))
	qt.Check(t, qt.IsTrue(h.NeedsRehash("garbage")))
}

func TestBcryptVerifyEmpty(t *testing.T) {
	h := NewBcrypt(Cost(4))
	h.VerifyEmpty("secret")
	h.VerifyEmpty("")
}
