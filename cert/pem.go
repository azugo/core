// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cert

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/pem"
	"fmt"

	"github.com/lafriks/pkcs8"
)

const (
	PEMBlockRSAPrivateKey       = "RSA PRIVATE KEY"
	PEMBlockECPrivateKey        = "EC PRIVATE KEY"
	PEMBlockEncryptedPrivateKey = "ENCRYPTED PRIVATE KEY"
	PEMBlockPrivateKey          = "PRIVATE KEY"
	PEMBlockCertificate         = "CERTIFICATE"
)

func publicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv any, opt ...Option) (*pem.Block, error) {
	opts := opts(opt...)

	b, err := pkcs8.MarshalPrivateKey(priv, opts.Password, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal private key: %w", err)
	}

	if len(opts.Password) > 0 {
		return &pem.Block{Type: PEMBlockEncryptedPrivateKey, Bytes: b}, nil
	}

	return &pem.Block{Type: PEMBlockPrivateKey, Bytes: b}, nil
}

// DERBytesToPEMBlocks converts certificate DER bytes and optional private key
// to PEM blocks.
// Returns certificate PEM block and private key PEM block.
func DERBytesToPEMBlocks(der []byte, priv any, opt ...Option) ([]byte, []byte, error) {
	out := &bytes.Buffer{}
	if err := pem.Encode(out, &pem.Block{Type: PEMBlockCertificate, Bytes: der}); err != nil {
		return nil, nil, err
	}

	cert := append([]byte{}, out.Bytes()...)

	var key []byte

	if priv != nil {
		out.Reset()

		block, err := pemBlockForKey(priv, opt...)
		if err != nil {
			return nil, nil, err
		}

		if err := pem.Encode(out, block); err != nil {
			return nil, nil, err
		}

		key = append([]byte{}, out.Bytes()...)
	}

	return cert, key, nil
}
