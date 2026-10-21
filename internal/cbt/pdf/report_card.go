// Package pdf renders report-card data (internal/cbt/dto) into a
// printable PDF document - a pure-Go generator (go-pdf/fpdf, no
// external binary or headless-browser dependency) so it works
// unmodified on a resource-constrained offline LAN school-node server,
// not just the cloud deployment.
package pdf

import (
	"bytes"
	"fmt"
	"time"

	"cbt-api/internal/cbt/dto"

	"github.com/go-pdf/fpdf"
)

const (
	pageWidthMM    = 210.0 // A4
	marginMM       = 15.0
	usableWidthMM  = pageWidthMM - 2*marginMM
	navyR, navyG, navyB = 23, 37, 84 // Tailwind's blue-950-ish, matches the web report card's header banner
	amberR, amberG, amberB = 251, 191, 36
)

// GenerateTermlyReportCard renders one student's termly report card
// (whatever GetReportCardForUser/dto.TermlyReportCardResponse produced,
// real data only - this function fabricates nothing) as a PDF document
// and returns the raw bytes.
func GenerateTermlyReportCard(data *dto.TermlyReportCardResponse) ([]byte, error) {
	pdfDoc := fpdf.New("P", "mm", "A4", "")
	pdfDoc.SetMargins(marginMM, marginMM, marginMM)
	pdfDoc.SetAutoPageBreak(true, marginMM)
	pdfDoc.AddPage()

	drawHeader(pdfDoc, data)
	drawStudentInfo(pdfDoc, data)
	componentColumns := collectComponentColumns(data.Subjects)
	drawSubjectTable(pdfDoc, data.Subjects, componentColumns)
	drawAttendance(pdfDoc, data.Attendance)
	drawSummary(pdfDoc, data.Summary)
	drawRemarks(pdfDoc, data)
	drawSignatures(pdfDoc)
	drawFooter(pdfDoc, data)

	var buf bytes.Buffer
	if err := pdfDoc.Output(&buf); err != nil {
		return nil, fmt.Errorf("render report card pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func drawHeader(pdfDoc *fpdf.Fpdf, data *dto.TermlyReportCardResponse) {
	pdfDoc.SetFillColor(navyR, navyG, navyB)
	pdfDoc.Rect(0, 0, pageWidthMM, 32, "F")
	pdfDoc.SetTextColor(255, 255, 255)

	pdfDoc.SetFont("helvetica", "B", 18)
	pdfDoc.SetXY(0, 7)
	pdfDoc.CellFormat(pageWidthMM, 9, data.School.Name, "", 1, "C", false, 0, "")

	pdfDoc.SetFont("helvetica", "", 10)
	pdfDoc.SetXY(0, 16)
	pdfDoc.CellFormat(pageWidthMM, 6, data.School.Address, "", 1, "C", false, 0, "")

	pdfDoc.SetFont("helvetica", "B", 10)
	pdfDoc.SetTextColor(amberR, amberG, amberB)
	pdfDoc.SetXY(0, 23)
	pdfDoc.CellFormat(pageWidthMM, 6, "TERMLY REPORT CARD", "", 1, "C", false, 0, "")

	pdfDoc.SetFillColor(amberR, amberG, amberB)
	pdfDoc.Rect(0, 32, pageWidthMM, 1.5, "F")

	pdfDoc.SetTextColor(0, 0, 0)
	pdfDoc.SetXY(marginMM, 38)
}

func drawStudentInfo(pdfDoc *fpdf.Fpdf, data *dto.TermlyReportCardResponse) {
	classPosition := "Not ranked"
	if data.Summary.ClassPosition > 0 {
		classPosition = fmt.Sprintf("%d of %d", data.Summary.ClassPosition, data.Summary.TotalStudentsInClass)
	}
	conduct := data.Conduct
	if conduct == "" {
		conduct = "-"
	}

	fields := [][2]string{
		{"Student Name", data.Student.Name},
		{"Admission Number", data.Student.AdmissionNumber},
		{"Class", data.Class.Name},
		{"Session / Term", fmt.Sprintf("%s - %s", data.Session.Name, data.Term.Name)},
		{"Class Position", classPosition},
		{"Conduct", conduct},
	}

	// Each column in a row must share one baseline Y: CellFormat's ln=2
	// moves the cursor to directly below the cell just drawn, which
	// drifts further down with every field drawn - reading GetY() again
	// for the next column (rather than reusing the row's own starting
	// Y) stacked that drift across every field in the same visual row.
	const rowH = 12.0
	colW := usableWidthMM / 3
	rowY := pdfDoc.GetY()
	for i, field := range fields {
		col := i % 3
		if col == 0 && i > 0 {
			rowY += rowH
		}
		x := marginMM + float64(col)*colW
		pdfDoc.SetXY(x, rowY)
		pdfDoc.SetFont("helvetica", "", 8)
		pdfDoc.SetTextColor(100, 100, 100)
		pdfDoc.CellFormat(colW-2, 4, field[0], "", 0, "L", false, 0, "")
		pdfDoc.SetXY(x, rowY+4)
		pdfDoc.SetFont("helvetica", "B", 10)
		pdfDoc.SetTextColor(0, 0, 0)
		pdfDoc.CellFormat(colW-2, 5, field[1], "", 0, "L", false, 0, "")
	}
	pdfDoc.SetXY(marginMM, rowY+rowH+2)
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

func drawSubjectTable(pdfDoc *fpdf.Fpdf, subjects []dto.ReportCardSubject, componentColumns []componentColumn) {
	const (
		subjectW   = 38.0
		totalW     = 18.0
		gradeW     = 14.0
		positionW  = 14.0
		classAvgW  = 18.0
		remarksMin = 30.0
	)
	fixed := subjectW + totalW + gradeW + positionW + classAvgW
	componentBudget := usableWidthMM - fixed - remarksMin
	componentW := 0.0
	if len(componentColumns) > 0 {
		componentW = componentBudget / float64(len(componentColumns))
		if componentW > 20 {
			componentW = 20
		}
	}
	remarksW := usableWidthMM - fixed - componentW*float64(len(componentColumns))

	totalLabel := "Score"
	if len(componentColumns) > 0 {
		totalLabel = "Total %"
	}

	pdfDoc.SetFont("helvetica", "B", 8)
	pdfDoc.SetFillColor(230, 230, 235)
	pdfDoc.SetTextColor(60, 60, 60)
	pdfDoc.CellFormat(subjectW, 7, "Subject", "1", 0, "L", true, 0, "")
	for _, col := range componentColumns {
		pdfDoc.CellFormat(componentW, 7, col.name, "1", 0, "C", true, 0, "")
	}
	pdfDoc.CellFormat(totalW, 7, totalLabel, "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(gradeW, 7, "Grade", "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(positionW, 7, "Pos.", "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(classAvgW, 7, "Avg.", "1", 0, "C", true, 0, "")
	pdfDoc.CellFormat(remarksW, 7, "Remarks", "1", 1, "L", true, 0, "")

	pdfDoc.SetFont("helvetica", "", 8)
	pdfDoc.SetTextColor(0, 0, 0)
	for _, s := range subjects {
		pdfDoc.CellFormat(subjectW, 7, fmt.Sprintf("%s (%s)", s.SubjectName, s.SubjectCode), "1", 0, "L", false, 0, "")

		for _, col := range componentColumns {
			text := "-"
			for _, c := range s.Components {
				if c.Code == col.code {
					if c.HasResults {
						text = fmt.Sprintf("%.1f%%", c.Percentage)
					} else {
						text = "pending"
					}
				}
			}
			pdfDoc.CellFormat(componentW, 7, text, "1", 0, "C", false, 0, "")
		}

		if s.IsComplete {
			totalText := fmt.Sprintf("%.0f", float64(s.TotalScore))
			if len(componentColumns) > 0 {
				totalText = fmt.Sprintf("%.1f%%", s.Percentage)
			}
			if s.IsProvisional {
				totalText += "*"
			}
			pdfDoc.CellFormat(totalW, 7, totalText, "1", 0, "C", false, 0, "")
			pdfDoc.CellFormat(gradeW, 7, s.Grade, "1", 0, "C", false, 0, "")
			pdfDoc.CellFormat(positionW, 7, fmt.Sprintf("%d", s.Position), "1", 0, "C", false, 0, "")
		} else {
			pdfDoc.CellFormat(totalW, 7, "pending", "1", 0, "C", false, 0, "")
			pdfDoc.CellFormat(gradeW, 7, "-", "1", 0, "C", false, 0, "")
			pdfDoc.CellFormat(positionW, 7, "-", "1", 0, "C", false, 0, "")
		}
		pdfDoc.CellFormat(classAvgW, 7, fmt.Sprintf("%.1f%%", s.ClassAverage), "1", 0, "C", false, 0, "")
		pdfDoc.CellFormat(remarksW, 7, s.Remarks, "1", 1, "L", false, 0, "")
	}

	hasProvisional := false
	for _, s := range subjects {
		if s.IsProvisional {
			hasProvisional = true
		}
	}
	if hasProvisional {
		pdfDoc.SetFont("helvetica", "I", 7)
		pdfDoc.SetTextColor(120, 90, 0)
		pdfDoc.CellFormat(usableWidthMM, 5, "* Provisional - includes a result still awaiting manual marking.", "", 1, "L", false, 0, "")
		pdfDoc.SetTextColor(0, 0, 0)
	}
	pdfDoc.Ln(3)
}

func drawAttendance(pdfDoc *fpdf.Fpdf, attendance *dto.AttendanceSummary) {
	pdfDoc.SetFont("helvetica", "B", 10)
	pdfDoc.CellFormat(usableWidthMM, 6, "Attendance", "", 1, "L", false, 0, "")
	pdfDoc.SetFont("helvetica", "", 9)
	if attendance == nil {
		pdfDoc.SetTextColor(120, 120, 120)
		pdfDoc.CellFormat(usableWidthMM, 5, "Attendance tracking is not yet available for this school.", "", 1, "L", false, 0, "")
		pdfDoc.SetTextColor(0, 0, 0)
	} else {
		text := fmt.Sprintf("Present: %d    Absent: %d    School Days Open: %d",
			attendance.DaysPresent, attendance.DaysAbsent, attendance.DaysSchoolOpen)
		pdfDoc.CellFormat(usableWidthMM, 5, text, "", 1, "L", false, 0, "")
	}
	pdfDoc.Ln(3)
}

func drawSummary(pdfDoc *fpdf.Fpdf, summary dto.ReportCardSummary) {
	pdfDoc.SetFont("helvetica", "B", 10)
	pdfDoc.CellFormat(usableWidthMM, 6, "Summary", "", 1, "L", false, 0, "")

	colW := usableWidthMM / 4
	cells := []string{
		fmt.Sprintf("Overall: %.1f%%", summary.OverallPercentage),
		fmt.Sprintf("Grade: %s", summary.OverallGrade),
		fmt.Sprintf("Passed: %d/%d", summary.NumberPassed, summary.TotalSubjects),
		fmt.Sprintf("Position: %d of %d", summary.ClassPosition, summary.TotalStudentsInClass),
	}
	pdfDoc.SetFont("helvetica", "", 9)
	for _, c := range cells {
		pdfDoc.CellFormat(colW, 6, c, "1", 0, "C", false, 0, "")
	}
	pdfDoc.Ln(9)
}

func drawRemarks(pdfDoc *fpdf.Fpdf, data *dto.TermlyReportCardResponse) {
	colW := usableWidthMM / 2
	startY := pdfDoc.GetY()

	pdfDoc.SetFont("helvetica", "B", 8)
	pdfDoc.SetTextColor(100, 100, 100)
	pdfDoc.SetXY(marginMM, startY)
	pdfDoc.CellFormat(colW-2, 5, "Class Teacher's Comment", "", 1, "L", false, 0, "")
	pdfDoc.SetFont("helvetica", "", 9)
	pdfDoc.SetTextColor(0, 0, 0)
	pdfDoc.SetXY(marginMM, startY+5)
	teacherRemark := data.TeacherRemarks
	if teacherRemark == "" {
		teacherRemark = "No comment yet."
	}
	pdfDoc.MultiCell(colW-2, 5, teacherRemark, "", "L", false)

	pdfDoc.SetFont("helvetica", "B", 8)
	pdfDoc.SetTextColor(100, 100, 100)
	pdfDoc.SetXY(marginMM+colW, startY)
	pdfDoc.CellFormat(colW-2, 5, "Principal's Comment", "", 1, "L", false, 0, "")
	pdfDoc.SetFont("helvetica", "", 9)
	pdfDoc.SetTextColor(0, 0, 0)
	pdfDoc.SetXY(marginMM+colW, startY+5)
	principalRemark := data.PrincipalRemarks
	if principalRemark == "" {
		principalRemark = "No comment yet."
	}
	pdfDoc.MultiCell(colW-2, 5, principalRemark, "", "L", false)

	pdfDoc.SetXY(marginMM, startY+22)
}

func drawSignatures(pdfDoc *fpdf.Fpdf) {
	y := pdfDoc.GetY() + 10
	colW := usableWidthMM / 2

	pdfDoc.SetDrawColor(150, 150, 150)
	pdfDoc.Line(marginMM+10, y, marginMM+colW-10, y)
	pdfDoc.Line(marginMM+colW+10, y, pageWidthMM-marginMM-10, y)

	pdfDoc.SetFont("helvetica", "", 8)
	pdfDoc.SetTextColor(100, 100, 100)
	pdfDoc.SetXY(marginMM, y+1)
	pdfDoc.CellFormat(colW, 5, "Class Teacher's Signature", "", 0, "C", false, 0, "")
	pdfDoc.SetXY(marginMM+colW, y+1)
	pdfDoc.CellFormat(colW, 5, "Principal's Signature", "", 1, "C", false, 0, "")

	pdfDoc.SetY(y + 10)
}

func drawFooter(pdfDoc *fpdf.Fpdf, data *dto.TermlyReportCardResponse) {
	pdfDoc.SetFont("helvetica", "I", 7)
	pdfDoc.SetTextColor(120, 120, 120)
	if data.NextTermBegins != nil {
		pdfDoc.CellFormat(usableWidthMM, 5,
			fmt.Sprintf("Next term begins on %s", data.NextTermBegins.Format("January 2, 2006")),
			"", 1, "C", false, 0, "")
	}
	pdfDoc.CellFormat(usableWidthMM, 5,
		fmt.Sprintf("Generated on %s", time.Now().Format("January 2, 2006")),
		"", 1, "C", false, 0, "")
	pdfDoc.CellFormat(usableWidthMM, 5,
		"This report card is computer-generated and valid only with an authorized signature.",
		"", 1, "C", false, 0, "")
}
