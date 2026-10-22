package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestGetSchoolBroadSheet_PromotionReviewTalliesRecommendations proves
// Section C's counts are the exact same SystemRecommendation the
// Promotion Examination Report Card already computes for each student,
// aggregated across the school - not a second, independently computed
// eligibility check.
func TestGetSchoolBroadSheet_PromotionReviewTalliesRecommendations(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Greenfield Academy", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "Final Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES ('teacher-1','Ada','Lovelace','teacher')").Error; err != nil {
		t.Fatalf("seed teacher failed: %v", err)
	}
	if err := db.Exec("INSERT INTO teacher_subject_assignments (id, school_id, teacher_id, subject_id, class_id, session_id, is_active) VALUES ('a1','school-1','teacher-1','subj-1','class-1','session-1',1)").Error; err != nil {
		t.Fatalf("seed assignment failed: %v", err)
	}

	examID := "exam-1"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)

	// student-1: well above the builtin 50% threshold -> eligible.
	user1 := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", user1, schoolID, classID)
	seedReportCardResult(t, db, "student-1", examID, 75, true)

	// student-2: below threshold -> not eligible / requires review.
	user2 := uuid.NewString()
	seedReportCardStudent(t, db, "student-2", user2, schoolID, classID)
	seedReportCardResult(t, db, "student-2", examID, 30, true)

	// student-3: no result recorded at all -> incomplete/blocked.
	user3 := uuid.NewString()
	seedReportCardStudent(t, db, "student-3", user3, schoolID, classID)

	resp, err := svc.GetSchoolBroadSheet(context.Background(), schoolID, false, "", termID, sessionID, "promotion")
	if err != nil {
		t.Fatalf("GetSchoolBroadSheet failed: %v", err)
	}
	if resp.PromotionReview == nil {
		t.Fatal("expected PromotionReview to be populated for reportType=promotion")
	}
	pr := resp.PromotionReview
	if pr.EligibleCount != 1 {
		t.Errorf("EligibleCount = %d, want 1 (student-1)", pr.EligibleCount)
	}
	if pr.RequiresReviewCount != 1 {
		t.Errorf("RequiresReviewCount = %d, want 1 (student-2)", pr.RequiresReviewCount)
	}
	if pr.IncompleteBlockedCount != 1 {
		t.Errorf("IncompleteBlockedCount = %d, want 1 (student-3, no result at all)", pr.IncompleteBlockedCount)
	}
	if pr.DecisionStatus != "Pending" {
		t.Errorf("DecisionStatus = %q, want Pending (no FinalDecision recorded for anyone yet)", pr.DecisionStatus)
	}
	if pr.EligibilityRulesApplied == "" {
		t.Error("EligibilityRulesApplied should describe the (built-in default) policy, got empty string")
	}

	// A non-promotion report must not populate Section C at all.
	termlyResp, err := svc.GetSchoolBroadSheet(context.Background(), schoolID, false, "", termID, sessionID, "termly")
	if err != nil {
		t.Fatalf("GetSchoolBroadSheet(termly) failed: %v", err)
	}
	if termlyResp.PromotionReview != nil {
		t.Error("expected PromotionReview to be nil for a termly report")
	}
}
