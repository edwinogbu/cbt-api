package dto

import "encoding/json"

// SyncRequest is the envelope for POST /api/v1/sync/:type, matching the
// contract already implemented (unused until now) by the frontend's
// offline-first sync engine (src/lib/storage/sync/{lanSync,cloudSync}.ts).
type SyncRequest struct {
	OperationID          string          `json:"operationId" binding:"required"`
	IdempotencyKey       string          `json:"idempotencyKey" binding:"required"`
	RecordID             string          `json:"recordId" binding:"required"`
	SessionID            string          `json:"sessionId" binding:"required,uuid"`
	Type                 string          `json:"type" binding:"required,oneof=answer submission violation event"`
	Payload              json.RawMessage `json:"payload" binding:"required"`
	ClientTimestamp      string          `json:"clientTimestamp"`
	BaseServerVersion    int             `json:"baseServerVersion"`
	LocalMutationVersion int             `json:"localMutationVersion"`
	ServerVersion        *int            `json:"serverVersion"`
}

// AnswerSyncPayload is the wire format for type="answer". Sent in
// PLAINTEXT: unlike the client's local Dexie AnswerRecord (which stores an
// AES-256-GCM encrypted blob for at-rest protection against another user of
// a shared device), the value the server needs to grade with must be
// readable server-side. The client must decrypt before building this
// payload - the server has no way to derive the client's local encryption
// key (by design; see the original security audit's note on
// getEncryptionSecret()).
type AnswerSyncPayload struct {
	QuestionID     string `json:"questionId" binding:"required,uuid"`
	SelectedAnswer string `json:"selectedAnswer" binding:"required,oneof=A B C D"`
	IsMarked       bool   `json:"isMarked"`
	TimeSpent      int    `json:"timeSpent"`
}

// SubmissionSyncPayload for type="submission" carries no fields beyond the
// envelope - the submission is identified entirely by SessionID
// (== the server's ExamAttempt.ID; see package doc in sync_service.go for
// the sessionId/attemptId convention this endpoint relies on).
type SubmissionSyncPayload struct{}

// ViolationSyncPayload is the wire format for type="violation".
type ViolationSyncPayload struct {
	ViolationType string `json:"violationType" binding:"required"`
	Severity      string `json:"severity"`
	Details       string `json:"details"`
}

// EventSyncPayload is the wire format for type="event" (audit/event-log
// entries from the client's exam-taking event stream).
type EventSyncPayload struct {
	EventType string          `json:"eventType" binding:"required"`
	Sequence  int             `json:"sequence"`
	Payload   json.RawMessage `json:"payload"`
}

// Outcome mirrors the frontend's SyncResponse discriminated union
// (src/lib/storage/sync/lanSync.ts) so the handler can map it directly to
// the HTTP response the sync engine already knows how to parse.
type Outcome string

const (
	OutcomeAccepted         Outcome = "ACCEPTED"
	OutcomeAlreadyProcessed Outcome = "ALREADY_PROCESSED"
	OutcomeConflict         Outcome = "CONFLICT"
	OutcomeRejected         Outcome = "REJECTED"
)

type SyncResult struct {
	Outcome         Outcome
	ServerVersion   int
	ServerSequence  *int
	ServerTimestamp string
	Result          interface{}
	ServerRecord    interface{} // only for CONFLICT
	ErrorCode       string      // only for REJECTED
	ErrorMessage    string      // only for REJECTED
	Retryable       bool        // only for REJECTED
	HTTPStatus      int
}

type SyncSuccessResponse struct {
	AlreadyProcessed bool        `json:"alreadyProcessed,omitempty"`
	ServerVersion    int         `json:"serverVersion"`
	ServerSequence   *int        `json:"serverSequence,omitempty"`
	ServerTimestamp  string      `json:"serverTimestamp"`
	Result           interface{} `json:"result,omitempty"`
}

type SyncConflictResponse struct {
	ServerVersion   int         `json:"serverVersion"`
	ServerRecord    interface{} `json:"serverRecord"`
	ServerTimestamp string      `json:"serverTimestamp"`
}

type SyncErrorResponse struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}
