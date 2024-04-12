package cert

import (
	"bytes"
	"testing"

	"github.com/go-quicktest/qt"
)

var testCerts = map[string]struct {
	Buf      []byte
	Password []byte
}{
	"ECPrivateKey": {
		Buf: []byte(`-----BEGIN CERTIFICATE-----
MIIBsjCCAVegAwIBAgIUcsdc1RWWC4jFpM99yHdW+UZWy9IwCgYIKoZIzj0EAwIw
HDELMAkGA1UEBhMCTFYxDTALBgNVBAgMBFJpZ2EwIBcNMjMxMDA5MjIxNDE0WhgP
MjEyMzA5MTUyMjE0MTRaMBwxCzAJBgNVBAYTAkxWMQ0wCwYDVQQIDARSaWdhMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5RcfSL0OpInZenK+2Ig6njgQt2KiHBwA
RtlHBhhYXjfR2cuY7nHRj7aX5HO97o4kVpcgJrj2/8GSUIYpQ7MwZaN1MHMwHQYD
VR0OBBYEFF9QAjCY99mQiKzdBXh9YjMzXEP4MB8GA1UdIwQYMBaAFF9QAjCY99mQ
iKzdBXh9YjMzXEP4MAsGA1UdDwQEAwIHgDATBgNVHSUEDDAKBggrBgEFBQcDAjAP
BgNVHREECDAGggR0ZXN0MAoGCCqGSM49BAMCA0kAMEYCIQCHn2M0KJSpAFd3jcsu
XZNebZ4EdX/aJTWMC+dRHPaQggIhAKOH4OCJdESow4Mr4a1yHmKFj+86BWfP/fFV
zItKSlmL
-----END CERTIFICATE-----
-----BEGIN EC PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgogbp77P3S3/T3o/j
fCHYG5moMYK3k/OhxRgQAQ/XI0ChRANCAATlFx9IvQ6kidl6cr7YiDqeOBC3YqIc
HABG2UcGGFheN9HZy5jucdGPtpfkc73ujiRWlyAmuPb/wZJQhilDszBl
-----END EC PRIVATE KEY-----`),
	},
	"PrivateKey": {
		Buf: []byte(`-----BEGIN CERTIFICATE-----
MIIBsjCCAVegAwIBAgIUcsdc1RWWC4jFpM99yHdW+UZWy9IwCgYIKoZIzj0EAwIw
HDELMAkGA1UEBhMCTFYxDTALBgNVBAgMBFJpZ2EwIBcNMjMxMDA5MjIxNDE0WhgP
MjEyMzA5MTUyMjE0MTRaMBwxCzAJBgNVBAYTAkxWMQ0wCwYDVQQIDARSaWdhMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5RcfSL0OpInZenK+2Ig6njgQt2KiHBwA
RtlHBhhYXjfR2cuY7nHRj7aX5HO97o4kVpcgJrj2/8GSUIYpQ7MwZaN1MHMwHQYD
VR0OBBYEFF9QAjCY99mQiKzdBXh9YjMzXEP4MB8GA1UdIwQYMBaAFF9QAjCY99mQ
iKzdBXh9YjMzXEP4MAsGA1UdDwQEAwIHgDATBgNVHSUEDDAKBggrBgEFBQcDAjAP
BgNVHREECDAGggR0ZXN0MAoGCCqGSM49BAMCA0kAMEYCIQCHn2M0KJSpAFd3jcsu
XZNebZ4EdX/aJTWMC+dRHPaQggIhAKOH4OCJdESow4Mr4a1yHmKFj+86BWfP/fFV
zItKSlmL
-----END CERTIFICATE-----
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgogbp77P3S3/T3o/j
fCHYG5moMYK3k/OhxRgQAQ/XI0ChRANCAATlFx9IvQ6kidl6cr7YiDqeOBC3YqIc
HABG2UcGGFheN9HZy5jucdGPtpfkc73ujiRWlyAmuPb/wZJQhilDszBl
-----END PRIVATE KEY-----`),
	},
	"EncryptedPrivateKey": {
		Buf: []byte(`-----BEGIN CERTIFICATE-----
MIIBsjCCAVegAwIBAgIUcsdc1RWWC4jFpM99yHdW+UZWy9IwCgYIKoZIzj0EAwIw
HDELMAkGA1UEBhMCTFYxDTALBgNVBAgMBFJpZ2EwIBcNMjMxMDA5MjIxNDE0WhgP
MjEyMzA5MTUyMjE0MTRaMBwxCzAJBgNVBAYTAkxWMQ0wCwYDVQQIDARSaWdhMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5RcfSL0OpInZenK+2Ig6njgQt2KiHBwA
RtlHBhhYXjfR2cuY7nHRj7aX5HO97o4kVpcgJrj2/8GSUIYpQ7MwZaN1MHMwHQYD
VR0OBBYEFF9QAjCY99mQiKzdBXh9YjMzXEP4MB8GA1UdIwQYMBaAFF9QAjCY99mQ
iKzdBXh9YjMzXEP4MAsGA1UdDwQEAwIHgDATBgNVHSUEDDAKBggrBgEFBQcDAjAP
BgNVHREECDAGggR0ZXN0MAoGCCqGSM49BAMCA0kAMEYCIQCHn2M0KJSpAFd3jcsu
XZNebZ4EdX/aJTWMC+dRHPaQggIhAKOH4OCJdESow4Mr4a1yHmKFj+86BWfP/fFV
zItKSlmL
-----END CERTIFICATE-----
-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHsMFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAgZLH6rDVE/XwICCAAw
DAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEEuut1zE+prK1sE29S9WeRkEgZC1
SLgn1Ty191AY+WU0FwSOV+IW+yCPpSV0k97SXgrYI2VOzMS/+wgQqCtV0ZDJkCJb
urzYQCZWj+PDVea/Kmy0Kq0Ts9nr/AjPjxGYNM5OG6GtZMWqLiW6dgmGJBLG5ZoZ
k3PAdb2lvRP5Qax6LIRtHBK6t9hhTc+yjyyDccTj1axM73l5LkYffZYDSonhBi8=
-----END ENCRYPTED PRIVATE KEY-----`),
		Password: []byte("testtest"),
	},
}

func TestParseTLSCertificateFromReader(t *testing.T) {
	for name, test := range testCerts {
		t.Run(name, func(t *testing.T) {
			cert, err := ParseTLSCertificateFromReader(bytes.NewBuffer(test.Buf), Password(test.Password))
			qt.Assert(t, qt.IsNil(err))
			qt.Check(t, qt.IsNotNil(cert.PrivateKey))
			qt.Check(t, qt.HasLen(cert.Certificate, 1))
		})
	}
}

func TestDERBytesToPEMBlocks(t *testing.T) {
	for name, test := range testCerts {
		t.Run(name, func(t *testing.T) {
			cert, err := ParseTLSCertificateFromReader(bytes.NewBuffer(test.Buf), Password(test.Password))
			qt.Assert(t, qt.IsNil(err))

			crt, key, err := DERBytesToPEMBlocks(cert.Certificate[0], cert.PrivateKey, Password(test.Password))
			qt.Assert(t, qt.IsNil(err))
			qt.Check(t, qt.IsNotNil(crt))
			qt.Check(t, qt.IsNotNil(key))
		})
	}
}
