// pkg/excel/reader.go
package excel

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xuri/excelize/v2"
)

// StudentRow represents a row from the Excel file
type StudentRow struct {
	FirstName     string
	LastName      string
	AdmissionNo   string
	Username      string // optional
	Gender        string
	DateOfBirth   *time.Time
	Address       string
	GuardianName  string
	GuardianPhone string
	GuardianEmail string
	RowNumber     int
}

// ReadStudentsFromExcel reads the uploaded Excel file and returns a slice of StudentRow
func ReadStudentsFromExcel(file io.Reader) ([]StudentRow, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get the first sheet name
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("no sheets found in Excel file")
	}
	sheetName := sheets[0]

	// Read all rows
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("Excel file must have at least a header row and one data row")
	}

	// Expected headers (case-insensitive)
	expectedHeaders := map[string]int{
		"first_name":      -1,
		"last_name":       -1,
		"admission_no":    -1,
		"username":        -1,
		"gender":          -1,
		"date_of_birth":   -1,
		"address":         -1,
		"guardian_name":   -1,
		"guardian_phone":  -1,
		"guardian_email":  -1,
	}

	// Map header row
	headerRow := rows[0]
	for i, col := range headerRow {
		colLower := normalizeHeader(col)
		if _, ok := expectedHeaders[colLower]; ok {
			expectedHeaders[colLower] = i
		}
	}

	// Validate required columns
	required := []string{"first_name", "last_name", "admission_no"}
	for _, req := range required {
		if expectedHeaders[req] == -1 {
			return nil, fmt.Errorf("required column '%s' not found", req)
		}
	}

	var students []StudentRow
	// Start from row 1 (skip header)
	for idx := 1; idx < len(rows); idx++ {
		row := rows[idx]
		student := StudentRow{RowNumber: idx + 1}

		// Helper to get cell value
		getCell := func(colName string) string {
			colIdx := expectedHeaders[colName]
			if colIdx >= 0 && colIdx < len(row) {
				return row[colIdx]
			}
			return ""
		}

		student.FirstName = getCell("first_name")
		student.LastName = getCell("last_name")
		student.AdmissionNo = getCell("admission_no")
		student.Username = getCell("username")
		student.Gender = getCell("gender")
		student.Address = getCell("address")
		student.GuardianName = getCell("guardian_name")
		student.GuardianPhone = getCell("guardian_phone")
		student.GuardianEmail = getCell("guardian_email")

		// Parse date of birth if present
		dobStr := getCell("date_of_birth")
		if dobStr != "" {
			// Try common formats
			layouts := []string{"2006-01-02", "02/01/2006", "01/02/2006", "2006/01/02"}
			var parsedTime time.Time
			var parseErr error
			for _, layout := range layouts {
				parsedTime, parseErr = time.Parse(layout, dobStr)
				if parseErr == nil {
					break
				}
			}
			if parseErr == nil {
				student.DateOfBirth = &parsedTime
			} else {
				// optional: log warning but continue
			}
		}

		// Validate required fields
		if student.FirstName == "" {
			return nil, fmt.Errorf("row %d: first name is required", student.RowNumber)
		}
		if student.LastName == "" {
			return nil, fmt.Errorf("row %d: last name is required", student.RowNumber)
		}
		if student.AdmissionNo == "" {
			return nil, fmt.Errorf("row %d: admission number is required", student.RowNumber)
		}

		students = append(students, student)
	}
	return students, nil
}

func normalizeHeader(s string) string {
	// Convert to lowercase and replace spaces with underscores
	// e.g., "First Name" -> "first_name"
	result := ""
	for _, ch := range s {
		if ch == ' ' || ch == '-' {
			result += "_"
		} else {
			result += string(ch)
		}
	}
	// simple lowercase conversion
	// we'll implement a basic version
	return toLowerKeepUnderscore(result)
}

func toLowerKeepUnderscore(s string) string {
	b := make([]byte, len(s))
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			b[i] = byte(c + 32)
		} else {
			b[i] = byte(c)
		}
	}
	return string(b)
}