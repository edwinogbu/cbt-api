package service

import (
	"context"
	"testing"
	"time"

	"cbt-api/internal/models"
	"cbt-api/internal/nodesync/dto"
	"cbt-api/internal/nodesync/repository"
	"cbt-api/pkg/utils"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// setupTestDB mirrors internal/sync/service/sync_service_test.go's pattern:
// an in-memory SQLite DB with hand-written, SQLite-compatible DDL for just
// the tables NodeSyncService.Push/AuthenticateNode touch (production uses
// AutoMigrate against Postgres, whose gen_random_uuid() default SQLite's
// DDL parser rejects).
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE school_node_credentials (
			id TEXT PRIMARY KEY, school_id TEXT NOT NULL, label TEXT,
			key_hash TEXT NOT NULL, last_used_at DATETIME, revoked_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE students (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, school_id TEXT NOT NULL,
			class_id TEXT, admission_no TEXT, date_of_birth DATETIME, gender TEXT,
			address TEXT, guardian_name TEXT, guardian_phone TEXT, guardian_email TEXT,
			is_active INTEGER DEFAULT 1,
			status TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE exam_attempts (
			id TEXT PRIMARY KEY, student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			start_time DATETIME, end_time DATETIME, score INTEGER, percentage REAL,
			status TEXT, device_info TEXT, ip_address TEXT, signature TEXT,
			synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE student_answers (
			id TEXT PRIMARY KEY, attempt_id TEXT NOT NULL, question_id TEXT NOT NULL,
			selected_answer TEXT, selected_option_keys TEXT, essay_response TEXT,
			is_correct INTEGER DEFAULT 0, is_marked INTEGER DEFAULT 0, time_spent INTEGER,
			synced_at DATETIME, synced_to_cloud_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE results (
			id TEXT PRIMARY KEY, student_id TEXT NOT NULL, exam_id TEXT NOT NULL,
			attempt_id TEXT NOT NULL, total_score INTEGER, percentage REAL, grade TEXT,
			remarks TEXT, passed INTEGER DEFAULT 0, fully_graded INTEGER DEFAULT 1, published_at DATETIME,
			synced_to_cloud_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE exam_event_logs (
			id TEXT PRIMARY KEY, attempt_id TEXT NOT NULL, student_id TEXT NOT NULL,
			event_type TEXT NOT NULL, client_sequence INTEGER, payload_json TEXT,
			client_timestamp TEXT, synced_to_cloud_at DATETIME, created_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create table: %v\nDDL: %s", err, stmt)
		}
	}
	return db
}

func seedStudent(t *testing.T, db *gorm.DB, studentID, schoolID string) {
	t.Helper()
	s := models.Student{ID: studentID, UserID: uuid.NewString(), SchoolID: schoolID, AdmissionNo: uuid.NewString(), IsActive: true}
	if err := db.Table("students").Create(&s).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

func seedCredential(t *testing.T, db *gorm.DB, schoolID, plaintextKey string) {
	t.Helper()
	hash, err := utils.HashPassword(plaintextKey)
	if err != nil {
		t.Fatalf("failed to hash credential: %v", err)
	}
	cred := models.SchoolNodeCredential{ID: uuid.NewString(), SchoolID: schoolID, Label: "test node", KeyHash: hash}
	if err := db.Table("school_node_credentials").Create(&cred).Error; err != nil {
		t.Fatalf("failed to seed credential: %v", err)
	}
}

// TestPush_RejectsRecordsForStudentsOutsideAuthenticatedSchool is the
// regression test for Push's core security boundary (see the function's
// own comment: "this is what stops one compromised or misconfigured node
// from writing data attributed to another school"). A node authenticated
// as School A must never be able to write a session/result/event for a
// student who belongs to School B, even if it knows that student's real
// ID - this is exactly the "cross-school data leakage" / tenant-isolation
// risk the project's own audit trail (config.go) flags as a known gap
// elsewhere; this at least locks down the one place that already has an
// explicit check so a future refactor can't silently remove it.
func TestPush_RejectsRecordsForStudentsOutsideAuthenticatedSchool(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	schoolB := uuid.NewString()
	studentInSchoolA := uuid.NewString()
	studentInSchoolB := uuid.NewString()
	seedStudent(t, db, studentInSchoolA, schoolA)
	seedStudent(t, db, studentInSchoolB, schoolB)

	req := &dto.PushRequest{
		Sessions: []dto.SessionPush{
			{ID: uuid.NewString(), StudentID: studentInSchoolA, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"},
			{ID: uuid.NewString(), StudentID: studentInSchoolB, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"},
		},
		Results: []dto.ResultPush{
			{ID: uuid.NewString(), StudentID: studentInSchoolA, ExamID: uuid.NewString(), AttemptID: uuid.NewString(), TotalScore: 10, Grade: "A"},
			{ID: uuid.NewString(), StudentID: studentInSchoolB, ExamID: uuid.NewString(), AttemptID: uuid.NewString(), TotalScore: 10, Grade: "A"},
		},
		Events: []dto.EventPush{
			{ID: uuid.NewString(), AttemptID: uuid.NewString(), StudentID: studentInSchoolA, EventType: "tab_switch"},
			{ID: uuid.NewString(), AttemptID: uuid.NewString(), StudentID: studentInSchoolB, EventType: "tab_switch"},
		},
	}

	resp, err := svc.Push(context.Background(), schoolA, req)
	if err != nil {
		t.Fatalf("Push returned an error: %v", err)
	}

	if resp.SessionsAccepted != 1 {
		t.Errorf("expected exactly 1 session accepted (School A's own), got %d", resp.SessionsAccepted)
	}
	if resp.ResultsAccepted != 1 {
		t.Errorf("expected exactly 1 result accepted (School A's own), got %d", resp.ResultsAccepted)
	}
	if resp.EventsAccepted != 1 {
		t.Errorf("expected exactly 1 event accepted (School A's own), got %d", resp.EventsAccepted)
	}

	if len(resp.Rejected) != 3 {
		t.Fatalf("expected exactly 3 rejected records (School B's session/result/event), got %d: %+v", len(resp.Rejected), resp.Rejected)
	}
	for _, r := range resp.Rejected {
		if r.Reason != "student does not belong to authenticated school" {
			t.Errorf("unexpected rejection reason for %s %s: %q", r.Type, r.ID, r.Reason)
		}
	}

	// Verify School B's session was genuinely never written, not just
	// reported as rejected.
	var count int64
	db.Table("exam_attempts").Where("student_id = ?", studentInSchoolB).Count(&count)
	if count != 0 {
		t.Errorf("School B's session was written to the DB despite being rejected: count=%d", count)
	}
}

// TestAuthenticateNode_RejectsWrongKeyAndAcceptsRight verifies the
// bcrypt-based node credential lookup actually distinguishes keys rather
// than accepting anything, and that a correct key resolves to the right
// school.
func TestAuthenticateNode_RejectsWrongKeyAndAcceptsRight(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	seedCredential(t, db, schoolA, "correct-plaintext-key")

	if _, _, err := svc.AuthenticateNode(context.Background(), "wrong-plaintext-key"); err == nil {
		t.Error("expected AuthenticateNode to reject a wrong key, got no error")
	}

	resolvedSchool, credID, err := svc.AuthenticateNode(context.Background(), "correct-plaintext-key")
	if err != nil {
		t.Fatalf("expected the correct key to authenticate, got error: %v", err)
	}
	if resolvedSchool != schoolA {
		t.Errorf("expected resolved school %q, got %q", schoolA, resolvedSchool)
	}
	if credID == "" {
		t.Error("expected a non-empty credential ID")
	}
}

// TestPush_UpsertIsIdempotent verifies that pushing the identical session
// twice (the real-world case: a node retries a push whose response it
// never received) does not create a duplicate row or error - this is the
// "safe retry and conflict handling during synchronisation" requirement.
func TestPush_UpsertIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	studentID := uuid.NewString()
	seedStudent(t, db, studentID, schoolA)

	session := dto.SessionPush{ID: uuid.NewString(), StudentID: studentID, ExamID: uuid.NewString(), StartTime: time.Now(), Status: "in_progress"}
	req := &dto.PushRequest{Sessions: []dto.SessionPush{session}}

	if _, err := svc.Push(context.Background(), schoolA, req); err != nil {
		t.Fatalf("first push failed: %v", err)
	}
	if _, err := svc.Push(context.Background(), schoolA, req); err != nil {
		t.Fatalf("retried push of the same session failed: %v", err)
	}

	var count int64
	db.Table("exam_attempts").Where("id = ?", session.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 row after retrying the same push, got %d (not idempotent)", count)
	}
}
