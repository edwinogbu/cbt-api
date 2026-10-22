package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/pdf"

	"github.com/xuri/excelize/v2"
)

// This file is the export counterpart to broadsheet_views_service.go:
// it builds a broad sheet through the exact same Get*BroadSheet
// functions the JSON endpoints use, then renders that one authoritative
// response as PDF, CSV, or Excel - never a second, independently
// computed export path that could drift from what staff see on screen.

var errUnsupportedBroadSheetFormat = errors.New("format must be one of pdf, csv, excel")

// schoolInfoOrEmpty looks up a school for the PDF masthead, tolerating
// failure - an export must never 500 just because a school record
// couldn't be resolved in some corner case (e.g. a platform_owner
// export for a subject taught in no class yet); the PDF simply renders
// with a blank school name rather than failing outright.
func (s *ExamService) schoolInfoOrEmpty(ctx context.Context, schoolID string) dto.SchoolInfo {
	if schoolID == "" {
		return dto.SchoolInfo{}
	}
	school, err := s.getSchoolByID(ctx, schoolID)
	if err != nil {
		return dto.SchoolInfo{}
	}
	return dto.SchoolInfo{ID: school.ID, Name: school.Name, Address: school.Address, Logo: school.Logo, PrimaryColor: school.PrimaryColor}
}

// safeExportFilenamePart strips characters that are awkward in a
// downloaded filename (path separators, quotes) and collapses
// whitespace - mirrors reportCardFilenameFor's replacer, reused here
// for broad-sheet export filenames.
func safeExportFilenamePart(s string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "_", "\"", "", ":", "-")
	out := replacer.Replace(strings.TrimSpace(s))
	if out == "" {
		return "broadsheet"
	}
	return out
}

// ExportClassBroadSheet builds the Class Teacher Broad Sheet (the same
// response GetClassBroadSheet returns) as PDF, CSV, or Excel.
func (s *ExamService) ExportClassBroadSheet(ctx context.Context, schoolID string, isPlatformOwner bool, classID, reportType, termID, sessionID, format string) (data []byte, contentType, filename string, err error) {
	resp, err := s.GetClassBroadSheet(ctx, schoolID, isPlatformOwner, classID, reportType, termID, sessionID)
	if err != nil {
		return nil, "", "", err
	}
	safeName := safeExportFilenamePart(resp.ClassName) + "_" + reportType + "_broadsheet"

	switch format {
	case "pdf":
		school := s.schoolInfoOrEmpty(ctx, classSchoolIDFallback(s, ctx, classID, schoolID, isPlatformOwner))
		data, err = pdf.GenerateClassBroadSheetPDF(school, resp)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/pdf", safeName + ".pdf", nil
	case "csv":
		return classBroadSheetToCSV(resp), "text/csv", safeName + ".csv", nil
	case "excel":
		data, err = classBroadSheetToExcel(resp)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", safeName + ".xlsx", nil
	default:
		return nil, "", "", errUnsupportedBroadSheetFormat
	}
}

// ExportSubjectBroadSheet builds the Subject (HOD/Examination Officer)
// Broad Sheet as PDF, CSV, or Excel.
func (s *ExamService) ExportSubjectBroadSheet(ctx context.Context, schoolID string, isPlatformOwner bool, subjectID, termID, sessionID, format string) (data []byte, contentType, filename string, err error) {
	resp, err := s.GetSubjectBroadSheet(ctx, schoolID, isPlatformOwner, subjectID, termID, sessionID)
	if err != nil {
		return nil, "", "", err
	}
	safeName := safeExportFilenamePart(resp.SubjectName) + "_subject_broadsheet"

	switch format {
	case "pdf":
		school := s.schoolInfoOrEmpty(ctx, resolveSubjectBroadSheetSchoolID(s, ctx, resp, schoolID, isPlatformOwner))
		data, err = pdf.GenerateSubjectBroadSheetPDF(school, resp)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/pdf", safeName + ".pdf", nil
	case "csv":
		return subjectBroadSheetToCSV(resp), "text/csv", safeName + ".csv", nil
	case "excel":
		data, err = subjectBroadSheetToExcel(resp)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", safeName + ".xlsx", nil
	default:
		return nil, "", "", errUnsupportedBroadSheetFormat
	}
}

// resolveSubjectBroadSheetSchoolID mirrors classSchoolIDFallback for the
// Subject Broad Sheet: a non-platform-owner caller's own schoolID is
// always correct (every row is already scoped to it); a platform_owner
// caller may have none, so fall back to the first row's own class.
func resolveSubjectBroadSheetSchoolID(s *ExamService, ctx context.Context, resp *dto.SubjectBroadSheetResponse, schoolID string, isPlatformOwner bool) string {
	if !isPlatformOwner && schoolID != "" {
		return schoolID
	}
	if len(resp.Rows) > 0 {
		if class, err := s.getClassByID(ctx, resp.Rows[0].ClassID); err == nil {
			return class.SchoolID
		}
	}
	return schoolID
}

// ExportSchoolBroadSheet builds the VP Academics / School General Broad
// Sheet as PDF, CSV, or Excel.
func (s *ExamService) ExportSchoolBroadSheet(ctx context.Context, schoolID string, isPlatformOwner bool, targetSchoolID, termID, sessionID, reportType, format string) (data []byte, contentType, filename string, err error) {
	resp, err := s.GetSchoolBroadSheet(ctx, schoolID, isPlatformOwner, targetSchoolID, termID, sessionID, reportType)
	if err != nil {
		return nil, "", "", err
	}
	safeName := "school_broadsheet_" + reportType

	switch format {
	case "pdf":
		effSchool := examEffectiveSchoolID(targetSchoolID, schoolID, isPlatformOwner)
		school := s.schoolInfoOrEmpty(ctx, effSchool)
		data, err = pdf.GenerateSchoolBroadSheetPDF(school, resp)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/pdf", safeName + ".pdf", nil
	case "csv":
		return schoolBroadSheetToCSV(resp), "text/csv", safeName + ".csv", nil
	case "excel":
		data, err = schoolBroadSheetToExcel(resp)
		if err != nil {
			return nil, "", "", err
		}
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", safeName + ".xlsx", nil
	default:
		return nil, "", "", errUnsupportedBroadSheetFormat
	}
}

// formatScoreCellForExport mirrors the pdf package's own
// formatBroadSheetScoreCell for CSV/Excel cells - kept as a small
// separate copy rather than exporting the pdf package's unexported
// helper, since the two renderers have no other reason to share code.
func formatScoreCellForExport(cell dto.ClassBroadSheetSubjectCell) string {
	if cell.Score == nil {
		return ""
	}
	s := strconv.Itoa(*cell.Score)
	if cell.Grade != "" {
		s += " (" + cell.Grade + ")"
	}
	if !cell.IsComplete {
		s += " *incomplete"
	}
	return s
}

func classBroadSheetHeaders(resp *dto.ClassBroadSheetResponse) []string {
	headers := []string{"Admission No.", "Student Name"}
	headers = append(headers, resp.SubjectNames...)
	return append(headers, "Overall %", "Pass", "Fail", "Incomplete", "Days Present", "Days Open")
}

func classBroadSheetRowValues(row dto.ClassBroadSheetRow) []string {
	values := []string{sanitizeSpreadsheetCell(row.AdmissionNumber), sanitizeSpreadsheetCell(row.StudentName)}
	for _, cell := range row.Subjects {
		values = append(values, sanitizeSpreadsheetCell(formatScoreCellForExport(cell)))
	}
	overall := ""
	if row.OverallAverage != nil {
		overall = strconv.FormatFloat(*row.OverallAverage, 'f', 1, 64)
	}
	values = append(values,
		overall, strconv.Itoa(row.Pass), strconv.Itoa(row.Fail), strconv.Itoa(row.Incomplete),
		strconv.Itoa(row.DaysPresent), strconv.Itoa(row.DaysOpen),
	)
	return values
}

func classBroadSheetToCSV(resp *dto.ClassBroadSheetResponse) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(classBroadSheetHeaders(resp))
	for _, row := range resp.Rows {
		w.Write(classBroadSheetRowValues(row))
	}
	w.Flush()
	return buf.Bytes()
}

func classBroadSheetToExcel(resp *dto.ClassBroadSheetResponse) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Class Broad Sheet"
	f.SetSheetName("Sheet1", sheet)
	headers := classBroadSheetHeaders(resp)
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, row := range resp.Rows {
		for j, v := range classBroadSheetRowValues(row) {
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

var subjectBroadSheetHeaders = []string{"Class", "Teacher", "Enrolled", "Marks Entered", "Incomplete", "Class Avg %", "Pass Rate %", "Status"}

func subjectBroadSheetRowValues(r dto.SubjectBroadSheetRow) []string {
	return []string{
		sanitizeSpreadsheetCell(r.ClassName), sanitizeSpreadsheetCell(r.TeacherName),
		strconv.Itoa(r.Enrolled), strconv.Itoa(r.MarksEntered), strconv.Itoa(r.Incomplete),
		strconv.FormatFloat(r.ClassAverage, 'f', 1, 64), strconv.FormatFloat(r.PassRate, 'f', 1, 64),
		sanitizeSpreadsheetCell(r.SubmissionStatus),
	}
}

func subjectBroadSheetToCSV(resp *dto.SubjectBroadSheetResponse) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(subjectBroadSheetHeaders)
	for _, row := range resp.Rows {
		w.Write(subjectBroadSheetRowValues(row))
	}
	w.Flush()
	return buf.Bytes()
}

func subjectBroadSheetToExcel(resp *dto.SubjectBroadSheetResponse) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Subject Broad Sheet"
	f.SetSheetName("Sheet1", sheet)
	for i, h := range subjectBroadSheetHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, row := range resp.Rows {
		for j, v := range subjectBroadSheetRowValues(row) {
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

var schoolBroadSheetHeaders = []string{"Class", "Subject", "Teacher", "Enrolled", "Entered", "Incomplete", "Class Avg %", "Pass Rate %", "Submitted", "Reviewed", "Approved/Published"}

func boolCell(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func schoolBroadSheetRowValues(r dto.SchoolBroadSheetRow) []string {
	return []string{
		sanitizeSpreadsheetCell(r.ClassName), sanitizeSpreadsheetCell(r.SubjectName), sanitizeSpreadsheetCell(r.TeacherName),
		strconv.Itoa(r.Enrolled), strconv.Itoa(r.MarksEntered), strconv.Itoa(r.Incomplete),
		strconv.FormatFloat(r.ClassAveragePct, 'f', 1, 64), strconv.FormatFloat(r.PassRatePct, 'f', 1, 64),
		boolCell(r.Submitted), boolCell(r.Reviewed), boolCell(r.ApprovedPublished),
	}
}

func schoolBroadSheetToCSV(resp *dto.SchoolBroadSheetResponse) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(schoolBroadSheetHeaders)
	for _, row := range resp.Rows {
		w.Write(schoolBroadSheetRowValues(row))
	}
	w.Flush()
	return buf.Bytes()
}

func schoolBroadSheetToExcel(resp *dto.SchoolBroadSheetResponse) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "School Broad Sheet"
	f.SetSheetName("Sheet1", sheet)
	for i, h := range schoolBroadSheetHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, row := range resp.Rows {
		for j, v := range schoolBroadSheetRowValues(row) {
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
