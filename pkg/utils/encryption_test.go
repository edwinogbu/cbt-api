package utils

import (
	"strings"
	"testing"
)

// ---- AES master key: hardcoded secret, same bug class as jwtSecret ----
//
// masterKey (used to "encrypt" every user's recoverable password - see
// SavePlainPassword/GetPlainPassword in user_repo.go) was hardcoded to a
// literal 32-byte string checked into this public source file, bypassing
// ENCRYPTION_MASTER_KEY entirely. Anyone who can read this repository
// could decrypt every stored password with it. This is the exact same
// bug class pkg/utils/utils.go's jwtSecret was already fixed for.

func TestResolveMasterKey_UsesEnvVarWhenSet(t *testing.T) {
	realKey := "a-real-32-byte-per-deployment-ky" // exactly 32 bytes
	if len(realKey) != 32 {
		t.Fatalf("test fixture key must be 32 bytes, got %d", len(realKey))
	}
	key, err := resolveMasterKey(realKey, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(key) != realKey {
		t.Fatalf("expected the env-provided key to be used, got %q", string(key))
	}
}

func TestResolveMasterKey_FallsBackToInsecureDefaultOutsideProduction(t *testing.T) {
	key, err := resolveMasterKey("", "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(key) != insecureDefaultMasterKey {
		t.Fatalf("expected the insecure default outside production, got %q", string(key))
	}
}

func TestResolveMasterKey_RefusesInsecureDefaultInProduction(t *testing.T) {
	_, err := resolveMasterKey("", "production")
	if err == nil {
		t.Fatal("expected an error when ENCRYPTION_MASTER_KEY is unset in production, got none")
	}
	if !strings.Contains(err.Error(), "ENCRYPTION_MASTER_KEY") {
		t.Fatalf("expected the error to name ENCRYPTION_MASTER_KEY, got: %v", err)
	}
}

func TestResolveMasterKey_RefusesWrongLengthKey(t *testing.T) {
	_, err := resolveMasterKey("too-short", "development")
	if err == nil {
		t.Fatal("expected an error for a key that isn't exactly 32 bytes, got none")
	}
}

func TestEncryptDecryptPlainPassword_RoundTrip(t *testing.T) {
	encrypted, err := EncryptPlainPassword("StudentPass123!")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if encrypted == "StudentPass123!" {
		t.Fatal("encrypted output must not equal the plaintext")
	}
	decrypted, err := DecryptPlainPassword(encrypted)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != "StudentPass123!" {
		t.Fatalf("expected round-trip to recover the original password, got %q", decrypted)
	}
}
