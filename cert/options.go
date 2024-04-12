package cert

type options struct {
	Password []byte
}

func opts(opt ...Option) *options {
	o := &options{}
	for _, opt := range opt {
		opt.apply(o)
	}

	return o
}

// Option for loading a certificates.
type Option interface {
	apply(opt *options)
}

// Password to decrypt the private key.
type Password []byte

func (p Password) apply(o *options) {
	o.Password = p[:]
}
