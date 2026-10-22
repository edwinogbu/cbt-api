package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"
)

// These tests confirm the new Academic Broad Sheet endpoints added in
// this round - the score-register remark and the Class Broad Sheet
// review workflow - reject a cross-school caller exactly like every
// other exam sub-resource already does (see
// exam_subresource_tenant_isolation_test.go), since each one's first
// line is the same shared authorizeClassTenant(ctx, classID, schoolID,
// isPlatformOwner) helper.

func TestSaveScoreRegisterRemark_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	err := s.SaveScoreRegisterRemark(context.Background(), "school-B", "actor-1", false, &dto.SaveScoreRegisterRemarkRequest{
		ClassID: classA, SubjectID: "subj-1", TermID: "term-1", SessionID: "session-1",
		StudentID: "student-1", Remark: "attempted cross-tenant write",
	})
	if err == nil {
		t.Fatal("expected School B's caller to be rejected writing a remark on School A's class, got no error")
	}
	if err.Error() != "class not found" {
		t.Fatalf(`expected "class not found", got: %v`, err)
	}
}

func TestSaveClassBroadSheetReviewSummary_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	_, err := s.SaveClassBroadSheetReviewSummary(context.Background(), "school-B", "actor-1", false, &dto.SaveClassBroadSheetReviewRequest{
		ClassID: classA, ReportType: "promotion", SessionID: "session-1", Summary: "attempted cross-tenant write",
	})
	if err == nil {
		t.Fatal("expected School B's caller to be rejected saving a review summary on School A's class, got no error")
	}
	if err.Error() != "class not found" {
		t.Fatalf(`expected "class not found", got: %v`, err)
	}
}

func TestSubmitClassBroadSheetReview_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")
	// Give School A's own review a real summary first, so a failure
	// below can only be the tenant check, not the "summary required" rule.
	if _, err := s.SaveClassBroadSheetReviewSummary(context.Background(), "school-A", "actor-1", false, &dto.SaveClassBroadSheetReviewRequest{
		ClassID: classA, ReportType: "promotion", SessionID: "session-1", Summary: "real summary",
	}); err != nil {
		t.Fatalf("seeding School A's own review failed: %v", err)
	}

	req := &dto.ClassBroadSheetReviewActionRequest{ClassID: classA, ReportType: "promotion", SessionID: "session-1"}
	if _, err := s.SubmitClassBroadSheetReview(context.Background(), "school-B", "actor-1", false, req); err == nil {
		t.Fatal("expected School B's caller to be rejected submitting School A's class review, got no error")
	} else if err.Error() != "class not found" {
		t.Fatalf(`expected "class not found", got: %v`, err)
	}

	if _, err := s.ApproveClassBroadSheetReview(context.Background(), "school-B", "actor-1", false, req); err == nil {
		t.Fatal("expected School B's caller to be rejected approving School A's class review, got no error")
	} else if err.Error() != "class not found" {
		t.Fatalf(`expected "class not found", got: %v`, err)
	}

	withNotes := &dto.ClassBroadSheetReviewActionRequest{ClassID: classA, ReportType: "promotion", SessionID: "session-1", Notes: "notes"}
	if _, err := s.ReturnClassBroadSheetReview(context.Background(), "school-B", "actor-1", false, withNotes); err == nil {
		t.Fatal("expected School B's caller to be rejected returning School A's class review, got no error")
	} else if err.Error() != "class not found" {
		t.Fatalf(`expected "class not found", got: %v`, err)
	}
}

func TestExportClassBroadSheet_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	s := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, _, _, err := s.ExportClassBroadSheet(context.Background(), "school-B", false, classA, "promotion", "", "session-1", "pdf"); err == nil {
		t.Fatal("expected School B's caller to be rejected exporting School A's class broad sheet, got no error")
	} else if err.Error() != "class not found" {
		t.Fatalf(`expected "class not found", got: %v`, err)
	}
}
