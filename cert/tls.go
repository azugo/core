package cert

import (
	"crypto/tls"
	"io"
)

// LoadTLSCertificate parses a public/private key pair from a pair of PEM encoded data.
func LoadTLSCertificate(cert, key []byte) (*tls.Certificate, error) {
	c, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ParseTLSCertificateFromReader parses a public/private key pair from a PEM encoded data io.Reader source.
func ParseTLSCertificateFromReader(r io.Reader) (*tls.Certificate, error) {
	cert, key, err := LoadPEMFromReader(r)
	if err != nil {
		return nil, err
	}
	return LoadTLSCertificate(cert, key)
}

// ParseTLSCertificateFromReader parses a public/private key pair from a PEM encoded file.
func ParseTLSCertificateFromFile(path string) (*tls.Certificate, error) {
	cert, key, err := LoadPEMFromFile(path)
	if err != nil {
		return nil, err
	}
	return LoadTLSCertificate(cert, key)
}
