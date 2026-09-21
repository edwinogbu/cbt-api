package repository

import (
	"context"
	"time"

	"cbt-api/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NodeSyncRepository struct {
	db *gorm.DB
}

func NewNodeSyncRepository(db *gorm.DB) *NodeSyncRepository {
	return &NodeSyncRepository{db: db}
}

// ============================================================
// Credentials
// ============================================================

func (r *NodeSyncRepository) FindActiveCredentials(ctx context.Context) ([]models.SchoolNodeCredential, error) {
	var creds []models.SchoolNodeCredential
	err := r.db.WithContext(ctx).Where("revoked_at IS NULL").Find(&creds).Error
	return creds, err
}

func (r *NodeSyncRepository) CreateCredential(ctx context.Context, cred *models.SchoolNodeCredential) error {
	return r.db.WithContext(ctx).Create(cred).Error
}

func (r *NodeSyncRepository) TouchLastUsed(ctx context.Context, credID string) {
	now := time.Now()
	r.db.WithContext(ctx).Model(&models.SchoolNodeCredential{}).Where("id = ?", credID).Update("last_used_at", now)
}

// ============================================================
// Tenant-scoping check: does this student belong to this school?
// ============================================================

func (r *NodeSyncRepository) StudentSchoolIDs(ctx context.Context, studentIDs []string) (map[string]string, error) {
	if len(studentIDs) == 0 {
		return map[string]string{}, nil
	}
	var rows []struct {
		ID       string
		SchoolID string
	}
	if err := r.db.WithContext(ctx).Model(&models.Student{}).
		Select("id, school_id").
		Where("id IN ?", studentIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.ID] = row.SchoolID
	}
	return out, nil
}

// ============================================================
// Push: upsert by primary key (the node-generated ID is the idempotency
// key - ON CONFLICT DO UPDATE makes a retried push of the same record a
// safe no-op instead of a duplicate or an error).
// ============================================================

func (r *NodeSyncRepository) UpsertSession(ctx context.Context, a *models.ExamAttempt) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *NodeSyncRepository) UpsertAnswer(ctx context.Context, a *models.StudentAnswer) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *NodeSyncRepository) UpsertResult(ctx context.Context, res *models.Result) error {
	return r.db.WithContext(ctx).Save(res).Error
}

func (r *NodeSyncRepository) UpsertEvent(ctx context.Context, e *models.ExamEventLog) error {
	return r.db.WithContext(ctx).Save(e).Error
}

// ============================================================
// Pull: entities changed since a cursor, scoped to one school.
// ============================================================

func (r *NodeSyncRepository) ClassesByIDs(ctx context.Context, ids []string) ([]models.Class, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var classes []models.Class
	err := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&classes).Error
	return classes, err
}

func (r *NodeSyncRepository) ClassLevelsByIDs(ctx context.Context, ids []string) ([]models.ClassLevel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []models.ClassLevel
	err := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func (r *NodeSyncRepository) ClassArmsByIDs(ctx context.Context, ids []string) ([]models.ClassArm, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []models.ClassArm
	err := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func (r *NodeSyncRepository) UsersByIDs(ctx context.Context, ids []string) ([]models.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []models.User
	err := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Find(&users).Error
	return users, err
}

func (r *NodeSyncRepository) StudentsUpdatedSince(ctx context.Context, schoolID string, since time.Time) ([]models.Student, error) {
	var students []models.Student
	err := r.db.WithContext(ctx).Unscoped().
		Where("school_id = ? AND updated_at > ?", schoolID, since).
		Find(&students).Error
	return students, err
}

// ExamsAssignedToSchoolClassesUpdatedSince returns exams whose class is one
// of this school's classes and that changed since the cursor. Scoping
// through class->school (rather than a direct exam.school_id, which the
// Exam model doesn't have) mirrors how exam-to-student authorization
// already works elsewhere in this codebase (see ExamService.
// StartExamForStudent's exam.ClassID == student.ClassID check).
func (r *NodeSyncRepository) ExamsAssignedToSchoolClassesUpdatedSince(ctx context.Context, schoolID string, since time.Time) ([]models.Exam, error) {
	var exams []models.Exam
	err := r.db.WithContext(ctx).Unscoped().
		Joins("JOIN classes ON classes.id = exams.class_id").
		Where("classes.school_id = ? AND exams.updated_at > ?", schoolID, since).
		Find(&exams).Error
	return exams, err
}

func (r *NodeSyncRepository) QuestionsForExam(ctx context.Context, examID string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Joins("JOIN exam_questions ON exam_questions.question_id = question_bank.id").
		Where("exam_questions.exam_id = ?", examID).
		Order("exam_questions.sort_order ASC").
		Find(&questions).Error
	return questions, err
}

// ============================================================
// Node-side outbox: local rows this node hasn't pushed to cloud yet.
// Used by internal/nodesync/client, running against the NODE's own local
// Postgres (this is the same package/methods as the cloud-side handler
// above uses against the cloud's Postgres - "same binary, two roles").
// ============================================================

const outboxBatchSize = 200

func (r *NodeSyncRepository) UnsyncedSessions(ctx context.Context) ([]models.ExamAttempt, error) {
	var rows []models.ExamAttempt
	err := r.db.WithContext(ctx).Where("synced_to_cloud_at IS NULL").Limit(outboxBatchSize).Find(&rows).Error
	return rows, err
}

func (r *NodeSyncRepository) UnsyncedAnswers(ctx context.Context) ([]models.StudentAnswer, error) {
	var rows []models.StudentAnswer
	err := r.db.WithContext(ctx).Where("synced_to_cloud_at IS NULL").Limit(outboxBatchSize).Find(&rows).Error
	return rows, err
}

func (r *NodeSyncRepository) UnsyncedResults(ctx context.Context) ([]models.Result, error) {
	var rows []models.Result
	err := r.db.WithContext(ctx).Where("synced_to_cloud_at IS NULL").Limit(outboxBatchSize).Find(&rows).Error
	return rows, err
}

func (r *NodeSyncRepository) UnsyncedEvents(ctx context.Context) ([]models.ExamEventLog, error) {
	var rows []models.ExamEventLog
	err := r.db.WithContext(ctx).Where("synced_to_cloud_at IS NULL").Limit(outboxBatchSize).Find(&rows).Error
	return rows, err
}

func (r *NodeSyncRepository) MarkSessionsSynced(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.ExamAttempt{}).Where("id IN ?", ids).Update("synced_to_cloud_at", now).Error
}

func (r *NodeSyncRepository) MarkAnswersSynced(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.StudentAnswer{}).Where("id IN ?", ids).Update("synced_to_cloud_at", now).Error
}

func (r *NodeSyncRepository) MarkResultsSynced(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Result{}).Where("id IN ?", ids).Update("synced_to_cloud_at", now).Error
}

func (r *NodeSyncRepository) MarkEventsSynced(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.ExamEventLog{}).Where("id IN ?", ids).Update("synced_to_cloud_at", now).Error
}

// ============================================================
// Node-side pull cursor + applying pulled entities locally.
// ============================================================

func (r *NodeSyncRepository) GetPullCursor(ctx context.Context) (time.Time, error) {
	var state models.NodeSyncState
	err := r.db.WithContext(ctx).Where("id = ?", "default").First(&state).Error
	if err == gorm.ErrRecordNotFound {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return state.LastPullCursor, nil
}

func (r *NodeSyncRepository) SetPullCursor(ctx context.Context, cursor time.Time) error {
	state := models.NodeSyncState{ID: "default", LastPullCursor: cursor, UpdatedAt: time.Now()}
	return r.db.WithContext(ctx).Save(&state).Error
}

// ApplyPulled{Student,Exam,Question} use an ON CONFLICT upsert that only
// touches the columns the pull payload actually carries. A blind .Save()
// here would overwrite every other column - notably created_at - with its
// Go zero value on every single pull of an already-known row, silently
// corrupting local data the node never intended to change. On first
// INSERT of a genuinely new row, created_at/updated_at are still set by
// GORM's normal auto-timestamp convention.
func (r *NodeSyncRepository) ApplyPulledClassLevel(ctx context.Context, cl *models.ClassLevel) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"school_id", "name", "level_number", "category", "sort_order", "is_active", "updated_at",
		}),
	}).Create(cl).Error
}

func (r *NodeSyncRepository) ApplyPulledClassArm(ctx context.Context, ca *models.ClassArm) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"school_id", "name", "arm_code", "capacity", "sort_order", "is_active", "updated_at",
		}),
	}).Create(ca).Error
}

func (r *NodeSyncRepository) ApplyPulledClass(ctx context.Context, c *models.Class) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"school_id", "session_id", "class_level_id", "class_arm_id", "class_code", "is_active", "updated_at",
		}),
	}).Create(c).Error
}

func (r *NodeSyncRepository) ApplyPulledUser(ctx context.Context, u *models.User) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"username", "email", "password", "first_name", "last_name", "phone_number",
			"role", "status", "email_verified", "is_active", "school_id", "updated_at",
		}),
	}).Create(u).Error
}

func (r *NodeSyncRepository) ApplyPulledStudent(ctx context.Context, s *models.Student) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "school_id", "class_id", "admission_no", "is_active", "status", "updated_at",
		}),
	}).Create(s).Error
}

func (r *NodeSyncRepository) ApplyPulledExam(ctx context.Context, e *models.Exam) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "subject_id", "class_id", "session_id", "term_id", "exam_type", "status",
			"duration_minutes", "total_marks", "pass_mark", "instructions", "start_time", "end_time",
			"shuffle_questions", "shuffle_options", "is_active", "updated_at",
		}),
	}).Create(e).Error
}

func (r *NodeSyncRepository) ApplyPulledQuestion(ctx context.Context, q *models.QuestionBank) error {
	// updated_by is a plain (non-pointer) uuid column with no default -
	// GORM's Create() has no way to send SQL NULL for a Go zero-value
	// string, and an empty string is invalid uuid syntax, so it's
	// explicitly omitted here and left NULL at the DB level instead.
	return r.db.WithContext(ctx).Omit("updated_by").Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"school_id", "session_id", "term_id", "class_level_id", "class_id", "subject_id",
			"exam_type", "topic", "created_by", "question_text", "options", "correct_answer", "marks", "updated_at",
		}),
	}).Create(q).Error
}

func (r *NodeSyncRepository) ApplyPulledExamQuestionLink(ctx context.Context, examID, questionID string, sortOrder int) error {
	link := models.ExamQuestion{ExamID: examID, QuestionID: questionID, SortOrder: sortOrder}
	return r.db.WithContext(ctx).
		Where("exam_id = ? AND question_id = ?", examID, questionID).
		Assign(link).
		FirstOrCreate(&link).Error
}
