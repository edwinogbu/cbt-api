package pdf

import (
	"bytes"
	"strings"
	"testing"

	"cbt-api/internal/cbt/dto"

	pdfreader "github.com/ledongthuc/pdf"
)

func sampleSchoolInfo() dto.SchoolInfo {
	return dto.SchoolInfo{ID: "school-1", Name: "Greenfield Academy", Address: "12 Victoria Island Road, Lagos", PrimaryColor: "#0f4c3a"}
}

func extractPDFText(t *testing.T, out []byte) string {
	t.Helper()
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
	return text.String()
}

func mustContainAll(t *testing.T, got string, want []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("rendered PDF text does not contain %q\nfull extracted text:\n%s", w, got)
		}
	}
}

func sampleClassBroadSheet() *dto.ClassBroadSheetResponse {
	score1, score2 := 82, 58
	overall1, overall2 := 82.0, 58.0
	return &dto.ClassBroadSheetResponse{
		ClassID: "class-1", ClassName: "SSS 2 A", TermID: "term-1", SessionID: "session-1",
		ReportType: "termly", StudentCount: 2,
		SubjectCodes: []string{"MTH", "ENG"}, SubjectNames: []string{"Mathematics", "English Language"},
		Rows: []dto.ClassBroadSheetRow{
			{
				StudentID: "s1", AdmissionNumber: "GA/2024/0137", StudentName: "Amara Okonkwo",
				Subjects: []dto.ClassBroadSheetSubjectCell{
					{SubjectCode: "MTH", Score: &score1, MaxMarks: 100, Grade: "A", IsComplete: true},
					{SubjectCode: "ENG", IsComplete: false},
				},
				OverallAverage: &overall1, Pass: 1, Fail: 0, Incomplete: 1, DaysPresent: 18, DaysOpen: 20,
			},
			{
				StudentID: "s2", AdmissionNumber: "GA/2024/0138", StudentName: "Chidi Eze",
				Subjects: []dto.ClassBroadSheetSubjectCell{
					{SubjectCode: "MTH", Score: &score2, MaxMarks: 100, Grade: "C", IsComplete: true},
					{SubjectCode: "ENG", Score: &score2, MaxMarks: 100, Grade: "C", IsComplete: true},
				},
				OverallAverage: &overall2, Pass: 2, Fail: 0, Incomplete: 0, DaysPresent: 20, DaysOpen: 20,
			},
		},
		StudentsWithIncompleteResults: 1,
	}
}

func TestGenerateClassBroadSheetPDF_ProducesValidPDFWithRealData(t *testing.T) {
	out, err := GenerateClassBroadSheetPDF(sampleSchoolInfo(), sampleClassBroadSheet())
	if err != nil {
		t.Fatalf("GenerateClassBroadSheetPDF failed: %v", err)
	}
	got := extractPDFText(t, out)
	mustContainAll(t, got, []string{
		"Greenfield Academy",
		"Class Teacher Broad Sheet: SSS 2 A",
		"Amara Okonkwo",
		"Chidi Eze",
		"Mathematics",
		"English Language",
		"82 (A)",
		"58 (C)",
		"82.0%",
		"1 of 2 student(s) have at least one incomplete subject result.",
	})
}

func TestGenerateClassBroadSheetPDF_ManySubjectsAndStudentsPaginates(t *testing.T) {
	resp := &dto.ClassBroadSheetResponse{
		ClassID: "class-1", ClassName: "SSS 3 Science", TermID: "term-1", SessionID: "session-1",
		ReportType: "termly",
	}
	for i := 0; i < 10; i++ {
		resp.SubjectNames = append(resp.SubjectNames, "Subject")
		resp.SubjectCodes = append(resp.SubjectCodes, "SUBJ")
	}
	for i := 0; i < 60; i++ {
		score := 70
		row := dto.ClassBroadSheetRow{StudentID: "s", AdmissionNumber: "ADM", StudentName: "Student Row"}
		for range resp.SubjectNames {
			row.Subjects = append(row.Subjects, dto.ClassBroadSheetSubjectCell{Score: &score, Grade: "B", IsComplete: true})
		}
		resp.Rows = append(resp.Rows, row)
	}
	resp.StudentCount = len(resp.Rows)

	out, err := GenerateClassBroadSheetPDF(sampleSchoolInfo(), resp)
	if err != nil {
		t.Fatalf("GenerateClassBroadSheetPDF failed: %v", err)
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	if reader.NumPage() < 2 {
		t.Fatalf("NumPage() = %d, want at least 2 (60 rows should overflow one landscape page)", reader.NumPage())
	}
	// The header row must be redrawn on the second page too - proves
	// ensureRoom's manual page break actually redraws it rather than
	// leaving a headerless continuation table.
	page2Text, err := reader.Page(2).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text from page 2: %v", err)
	}
	if !strings.Contains(page2Text, "Student") {
		t.Errorf("page 2 does not contain the redrawn table header; got:\n%s", page2Text)
	}
}

func sampleSubjectBroadSheet() *dto.SubjectBroadSheetResponse {
	return &dto.SubjectBroadSheetResponse{
		SubjectID: "subj-1", SubjectName: "Mathematics", TermID: "term-1", SessionID: "session-1",
		Rows: []dto.SubjectBroadSheetRow{
			{ClassID: "class-1", ClassName: "SSS 2 A", TeacherName: "Ada Lovelace", Enrolled: 30, MarksEntered: 28, Incomplete: 2, ClassAverage: 68.4, PassRate: 82.1, SubmissionStatus: "submitted"},
			{ClassID: "class-2", ClassName: "SSS 2 B", TeacherName: "Bayo Johnson", Enrolled: 28, MarksEntered: 10, Incomplete: 18, ClassAverage: 55.2, PassRate: 60.0, SubmissionStatus: "draft"},
		},
	}
}

func TestGenerateSubjectBroadSheetPDF_ProducesValidPDFWithRealData(t *testing.T) {
	out, err := GenerateSubjectBroadSheetPDF(sampleSchoolInfo(), sampleSubjectBroadSheet())
	if err != nil {
		t.Fatalf("GenerateSubjectBroadSheetPDF failed: %v", err)
	}
	got := extractPDFText(t, out)
	mustContainAll(t, got, []string{
		"Greenfield Academy",
		"Subject Broad Sheet: Mathematics",
		"SSS 2 A",
		"Ada Lovelace",
		"68.4%",
		"82.1%",
		"Submitted",
		"SSS 2 B",
		"Draft",
	})
}

func sampleSchoolBroadSheet() *dto.SchoolBroadSheetResponse {
	return &dto.SchoolBroadSheetResponse{
		TermID: "term-1", SessionID: "session-1", ReportType: "termly",
		Rows: []dto.SchoolBroadSheetRow{
			{ClassID: "class-1", ClassName: "SSS 2 A", SubjectID: "subj-1", SubjectName: "Mathematics", TeacherName: "Ada Lovelace", Enrolled: 30, MarksEntered: 30, Incomplete: 0, ClassAveragePct: 68.4, PassRatePct: 82.1, Submitted: true, Reviewed: true, ApprovedPublished: true},
			{ClassID: "class-1", ClassName: "SSS 2 A", SubjectID: "subj-2", SubjectName: "English Language", TeacherName: "Chika Obi", Enrolled: 30, MarksEntered: 12, Incomplete: 18, ClassAveragePct: 50.1, PassRatePct: 40.0, Submitted: false, Reviewed: false, ApprovedPublished: false},
		},
		TotalStudents: 30, ClassesIncluded: 1, SubjectResultSets: 2, SubmissionsComplete: 1, PendingIncomplete: 1, AwaitingReview: 0, Approved: 1, Published: 1,
	}
}

func TestGenerateSchoolBroadSheetPDF_ProducesValidPDFWithRealData(t *testing.T) {
	out, err := GenerateSchoolBroadSheetPDF(sampleSchoolInfo(), sampleSchoolBroadSheet())
	if err != nil {
		t.Fatalf("GenerateSchoolBroadSheetPDF failed: %v", err)
	}
	got := extractPDFText(t, out)
	mustContainAll(t, got, []string{
		"Greenfield Academy",
		"VP Academics / School General Broad Sheet",
		"Mathematics",
		"English Language",
		"Ada Lovelace",
		"Chika Obi",
		"68.4%",
		"50.1%",
		"30 student(s)",
		"1 class(es)",
	})
}
