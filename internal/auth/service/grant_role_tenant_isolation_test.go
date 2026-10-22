package service

import (
	"testing"

	"cbt-api/internal/auth/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- GrantRole/RevokeRole tenant isolation ----
//
// Before this fix, GrantRole/RevokeRole had no school check of any kind:
// FindUserByID looks up any user by raw ID regardless of school, and
// RevokeRole didn't even look the target user up at all before deleting
// their role grant. A role grant never changes which school the target
// user's own JWT claims (completeLoginWithRole always uses the target's
// own user.SchoolID - granting is purely additive), so this let a School
// A admin/super_admin elevate or strip privileges for a School B user
// without ever being School B staff themselves - tampering with another
// tenant's user roster, not just reading its data.

func setupGrantRoleTenantTestDB(t *testing.T) *gorm.DB {
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
		`CREATE TABLE user_role_grants (
			id TEXT PRIMARY KEY, user_id TEXT, role TEXT, granted_by TEXT, created_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newTenantTestAuthService(db *gorm.DB) *AuthService {
	return &AuthService{repo: repository.NewAuthRepository(db)}
}

func seedAuthUser(t *testing.T, db *gorm.DB, id, role, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO users (id, username, role, status, school_id) VALUES (?, ?, ?, ?, ?)",
		id, id, role, "active", schoolID,
	).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedRoleGrant(t *testing.T, db *gorm.DB, id, userID, role string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO user_role_grants (id, user_id, role) VALUES (?, ?, ?)",
		id, userID, role,
	).Error; err != nil {
		t.Fatalf("failed to seed role grant: %v", err)
	}
}

func TestGrantRole_CrossTenant_Rejected(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)
	seedAuthUser(t, db, "student-b", "student", "school-B")

	// School A's admin trying to grant "admin" to a School B user.
	if err := s.GrantRole("student-b", "admin", "admin-a", "school-A", false); err == nil {
		t.Fatal("expected School A's admin to be rejected granting a role to a School B user, got no error")
	}

	var count int64
	db.Table("user_role_grants").Where("user_id = ?", "student-b").Count(&count)
	if count != 0 {
		t.Fatal("a role grant was created by a cross-tenant GrantRole call that should have been rejected")
	}
}

func TestGrantRole_SameTenant_Allowed(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)
	seedAuthUser(t, db, "teacher-a", "teacher", "school-A")

	if err := s.GrantRole("teacher-a", "admin", "admin-a", "school-A", false); err != nil {
		t.Fatalf("expected School A's admin to grant a role to its own School A user, got error: %v", err)
	}

	var count int64
	db.Table("user_role_grants").Where("user_id = ? AND role = ?", "teacher-a", "admin").Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 role grant to be created, got %d", count)
	}
}

func TestGrantRole_PlatformOwner_Bypasses(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)
	seedAuthUser(t, db, "student-b", "student", "school-B")

	if err := s.GrantRole("student-b", "admin", "owner-1", "", true); err != nil {
		t.Fatalf("expected platform_owner to grant a role to any school's user, got error: %v", err)
	}
}

func TestGrantRole_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)
	seedAuthUser(t, db, "teacher-a", "teacher", "school-A")

	if err := s.GrantRole("teacher-a", "admin", "admin-ghost", "", false); err == nil {
		t.Fatal("expected a non-platform-owner caller with no school_id of their own to be rejected, got no error")
	}
}

func TestGrantRole_TargetUserNotFound_Rejected(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)

	if err := s.GrantRole("ghost-user", "admin", "admin-a", "school-A", false); err == nil {
		t.Fatal("expected an unknown target user to be rejected, got no error")
	}
}

func TestRevokeRole_CrossTenant_Rejected(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)
	seedAuthUser(t, db, "teacher-b", "teacher", "school-B")
	seedRoleGrant(t, db, "grant-1", "teacher-b", "admin")

	if err := s.RevokeRole("teacher-b", "admin", "school-A", false); err == nil {
		t.Fatal("expected School A's admin to be rejected revoking a School B user's role, got no error")
	}

	var count int64
	db.Table("user_role_grants").Where("user_id = ? AND role = ?", "teacher-b", "admin").Count(&count)
	if count != 1 {
		t.Fatal("the role grant was deleted by a cross-tenant RevokeRole call that should have been rejected")
	}
}

func TestRevokeRole_SameTenant_Allowed(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)
	seedAuthUser(t, db, "teacher-a", "teacher", "school-A")
	seedRoleGrant(t, db, "grant-1", "teacher-a", "admin")

	if err := s.RevokeRole("teacher-a", "admin", "school-A", false); err != nil {
		t.Fatalf("expected School A's admin to revoke its own School A user's role, got error: %v", err)
	}

	var count int64
	db.Table("user_role_grants").Where("user_id = ? AND role = ?", "teacher-a", "admin").Count(&count)
	if count != 0 {
		t.Fatalf("expected the role grant to be deleted, got %d remaining", count)
	}
}

func TestRevokeRole_PlatformOwner_Bypasses(t *testing.T) {
	db := setupGrantRoleTenantTestDB(t)
	s := newTenantTestAuthService(db)
	seedAuthUser(t, db, "teacher-b", "teacher", "school-B")
	seedRoleGrant(t, db, "grant-1", "teacher-b", "admin")

	if err := s.RevokeRole("teacher-b", "admin", "", true); err != nil {
		t.Fatalf("expected platform_owner to revoke any school's user role, got error: %v", err)
	}
}
