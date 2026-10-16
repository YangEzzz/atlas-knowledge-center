package accounts

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// A versioned key ring keeps old ciphertext readable during key rotation.
type Crypt struct {
	active string
	keys   map[string]cipher.AEAD
}

func NewCrypt(active, raw string) (*Crypt, error) {
	var material map[string]string
	if json.Unmarshal([]byte(raw), &material) != nil || active == "" {
		return nil, errors.New("account encryption keys are not configured")
	}
	c := &Crypt{active: active, keys: map[string]cipher.AEAD{}}
	for id, value := range material {
		key, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(key) != 32 || strings.Contains(id, ":") || id == "" {
			return nil, errors.New("invalid account encryption key configuration")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCMWithRandomNonce(block)
		if err != nil {
			return nil, err
		}
		c.keys[id] = aead
	}
	if c.keys[active] == nil {
		return nil, errors.New("active account encryption key is missing")
	}
	return c, nil
}
func (c *Crypt) Seal(plain, aad string) string {
	return c.active + ":" + base64.StdEncoding.EncodeToString(c.keys[c.active].Seal(nil, nil, []byte(plain), []byte(aad)))
}
func (c *Crypt) Open(value, aad string) (string, error) {
	id, encoded, ok := strings.Cut(value, ":")
	aead := c.keys[id]
	if !ok || aead == nil {
		return "", errors.New("account data cannot be decrypted")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("account data cannot be decrypted")
	}
	plain, err := aead.Open(nil, nil, data, []byte(aad))
	if err != nil {
		return "", errors.New("account data cannot be decrypted")
	}
	return string(plain), nil
}
