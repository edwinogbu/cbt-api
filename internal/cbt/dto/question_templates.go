package dto

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ============================================================
// DOWNLOADABLE IMPORT TEMPLATES
//
// These MUST stay in lockstep with CSVParser/ExcelParser's column names
// (buildColumnMap/parseRow above) and TXTParser/DOCXParser's free-text
// Q&A pattern (parseQuestionsFromText) - a template that doesn't match
// its own parser is worse than no template at all.
// ============================================================

// templateColumns is the single source of truth for the column-based
// (CSV/Excel) template header row, shared with the human-readable
// instructions below so the two can't drift apart.
var templateColumns = []string{
	"question_text", "question_type", "difficulty", "bloom_level",
	"topic", "sub_topic", "option_a", "option_b", "option_c", "option_d",
	"correct_answer", "explanation", "marks", "tags",
}

var templateSampleRows = [][]string{
	{
		"What is the powerhouse of the cell?", "single_choice", "easy", "remember",
		"Cell Biology", "Organelles", "Mitochondria", "Nucleus", "Ribosome", "Golgi body",
		"A", "The mitochondria produces most of the cell's ATP through respiration.", "1", "biology,cells",
	},
	{
		"Explain the process of photosynthesis and state its overall chemical equation.", "essay", "medium", "understand",
		"Photosynthesis", "", "", "", "", "",
		"", "A full answer should mention sunlight, chlorophyll, carbon dioxide, water, glucose, and oxygen.", "10", "biology,photosynthesis",
	},
}

// GenerateCSVTemplate returns a downloadable CSV template matching
// CSVParser's exact column names, with two sample rows (one objective,
// one subjective) that MUST be removed before a real import.
func GenerateCSVTemplate() ([]byte, error) {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	if err := w.Write(templateColumns); err != nil {
		return nil, fmt.Errorf("failed to write template header: %w", err)
	}
	for _, row := range templateSampleRows {
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write template sample row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateExcelTemplate returns a downloadable .xlsx template: a
// "Questions" sheet matching ExcelParser's column names (identical to the
// CSV template's), plus an "Instructions" sheet explaining every field,
// valid values, and which rows are samples to delete.
func GenerateExcelTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Questions"
	f.SetSheetName(f.GetSheetName(0), sheet)
	for i, col := range templateColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col)
	}
	for r, row := range templateSampleRows {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cell, val)
		}
	}
	f.SetColWidth(sheet, "A", "A", 45)
	f.SetColWidth(sheet, "G", "J", 20)
	f.SetColWidth(sheet, "L", "L", 45)

	const instructionsSheet = "Instructions"
	f.NewSheet(instructionsSheet)
	instructions := [][]string{
		{"Field", "Required?", "Notes"},
		{"question_text", "Required", "The full question text."},
		{"question_type", "Required", "One of: single_choice, multiple_choice, true_false, essay, fill_blank"},
		{"difficulty", "Optional (defaults to medium)", "One of: easy, medium, hard, expert"},
		{"bloom_level", "Optional (defaults to remember)", "One of: remember, understand, apply, analyse, evaluate, create"},
		{"topic", "Optional (defaults to General)", "Topic this question belongs to."},
		{"sub_topic", "Optional", "A more specific sub-topic, if applicable."},
		{"option_a / option_b / option_c / option_d", "Required for single_choice/multiple_choice/true_false", "Leave option_c/option_d blank for true_false (use only option_a=True, option_b=False)."},
		{"correct_answer", "Required for objective question types", "Enter the OPTION LETTER only (A, B, C, or D) - not the option text."},
		{"explanation", "Optional", "Shown to students/teachers as the reason the answer is correct; for essay questions, use this for a suggested marking guide."},
		{"marks", "Optional (defaults to 2, or 1 for true_false/fill_blank)", "A whole number."},
		{"tags", "Optional", "Comma-separated, e.g. biology,cells"},
		{"", "", ""},
		{"IMPORTANT", "", "Delete the 2 sample rows on the Questions sheet before uploading - they are examples only, not real questions."},
	}
	for r, row := range instructions {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			f.SetCellValue(instructionsSheet, cell, val)
		}
	}
	f.SetColWidth(instructionsSheet, "A", "A", 30)
	f.SetColWidth(instructionsSheet, "C", "C", 70)
	if idx, err := f.GetSheetIndex(sheet); err == nil {
		f.SetActiveSheet(idx)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Excel template: %w", err)
	}
	return buf.Bytes(), nil
}

// txtTemplateBody is the exact format TXTParser/RTFParser/PDFParser's
// shared parseQuestionsFromText expects: a numbered "Q<n>." or "<n>."
// marker starts each question, lines beginning "A." / "B." / "C." / "D."
// are options (any of "A.", "A)" work), and a line containing
// "Answer:"/"Ans:" gives the correct option letter. Everything after the
// marker up to the next marker belongs to one question.
// NOTE: this instructions paragraph deliberately never writes "answer:" or
// "ans:" with no space before the colon - parseQuestionsFromText's
// answer-line detection matches that substring anywhere in a line, and
// this paragraph (as the text preceding the first "Q1." marker) is itself
// parsed as a candidate question, so writing the trigger literally here
// would make the instructions misparse as having a bogus correct answer.
const txtTemplateBody = `IMPORTANT: Delete this instructions block and the 2 sample questions below before uploading. Keep one blank line between questions. Each question must start with "Q<number>." on its own line. Each option must be on its own line, starting with "A.", "B.", "C." or "D.". The line giving the correct option must contain the word ANSWER followed by a colon and the option letter. Essay/subjective questions do not need options - just the question text and, optionally, a line with the marking guide.

Q1. What is the powerhouse of the cell?
A. Mitochondria
B. Nucleus
C. Ribosome
D. Golgi body
Answer: A

Q2. Explain the process of photosynthesis and state its overall chemical equation.
Answer: A full answer should mention sunlight, chlorophyll, carbon dioxide, water, glucose, and oxygen.
`

// GenerateTXTTemplate returns a downloadable plain-text template matching
// the exact pattern parseQuestionsFromText expects.
func GenerateTXTTemplate() []byte {
	return []byte(txtTemplateBody)
}

// GenerateDOCXTemplate returns a downloadable .docx template: a real,
// minimal OOXML document (one paragraph per line, exactly matching
// extractTextFromDOCX's paragraph-boundary-preserving extraction) using
// the same Q&A pattern as the TXT template.
func GenerateDOCXTemplate() ([]byte, error) {
	// See the comment on txtTemplateBody above: deliberately avoids
	// writing the literal "answer:"/"ans:" trigger substring, since this
	// whole paragraph (preceding the first "Q1." marker) is itself parsed
	// as a candidate question.
	lines := []string{
		`IMPORTANT: Delete this instructions paragraph and the 2 sample questions below before uploading. Each question must start with "Q<number>." on its own line/paragraph. Each option must be its own paragraph starting with "A.", "B.", "C." or "D.". The paragraph giving the correct option must contain the word ANSWER followed by a colon and the option letter.`,
		"",
		"Q1. What is the powerhouse of the cell?",
		"A. Mitochondria",
		"B. Nucleus",
		"C. Ribosome",
		"D. Golgi body",
		"Answer: A",
		"",
		"Q2. Explain the process of photosynthesis and state its overall chemical equation.",
		"Answer: A full answer should mention sunlight, chlorophyll, carbon dioxide, water, glucose, and oxygen.",
	}

	var paragraphs bytes.Buffer
	for _, line := range lines {
		paragraphs.WriteString(`<w:p><w:r><w:t xml:space="preserve">`)
		paragraphs.WriteString(xmlEscape(line))
		paragraphs.WriteString(`</w:t></w:r></w:p>`)
	}

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + paragraphs.String() + `</w:body></w:document>`

	contentTypesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`</Types>`

	rootRelsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`</Relationships>`

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	files := map[string]string{
		"[Content_Types].xml": contentTypesXML,
		"_rels/.rels":         rootRelsXML,
		"word/document.xml":   documentXML,
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s in docx template: %w", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return nil, fmt.Errorf("failed to write %s in docx template: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize docx template: %w", err)
	}
	return buf.Bytes(), nil
}

func xmlEscape(s string) string {
	var out bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			out.WriteString("&amp;")
		case '<':
			out.WriteString("&lt;")
		case '>':
			out.WriteString("&gt;")
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
