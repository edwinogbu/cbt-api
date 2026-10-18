package dto

import (
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
