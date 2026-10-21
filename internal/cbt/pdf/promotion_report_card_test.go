package pdf

import (
	"bytes"
	"strings"
	"testing"

	"cbt-api/internal/cbt/dto"

	pdfreader "github.com/ledongthuc/pdf"
)

func samplePromotionReportCard() *dto.PromotionReportCardResponse {
	return &dto.PromotionReportCardResponse{
		School:  dto.SchoolInfo{ID: "school-1", Name: "Greenfield Academy", Address: "12 Victoria Island Road, Lagos", PrimaryColor: "#0f4c3a"},
		Student: dto.StudentInfo{ID: "student-1", Name: "Amara Okonkwo", AdmissionNumber: "GA/2024/0137"},
		Class:   dto.ClassInfo{ID: "class-1", Name: "SSS 2A", ClassTeacher: "Mrs. A. Ibrahim"},
		Session: dto.SessionInfo{ID: "session-1", Name: "2025/2026"},
		Subjects: []dto.ReportCardSubject{
			{
				SubjectName: "Mathematics", SubjectCode: "MTH",
				TotalScore: 91, Percentage: 91, Grade: "A", Position: 2, ClassAverage: 61.3,
				IsComplete: true,
			},
		},
		Summary: dto.ReportCardSummary{
			TotalSubjects: 1, TotalScore: 891, TotalMarks: 1000,
			OverallPercentage: 89.1, OverallGrade: "A", NumberPassed: 1, ClassPosition: 4, TotalStudentsInClass: 32,
		},
		ProposedNextClass:    "SSS 3",
		PolicyChecksSummary:  "Overall threshold: 50% - Core-subject rules: school-configured",
		ResultCompleteness:   "Complete in this sample",
		SystemRecommendation: "ELIGIBLE FOR REVIEW",
		// FinalDecision deliberately left empty - must render the safe
		// default, never silently copy SystemRecommendation.
		FinalDecision: "",
		Strengths:     "Strong performance across most subjects in this illustrative example.",
		FollowUp:      "Review subject-specific results before making the official decision.",
	}
}

func TestGeneratePromotionExaminationReport_ProducesValidPDFWithRealData(t *testing.T) {
	out, err := GeneratePromotionExaminationReport(samplePromotionReportCard())
	if err != nil {
		t.Fatalf("GeneratePromotionExaminationReport failed: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatalf("output does not start with the PDF magic header, got first bytes: %q", out[:min(20, len(out))])
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	pageText, err := reader.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}

	mustContain := []string{
		"Greenfield Academy", "Amara Okonkwo", "GA/2024/0137", "SSS 2A", "Mrs. A. Ibrahim",
		"PROMOTION EXAMINATION REPORT CARD",
		"Mathematics",
		"SSS 3", // proposed next class
		"ELIGIBLE FOR REVIEW",
		"recommendation only; apply the school's approved policy.",
		defaultFinalDecision, // FinalDecision was left empty - must show the safe default
		"Strong performance across most subjects",
		"Promotion is not automatic from report generation.",
	}
	for _, want := range mustContain {
		if !strings.Contains(pageText, want) {
			t.Errorf("rendered PDF text does not contain %q\nfull extracted text:\n%s", want, pageText)
		}
	}
}

func TestGeneratePromotionExaminationReport_NeverCopiesRecommendationIntoFinalDecision(t *testing.T) {
	data := samplePromotionReportCard()
	data.SystemRecommendation = "ELIGIBLE FOR REVIEW"
	data.FinalDecision = "" // never set by any caller in this sample

	out, err := GeneratePromotionExaminationReport(data)
	if err != nil {
		t.Fatalf("GeneratePromotionExaminationReport failed: %v", err)
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	pageText, err := reader.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}
	if !strings.Contains(pageText, defaultFinalDecision) {
		t.Errorf("an empty FinalDecision must render %q, not be left blank or inferred from SystemRecommendation, got:\n%s", defaultFinalDecision, pageText)
	}
}

func TestGeneratePromotionExaminationReport_RecordedFinalDecisionIsShownVerbatim(t *testing.T) {
	data := samplePromotionReportCard()
	data.FinalDecision = "Approved by Academic Board - 14 Dec 2026"

	out, err := GeneratePromotionExaminationReport(data)
	if err != nil {
		t.Fatalf("GeneratePromotionExaminationReport failed: %v", err)
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	pageText, err := reader.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}
	if !strings.Contains(pageText, "Approved by Academic Board") {
		t.Errorf("expected the real recorded FinalDecision to appear verbatim, got:\n%s", pageText)
	}
	if strings.Contains(pageText, defaultFinalDecision) {
		t.Errorf("a recorded FinalDecision must replace the %q placeholder, not appear alongside it, got:\n%s", defaultFinalDecision, pageText)
	}
}
