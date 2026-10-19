package orgemp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"
)

// Decrypter reverses the Selamnew payload encryption (Operation EncryptionService):
// AES-256-GCM, key = PBKDF2-SHA256(ENCRYPTION_KEY, utf8(hex(ENCRYPTION_SALT)), 1000, 32),
// wire format "<ivHex>:<cipherHex><16-byte tagHex>".
type Decrypter struct {
	key []byte
}

// NewDecrypter returns nil (encryption off) when key or iv is empty.
func NewDecrypter(key, iv, salt string) (*Decrypter, error) {
	if key == "" || iv == "" {
		return nil, nil
	}
	k, err := pbkdf2.Key(sha256.New, key, nodeHexUTF8(salt), 1000, 32)
	if err != nil {
		return nil, err
	}
	return &Decrypter{key: k}, nil
}

// LooksEncrypted mirrors EncryptionService.isEncrypted.
func LooksEncrypted(s string) bool {
	iv, data, ok := strings.Cut(s, ":")
	return ok && iv != "" && len(data) >= 32 && !strings.Contains(data, ":")
}

// Decrypt returns the plaintext of an encrypted payload string.
func (d *Decrypter) Decrypt(s string) (string, error) {
	ivHex, dataHex, ok := strings.Cut(s, ":")
	if !ok {
		return "", errors.New("invalid encrypted text format")
	}
	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return "", err
	}
	data, err := hex.DecodeString(dataHex) // ciphertext || tag — what GCM.Open expects
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(d.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, iv, data, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// nodeHexUTF8 reproduces Node's Buffer.from(s, 'hex').toString('utf8'): decode hex
// pairs up to the first invalid one, then decode as UTF-8 with one U+FFFD per
// invalid byte. (With the default non-hex salt this is what the services derive.)
func nodeHexUTF8(s string) []byte {
	var raw []byte
	for i := 0; i+1 < len(s); i += 2 {
		b, err := hex.DecodeString(s[i : i+2])
		if err != nil {
			break
		}
		raw = append(raw, b[0])
	}
	var sb strings.Builder
	for len(raw) > 0 {
		r, size := utf8.DecodeRune(raw)
		sb.WriteRune(r) // RuneError for each invalid byte
		raw = raw[size:]
	}
	return []byte(sb.String())
}
