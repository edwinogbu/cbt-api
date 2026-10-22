package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// reportCardAuditContentHash computes the tamper-evidence hash for one
// models.ReportCardAuditLog row - see that field's own doc comment. Takes
// the row's fields directly (not the struct) so the same function
// computes the hash both at write time and at verify time without caring
// whether the row came from a fresh Create or a later SELECT.
func reportCardAuditContentHash(schoolID, actorID, action, reportType, studentID, classID, termID, sessionID, notes string, createdAt time.Time) string {
	h := sha256.New()
	h.Write([]byte(schoolID))
	h.Write([]byte{0})
	h.Write([]byte(actorID))
	h.Write([]byte{0})
	h.Write([]byte(action))
	h.Write([]byte{0})
	h.Write([]byte(reportType))
	h.Write([]byte{0})
	h.Write([]byte(studentID))
	h.Write([]byte{0})
	h.Write([]byte(classID))
	h.Write([]byte{0})
	h.Write([]byte(termID))
	h.Write([]byte{0})
	h.Write([]byte(sessionID))
	h.Write([]byte{0})
	h.Write([]byte(notes))
	h.Write([]byte{0})
	h.Write([]byte(createdAt.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))
}

// validReportCardTypes are the only report-card instance types the
// approval/batch workflow understands - the same three types
// internal/cbt/pdf generates (GenerateTermlyReportCard /
// GenerateMidTermProgressReport / GeneratePromotionExaminationReport).
var validReportCardTypes = map[string]bool{
	"termly": true, "midterm": true, "promotion": true,
}

// normalizeApprovalScope validates a ReportCardScope and normalizes
// TermID: "promotion" is session-scoped (no single term), so its
// TermID is always forced to "" regardless of what the caller sent -
// never left as whatever stray value happened to be posted, which
// would otherwise let the same promotion report be approved under
// more than one distinct models.ReportCardApproval row (its unique
// index is keyed on the literal TermID value).
func normalizeApprovalScope(scope *dto.ReportCardScope) error {
	if !validReportCardTypes[scope.ReportType] {
		return errors.New("report_type must be one of termly, midterm, promotion")
	}
	if scope.ReportType == "promotion" {
		scope.TermID = ""
	} else if scope.TermID == "" {
		return errors.New("term_id is required for this report type")
	}
	return nil
}

// buildReportCardSnapshot generates the live report-card payload for
// one student/scope and marshals it to JSON. This IS the generation
// step for an approval: a report that cannot even be built (e.g. a
// student with no results recorded at all yet) is refused rather than
// freezing an error as if it were an official approved copy.
func (s *ExamService) buildReportCardSnapshot(ctx context.Context, student *models.Student, scope dto.ReportCardScope) (string, error) {
	var (
		payload interface{}
		err     error
	)
	switch scope.ReportType {
	case "termly":
		payload, err = s.GetTermlyReportCardForStudent(ctx, student, scope.TermID, scope.SessionID)
	case "midterm":
		payload, err = s.GetMidTermProgressReport(ctx, student, scope.TermID, scope.SessionID)
	case "promotion":
		payload, err = s.GetPromotionReportCard(ctx, student, scope.SessionID)
	default:
		return "", errors.New("unsupported report type")
	}
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeReportCardAudit appends one entry to the school-scoped,
// append-only models.ReportCardAuditLog trail - fire-and-forget,
// mirroring internal/platform/service's own writeAudit: a logging
// failure must never fail (or roll back) the approval action it is
// recording.
func (s *ExamService) writeReportCardAudit(ctx context.Context, log *models.ReportCardAuditLog) {
	log.ID = models.GenerateID()
	log.CreatedAt = time.Now()
	log.ContentHash = reportCardAuditContentHash(
		log.SchoolID, log.ActorID, log.Action, log.ReportType,
		log.StudentID, log.ClassID, log.TermID, log.SessionID, log.Notes, log.CreatedAt,
	)
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		fmt.Printf("failed to write report-card audit log: %v\n", err)
	}
}

// VerifyReportCardAuditIntegrity recomputes every stored entry's
// ContentHash from its own columns and reports any row whose recomputed
// hash no longer matches what was stored at write time - evidence the row
// was altered directly (not through ApproveReportCard/RejectReportCard/
// BatchApproveReportCards, the only legitimate writers of this table)
// since it was created. A clean result does not prove no row was ever
// deleted outright (no per-row hash can detect that), only that every
// row currently present is exactly as it was written.
func (s *ExamService) VerifyReportCardAuditIntegrity(ctx context.Context, schoolID string) (*dto.AuditIntegrityReport, error) {
	var logs []models.ReportCardAuditLog
	if err := s.db.WithContext(ctx).Where("school_id = ?", schoolID).Find(&logs).Error; err != nil {
		return nil, err
	}

	report := &dto.AuditIntegrityReport{Checked: len(logs)}
	for _, l := range logs {
		expected := reportCardAuditContentHash(
			l.SchoolID, l.ActorID, l.Action, l.ReportType,
			l.StudentID, l.ClassID, l.TermID, l.SessionID, l.Notes, l.CreatedAt,
		)
		if expected != l.ContentHash {
			report.Issues = append(report.Issues, dto.AuditIntegrityIssue{
				ID: l.ID, Action: l.Action, StudentID: l.StudentID, CreatedAt: l.CreatedAt,
			})
		}
	}
	return report, nil
}

func approvalToResponse(a *models.ReportCardApproval) *dto.ReportCardApprovalResponse {
	resp := &dto.ReportCardApprovalResponse{
		StudentID: a.StudentID, ReportType: a.ReportType, TermID: a.TermID,
		SessionID: a.SessionID, Status: a.Status, Version: a.Version,
		Notes: a.Notes, ActorID: a.ActorID,
	}
	if !a.UpdatedAt.IsZero() {
		t := a.UpdatedAt
		resp.UpdatedAt = &t
	}
	return resp
}

// findOrInitApproval fetches the one models.ReportCardApproval row for
// a scope, or returns a new unsaved one (Version 0, so the first
// ApproveReportCard call bumps it to 1).
func (s *ExamService) findOrInitApproval(ctx context.Context, schoolID string, scope dto.ReportCardScope) (models.ReportCardApproval, error) {
	var approval models.ReportCardApproval
	err := s.db.WithContext(ctx).
		Where("student_id = ? AND report_type = ? AND term_id = ? AND session_id = ?",
			scope.StudentID, scope.ReportType, scope.TermID, scope.SessionID).
		First(&approval).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return models.ReportCardApproval{
			ID: models.GenerateID(), SchoolID: schoolID,
			StudentID: scope.StudentID, ReportType: scope.ReportType,
			TermID: scope.TermID, SessionID: scope.SessionID, Version: 0,
		}, nil
	case err != nil:
		return models.ReportCardApproval{}, err
	default:
		return approval, nil
	}
}

// ApproveReportCard approves one student's report card, freezing a
// snapshot of its current, live-computed figures as the official
// approved copy (models.ReportCardApproval.SnapshotJSON). Re-approving
// an already-approved scope bumps Version and replaces the snapshot -
// the prior approval is still recoverable from
// models.ReportCardAuditLog (GetApprovalHistory), even though the
// ReportCardApproval row itself only ever holds the latest state.
func (s *ExamService) ApproveReportCard(ctx context.Context, schoolID, actorID string, req *dto.ApproveReportCardRequest) (*dto.ReportCardApprovalResponse, error) {
	if err := normalizeApprovalScope(&req.ReportCardScope); err != nil {
		return nil, err
	}
	student, err := s.examRepo.GetStudentByID(ctx, req.StudentID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	if student.SchoolID != schoolID {
		return nil, errors.New("student does not belong to your school")
	}

	snapshot, err := s.buildReportCardSnapshot(ctx, student, req.ReportCardScope)
	if err != nil {
		return nil, fmt.Errorf("report card cannot be approved: %w", err)
	}

	approval, err := s.findOrInitApproval(ctx, schoolID, req.ReportCardScope)
	if err != nil {
		return nil, err
	}
	approval.Version++
	approval.Status = "approved"
	approval.SnapshotJSON = snapshot
	approval.Notes = req.Notes
	approval.ActorID = actorID
	if err := s.db.WithContext(ctx).Save(&approval).Error; err != nil {
		return nil, err
	}

	s.writeReportCardAudit(ctx, &models.ReportCardAuditLog{
		SchoolID: schoolID, ActorID: actorID, Action: "approved",
		ReportType: req.ReportType, StudentID: req.StudentID,
		TermID: req.TermID, SessionID: req.SessionID, Notes: req.Notes,
	})
	return approvalToResponse(&approval), nil
}

// RejectReportCard marks one student's report card as rejected - sent
// back for correction. Unlike ApproveReportCard it never builds or
// freezes a snapshot: a rejected report is, by definition, not the
// official copy, so there is nothing to freeze, and Version is left
// exactly as it was (rejecting does not consume a version).
func (s *ExamService) RejectReportCard(ctx context.Context, schoolID, actorID string, req *dto.RejectReportCardRequest) (*dto.ReportCardApprovalResponse, error) {
	if err := normalizeApprovalScope(&req.ReportCardScope); err != nil {
		return nil, err
	}
	if req.Notes == "" {
		return nil, errors.New("notes (the reason for rejection) is required")
	}
	student, err := s.examRepo.GetStudentByID(ctx, req.StudentID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	if student.SchoolID != schoolID {
		return nil, errors.New("student does not belong to your school")
	}

	approval, err := s.findOrInitApproval(ctx, schoolID, req.ReportCardScope)
	if err != nil {
		return nil, err
	}
	approval.Status = "rejected"
	approval.Notes = req.Notes
	approval.ActorID = actorID
	if err := s.db.WithContext(ctx).Save(&approval).Error; err != nil {
		return nil, err
	}

	s.writeReportCardAudit(ctx, &models.ReportCardAuditLog{
		SchoolID: schoolID, ActorID: actorID, Action: "rejected",
		ReportType: req.ReportType, StudentID: req.StudentID,
		TermID: req.TermID, SessionID: req.SessionID, Notes: req.Notes,
	})
	return approvalToResponse(&approval), nil
}

// GetApprovalStatus reports a student's report card's current sign-off
// state, or "not_submitted" when no models.ReportCardApproval row
// exists yet - never inferred as approved/rejected by default.
func (s *ExamService) GetApprovalStatus(ctx context.Context, schoolID string, isPlatformOwner bool, scope dto.ReportCardScope) (*dto.ReportCardApprovalResponse, error) {
	if err := normalizeApprovalScope(&scope); err != nil {
		return nil, err
	}
	student, err := s.examRepo.GetStudentByID(ctx, scope.StudentID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	if !isPlatformOwner && student.SchoolID != schoolID {
		return nil, errors.New("student not found")
	}

	var approval models.ReportCardApproval
	err = s.db.WithContext(ctx).
		Where("student_id = ? AND report_type = ? AND term_id = ? AND session_id = ?",
			scope.StudentID, scope.ReportType, scope.TermID, scope.SessionID).
		First(&approval).Error
	if err == gorm.ErrRecordNotFound {
		return &dto.ReportCardApprovalResponse{
			StudentID: scope.StudentID, ReportType: scope.ReportType,
			TermID: scope.TermID, SessionID: scope.SessionID, Status: "not_submitted",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return approvalToResponse(&approval), nil
}

// GetApprovalHistory returns the full historical trail
// (models.ReportCardAuditLog) for a school, optionally narrowed to one
// student and/or report type - most recent first. Unlike
// GetApprovalStatus (the current state), this can show every past
// approve/reject/batch action, including ones a later re-approval has
// since overwritten on the ReportCardApproval row itself.
func (s *ExamService) GetApprovalHistory(ctx context.Context, schoolID, studentID, reportType string) ([]dto.ReportCardAuditEntry, error) {
	var logs []models.ReportCardAuditLog
	q := s.db.WithContext(ctx).Where("school_id = ?", schoolID).Order("created_at DESC").Limit(200)
	if studentID != "" {
		q = q.Where("student_id = ?", studentID)
	}
	if reportType != "" {
		q = q.Where("report_type = ?", reportType)
	}
	if err := q.Find(&logs).Error; err != nil {
		return nil, err
	}
	entries := make([]dto.ReportCardAuditEntry, len(logs))
	for i, l := range logs {
		entries[i] = dto.ReportCardAuditEntry{
			ID: l.ID, Action: l.Action, ReportType: l.ReportType,
			StudentID: l.StudentID, ClassID: l.ClassID, TermID: l.TermID,
			SessionID: l.SessionID, ActorID: l.ActorID, Notes: l.Notes,
			CreatedAt: l.CreatedAt,
		}
	}
	return entries, nil
}

// BatchApproveReportCards approves every named (or, with no explicit
// StudentIDs, every currently active) student in one class for one
// report-card scope - never all-or-nothing: each student's approval is
// attempted independently via ApproveReportCard (the same tenant and
// buildability checks apply per student), and one student's failure
// (e.g. no results recorded yet) never blocks the rest of the class.
func (s *ExamService) BatchApproveReportCards(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.BatchApproveRequest) (*dto.BatchApproveResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	scope := dto.ReportCardScope{ReportType: req.ReportType, TermID: req.TermID, SessionID: req.SessionID}
	// StudentID is validated per-student inside the loop below -
	// normalizeApprovalScope only needs ReportType/TermID here to
	// resolve the real, normalized TermID once for the whole batch.
	scope.StudentID = "validated-per-student"
	if err := normalizeApprovalScope(&scope); err != nil {
		return nil, err
	}
	req.TermID = scope.TermID

	// An explicit StudentIDs list is attacker/caller-controlled in size
	// (unlike the "whole class roster" default below, which is bounded
	// by how many real students a class actually has) - cap it so one
	// request can't force an unbounded number of individual report-card
	// builds/snapshots/DB writes in a single call.
	const maxExplicitBatchSize = 1000
	if len(req.StudentIDs) > maxExplicitBatchSize {
		return nil, fmt.Errorf("too many student_ids in one batch (%d); max is %d", len(req.StudentIDs), maxExplicitBatchSize)
	}

	studentIDs := req.StudentIDs
	if len(studentIDs) == 0 {
		students, err := s.examRepo.GetStudentsByClass(ctx, req.ClassID)
		if err != nil {
			return nil, err
		}
		for _, st := range students {
			studentIDs = append(studentIDs, st.ID)
		}
	}

	resp := &dto.BatchApproveResponse{ApprovedStudentIDs: []string{}, Failed: []dto.BatchApprovalFailure{}}
	for _, studentID := range studentIDs {
		_, err := s.ApproveReportCard(ctx, schoolID, actorID, &dto.ApproveReportCardRequest{
			ReportCardScope: dto.ReportCardScope{
				StudentID: studentID, ReportType: req.ReportType,
				TermID: req.TermID, SessionID: req.SessionID,
			},
			Notes: req.Notes,
		})
		if err != nil {
			resp.Failed = append(resp.Failed, dto.BatchApprovalFailure{StudentID: studentID, Error: err.Error()})
			continue
		}
		resp.ApprovedStudentIDs = append(resp.ApprovedStudentIDs, studentID)
	}

	s.writeReportCardAudit(ctx, &models.ReportCardAuditLog{
		SchoolID: schoolID, ActorID: actorID, Action: "batch_approved",
		ReportType: req.ReportType, ClassID: req.ClassID,
		TermID: req.TermID, SessionID: req.SessionID, Notes: req.Notes,
	})
	return resp, nil
}
