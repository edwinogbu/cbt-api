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
		School: dto.SchoolInfo{
			ID: "school-1", Name: "Greenfield Academy", Address: "12 Victoria Island Road, Lagos",
			// A real, distinctive configured brand color - not the
			// neutral default - so a test can prove each school's own
			// color is actually used, not a hardcoded platform color.
			PrimaryColor: "#0f4c3a",
		},
		Student: dto.StudentInfo{ID: "student-1", Name: "Amara Okonkwo", AdmissionNumber: "GA/2024/0137"},
		Class:   dto.ClassInfo{ID: "class-1", Name: "SSS 2A", ClassTeacher: "Mrs. A. Ibrahim"},
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
			TotalSubjects: 2, TotalScore: 891, TotalMarks: 1000,
			OverallPercentage: 68.7, OverallGrade: "B",
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
		"12 Victoria Island Road, Lagos",
		"Amara Okonkwo",
		"GA/2024/0137",
		"SSS 2A",
		"Mrs. A. Ibrahim", // real class teacher name
		"Mathematics",
		"Biology",
		"Excellent",   // Mathematics' remark and the conduct value
		"pending",     // Biology's missing exam component / incomplete subject
		"891",         // Summary.TotalScore
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
	if !strings.Contains(pageText, "SCORE") {
		t.Errorf("expected the unconfigured-scheme fallback header \"SCORE\" in the PDF, got:\n%s", pageText)
	}
	if strings.Contains(pageText, "TOTAL %") {
		t.Errorf("did not expect a \"TOTAL %%\" column header when no subject has configured components, got:\n%s", pageText)
	}
}

func TestResolveAccent_UsesEachSchoolsOwnConfiguredColorIndependently(t *testing.T) {
	schoolA := resolveAccent("#0f4c3a")
	schoolB := resolveAccent("#8a1f2b")
	if schoolA == schoolB {
		t.Fatalf("two schools with different configured colors resolved to the same accent %v - School B's color must never leak onto School A's report", schoolA)
	}
	wantA := [3]int{0x0f, 0x4c, 0x3a}
	if schoolA != wantA {
		t.Errorf("resolveAccent(#0f4c3a) = %v, want %v", schoolA, wantA)
	}
}

func TestResolveAccent_NoConfiguredColorFallsBackToNeutralDefault(t *testing.T) {
	got := resolveAccent("")
	if got != defaultAccent {
		t.Errorf("resolveAccent(\"\") = %v, want the neutral default %v - never fabricate a brand color for a school with none configured", got, defaultAccent)
	}
	gotMalformed := resolveAccent("not-a-color")
	if gotMalformed != defaultAccent {
		t.Errorf("resolveAccent(malformed) = %v, want the neutral default %v", gotMalformed, defaultAccent)
	}
}

func TestGenerateTermlyReportCard_UnreachableLogoFallsBackToPlaceholder(t *testing.T) {
	data := sampleReportCard()
	// A school-configured logo URL that cannot be reached (exactly the
	// situation an offline LAN school-node server is in for a
	// cloud-hosted logo) must never fail or hang generation - it must
	// fall back to the plain placeholder box.
	data.School.Logo = "http://127.0.0.1:1/unreachable-logo.png"

	out, err := GenerateTermlyReportCard(data)
	if err != nil {
		t.Fatalf("GenerateTermlyReportCard failed with an unreachable logo URL (must degrade gracefully): %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatal("output is not a valid PDF after an unreachable logo URL")
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	pageText, err := reader.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}
	if !strings.Contains(pageText, "LOGO") || !strings.Contains(pageText, "PLACEHOLDER") {
		t.Errorf("expected the LOGO PLACEHOLDER fallback text when the configured logo URL is unreachable, got:\n%s", pageText)
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
