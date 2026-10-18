package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// setupAuthTestDB mirrors setupExamTestDB (exam_service_test.go): an
// in-memory SQLite DB with a hand-written, SQLite-compatible subset of the
// schema actually touched by authorizeQuestionWrite/authorizeRoleOnly -
// just users and classes, since that's all those two methods read.
func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY, role TEXT, school_id TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE classes (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, session_id TEXT, class_level_id TEXT,
			class_arm_id TEXT, class_code TEXT, teacher_id TEXT, room_number TEXT, is_active INTEGER DEFAULT 1,
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

func newTestQuestionService(db *gorm.DB) *QuestionService {
	// BulkCreate/CreateQuestion/engine/queue aren't exercised by these
	// tests - only authorizeQuestionWrite/authorizeRoleOnly, which only
	// need classRepo/userRepo (constructed internally from db by
	// NewQuestionService).
	return NewQuestionService(nil, nil, db, nil, nil)
}

func seedUser(t *testing.T, db *gorm.DB, id, role, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO users (id, role, school_id) VALUES (?, ?, ?)", id, role, schoolID).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedClass(t *testing.T, db *gorm.DB, id, schoolID, teacherID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO classes (id, school_id, teacher_id) VALUES (?, ?, ?)", id, schoolID, teacherID).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func TestAuthorizeQuestionWrite_AdminBypassesClassOwnership(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "admin-1", "admin", "school-1")
	// No class row at all - an admin must not need one.
	if err := s.authorizeQuestionWrite("admin-1", "school-1", "nonexistent-class"); err != nil {
		t.Fatalf("expected admin to bypass class ownership check, got error: %v", err)
	}
}

func TestAuthorizeQuestionWrite_SuperAdminBypassesClassOwnership(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "super-1", "super_admin", "school-1")
	if err := s.authorizeQuestionWrite("super-1", "school-1", "nonexistent-class"); err != nil {
		t.Fatalf("expected super_admin to bypass class ownership check, got error: %v", err)
	}
}

func TestAuthorizeQuestionWrite_TeacherOwnsClass_Allowed(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "teacher-1", "teacher", "school-1")
	seedClass(t, db, "class-1", "school-1", "teacher-1")

	if err := s.authorizeQuestionWrite("teacher-1", "school-1", "class-1"); err != nil {
		t.Fatalf("expected teacher assigned to class-1 to be authorized, got error: %v", err)
	}
}

func TestAuthorizeQuestionWrite_TeacherDoesNotOwnClass_Denied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "teacher-1", "teacher", "school-1")
	seedUser(t, db, "teacher-2", "teacher", "school-1")
	seedClass(t, db, "class-1", "school-1", "teacher-2") // owned by teacher-2, not teacher-1

	err := s.authorizeQuestionWrite("teacher-1", "school-1", "class-1")
	if err == nil {
		t.Fatal("expected teacher-1 to be denied write access to a class they are not assigned to, got nil error")
	}
}

func TestAuthorizeQuestionWrite_TeacherClassInDifferentSchool_Denied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "teacher-1", "teacher", "school-1")
	// Class is really in school-2, but the request claims school-1 for it.
	seedClass(t, db, "class-1", "school-2", "teacher-1")

	err := s.authorizeQuestionWrite("teacher-1", "school-1", "class-1")
	if err == nil {
		t.Fatal("expected a school_id/class mismatch to be denied, got nil error")
	}
}

func TestAuthorizeQuestionWrite_StudentDenied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "student-1", "student", "school-1")
	seedClass(t, db, "class-1", "school-1", "")

	err := s.authorizeQuestionWrite("student-1", "school-1", "class-1")
	if err == nil {
		t.Fatal("expected a student to be denied question-bank write access, got nil error")
	}
}

func TestAuthorizeQuestionWrite_UnknownUserDenied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	// No user seeded at all.
	err := s.authorizeQuestionWrite("ghost-user", "school-1", "class-1")
	if err == nil {
		t.Fatal("expected an unknown user_id to be denied, got nil error")
	}
}

func TestAuthorizeQuestionWrite_EmptyUserIDDenied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	err := s.authorizeQuestionWrite("", "school-1", "class-1")
	if err == nil {
		t.Fatal("expected an empty user_id to be denied, got nil error")
	}
}

func TestAuthorizeQuestionWrite_TeacherMissingClassIDDenied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "teacher-1", "teacher", "school-1")

	err := s.authorizeQuestionWrite("teacher-1", "school-1", "")
	if err == nil {
		t.Fatal("expected a teacher request with no class_id to be denied, got nil error")
	}
}

func TestAuthorizeRoleOnly_TeacherAllowed(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "teacher-1", "teacher", "school-1")
	if err := s.authorizeRoleOnly("teacher-1"); err != nil {
		t.Fatalf("expected teacher to be allowed, got error: %v", err)
	}
}

func TestAuthorizeRoleOnly_StudentDenied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "student-1", "student", "school-1")
	if err := s.authorizeRoleOnly("student-1"); err == nil {
		t.Fatal("expected student to be denied, got nil error")
	}
}

func TestAuthorizeRoleOnly_ParentDenied(t *testing.T) {
	db := setupAuthTestDB(t)
	s := newTestQuestionService(db)
	seedUser(t, db, "parent-1", "parent", "school-1")
	if err := s.authorizeRoleOnly("parent-1"); err == nil {
		t.Fatal("expected parent to be denied, got nil error")
	}
}

func TestNormalizeQuestionText(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"What is the capital?", "what is the capital?", true},
		{"What   is  the capital?", "What is the capital?", true},
		{"  What is the capital?  ", "What is the capital?", true},
		{"What is the capital?", "What is the capital of France?", false},
	}
	for _, c := range cases {
		got := normalizeQuestionText(c.a) == normalizeQuestionText(c.b)
		if got != c.want {
			t.Errorf("normalizeQuestionText(%q) == normalizeQuestionText(%q): got %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
