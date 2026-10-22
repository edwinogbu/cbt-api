package service

import (
	"context"
	"testing"
	"time"

	"cbt-api/internal/models"
	"cbt-api/internal/nodesync/dto"
	"cbt-api/internal/nodesync/repository"

	"github.com/google/uuid"
)

// TestPush_ReportCardApproval_RejectsOtherSchool is the report-card
// counterpart of TestPush_RejectsRecordsForStudentsOutsideAuthenticatedSchool:
// a report-card approval/audit log carries its own SchoolID (rather than
// being derived from a StudentID join, since a batch action can be
// class-wide with no single student), so Push must check THAT field
// directly against the authenticated node's school - never trust it.
func TestPush_ReportCardApproval_RejectsOtherSchool(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	schoolB := uuid.NewString()

	req := &dto.PushRequest{
		ReportCardApprovals: []dto.ReportCardApprovalPush{
			{ID: uuid.NewString(), SchoolID: schoolA, StudentID: uuid.NewString(), ReportType: "termly", TermID: "term-1", SessionID: "session-1", Status: "approved", Version: 1, ActorID: uuid.NewString()},
			{ID: uuid.NewString(), SchoolID: schoolB, StudentID: uuid.NewString(), ReportType: "termly", TermID: "term-1", SessionID: "session-1", Status: "approved", Version: 1, ActorID: uuid.NewString()},
		},
	}

	resp, err := svc.Push(context.Background(), schoolA, req)
	if err != nil {
		t.Fatalf("Push returned an error: %v", err)
	}
	if resp.ReportCardApprovalsAccepted != 1 {
		t.Errorf("ReportCardApprovalsAccepted = %d, want 1 (only School A's own)", resp.ReportCardApprovalsAccepted)
	}
	if len(resp.Rejected) != 1 || resp.Rejected[0].Reason != "school_id does not match authenticated node's school" {
		t.Fatalf("Rejected = %+v, want exactly 1 rejection for School B's record", resp.Rejected)
	}

	var count int64
	db.Model(&models.ReportCardApproval{}).Where("school_id = ?", schoolB).Count(&count)
	if count != 0 {
		t.Error("School B's report-card approval was written despite being rejected")
	}
}

// TestPush_ReportCardApproval_RejectsStaleVersion verifies the
// version-aware conflict rule: an incoming push whose Version is lower
// than what the cloud already has for the same (student, report_type,
// term, session) scope must be rejected, not silently overwrite a more
// recent approval - the report-card analogue of the exam-attempt
// status-rank check Push already applies to Sessions.
func TestPush_ReportCardApproval_RejectsStaleVersion(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)
	ctx := context.Background()

	schoolID := uuid.NewString()
	approvalID := uuid.NewString()
	studentID := uuid.NewString()
	actorID := uuid.NewString()

	// First push: version 1, accepted.
	first := &dto.PushRequest{ReportCardApprovals: []dto.ReportCardApprovalPush{
		{ID: approvalID, SchoolID: schoolID, StudentID: studentID, ReportType: "termly", TermID: "term-1", SessionID: "session-1", Status: "approved", Version: 2, ActorID: actorID, Notes: "second approval"},
	}}
	if _, err := svc.Push(ctx, schoolID, first); err != nil {
		t.Fatalf("first Push failed: %v", err)
	}

	// Second push: the SAME record ID but an OLDER version (simulating a
	// node that queued this write before it had seen the newer approval
	// made elsewhere, then came back online and tried to push its stale
	// local copy) - must be rejected, not silently accepted and overwrite.
	stale := &dto.PushRequest{ReportCardApprovals: []dto.ReportCardApprovalPush{
		{ID: approvalID, SchoolID: schoolID, StudentID: studentID, ReportType: "termly", TermID: "term-1", SessionID: "session-1", Status: "approved", Version: 1, ActorID: actorID, Notes: "stale first approval"},
	}}
	resp, err := svc.Push(ctx, schoolID, stale)
	if err != nil {
		t.Fatalf("second Push failed: %v", err)
	}
	if resp.ReportCardApprovalsAccepted != 0 {
		t.Errorf("ReportCardApprovalsAccepted = %d, want 0 (stale version must be rejected)", resp.ReportCardApprovalsAccepted)
	}
	if len(resp.Rejected) != 1 {
		t.Fatalf("Rejected = %+v, want exactly 1 (the stale push)", resp.Rejected)
	}

	var stored models.ReportCardApproval
	if err := db.Where("id = ?", approvalID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored approval: %v", err)
	}
	if stored.Version != 2 || stored.Notes != "second approval" {
		t.Errorf("stored approval = {version=%d, notes=%q}, want the version-2 record untouched by the stale push", stored.Version, stored.Notes)
	}
}

// TestPush_ReportCardAuditLog_InsertOnlyIdempotent verifies the audit
// log table's append-only contract survives a push: a retried push of
// the same already-accepted entry is a true no-op (never updates the
// stored row, which would risk invalidating its own ContentHash), and
// is still reported as accepted (not an error) so the node's outbox
// marks it synced and stops retrying it.
func TestPush_ReportCardAuditLog_InsertOnlyIdempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)
	ctx := context.Background()

	schoolID := uuid.NewString()
	entryID := uuid.NewString()
	createdAt := time.Now().Truncate(time.Second)

	push := func() *dto.PushResponse {
		req := &dto.PushRequest{ReportCardAuditLogs: []dto.ReportCardAuditLogPush{
			{ID: entryID, SchoolID: schoolID, ActorID: uuid.NewString(), Action: "approved",
				ReportType: "termly", StudentID: uuid.NewString(), SessionID: "session-1",
				ContentHash: "abc123", CreatedAt: createdAt},
		}}
		resp, err := svc.Push(ctx, schoolID, req)
		if err != nil {
			t.Fatalf("Push failed: %v", err)
		}
		return resp
	}

	first := push()
	if first.ReportCardAuditLogsAccepted != 1 {
		t.Fatalf("first push: ReportCardAuditLogsAccepted = %d, want 1", first.ReportCardAuditLogsAccepted)
	}
	second := push()
	if second.ReportCardAuditLogsAccepted != 1 {
		t.Errorf("retried push: ReportCardAuditLogsAccepted = %d, want 1 (idempotent no-op still counts as accepted)", second.ReportCardAuditLogsAccepted)
	}

	var count int64
	db.Model(&models.ReportCardAuditLog{}).Where("id = ?", entryID).Count(&count)
	if count != 1 {
		t.Errorf("stored row count = %d, want exactly 1 (no duplicate from the retried push)", count)
	}
}

func TestPush_AttendanceRecord_RejectsOtherSchool(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)

	schoolA := uuid.NewString()
	schoolB := uuid.NewString()

	req := &dto.PushRequest{
		AttendanceRecords: []dto.AttendanceRecordPush{
			{ID: uuid.NewString(), SchoolID: schoolA, StudentID: uuid.NewString(), ClassID: uuid.NewString(), TermID: uuid.NewString(), SessionID: uuid.NewString(), Date: time.Now(), Status: "present", MarkedBy: uuid.NewString()},
			{ID: uuid.NewString(), SchoolID: schoolB, StudentID: uuid.NewString(), ClassID: uuid.NewString(), TermID: uuid.NewString(), SessionID: uuid.NewString(), Date: time.Now(), Status: "present", MarkedBy: uuid.NewString()},
		},
	}

	resp, err := svc.Push(context.Background(), schoolA, req)
	if err != nil {
		t.Fatalf("Push returned an error: %v", err)
	}
	if resp.AttendanceRecordsAccepted != 1 {
		t.Errorf("AttendanceRecordsAccepted = %d, want 1 (only School A's own)", resp.AttendanceRecordsAccepted)
	}
	if len(resp.Rejected) != 1 || resp.Rejected[0].Reason != "school_id does not match authenticated node's school" {
		t.Fatalf("Rejected = %+v, want exactly 1 rejection for School B's record", resp.Rejected)
	}

	var count int64
	db.Model(&models.AttendanceRecord{}).Where("school_id = ?", schoolB).Count(&count)
	if count != 0 {
		t.Error("School B's attendance record was written despite being rejected")
	}
}

// TestPush_AttendanceRecord_LaterPushCorrectsEarlierMark verifies
// attendance's deliberately different conflict rule from Session/
// Result: unlike those, there is no "more final" state to protect, so a
// later push for the same (student, term, date) is accepted as a
// correction and replaces the stored status, never rejected as a
// conflict.
func TestPush_AttendanceRecord_LaterPushCorrectsEarlierMark(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNodeSyncRepository(db)
	svc := NewNodeSyncService(repo)
	ctx := context.Background()

	schoolID := uuid.NewString()
	recordID := uuid.NewString()
	studentID := uuid.NewString()
	termID := uuid.NewString()
	date := time.Now().Truncate(24 * time.Hour)

	push := func(status string) *dto.PushResponse {
		req := &dto.PushRequest{AttendanceRecords: []dto.AttendanceRecordPush{
			{ID: recordID, SchoolID: schoolID, StudentID: studentID, ClassID: uuid.NewString(),
				TermID: termID, SessionID: uuid.NewString(), Date: date, Status: status, MarkedBy: uuid.NewString()},
		}}
		resp, err := svc.Push(ctx, schoolID, req)
		if err != nil {
			t.Fatalf("Push failed: %v", err)
		}
		return resp
	}

	if resp := push("absent"); resp.AttendanceRecordsAccepted != 1 {
		t.Fatalf("first push: AttendanceRecordsAccepted = %d, want 1", resp.AttendanceRecordsAccepted)
	}
	if resp := push("present"); resp.AttendanceRecordsAccepted != 1 {
		t.Fatalf("correction push: AttendanceRecordsAccepted = %d, want 1", resp.AttendanceRecordsAccepted)
	}

	var stored models.AttendanceRecord
	if err := db.Where("id = ?", recordID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored record: %v", err)
	}
	if stored.Status != "present" {
		t.Errorf("stored status = %q, want present (the correction must have replaced the original absent mark)", stored.Status)
	}
}
