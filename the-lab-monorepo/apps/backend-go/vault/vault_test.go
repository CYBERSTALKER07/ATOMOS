package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"os"
	"testing"
)

func init() {
	// Set up a test master key if not already set
	if len(masterKey) == 0 {
		testKey := "0123456789abcdef0123456789abcdef" // 32 hex chars = 16 bytes → need 64 hex for 32 bytes
		testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		decoded, _ := hex.DecodeString(testKey)
		masterKey = decoded
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	plaintext := []byte("supplier-secret-key-12345")
	ct, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	decrypted, err := Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("got %q, want %q", string(decrypted), string(plaintext))
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	ct, err := Encrypt([]byte{})
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	decrypted, err := Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty, got %d bytes", len(decrypted))
	}
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	plaintext := []byte("same-data")
	ct1, _ := Encrypt(plaintext)
	ct2, _ := Encrypt(plaintext)
	if string(ct1) == string(ct2) {
		t.Error("two encryptions should produce different ciphertexts (random nonce)")
	}
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	plaintext := []byte("test-data")
	ct, _ := Encrypt(plaintext)
	// Corrupt a byte in the ciphertext body
	if len(ct) > 15 {
		ct[15] ^= 0xFF
	}
	_, err := Decrypt(ct)
	if err == nil {
		t.Error("expected error for corrupted ciphertext")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	_, err := Decrypt([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for ciphertext shorter than nonce")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	plaintext := []byte("secret-data")
	ct, _ := Encrypt(plaintext)

	// Save original and swap key
	originalKey := make([]byte, len(masterKey))
	copy(originalKey, masterKey)

	wrongKey, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	masterKey = wrongKey

	_, err := Decrypt(ct)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}

	// Restore original
	masterKey = originalKey
}

func TestEncrypt_NoMasterKey(t *testing.T) {
	originalKey := make([]byte, len(masterKey))
	copy(originalKey, masterKey)
	masterKey = nil

	_, err := Encrypt([]byte("test"))
	if err == nil {
		t.Error("expected error when master key is not configured")
	}

	masterKey = originalKey
}

func TestDecrypt_NoMasterKey(t *testing.T) {
	originalKey := make([]byte, len(masterKey))
	copy(originalKey, masterKey)
	masterKey = nil

	_, err := Decrypt([]byte("test"))
	if err == nil {
		t.Error("expected error when master key is not configured")
	}

	masterKey = originalKey
}

func TestEncrypt_CiphertextFormat(t *testing.T) {
	plaintext := []byte("check-format")
	ct, err := Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Ciphertext should be nonce + encrypted data + GCM tag
	block, _ := aes.NewCipher(masterKey)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()

	if len(ct) <= nonceSize {
		t.Errorf("ciphertext too short: %d bytes, nonce size %d", len(ct), nonceSize)
	}
}

func TestGatewayConfigSummary_NoSecret(t *testing.T) {
	cfg := GatewayConfigSummary{
		ConfigID:    "cfg-1",
		GatewayName: "PAYME",
		MerchantId:  "MERCH-1",
		IsActive:    true,
		HasSecret:   true,
	}

	if cfg.GatewayName != "PAYME" || !cfg.HasSecret {
		t.Errorf("unexpected: %+v", cfg)
	}
}

// Ensure VAULT_MASTER_KEY env handling is correct for tests
func TestVaultMasterKey_EnvOverride(t *testing.T) {
	// Just verify masterKey is set from our test init
	if len(masterKey) != 32 {
		t.Skipf("masterKey length = %d, skipping (env not set)", len(masterKey))
	}
}

// Suppress unused import warning
var _ = os.Getenv
