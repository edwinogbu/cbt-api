package service

import (
	"strings"
	"testing"

	"cbt-api/internal/models"
)

// TestGrantRole_RejectsPlatformOwner is a regression test for the
// multi-tenancy plan's Phase 1 security boundary: GrantRole must never
// be able to mint models.RolePlatformOwner, no matter who calls it,
// because it is the only route by which a school-scoped admin/
// super_admin could otherwise escalate themselves (or anyone else) to
// platform-wide authority. See models.RolePlatformOwner's doc comment
// and GrantRole's own comment for the full reasoning.
//
// Uses a zero-value *AuthService (nil repo and friends): this is valid
// here specifically because the platform_owner check in GrantRole runs
// and returns before the function ever touches s.repo - if that
// ordering ever changes, this test will panic on a nil pointer instead
// of silently passing, which is the correct failure mode (it would mean
// the guard moved somewhere the repo is already in use, at which point
// this test needs a real repository, not a signal to delete the check).
func TestGrantRole_RejectsPlatformOwner(t *testing.T) {
	s := &AuthService{}

	err := s.GrantRole("some-user-id", string(models.RolePlatformOwner), "some-admin-id", "school-1", false)
	if err == nil {
		t.Fatal("GrantRole(platform_owner) succeeded - this must always be rejected, see models.RolePlatformOwner")
	}
	if !strings.Contains(err.Error(), "platform_owner") {
		t.Fatalf("GrantRole(platform_owner) failed for the wrong reason (expected it to name platform_owner): %v", err)
	}
}
