package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"cbt-api/internal/cbt/dto"

	"github.com/go-pdf/fpdf"
)

// This file renders the three Academic Broad Sheet PDFs (Class Teacher,
// Subject/HOD, School/VP Academics) as wide landscape tables - a
// different shape from the portrait per-student report-card PDFs
// elsewhere in this package (styles.go), since a broad sheet is one row
// per student/class/subject with many columns, not one page per
// student. The shared design language (each school's own accent color)
// carries over via resolveAccent from styles.go; the page geometry and
// table drawing below are broad-sheet-specific landscape constants, not
// portrait styles.go's marginMM/usableWidthMM.

const (
	bsPageHeightMM  = 210.0 // A4 landscape
	bsMarginMM      = 10.0
	bsUsableWidthMM = 297.0 - 2*bsMarginMM
	bsBottomMM      = bsPageHeightMM - 14 // leave room before triggering our own page break
)

// newBroadSheetPDF creates the shared landscape A4 document every
// broad-sheet export starts from. Auto page-break is deliberately off -
// broadSheetTable.ensureRoom manages page breaks itself so it can
// redraw the header row on the new page; leaving fpdf's own auto-break
// enabled would occasionally fire mid-row-group with no header redrawn.
func newBroadSheetPDF() *fpdf.Fpdf {
	pdfDoc := fpdf.New("L", "mm", "A4", "")
	pdfDoc.SetMargins(bsMarginMM, bsMarginMM, bsMarginMM)
	pdfDoc.SetAutoPageBreak(false, bsMarginMM)
	pdfDoc.AddPage()
	setDraw(pdfDoc, colorBorder)
	pdfDoc.SetLineWidth(0.2)
	return pdfDoc
}

// drawBroadSheetHeader renders the compact masthead shared by all three
// broad-sheet PDFs: school name, the report title in the school's own
// accent color, and a muted subtitle line. No logo and no identification
// grid (unlike the per-student report cards) - a wide table page has no
// room to spare, and a broad sheet is an internal working document for
// staff, not a record handed to a parent.
func drawBroadSheetHeader(pdfDoc *fpdf.Fpdf, school dto.SchoolInfo, title, subtitle string, accent [3]int) {
	pdfDoc.SetFont("helvetica", "B", 14)
	setText(pdfDoc, colorTextDark)
	pdfDoc.CellFormat(bsUsableWidthMM, 6, school.Name, "", 1, "C", false, 0, "")

	pdfDoc.SetFont("helvetica", "B", 11)
	setText(pdfDoc, accent)
	pdfDoc.CellFormat(bsUsableWidthMM, 6, title, "", 1, "C", false, 0, "")

	if subtitle != "" {
		pdfDoc.SetFont("helvetica", "", 8.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(bsUsableWidthMM, 5, subtitle, "", 1, "C", false, 0, "")
	}
	pdfDoc.Ln(2)
}

// drawBroadSheetFooter renders small italic muted footnote lines - kept
// separate from styles.go's drawFooterDisclaimer since that one centers
// against the portrait page's narrower usableWidthMM, which would
// misalign on this landscape page.
func drawBroadSheetFooter(pdfDoc *fpdf.Fpdf, lines ...string) {
	pdfDoc.Ln(3)
	pdfDoc.SetFont("helvetica", "I", 7)
	setText(pdfDoc, colorTextMuted)
	for _, line := range lines {
		pdfDoc.CellFormat(bsUsableWidthMM, 4, line, "", 1, "L", false, 0, "")
	}
}

// broadSheetTable draws a thin data table with the header row (in the
// school's accent color) redrawn at the top of every page, and
// alternating pale/white row shading for readability across many
// columns.
type broadSheetTable struct {
	headers []string
	widths  []float64 // must sum to bsUsableWidthMM
	accent  [3]int
}

func (t broadSheetTable) drawHeaderRow(pdfDoc *fpdf.Fpdf) {
	pdfDoc.SetFont("helvetica", "B", 7.5)
	setFill(pdfDoc, t.accent)
	setText(pdfDoc, colorWhite)
	setDraw(pdfDoc, colorBorder)
	x, y := bsMarginMM, pdfDoc.GetY()
	for i, h := range t.headers {
		pdfDoc.SetXY(x, y)
		pdfDoc.CellFormat(t.widths[i], 6.5, h, "1", 0, "C", true, 0, "")
		x += t.widths[i]
	}
	pdfDoc.SetXY(bsMarginMM, y+6.5)
}

func (t broadSheetTable) ensureRoom(pdfDoc *fpdf.Fpdf, rowH float64) {
	if pdfDoc.GetY()+rowH > bsBottomMM {
		pdfDoc.AddPage()
		t.drawHeaderRow(pdfDoc)
	}
}

func (t broadSheetTable) drawRow(pdfDoc *fpdf.Fpdf, cells []string, shaded bool) {
	const rowH = 6.0
	t.ensureRoom(pdfDoc, rowH)
	x, y := bsMarginMM, pdfDoc.GetY()
	setDraw(pdfDoc, colorBorder)
	if shaded {
		setFill(pdfDoc, colorPaleBg)
	} else {
		setFill(pdfDoc, colorWhite)
	}
	setText(pdfDoc, colorTextDark)
	pdfDoc.SetFont("helvetica", "", 7.5)
	for i, cell := range cells {
		pdfDoc.SetXY(x, y)
		align := "C"
		if i == 0 {
			align = "L"
		}
		pdfDoc.CellFormat(t.widths[i], rowH, cell, "1", 0, align, true, 0, "")
		x += t.widths[i]
	}
	pdfDoc.SetXY(bsMarginMM, y+rowH)
}

func repeatWidth(v float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// formatBroadSheetScoreCell renders one Class Broad Sheet subject cell
// as "score (grade)", with a trailing "*" when the result is still
// incomplete (footnoted below the table) - never a silently blank cell
// that looks identical to "no subject taken".
func formatBroadSheetScoreCell(cell dto.ClassBroadSheetSubjectCell) string {
	if cell.Score == nil {
		return "-"
	}
	s := strconv.Itoa(*cell.Score)
	if cell.Grade != "" {
		s += " (" + cell.Grade + ")"
	}
	if !cell.IsComplete {
		s += " *"
	}
	return s
}

// GenerateClassBroadSheetPDF renders the Class Teacher Broad Sheet
// (template B): one row per student, one column per subject taught in
// the class, plus overall average/pass/fail/incomplete/attendance -
// built from the exact same ClassBroadSheetResponse the JSON endpoint
// and the frontend page render, so the PDF can never show different
// figures than what staff see on screen.
func GenerateClassBroadSheetPDF(school dto.SchoolInfo, resp *dto.ClassBroadSheetResponse) ([]byte, error) {
	accent := resolveAccent(school.PrimaryColor)
	pdfDoc := newBroadSheetPDF()

	subtitle := fmt.Sprintf("%s Report - %d student(s)", titleCase(resp.ReportType), resp.StudentCount)
	if resp.ClassTeacherName != "" {
		subtitle += " - Class Teacher: " + resp.ClassTeacherName
	}
	drawBroadSheetHeader(pdfDoc, school, "Class Teacher Broad Sheet: "+resp.ClassName, subtitle, accent)

	const studentW = 46.0
	const fixedColW = 22.0
	fixedCols := 5 // Overall %, Pass, Fail, Incomplete, Attendance
	subjCount := len(resp.SubjectNames)
	subjW := (bsUsableWidthMM - studentW - fixedColW*float64(fixedCols)) / float64(maxInt(subjCount, 1))
	if subjW < 14 {
		subjW = 14 // floor so a dense subject list stays legible, even if the table then runs slightly past the page width
	}

	headers := append([]string{"Student"}, resp.SubjectNames...)
	headers = append(headers, "Overall %", "Pass", "Fail", "Incomplete", "Attendance")
	widths := append([]float64{studentW}, repeatWidth(subjW, subjCount)...)
	widths = append(widths, fixedColW, fixedColW, fixedColW, fixedColW, fixedColW)

	table := broadSheetTable{headers: headers, widths: widths, accent: accent}
	table.drawHeaderRow(pdfDoc)

	for i, row := range resp.Rows {
		cells := make([]string, 0, len(headers))
		cells = append(cells, row.StudentName)
		for _, subj := range row.Subjects {
			cells = append(cells, formatBroadSheetScoreCell(subj))
		}
		overall := "-"
		if row.OverallAverage != nil {
			overall = fmt.Sprintf("%.1f%%", *row.OverallAverage)
		}
		attendance := "-"
		if row.DaysOpen > 0 {
			attendance = fmt.Sprintf("%d/%d", row.DaysPresent, row.DaysOpen)
		}
		cells = append(cells, overall, strconv.Itoa(row.Pass), strconv.Itoa(row.Fail), strconv.Itoa(row.Incomplete), attendance)
		table.drawRow(pdfDoc, cells, i%2 == 1)
	}

	footerLines := []string{
		"* result not yet complete for that subject.",
		fmt.Sprintf("%d of %d student(s) have at least one incomplete subject result.", resp.StudentsWithIncompleteResults, resp.StudentCount),
	}
	if resp.Review != nil && resp.Review.Summary != "" {
		footerLines = append(footerLines,
			"",
			fmt.Sprintf("Class teacher summary (%s): %s", titleCase(resp.Review.Status), resp.Review.Summary),
		)
	}
	drawBroadSheetFooter(pdfDoc, footerLines...)

	var buf bytes.Buffer
	if err := pdfDoc.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateSubjectBroadSheetPDF renders the Subject Broad Sheet (the
// HOD/Examination Officer view): one row per class a subject is taught
// in, with completeness, performance, and submission-status figures.
func GenerateSubjectBroadSheetPDF(school dto.SchoolInfo, resp *dto.SubjectBroadSheetResponse) ([]byte, error) {
	accent := resolveAccent(school.PrimaryColor)
	pdfDoc := newBroadSheetPDF()

	subtitle := fmt.Sprintf("%d class(es)", len(resp.Rows))
	drawBroadSheetHeader(pdfDoc, school, "Subject Broad Sheet: "+resp.SubjectName, subtitle, accent)

	headers := []string{"Class", "Teacher", "Enrolled", "Marks Entered", "Incomplete", "Class Avg %", "Pass Rate %", "Status"}
	widths := []float64{55, 55, 26, 30, 26, 28, 28, bsUsableWidthMM - (55 + 55 + 26 + 30 + 26 + 28 + 28)}

	table := broadSheetTable{headers: headers, widths: widths, accent: accent}
	table.drawHeaderRow(pdfDoc)

	for i, row := range resp.Rows {
		teacher := row.TeacherName
		if teacher == "" {
			teacher = "-"
		}
		cells := []string{
			row.ClassName, teacher,
			strconv.Itoa(row.Enrolled), strconv.Itoa(row.MarksEntered), strconv.Itoa(row.Incomplete),
			fmt.Sprintf("%.1f%%", row.ClassAverage), fmt.Sprintf("%.1f%%", row.PassRate),
			titleCase(row.SubmissionStatus),
		}
		table.drawRow(pdfDoc, cells, i%2 == 1)
	}

	var buf bytes.Buffer
	if err := pdfDoc.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateSchoolBroadSheetPDF renders the VP Academics / School General
// Broad Sheet: one row per class+subject combination school-wide, with
// a summary line of the same aggregate figures the JSON response and
// frontend summary tiles show.
func GenerateSchoolBroadSheetPDF(school dto.SchoolInfo, resp *dto.SchoolBroadSheetResponse) ([]byte, error) {
	accent := resolveAccent(school.PrimaryColor)
	pdfDoc := newBroadSheetPDF()

	subtitle := fmt.Sprintf(
		"%d student(s) - %d class(es) - %d subject result set(s) - %d complete, %d pending, %d approved - Status: %s - Generated %s",
		resp.TotalStudents, resp.ClassesIncluded, resp.SubjectResultSets,
		resp.SubmissionsComplete, resp.PendingIncomplete, resp.Approved,
		resp.OverallStatus, resp.GeneratedAt.Format("Jan 2, 2006 15:04"),
	)
	drawBroadSheetHeader(pdfDoc, school, "VP Academics / School General Broad Sheet", subtitle, accent)

	headers := []string{"Class", "Subject", "Teacher", "Enrolled", "Entered", "Incomplete", "Class Avg %", "Pass Rate %", "Submitted", "Reviewed", "Approved"}
	fixed := []float64{40, 40, 40, 20, 20, 22, 25, 25}
	var fixedSum float64
	for _, w := range fixed {
		fixedSum += w
	}
	statusW := (bsUsableWidthMM - fixedSum) / 3
	widths := append(append([]float64{}, fixed...), statusW, statusW, statusW)

	table := broadSheetTable{headers: headers, widths: widths, accent: accent}
	table.drawHeaderRow(pdfDoc)

	checkmark := func(b bool) string {
		if b {
			return "Yes"
		}
		return "-"
	}
	for i, row := range resp.Rows {
		teacher := row.TeacherName
		if teacher == "" {
			teacher = "-"
		}
		cells := []string{
			row.ClassName, row.SubjectName, teacher,
			strconv.Itoa(row.Enrolled), strconv.Itoa(row.MarksEntered), strconv.Itoa(row.Incomplete),
			fmt.Sprintf("%.1f%%", row.ClassAveragePct), fmt.Sprintf("%.1f%%", row.PassRatePct),
			checkmark(row.Submitted), checkmark(row.Reviewed), checkmark(row.ApprovedPublished),
		}
		table.drawRow(pdfDoc, cells, i%2 == 1)
	}

	if resp.PromotionReview != nil {
		drawSchoolPromotionReviewSection(pdfDoc, resp.PromotionReview, accent)
	}

	var buf bytes.Buffer
	if err := pdfDoc.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawSchoolPromotionReviewSection renders template C's "Section C" -
// the school-wide promotion-eligibility tally - only ever called when
// resp.PromotionReview is non-nil (a termly/midterm report has no
// eligibility decision to monitor).
func drawSchoolPromotionReviewSection(pdfDoc *fpdf.Fpdf, pr *dto.SchoolPromotionReviewSummary, accent [3]int) {
	pdfDoc.Ln(4)
	pdfDoc.SetFont("helvetica", "B", 10.5)
	setText(pdfDoc, accent)
	pdfDoc.CellFormat(bsUsableWidthMM, 6, "C. PROMOTION REVIEW", "", 1, "L", false, 0, "")

	pdfDoc.SetFont("helvetica", "", 8.5)
	setText(pdfDoc, colorTextDark)
	lines := []string{
		"Eligibility rules applied: " + pr.EligibilityRulesApplied,
		fmt.Sprintf("Eligible - subject to authorisation: %d", pr.EligibleCount),
		fmt.Sprintf("Requires review / remediation: %d", pr.RequiresReviewCount),
		fmt.Sprintf("Incomplete records - decision blocked: %d", pr.IncompleteBlockedCount),
		"Decision status: " + pr.DecisionStatus,
	}
	for _, line := range lines {
		pdfDoc.CellFormat(bsUsableWidthMM, 5, line, "", 1, "L", false, 0, "")
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
