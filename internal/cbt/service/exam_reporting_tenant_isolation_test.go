package service

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

// ---- GetTeacherSubjectResults, GetTeacherClassResults,
// GetSchoolPerformanceOverview, GetSubjectQuestionsForExam: before this
// fix, none of the four took a school parameter that was ever actually
// checked against the caller's own school. The first three leaked
// student result data cross-tenant via a client-supplied class ID or
// (for GetSchoolPerformanceOverview) a client-supplied school ID
// trusted outright with zero comparison to the caller's own verified
// school - exactly the pattern this engagement's mandate names
// directly. GetSubjectQuestionsForExam leaked question-bank content
// (text, options) the same way PreviewExam did. All four now call
// authorizeClassTenant or resolve via examEffectiveSchoolID. Reuses
// seedExamClass from exam_service_test.go.

func seedStudentForSchoolExamOverview(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO students (id, school_id) VALUES (?, ?)", id, schoolID).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

// seedSubjectsTable creates and seeds a minimal `subjects` table (not
// part of setupExamTestDB) so GetSubjectQuestionsForExam's downstream
// subject lookup doesn't itself error out and mask whether the
// ownership gate ran.
func seedSubjectsTable(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS subjects (
		id TEXT PRIMARY KEY, name TEXT, code TEXT, description TEXT,
		is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("failed to create subjects table: %v", err)
	}
	if err := db.Exec("INSERT INTO subjects (id, name) VALUES (?, ?)", id, "Test Subject").Error; err != nil {
		t.Fatalf("failed to seed subject: %v", err)
	}
}

func TestGetTeacherSubjectResults_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, err := s.GetTeacherSubjectResults(context.Background(), "subj-1", classA, "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's class results, got no error")
	}
}

func TestGetTeacherSubjectResults_SameTenant_Allowed(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, err := s.GetTeacherSubjectResults(context.Background(), "subj-1", classA, "school-A", false); err != nil {
		t.Fatalf("expected School A's caller to read its own class's results, got error: %v", err)
	}
}

func TestGetTeacherSubjectResults_PlatformOwner_Bypasses(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, err := s.GetTeacherSubjectResults(context.Background(), "subj-1", classA, "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's class results, got error: %v", err)
	}
}

func TestGetTeacherClassResults_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, err := s.GetTeacherClassResults(context.Background(), classA, "", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's class results, got no error")
	}
}

func TestGetTeacherClassResults_SameTenant_Allowed(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, err := s.GetTeacherClassResults(context.Background(), classA, "", "school-A", false); err != nil {
		t.Fatalf("expected School A's caller to read its own class's results, got error: %v", err)
	}
}

func TestGetSubjectQuestionsForExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	seedSubjectsTable(t, db, "subj-1")
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	_, err := s.GetSubjectQuestionsForExam(context.Background(), "subj-1", "term-1", "sess-1", classA, "school-B", false)
	if err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's subject questions, got no error")
	}
	if err.Error() != "class not found" {
		t.Fatalf(`expected the ownership gate's "class not found", got a different error: %v`, err)
	}
}

func TestGetSubjectQuestionsForExam_PlatformOwner_Bypasses(t *testing.T) {
	db := setupExamTestDB(t)
	seedSubjectsTable(t, db, "subj-1")
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, err := s.GetSubjectQuestionsForExam(context.Background(), "subj-1", "term-1", "sess-1", classA, "", true); err != nil {
		t.Fatalf("expected platform_owner to pass the ownership gate and read any school's subject questions, got error: %v", err)
	}
}

func TestGetSchoolPerformanceOverview_CrossTenant_IgnoresClientSchoolID(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	seedStudentForSchoolExamOverview(t, db, "student-a1", "school-A")
	seedStudentForSchoolExamOverview(t, db, "student-b1", "school-B")
	seedStudentForSchoolExamOverview(t, db, "student-b2", "school-B")

	// A School A caller naming School B's school_id in the request:
	// examEffectiveSchoolID must resolve to the caller's own ctxSchoolID
	// ("school-A"), never the client-supplied value, for any
	// non-platform-owner. School A has 1 student seeded, School B has 2 -
	// if the client-supplied value leaked through, total_students would
	// read 2, not 1.
	overview, err := s.GetSchoolPerformanceOverview(context.Background(), "school-B", "school-A", false)
	if err != nil {
		t.Fatalf("expected no error resolving to the caller's own school, got: %v", err)
	}
	data, ok := overview.(map[string]interface{})
	if !ok {
		t.Fatalf("expected a map response, got %T", overview)
	}
	if got := data["total_students"]; got != int64(1) {
		t.Fatalf("expected School A's own student count (1), got %v - the client-supplied school-B leaked through", got)
	}
}

func TestGetSchoolPerformanceOverview_PlatformOwner_HonorsClientSchoolID(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	seedStudentForSchoolExamOverview(t, db, "student-a1", "school-A")
	seedStudentForSchoolExamOverview(t, db, "student-b1", "school-B")
	seedStudentForSchoolExamOverview(t, db, "student-b2", "school-B")

	overview, err := s.GetSchoolPerformanceOverview(context.Background(), "school-B", "", true)
	if err != nil {
		t.Fatalf("expected platform_owner to target any school, got error: %v", err)
	}
	data, ok := overview.(map[string]interface{})
	if !ok {
		t.Fatalf("expected a map response, got %T", overview)
	}
	if got := data["total_students"]; got != int64(2) {
		t.Fatalf("expected platform_owner's explicit target school-B's student count (2), got %v", got)
	}
}

func TestGetSchoolPerformanceOverview_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)

	if _, err := s.GetSchoolPerformanceOverview(context.Background(), "school-B", "", false); err == nil {
		t.Fatal("expected a non-platform-owner caller with no school of their own to be rejected, got no error")
	}
}
