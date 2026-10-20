package service

import (
	"testing"

	"cbt-api/internal/academic/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---- AutoLinkParentOnRegister: cross-school linking ----
//
// Before this fix, FindStudentsByAdmissionNumber searched every school's
// students, with no scope at all. Parent registration only requires a
// school_code for the REGISTERING parent's own school - it says nothing
// about which school the target student must belong to. admission_no is
// unique platform-wide but is a short, often-sequential, non-secret
// identifier (printed on ID cards, report slips, etc.), so any
// registrant who learned another school's admission number could
// register as "parent" under their OWN, unrelated school and still get
// auto-linked to that other school's student - then read that student's
// full exam results via GetChildResults, with zero actual guardianship
// and zero relationship to that school at all. The fix scopes the
// lookup to the registering parent's own, server-validated school.

func setupAutoLinkTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE students (
			id TEXT PRIMARY KEY, user_id TEXT, school_id TEXT, class_id TEXT,
			admission_no TEXT, is_active INTEGER DEFAULT 1, status TEXT DEFAULT 'active',
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE parent_students (
			id TEXT PRIMARY KEY, parent_id TEXT, student_id TEXT, created_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func seedStudentForAutoLink(t *testing.T, db *gorm.DB, id, schoolID, admissionNo string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO students (id, school_id, admission_no) VALUES (?, ?, ?)",
		id, schoolID, admissionNo,
	).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

func countParentLinks(t *testing.T, db *gorm.DB, parentID string) int64 {
	t.Helper()
	var count int64
	if err := db.Table("parent_students").Where("parent_id = ?", parentID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count links: %v", err)
	}
	return count
}

func newTestParentService(db *gorm.DB) *ParentService {
	return &ParentService{
		parentRepo: repository.NewParentRepository(db),
		logger:     zap.NewNop(),
	}
}

func TestAutoLinkParentOnRegister_CrossSchoolStudent_NotLinked(t *testing.T) {
	db := setupAutoLinkTestDB(t)
	seedStudentForAutoLink(t, db, "student-b1", "school-b", "ADM001")

	s := newTestParentService(db)
	err := s.AutoLinkParentOnRegister("parent-a1", "ADM001", "school-a")
	if err == nil {
		t.Fatal("expected an error when no student with this admission number exists in the registrant's own school")
	}
	if got := countParentLinks(t, db, "parent-a1"); got != 0 {
		t.Fatalf("expected no parent-student link to be created across schools, got %d", got)
	}
}

func TestAutoLinkParentOnRegister_SameSchoolStudent_Linked(t *testing.T) {
	db := setupAutoLinkTestDB(t)
	seedStudentForAutoLink(t, db, "student-a1", "school-a", "ADM002")

	s := newTestParentService(db)
	if err := s.AutoLinkParentOnRegister("parent-a2", "ADM002", "school-a"); err != nil {
		t.Fatalf("expected a same-school link to succeed, got error: %v", err)
	}
	if got := countParentLinks(t, db, "parent-a2"); got != 1 {
		t.Fatalf("expected exactly 1 parent-student link, got %d", got)
	}
}
