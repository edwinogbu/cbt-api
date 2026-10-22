package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"
)

func TestClassBroadSheetReview_SaveSubmitApproveFlow(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	schoolID, classID, sessionID := "school-1", "class-1", "session-1"
	seedSchool(t, db, schoolID, "Greenfield Academy", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")

	ctx := context.Background()
	scope := &dto.SaveClassBroadSheetReviewRequest{
		ClassID: classID, ReportType: "promotion", TermID: "", SessionID: sessionID,
		Summary: "Strong overall performance; two students need extra support in Mathematics.",
	}

	saved, err := svc.SaveClassBroadSheetReviewSummary(ctx, schoolID, "teacher-1", false, scope)
	if err != nil {
		t.Fatalf("SaveClassBroadSheetReviewSummary failed: %v", err)
	}
	if saved.Status != "draft" {
		t.Errorf("Status after save = %q, want draft", saved.Status)
	}

	// Submitting with an empty summary must be rejected.
	emptyScope := &dto.SaveClassBroadSheetReviewRequest{ClassID: classID, ReportType: "promotion", SessionID: sessionID, Summary: ""}
	if _, err := svc.SaveClassBroadSheetReviewSummary(ctx, schoolID, "teacher-1", false, emptyScope); err != nil {
		t.Fatalf("saving an empty summary over the draft should succeed: %v", err)
	}
	actionReq := &dto.ClassBroadSheetReviewActionRequest{ClassID: classID, ReportType: "promotion", SessionID: sessionID}
	if _, err := svc.SubmitClassBroadSheetReview(ctx, schoolID, "teacher-1", false, actionReq); err == nil {
		t.Fatal("expected submitting an empty summary to be rejected")
	}

	// Put the real summary back, then submit for real.
	if _, err := svc.SaveClassBroadSheetReviewSummary(ctx, schoolID, "teacher-1", false, scope); err != nil {
		t.Fatalf("re-saving summary failed: %v", err)
	}
	submitted, err := svc.SubmitClassBroadSheetReview(ctx, schoolID, "teacher-1", false, actionReq)
	if err != nil {
		t.Fatalf("SubmitClassBroadSheetReview failed: %v", err)
	}
	if submitted.Status != "submitted" || submitted.SubmittedAt == nil {
		t.Errorf("after submit: status=%q submittedAt=%v, want submitted + non-nil", submitted.Status, submitted.SubmittedAt)
	}

	approved, err := svc.ApproveClassBroadSheetReview(ctx, schoolID, "hod-1", false, actionReq)
	if err != nil {
		t.Fatalf("ApproveClassBroadSheetReview failed: %v", err)
	}
	if approved.Status != "approved" || approved.ReviewerID != "hod-1" {
		t.Errorf("after approve: status=%q reviewerID=%q, want approved/hod-1", approved.Status, approved.ReviewerID)
	}

	// An approved review can no longer be edited.
	if _, err := svc.SaveClassBroadSheetReviewSummary(ctx, schoolID, "teacher-1", false, scope); err == nil {
		t.Fatal("expected editing an approved review to be rejected")
	}

	// GetClassBroadSheet must fold this same review state into its own response.
	resp, err := svc.GetClassBroadSheet(ctx, schoolID, false, classID, "promotion", "", sessionID)
	if err != nil {
		t.Fatalf("GetClassBroadSheet failed: %v", err)
	}
	if resp.Review == nil || resp.Review.Status != "approved" {
		t.Fatalf("GetClassBroadSheet's embedded Review = %+v, want status=approved", resp.Review)
	}
}

func TestClassBroadSheetReview_ReturnRequiresNotes(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	schoolID, classID, sessionID := "school-1", "class-1", "session-1"
	seedSchool(t, db, schoolID, "Greenfield Academy", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")

	ctx := context.Background()
	if _, err := svc.SaveClassBroadSheetReviewSummary(ctx, schoolID, "teacher-1", false, &dto.SaveClassBroadSheetReviewRequest{
		ClassID: classID, ReportType: "promotion", SessionID: sessionID, Summary: "Summary text.",
	}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	actionReq := &dto.ClassBroadSheetReviewActionRequest{ClassID: classID, ReportType: "promotion", SessionID: sessionID}
	if _, err := svc.SubmitClassBroadSheetReview(ctx, schoolID, "teacher-1", false, actionReq); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	if _, err := svc.ReturnClassBroadSheetReview(ctx, schoolID, "hod-1", false, actionReq); err == nil {
		t.Fatal("expected returning without notes to be rejected")
	}
	withNotes := &dto.ClassBroadSheetReviewActionRequest{ClassID: classID, ReportType: "promotion", SessionID: sessionID, Notes: "Please double-check the Mathematics scores."}
	returned, err := svc.ReturnClassBroadSheetReview(ctx, schoolID, "hod-1", false, withNotes)
	if err != nil {
		t.Fatalf("ReturnClassBroadSheetReview failed: %v", err)
	}
	if returned.Status != "returned" || returned.ReturnNotes == "" {
		t.Errorf("after return: status=%q returnNotes=%q, want returned + non-empty", returned.Status, returned.ReturnNotes)
	}
}
