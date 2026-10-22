package service

import (
	"testing"

	academicRepo "cbt-api/internal/academic/repository"
	"cbt-api/internal/auth/repository"
	"cbt-api/internal/models"
	subRepo "cbt-api/internal/subscription/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- CheckSubscription: super_admin was missing from the school-ID
// resolution ----
//
// CheckSubscription resolves schoolID entirely server-side from the
// caller's own record - never a client-supplied value - so it was
// never a cross-tenant leak. But its role switch grouped only
// RoleTeacher/RoleAdmin as school-scoped roles that read user.SchoolID
// directly, missing RoleSuperAdmin even though super_admin is - like
// them, and unlike platform_owner - a school-level role with its own
// user.SchoolID. That gap failed closed ("unsupported user role")
// rather than leaking anything, but meant a super_admin could never
// check their own school's subscription.

func setupCheckSubscriptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY, username TEXT, email TEXT, password TEXT,
			first_name TEXT, last_name TEXT, phone_number TEXT, role TEXT,
			status TEXT, email_verified INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			two_factor_secret TEXT, two_factor_enabled INTEGER DEFAULT 0, last_login_at DATETIME,
			school_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			plain_password_encrypted TEXT
		)`,
		`CREATE TABLE subscriptions (
			id TEXT PRIMARY KEY, school_id TEXT, status TEXT, tier TEXT,
			features TEXT, end_date DATETIME, created_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newCheckSubscriptionTestAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		repo:              repository.NewAuthRepository(db),
		studentRepo:       academicRepo.NewStudentRepository(db),
		parentStudentRepo: academicRepo.NewParentRepository(db),
		subscriptionRepo:  subRepo.NewSubscriptionRepository(db),
	}
}

func seedCheckSubscriptionUser(t *testing.T, db *gorm.DB, id, role, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO users (id, username, role, status, school_id) VALUES (?, ?, ?, 'active', ?)",
		id, id, role, schoolID,
	).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedActiveSubscriptionForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO subscriptions (id, school_id, status, tier, end_date, created_at) VALUES (?, ?, 'active', 'premium', datetime('now', '+30 days'), datetime('now'))",
		id, schoolID,
	).Error; err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}
}

func TestCheckSubscription_SuperAdmin_ReportsOwnSchool(t *testing.T) {
	db := setupCheckSubscriptionTestDB(t)
	s := newCheckSubscriptionTestAuthService(db)
	seedCheckSubscriptionUser(t, db, "super-a", string(models.RoleSuperAdmin), "school-A")

	// FindCurrentBySchool's query uses Postgres's NOW(), which SQLite
	// (this test's stand-in) doesn't support - a pre-existing,
	// unrelated cross-dialect quirk the fix under test doesn't touch.
	// What this test actually checks is that a super_admin no longer
	// hits "unsupported user role": the role resolves to their own
	// school_id and the call returns no error, even though the
	// resulting subscription lookup itself can't succeed in this
	// SQLite harness.
	resp, err := s.CheckSubscription("super-a")
	if err != nil {
		t.Fatalf("expected a super_admin to resolve to their own school without an 'unsupported role' error, got: %v", err)
	}
	if resp.SchoolID != "school-A" {
		t.Fatalf("expected school-A, got %s", resp.SchoolID)
	}
}

func TestCheckSubscription_Admin_StillWorks(t *testing.T) {
	db := setupCheckSubscriptionTestDB(t)
	s := newCheckSubscriptionTestAuthService(db)
	seedCheckSubscriptionUser(t, db, "admin-a", string(models.RoleAdmin), "school-A")
	seedActiveSubscriptionForSchool(t, db, "sub-a", "school-A")

	resp, err := s.CheckSubscription("admin-a")
	if err != nil {
		t.Fatalf("expected an admin to check their own school's subscription, got error: %v", err)
	}
	if resp.SchoolID != "school-A" {
		t.Fatalf("expected school-A, got %s", resp.SchoolID)
	}
}

func TestCheckSubscription_PlatformOwner_StillUnsupported(t *testing.T) {
	db := setupCheckSubscriptionTestDB(t)
	s := newCheckSubscriptionTestAuthService(db)
	seedCheckSubscriptionUser(t, db, "owner-1", string(models.RolePlatformOwner), "")

	// platform_owner has no "own school" for this self-scoped endpoint
	// to resolve - correctly left out of the role switch, unlike
	// super_admin which does have one.
	if _, err := s.CheckSubscription("owner-1"); err == nil {
		t.Fatal("expected platform_owner (no own school) to be rejected by this self-scoped endpoint, got no error")
	}
}
