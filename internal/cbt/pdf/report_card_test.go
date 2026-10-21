package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"cbt-api/internal/cbt/dto"

	pdfreader "github.com/ledongthuc/pdf"
)

func sampleReportCard() *dto.TermlyReportCardResponse {
	nextTerm := time.Date(2026, time.January, 12, 0, 0, 0, 0, time.UTC)
	return &dto.TermlyReportCardResponse{
		School:  dto.SchoolInfo{ID: "school-1", Name: "Greenfield Academy", Address: "12 Victoria Island Road, Lagos"},
		Student: dto.StudentInfo{ID: "student-1", Name: "Amara Okonkwo", AdmissionNumber: "GA/2024/0137"},
		Class:   dto.ClassInfo{ID: "class-1", Name: "SSS 2A"},
		Term:    dto.TermInfo{ID: "term-1", Name: "First Term", Number: 1},
		Session: dto.SessionInfo{ID: "session-1", Name: "2025/2026"},
		Subjects: []dto.ReportCardSubject{
			{
				SubjectName: "Mathematics", SubjectCode: "MTH",
				TotalScore: 72, Percentage: 79.5, Grade: "A", GradePoint: 5,
				Position: 2, ClassAverage: 61.3, Remarks: "Excellent",
				IsComplete: true, IsProvisional: false,
				Components: []dto.SubjectComponentResult{
					{Code: "ca", Name: "CA", Weight: 30, Percentage: 100, HasResults: true},
					{Code: "exam", Name: "Exam", Weight: 70, Percentage: 60, HasResults: true},
				},
			},
			{
				SubjectName: "Biology", SubjectCode: "BIO",
				IsComplete: false, IsProvisional: false, Remarks: "",
				Components: []dto.SubjectComponentResult{
					{Code: "ca", Name: "CA", Weight: 30, Percentage: 90, HasResults: true},
					{Code: "exam", Name: "Exam", Weight: 70, HasResults: false},
				},
			},
		},
		Summary: dto.ReportCardSummary{
			TotalSubjects: 2, OverallPercentage: 68.7, OverallGrade: "B",
			NumberPassed: 1, ClassPosition: 4, TotalStudentsInClass: 32,
		},
		TeacherRemarks:   "Amara has shown consistent improvement this term.",
		PrincipalRemarks: "A commendable result.",
		Conduct:          "Excellent",
		NextTermBegins:   &nextTerm,
		Attendance:       nil,
	}
}

func TestGenerateTermlyReportCard_ProducesValidPDFWithRealData(t *testing.T) {
	out, err := GenerateTermlyReportCard(sampleReportCard())
	if err != nil {
		t.Fatalf("GenerateTermlyReportCard failed: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatalf("output does not start with the PDF magic header, got first bytes: %q", out[:min(20, len(out))])
	}

	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	if reader.NumPage() < 1 {
		t.Fatalf("NumPage() = %d, want at least 1", reader.NumPage())
	}

	var text strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		pageText, err := reader.Page(i).GetPlainText(nil)
		if err != nil {
			t.Fatalf("failed to extract text from page %d: %v", i, err)
		}
		text.WriteString(pageText)
	}
	got := text.String()

	// These must all be the REAL values from sampleReportCard(), not
	// placeholders - proves the actual report data reached the
	// rendered content stream, not just that some PDF was produced.
	mustContain := []string{
		"Greenfield Academy",
		"Amara Okonkwo",
		"GA/2024/0137",
		"SSS 2A",
		"Mathematics",
		"Biology",
		"Excellent",   // Mathematics' remark and the conduct value
		"pending",     // Biology's missing exam component / incomplete subject
		"Attendance tracking is not yet available for this school.",
		"Amara has shown consistent improvement this term.",
		"A commendable result.",
		"January 12, 2026", // next term date
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("rendered PDF text does not contain %q\nfull extracted text:\n%s", want, got)
		}
	}
}

func TestGenerateTermlyReportCard_NoComponentsFallsBackToScoreColumn(t *testing.T) {
	data := sampleReportCard()
	// Strip every subject's Components, simulating a school with no
	// AssessmentScheme configured - the common case today.
	for i := range data.Subjects {
		data.Subjects[i].Components = nil
	}

	out, err := GenerateTermlyReportCard(data)
	if err != nil {
		t.Fatalf("GenerateTermlyReportCard failed: %v", err)
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	pageText, err := reader.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}
	if !strings.Contains(pageText, "Score") {
		t.Errorf("expected the unconfigured-scheme fallback header \"Score\" in the PDF, got:\n%s", pageText)
	}
	if strings.Contains(pageText, "Total %") {
		t.Errorf("did not expect a \"Total %%\" column header when no subject has configured components, got:\n%s", pageText)
	}
}

func TestGenerateTermlyReportCard_EmptyRemarksShowPlaceholder(t *testing.T) {
	data := sampleReportCard()
	data.TeacherRemarks = ""
	data.PrincipalRemarks = ""

	out, err := GenerateTermlyReportCard(data)
	if err != nil {
		t.Fatalf("GenerateTermlyReportCard failed: %v", err)
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	pageText, err := reader.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}
	if strings.Count(pageText, "No comment yet.") != 2 {
		t.Errorf("expected both empty remarks to show the \"No comment yet.\" placeholder, got:\n%s", pageText)
	}
}
