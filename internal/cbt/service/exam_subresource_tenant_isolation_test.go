package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"

	"gorm.io/gorm"
)

func seedExamAssignmentsTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS exam_assignments (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		exam_id TEXT NOT NULL, class_id TEXT, student_id TEXT, assigned_by TEXT NOT NULL,
		start_time DATETIME, end_time DATETIME, attempts_allowed INTEGER DEFAULT 1,
		status TEXT DEFAULT 'active', created_at DATETIME, updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("failed to create exam_assignments table: %v", err)
	}
}

// ---- Exam sub-resource operations: AddQuestionsToExam,
// BulkAddQuestionsToExam, RemoveQuestionFromExam, AssignExam,
// PreviewExam, GetClassResults, GetExamRankings, GetExamStatistics,
// ExportExamResults, GetExamAssignments.
//
// Before this fix, none of these ten took a school parameter at all.
// All ten are reachable on live routes with no role gate beyond
// AuthMiddleware. The mutations (Add/BulkAdd/Remove/Assign) let any
// authenticated caller tamper with another school's exam; the reads
// (Preview/ClassResults/Rankings/Statistics/Export/Assignments) leaked
// question content and student result data cross-tenant. All ten now
// call authorizeExamTenant first, the same helper GetExam/UpdateExam/
// DeleteExam already use. Reuses seedExamClass/seedExamForClass from
// exam_service_test.go.

func TestAddQuestionsToExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	err := s.AddQuestionsToExam(context.Background(), examA, []string{"q-1"}, "school-B", false)
	if err == nil {
		t.Fatal("expected School B's caller to be rejected adding questions to School A's exam, got no error")
	}
	if err.Error() != "exam not found" {
		t.Fatalf(`expected "exam not found", got: %v`, err)
	}
}

func TestBulkAddQuestionsToExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	err := s.BulkAddQuestionsToExam(context.Background(), examA, &dto.BulkAddQuestionsToExamRequest{QuestionIDs: []string{"q-1"}}, "school-B", false)
	if err == nil {
		t.Fatal("expected School B's caller to be rejected bulk-adding questions to School A's exam, got no error")
	}
	if err.Error() != "exam not found" {
		t.Fatalf(`expected "exam not found", got: %v`, err)
	}
}

func TestRemoveQuestionFromExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	err := s.RemoveQuestionFromExam(context.Background(), examA, "q-1", "school-B", false)
	if err == nil {
		t.Fatal("expected School B's caller to be rejected removing a question from School A's exam, got no error")
	}
	if err.Error() != "exam not found" {
		t.Fatalf(`expected "exam not found", got: %v`, err)
	}
}

func TestRemoveQuestionFromExam_SameTenant_Allowed(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	// No matching exam_questions row exists, so the DELETE affects 0
	// rows - what matters here is that the ownership gate itself does
	// not reject a legitimate same-school caller.
	if err := s.RemoveQuestionFromExam(context.Background(), examA, "q-1", "school-A", false); err != nil {
		t.Fatalf("expected School A's caller to pass the ownership gate for its own exam, got error: %v", err)
	}
}

func TestAssignExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	err := s.AssignExam(context.Background(), examA, &dto.AssignExamRequest{ClassID: classA}, "school-B", false)
	if err == nil {
		t.Fatal("expected School B's caller to be rejected assigning School A's exam, got no error")
	}
	if err.Error() != "exam not found" {
		t.Fatalf(`expected "exam not found", got: %v`, err)
	}
}

func TestAssignExam_SameTenant_Allowed(t *testing.T) {
	db := setupExamTestDB(t)
	seedExamAssignmentsTable(t, db)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if err := s.AssignExam(context.Background(), examA, &dto.AssignExamRequest{ClassID: classA}, "school-A", false); err != nil {
		t.Fatalf("expected School A's caller to assign its own exam, got error: %v", err)
	}
}

func TestPreviewExam_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.PreviewExam(context.Background(), examA, "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected previewing School A's exam, got no error")
	}
}

func TestPreviewExam_SameTenant_Allowed(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.PreviewExam(context.Background(), examA, "school-A", false); err != nil {
		t.Fatalf("expected School A's caller to preview its own exam, got error: %v", err)
	}
}

func TestPreviewExam_PlatformOwner_Bypasses(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.PreviewExam(context.Background(), examA, "", true); err != nil {
		t.Fatalf("expected platform_owner to preview any school's exam, got error: %v", err)
	}
}

func TestGetClassResults_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.GetClassResults(context.Background(), examA, classA, 1, 20, "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's exam results, got no error")
	}
}

func TestGetExamRankings_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.GetExamRankings(context.Background(), examA, 10, "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's exam rankings, got no error")
	}
}

func TestGetExamStatistics_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.GetExamStatistics(context.Background(), examA, "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's exam statistics, got no error")
	}
}

func TestGetExamStatistics_SameTenant_Allowed(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.GetExamStatistics(context.Background(), examA, "school-A", false); err != nil {
		t.Fatalf("expected School A's caller to read its own exam statistics, got error: %v", err)
	}
}

func TestExportExamResults_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, _, err := s.ExportExamResults(context.Background(), examA, classA, "csv", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected exporting School A's exam results, got no error")
	}
}

func TestGetExamAssignments_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	seedExamAssignmentsTable(t, db)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.GetExamAssignments(context.Background(), examA, "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's exam assignments, got no error")
	}
}

func TestGetExamAssignments_PlatformOwner_Bypasses(t *testing.T) {
	db := setupExamTestDB(t)
	seedExamAssignmentsTable(t, db)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	examA := seedExamForClass(t, db, classA)

	if _, err := s.GetExamAssignments(context.Background(), examA, "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's exam assignments, got error: %v", err)
	}
}
