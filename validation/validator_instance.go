// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"github.com/go-playground/validator/v10"
)

type Validate struct {
	*validator.Validate
}

// New returns a new instance of 'validate' with sane defaults.
// Validate is designed to be thread-safe and used as a singleton instance.
// It caches information about your struct and validations,
// in essence only parsing your validation tags once per struct type.
// Using multiple instances neglects the benefit of caching.
func New() *Validate {
	return &Validate{validator.New()}
}
