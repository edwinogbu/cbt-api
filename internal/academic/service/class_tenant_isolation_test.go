package service

import (
	"context"
	"testing"

	"cbt-api/internal/academic/dto"
	"cbt-api/internal/academic/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Phase 2: academic-module tenant isolation (Class, representative
// of Class/Session/Term/ClassLevel/ClassArm/Student, which all share
// effectiveSchoolID/authorizeSchoolOwnership from tenant.go) ----
//
// Before this fix: GetByID/Update/Delete took no school parameter at
// all (any authenticated user who knew or enumerated another school's
// class ID could read, modify, or delete it); Create/List/Search/Stats
// trusted a client-supplied school_id outright; BulkDeleteClasses had no
// school check whatsoever.

func setupClassTenantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE classes (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, session_id TEXT, class_level_id TEXT,
			class_arm_id TEXT, class_code TEXT, teacher_id TEXT, room_number TEXT, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE class_levels (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, name TEXT, level_number INTEGER, category TEXT,
			promotion_to TEXT, sort_order INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE class_arms (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, name TEXT, arm_code TEXT, capacity INTEGER DEFAULT 40,
			room_number TEXT, sort_order INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE academic_sessions (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, name TEXT, start_date DATETIME, end_date DATETIME,
			is_current INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newTestClassService(db *gorm.DB) *ClassService {
	return NewClassService(
		repository.NewClassRepository(db),
		repository.NewClassLevelRepository(db),
		repository.NewClassArmRepository(db),
		repository.NewSessionRepository(db),
	)
}

func seedClassForSchool(t *testing.T, db *gorm.DB, id, schoolID, sessionID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO classes (id, school_id, session_id, class_code, room_number) VALUES (?, ?, ?, ?, ?)",
		id, schoolID, sessionID, id+"-code", "",
	).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func seedSessionForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO academic_sessions (id, school_id, name) VALUES (?, ?, ?)", id, schoolID, "2026/2027").Error; err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
}

func seedClassLevelForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO class_levels (id, school_id, name) VALUES (?, ?, ?)", id, schoolID, "JSS1").Error; err != nil {
		t.Fatalf("failed to seed class level: %v", err)
	}
}

func seedClassArmForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO class_arms (id, school_id, name) VALUES (?, ?, ?)", id, schoolID, "A").Error; err != nil {
		t.Fatalf("failed to seed class arm: %v", err)
	}
}

// --- GetByID: cross-tenant read, same-tenant read, platform_owner, missing scope ---

func TestClassGetByID_CrossTenant_Rejected(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedClassForSchool(t, db, "class-1", "school-A", "sess-A")

	if _, err := s.GetByID("class-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's class, got no error")
	}
}

func TestClassGetByID_SameTenant_Allowed(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedClassForSchool(t, db, "class-1", "school-A", "sess-A")

	resp, err := s.GetByID("class-1", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own class, got error: %v", err)
	}
	if resp.ID != "class-1" {
		t.Fatalf("expected class-1, got %s", resp.ID)
	}
}

func TestClassGetByID_PlatformOwner_Bypasses(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedClassForSchool(t, db, "class-1", "school-A", "sess-A")

	if _, err := s.GetByID("class-1", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's class, got error: %v", err)
	}
}

func TestClassGetByID_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedClassForSchool(t, db, "class-1", "school-A", "sess-A")

	// A legacy/unscoped account must never be treated as "sees everything".
	if _, err := s.GetByID("class-1", "", false); err == nil {
		t.Fatal("expected an empty school_id on a non-platform-owner caller to be rejected, got no error")
	}
}

// --- Create: a client-supplied school_id in the request body must never
// establish which school the new class belongs to ---

func TestClassCreate_ClientSuppliedSchoolIDInBody_Ignored(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	// ClassLevelRepository/ClassArmRepository.FindByID both require a
	// parseable UUID, unlike ClassRepository/SessionRepository - so
	// these three IDs must be real UUIDs, not plain test strings.
	sessID, levelID, armID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	seedSessionForSchool(t, db, sessID, "school-A")
	seedClassLevelForSchool(t, db, levelID, "school-A")
	seedClassArmForSchool(t, db, armID, "school-A")

	req := &dto.CreateClassRequest{
		SchoolID:     "school-B", // attacker-controlled: claims School B
		SessionID:    sessID,
		ClassLevelID: levelID,
		ClassArmID:   armID,
	}
	resp, err := s.Create(req, "school-A", false) // caller is actually School A
	if err != nil {
		t.Fatalf("expected School A's create to succeed, got error: %v", err)
	}
	if resp.SchoolID != "school-A" {
		t.Fatalf("expected the new class to belong to the caller's real school (school-A), got %q - client-supplied school_id was honored", resp.SchoolID)
	}
}

func TestClassCreate_CrossSchoolSessionReference_Rejected(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	sessID, levelID, armID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	// Session belongs to School B, but the caller is School A.
	seedSessionForSchool(t, db, sessID, "school-B")
	seedClassLevelForSchool(t, db, levelID, "school-A")
	seedClassArmForSchool(t, db, armID, "school-A")

	req := &dto.CreateClassRequest{
		SessionID:    sessID,
		ClassLevelID: levelID,
		ClassArmID:   armID,
	}
	if _, err := s.Create(req, "school-A", false); err == nil {
		t.Fatal("expected a cross-school session reference to be rejected, got no error")
	}
}

// --- Update / Delete: cross-tenant rejected ---

func TestClassUpdate_CrossTenant_Rejected(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedClassForSchool(t, db, "class-1", "school-A", "sess-A")
	newRoom := "Hijacked Room"

	if _, err := s.Update("class-1", &dto.UpdateClassRequest{RoomNumber: newRoom}, "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected updating School A's class, got no error")
	}

	var room string
	db.Table("classes").Select("room_number").Where("id = ?", "class-1").Scan(&room)
	if room == newRoom {
		t.Fatal("class was modified by a cross-tenant update that should have been rejected")
	}
}

func TestClassDelete_CrossTenant_Rejected(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedClassForSchool(t, db, "class-1", "school-A", "sess-A")

	if err := s.Delete("class-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected deleting School A's class, got no error")
	}

	var count int64
	db.Table("classes").Where("id = ? AND deleted_at IS NULL", "class-1").Count(&count)
	if count != 1 {
		t.Fatal("class was deleted by a cross-tenant delete that should have been rejected")
	}
}

// --- BulkDeleteClasses: a mixed-school ID list must only delete the
// caller's own IDs, never silently deleting another school's rows and
// never erroring out the whole batch either ---

func TestClassBulkDelete_MixedSchools_OnlyOwnIDsDeleted(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedClassForSchool(t, db, "class-a1", "school-A", "sess-A")
	seedClassForSchool(t, db, "class-a2", "school-A", "sess-A")
	seedClassForSchool(t, db, "class-b1", "school-B", "sess-B")

	deleted, err := s.BulkDeleteClasses(context.Background(), []string{"class-a1", "class-a2", "class-b1"}, "school-A", false)
	if err != nil {
		t.Fatalf("expected bulk delete to succeed for owned IDs, got error: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deletions (School A's own classes only), got %d", deleted)
	}

	var bCount int64
	db.Table("classes").Where("id = ? AND deleted_at IS NULL", "class-b1").Count(&bCount)
	if bCount != 1 {
		t.Fatal("School B's class was deleted by School A's bulk-delete request - cross-tenant bulk delete")
	}
}

// --- GetBySession: a session ID from another school must not leak that
// school's classes just because the URL names no school at all ---

func TestClassGetBySession_CrossTenant_Rejected(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedSessionForSchool(t, db, "sess-B", "school-B")
	seedClassForSchool(t, db, "class-b1", "school-B", "sess-B")

	if _, err := s.GetBySession("sess-B", "school-A", false); err == nil {
		t.Fatal("expected School A's caller to be rejected reading classes via School B's session ID, got no error")
	}
}

func TestClassGetBySession_PlatformOwner_Bypasses(t *testing.T) {
	db := setupClassTenantTestDB(t)
	s := newTestClassService(db)
	seedSessionForSchool(t, db, "sess-B", "school-B")
	seedClassForSchool(t, db, "class-b1", "school-B", "sess-B")

	if _, err := s.GetBySession("sess-B", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's session classes, got error: %v", err)
	}
}
