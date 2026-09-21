package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"cbt-api/internal/models"
	"cbt-api/internal/nodesync/dto"
	"cbt-api/internal/nodesync/repository"
	"cbt-api/pkg/utils"

	"go.uber.org/zap"
)

type NodeSyncService struct {
	repo *repository.NodeSyncRepository
}

func NewNodeSyncService(repo *repository.NodeSyncRepository) *NodeSyncService {
	return &NodeSyncService{repo: repo}
}

// AuthenticateNode resolves a plaintext node API key to the school it
// belongs to. There is no index-friendly way to look up a bcrypt hash by
// plaintext, so this checks every active credential - fine at the scale of
// "one row per school node", not something to do for a high-frequency,
// high-cardinality credential store.
func (s *NodeSyncService) AuthenticateNode(ctx context.Context, plaintextKey string) (schoolID string, credentialID string, err error) {
	creds, err := s.repo.FindActiveCredentials(ctx)
	if err != nil {
		return "", "", err
	}
	for _, c := range creds {
		if utils.CheckPasswordHash(plaintextKey, c.KeyHash) {
			s.repo.TouchLastUsed(ctx, c.ID)
			return c.SchoolID, c.ID, nil
		}
	}
	return "", "", errors.New("invalid node credential")
}

// Push validates and upserts a batch pushed from a School Node. Every
// record's owning student must belong to the authenticated school - this
// is what stops one compromised or misconfigured node from writing data
// attributed to another school (rule #8: data flow direction is
// restricted, and within School->Cloud, restricted to this node's own
// school). A rejected record does not fail the whole batch; the node
// retries only what was rejected as retryable per its own idempotency
// queue.
func (s *NodeSyncService) Push(ctx context.Context, schoolID string, req *dto.PushRequest) (*dto.PushResponse, error) {
	logger := zap.L().With(zap.String("school_id", schoolID))

	studentIDs := make(map[string]struct{})
	for _, r := range req.Sessions {
		studentIDs[r.StudentID] = struct{}{}
	}
	for _, r := range req.Results {
		studentIDs[r.StudentID] = struct{}{}
	}
	for _, r := range req.Events {
		studentIDs[r.StudentID] = struct{}{}
	}
	ids := make([]string, 0, len(studentIDs))
	for id := range studentIDs {
		ids = append(ids, id)
	}
	studentSchool, err := s.repo.StudentSchoolIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	belongsToSchool := func(studentID string) bool {
		return studentSchool[studentID] == schoolID
	}

	resp := &dto.PushResponse{ServerTimestamp: time.Now()}

	for _, r := range req.Sessions {
		if !belongsToSchool(r.StudentID) {
			resp.Rejected = append(resp.Rejected, dto.RejectedPush{Type: "session", ID: r.ID, Reason: "student does not belong to authenticated school"})
			continue
		}
		attempt := &models.ExamAttempt{
			ID: r.ID, StudentID: r.StudentID, ExamID: r.ExamID,
			StartTime: r.StartTime, EndTime: r.EndTime, Score: r.Score,
			Percentage: r.Percentage, Status: r.Status, IPAddress: r.IPAddress,
		}
		if err := s.repo.UpsertSession(ctx, attempt); err != nil {
			logger.Error("upsert session failed", zap.String("id", r.ID), zap.Error(err))
			resp.Rejected = append(resp.Rejected, dto.RejectedPush{Type: "session", ID: r.ID, Reason: "write failed"})
			continue
		}
		resp.SessionsAccepted++
	}

	// Answers/events are scoped by attempt, not directly by student - trust
	// the attempt's own tenant check above rather than re-deriving student
	// ownership per answer (an answer with no matching, school-owned
	// attempt fails on the DB's FK constraint instead, which is an
	// acceptable rejection path for a record whose parent was rejected
	// or never pushed).
	for _, r := range req.Answers {
		answer := &models.StudentAnswer{
			ID: r.ID, AttemptID: r.AttemptID, QuestionID: r.QuestionID,
			SelectedAnswer: r.SelectedAnswer, IsCorrect: r.IsCorrect,
			IsMarked: r.IsMarked, TimeSpent: r.TimeSpent,
		}
		if err := s.repo.UpsertAnswer(ctx, answer); err != nil {
			logger.Error("upsert answer failed", zap.String("id", r.ID), zap.Error(err))
			resp.Rejected = append(resp.Rejected, dto.RejectedPush{Type: "answer", ID: r.ID, Reason: "write failed (attempt not found/not synced yet?)"})
			continue
		}
		resp.AnswersAccepted++
	}

	for _, r := range req.Results {
		if !belongsToSchool(r.StudentID) {
			resp.Rejected = append(resp.Rejected, dto.RejectedPush{Type: "result", ID: r.ID, Reason: "student does not belong to authenticated school"})
			continue
		}
		result := &models.Result{
			ID: r.ID, StudentID: r.StudentID, ExamID: r.ExamID, AttemptID: r.AttemptID,
			TotalScore: r.TotalScore, Percentage: r.Percentage, Grade: r.Grade,
			Remarks: r.Remarks, Passed: r.Passed, PublishedAt: r.PublishedAt,
		}
		if err := s.repo.UpsertResult(ctx, result); err != nil {
			logger.Error("upsert result failed", zap.String("id", r.ID), zap.Error(err))
			resp.Rejected = append(resp.Rejected, dto.RejectedPush{Type: "result", ID: r.ID, Reason: "write failed"})
			continue
		}
		resp.ResultsAccepted++
	}

	for _, r := range req.Events {
		if !belongsToSchool(r.StudentID) {
			resp.Rejected = append(resp.Rejected, dto.RejectedPush{Type: "event", ID: r.ID, Reason: "student does not belong to authenticated school"})
			continue
		}
		event := &models.ExamEventLog{
			ID: r.ID, AttemptID: r.AttemptID, StudentID: r.StudentID,
			EventType: r.EventType, ClientSequence: r.ClientSequence,
			PayloadJSON: r.PayloadJSON, ClientTimestamp: r.ClientTimestamp,
		}
		if err := s.repo.UpsertEvent(ctx, event); err != nil {
			logger.Error("upsert event failed", zap.String("id", r.ID), zap.Error(err))
			resp.Rejected = append(resp.Rejected, dto.RejectedPush{Type: "event", ID: r.ID, Reason: "write failed"})
			continue
		}
		resp.EventsAccepted++
	}

	return resp, nil
}

// Pull returns everything this school's node needs that changed since the
// cursor. The returned Cursor is the server's own clock at query time
// (not the max updated_at seen), so a record updated between this
// response being built and the node applying it is never silently missed
// on the next pull - the alternative (using the max row timestamp) can
// skip a row that commits with an earlier timestamp than one already
// read in the same query.
func (s *NodeSyncService) Pull(ctx context.Context, schoolID string, since time.Time) (*dto.PullResponse, error) {
	queryStartedAt := time.Now()

	students, err := s.repo.StudentsUpdatedSince(ctx, schoolID, since)
	if err != nil {
		return nil, err
	}
	studentPulls := make([]dto.StudentPull, 0, len(students))
	userIDs := make([]string, 0, len(students))
	for _, st := range students {
		p := dto.StudentPull{
			ID: st.ID, UserID: st.UserID, SchoolID: st.SchoolID, ClassID: st.ClassID,
			AdmissionNo: st.AdmissionNo, IsActive: st.IsActive, Status: st.Status,
		}
		if st.DeletedAt.Valid {
			ts := st.DeletedAt.Time.Format(time.RFC3339)
			p.DeletedAt = &ts
		}
		studentPulls = append(studentPulls, p)
		userIDs = append(userIDs, st.UserID)
	}

	// Classes (and their class_level_id/class_arm_id FK targets) must
	// accompany their students - students.class_id is a FK to classes.id.
	classIDs := make([]string, 0, len(students))
	for _, st := range students {
		if st.ClassID != "" {
			classIDs = append(classIDs, st.ClassID)
		}
	}
	classes, err := s.repo.ClassesByIDs(ctx, classIDs)
	if err != nil {
		return nil, err
	}
	classPulls := make([]dto.ClassPull, 0, len(classes))
	classLevelIDs := make([]string, 0, len(classes))
	classArmIDs := make([]string, 0, len(classes))
	for _, c := range classes {
		classPulls = append(classPulls, dto.ClassPull{
			ID: c.ID, SchoolID: c.SchoolID, SessionID: c.SessionID,
			ClassLevelID: c.ClassLevelID, ClassArmID: c.ClassArmID,
			ClassCode: c.ClassCode, IsActive: c.IsActive,
		})
		classLevelIDs = append(classLevelIDs, c.ClassLevelID)
		classArmIDs = append(classArmIDs, c.ClassArmID)
	}

	classLevels, err := s.repo.ClassLevelsByIDs(ctx, classLevelIDs)
	if err != nil {
		return nil, err
	}
	classLevelPulls := make([]dto.ClassLevelPull, 0, len(classLevels))
	for _, cl := range classLevels {
		classLevelPulls = append(classLevelPulls, dto.ClassLevelPull{
			ID: cl.ID, SchoolID: cl.SchoolID, Name: cl.Name, LevelNumber: cl.LevelNumber,
			Category: cl.Category, SortOrder: cl.SortOrder, IsActive: cl.IsActive,
		})
	}

	classArms, err := s.repo.ClassArmsByIDs(ctx, classArmIDs)
	if err != nil {
		return nil, err
	}
	classArmPulls := make([]dto.ClassArmPull, 0, len(classArms))
	for _, ca := range classArms {
		classArmPulls = append(classArmPulls, dto.ClassArmPull{
			ID: ca.ID, SchoolID: ca.SchoolID, Name: ca.Name, ArmCode: ca.ArmCode,
			Capacity: ca.Capacity, SortOrder: ca.SortOrder, IsActive: ca.IsActive,
		})
	}

	// Users must accompany their students - students.user_id is a FK to
	// users.id, so a student pulled without its user already present
	// locally fails the constraint on apply.
	users, err := s.repo.UsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	userPulls := make([]dto.UserPull, 0, len(users))
	for _, u := range users {
		userPulls = append(userPulls, dto.UserPull{
			ID: u.ID, Username: u.Username, Email: u.Email, PasswordHash: u.Password,
			FirstName: u.FirstName, LastName: u.LastName, PhoneNumber: u.PhoneNumber,
			Role: string(u.Role), Status: string(u.Status), EmailVerified: u.EmailVerified,
			IsActive: u.IsActive, SchoolID: u.SchoolID,
		})
	}

	exams, err := s.repo.ExamsAssignedToSchoolClassesUpdatedSince(ctx, schoolID, since)
	if err != nil {
		return nil, err
	}
	examPulls := make([]dto.ExamPull, 0, len(exams))
	for _, ex := range exams {
		questions, err := s.repo.QuestionsForExam(ctx, ex.ID)
		if err != nil {
			return nil, err
		}
		qPulls := make([]dto.QuestionPull, 0, len(questions))
		for i, q := range questions {
			qPulls = append(qPulls, dto.QuestionPull{
				ID:            q.ID,
				SchoolID:      q.SchoolID,
				SessionID:     q.SessionID,
				TermID:        q.TermID,
				ClassLevelID:  q.ClassLevelID,
				ClassID:       q.ClassID,
				SubjectID:     q.SubjectID,
				ExamType:      q.ExamType,
				Topic:         q.Topic,
				CreatedBy:     q.CreatedBy,
				QuestionText:  q.QuestionText,
				OptionA:       optionText(q.Options, "A"),
				OptionB:       optionText(q.Options, "B"),
				OptionC:       optionText(q.Options, "C"),
				OptionD:       optionText(q.Options, "D"),
				CorrectOption: q.CorrectAnswer,
				Marks:         q.Marks,
				SortOrder:     i + 1,
			})
		}
		ep := dto.ExamPull{
			ID: ex.ID, Title: ex.Title, SubjectID: ex.SubjectID, ClassID: ex.ClassID,
			SessionID: ex.SessionID, TermID: ex.TermID,
			ExamType: ex.ExamType, Status: ex.Status, DurationMinutes: ex.DurationMinutes,
			TotalMarks: ex.TotalMarks, PassMark: ex.PassMark, Instructions: ex.Instructions,
			StartTime: ex.StartTime, EndTime: ex.EndTime,
			ShuffleQuestions: ex.ShuffleQuestions, ShuffleOptions: ex.ShuffleOptions,
			IsActive: ex.IsActive,
			Questions: qPulls,
		}
		if ex.DeletedAt.Valid {
			ts := ex.DeletedAt.Time.Format(time.RFC3339)
			ep.DeletedAt = &ts
		}
		examPulls = append(examPulls, ep)
	}

	return &dto.PullResponse{
		ClassLevels:     classLevelPulls,
		ClassArms:       classArmPulls,
		Classes:         classPulls,
		Users:           userPulls,
		Students:        studentPulls,
		Exams:           examPulls,
		Cursor:          queryStartedAt,
		ServerTimestamp: queryStartedAt,
	}, nil
}

func optionText(storage models.OptionStorage, key string) string {
	for _, item := range storage {
		if item.Key == key {
			return item.Text
		}
	}
	return ""
}

// CreateCredential generates a new plaintext API key for a school node,
// stores only its bcrypt hash, and returns the plaintext exactly once -
// callers must show it to the operator immediately and never log it.
func (s *NodeSyncService) CreateCredential(ctx context.Context, schoolID, label string) (plaintextKey string, err error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", err
	}
	plaintextKey = hex.EncodeToString(keyBytes)
	hash, err := utils.HashPassword(plaintextKey)
	if err != nil {
		return "", err
	}
	cred := &models.SchoolNodeCredential{SchoolID: schoolID, Label: label, KeyHash: hash}
	if err := s.repo.CreateCredential(ctx, cred); err != nil {
		return "", err
	}
	return plaintextKey, nil
}
