package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// This file is the Class Teacher Broad Sheet's own actionable workflow
// (template B, page 4) - the class teacher's written class-wide
// summary, an optional HOD/Examination Officer review, and a "next
// action" control. See models.ClassBroadSheetReview's doc comment for
// why this is deliberately simpler than SubjectResultSubmission (no
// separate audit-log/content-hash trail).

// normalizeReviewTermID mirrors GetClassBroadSheet's own normalization
// (promotion reports have no single term) so a review row's scope key
// always matches the broad sheet it belongs to.
func normalizeReviewTermID(reportType, termID string) string {
	if reportType == "promotion" {
		return ""
	}
	return termID
}

func (s *ExamService) findClassBroadSheetReview(ctx context.Context, classID, reportType, termID, sessionID string) (*models.ClassBroadSheetReview, error) {
	var r models.ClassBroadSheetReview
	err := s.db.WithContext(ctx).
		Where("class_id = ? AND report_type = ? AND term_id = ? AND session_id = ?", classID, reportType, normalizeReviewTermID(reportType, termID), sessionID).
		First(&r).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func toClassBroadSheetReviewResponse(r *models.ClassBroadSheetReview, classID, reportType, termID, sessionID string) *dto.ClassBroadSheetReviewResponse {
	if r == nil {
		return &dto.ClassBroadSheetReviewResponse{
			ClassID: classID, ReportType: reportType, TermID: normalizeReviewTermID(reportType, termID), SessionID: sessionID,
			Status: "draft",
		}
	}
	resp := &dto.ClassBroadSheetReviewResponse{
		ID: r.ID, ClassID: r.ClassID, ReportType: r.ReportType, TermID: r.TermID, SessionID: r.SessionID,
		Status: r.Status, Summary: r.Summary, ClassTeacherID: r.ClassTeacherID,
		SubmittedAt: r.SubmittedAt, ReviewedAt: r.ReviewedAt, ReturnNotes: r.ReturnNotes, NextAction: r.NextAction,
	}
	if r.ReviewerID != nil {
		resp.ReviewerID = *r.ReviewerID
	}
	return resp
}

// GetClassBroadSheetReview is the read path GetClassBroadSheet folds
// into its own response - never returns an error for "no review yet",
// since a fresh draft state is the normal starting point, not a
// failure.
func (s *ExamService) GetClassBroadSheetReview(ctx context.Context, classID, reportType, termID, sessionID string) *dto.ClassBroadSheetReviewResponse {
	review, err := s.findClassBroadSheetReview(ctx, classID, reportType, termID, sessionID)
	if err != nil {
		return toClassBroadSheetReviewResponse(nil, classID, reportType, termID, sessionID)
	}
	return toClassBroadSheetReviewResponse(review, classID, reportType, termID, sessionID)
}

// SaveClassBroadSheetReviewSummary upserts the class teacher's draft
// summary/next-action without changing Status - a teacher may redraft
// as many times as needed before Submit.
func (s *ExamService) SaveClassBroadSheetReviewSummary(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.SaveClassBroadSheetReviewRequest) (*dto.ClassBroadSheetReviewResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	class, err := s.getClassByID(ctx, req.ClassID)
	if err != nil {
		return nil, err
	}
	termID := normalizeReviewTermID(req.ReportType, req.TermID)

	review, err := s.findClassBroadSheetReview(ctx, req.ClassID, req.ReportType, req.TermID, req.SessionID)
	switch {
	case err == nil:
		if review.Status == "approved" {
			return nil, errors.New("this class broad sheet has already been approved and can no longer be edited")
		}
		review.Summary, review.NextAction, review.ClassTeacherID = req.Summary, req.NextAction, actorID
		if saveErr := s.db.WithContext(ctx).Save(review).Error; saveErr != nil {
			return nil, saveErr
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		review = &models.ClassBroadSheetReview{
			SchoolID: class.SchoolID, ClassID: req.ClassID, ReportType: req.ReportType, TermID: termID, SessionID: req.SessionID,
			Status: "draft", Summary: req.Summary, NextAction: req.NextAction, ClassTeacherID: actorID,
		}
		if createErr := s.db.WithContext(ctx).Create(review).Error; createErr != nil {
			return nil, createErr
		}
	default:
		return nil, err
	}
	return toClassBroadSheetReviewResponse(review, req.ClassID, req.ReportType, req.TermID, req.SessionID), nil
}

// SubmitClassBroadSheetReview moves a draft (or a previously returned)
// review to "submitted" - requires a non-empty summary, since an
// empty submission would defeat the whole point of the class teacher's
// written overview.
func (s *ExamService) SubmitClassBroadSheetReview(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.ClassBroadSheetReviewActionRequest) (*dto.ClassBroadSheetReviewResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	review, err := s.findClassBroadSheetReview(ctx, req.ClassID, req.ReportType, req.TermID, req.SessionID)
	if err != nil {
		return nil, errors.New("save a summary before submitting")
	}
	if strings.TrimSpace(review.Summary) == "" {
		return nil, errors.New("a class teacher summary is required before submitting")
	}
	if review.Status == "approved" {
		return nil, errors.New("this class broad sheet has already been approved")
	}
	now := time.Now()
	review.Status, review.SubmittedAt = "submitted", &now
	if err := s.db.WithContext(ctx).Save(review).Error; err != nil {
		return nil, err
	}
	return toClassBroadSheetReviewResponse(review, req.ClassID, req.ReportType, req.TermID, req.SessionID), nil
}

// ApproveClassBroadSheetReview and ReturnClassBroadSheetReview are the
// reviewer's two possible actions on a submitted review - mirroring
// SubjectResultSubmission's review/return pair at the class-wide
// level.
func (s *ExamService) ApproveClassBroadSheetReview(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.ClassBroadSheetReviewActionRequest) (*dto.ClassBroadSheetReviewResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	review, err := s.findClassBroadSheetReview(ctx, req.ClassID, req.ReportType, req.TermID, req.SessionID)
	if err != nil || review.Status != "submitted" {
		return nil, errors.New("only a submitted class broad sheet review can be approved")
	}
	now := time.Now()
	review.Status, review.ReviewerID, review.ReviewedAt = "approved", &actorID, &now
	if err := s.db.WithContext(ctx).Save(review).Error; err != nil {
		return nil, err
	}
	return toClassBroadSheetReviewResponse(review, req.ClassID, req.ReportType, req.TermID, req.SessionID), nil
}

func (s *ExamService) ReturnClassBroadSheetReview(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.ClassBroadSheetReviewActionRequest) (*dto.ClassBroadSheetReviewResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Notes) == "" {
		return nil, errors.New("a return reason is required")
	}
	review, err := s.findClassBroadSheetReview(ctx, req.ClassID, req.ReportType, req.TermID, req.SessionID)
	if err != nil || review.Status != "submitted" {
		return nil, errors.New("only a submitted class broad sheet review can be returned")
	}
	now := time.Now()
	review.Status, review.ReviewerID, review.ReviewedAt, review.ReturnNotes = "returned", &actorID, &now, req.Notes
	if err := s.db.WithContext(ctx).Save(review).Error; err != nil {
		return nil, err
	}
	return toClassBroadSheetReviewResponse(review, req.ClassID, req.ReportType, req.TermID, req.SessionID), nil
}
