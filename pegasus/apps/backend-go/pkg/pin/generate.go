package pin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// PINLength is the number of digits in a generated PIN.
const PINLength = 8

// Generate produces a cryptographically random PIN of PINLength digits.
// It returns the plaintext PIN string. Callers must bcrypt-hash separately
// for storage in the entity table.
func Generate() (string, error) {
	buf := make([]byte, PINLength)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("pin.Generate: crypto/rand failure at digit %d: %w", i, err)
		}
		buf[i] = '0' + byte(n.Int64())
	}
	return string(buf), nil
}

// SHA256Hex returns the lowercase hex-encoded SHA-256 digest of a plaintext
// PIN. This deterministic hash is stored in GlobalPins for uniqueness checks.
func SHA256Hex(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}
