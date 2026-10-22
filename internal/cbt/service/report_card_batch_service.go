package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/pdf"
	"cbt-api/internal/models"

	"github.com/xuri/excelize/v2"
)

// classReportCardEntry pairs one roster student with the one
// report-card payload type their scope's ReportType produced - exactly
// one of termly/midterm/promotion is ever set, matching the three-way
// switch in buildReportCardSnapshot. Err is set instead when this
// student's report could not be generated (e.g. no results recorded at
// all yet); such a student still gets an entry, never a silently
// dropped roster row.
type classReportCardEntry struct {
	student   *models.Student
	termly    *dto.TermlyReportCardResponse
	midterm   *dto.MidTermReportCardResponse
	promotion *dto.PromotionReportCardResponse
	err       error
}

func (e classReportCardEntry) studentName() string {
	return strings.TrimSpace(e.student.User.FirstName + " " + e.student.User.LastName)
}

// overallFigures returns the one shared (percentage, grade, position)
// triple every report type's dto.ReportCardSummary exposes - a
// Mid-Term entry naturally yields an empty grade and a zero position,
// since its own builder (GetMidTermProgressReport) never sets either
// field, matching "no letter grade is ever assigned at mid-term".
func (e classReportCardEntry) overallFigures() (percentage float64, grade string, position int) {
	switch {
	case e.termly != nil:
		return e.termly.Summary.OverallPercentage, e.termly.Summary.OverallGrade, e.termly.Summary.ClassPosition
	case e.midterm != nil:
		return e.midterm.Summary.OverallPercentage, "", 0
	case e.promotion != nil:
		return e.promotion.Summary.OverallPercentage, e.promotion.Summary.OverallGrade, e.promotion.Summary.ClassPosition
	default:
		return 0, "", 0
	}
}

// collectClassReportCards resolves a class's current active roster and
// builds every student's live report card for one scope - the shared
// data-gathering step behind the class summary, ZIP, and spreadsheet
// export paths, so all three see exactly the same roster and figures.
func (s *ExamService) collectClassReportCards(ctx context.Context, schoolID string, isPlatformOwner bool, classID, reportType, termID, sessionID string) ([]classReportCardEntry, string, error) {
	if err := s.authorizeClassTenant(ctx, classID, schoolID, isPlatformOwner); err != nil {
		return nil, "", err
	}
	if !validReportCardTypes[reportType] {
		return nil, "", errors.New("report_type must be one of termly, midterm, promotion")
	}
	if reportType == "promotion" {
		termID = ""
	} else if termID == "" {
		return nil, "", errors.New("term_id is required for this report type")
	}

	className, _, _ := s.resolveClassWithTeacher(ctx, classID)

	var students []models.Student
	if err := s.db.WithContext(ctx).
		Preload("User").
		Where("class_id = ? AND is_active = ? AND deleted_at IS NULL", classID, true).
		Find(&students).Error; err != nil {
		return nil, className, err
	}
	sort.Slice(students, func(i, j int) bool {
		return students[i].AdmissionNo < students[j].AdmissionNo
	})

	entries := make([]classReportCardEntry, 0, len(students))
	for i := range students {
		student := &students[i]
		entry := classReportCardEntry{student: student}
		switch reportType {
		case "termly":
			entry.termly, entry.err = s.GetTermlyReportCardForStudent(ctx, student, termID, sessionID)
		case "midterm":
			entry.midterm, entry.err = s.GetMidTermProgressReport(ctx, student, termID, sessionID)
		case "promotion":
			entry.promotion, entry.err = s.GetPromotionReportCard(ctx, student, sessionID)
		}
		entries = append(entries, entry)
	}
	return entries, className, nil
}

// classApprovalStatuses looks up every listed student's current
// approval state for one scope in a single query, keyed by student ID -
// used to annotate a class summary without an N+1 query per row.
func (s *ExamService) classApprovalStatuses(ctx context.Context, studentIDs []string, reportType, termID, sessionID string) map[string]models.ReportCardApproval {
	result := make(map[string]models.ReportCardApproval, len(studentIDs))
	if len(studentIDs) == 0 {
		return result
	}
	var approvals []models.ReportCardApproval
	if err := s.db.WithContext(ctx).
		Where("student_id IN ? AND report_type = ? AND term_id = ? AND session_id = ?", studentIDs, reportType, termID, sessionID).
		Find(&approvals).Error; err != nil {
		return result
	}
	for _, a := range approvals {
		result[a.StudentID] = a
	}
	return result
}

// GetClassReportCardSummary builds the class-wide roster review screen
// an admin/teacher uses before approving or exporting a whole class's
// report cards - each row's overall figures plus its current approval
// state, not a full per-subject breakdown (that stays in the
// individual report card / PDF).
func (s *ExamService) GetClassReportCardSummary(ctx context.Context, schoolID string, isPlatformOwner bool, classID, reportType, termID, sessionID string) (*dto.ClassReportCardSummaryResponse, error) {
	entries, className, err := s.collectClassReportCards(ctx, schoolID, isPlatformOwner, classID, reportType, termID, sessionID)
	if err != nil {
		return nil, err
	}

	studentIDs := make([]string, len(entries))
	for i, e := range entries {
		studentIDs[i] = e.student.ID
	}
	normalizedTermID := termID
	if reportType == "promotion" {
		normalizedTermID = ""
	}
	approvals := s.classApprovalStatuses(ctx, studentIDs, reportType, normalizedTermID, sessionID)

	rows := make([]dto.ClassReportCardSummaryRow, 0, len(entries))
	for _, e := range entries {
		row := dto.ClassReportCardSummaryRow{
			StudentID: e.student.ID, StudentName: e.studentName(),
			AdmissionNumber: e.student.AdmissionNo, ApprovalStatus: "not_submitted",
		}
		if a, ok := approvals[e.student.ID]; ok {
			row.ApprovalStatus = a.Status
			row.ApprovalVersion = a.Version
		}
		if e.err != nil {
			row.Error = e.err.Error()
		} else {
			row.OverallPercentage, row.OverallGrade, row.ClassPosition = e.overallFigures()
		}
		rows = append(rows, row)
	}

	return &dto.ClassReportCardSummaryResponse{
		ClassID: classID, ClassName: className, ReportType: reportType,
		TermID: normalizedTermID, SessionID: sessionID, Rows: rows,
	}, nil
}

// reportCardFilenameFor builds one student's PDF filename inside a
// batch ZIP - admission number first so files sort the same order as
// the roster, independent of whatever punctuation is in a student's
// name.
func reportCardFilenameFor(admissionNo, name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "_", "\"", "")
	return replacer.Replace(admissionNo) + "_" + replacer.Replace(name) + ".pdf"
}

// GenerateClassReportCardsZIP renders every student in a class's
// individual report-card PDF (the exact same per-student generator
// the student-facing download endpoint uses) and packs them into one
// ZIP archive. A student whose report could not be generated is
// skipped from the archive but still named in the returned skipped
// list, so the caller can tell a partial batch from a complete one.
func (s *ExamService) GenerateClassReportCardsZIP(ctx context.Context, schoolID string, isPlatformOwner bool, classID, reportType, termID, sessionID string) (data []byte, filename string, skipped []string, err error) {
	entries, className, err := s.collectClassReportCards(ctx, schoolID, isPlatformOwner, classID, reportType, termID, sessionID)
	if err != nil {
		return nil, "", nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		var pdfBytes []byte
		var genErr error
		switch {
		case e.err != nil:
			genErr = e.err
		case e.termly != nil:
			pdfBytes, genErr = pdf.GenerateTermlyReportCard(e.termly)
		case e.midterm != nil:
			pdfBytes, genErr = pdf.GenerateMidTermProgressReport(e.midterm)
		case e.promotion != nil:
			pdfBytes, genErr = pdf.GeneratePromotionExaminationReport(e.promotion)
		}
		if genErr != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%s): %s", e.studentName(), e.student.AdmissionNo, genErr.Error()))
			continue
		}
		w, werr := zw.Create(reportCardFilenameFor(e.student.AdmissionNo, e.studentName()))
		if werr != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%s): %s", e.studentName(), e.student.AdmissionNo, werr.Error()))
			continue
		}
		if _, werr := w.Write(pdfBytes); werr != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%s): %s", e.studentName(), e.student.AdmissionNo, werr.Error()))
		}
	}
	if err := zw.Close(); err != nil {
		return nil, "", skipped, err
	}

	safeClass := strings.NewReplacer(" ", "_", "/", "-").Replace(className)
	if safeClass == "" {
		safeClass = classID
	}
	return buf.Bytes(), fmt.Sprintf("%s_%s_report_cards.zip", safeClass, reportType), skipped, nil
}

// ExportClassReportCards builds a class-wide result sheet (one row per
// student: admission no, name, overall percentage/grade/position,
// approval status) as CSV or Excel - the spreadsheet counterpart to
// GenerateClassReportCardsZIP's per-student PDFs, for a school's own
// record-keeping and printed sign-off sheets.
func (s *ExamService) ExportClassReportCards(ctx context.Context, schoolID string, isPlatformOwner bool, classID, reportType, termID, sessionID, format string) (data []byte, contentType, filename string, err error) {
	summary, err := s.GetClassReportCardSummary(ctx, schoolID, isPlatformOwner, classID, reportType, termID, sessionID)
	if err != nil {
		return nil, "", "", err
	}

	switch format {
	case "csv":
		data = exportClassSummaryToCSV(summary)
		return data, "text/csv", fmt.Sprintf("%s_report_cards.csv", reportType), nil
	case "excel":
		data, err = exportClassSummaryToExcel(summary)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fmt.Sprintf("%s_report_cards.xlsx", reportType), nil
	default:
		return nil, "", "", errors.New("unsupported format")
	}
}

var classSummaryHeaders = []string{"Admission No.", "Student Name", "Overall %", "Overall Grade", "Class Position", "Approval Status", "Approval Version", "Error"}

// sanitizeSpreadsheetCell defends against CSV/Excel formula injection: a
// cell value that opens with =, +, -, @, tab, or CR is interpreted as a
// formula by Excel/LibreOffice/Google Sheets the moment the exported
// file is opened, not displayed as the literal text - a classic way a
// crafted name/remark turns into code execution or data exfiltration for
// whoever opens the sheet. Prefixing a leading apostrophe forces the
// cell to be read as plain text in every one of those programs. Applied
// to every string-origin column here (admission number, name, grade,
// status, error) - none of them are guaranteed free of attacker-
// influenced content (a student record, however implausible, is still
// data this export did not originate itself).
func sanitizeSpreadsheetCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	default:
		return v
	}
}

func classSummaryRowValues(r dto.ClassReportCardSummaryRow) []string {
	position := ""
	if r.ClassPosition > 0 {
		position = strconv.Itoa(r.ClassPosition)
	}
	version := ""
	if r.ApprovalVersion > 0 {
		version = strconv.Itoa(r.ApprovalVersion)
	}
	percentage := ""
	if r.Error == "" {
		percentage = strconv.FormatFloat(r.OverallPercentage, 'f', 1, 64)
	}
	return []string{
		sanitizeSpreadsheetCell(r.AdmissionNumber), sanitizeSpreadsheetCell(r.StudentName),
		percentage, sanitizeSpreadsheetCell(r.OverallGrade), position,
		sanitizeSpreadsheetCell(r.ApprovalStatus), version, sanitizeSpreadsheetCell(r.Error),
	}
}

func exportClassSummaryToCSV(summary *dto.ClassReportCardSummaryResponse) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(classSummaryHeaders)
	for _, row := range summary.Rows {
		w.Write(classSummaryRowValues(row))
	}
	w.Flush()
	return buf.Bytes()
}

func exportClassSummaryToExcel(summary *dto.ClassReportCardSummaryResponse) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Report Cards"
	f.SetSheetName("Sheet1", sheet)

	for i, h := range classSummaryHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, row := range summary.Rows {
		values := classSummaryRowValues(row)
		for j, v := range values {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, v)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
