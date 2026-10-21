package pdf

import (
	"bytes"
	"strings"
	"testing"

	"cbt-api/internal/cbt/dto"

	pdfreader "github.com/ledongthuc/pdf"
)

func sampleMidTermReportCard() *dto.MidTermReportCardResponse {
	return &dto.MidTermReportCardResponse{
		School:  dto.SchoolInfo{ID: "school-1", Name: "Greenfield Academy", Address: "12 Victoria Island Road, Lagos", PrimaryColor: "#0f4c3a"},
		Student: dto.StudentInfo{ID: "student-1", Name: "Amara Okonkwo", AdmissionNumber: "GA/2024/0137"},
		Class:   dto.ClassInfo{ID: "class-1", Name: "SSS 2A", ClassTeacher: "Mrs. A. Ibrahim"},
		Term:    dto.TermInfo{ID: "term-1", Name: "First Term", Number: 1},
		Session: dto.SessionInfo{ID: "session-1", Name: "2025/2026"},
		Subjects: []dto.MidTermSubjectResult{
			{
				SubjectName: "Mathematics", SubjectCode: "MTH",
				Percentage: 70, IsComplete: true,
				Components: []dto.SubjectComponentResult{
					{Code: "ca", Name: "CA", Percentage: 85, HasResults: true},
					{Code: "test", Name: "Test", Percentage: 60, HasResults: true},
				},
			},
			{
				SubjectName: "Biology", SubjectCode: "BIO",
				IsComplete: false,
				Components: []dto.SubjectComponentResult{
					{Code: "ca", Name: "CA", Percentage: 90, HasResults: true},
					{Code: "test", Name: "Test", HasResults: false},
				},
			},
		},
		Summary: dto.ReportCardSummary{
			TotalSubjects: 2, TotalScore: 891, TotalMarks: 1000,
			OverallPercentage: 89.1, NumberPassed: 2, ClassPosition: 4, TotalStudentsInClass: 32,
		},
		Attendance: &dto.MidTermAttendance{
			DaysPossible: 42, DaysPresent: 40, DaysAbsent: 2, HomeworkNote: "Usually completed",
		},
		ClassTeacherComment: "Amara participates actively and shows a strong understanding of most topics.",
		NextSteps:           "Maintain a regular revision timetable.",
	}
}

func TestGenerateMidTermProgressReport_ProducesValidPDFWithRealData(t *testing.T) {
	out, err := GenerateMidTermProgressReport(sampleMidTermReportCard())
	if err != nil {
		t.Fatalf("GenerateMidTermProgressReport failed: %v", err)
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
		"MID-TERM PROGRESS REPORT CARD",
		"Mathematics", "Biology",
		"pending", // Biology's missing test component
		"891", "89.1%",
		"40", "42", "Usually completed", // attendance figures
		"Amara participates actively", "Maintain a regular revision timetable.",
		"Mid-term report: a progress snapshot, not necessarily the final term result.",
	}
	for _, want := range mustContain {
		if !strings.Contains(pageText, want) {
			t.Errorf("rendered PDF text does not contain %q\nfull extracted text:\n%s", want, pageText)
		}
	}
}

// A mid-term snapshot must never present a final letter grade (the
// reference template's GRADE column is always "—"/"-" for every row,
// since this is a progress check, not a final result). This is
// guaranteed structurally, not just by the renderer's own choice to
// write "-": dto.MidTermSubjectResult has no Grade field at all for a
// future caller to accidentally populate and have rendered - unlike
// dto.ReportCardSubject, which does carry one for the Main Termly and
// Promotion reports where a final grade is appropriate.

func TestGenerateMidTermProgressReport_NoAttendanceShowsPlaceholder(t *testing.T) {
	data := sampleMidTermReportCard()
	data.Attendance = nil

	out, err := GenerateMidTermProgressReport(data)
	if err != nil {
		t.Fatalf("GenerateMidTermProgressReport failed: %v", err)
	}
	reader, err := pdfreader.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatalf("generated PDF could not be parsed back: %v", err)
	}
	pageText, err := reader.Page(1).GetPlainText(nil)
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}
	if !strings.Contains(pageText, "Attendance tracking is not yet available for this school.") {
		t.Errorf("expected the attendance placeholder when Attendance is nil, got:\n%s", pageText)
	}
}
