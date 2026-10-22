package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Phase 2o: GetJobStatus + question-bank "context" family tenant isolation ----
//
// Before this fix:
//   - GetJobStatus had NO ownership check of any kind - any authenticated
//     caller could poll ANY AI job's status/error message just by naming
//     its UUID. AIQuestionGenerationJob has no school_id column at all, so
//     the fix checks the job's own creator (user_id) instead.
//   - GetQuestionsByTerm/GetQuestionsBySession/GetQuestionsByClass called
//     FindByTerm/FindBySession/FindByClass with no school filter at all.
//   - GetQuestionContextSummary ran a raw SQL query filtered only by the
//     (global) subject_id, with no school filter - any authenticated
//     caller could see every school's term-by-term question counts, and
//     the underlying questions themselves, for any subject.
//   - GetQuestionsWithContext trusted req.SchoolID straight from the
//     client with no override.
//   - GetQuestionsForExam queried with no school filter at all.

func setupJobTenantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE ai_question_generation_jobs (
			id TEXT PRIMARY KEY, user_id TEXT, subject_id TEXT, topic TEXT,
			number_of_questions INTEGER, difficulty TEXT, bloom_level TEXT,
			source_text TEXT, generated_questions TEXT, status TEXT DEFAULT 'pending',
			error_message TEXT, created_at DATETIME, completed_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func seedAIJob(t *testing.T, db *gorm.DB, id, userID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO ai_question_generation_jobs (id, user_id, subject_id, status, created_at) VALUES (?, ?, ?, ?, datetime('now'))",
		id, userID, "subj-1", "completed",
	).Error; err != nil {
		t.Fatalf("failed to seed AI job: %v", err)
	}
}

func TestGetJobStatus_CrossUser_Rejected(t *testing.T) {
	db := setupJobTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAIJob(t, db, "job-1", "teacher-A")

	if _, err := s.GetJobStatus(context.Background(), "job-1", "teacher-B", false); err == nil {
		t.Fatal("expected a different user polling job-1 to be rejected, got no error")
	}
}

func TestGetJobStatus_SameUser_Allowed(t *testing.T) {
	db := setupJobTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAIJob(t, db, "job-1", "teacher-A")

	resp, err := s.GetJobStatus(context.Background(), "job-1", "teacher-A", false)
	if err != nil {
		t.Fatalf("expected the job's own creator to read its status, got error: %v", err)
	}
	if resp.JobID != "job-1" {
		t.Fatalf("expected job-1, got %s", resp.JobID)
	}
}

func TestGetJobStatus_PlatformOwner_Bypasses(t *testing.T) {
	db := setupJobTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAIJob(t, db, "job-1", "teacher-A")

	if _, err := s.GetJobStatus(context.Background(), "job-1", "owner-user", true); err != nil {
		t.Fatalf("expected platform_owner to read any user's job, got error: %v", err)
	}
}

func TestGetQuestionsByTerm_CrossTenant_Empty(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "term-A")

	questions, err := s.GetQuestionsByTerm(context.Background(), "subj-1", "term-A", "school-B", false)
	if err != nil {
		t.Fatalf("expected School B's caller to get its own (empty) results, not an error: %v", err)
	}
	if len(questions) != 0 {
		t.Fatalf("expected School B to see 0 questions for School A's term, got %d - cross-tenant leak", len(questions))
	}
}

func TestGetQuestionsByTerm_SameTenant_Allowed(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "term-A")

	questions, err := s.GetQuestionsByTerm(context.Background(), "subj-1", "term-A", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own term's questions, got error: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question for School A's own term, got %d", len(questions))
	}
}

func TestGetQuestionsByTerm_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)

	if _, err := s.GetQuestionsByTerm(context.Background(), "subj-1", "term-A", "", false); err == nil {
		t.Fatal("expected a non-platform-owner with no school_id of their own to be rejected, got no error")
	}
}

func TestGetQuestionsBySession_CrossTenant_Empty(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "")

	questions, err := s.GetQuestionsBySession(context.Background(), "subj-1", "sess-A", "school-B", false)
	if err != nil {
		t.Fatalf("expected School B's caller to get its own (empty) results, not an error: %v", err)
	}
	if len(questions) != 0 {
		t.Fatalf("expected School B to see 0 questions for School A's session, got %d - cross-tenant leak", len(questions))
	}
}

func TestGetQuestionsBySession_SameTenant_Allowed(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "")

	questions, err := s.GetQuestionsBySession(context.Background(), "subj-1", "sess-A", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own session's questions, got error: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question for School A's own session, got %d", len(questions))
	}
}

func TestGetQuestionsByClass_CrossTenant_Empty(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	if err := db.Exec(
		"INSERT INTO question_bank (id, school_id, subject_id, class_id, question_text, marks, status, exam_type, question_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"q-A", "school-A", "subj-1", "class-A", "What is 2+2?", 1, "published", "practice", "single_choice",
	).Error; err != nil {
		t.Fatalf("failed to seed question: %v", err)
	}

	questions, err := s.GetQuestionsByClass(context.Background(), "class-A", "school-B", false)
	if err != nil {
		t.Fatalf("expected School B's caller to get its own (empty) results, not an error: %v", err)
	}
	if len(questions) != 0 {
		t.Fatalf("expected School B to see 0 questions for School A's class, got %d - cross-tenant leak", len(questions))
	}
}

func TestGetQuestionContextSummary_CrossTenant_Empty(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAcademicSessionForSchool(t, db, "sess-A", "school-A")
	seedTermForSession(t, db, "term-A", "sess-A")
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "term-A")
	if err := db.Exec("INSERT INTO subjects (id, name) VALUES (?, ?)", "subj-1", "Mathematics").Error; err != nil {
		t.Fatalf("failed to seed subject: %v", err)
	}

	summary, err := s.GetQuestionContextSummary(context.Background(), "subj-1", "school-B", false)
	if err != nil {
		t.Fatalf("expected School B's caller to get its own (empty) summary, not an error: %v", err)
	}
	if summary.TotalQuestions != 0 {
		t.Fatalf("expected School B to see 0 questions for subj-1, got %d - cross-tenant leak", summary.TotalQuestions)
	}
}

func TestGetQuestionContextSummary_SameTenant_Allowed(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedAcademicSessionForSchool(t, db, "sess-A", "school-A")
	seedTermForSession(t, db, "term-A", "sess-A")
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "term-A")
	if err := db.Exec("INSERT INTO subjects (id, name) VALUES (?, ?)", "subj-1", "Mathematics").Error; err != nil {
		t.Fatalf("failed to seed subject: %v", err)
	}

	summary, err := s.GetQuestionContextSummary(context.Background(), "subj-1", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own summary, got error: %v", err)
	}
	if summary.TotalQuestions != 1 {
		t.Fatalf("expected 1 question for School A's own subject/term, got %d", summary.TotalQuestions)
	}
}

func TestGetQuestionsWithContext_CrossTenant_ClientSchoolIDIgnored(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "")
	seedGroupedQuestion(t, db, "q-B", "school-B", "subj-1", "", "")

	req := &dto.FilterQuestionsWithContextRequest{SchoolID: "school-A", SubjectID: "subj-1", Page: 1, Limit: 20}
	resp, err := s.GetQuestionsWithContext(context.Background(), req, "school-B", false)
	if err != nil {
		t.Fatalf("expected School B's caller to get its own results, not an error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected the client-supplied school_id=school-A to be ignored and only School B's 1 question returned, got %d", resp.Total)
	}
}

func TestGetQuestionsWithContext_PlatformOwner_HonorsClientSchoolID(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "", "")
	seedGroupedQuestion(t, db, "q-B", "school-B", "subj-1", "", "")

	req := &dto.FilterQuestionsWithContextRequest{SchoolID: "school-A", SubjectID: "subj-1", Page: 1, Limit: 20}
	resp, err := s.GetQuestionsWithContext(context.Background(), req, "", true)
	if err != nil {
		t.Fatalf("expected platform_owner to target School A explicitly, got error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected platform_owner's explicit school_id=school-A to return School A's 1 question, got %d", resp.Total)
	}
}

func TestGetQuestionsForExam_CrossTenant_Empty(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "term-A")

	questions, err := s.GetQuestionsForExam(context.Background(), "subj-1", "term-A", "sess-A", "school-B", false)
	if err != nil {
		t.Fatalf("expected School B's caller to get its own (empty) results, not an error: %v", err)
	}
	if len(questions) != 0 {
		t.Fatalf("expected School B to see 0 questions for School A's term/session, got %d - cross-tenant leak", len(questions))
	}
}

func TestGetQuestionsForExam_SameTenant_Allowed(t *testing.T) {
	db := setupGroupedTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedGroupedQuestion(t, db, "q-A", "school-A", "subj-1", "sess-A", "term-A")

	questions, err := s.GetQuestionsForExam(context.Background(), "subj-1", "term-A", "sess-A", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own term/session questions, got error: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question for School A's own term/session, got %d", len(questions))
	}
}
