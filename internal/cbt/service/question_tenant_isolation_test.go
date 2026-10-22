package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Phase 2: question-bank tenant isolation ----
//
// questionEffectiveSchoolID and questionAuthorizeSchoolOwnership are the
// two helpers every fixed QuestionService method (CreateQuestion,
// GetQuestion, UpdateQuestion, DeleteQuestion, BulkCreateQuestionsFromJSON,
// BulkUploadFromFile, PreviewBulkUpload, GenerateQuestionsWithAI,
// ExtractQuestionsFromText, GenerateGroundedQuestions) now goes through.
// Before this fix, authorizeQuestionWrite trusted a client-supplied
// school_id outright for admin/super_admin (see its own doc comment),
// and GetQuestion/UpdateQuestion/DeleteQuestion had no school check of
// any kind.

func TestQuestionEffectiveSchoolID_NonPlatformOwner_IgnoresClientValue(t *testing.T) {
	// The actual vulnerability: an admin in school-A submitting
	// school_id=school-B in the request body must never end up writing
	// under school-B. The resolved value must always be the admin's own
	// ctx school, regardless of what the client sent.
	got := questionEffectiveSchoolID("school-B", "school-A", false)
	if got != "school-A" {
		t.Fatalf("expected client-supplied school_id to be ignored for a non-platform-owner, got %q", got)
	}
}

func TestQuestionEffectiveSchoolID_PlatformOwner_HonorsClientValue(t *testing.T) {
	got := questionEffectiveSchoolID("school-B", "", true)
	if got != "school-B" {
		t.Fatalf("expected platform_owner's explicit target school to be honored, got %q", got)
	}
}

func TestQuestionAuthorizeSchoolOwnership_CrossTenant_Rejected(t *testing.T) {
	if err := questionAuthorizeSchoolOwnership("school-A", "school-B", false); err == nil {
		t.Fatal("expected a record in school-A to be rejected for a school-B caller, got nil error")
	}
}

func TestQuestionAuthorizeSchoolOwnership_SameTenant_Allowed(t *testing.T) {
	if err := questionAuthorizeSchoolOwnership("school-A", "school-A", false); err != nil {
		t.Fatalf("expected same-school access to be allowed, got error: %v", err)
	}
}

func TestQuestionAuthorizeSchoolOwnership_PlatformOwner_Bypasses(t *testing.T) {
	if err := questionAuthorizeSchoolOwnership("school-A", "", true); err != nil {
		t.Fatalf("expected platform_owner to bypass the tenant check, got error: %v", err)
	}
}

func TestQuestionAuthorizeSchoolOwnership_EmptySchoolID_NonPlatformOwner_Rejected(t *testing.T) {
	// A non-platform-owner caller with no school_id of their own (e.g. a
	// legacy account seeded with none) must never be treated as
	// "unscoped" - "" must not coincidentally match a real record.
	if err := questionAuthorizeSchoolOwnership("school-A", "", false); err == nil {
		t.Fatal("expected an empty school_id on a non-platform-owner caller to be rejected, got no error")
	}
}

// setupQuestionTenantTestDB is a minimal SQLite fixture for GetQuestion/
// DeleteQuestion: just question_bank plus the three tables
// QuestionRepository.Delete cascades into (none of which are soft-delete
// models, so plain DELETE works against empty tables).
func setupQuestionTenantTestDB(t *testing.T) *gorm.DB {
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
		`CREATE TABLE question_tag_mappings (id TEXT PRIMARY KEY, question_id TEXT, tag_id TEXT, created_at DATETIME)`,
		`CREATE TABLE exam_questions (id TEXT PRIMARY KEY, exam_id TEXT, question_id TEXT, sort_order INTEGER, created_at DATETIME)`,
		`CREATE TABLE question_bank_attachments (id TEXT PRIMARY KEY, question_id TEXT, file_name TEXT, file_type TEXT, file_url TEXT, file_size INTEGER, created_at DATETIME)`,
		`CREATE TABLE subjects (id TEXT PRIMARY KEY, name TEXT, code TEXT, description TEXT, is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func seedQuestionForSchool(t *testing.T, db *gorm.DB, id, schoolID, createdBy string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO question_bank (id, school_id, question_text, marks, created_by, status) VALUES (?, ?, ?, ?, ?, ?)",
		id, schoolID, "What is 2+2?", 1, createdBy, "draft",
	).Error; err != nil {
		t.Fatalf("failed to seed question: %v", err)
	}
}

func newTenantTestQuestionService(db *gorm.DB) *QuestionService {
	qRepo := repository.NewQuestionRepository(db)
	subRepo := repository.NewSubjectRepository(db)
	return NewQuestionService(qRepo, subRepo, db, nil, nil)
}

func TestGetQuestion_CrossTenant_Rejected(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedQuestionForSchool(t, db, "q-1", "school-A", "teacher-1")

	if _, err := s.GetQuestion(context.Background(), "q-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected reading School A's question, got no error")
	}

	resp, err := s.GetQuestion(context.Background(), "q-1", "school-A", false)
	if err != nil {
		t.Fatalf("expected School A to read its own question, got error: %v", err)
	}
	if resp.ID != "q-1" {
		t.Fatalf("expected question q-1, got %s", resp.ID)
	}
}

func TestGetQuestion_PlatformOwner_Bypasses(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedQuestionForSchool(t, db, "q-1", "school-A", "teacher-1")

	if _, err := s.GetQuestion(context.Background(), "q-1", "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's question, got error: %v", err)
	}
}

func TestDeleteQuestion_CrossTenant_Rejected(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedQuestionForSchool(t, db, "q-1", "school-A", "teacher-1")

	if err := s.DeleteQuestion(context.Background(), "q-1", "teacher-1", "school-B", false); err == nil {
		t.Fatal("expected School B's caller to be rejected deleting School A's question, got no error")
	}

	var count int64
	db.Table("question_bank").Where("id = ? AND deleted_at IS NULL", "q-1").Count(&count)
	if count != 1 {
		t.Fatal("question was deleted by a cross-tenant delete that should have been rejected")
	}
}

func TestDeleteQuestion_SameTenant_OwnCreator_Allowed(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedQuestionForSchool(t, db, "q-1", "school-A", "teacher-1")

	if err := s.DeleteQuestion(context.Background(), "q-1", "teacher-1", "school-A", false); err != nil {
		t.Fatalf("expected the question's own creator in its own school to delete it, got error: %v", err)
	}
}

// ---- BulkDelete/BulkUpdateStatus: before this fix, neither took a
// school or ownership parameter at all - any authenticated teacher/admin
// could bulk-delete or bulk-update the status of ANY question in ANY
// school just by supplying its UUID, with zero check. Mirrors
// DeleteQuestion/UpdateQuestion's per-question checks, applied to each
// ID in the batch; IDs that fail are silently dropped (same policy as
// BulkDeleteClasses) rather than failing the whole batch.

func TestBulkDelete_CrossTenant_NotDeleted(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedQuestionForSchool(t, db, "q-a", "school-A", "teacher-a")
	seedQuestionForSchool(t, db, "q-b", "school-B", "teacher-b")

	req := &dto.BulkDeleteRequest{QuestionIDs: []string{"q-a", "q-b"}}
	deleted, err := s.BulkDelete(context.Background(), req, "teacher-a", "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected exactly 1 question deleted (the caller's own), got %d", deleted)
	}

	var countA, countB int64
	db.Table("question_bank").Where("id = ? AND deleted_at IS NULL", "q-a").Count(&countA)
	db.Table("question_bank").Where("id = ? AND deleted_at IS NULL", "q-b").Count(&countB)
	if countA != 0 {
		t.Fatal("expected School A's own question to be deleted")
	}
	if countB != 1 {
		t.Fatal("School B's question was deleted by a cross-tenant bulk-delete request - should have been silently dropped")
	}
}

func TestBulkDelete_PlatformOwner_BypassesSchoolCheck(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	// checkQuestionOwnership (creator == caller) applies unconditionally
	// to every role, same as the pre-existing single-item DeleteQuestion -
	// this test isolates the school-ownership gate platform_owner is
	// meant to bypass by making "owner-1" the creator of both, across
	// two different schools.
	seedQuestionForSchool(t, db, "q-a", "school-A", "owner-1")
	seedQuestionForSchool(t, db, "q-b", "school-B", "owner-1")

	req := &dto.BulkDeleteRequest{QuestionIDs: []string{"q-a", "q-b"}}
	deleted, err := s.BulkDelete(context.Background(), req, "owner-1", "", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected platform_owner to delete both questions across schools, got %d", deleted)
	}
}

func TestBulkUpdateStatus_CrossTenant_NotUpdated(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedQuestionForSchool(t, db, "q-a", "school-A", "teacher-a")
	seedQuestionForSchool(t, db, "q-b", "school-B", "teacher-b")

	updated, err := s.BulkUpdateStatus(context.Background(), []string{"q-a", "q-b"}, "published", "teacher-a", "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected exactly 1 question updated (the caller's own), got %d", updated)
	}

	var statusA, statusB string
	db.Table("question_bank").Select("status").Where("id = ?", "q-a").Scan(&statusA)
	db.Table("question_bank").Select("status").Where("id = ?", "q-b").Scan(&statusB)
	if statusA != "published" {
		t.Fatalf("expected School A's own question to be updated, got status %q", statusA)
	}
	if statusB != "draft" {
		t.Fatalf("School B's question status was changed by a cross-tenant bulk-status request (got %q) - should have been silently dropped", statusB)
	}
}

func TestBulkUpdateStatus_PlatformOwner_BypassesSchoolCheck(t *testing.T) {
	db := setupQuestionTenantTestDB(t)
	s := newTenantTestQuestionService(db)
	seedQuestionForSchool(t, db, "q-a", "school-A", "owner-1")
	seedQuestionForSchool(t, db, "q-b", "school-B", "owner-1")

	updated, err := s.BulkUpdateStatus(context.Background(), []string{"q-a", "q-b"}, "published", "owner-1", "", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if updated != 2 {
		t.Fatalf("expected platform_owner to update both questions across schools, got %d", updated)
	}
}
