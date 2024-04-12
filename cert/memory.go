package cert

import (
	"bytes"
	"encoding/pem"
	"errors"
	"io"

	"github.com/lafriks/pkcs8"
)

// LoadPEMFromReader loads a PEM-encoded certificate and private key from
// the io.Reader.
func LoadPEMFromReader(r io.Reader, opt ...Option) ([]byte, []byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	opts := opts(opt...)

	var cert, key []byte

	for {
		block, rest := pem.Decode(raw)
		if block == nil {
			break
		}

		switch block.Type {
		case PEMBlockCertificate:
			out := &bytes.Buffer{}
			if err = pem.Encode(out, block); err != nil {
				return nil, nil, err
			}

			cert = out.Bytes()
		case PEMBlockPrivateKey, PEMBlockRSAPrivateKey, PEMBlockECPrivateKey:
			out := &bytes.Buffer{}
			if err = pem.Encode(out, block); err != nil {
				return nil, nil, err
			}

			key = out.Bytes()
		case PEMBlockEncryptedPrivateKey:
			if len(opts.Password) == 0 {
				return nil, nil, errors.New("password required to decrypt private key")
			}

			p, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, opts.Password)
			if err != nil {
				return nil, nil, err
			}

			block, err = pemBlockForKey(p)
			if err != nil {
				return nil, nil, err
			}

			out := &bytes.Buffer{}
			if err = pem.Encode(out, block); err != nil {
				return nil, nil, err
			}

			key = out.Bytes()
		}

		raw = rest
	}

	return cert, key, nil
}
