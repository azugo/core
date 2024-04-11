package cert

import (
	"crypto/tls"
	"io"
)

// LoadTLSCertificate parses a public/private key pair from a pair of PEM encoded data.
func LoadTLSCertificate(cert, key []byte) (*tls.Certificate, error) {
	crt, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	return &crt, nil
}

// ParseTLSCertificateFromReader parses a public/private key pair from a PEM encoded data io.Reader source.
func ParseTLSCertificateFromReader(r io.Reader, opt ...Option) (*tls.Certificate, error) {
	crt, key, err := LoadPEMFromReader(r, opt...)
	if err != nil {
		return nil, err
	}

	return LoadTLSCertificate(crt, key)
}

// ParseTLSCertificateFromReader parses a public/private key pair from a PEM encoded file.
func ParseTLSCertificateFromFile(path string, opt ...Option) (*tls.Certificate, error) {
	crt, key, err := LoadPEMFromFile(path, opt...)
	if err != nil {
		return nil, err
	}

	return LoadTLSCertificate(crt, key)
}
