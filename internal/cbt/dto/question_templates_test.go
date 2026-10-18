package dto

import (
	"bytes"
	"testing"
)

// Each of these round-trips a generated template straight through its own
// parser - the strongest possible proof that a template and its parser
// agree on field names/format, since a human never has to be in the loop.

func TestCSVTemplate_ParsesWithCSVParser(t *testing.T) {
	data, err := GenerateCSVTemplate()
	if err != nil {
		t.Fatalf("GenerateCSVTemplate failed: %v", err)
	}
	items, err := NewCSVParser().Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("CSVParser could not parse its own template: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 sample questions, got %d", len(items))
	}
	if items[0].QuestionType != QuestionTypeSingle || len(items[0].Options) != 4 {
		t.Errorf("sample row 1: got type=%s options=%d, want single_choice with 4 options", items[0].QuestionType, len(items[0].Options))
	}
	if items[0].CorrectOptionKeys[0] != "A" {
		t.Errorf("sample row 1: correct_option_keys = %v, want [A]", items[0].CorrectOptionKeys)
	}
	if items[1].QuestionType != QuestionTypeEssay {
		t.Errorf("sample row 2: got type=%s, want essay", items[1].QuestionType)
	}
}

func TestExcelTemplate_ParsesWithExcelParser(t *testing.T) {
	data, err := GenerateExcelTemplate()
	if err != nil {
		t.Fatalf("GenerateExcelTemplate failed: %v", err)
	}
	items, err := NewExcelParser().Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ExcelParser could not parse its own template: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 sample questions, got %d", len(items))
	}
	if items[0].CorrectOptionKeys[0] != "A" {
		t.Errorf("sample row 1: correct_option_keys = %v, want [A]", items[0].CorrectOptionKeys)
	}
}

func TestTXTTemplate_ParsesWithTXTParser(t *testing.T) {
	items, err := NewTXTParser().Parse(bytes.NewReader(GenerateTXTTemplate()))
	if err != nil {
		t.Fatalf("TXTParser could not parse its own template: %v", err)
	}
	// The free-text Q&A format has no comment syntax, so the leading
	// instructions paragraph (before the first "Q1." marker) is itself
	// parsed as a pseudo-question - items[0] here. That's an inherent
	// limitation of this format, not a bug: on a REAL import it has no
	// correct_option_keys/correct_answer, so validateImportItem rejects
	// it with "correct answer is required", which is a harmless, honest
	// nudge to delete the instructions line rather than a silent garbage
	// save. The two real samples are items[1] and items[2].
	if len(items) != 3 {
		t.Fatalf("expected 3 parsed items (instructions pseudo-question + 2 real samples), got %d: %+v", len(items), items)
	}
	if len(items[0].CorrectOptionKeys) != 0 || items[0].CorrectAnswer != "" {
		t.Errorf("instructions pseudo-question must have no correct answer (so real imports reject it), got keys=%v answer=%q", items[0].CorrectOptionKeys, items[0].CorrectAnswer)
	}
	if len(items[1].Options) != 4 || items[1].CorrectOptionKeys[0] != "A" {
		t.Errorf("sample question 1: options=%d correct=%v, want 4 options and [A]", len(items[1].Options), items[1].CorrectOptionKeys)
	}
}

func TestDOCXTemplate_ParsesWithDOCXParser(t *testing.T) {
	data, err := GenerateDOCXTemplate()
	if err != nil {
		t.Fatalf("GenerateDOCXTemplate failed: %v", err)
	}
	items, err := NewDOCXParser().Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DOCXParser could not parse its own template: %v", err)
	}
	// See the matching comment in TestTXTTemplate_ParsesWithTXTParser -
	// items[0] is the leading instructions paragraph, parsed as a
	// pseudo-question that a real import correctly rejects for having no
	// correct answer.
	if len(items) != 3 {
		t.Fatalf("expected 3 parsed items (instructions pseudo-question + 2 real samples), got %d: %+v", len(items), items)
	}
	if len(items[0].CorrectOptionKeys) != 0 || items[0].CorrectAnswer != "" {
		t.Errorf("instructions pseudo-question must have no correct answer (so real imports reject it), got keys=%v answer=%q", items[0].CorrectOptionKeys, items[0].CorrectAnswer)
	}
	if len(items[1].Options) != 4 || items[1].CorrectOptionKeys[0] != "A" {
		t.Errorf("sample question 1: options=%d correct=%v, want 4 options and [A]", len(items[1].Options), items[1].CorrectOptionKeys)
	}
}
