package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"
)

func TestSubjectResultSubmission_FullApprovalChain(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")

	scope := &dto.SubjectResultScopeRequest{ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1"}

	// Starts as draft (no row yet).
	status, err := svc.GetSubjectResultSubmissionStatus(context.Background(), "school-1", "class-1", "subj-1", "term-1", "session-1")
	if err != nil {
		t.Fatalf("GetSubjectResultSubmissionStatus failed: %v", err)
	}
	if status.Status != "draft" {
		t.Fatalf("expected initial status draft, got %s", status.Status)
	}

	// Teacher submits.
	submitted, err := svc.SubmitSubjectResults(context.Background(), "school-1", "teacher-1", false, scope)
	if err != nil {
		t.Fatalf("SubmitSubjectResults failed: %v", err)
	}
	if submitted.Status != "submitted" || submitted.Version != 1 {
		t.Fatalf("expected status=submitted version=1, got status=%s version=%d", submitted.Status, submitted.Version)
	}

	// HOD reviews.
	reviewed, err := svc.ReviewSubjectResults(context.Background(), "school-1", "hod-1", false, scope)
	if err != nil {
		t.Fatalf("ReviewSubjectResults failed: %v", err)
	}
	if reviewed.Status != "reviewed" || reviewed.ReviewedBy != "hod-1" {
		t.Fatalf("expected status=reviewed reviewed_by=hod-1, got status=%s reviewed_by=%s", reviewed.Status, reviewed.ReviewedBy)
	}

	// Admin approves.
	approved, err := svc.ApproveSubjectResults(context.Background(), "school-1", "admin-1", false, scope)
	if err != nil {
		t.Fatalf("ApproveSubjectResults failed: %v", err)
	}
	if approved.Status != "approved" || approved.ApprovedBy != "admin-1" {
		t.Fatalf("expected status=approved approved_by=admin-1, got status=%s approved_by=%s", approved.Status, approved.ApprovedBy)
	}

	// Submitting again on an approved (locked) scope must fail.
	if _, err := svc.SubmitSubjectResults(context.Background(), "school-1", "teacher-1", false, scope); err == nil {
		t.Error("expected re-submitting an approved scope to fail")
	}
}

func TestSubjectResultSubmission_ReturnUnlocksAndKeepsVersion(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	scope := &dto.SubjectResultScopeRequest{ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1"}

	submitted, err := svc.SubmitSubjectResults(context.Background(), "school-1", "teacher-1", false, scope)
	if err != nil {
		t.Fatalf("SubmitSubjectResults failed: %v", err)
	}

	returnScope := *scope
	returnScope.Notes = "Please re-check the Mathematics CA totals"
	returned, err := svc.ReturnSubjectResults(context.Background(), "school-1", "hod-1", false, &returnScope)
	if err != nil {
		t.Fatalf("ReturnSubjectResults failed: %v", err)
	}
	if returned.Status != "returned" {
		t.Fatalf("expected status=returned, got %s", returned.Status)
	}
	if returned.Version != submitted.Version {
		t.Errorf("Version changed on return (%d -> %d); returning must not consume a version", submitted.Version, returned.Version)
	}

	// A score-register save must now succeed with no correction_reason -
	// "returned" is not locked.
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	seedRegisterExam(t, db, "exam-ca-1", "subj-1", "class-1", "term-1", "session-1", "continuous_assessment", 20, nil)
	score := 15
	if _, err := svc.SaveScoreRegister(context.Background(), "school-1", "teacher-1", false, &dto.SaveScoreRegisterRequest{
		ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		ComponentCode: "continuous_assessment", Entries: []dto.ScoreEntryInput{{StudentID: "student-1", Score: &score}},
	}); err != nil {
		t.Fatalf("expected a save on a 'returned' (unlocked) scope to succeed, got: %v", err)
	}
}

func TestApproveSubjectResults_CanSkipReviewStep(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	scope := &dto.SubjectResultScopeRequest{ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1"}

	if _, err := svc.SubmitSubjectResults(context.Background(), "school-1", "teacher-1", false, scope); err != nil {
		t.Fatalf("SubmitSubjectResults failed: %v", err)
	}
	// Approve directly from "submitted" - review is optional per the spec.
	approved, err := svc.ApproveSubjectResults(context.Background(), "school-1", "admin-1", false, scope)
	if err != nil {
		t.Fatalf("expected ApproveSubjectResults to succeed straight from 'submitted' (review is optional), got: %v", err)
	}
	if approved.Status != "approved" {
		t.Fatalf("expected status=approved, got %s", approved.Status)
	}
}

func TestApproveSubjectResults_RejectsFromDraft(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	scope := &dto.SubjectResultScopeRequest{ClassID: "class-1", SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1"}

	if _, err := svc.ApproveSubjectResults(context.Background(), "school-1", "admin-1", false, scope); err == nil {
		t.Error("expected approving a draft (never submitted) scope to fail")
	}
}
