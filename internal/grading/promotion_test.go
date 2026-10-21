package grading

import (
	"context"
	"testing"

	"cbt-api/internal/models"

	"gorm.io/gorm"
)

func TestResolvePromotionPolicy_NoConfigFallsBackToBuiltin(t *testing.T) {
	db := setupGradingTestDB(t)
	svc := NewService(db)
	resolved := svc.ResolvePromotionPolicy(context.Background(), "school-1", "level-1")
	if resolved.Source != "builtin_default" {
		t.Errorf("Source = %s, want builtin_default", resolved.Source)
	}
	if resolved.OverallThreshold != BuiltinPromotionThreshold {
		t.Errorf("OverallThreshold = %.1f, want %.1f", resolved.OverallThreshold, BuiltinPromotionThreshold)
	}
	if len(resolved.CoreSubjectIDs) != 0 {
		t.Errorf("CoreSubjectIDs = %v, want none configured", resolved.CoreSubjectIDs)
	}
}

func TestResolvePromotionPolicy_NilServiceFallsBackToBuiltin(t *testing.T) {
	var svc *Service
	resolved := svc.ResolvePromotionPolicy(context.Background(), "school-1", "")
	if resolved.Source != "builtin_default" {
		t.Errorf("a nil Service should always fall back to builtin_default, got Source=%s", resolved.Source)
	}
}

func seedSchoolWidePromotionPolicy(t *testing.T, db *gorm.DB, schoolID string, threshold float64, coreSubjects []string) {
	t.Helper()
	policy := models.PromotionPolicy{
		SchoolID: schoolID, ScopeKey: "school", OverallThreshold: threshold,
		CoreSubjectIDs: models.StringArray(coreSubjects), CoreSubjectPassMark: 45, IsActive: true,
	}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("failed to seed promotion policy: %v", err)
	}
}

func TestResolvePromotionPolicy_UsesSchoolConfiguredThreshold(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWidePromotionPolicy(t, db, "school-1", 60, []string{"subj-math"})
	svc := NewService(db)

	resolved := svc.ResolvePromotionPolicy(context.Background(), "school-1", "")
	if resolved.Source != "school" {
		t.Fatalf("Source = %s, want school", resolved.Source)
	}
	if resolved.OverallThreshold != 60 {
		t.Errorf("OverallThreshold = %.1f, want this school's configured 60", resolved.OverallThreshold)
	}
	if len(resolved.CoreSubjectIDs) != 1 || resolved.CoreSubjectIDs[0] != "subj-math" {
		t.Errorf("CoreSubjectIDs = %v, want [subj-math]", resolved.CoreSubjectIDs)
	}
}

func TestResolvePromotionPolicy_DifferentSchoolsAreIsolated(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWidePromotionPolicy(t, db, "school-1", 60, nil)
	svc := NewService(db)

	resolved := svc.ResolvePromotionPolicy(context.Background(), "school-2", "")
	if resolved.Source != "builtin_default" {
		t.Errorf("school-2 must never see school-1's configured policy, got Source=%s", resolved.Source)
	}
}

// ---- EvaluatePromotion: pure function ----

func TestEvaluatePromotion_AboveThresholdNoCoreSubjects_Eligible(t *testing.T) {
	policy := ResolvedPromotionPolicy{OverallThreshold: 50}
	subjects := []SubjectStanding{
		{SubjectID: "s1", Percentage: 70, IsComplete: true},
		{SubjectID: "s2", Percentage: 55, IsComplete: true},
	}
	got := EvaluatePromotion(policy, 62.5, subjects)
	if got.Recommendation != "ELIGIBLE FOR REVIEW" {
		t.Errorf("Recommendation = %q, want ELIGIBLE FOR REVIEW", got.Recommendation)
	}
	if !got.ResultsComplete {
		t.Error("ResultsComplete = false, want true")
	}
}

func TestEvaluatePromotion_BelowThreshold_NotEligible(t *testing.T) {
	policy := ResolvedPromotionPolicy{OverallThreshold: 50}
	subjects := []SubjectStanding{{SubjectID: "s1", Percentage: 40, IsComplete: true}}
	got := EvaluatePromotion(policy, 40, subjects)
	if got.Recommendation != "NOT ELIGIBLE FOR REVIEW" {
		t.Errorf("Recommendation = %q, want NOT ELIGIBLE FOR REVIEW", got.Recommendation)
	}
	if len(got.Reasons) == 0 {
		t.Error("expected at least one reason explaining the below-threshold result")
	}
}

func TestEvaluatePromotion_CoreSubjectBelowPassMark_NotEligibleEvenIfOverallPasses(t *testing.T) {
	policy := ResolvedPromotionPolicy{OverallThreshold: 50, CoreSubjectIDs: []string{"subj-math"}, CoreSubjectPassMark: 45}
	subjects := []SubjectStanding{
		{SubjectID: "subj-math", Percentage: 30, IsComplete: true}, // core subject, below its pass mark
		{SubjectID: "subj-other", Percentage: 90, IsComplete: true},
	}
	// Overall average (60) clears the 50% threshold, but the core
	// subject rule must still block eligibility.
	got := EvaluatePromotion(policy, 60, subjects)
	if got.Recommendation != "NOT ELIGIBLE FOR REVIEW" {
		t.Errorf("Recommendation = %q, want NOT ELIGIBLE FOR REVIEW (core subject failed despite a passing overall average)", got.Recommendation)
	}
}

func TestEvaluatePromotion_MissingCoreSubjectResult_NotEligible(t *testing.T) {
	policy := ResolvedPromotionPolicy{OverallThreshold: 50, CoreSubjectIDs: []string{"subj-math"}, CoreSubjectPassMark: 45}
	// subj-math isn't in the student's subject list at all.
	subjects := []SubjectStanding{{SubjectID: "subj-other", Percentage: 90, IsComplete: true}}
	got := EvaluatePromotion(policy, 90, subjects)
	if got.Recommendation != "NOT ELIGIBLE FOR REVIEW" {
		t.Errorf("Recommendation = %q, want NOT ELIGIBLE FOR REVIEW when a required core subject has no result at all", got.Recommendation)
	}
}

func TestEvaluatePromotion_IncompleteSubject_CannotEvaluate(t *testing.T) {
	policy := ResolvedPromotionPolicy{OverallThreshold: 50}
	subjects := []SubjectStanding{
		{SubjectID: "s1", Percentage: 80, IsComplete: true},
		{SubjectID: "s2", IsComplete: false}, // still pending
	}
	got := EvaluatePromotion(policy, 80, subjects)
	if got.Recommendation != "INCOMPLETE RESULTS - CANNOT EVALUATE" {
		t.Errorf("Recommendation = %q, want INCOMPLETE RESULTS - CANNOT EVALUATE", got.Recommendation)
	}
	if got.ResultsComplete {
		t.Error("ResultsComplete = true, want false")
	}
}

func TestEvaluatePromotion_NoSubjectsAtAll_CannotEvaluate(t *testing.T) {
	got := EvaluatePromotion(ResolvedPromotionPolicy{OverallThreshold: 50}, 0, nil)
	if got.Recommendation != "INCOMPLETE RESULTS - CANNOT EVALUATE" {
		t.Errorf("Recommendation = %q, want INCOMPLETE RESULTS - CANNOT EVALUATE for a student with no subjects at all", got.Recommendation)
	}
}
