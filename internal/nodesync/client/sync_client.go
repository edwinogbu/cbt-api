// Package client is the School-Node-side half of nodesync: a background
// loop that pushes this node's accumulated sessions/answers/results/events
// up to the cloud instance and pulls down students/exams/questions
// changed since its last successful pull. It runs inside the very same
// cbt-api binary as the cloud instance and the HTTP handlers in
// internal/nodesync/handler - "same binary, two roles" - activated only
// when CLOUD_SYNC_ENABLED=true (see cmd/server/main.go).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cbt-api/internal/models"
	"cbt-api/internal/nodesync/dto"
	"cbt-api/internal/nodesync/repository"

	"go.uber.org/zap"
)

type SyncClient struct {
	repo         *repository.NodeSyncRepository
	cloudBaseURL string
	apiKey       string
	httpClient   *http.Client
}

func NewSyncClient(repo *repository.NodeSyncRepository, cloudBaseURL, apiKey string) *SyncClient {
	return &SyncClient{
		repo:         repo,
		cloudBaseURL: cloudBaseURL,
		apiKey:       apiKey,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// RunSyncLoop pushes then pulls on every tick until ctx is cancelled. A
// failed tick (cloud unreachable, auth rejected, etc.) is logged and
// simply retried on the next tick - this node keeps accumulating locally
// regardless (rule #1: local write is never blocked on the network), so a
// sync failure here delays reconciliation, never data loss.
func (c *SyncClient) RunSyncLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runOnce(ctx)
		}
	}
}

func (c *SyncClient) runOnce(ctx context.Context) {
	logger := zap.L().With(zap.String("component", "nodesync.client"))

	if err := c.push(ctx); err != nil {
		logger.Warn("push to cloud failed, will retry next tick", zap.Error(err))
	}
	if err := c.pull(ctx); err != nil {
		logger.Warn("pull from cloud failed, will retry next tick", zap.Error(err))
	}
}

func (c *SyncClient) push(ctx context.Context) error {
	sessions, err := c.repo.UnsyncedSessions(ctx)
	if err != nil {
		return fmt.Errorf("load unsynced sessions: %w", err)
	}
	answers, err := c.repo.UnsyncedAnswers(ctx)
	if err != nil {
		return fmt.Errorf("load unsynced answers: %w", err)
	}
	results, err := c.repo.UnsyncedResults(ctx)
	if err != nil {
		return fmt.Errorf("load unsynced results: %w", err)
	}
	events, err := c.repo.UnsyncedEvents(ctx)
	if err != nil {
		return fmt.Errorf("load unsynced events: %w", err)
	}

	if len(sessions) == 0 && len(answers) == 0 && len(results) == 0 && len(events) == 0 {
		return nil
	}

	req := dto.PushRequest{}
	for _, a := range sessions {
		req.Sessions = append(req.Sessions, dto.SessionPush{
			ID: a.ID, StudentID: a.StudentID, ExamID: a.ExamID, StartTime: a.StartTime,
			EndTime: a.EndTime, Score: a.Score, Percentage: a.Percentage, Status: a.Status, IPAddress: a.IPAddress,
		})
	}
	for _, a := range answers {
		req.Answers = append(req.Answers, dto.AnswerPush{
			ID: a.ID, AttemptID: a.AttemptID, QuestionID: a.QuestionID, SelectedAnswer: a.SelectedAnswer,
			IsCorrect: a.IsCorrect, IsMarked: a.IsMarked, TimeSpent: a.TimeSpent,
		})
	}
	for _, r := range results {
		req.Results = append(req.Results, dto.ResultPush{
			ID: r.ID, StudentID: r.StudentID, ExamID: r.ExamID, AttemptID: r.AttemptID,
			TotalScore: r.TotalScore, Percentage: r.Percentage, Grade: r.Grade,
			Remarks: r.Remarks, Passed: r.Passed, PublishedAt: r.PublishedAt,
		})
	}
	for _, e := range events {
		req.Events = append(req.Events, dto.EventPush{
			ID: e.ID, AttemptID: e.AttemptID, StudentID: e.StudentID, EventType: e.EventType,
			ClientSequence: e.ClientSequence, PayloadJSON: e.PayloadJSON, ClientTimestamp: e.ClientTimestamp,
		})
	}

	var pushResp struct {
		Data dto.PushResponse `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/nodesync/push", req, &pushResp); err != nil {
		return err
	}

	rejected := make(map[string]bool, len(pushResp.Data.Rejected))
	for _, r := range pushResp.Data.Rejected {
		rejected[r.ID] = true
	}

	synced := func(all []string) []string {
		out := make([]string, 0, len(all))
		for _, id := range all {
			if !rejected[id] {
				out = append(out, id)
			}
		}
		return out
	}

	sessionIDs := make([]string, len(sessions))
	for i, a := range sessions {
		sessionIDs[i] = a.ID
	}
	answerIDs := make([]string, len(answers))
	for i, a := range answers {
		answerIDs[i] = a.ID
	}
	resultIDs := make([]string, len(results))
	for i, r := range results {
		resultIDs[i] = r.ID
	}
	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.ID
	}
	if err := c.repo.MarkSessionsSynced(ctx, synced(sessionIDs)); err != nil {
		return err
	}
	if err := c.repo.MarkAnswersSynced(ctx, synced(answerIDs)); err != nil {
		return err
	}
	if err := c.repo.MarkResultsSynced(ctx, synced(resultIDs)); err != nil {
		return err
	}
	if err := c.repo.MarkEventsSynced(ctx, synced(eventIDs)); err != nil {
		return err
	}

	zap.L().Info("nodesync push complete",
		zap.Int("sessions_accepted", pushResp.Data.SessionsAccepted),
		zap.Int("answers_accepted", pushResp.Data.AnswersAccepted),
		zap.Int("results_accepted", pushResp.Data.ResultsAccepted),
		zap.Int("events_accepted", pushResp.Data.EventsAccepted),
		zap.Int("rejected", len(pushResp.Data.Rejected)),
	)
	return nil
}

func (c *SyncClient) pull(ctx context.Context) error {
	cursor, err := c.repo.GetPullCursor(ctx)
	if err != nil {
		return fmt.Errorf("load pull cursor: %w", err)
	}

	path := "/api/v1/nodesync/pull"
	if !cursor.IsZero() {
		path += "?since=" + cursor.Format(time.RFC3339)
	}

	var pullResp struct {
		Data dto.PullResponse `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &pullResp); err != nil {
		return err
	}

	// Apply order: ClassLevels/ClassArms -> Classes -> Users -> Students,
	// matching the FK dependency chain (classes.class_level_id/
	// class_arm_id, students.class_id/user_id).
	for _, cl := range pullResp.Data.ClassLevels {
		classLevel := &models.ClassLevel{
			ID: cl.ID, SchoolID: cl.SchoolID, Name: cl.Name, LevelNumber: cl.LevelNumber,
			Category: cl.Category, SortOrder: cl.SortOrder, IsActive: cl.IsActive,
		}
		if err := c.repo.ApplyPulledClassLevel(ctx, classLevel); err != nil {
			zap.L().Warn("apply pulled class level failed", zap.String("id", cl.ID), zap.Error(err))
		}
	}
	for _, ca := range pullResp.Data.ClassArms {
		classArm := &models.ClassArm{
			ID: ca.ID, SchoolID: ca.SchoolID, Name: ca.Name, ArmCode: ca.ArmCode,
			Capacity: ca.Capacity, SortOrder: ca.SortOrder, IsActive: ca.IsActive,
		}
		if err := c.repo.ApplyPulledClassArm(ctx, classArm); err != nil {
			zap.L().Warn("apply pulled class arm failed", zap.String("id", ca.ID), zap.Error(err))
		}
	}
	for _, cp := range pullResp.Data.Classes {
		class := &models.Class{
			ID: cp.ID, SchoolID: cp.SchoolID, SessionID: cp.SessionID,
			ClassLevelID: cp.ClassLevelID, ClassArmID: cp.ClassArmID,
			ClassCode: cp.ClassCode, IsActive: cp.IsActive,
		}
		if err := c.repo.ApplyPulledClass(ctx, class); err != nil {
			zap.L().Warn("apply pulled class failed", zap.String("id", cp.ID), zap.Error(err))
		}
	}

	for _, u := range pullResp.Data.Users {
		user := &models.User{
			ID: u.ID, Username: u.Username, Email: u.Email, Password: u.PasswordHash,
			FirstName: u.FirstName, LastName: u.LastName, PhoneNumber: u.PhoneNumber,
			Role: models.UserRole(u.Role), Status: models.UserStatus(u.Status),
			EmailVerified: u.EmailVerified, IsActive: u.IsActive, SchoolID: u.SchoolID,
		}
		if err := c.repo.ApplyPulledUser(ctx, user); err != nil {
			zap.L().Warn("apply pulled user failed", zap.String("id", u.ID), zap.Error(err))
		}
	}

	for _, s := range pullResp.Data.Students {
		student := &models.Student{
			ID: s.ID, UserID: s.UserID, SchoolID: s.SchoolID, ClassID: s.ClassID, AdmissionNo: s.AdmissionNo,
			IsActive: s.IsActive, Status: s.Status,
		}
		if err := c.repo.ApplyPulledStudent(ctx, student); err != nil {
			zap.L().Warn("apply pulled student failed", zap.String("id", s.ID), zap.Error(err))
		}
	}

	for _, e := range pullResp.Data.Exams {
		exam := &models.Exam{
			ID: e.ID, Title: e.Title, SubjectID: e.SubjectID, ClassID: e.ClassID,
			SessionID: e.SessionID, TermID: e.TermID,
			ExamType: e.ExamType, Status: e.Status, DurationMinutes: e.DurationMinutes,
			TotalMarks: e.TotalMarks, PassMark: e.PassMark, Instructions: e.Instructions,
			StartTime: e.StartTime, EndTime: e.EndTime,
			ShuffleQuestions: e.ShuffleQuestions, ShuffleOptions: e.ShuffleOptions,
			IsActive: e.IsActive,
		}
		if err := c.repo.ApplyPulledExam(ctx, exam); err != nil {
			zap.L().Warn("apply pulled exam failed", zap.String("id", e.ID), zap.Error(err))
			continue
		}
		for i, q := range e.Questions {
			question := &models.QuestionBank{
				ID: q.ID, SchoolID: q.SchoolID, SessionID: q.SessionID, TermID: q.TermID,
				ClassLevelID: q.ClassLevelID, ClassID: q.ClassID, SubjectID: q.SubjectID,
				ExamType: q.ExamType, Topic: q.Topic, CreatedBy: q.CreatedBy,
				QuestionText: q.QuestionText, Marks: q.Marks,
				CorrectAnswer: q.CorrectOption,
				Options: models.OptionStorage{
					{Key: "A", Text: q.OptionA}, {Key: "B", Text: q.OptionB},
					{Key: "C", Text: q.OptionC}, {Key: "D", Text: q.OptionD},
				},
			}
			if err := c.repo.ApplyPulledQuestion(ctx, question); err != nil {
				zap.L().Warn("apply pulled question failed", zap.String("id", q.ID), zap.Error(err))
				continue
			}
			if err := c.repo.ApplyPulledExamQuestionLink(ctx, e.ID, q.ID, i+1); err != nil {
				zap.L().Warn("apply pulled exam-question link failed", zap.String("exam_id", e.ID), zap.String("question_id", q.ID), zap.Error(err))
			}
		}
	}

	if err := c.repo.SetPullCursor(ctx, pullResp.Data.Cursor); err != nil {
		return fmt.Errorf("save pull cursor: %w", err)
	}

	zap.L().Info("nodesync pull complete",
		zap.Int("students", len(pullResp.Data.Students)),
		zap.Int("exams", len(pullResp.Data.Exams)),
		zap.Time("new_cursor", pullResp.Data.Cursor),
	)
	return nil
}

func (c *SyncClient) doJSON(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cloudBaseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloud returned %d: %s", resp.StatusCode, string(b))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
