package dto

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestExtractTextFromRTF(t *testing.T) {
	// A minimal, realistic RTF document as produced by WordPad/Word: a
	// font table, a color table, and two paragraphs of plain text.
	rtf := `{\rtf1\ansi\deff0
{\fonttbl{\f0\fswiss\fcharset0 Arial;}}
{\colortbl ;\red0\green0\blue0;}
\f0\fs24 Q: What is the capital of Nigeria?\par
A. Lagos\par
B. Abuja\par
Answer: B\par
}`
	got := extractTextFromRTF(rtf)

	if strings.Contains(got, "fonttbl") || strings.Contains(got, "colortbl") {
		t.Errorf("extractTextFromRTF leaked table group content: %q", got)
	}
	if !strings.Contains(got, "What is the capital of Nigeria?") {
		t.Errorf("extractTextFromRTF dropped question text: %q", got)
	}
	if !strings.Contains(got, "Abuja") || !strings.Contains(got, "Lagos") {
		t.Errorf("extractTextFromRTF dropped option text: %q", got)
	}
	if !strings.Contains(got, "Answer: B") {
		t.Errorf("extractTextFromRTF dropped the answer line: %q", got)
	}
}

func TestRTFParser_Parse(t *testing.T) {
	rtf := `{\rtf1\ansi\deff0
{\fonttbl{\f0\fswiss\fcharset0 Arial;}}
\f0\fs24 Q: What is the capital of Nigeria?\par
A. Lagos\par
B. Abuja\par
C. Kano\par
D. Ibadan\par
Answer: B\par
}`
	parser := NewRTFParser()
	items, err := parser.Parse(strings.NewReader(rtf))
	if err != nil {
		t.Fatalf("RTFParser.Parse failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 question, got %d: %+v", len(items), items)
	}
	if !strings.Contains(items[0].QuestionText, "capital of Nigeria") {
		t.Errorf("unexpected question text: %q", items[0].QuestionText)
	}
}

func TestXLSParser_RejectsGarbage(t *testing.T) {
	parser := NewXLSParser()
	_, err := parser.Parse(strings.NewReader("this is not a real xls file"))
	if err == nil {
		t.Fatal("expected an error for a non-XLS file, got nil")
	}
}

func TestPDFParser_RejectsGarbage(t *testing.T) {
	parser := NewPDFParser()
	_, err := parser.Parse(strings.NewReader("this is not a real pdf file"))
	if err == nil {
		t.Fatal("expected an error for a non-PDF file, got nil")
	}
}

func TestDetectFormatFromExtension_LegacyXLSIsDistinctFromXLSX(t *testing.T) {
	if DetectFormatFromExtension("questions.xlsx") != UploadFormatExcel {
		t.Errorf(".xlsx should detect as %q", UploadFormatExcel)
	}
	if DetectFormatFromExtension("questions.xls") != UploadFormatXLS {
		t.Errorf(".xls should detect as %q (legacy, excelize-incompatible), not %q", UploadFormatXLS, UploadFormatExcel)
	}
	if DetectFormatFromExtension("questions.rtf") != UploadFormatRTF {
		t.Errorf(".rtf should detect as %q", UploadFormatRTF)
	}
	if DetectFormatFromExtension("questions.pdf") != UploadFormatPDF {
		t.Errorf(".pdf should detect as %q", UploadFormatPDF)
	}
}

func TestParserFactory_SupportsAllAdvertisedFormats(t *testing.T) {
	factory := NewParserFactory()
	for _, format := range []string{UploadFormatCSV, UploadFormatExcel, UploadFormatJSON, UploadFormatDOCX, UploadFormatTXT, UploadFormatRTF, UploadFormatPDF, UploadFormatXLS} {
		parser, err := factory.GetParser(format)
		if err != nil {
			t.Errorf("GetParser(%q) returned an error: %v", format, err)
			continue
		}
		if parser.SupportedFormat() != format {
			t.Errorf("GetParser(%q).SupportedFormat() = %q", format, parser.SupportedFormat())
		}
	}
}

func TestExcelParser_ParseInt_RecoversXLSDateMisread(t *testing.T) {
	p := &ExcelParser{}
	cases := map[string]int{
		"1":                    1,
		"  42 ":                42,
		"":                     0,
		"not a number":         0,
		"1899-12-31T00:00:00Z": 1, // extrame/xls misreading marks=1 as a date (found live)
		"1900-01-01T00:00:00Z": 2,
	}
	for input, want := range cases {
		if got := p.parseInt(input); got != want {
			t.Errorf("parseInt(%q) = %d, want %d", input, got, want)
		}
	}
}

// buildTestDOCX constructs a minimal, real OOXML .docx in memory (just a
// zip with a word/document.xml entry, which is all DOCXParser reads), one
// paragraph per line, mirroring exactly how a real Word document or
// python-docx encodes separate lines - this is what caught the paragraph-
// boundary bug live and pins the fix as a regression test.
func buildTestDOCX(t *testing.T, lines []string) []byte {
	t.Helper()
	var paragraphs strings.Builder
	for _, line := range lines {
		paragraphs.WriteString(`<w:p><w:r><w:t>`)
		paragraphs.WriteString(line)
		paragraphs.WriteString(`</w:t></w:r></w:p>`)
	}
	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>` + paragraphs.String() + `</w:body></w:document>`

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := w.Write([]byte(documentXML)); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}
	return buf.Bytes()
}

func TestDOCXParser_PreservesParagraphBoundaries(t *testing.T) {
	docxBytes := buildTestDOCX(t, []string{
		"Q1. What is the capital of Nigeria?",
		"A. Lagos",
		"B. Abuja",
		"C. Kano",
		"D. Ibadan",
		"Answer: B",
	})

	items, err := NewDOCXParser().Parse(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("DOCXParser.Parse failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 question, got %d: %+v", len(items), items)
	}
	q := items[0]
	if strings.Contains(q.QuestionText, "Answer:") || strings.Contains(q.QuestionText, "Lagos") {
		t.Errorf("question text leaked options/answer (paragraph boundaries lost): %q", q.QuestionText)
	}
	if len(q.Options) != 4 {
		t.Fatalf("expected 4 options, got %d: %+v", len(q.Options), q.Options)
	}
	want := map[string]string{"A": "Lagos", "B": "Abuja", "C": "Kano", "D": "Ibadan"}
	for _, opt := range q.Options {
		if want[opt.Key] != opt.Text {
			t.Errorf("option %s: got %q, want %q", opt.Key, opt.Text, want[opt.Key])
		}
	}
	if len(q.CorrectOptionKeys) != 1 || q.CorrectOptionKeys[0] != "B" {
		t.Errorf("correct_option_keys = %v, want [B]", q.CorrectOptionKeys)
	}
}
