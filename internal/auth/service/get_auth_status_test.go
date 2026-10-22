package service

import (
	"database/sql/driver"
	"testing"
	"time"

	academicRepo "cbt-api/internal/academic/repository"
	"cbt-api/internal/auth/repository"
	"cbt-api/internal/models"
	subRepo "cbt-api/internal/subscription/repository"

	gosqlite "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The subscription repository's FindCurrentBySchool query uses Postgres's
// NOW(), which plain SQLite doesn't provide. Registering it once as a
// scalar function (process-wide, per the driver's own doc comment) lets
// this test exercise the real repository query instead of only asserting
// "no error", the way check_subscription_test.go had to. gorm's
// github.com/glebarez/sqlite dialector runs on this same
// github.com/glebarez/go-sqlite driver (both register the "sqlite"
// database/sql driver name), not modernc.org/sqlite directly - importing
// that instead double-registers the driver name and panics.
func init() {
	gosqlite.MustRegisterScalarFunction("NOW", 0, func(_ *gosqlite.FunctionContext, _ []driver.Value) (driver.Value, error) {
		return time.Now().UTC().Format("2006-01-02 15:04:05"), nil
	})
}

// ---- GetAuthStatus: two super_admin gaps found via live testing ----
//
// A real super_admin's dashboard showed "no active subscription found"
// and getDashboardRoute(role) returned the hyphenated, non-existent
// "/dashboard/super-admin" (the frontend has no such route - only
// /dashboard/admin, which both admin and super_admin share). Both came
// from this one function only ever treating RoleAdmin as a "real"
// school admin, mirroring the same oversight already fixed in
// CheckSubscription (see check_subscription_test.go) and in the /admin
// route group's gate (api/routes/routes.go).

func setupGetAuthStatusTestDB(t *testing.T) *gorm.DB {
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
		`CREATE TABLE schools (
			id TEXT PRIMARY KEY, name TEXT, code TEXT, address TEXT, phone TEXT,
			email TEXT, logo TEXT, website TEXT, status TEXT, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
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

func newGetAuthStatusTestAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		repo:             repository.NewAuthRepository(db),
		schoolRepo:       academicRepo.NewSchoolRepository(db),
		subscriptionRepo: subRepo.NewSubscriptionRepository(db),
	}
}

func seedGetAuthStatusUser(t *testing.T, db *gorm.DB, id, role, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO users (id, username, role, status, school_id) VALUES (?, ?, ?, 'active', ?)",
		id, id, role, schoolID,
	).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedGetAuthStatusSchool(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO schools (id, name, code, status, is_active) VALUES (?, ?, ?, 'ACTIVE', 1)",
		id, "School "+id, id,
	).Error; err != nil {
		t.Fatalf("failed to seed school: %v", err)
	}
}

func seedGetAuthStatusActiveSubscription(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO subscriptions (id, school_id, status, tier, end_date, created_at) VALUES (?, ?, 'active', 'premium', datetime('now', '+30 days'), datetime('now'))",
		id, schoolID,
	).Error; err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}
}

func TestGetAuthStatus_SuperAdmin_SeesSchoolSubscription(t *testing.T) {
	db := setupGetAuthStatusTestDB(t)
	s := newGetAuthStatusTestAuthService(db)
	seedGetAuthStatusSchool(t, db, "school-A")
	seedGetAuthStatusUser(t, db, "super-a", string(models.RoleSuperAdmin), "school-A")
	seedGetAuthStatusActiveSubscription(t, db, "sub-a", "school-A")

	resp, err := s.GetAuthStatus("super-a")
	if err != nil {
		t.Fatalf("GetAuthStatus failed: %v", err)
	}
	if !resp.HasSubscription {
		t.Fatal("expected a super_admin to see their own school's active subscription, got HasSubscription=false")
	}
	if resp.SubscriptionTier != "premium" {
		t.Fatalf("expected tier 'premium', got %q", resp.SubscriptionTier)
	}
}

func TestGetAuthStatus_Admin_StillSeesSchoolSubscription(t *testing.T) {
	db := setupGetAuthStatusTestDB(t)
	s := newGetAuthStatusTestAuthService(db)
	seedGetAuthStatusSchool(t, db, "school-A")
	seedGetAuthStatusUser(t, db, "admin-a", string(models.RoleAdmin), "school-A")
	seedGetAuthStatusActiveSubscription(t, db, "sub-a", "school-A")

	resp, err := s.GetAuthStatus("admin-a")
	if err != nil {
		t.Fatalf("GetAuthStatus failed: %v", err)
	}
	if !resp.HasSubscription {
		t.Fatal("expected an admin to still see their own school's active subscription, got HasSubscription=false")
	}
}

func TestGetAuthStatus_Teacher_NeverSeesSubscription(t *testing.T) {
	db := setupGetAuthStatusTestDB(t)
	s := newGetAuthStatusTestAuthService(db)
	seedGetAuthStatusSchool(t, db, "school-A")
	seedGetAuthStatusUser(t, db, "teacher-a", string(models.RoleTeacher), "school-A")
	seedGetAuthStatusActiveSubscription(t, db, "sub-a", "school-A")

	resp, err := s.GetAuthStatus("teacher-a")
	if err != nil {
		t.Fatalf("GetAuthStatus failed: %v", err)
	}
	if resp.HasSubscription {
		t.Fatal("expected a non-admin role to never see subscription details, got HasSubscription=true")
	}
}

func TestGetAuthStatus_SuperAdmin_DashboardRouteMatchesRealFrontendRoute(t *testing.T) {
	db := setupGetAuthStatusTestDB(t)
	s := newGetAuthStatusTestAuthService(db)
	seedGetAuthStatusUser(t, db, "super-b", string(models.RoleSuperAdmin), "")

	resp, err := s.GetAuthStatus("super-b")
	if err != nil {
		t.Fatalf("GetAuthStatus failed: %v", err)
	}
	// The frontend has no /dashboard/super-admin or /dashboard/super_admin
	// route - super_admin shares /dashboard/admin with admin.
	if resp.DashboardRoute != "/dashboard/admin" {
		t.Fatalf("expected DashboardRoute '/dashboard/admin', got %q (a hyphenated or underscored super_admin-specific path does not exist in the frontend)", resp.DashboardRoute)
	}
}
