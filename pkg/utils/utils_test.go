package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestGenerateAndValidateJWT_RoundTrip confirms the basic happy path: a
// token this package issues validates and returns the same claims.
func TestGenerateAndValidateJWT_RoundTrip(t *testing.T) {
	token, err := GenerateJWT("user-123", "user@example.com", "student", "school-1", time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT rejected a token this package just issued: %v", err)
	}
	if claims["user_id"] != "user-123" || claims["role"] != "student" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

// TestValidateJWT_RejectsExpiredToken confirms expiry is actually
// enforced, not just stored as a claim.
func TestValidateJWT_RejectsExpiredToken(t *testing.T) {
	token, err := GenerateJWT("user-123", "user@example.com", "student", "school-1", -time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	if _, err := ValidateJWT(token); err == nil {
		t.Fatal("expected an expired token to be rejected, got no error")
	}
}

// TestValidateJWT_RejectsForgedSecret is the core "can't forge a token
// for another user" check: a token built with the correct shape but a
// DIFFERENT HMAC secret (what an attacker without the real server
// secret would have to use) must be rejected, even though it claims
// role=admin.
func TestValidateJWT_RejectsForgedSecret(t *testing.T) {
	forged := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "attacker",
		"email":   "attacker@example.com",
		"role":    "super_admin",
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	})
	signed, err := forged.SignedString([]byte("not-the-real-server-secret"))
	if err != nil {
		t.Fatalf("failed to build forged token: %v", err)
	}
	if _, err := ValidateJWT(signed); err == nil {
		t.Fatal("expected a token signed with the wrong secret to be rejected, got no error")
	}
}

// TestValidateJWT_RejectsNoneAlgorithm guards against the classic "alg:
// none" JWT bypass, where a token with no signature at all is crafted
// and submitted hoping a lenient parser accepts it as valid.
func TestValidateJWT_RejectsNoneAlgorithm(t *testing.T) {
	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": "attacker",
		"role":    "super_admin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, err := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to build none-alg token: %v", err)
	}
	if _, err := ValidateJWT(signed); err == nil {
		t.Fatal("expected an alg:none token to be rejected, got no error")
	}
}

// TestValidateJWT_RejectsNonHS256Algorithm guards against JWT algorithm
// confusion: a token signed with the REAL server secret but under a
// different algorithm (HS384 here) must still be rejected, since
// ValidateJWT now pins jwt.WithValidMethods to HS256 explicitly rather
// than accepting whatever algorithm the token's own header claims.
func TestValidateJWT_RejectsNonHS256Algorithm(t *testing.T) {
	wrongAlg := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"user_id": "user-123",
		"role":    "super_admin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, err := wrongAlg.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("failed to build HS384 token: %v", err)
	}
	if _, err := ValidateJWT(signed); err == nil {
		t.Fatal("expected a non-HS256 token (even with the real secret) to be rejected, got no error")
	}
}

// TestValidateJWT_RejectsTamperedPayload confirms that flipping the role
// claim in an otherwise-valid token (without re-signing, as an attacker
// who can only read a token but not the server secret would have to try)
// invalidates the signature.
func TestValidateJWT_RejectsTamperedPayload(t *testing.T) {
	token, err := GenerateJWT("user-123", "user@example.com", "student", "school-1", time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected a 3-part JWT, got %d parts", len(parts))
	}
	// Swap in the payload segment from a token that claims admin, built
	// with a throwaway key - simulates an attacker splicing a
	// higher-privilege payload onto a validly-signed token's header/sig.
	adminPayloadToken, err := GenerateJWT("user-123", "user@example.com", "super_admin", "school-1", time.Hour)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	adminParts := strings.Split(adminPayloadToken, ".")
	tampered := parts[0] + "." + adminParts[1] + "." + parts[2]
	if _, err := ValidateJWT(tampered); err == nil {
		t.Fatal("expected a token with a swapped payload to be rejected, got no error")
	}
}
