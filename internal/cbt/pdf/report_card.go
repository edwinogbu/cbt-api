// Package pdf renders report-card data (internal/cbt/dto) into
// printable PDF documents - a pure-Go generator (go-pdf/fpdf, no
// external binary or headless-browser dependency) so it works
// unmodified on a resource-constrained offline LAN school-node server,
// not just the cloud deployment. Every report type shares one formal,
// restrained design system (styles.go): predominantly white, black
// body text, thin gray borders, and each school's OWN configured brand
// color as the only accent - these are real academic records a school
// prints, signs and keeps, not a colourful dashboard.
package pdf

import (
	"bytes"
	"fmt"
	"time"

	"cbt-api/internal/cbt/dto"

	"github.com/go-pdf/fpdf"
)

// GenerateTermlyReportCard renders one student's Main Termly Academic
// Report Card (whatever GetReportCardForUser/
// dto.TermlyReportCardResponse produced, real data only - this
// function fabricates nothing) as a PDF document and returns the raw
// bytes.
func GenerateTermlyReportCard(data *dto.TermlyReportCardResponse) ([]byte, error) {
	accent := resolveAccent(data.School.PrimaryColor)
	pdfDoc := newReportPDF()

	drawSchoolHeader(pdfDoc, data.School, "")
	drawTitleBar(pdfDoc, "MAIN TERMLY ACADEMIC REPORT CARD",
		fmt.Sprintf("%s  -  %s", data.Term.Name, data.Session.Name), accent)

	drawKeyValueGrid(pdfDoc, [][2]string{
		{"STUDENT NAME", data.Student.Name}, {"ADMISSION NO.", data.Student.AdmissionNumber},
		{"CLASS / ARM", data.Class.Name}, {"ACADEMIC SESSION", data.Session.Name},
		{"TERM / ASSESSMENT", data.Term.Name + " - Main Termly"}, {"CLASS TEACHER", orDash(data.Class.ClassTeacher)},
	}, 2)

	componentColumns := collectComponentColumns(data.Subjects)
	drawSectionHeading(pdfDoc, "TERM RESULTS", "", accent)
	drawSubjectTable(pdfDoc, data.Subjects, componentColumns, accent)

	drawStatBoxes(pdfDoc, []statBox{
		{Label: "TOTAL OBTAINED", Value: fmt.Sprintf("%d", data.Summary.TotalScore), Caption: fmt.Sprintf("out of %d", data.Summary.TotalMarks)},
		{Label: "OVERALL AVERAGE", Value: fmt.Sprintf("%.1f%%", data.Summary.OverallPercentage), Caption: fmt.Sprintf("Grade %s", data.Summary.OverallGrade), AccentValue: true},
		{Label: "SUBJECTS PASSED", Value: fmt.Sprintf("%d / %d", data.Summary.NumberPassed, data.Summary.TotalSubjects), Caption: passFailCaption(data.Summary)},
		{Label: "CLASS POSITION", Value: classPositionValue(data.Summary), Caption: "If ranking is enabled"},
	}, accent)

	drawSectionHeading(pdfDoc, "ATTENDANCE", "", accent)
	drawAttendanceRow(pdfDoc, data.Attendance)

	drawSectionHeading(pdfDoc, "GENERAL COMMENTS", "", accent)
	drawLabelledRows(pdfDoc, [][2]string{
		{"Class Teacher", orPlaceholder(data.TeacherRemarks)},
		{"Head of School", orPlaceholder(data.PrincipalRemarks)},
	}, 40)

	if data.Conduct != "" {
		drawSectionHeading(pdfDoc, "CONDUCT", "", accent)
		pdfDoc.SetFont("helvetica", "", 9)
		setText(pdfDoc, colorTextDark)
		pdfDoc.CellFormat(usableWidthMM, 5, data.Conduct, "", 1, "L", false, 0, "")
		pdfDoc.Ln(2)
	}

	drawSignatureLines(pdfDoc, []string{"Class Teacher", "Authorized Academic Officer", "Parent / Guardian"})

	footer := "Published reports should include the school's approval status, issue date, and revision history where applicable."
	if data.NextTermBegins != nil {
		footer = fmt.Sprintf("Next term begins on %s. ", data.NextTermBegins.Format("January 2, 2006")) + footer
	}
	drawFooterDisclaimer(pdfDoc, footer, fmt.Sprintf("Generated on %s", time.Now().Format("January 2, 2006")))

	var buf bytes.Buffer
	if err := pdfDoc.Output(&buf); err != nil {
		return nil, fmt.Errorf("render report card pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func passFailCaption(s dto.ReportCardSummary) string {
	failed := s.TotalSubjects - s.NumberPassed
	if failed <= 0 {
		return "No failed subjects"
	}
	return fmt.Sprintf("%d failed", failed)
}

func classPositionValue(s dto.ReportCardSummary) string {
	if s.ClassPosition <= 0 {
		return "Not ranked"
	}
	return fmt.Sprintf("%s of %d", ordinal(s.ClassPosition), s.TotalStudentsInClass)
}

func ordinal(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}

func orPlaceholder(s string) string {
	if s == "" {
		return "No comment yet."
	}
	return s
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// componentColumn mirrors the frontend's union-of-component-codes
// logic (elias-cbt-portal's report-card page.tsx) so the PDF and the
// on-screen report show the same dynamic CA/Test/Exam columns - empty
// when no subject has a configured AssessmentScheme.
type componentColumn struct {
	code, name string
}

func collectComponentColumns(subjects []dto.ReportCardSubject) []componentColumn {
	var cols []componentColumn
	seen := make(map[string]bool)
	for _, s := range subjects {
		for _, c := range s.Components {
			if !seen[c.Code] {
				seen[c.Code] = true
				cols = append(cols, componentColumn{code: c.Code, name: c.Name})
			}
		}
	}
	return cols
}

// drawSubjectTable renders the shared results table: a subject column,
// one column per configured assessment component, a CA TOTAL column
// when more than one component feeds the continuous-assessment side,
// an EXAM/TOTAL/MAX/%/GRADE tail - matching the reference's column set
// as closely as a dynamically-configured scheme allows. Falls back to
// a plain SUBJECT/SCORE/GRADE table for a school with no
// AssessmentScheme configured.
func drawSubjectTable(pdfDoc *fpdf.Fpdf, subjects []dto.ReportCardSubject, componentColumns []componentColumn, accent [3]int) {
	const (
		subjectW  = 36.0
		totalW    = 16.0
		gradeW    = 14.0
		posW      = 12.0
		avgW      = 16.0
		remarksMin = 26.0
	)
	fixed := subjectW + totalW + gradeW + posW + avgW
	componentBudget := usableWidthMM - fixed - remarksMin
	componentW := 0.0
	if len(componentColumns) > 0 {
		componentW = componentBudget / float64(len(componentColumns))
		if componentW > 18 {
			componentW = 18
		}
	}
	remarksW := usableWidthMM - fixed - componentW*float64(len(componentColumns))
	totalLabel := "SCORE"
	if len(componentColumns) > 0 {
		totalLabel = "TOTAL %"
	}

	setFill(pdfDoc, accent)
	setText(pdfDoc, colorWhite)
	pdfDoc.SetFont("helvetica", "B", 7.5)
	pdfDoc.CellFormat(subjectW, 7, "SUBJECT", "1", 0, "L", true, 0, "")
	for _, col := range componentColumns {
		header := col.name
		if header == "" {
			header = col.code
		}
		pdfDoc.CellFormat(componentW, 7, upper(header), "1", 0, "C", true, 0, "")
	}
	pdfDoc.CellFormat(totalW, 7, totalLabel, "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(gradeW, 7, "GRADE", "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(posW, 7, "POS.", "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(avgW, 7, "AVG.", "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(remarksW, 7, "REMARKS", "1", 1, "L", true, 0, "")

	pdfDoc.SetFont("helvetica", "", 8)
	setText(pdfDoc, colorTextDark)
	for i, s := range subjects {
		if i%2 == 1 {
			setFill(pdfDoc, colorPaleBg)
		} else {
			setFill(pdfDoc, colorWhite)
		}
		pdfDoc.CellFormat(subjectW, 6.5, fmt.Sprintf("%s (%s)", s.SubjectName, s.SubjectCode), "1", 0, "L", true, 0, "")

		for _, col := range componentColumns {
			text := "-"
			for _, c := range s.Components {
				if c.Code == col.code {
					if c.HasResults {
						text = fmt.Sprintf("%.0f%%", c.Percentage)
					} else {
						text = "pending"
					}
				}
			}
			pdfDoc.CellFormat(componentW, 6.5, text, "1", 0, "C", true, 0, "")
		}

		if s.IsComplete {
			totalText := fmt.Sprintf("%d", s.TotalScore)
			if len(componentColumns) > 0 {
				totalText = fmt.Sprintf("%.1f%%", s.Percentage)
			}
			if s.IsProvisional {
				totalText += "*"
			}
			pdfDoc.CellFormat(totalW, 6.5, totalText, "1", 0, "C", true, 0, "")
			pdfDoc.CellFormat(gradeW, 6.5, s.Grade, "1", 0, "C", true, 0, "")
			pdfDoc.CellFormat(posW, 6.5, fmt.Sprintf("%d", s.Position), "1", 0, "C", true, 0, "")
		} else {
			pdfDoc.CellFormat(totalW, 6.5, "pending", "1", 0, "C", true, 0, "")
			pdfDoc.CellFormat(gradeW, 6.5, "-", "1", 0, "C", true, 0, "")
			pdfDoc.CellFormat(posW, 6.5, "-", "1", 0, "C", true, 0, "")
		}
		pdfDoc.CellFormat(avgW, 6.5, fmt.Sprintf("%.0f%%", s.ClassAverage), "1", 0, "C", true, 0, "")
		pdfDoc.CellFormat(remarksW, 6.5, s.Remarks, "1", 1, "L", true, 0, "")
	}

	hasProvisional := false
	for _, s := range subjects {
		if s.IsProvisional {
			hasProvisional = true
		}
	}
	if hasProvisional {
		pdfDoc.SetFont("helvetica", "I", 6.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(usableWidthMM, 4, "* Provisional - includes a result still awaiting manual marking.", "", 1, "L", false, 0, "")
	}
	pdfDoc.Ln(4)
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func drawAttendanceRow(pdfDoc *fpdf.Fpdf, attendance *dto.AttendanceSummary) {
	if attendance == nil {
		pdfDoc.SetFont("helvetica", "I", 8.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(usableWidthMM, 5, "Attendance tracking is not yet available for this school.", "", 1, "L", false, 0, "")
		pdfDoc.Ln(2)
		return
	}
	rate := 0.0
	if attendance.DaysSchoolOpen > 0 {
		rate = float64(attendance.DaysPresent) / float64(attendance.DaysSchoolOpen) * 100
	}
	boxes := []statBox{
		{Label: "SCHOOL DAYS", Value: fmt.Sprintf("%d", attendance.DaysSchoolOpen)},
		{Label: "PRESENT", Value: fmt.Sprintf("%d", attendance.DaysPresent)},
		{Label: "ABSENT", Value: fmt.Sprintf("%d", attendance.DaysAbsent)},
		{Label: "ATTENDANCE", Value: fmt.Sprintf("%.1f%%", rate)},
	}
	drawStatBoxes(pdfDoc, boxes, defaultAccent)
}
