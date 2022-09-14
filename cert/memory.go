package cert

import (
	"bytes"
	"encoding/pem"
	"io"
)

// LoadPEMFromReader loads a PEM-encoded certificate and private key from
// the io.Reader.
func LoadPEMFromReader(r io.Reader) ([]byte, []byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	var cert, key []byte
	for {
		block, rest := pem.Decode(raw)
		if block == nil {
			break
		}
		if block.Type == PEMBlockCertificate {
			out := &bytes.Buffer{}
			if err = pem.Encode(out, block); err != nil {
				return nil, nil, err
			}
			cert = out.Bytes()
		} else if block.Type == PEMBlockRSAPrivateKey || block.Type == PEMBlockECPrivateKey {
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
