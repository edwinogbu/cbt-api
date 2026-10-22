package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Phase 2n: question-bank grouped-endpoint tenant isolation ----
//
// Before this fix:
//   - GetAllQuestionsGrouped ran s.db.Find(&questions) with NO filter at
//     all - GET /questions/all-grouped (and POST /questions/grouped with an
//     empty body, which falls through to it) returned every school's
//     entire question bank, draft/published/archived alike, to any
//     authenticated caller.
//   - GetQuestionsGrouped trusted req.SchoolID straight from the client
//     with no override, so any authenticated caller could read another
//     school's question bank just by naming its school_id.
//   - GetQuestionsByTermWithGrouping/GetQuestionsBySessionWithGrouping
//     looked up questions by term_id/session_id with no school check at
//     all, then derived the "school" from the first matching question's
//     OWN record instead of verifying it against the caller - a valid
//     term_id/session_id belonging to another school leaked straight
//     through.

func setupGroupedTenantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE question_bank (
			id TEXT PRIMARY KEY, school_id TEXT, session_id TEXT, term_id TEXT,
			class_level_id TEXT, class_id TEXT, subject_id TEXT, exam_type TEXT,
			question_text TEXT, question_type TEXT, difficulty TEXT, bloom_level TEXT,
			topic TEXT, sub_topic TEXT, learning_objective TEXT,
			options TEXT, correct_option_keys TEXT, correct_answer TEXT, rubric TEXT, tags TEXT,
			explanation TEXT, marks INTEGER DEFAULT 1, negative_marks REAL DEFAULT 0,
			time_limit_seconds INTEGER, "order" INTEGER DEFAULT 0, is_required INTEGER DEFAULT 0,
			status TEXT DEFAULT 'draft', version INTEGER DEFAULT 1, parent_id TEXT,
			created_by TEXT, updated_by TEXT, usage_count INTEGER DEFAULT 0, success_rate REAL,
			curriculum_type TEXT, source_type TEXT, external_id TEXT, metadata TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE academic_sessions (id TEXT PRIMARY KEY, school_id TEXT, name TEXT, start_date DATETIME, end_date DATETIME, is_current INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE terms (id TEXT PRIMARY KEY, session_id TEXT, term_number INTEGER, name TEXT, start_date DATETIME, end_date DATETIME, is_current INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE subjects (id TEXT PRIMARY KEY, name TEXT, code TEXT, description TEXT, is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func seedGroupedQuestion(t *testing.T, db *gorm.DB, id, schoolID, subjectID, sessionID, termID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO question_bank (id, school_id, subject_id, session_id, term_id, question_text, marks, status, exam_type, question_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, schoolID, subjectID, sessionID, termID, "What is 2+2?", 1, "published", "practice", "single_choice",
	).Error; err != nil {
		t.Fatalf("failed to seed question: %v", err)
	}
}

func seedAcademicSessionForSchool(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO academic_sessions (id, school_id, name) VALUES (?, ?, ?)", id, schoolID, "2024/2025").Error; err != nil {
		t.Fatalf("failed to seed academic session: %v", err)
	}
}

func seedTermForSession(t *testing.T, db *gorm.DB, id, sessionID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO terms (id, session_id, term_number, name) VALUES (?, ?, ?, ?)", id, sessionID, 1, "First Term").Error; err != nil {
		t.Fatalf("failed to seed term: %v", err)
	}
}

func countGroupedQuestions(resp *dto.QuestionGroupsResponse) int {
	total := 0
	for _, group := range resp.QuestionGroups {
		total += group.TotalQuestions
	}
	return total
}

func TestGetAllQuestionsGrouped_NonPlatformOwner_OnlyOwnSchool(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "")
	seedGroupedQuestion(t, db, "q-B", "school-B", "subj-1", "", "")

	resp, err := s.GetAllQuestionsGrouped(context.Background(), "school-B", false)
	if err != nil {
		t.Fatalf("expected School B to read its own all-grouped questions, got error: %v", err)
	}
	if got := countGroupedQuestions(resp); got != 1 {
		t.Fatalf("expected School B to see exactly 1 question (its own), got %d - cross-tenant leak", got)
	}
}

func TestGetAllQuestionsGrouped_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "")

	if _, err := s.GetAllQuestionsGrouped(context.Background(), "", false); err == nil {
		t.Fatal("expected a non-platform-owner with no school_id of their own to be rejected, got no error")
	}
}

func TestGetAllQuestionsGrouped_PlatformOwner_SeesEverySchool(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "")
	seedGroupedQuestion(t, db, "q-B", "school-B", "subj-1", "", "")

	resp, err := s.GetAllQuestionsGrouped(context.Background(), "", true)
	if err != nil {
		t.Fatalf("expected platform_owner to read every school, got error: %v", err)
	}
	if got := countGroupedQuestions(resp); got != 2 {
		t.Fatalf("expected platform_owner to see both schools' questions (2), got %d", got)
	}
}

func TestGetQuestionsGrouped_CrossTenant_ClientSchoolIDIgnored(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "")
	seedGroupedQuestion(t, db, "q-B", "school-B", "subj-1", "", "")

	// Attacker is authenticated as School B but names School A's
	// school_id in the request body.
	req := &dto.FilterQuestionsGroupedRequest{SchoolID: "school-A", SubjectID: "subj-1"}
	resp, err := s.GetQuestionsGrouped(context.Background(), req, "school-B", false)
	if err != nil {
		t.Fatalf("expected School B's caller to get its own results, not an error: %v", err)
	}
	if got := countGroupedQuestions(resp); got != 1 {
		t.Fatalf("expected the client-supplied school_id=school-A to be ignored and only School B's 1 question returned, got %d", got)
	}
}

func TestGetQuestionsGrouped_PlatformOwner_HonorsClientSchoolID(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "")
	seedGroupedQuestion(t, db, "q-B", "school-B", "subj-1", "", "")

	req := &dto.FilterQuestionsGroupedRequest{SchoolID: "school-A", SubjectID: "subj-1"}
	resp, err := s.GetQuestionsGrouped(context.Background(), req, "", true)
	if err != nil {
		t.Fatalf("expected platform_owner to target School A explicitly, got error: %v", err)
	}
	if got := countGroupedQuestions(resp); got != 1 {
		t.Fatalf("expected platform_owner's explicit school_id=school-A to return School A's 1 question, got %d", got)
	}
}

func TestGetQuestionsByTermWithGrouping_CrossTenant_Rejected(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAcademicSessionForSchool(t, db, "sess-A", "school-A")
	seedTermForSession(t, db, "term-A", "sess-A")
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "term-A")

	if _, err := s.GetQuestionsByTermWithGrouping(context.Background(), "subj-1", "term-A", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected for a term belonging to School A, got no error")
	}
}

func TestGetQuestionsByTermWithGrouping_SameTenant_Allowed(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAcademicSessionForSchool(t, db, "sess-A", "school-A")
	seedTermForSession(t, db, "term-A", "sess-A")
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "term-A")

	resp, err := s.GetQuestionsByTermWithGrouping(context.Background(), "subj-1", "term-A", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own term's questions, got error: %v", err)
	}
	if got := countGroupedQuestions(resp); got != 1 {
		t.Fatalf("expected 1 question for School A's own term, got %d", got)
	}
}

func TestGetQuestionsByTermWithGrouping_PlatformOwner_Bypasses(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAcademicSessionForSchool(t, db, "sess-A", "school-A")
	seedTermForSession(t, db, "term-A", "sess-A")
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "term-A")

	if _, err := s.GetQuestionsByTermWithGrouping(context.Background(), "subj-1", "term-A", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's term questions, got error: %v", err)
	}
}

func TestGetQuestionsBySessionWithGrouping_CrossTenant_Rejected(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAcademicSessionForSchool(t, db, "sess-A", "school-A")
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "")

	if _, err := s.GetQuestionsBySessionWithGrouping(context.Background(), "subj-1", "sess-A", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected for a session belonging to School A, got no error")
	}
}

func TestGetQuestionsBySessionWithGrouping_SameTenant_Allowed(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAcademicSessionForSchool(t, db, "sess-A", "school-A")
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "")

	resp, err := s.GetQuestionsBySessionWithGrouping(context.Background(), "subj-1", "sess-A", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own session's questions, got error: %v", err)
	}
	if got := countGroupedQuestions(resp); got != 1 {
		t.Fatalf("expected 1 question for School A's own session, got %d", got)
	}
}
