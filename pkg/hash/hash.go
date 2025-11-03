package hash

import (
	"errors"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/matthewhartstonge/argon2"
)

var ErrMismatch = errors.New("hash and password mismatch")

type Argon struct {
	cfg argon2.Config
}

func NewArgon() *Argon {
	return &Argon{
		cfg: argon2.DefaultConfig(),
	}
}

func (a *Argon) Encode(v string) (string, error) {
	hashed, err := a.cfg.HashEncoded([]byte(v))

	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func (a *Argon) Compare(plain, hashed string) error {
	if ok, err := argon2.VerifyEncoded([]byte(plain), []byte(hashed)); !ok || err != nil {
		return ErrMismatch
	}
	return nil
}

type XXHash struct{}

func NewXXHash() *XXHash {
	return &XXHash{}
}

func (x *XXHash) Encode(v string) (string, error) {
	hashed := xxhash.Sum64String(v)
	return strconv.Itoa(int(hashed)), nil
}

func (x *XXHash) Compare(plain, hashed string) error {
	if ok := xxhash.Sum64([]byte(plain)) == xxhash.Sum64([]byte(hashed)); !ok {
		return ErrMismatch
	}
	return nil
}
