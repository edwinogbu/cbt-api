package service

import (
	"context"
	"errors"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// findOrInitSubjectResultSubmission mirrors
// ReportCardApprovalService's findOrInitApproval exactly - same
// "return a fresh, unsaved Version-0 row if none exists yet" shape,
// adapted to this entity's (class, subject, term, session) scope
// instead of (student, report-type, term).
func (s *ExamService) findOrInitSubjectResultSubmission(ctx context.Context, schoolID, classID, subjectID, termID, sessionID string) (models.SubjectResultSubmission, error) {
	var sub models.SubjectResultSubmission
	err := s.db.WithContext(ctx).
		Where("school_id = ? AND class_id = ? AND subject_id = ? AND term_id = ? AND session_id = ?", schoolID, classID, subjectID, termID, sessionID).
		First(&sub).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return models.SubjectResultSubmission{
			ID: models.GenerateID(), SchoolID: schoolID, ClassID: classID,
			SubjectID: subjectID, TermID: termID, SessionID: sessionID, Status: "draft", Version: 0,
		}, nil
	case err != nil:
		return models.SubjectResultSubmission{}, err
	default:
		return sub, nil
	}
}

func toSubjectResultSubmissionResponse(s *models.SubjectResultSubmission) *dto.SubjectResultSubmissionResponse {
	return &dto.SubjectResultSubmissionResponse{
		ID: s.ID, ClassID: s.ClassID, SubjectID: s.SubjectID, TermID: s.TermID, SessionID: s.SessionID,
		Status: s.Status, Version: s.Version, TeacherID: s.TeacherID, SubmittedAt: s.SubmittedAt,
		ReviewedBy: strDeref(s.ReviewedBy), ReviewedAt: s.ReviewedAt,
		ApprovedBy: strDeref(s.ApprovedBy), ApprovedAt: s.ApprovedAt, Notes: s.Notes,
	}
}

// GetSubjectResultSubmissionStatus reports the current approval-chain
// state for a scope, or "draft" with no row ID if nothing has been
// submitted yet - same "no row = not submitted" convention
// GetApprovalStatus already established for report cards.
func (s *ExamService) GetSubjectResultSubmissionStatus(ctx context.Context, schoolID string, classID, subjectID, termID, sessionID string) (*dto.SubjectResultSubmissionResponse, error) {
	sub, err := s.findOrInitSubjectResultSubmission(ctx, schoolID, classID, subjectID, termID, sessionID)
	if err != nil {
		return nil, err
	}
	return toSubjectResultSubmissionResponse(&sub), nil
}

// SubmitSubjectResults is a subject teacher's "I'm done entering
// marks for this class" action - moves draft/returned -> submitted.
// Does not require every cell to be filled (an incomplete submission
// is a real, visible state the HOD/Examination Officer/VP Academics
// broad sheets surface, per the spec's "missing marks"/"completeness"
// requirements), only that the caller has ordinary class-tenant
// access - the same boundary every other score-register action uses.
func (s *ExamService) SubmitSubjectResults(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.SubjectResultScopeRequest) (*dto.SubjectResultSubmissionResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	sub, err := s.findOrInitSubjectResultSubmission(ctx, schoolID, req.ClassID, req.SubjectID, req.TermID, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sub.IsLocked() {
		return nil, errors.New("these results have already been reviewed/approved - use a correction instead of re-submitting")
	}
	now := time.Now()
	sub.Status = "submitted"
	sub.Version++
	sub.TeacherID = actorID
	sub.SubmittedAt = &now
	sub.Notes = req.Notes
	if err := s.db.WithContext(ctx).Save(&sub).Error; err != nil {
		return nil, err
	}
	s.writeSubjectResultAudit(ctx, &models.SubjectResultAuditLog{
		SchoolID: schoolID, ActorID: actorID, Action: "submitted",
		ClassID: req.ClassID, SubjectID: req.SubjectID, TermID: req.TermID, SessionID: req.SessionID, Notes: req.Notes,
	})
	return toSubjectResultSubmissionResponse(&sub), nil
}

// ReviewSubjectResults is the HOD/Examination Officer's "I checked
// this - completeness and scoring look right" action - moves
// submitted -> reviewed. Per the spec, this review step is optional
// (a school may approve straight from "submitted"); this endpoint
// exists for schools that configure it, it is never required by
// ApproveSubjectResults below.
func (s *ExamService) ReviewSubjectResults(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.SubjectResultScopeRequest) (*dto.SubjectResultSubmissionResponse, error) {
	sub, err := s.findOrInitSubjectResultSubmission(ctx, schoolID, req.ClassID, req.SubjectID, req.TermID, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sub.Status != "submitted" {
		return nil, errors.New("only a submitted result set can be reviewed")
	}
	now := time.Now()
	sub.Status = "reviewed"
	sub.ReviewedBy = &actorID
	sub.ReviewedAt = &now
	if req.Notes != "" {
		sub.Notes = req.Notes
	}
	if err := s.db.WithContext(ctx).Save(&sub).Error; err != nil {
		return nil, err
	}
	s.writeSubjectResultAudit(ctx, &models.SubjectResultAuditLog{
		SchoolID: schoolID, ActorID: actorID, Action: "reviewed",
		ClassID: req.ClassID, SubjectID: req.SubjectID, TermID: req.TermID, SessionID: req.SessionID, Notes: req.Notes,
	})
	return toSubjectResultSubmissionResponse(&sub), nil
}

// ReturnSubjectResults sends a submitted (or reviewed) result set back
// to the subject teacher for correction - Version is left unchanged
// (returning doesn't consume a version, mirroring RejectReportCard's
// own doc comment on the same point), unlocking ordinary score-
// register saves again (IsLocked is false for "returned").
func (s *ExamService) ReturnSubjectResults(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.SubjectResultScopeRequest) (*dto.SubjectResultSubmissionResponse, error) {
	if req.Notes == "" {
		return nil, errors.New("notes (the reason for returning) is required")
	}
	sub, err := s.findOrInitSubjectResultSubmission(ctx, schoolID, req.ClassID, req.SubjectID, req.TermID, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sub.Status != "submitted" && sub.Status != "reviewed" {
		return nil, errors.New("only a submitted or reviewed result set can be returned")
	}
	sub.Status = "returned"
	sub.Notes = req.Notes
	if err := s.db.WithContext(ctx).Save(&sub).Error; err != nil {
		return nil, err
	}
	s.writeSubjectResultAudit(ctx, &models.SubjectResultAuditLog{
		SchoolID: schoolID, ActorID: actorID, Action: "returned",
		ClassID: req.ClassID, SubjectID: req.SubjectID, TermID: req.TermID, SessionID: req.SessionID, Notes: req.Notes,
	})
	return toSubjectResultSubmissionResponse(&sub), nil
}

// ApproveSubjectResults is the academic administrator's (admin/
// super_admin/vp_academics/platform_owner) final sign-off - moves
// submitted OR reviewed -> approved, locking ordinary score-register
// edits (SaveScoreRegister then requires a correction_reason). Runs
// from either status since the review step is optional per the spec.
func (s *ExamService) ApproveSubjectResults(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.SubjectResultScopeRequest) (*dto.SubjectResultSubmissionResponse, error) {
	sub, err := s.findOrInitSubjectResultSubmission(ctx, schoolID, req.ClassID, req.SubjectID, req.TermID, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sub.Status != "submitted" && sub.Status != "reviewed" {
		return nil, errors.New("only a submitted or reviewed result set can be approved")
	}
	now := time.Now()
	sub.Status = "approved"
	sub.ApprovedBy = &actorID
	sub.ApprovedAt = &now
	if req.Notes != "" {
		sub.Notes = req.Notes
	}
	if err := s.db.WithContext(ctx).Save(&sub).Error; err != nil {
		return nil, err
	}
	s.writeSubjectResultAudit(ctx, &models.SubjectResultAuditLog{
		SchoolID: schoolID, ActorID: actorID, Action: "approved",
		ClassID: req.ClassID, SubjectID: req.SubjectID, TermID: req.TermID, SessionID: req.SessionID, Notes: req.Notes,
	})
	return toSubjectResultSubmissionResponse(&sub), nil
}
