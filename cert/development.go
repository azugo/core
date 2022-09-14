// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CreateDevPEM creates a self-signed certificate for development.
// Returns certificate DER bytes and private key.
func CreateDevPEM(dns ...string) ([]byte, any, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	var max *big.Int = big.NewInt(0).Exp(big.NewInt(2), big.NewInt(130), nil)
	serial, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Azugo Development"},
		},
		NotBefore: time.Now(),
		// Valid for 5 years
		NotAfter: time.Now().Add(time.Hour * 24 * 1826),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	template.DNSNames = append(template.DNSNames, dns...)

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return nil, nil, err
	}
	return der, priv, nil
}

// DevPEMFile loads or generates new TLS certificate for development.
func DevPEMFile(name string, dns ...string) ([]byte, []byte, error) {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, err
	}
	path := filepath.Join(dataDir, name+".pem")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		der, priv, err := CreateDevPEM(dns...)
		if err != nil {
			return nil, nil, err
		}
		cert, key, err := DERBytesToPEMBlocks(der, priv)
		if err != nil {
			return nil, nil, err
		}
		f, err := os.Create(path)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		_, _ = f.Write(cert)
		_, _ = f.WriteString("\n")
		_, _ = f.Write(key)

		return cert, key, nil
	}

	return LoadPEMFromFile(path)
}
