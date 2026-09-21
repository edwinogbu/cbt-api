// Package service implements the offline-first sync protocol
// (POST /api/v1/sync/:type), closing the gap where the frontend's
// already-built sync engine (src/lib/storage/sync/{lanSync,cloudSync}.ts)
// called an endpoint that did not exist.
//
// sessionId/attemptId convention: the client's local Dexie ExamSession has
// no field mapping it to the server's ExamAttempt.ID. The convention this
// endpoint relies on is that SessionID IS the ExamAttempt.ID - i.e.
// whenever the client creates a local ExamSession, it must use the exact
// attemptId returned by POST /student/exams/start/:examId as the
// session's sessionId. This is not yet enforced anywhere in the frontend;
// flagged for the phase that wires the live exam-attempt page to
// src/lib/storage (Plan Phase 3 in the offline-first implementation plan).
//
// Grading authority: for type="answer" and type="submission" this service
// delegates to the existing, already-hardened internal/cbt/service
// ExamService.SaveAnswer/SubmitExam - the exact same functions the
// real-time /student/exams/answer and /student/exams/submit endpoints
// use - so there is never a second, divergent grading implementation
// (see rule #21 in the offline-first architecture document).
package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	cbtDto "cbt-api/internal/cbt/dto"
	cbtService "cbt-api/internal/cbt/service"
	"cbt-api/internal/models"
	"cbt-api/internal/sync/dto"
	"cbt-api/internal/sync/repository"

	"github.com/gin-gonic/gin/binding"
)

type SyncService struct {
	repo        *repository.SyncRepository
	examService *cbtService.ExamService
}

func NewSyncService(repo *repository.SyncRepository, examService *cbtService.ExamService) *SyncService {
	return &SyncService{repo: repo, examService: examService}
}

// Process handles one sync operation. studentID is the authenticated
// caller's student ID (from middleware.GetStudentID), never trusted from
// the request body.
func (s *SyncService) Process(ctx context.Context, studentID string, req *dto.SyncRequest) *dto.SyncResult {
	// 1. Idempotency check first, before any ownership/dispatch work - a
	// replayed request for an operation that has already completed should
	// always short-circuit here, even if e.g. the attempt has since been
	// deleted.
	existing, err := s.repo.FindByIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return rejected("INTERNAL_ERROR", "failed to check idempotency", true)
	}
	if existing != nil {
		var result interface{}
		if existing.ResultJSON != "" {
			_ = json.Unmarshal([]byte(existing.ResultJSON), &result)
		}
		return &dto.SyncResult{
			Outcome:         dto.OutcomeAlreadyProcessed,
			ServerVersion:   existing.ServerVersion,
			ServerSequence:  existing.ServerSequence,
			ServerTimestamp: existing.ProcessedAt.UTC().Format(time.RFC3339),
			Result:          result,
			HTTPStatus:      200,
		}
	}

	// 2. Ownership check - SessionID is treated as the ExamAttempt.ID (see
	// package doc). Never trust the client's claim that this session
	// belongs to them; always verify server-side (BE-03 pattern).
	ownerID, err := s.repo.FindAttemptOwner(req.SessionID)
	if err != nil {
		return rejected("INTERNAL_ERROR", "failed to verify attempt ownership", true)
	}
	if ownerID == "" {
		return rejected("NOT_FOUND", "attempt not found", false)
	}
	if ownerID != studentID {
		return rejected("FORBIDDEN", "attempt does not belong to this student", false)
	}

	// 3. Dispatch by type.
	var result interface{}
	var dispatchErr error
	switch req.Type {
	case "answer":
		result, dispatchErr = s.processAnswer(ctx, studentID, req)
	case "submission":
		result, dispatchErr = s.processSubmission(ctx, studentID, req)
	case "violation":
		result, dispatchErr = s.processViolation(req)
	case "event":
		result, dispatchErr = s.processEvent(studentID, req)
	default:
		return rejected("INVALID_TYPE", "unknown sync type", false)
	}

	if dispatchErr != nil {
		return classifyDispatchError(dispatchErr)
	}

	// 4. Record the operation so a retry of this exact idempotencyKey is
	// detected next time, instead of being reprocessed.
	resultJSON, _ := json.Marshal(result)
	op := &models.SyncOperation{
		IdempotencyKey: req.IdempotencyKey,
		OperationID:    req.OperationID,
		Type:           req.Type,
		RecordID:       req.RecordID,
		AttemptID:      req.SessionID,
		StudentID:      studentID,
		ServerVersion:  1,
		ResultJSON:     string(resultJSON),
	}
	if err := s.repo.RecordOperation(op); err != nil {
		// The operation itself already succeeded (e.g. the answer was
		// saved) - a failure to record the idempotency ledger entry must
		// not be reported to the client as a failure, or the client would
		// retry and could double-process. Worst case on a ledger-write
		// failure: a retried request reprocesses once more, which is safe
		// for answer/violation/event (idempotent by nature at the
		// row/upsert level) and guarded separately for submission by
		// ExamService.SubmitExam's own "exam already submitted" check.
	}

	return &dto.SyncResult{
		Outcome:         dto.OutcomeAccepted,
		ServerVersion:   1,
		ServerTimestamp: time.Now().UTC().Format(time.RFC3339),
		Result:          result,
		HTTPStatus:      200,
	}
}

func (s *SyncService) processAnswer(ctx context.Context, studentID string, req *dto.SyncRequest) (interface{}, error) {
	var payload dto.AnswerSyncPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return nil, errValidation("malformed answer payload")
	}
	if err := binding.Validator.ValidateStruct(&payload); err != nil {
		return nil, errValidation(err.Error())
	}

	saveReq := &cbtDto.SaveAnswerRequest{
		AttemptID:      req.SessionID,
		QuestionID:     payload.QuestionID,
		SelectedAnswer: payload.SelectedAnswer,
		TimeSpent:      payload.TimeSpent,
		IsMarked:       payload.IsMarked,
	}
	return s.examService.SaveAnswer(ctx, saveReq, studentID)
}

func (s *SyncService) processSubmission(ctx context.Context, studentID string, req *dto.SyncRequest) (interface{}, error) {
	submitReq := &cbtDto.SubmitExamRequest{
		AttemptID: req.SessionID,
	}
	return s.examService.SubmitExam(ctx, submitReq, studentID)
}

func (s *SyncService) processViolation(req *dto.SyncRequest) (interface{}, error) {
	var payload dto.ViolationSyncPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return nil, errValidation("malformed violation payload")
	}
	if err := binding.Validator.ValidateStruct(&payload); err != nil {
		return nil, errValidation(err.Error())
	}

	proctoringID, err := s.repo.FindProctoringSessionID(req.SessionID)
	if err != nil {
		return nil, err
	}
	if proctoringID == "" {
		return nil, errValidation("no proctoring session for this attempt")
	}

	severity := payload.Severity
	if severity == "" {
		severity = "warning"
	}

	violation := &models.ProctoringViolation{
		ProctoringID:  proctoringID,
		AttemptID:     req.SessionID,
		ViolationType: payload.ViolationType,
		Severity:      severity,
		Details:       payload.Details,
		Timestamp:     time.Now(),
	}
	if err := s.repo.SaveViolation(violation); err != nil {
		return nil, err
	}
	return map[string]string{"violationId": violation.ID}, nil
}

func (s *SyncService) processEvent(studentID string, req *dto.SyncRequest) (interface{}, error) {
	var payload dto.EventSyncPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return nil, errValidation("malformed event payload")
	}
	if err := binding.Validator.ValidateStruct(&payload); err != nil {
		return nil, errValidation(err.Error())
	}

	event := &models.ExamEventLog{
		AttemptID:       req.SessionID,
		StudentID:       studentID,
		EventType:       payload.EventType,
		ClientSequence:  payload.Sequence,
		PayloadJSON:     string(payload.Payload),
		ClientTimestamp: req.ClientTimestamp,
	}
	if err := s.repo.SaveEvent(event); err != nil {
		return nil, err
	}
	return map[string]string{"eventId": event.ID}, nil
}

// ---- error helpers ----

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }
func errValidation(msg string) error     { return &validationError{msg} }

func rejected(code, message string, retryable bool) *dto.SyncResult {
	return &dto.SyncResult{
		Outcome:      dto.OutcomeRejected,
		ErrorCode:    code,
		ErrorMessage: message,
		Retryable:    retryable,
		HTTPStatus:   httpStatusForCode(code),
	}
}

func httpStatusForCode(code string) int {
	switch code {
	case "NOT_FOUND":
		return 404
	case "FORBIDDEN":
		return 403
	case "VALIDATION_ERROR", "INVALID_TYPE":
		return 400
	default:
		return 500
	}
}

// classifyDispatchError maps errors from ExamService/repository calls to a
// REJECTED sync result. Mirrors the same status-mapping already used by
// the real-time exam handlers (internal/cbt/handler/exam_handler.go), so
// "exam already submitted" etc. behave identically whether the student was
// online (real-time endpoint) or is catching up after being offline (this
// endpoint).
func classifyDispatchError(err error) *dto.SyncResult {
	var vErr *validationError
	if errors.As(err, &vErr) {
		return rejected("VALIDATION_ERROR", vErr.msg, false)
	}
	switch err.Error() {
	case "attempt not found", "question not found":
		return rejected("NOT_FOUND", err.Error(), false)
	case "unauthorized":
		return rejected("FORBIDDEN", err.Error(), false)
	case "exam already submitted", "time limit exceeded":
		return rejected("CONFLICT_STATE", err.Error(), false)
	default:
		return rejected("INTERNAL_ERROR", err.Error(), true)
	}
}
