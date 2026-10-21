package pdf

import (
	"bytes"
	"fmt"
	"time"

	"cbt-api/internal/cbt/dto"

	"github.com/go-pdf/fpdf"
)

// GenerateMidTermProgressReport renders a Mid-Term Progress Report
// Card - a progress snapshot, explicitly distinct from a final termly
// result (no subject here is ever assigned a letter grade; see
// MidTermSubjectResult's own doc comment) - using the same shared,
// restrained design system as the Main Termly and Promotion reports.
func GenerateMidTermProgressReport(data *dto.MidTermReportCardResponse) ([]byte, error) {
	accent := resolveAccent(data.School.PrimaryColor)
	pdfDoc := newReportPDF()

	drawSchoolHeader(pdfDoc, data.School, "")
	drawTitleBar(pdfDoc, "MID-TERM PROGRESS REPORT CARD",
		fmt.Sprintf("%s  -  %s", data.Term.Name, data.Session.Name), accent)

	drawKeyValueGrid(pdfDoc, [][2]string{
		{"STUDENT NAME", data.Student.Name}, {"ADMISSION NO.", data.Student.AdmissionNumber},
		{"CLASS / ARM", data.Class.Name}, {"ACADEMIC SESSION", data.Session.Name},
		{"TERM / ASSESSMENT", data.Term.Name + " - Mid-Term"}, {"CLASS TEACHER", orDash(data.Class.ClassTeacher)},
	}, 2)

	componentColumns := collectMidTermComponentColumns(data.Subjects)
	drawSectionHeading(pdfDoc, "ACADEMIC PERFORMANCE",
		"Progress snapshot only - components configured per the school's assessment policy. No final grade is assigned at mid-term.", accent)
	drawMidTermSubjectTable(pdfDoc, data.Subjects, componentColumns, accent)

	drawStatBoxes(pdfDoc, []statBox{
		{Label: "TOTAL OBTAINED", Value: fmt.Sprintf("%d", data.Summary.TotalScore), Caption: fmt.Sprintf("out of %d", data.Summary.TotalMarks)},
		{Label: "OVERALL AVERAGE", Value: fmt.Sprintf("%.1f%%", data.Summary.OverallPercentage), Caption: "Illustrative progress figure", AccentValue: true},
		{Label: "SUBJECTS PASSED", Value: fmt.Sprintf("%d / %d", data.Summary.NumberPassed, data.Summary.TotalSubjects), Caption: passFailCaption(data.Summary)},
		{Label: "CLASS POSITION", Value: classPositionValue(data.Summary), Caption: "If ranking is enabled"},
	}, accent)

	drawSectionHeading(pdfDoc, "ATTENDANCE & LEARNING HABITS", "", accent)
	drawMidTermAttendance(pdfDoc, data.Attendance)

	drawSectionHeading(pdfDoc, "TEACHER FEEDBACK", "", accent)
	drawLabelledRows(pdfDoc, [][2]string{
		{"Class Teacher", orPlaceholder(data.ClassTeacherComment)},
		{"Next Steps", orPlaceholder(data.NextSteps)},
	}, 40)

	drawSignatureLines(pdfDoc, []string{"Class Teacher", "Parent / Guardian"})

	drawFooterDisclaimer(pdfDoc,
		"Mid-term report: a progress snapshot, not necessarily the final term result.",
		fmt.Sprintf("Generated on %s", time.Now().Format("January 2, 2006")))

	var buf bytes.Buffer
	if err := pdfDoc.Output(&buf); err != nil {
		return nil, fmt.Errorf("render mid-term report card pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func collectMidTermComponentColumns(subjects []dto.MidTermSubjectResult) []componentColumn {
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

// drawMidTermSubjectTable renders SUBJECT | [configured components] |
// TOTAL % | GRADE - GRADE is always "—" (a mid-term snapshot is never
// assigned a final letter grade, matching the reference template).
// Deliberately no Position/Class-Average/Remarks columns here - those
// belong to a FINAL result, not an in-progress one; the summary stat
// boxes below the table carry the student's standing instead.
func drawMidTermSubjectTable(pdfDoc *fpdf.Fpdf, subjects []dto.MidTermSubjectResult, componentColumns []componentColumn, accent [3]int) {
	const (
		subjectW   = 50.0
		totalW     = 22.0
		gradeW     = 18.0
		componentMin = 18.0
	)
	fixed := subjectW + totalW + gradeW
	componentW := componentMin
	if len(componentColumns) > 0 {
		budget := (usableWidthMM - fixed) / float64(len(componentColumns))
		if budget > componentMin {
			componentW = budget
		}
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
	pdfDoc.CellFormat(totalW, 7, "TOTAL %", "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(gradeW, 7, "GRADE", "1", 1, "C", true, 0, "")

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
		totalText := "pending"
		if s.IsComplete {
			totalText = fmt.Sprintf("%.1f%%", s.Percentage)
		}
		pdfDoc.CellFormat(totalW, 6.5, totalText, "1", 0, "C", true, 0, "")
		pdfDoc.CellFormat(gradeW, 6.5, "-", "1", 1, "C", true, 0, "")
	}
	pdfDoc.Ln(4)
}

func drawMidTermAttendance(pdfDoc *fpdf.Fpdf, attendance *dto.MidTermAttendance) {
	if attendance == nil {
		pdfDoc.SetFont("helvetica", "I", 8.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(usableWidthMM, 5, "Attendance tracking is not yet available for this school.", "", 1, "L", false, 0, "")
		pdfDoc.Ln(2)
		return
	}
	rate := 0.0
	if attendance.DaysPossible > 0 {
		rate = float64(attendance.DaysPresent) / float64(attendance.DaysPossible) * 100
	}
	boxes := []statBox{
		{Label: "DAYS POSSIBLE", Value: fmt.Sprintf("%d", attendance.DaysPossible)},
		{Label: "PRESENT", Value: fmt.Sprintf("%d", attendance.DaysPresent)},
		{Label: "ABSENT", Value: fmt.Sprintf("%d", attendance.DaysAbsent)},
		{Label: "ATTENDANCE", Value: fmt.Sprintf("%.1f%%", rate)},
		{Label: "HOMEWORK", Value: orDash(attendance.HomeworkNote)},
	}
	drawStatBoxes(pdfDoc, boxes, defaultAccent)
}
