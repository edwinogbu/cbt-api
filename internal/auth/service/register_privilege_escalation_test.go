package service

import (
	"testing"

	"cbt-api/internal/auth/dto"
)

// ---- Register: self-registration as admin for an EXISTING school ----
//
// Before this fix, role=admin was accepted here the same as teacher/
// parent: anyone who obtained an existing school's school_code
// (leaked, before a separate fix, to any teacher via GET /schools)
// could self-register as that school's admin and log in immediately -
// Login never checks user.Status, and IsActive is unconditionally true
// on this path - full administrative control over a real,
// already-operating tenant with zero human approval step. This is
// categorically different from the onboarding flow's admin creation,
// which always provisions a brand-new school for that admin rather
// than attaching to an existing one.
//
// The rejection happens before any repository is touched, so these
// tests use a zero-value *AuthService.

func TestRegister_AdminRole_Rejected(t *testing.T) {
	s := &AuthService{}

	_, err := s.Register(&dto.RegisterRequest{
		Role:       "admin",
		SchoolCode: "SCH-SOMEONEELSE",
		FirstName:  "Attacker",
		LastName:   "User",
		Email:      "attacker@example.com",
		Password:   "password123",
	})
	if err == nil {
		t.Fatal("expected self-registration with role=admin to be rejected, got no error")
	}
}
