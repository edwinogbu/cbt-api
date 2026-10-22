package grading

import (
	"context"
	"testing"

	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// ---- ComputeSubjectAssessment: pure-function, no DB ----

func TestComputeSubjectAssessment_DefaultModeMatchesExistingPooledBehavior(t *testing.T) {
	// This is exactly what buildSubjectTermlyResult did before this
	// package existed: sum every exam's raw score and marks for the
	// subject, regardless of ExamType, no weighting at all.
	scheme := ResolvedScheme{Source: DefaultAssessmentMode}
	exams := []ScoredExam{
		{ExamType: "continuous_assessment", TotalMarks: 20, TotalScore: 18, FullyGraded: true},
		{ExamType: "main_exam", TotalMarks: 80, TotalScore: 60, FullyGraded: true},
	}
	got := ComputeSubjectAssessment(scheme, exams)
	wantPct := (18.0 + 60.0) / (20.0 + 80.0) * 100 // 78%
	if got.TotalPercent != wantPct {
		t.Errorf("TotalPercent = %.4f, want %.4f (plain pooled sum)", got.TotalPercent, wantPct)
	}
	if !got.IsComplete {
		t.Errorf("IsComplete = false, want true when exams exist under the default mode")
	}
	if got.IsProvisional {
		t.Errorf("IsProvisional = true, want false - every exam is FullyGraded")
	}
}

func TestComputeSubjectAssessment_DefaultModeNoExamsIsIncompleteNotZero(t *testing.T) {
	// A subject with no results yet must never be presented as a 0% -
	// that would misrepresent "not assessed" as "failed".
	got := ComputeSubjectAssessment(ResolvedScheme{Source: DefaultAssessmentMode}, nil)
	if got.IsComplete {
		t.Errorf("IsComplete = true, want false when there are no exams at all")
	}
	if got.TotalPercent != 0 {
		t.Errorf("TotalPercent = %.2f, want 0 as the placeholder (caller must check IsComplete, not infer failure from this)", got.TotalPercent)
	}
}

func TestComputeSubjectAssessment_DefaultModeProvisionalWhenAnyExamNotFullyGraded(t *testing.T) {
	scheme := ResolvedScheme{Source: DefaultAssessmentMode}
	exams := []ScoredExam{
		{ExamType: "main_exam", TotalMarks: 100, TotalScore: 70, FullyGraded: false}, // essay still pending
	}
	got := ComputeSubjectAssessment(scheme, exams)
	if !got.IsProvisional {
		t.Errorf("IsProvisional = false, want true - the only contributing exam has an essay portion still awaiting manual marking")
	}
}

func TestComputeSubjectAssessment_WeightedSchemeAppliesConfiguredWeights(t *testing.T) {
	scheme := ResolvedScheme{
		Source: "school", MissingPolicy: "renormalize",
		Components: []ComponentSpec{
			{Code: "ca", Name: "Continuous Assessment", Weight: 30, ExamTypes: []string{"continuous_assessment"}, Aggregation: "pooled"},
			{Code: "exam", Name: "Exam", Weight: 70, ExamTypes: []string{"main_exam"}, Aggregation: "pooled"},
		},
	}
	exams := []ScoredExam{
		{ExamType: "continuous_assessment", TotalMarks: 20, TotalScore: 20, FullyGraded: true}, // 100%
		{ExamType: "main_exam", TotalMarks: 100, TotalScore: 50, FullyGraded: true},             // 50%
	}
	got := ComputeSubjectAssessment(scheme, exams)
	want := 100.0*0.30 + 50.0*0.70 // 65
	if got.TotalPercent != want {
		t.Errorf("TotalPercent = %.4f, want %.4f (30%% CA + 70%% Exam)", got.TotalPercent, want)
	}
	if !got.IsComplete {
		t.Errorf("IsComplete = false, want true - both components have results")
	}
	if len(got.Components) != 2 {
		t.Fatalf("Components length = %d, want 2", len(got.Components))
	}
}

func TestComputeSubjectAssessment_MissingComponentRenormalizes(t *testing.T) {
	// Only the CA component has a result yet (exam not yet sat). Under
	// "renormalize", the subject total should scale the CA score up to
	// fill 100% of the weight actually present, not silently treat the
	// missing exam as a 0.
	scheme := ResolvedScheme{
		Source: "school", MissingPolicy: "renormalize",
		Components: []ComponentSpec{
			{Code: "ca", Name: "CA", Weight: 30, ExamTypes: []string{"continuous_assessment"}, Aggregation: "pooled"},
			{Code: "exam", Name: "Exam", Weight: 70, ExamTypes: []string{"main_exam"}, Aggregation: "pooled"},
		},
	}
	exams := []ScoredExam{
		{ExamType: "continuous_assessment", TotalMarks: 20, TotalScore: 20, FullyGraded: true}, // 100%
	}
	got := ComputeSubjectAssessment(scheme, exams)
	if got.IsComplete {
		t.Errorf("IsComplete = true, want false - the exam component has no result yet")
	}
	if got.TotalPercent != 100 {
		t.Errorf("TotalPercent = %.2f, want 100 (the only present component scored 100%%, renormalized to fill the full weight)", got.TotalPercent)
	}
	if got.Components[1].HasResults {
		t.Errorf("exam component HasResults = true, want false")
	}
}

func TestComputeSubjectAssessment_MissingComponentZeroPolicyDoesNotRenormalize(t *testing.T) {
	scheme := ResolvedScheme{
		Source: "school", MissingPolicy: "zero",
		Components: []ComponentSpec{
			{Code: "ca", Name: "CA", Weight: 30, ExamTypes: []string{"continuous_assessment"}, Aggregation: "pooled"},
			{Code: "exam", Name: "Exam", Weight: 70, ExamTypes: []string{"main_exam"}, Aggregation: "pooled"},
		},
	}
	exams := []ScoredExam{
		{ExamType: "continuous_assessment", TotalMarks: 20, TotalScore: 20, FullyGraded: true}, // 100%
	}
	got := ComputeSubjectAssessment(scheme, exams)
	want := 100.0 * 0.30 // 30 - missing exam component contributes 0, no renormalizing
	if got.TotalPercent != want {
		t.Errorf("TotalPercent = %.4f, want %.4f under missing_policy=zero", got.TotalPercent, want)
	}
}

func TestComputeSubjectAssessment_BestAggregationTakesHighestScore(t *testing.T) {
	scheme := ResolvedScheme{
		Source: "school", MissingPolicy: "renormalize",
		Components: []ComponentSpec{
			{Code: "test", Name: "Tests", Weight: 100, ExamTypes: []string{"continuous_assessment"}, Aggregation: "best"},
		},
	}
	exams := []ScoredExam{
		{ExamType: "continuous_assessment", TotalMarks: 10, TotalScore: 4, FullyGraded: true},  // 40%
		{ExamType: "continuous_assessment", TotalMarks: 10, TotalScore: 9, FullyGraded: true},  // 90%
		{ExamType: "continuous_assessment", TotalMarks: 10, TotalScore: 6, FullyGraded: true},  // 60%
	}
	got := ComputeSubjectAssessment(scheme, exams)
	if got.TotalPercent != 90 {
		t.Errorf("TotalPercent = %.2f, want 90 (the best of the three test scores)", got.TotalPercent)
	}
}

func TestComputeSubjectAssessment_UnmatchedExamTypeIsIgnored(t *testing.T) {
	// An exam whose ExamType isn't mapped to any configured component
	// must not silently leak into the total.
	scheme := ResolvedScheme{
		Source: "school", MissingPolicy: "zero",
		Components: []ComponentSpec{
			{Code: "exam", Name: "Exam", Weight: 100, ExamTypes: []string{"main_exam"}, Aggregation: "pooled"},
		},
	}
	exams := []ScoredExam{
		{ExamType: "main_exam", TotalMarks: 100, TotalScore: 80, FullyGraded: true},
		{ExamType: "mock_exam", TotalMarks: 100, TotalScore: 10, FullyGraded: true}, // not configured, must be ignored
	}
	got := ComputeSubjectAssessment(scheme, exams)
	if got.TotalPercent != 80 {
		t.Errorf("TotalPercent = %.2f, want 80 (the unmapped mock_exam score must not affect the total)", got.TotalPercent)
	}
}

// ---- Service.ResolveScheme: DB-backed resolution ----

func seedSchoolWideScheme(t *testing.T, db *gorm.DB, schoolID string) {
	t.Helper()
	scheme := models.AssessmentScheme{
		SchoolID: schoolID, ScopeKey: "school", MissingPolicy: "renormalize", IsActive: true,
		Components: []models.AssessmentComponent{
			{Code: "ca", Name: "CA", Weight: 30, ExamTypes: models.StringArray{"continuous_assessment"}, Aggregation: "pooled"},
			{Code: "exam", Name: "Exam", Weight: 70, ExamTypes: models.StringArray{"main_exam"}, Aggregation: "pooled"},
		},
	}
	if err := db.Create(&scheme).Error; err != nil {
		t.Fatalf("failed to seed school-wide assessment scheme: %v", err)
	}
}

func TestResolveScheme_NoConfigFallsBackToDefault(t *testing.T) {
	db := setupGradingTestDB(t)
	svc := NewService(db)
	resolved := svc.ResolveScheme(context.Background(), "school-1", "level-1", "subj-1")
	if resolved.Source != DefaultAssessmentMode {
		t.Errorf("Source = %s, want %s for a school with no configured scheme", resolved.Source, DefaultAssessmentMode)
	}
}

func TestResolveScheme_NilServiceFallsBackToDefault(t *testing.T) {
	var svc *Service
	resolved := svc.ResolveScheme(context.Background(), "school-1", "", "")
	if resolved.Source != DefaultAssessmentMode {
		t.Errorf("a nil Service should always fall back to %s, got Source=%s", DefaultAssessmentMode, resolved.Source)
	}
}

func TestResolveScheme_UsesSchoolWideSchemeWhenConfigured(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWideScheme(t, db, "school-1")
	svc := NewService(db)

	resolved := svc.ResolveScheme(context.Background(), "school-1", "level-1", "subj-1")
	if resolved.Source != "school" {
		t.Fatalf("Source = %s, want school", resolved.Source)
	}
	if len(resolved.Components) != 2 {
		t.Fatalf("Components length = %d, want 2", len(resolved.Components))
	}
}

func TestResolveScheme_SubjectSpecificTakesPrecedenceOverSchoolWide(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWideScheme(t, db, "school-1")
	subjectScheme := models.AssessmentScheme{
		SchoolID: "school-1", ClassLevelID: strPtr("level-sss"), SubjectID: strPtr("subj-math"),
		ScopeKey: "level:level-sss/subject:subj-math", MissingPolicy: "zero", IsActive: true,
		Components: []models.AssessmentComponent{
			{Code: "ca", Name: "CA", Weight: 40, ExamTypes: models.StringArray{"continuous_assessment"}, Aggregation: "pooled"},
			{Code: "exam", Name: "Exam", Weight: 60, ExamTypes: models.StringArray{"main_exam"}, Aggregation: "pooled"},
		},
	}
	if err := db.Create(&subjectScheme).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	svc := NewService(db)

	resolved := svc.ResolveScheme(context.Background(), "school-1", "level-sss", "subj-math")
	if resolved.Source != "subject" {
		t.Fatalf("Source = %s, want subject", resolved.Source)
	}
	if resolved.MissingPolicy != "zero" {
		t.Errorf("MissingPolicy = %s, want zero (this subject's own scheme, not the school-wide renormalize one)", resolved.MissingPolicy)
	}

	// A different subject under the same class level, with no scheme of
	// its own, must fall back to the school-wide scheme.
	resolvedOtherSubject := svc.ResolveScheme(context.Background(), "school-1", "level-sss", "subj-english")
	if resolvedOtherSubject.Source != "school" {
		t.Errorf("Source = %s, want school for a subject with no scheme of its own", resolvedOtherSubject.Source)
	}
}

func TestResolveScheme_InactiveSchemeIsIgnored(t *testing.T) {
	db := setupGradingTestDB(t)
	scheme := models.AssessmentScheme{
		SchoolID: "school-1", ScopeKey: "school", MissingPolicy: "renormalize", IsActive: false,
		Components: []models.AssessmentComponent{
			{Code: "exam", Name: "Exam", Weight: 100, ExamTypes: models.StringArray{"main_exam"}, Aggregation: "pooled"},
		},
	}
	if err := db.Create(&scheme).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	svc := NewService(db)
	resolved := svc.ResolveScheme(context.Background(), "school-1", "", "")
	if resolved.Source != DefaultAssessmentMode {
		t.Errorf("an inactive scheme must be ignored, got Source=%s", resolved.Source)
	}
}

func strPtr(s string) *string { return &s }
