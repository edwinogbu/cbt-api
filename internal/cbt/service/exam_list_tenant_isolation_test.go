package service

import (
	"context"
	"testing"
)

// ---- ListExams/ListExamsBySubject/ByTerm/BySession/ByClass: before this
// fix, none of the five took a school parameter at all, and the route
// backing them (/exams/list, /admin/exams/subject|term|session|class/:id)
// has only AuthMiddleware - no role gate. Any authenticated user, of any
// role, in any school, could list every exam on the entire platform
// (titles, subjects, schedules, instructions, pass marks).
//
// Reuses seedExamClass/seedExamForClass from exam_service_test.go.

func TestListExams_CrossTenant_NotVisible(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	classB := seedExamClass(t, db, "school-B")
	examA := seedExamForClass(t, db, classA)
	examB := seedExamForClass(t, db, classB)
	_ = examB

	resp, err := s.ListExams(context.Background(), 1, 20, "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected School A caller to see exactly 1 exam (their own), got %d", resp.Total)
	}
	if resp.Exams[0].ID != examA {
		t.Fatalf("expected the listed exam to be School A's own, got %s", resp.Exams[0].ID)
	}
}

func TestListExams_PlatformOwner_SeesAllSchools(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	classB := seedExamClass(t, db, "school-B")
	seedExamForClass(t, db, classA)
	seedExamForClass(t, db, classB)

	resp, err := s.ListExams(context.Background(), 1, 20, "", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected platform_owner to see both schools' exams, got %d", resp.Total)
	}
}

func TestListExamsBySubject_CrossTenant_NotVisible(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	classB := seedExamClass(t, db, "school-B")
	examA := seedExamForClass(t, db, classA)
	examB := seedExamForClass(t, db, classB)
	if err := db.Exec("UPDATE exams SET subject_id = 'subj-shared' WHERE id IN (?, ?)", examA, examB).Error; err != nil {
		t.Fatalf("failed to set subject_id: %v", err)
	}

	resp, err := s.ListExamsBySubject(context.Background(), "subj-shared", 1, 20, "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected School A caller to see exactly 1 exam sharing subj-shared, got %d", resp.Total)
	}
}

func TestListExamsByTerm_CrossTenant_NotVisible(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	classB := seedExamClass(t, db, "school-B")
	examA := seedExamForClass(t, db, classA)
	examB := seedExamForClass(t, db, classB)
	if err := db.Exec("UPDATE exams SET term_id = 'term-shared' WHERE id IN (?, ?)", examA, examB).Error; err != nil {
		t.Fatalf("failed to set term_id: %v", err)
	}

	resp, err := s.ListExamsByTerm(context.Background(), "term-shared", 1, 20, "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected School A caller to see exactly 1 exam sharing term-shared, got %d", resp.Total)
	}
}

func TestListExamsBySession_CrossTenant_NotVisible(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	classB := seedExamClass(t, db, "school-B")
	examA := seedExamForClass(t, db, classA)
	examB := seedExamForClass(t, db, classB)
	if err := db.Exec("UPDATE exams SET session_id = 'sess-shared' WHERE id IN (?, ?)", examA, examB).Error; err != nil {
		t.Fatalf("failed to set session_id: %v", err)
	}

	resp, err := s.ListExamsBySession(context.Background(), "sess-shared", 1, 20, "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected School A caller to see exactly 1 exam sharing sess-shared, got %d", resp.Total)
	}
}

func TestListExamsByClass_CrossTenant_NotVisible(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	seedExamForClass(t, db, classA)

	// A School B caller naming School A's own class directly: the join
	// on classes.school_id means this simply matches nothing rather than
	// leaking School A's exam.
	resp, err := s.ListExamsByClass(context.Background(), classA, 1, 20, "school-B", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected School B caller to see 0 exams naming School A's class, got %d", resp.Total)
	}

	resp, err = s.ListExamsByClass(context.Background(), classA, 1, 20, "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected School A caller to see its own class's exam, got %d", resp.Total)
	}
}
