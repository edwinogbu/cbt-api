package repository

import (
	"context"
	"errors"
	"time"

	"cbt-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================
// CURRENT ACADEMIC CONTEXT
// ============================================

type CurrentAcademicContext struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
	TermID      string `json:"term_id"`
	TermName    string `json:"term_name"`
	TermNumber  int    `json:"term_number"`
}

// ============================================
// EXAM REPOSITORY STRUCT
// ============================================

type ExamRepository struct {
	db *gorm.DB
}

func NewExamRepository(db *gorm.DB) *ExamRepository {
	return &ExamRepository{db: db}
}

// ============================================
// HELPER - Generate ID
// ============================================

func generateID() string {
	return uuid.New().String()
}

// ============================================
// EXAM CRUD OPERATIONS
// ============================================

// CreateExam creates a new exam
func (r *ExamRepository) CreateExam(exam *models.Exam) error {
	return r.db.Create(exam).Error
}

// CreateExamWithContext creates a new exam with context
func (r *ExamRepository) CreateExamWithContext(ctx context.Context, exam *models.Exam) error {
	return r.db.WithContext(ctx).Create(exam).Error
}

// Add this method to ExamRepository - GetSubjectQuestionsWithContext
// GetSubjectQuestionsWithContext gets questions by subject with context
func (r *ExamRepository) GetSubjectQuestionsWithContext(ctx context.Context, subjectID, termID, sessionID, classID string) ([]models.QuestionBank, error) {
    var questions []models.QuestionBank
    query := r.db.WithContext(ctx).
        Where("subject_id = ? AND deleted_at IS NULL", subjectID)
    
    if termID != "" {
        query = query.Where("term_id = ?", termID)
    }
    if sessionID != "" {
        query = query.Where("session_id = ?", sessionID)
    }
    if classID != "" {
        query = query.Where("class_id = ?", classID)
    }
    
    err := query.Order("created_at DESC").Find(&questions).Error
    return questions, err
}



// // GetSubjectQuestionsWithContext - Add this method
// func (r *ExamRepository) GetSubjectQuestionsWithContext(ctx context.Context, subjectID, termID, sessionID, classID string) ([]models.QuestionBank, error) {
//     var questions []models.QuestionBank
//     query := r.db.WithContext(ctx).
//         Where("subject_id = ? AND deleted_at IS NULL", subjectID)
    
//     if termID != "" {
//         query = query.Where("term_id = ?", termID)
//     }
//     if sessionID != "" {
//         query = query.Where("session_id = ?", sessionID)
//     }
//     if classID != "" {
//         query = query.Where("class_id = ?", classID)
//     }
    
//     err := query.Order("created_at DESC").Find(&questions).Error
//     return questions, err
// }

// FindExamByID finds an exam by ID
func (r *ExamRepository) FindExamByID(id string) (*models.Exam, error) {
	var exam models.Exam
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&exam).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("exam not found")
		}
		return nil, err
	}
	return &exam, nil
}

// FindExamByIDWithContext finds an exam by ID with context
func (r *ExamRepository) FindExamByIDWithContext(ctx context.Context, id string) (*models.Exam, error) {
	var exam models.Exam
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&exam).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("exam not found")
		}
		return nil, err
	}
	return &exam, nil
}

// UpdateExam updates an existing exam
func (r *ExamRepository) UpdateExam(exam *models.Exam) error {
	return r.db.Save(exam).Error
}

// UpdateExamWithContext updates an exam with context
func (r *ExamRepository) UpdateExamWithContext(ctx context.Context, exam *models.Exam) error {
	return r.db.WithContext(ctx).Save(exam).Error
}

// DeleteExam soft deletes an exam
func (r *ExamRepository) DeleteExam(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Exam{}).Error
}

// DeleteExamWithContext soft deletes an exam with context
func (r *ExamRepository) DeleteExamWithContext(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Exam{}).Error
}

// ListExams lists all exams with pagination
func (r *ExamRepository) ListExams(ctx context.Context, page, limit int) ([]models.Exam, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var exams []models.Exam
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Exam{}).Where("deleted_at IS NULL")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&exams).Error
	return exams, total, err
}

// ListExamsBySubject lists exams by subject
func (r *ExamRepository) ListExamsBySubject(ctx context.Context, subjectID string, page, limit int) ([]models.Exam, int64, error) {
	var exams []models.Exam
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Exam{}).
		Where("subject_id = ? AND deleted_at IS NULL", subjectID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&exams).Error
	return exams, total, err
}

// ListExamsByTerm lists exams by term
func (r *ExamRepository) ListExamsByTerm(ctx context.Context, termID string, page, limit int) ([]models.Exam, int64, error) {
	var exams []models.Exam
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Exam{}).
		Where("term_id = ? AND deleted_at IS NULL", termID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&exams).Error
	return exams, total, err
}

// ListExamsBySession lists exams by session
func (r *ExamRepository) ListExamsBySession(ctx context.Context, sessionID string, page, limit int) ([]models.Exam, int64, error) {
	var exams []models.Exam
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Exam{}).
		Where("session_id = ? AND deleted_at IS NULL", sessionID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&exams).Error
	return exams, total, err
}

// ListExamsByClass lists exams by class
func (r *ExamRepository) ListExamsByClass(ctx context.Context, classID string, page, limit int) ([]models.Exam, int64, error) {
	var exams []models.Exam
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Exam{}).
		Where("class_id = ? AND deleted_at IS NULL", classID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&exams).Error
	return exams, total, err
}

// // GetExamsByStudentAndClass gets exams for a student by class
// func (r *ExamRepository) GetExamsByStudentAndClass(ctx context.Context, studentID, classID string) ([]models.Exam, error) {
// 	var exams []models.Exam

// 	err := r.db.WithContext(ctx).
// 		Model(&models.Exam{}).
// 		Joins("JOIN exam_assignments ON exam_assignments.exam_id = exams.id").
// 		Where("(exam_assignments.student_id = ? OR exam_assignments.class_id = ?)", studentID, classID).
// 		Where("exams.deleted_at IS NULL").
// 		Where("exams.is_active = ?", true).
// 		Where("exams.status IN ?", []string{string(models.ExamStatusPublished), string(models.ExamStatusActive)}).
// 		Order("exams.start_time ASC").
// 		Find(&exams).Error

// 	return exams, err
// }

func (r *ExamRepository) GetExamsByStudentAndClass(ctx context.Context, studentID, classID string) ([]models.Exam, error) {
    var exams []models.Exam

    err := r.db.WithContext(ctx).
        Model(&models.Exam{}).
        Joins("JOIN exam_assignments ON exam_assignments.exam_id = exams.id").
        Where("(exam_assignments.student_id = ? OR exam_assignments.class_id = ?)", studentID, classID).
        // ✅ FIX: Ensure exam's class matches student's class
        Where("exams.class_id = ?", classID).
        Where("exams.deleted_at IS NULL").
        Where("exams.is_active = ?", true).
        Where("exams.status IN ?", []string{string(models.ExamStatusPublished), string(models.ExamStatusActive)}).
        Order("exams.start_time ASC").
        Find(&exams).Error

    return exams, err
}

// ============================================
// EXAM QUESTIONS (PIVOT TABLE)
// ============================================

// AddQuestionToExam adds a question to an exam
func (r *ExamRepository) AddQuestionToExam(examID, questionID string, sortOrder int) error {
	eq := models.ExamQuestion{
		ID:         generateID(),
		ExamID:     examID,
		QuestionID: questionID,
		SortOrder:  sortOrder,
	}
	return r.db.Create(&eq).Error
}

// AddQuestionsToExam adds multiple questions to an exam
func (r *ExamRepository) AddQuestionsToExam(ctx context.Context, examID string, questionIDs []string) error {
	for i, qid := range questionIDs {
		eq := models.ExamQuestion{
			ID:         generateID(),
			ExamID:     examID,
			QuestionID: qid,
			SortOrder:  i + 1,
		}
		if err := r.db.WithContext(ctx).Create(&eq).Error; err != nil {
			return err
		}
	}
	return nil
}

// BulkAddQuestionsToExam bulk adds questions to an exam (replaces all existing)
func (r *ExamRepository) BulkAddQuestionsToExam(ctx context.Context, examID string, questionIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing questions
		if err := tx.Where("exam_id = ?", examID).Delete(&models.ExamQuestion{}).Error; err != nil {
			return err
		}
		// Add new questions
		for i, qid := range questionIDs {
			eq := models.ExamQuestion{
				ID:         generateID(),
				ExamID:     examID,
				QuestionID: qid,
				SortOrder:  i + 1,
			}
			if err := tx.Create(&eq).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// RemoveQuestionFromExam removes a question from an exam
func (r *ExamRepository) RemoveQuestionFromExam(examID, questionID string) error {
	return r.db.Where("exam_id = ? AND question_id = ?", examID, questionID).
		Delete(&models.ExamQuestion{}).Error
}

// RemoveQuestionFromExamWithContext removes a question with context
func (r *ExamRepository) RemoveQuestionFromExamWithContext(ctx context.Context, examID, questionID string) error {
	return r.db.WithContext(ctx).Where("exam_id = ? AND question_id = ?", examID, questionID).
		Delete(&models.ExamQuestion{}).Error
}

// GetExamQuestions gets all questions for an exam
func (r *ExamRepository) GetExamQuestions(examID string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.Table("question_bank").
		Joins("JOIN exam_questions ON exam_questions.question_id = question_bank.id").
		Where("exam_questions.exam_id = ? AND question_bank.deleted_at IS NULL", examID).
		Order("exam_questions.sort_order ASC").
		Find(&questions).Error
	return questions, err
}

// GetExamQuestionsWithContext gets questions with context
func (r *ExamRepository) GetExamQuestionsWithContext(ctx context.Context, examID string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).Table("question_bank").
		Joins("JOIN exam_questions ON exam_questions.question_id = question_bank.id").
		Where("exam_questions.exam_id = ? AND question_bank.deleted_at IS NULL", examID).
		Order("exam_questions.sort_order ASC").
		Find(&questions).Error
	return questions, err
}

// FindExamWithQuestions finds an exam with its questions
func (r *ExamRepository) FindExamWithQuestions(examID string) (*models.Exam, []models.QuestionBank, error) {
	exam, err := r.FindExamByID(examID)
	if err != nil {
		return nil, nil, err
	}
	questions, err := r.GetExamQuestions(examID)
	if err != nil {
		return nil, nil, err
	}
	return exam, questions, nil
}

// FindExamWithQuestionsWithContext finds an exam with questions with context
func (r *ExamRepository) FindExamWithQuestionsWithContext(ctx context.Context, examID string) (*models.Exam, []models.QuestionBank, error) {
	exam, err := r.FindExamByIDWithContext(ctx, examID)
	if err != nil {
		return nil, nil, err
	}
	questions, err := r.GetExamQuestionsWithContext(ctx, examID)
	if err != nil {
		return nil, nil, err
	}
	return exam, questions, nil
}

// CountQuestionsInExam counts questions in an exam
func (r *ExamRepository) CountQuestionsInExam(ctx context.Context, examID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ExamQuestion{}).
		Where("exam_id = ?", examID).Count(&count).Error
	return count, err
}

// ============================================
// EXAM ATTEMPTS
// ============================================

// CreateAttempt creates a new exam attempt
func (r *ExamRepository) CreateAttempt(attempt *models.ExamAttempt) error {
	return r.db.Create(attempt).Error
}

// FindAttemptByID finds an attempt by ID
func (r *ExamRepository) FindAttemptByID(id string) (*models.ExamAttempt, error) {
	var attempt models.ExamAttempt
	err := r.db.Where("id = ?", id).First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attempt not found")
		}
		return nil, err
	}
	return &attempt, nil
}

// FindAttemptByIDWithContext finds an attempt with context
func (r *ExamRepository) FindAttemptByIDWithContext(ctx context.Context, id string) (*models.ExamAttempt, error) {
	var attempt models.ExamAttempt
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attempt not found")
		}
		return nil, err
	}
	return &attempt, nil
}

// FindActiveAttempt finds an active attempt for a student and exam
func (r *ExamRepository) FindActiveAttempt(studentID, examID string) (*models.ExamAttempt, error) {
	var attempt models.ExamAttempt
	err := r.db.Where("student_id = ? AND exam_id = ? AND status = ?",
		studentID, examID, models.AttemptStatusInProgress).First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attempt, nil
}

// FindActiveAttemptWithContext finds an active attempt with context
func (r *ExamRepository) FindActiveAttemptWithContext(ctx context.Context, studentID, examID string) (*models.ExamAttempt, error) {
	var attempt models.ExamAttempt
	err := r.db.WithContext(ctx).Where("student_id = ? AND exam_id = ? AND status = ?",
		studentID, examID, models.AttemptStatusInProgress).First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attempt, nil
}

// FindAttemptByStudentAndExam finds an attempt by student and exam
func (r *ExamRepository) FindAttemptByStudentAndExam(ctx context.Context, studentID, examID string) (*models.ExamAttempt, error) {
	var attempt models.ExamAttempt
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND exam_id = ?", studentID, examID).
		Order("created_at DESC").
		First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attempt not found")
		}
		return nil, err
	}
	return &attempt, nil
}

// GetAttemptsByStudent gets all attempts for a student
func (r *ExamRepository) GetAttemptsByStudent(ctx context.Context, studentID string) ([]models.ExamAttempt, error) {
	var attempts []models.ExamAttempt
	err := r.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("created_at DESC").
		Find(&attempts).Error
	return attempts, err
}

// GetActiveAttemptsByStudent gets active attempts for a student
func (r *ExamRepository) GetActiveAttemptsByStudent(ctx context.Context, studentID string) ([]models.ExamAttempt, error) {
	var attempts []models.ExamAttempt
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND status = ?", studentID, models.AttemptStatusInProgress).
		Order("created_at DESC").
		Find(&attempts).Error
	return attempts, err
}

// GetCompletedAttemptsByStudent gets completed attempts for a student
func (r *ExamRepository) GetCompletedAttemptsByStudent(ctx context.Context, studentID string) ([]models.ExamAttempt, error) {
	var attempts []models.ExamAttempt
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND status = ?", studentID, models.AttemptStatusCompleted).
		Order("created_at DESC").
		Find(&attempts).Error
	return attempts, err
}

// UpdateAttempt updates an attempt
func (r *ExamRepository) UpdateAttempt(attempt *models.ExamAttempt) error {
	return r.db.Save(attempt).Error
}

// UpdateAttemptStatus updates the status of an attempt
func (r *ExamRepository) UpdateAttemptStatus(id string, status string) error {
	return r.db.Model(&models.ExamAttempt{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateAttemptStatusWithContext updates status with context
func (r *ExamRepository) UpdateAttemptStatusWithContext(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&models.ExamAttempt{}).Where("id = ?", id).Update("status", status).Error
}

// GetAnsweredCount gets the number of answered questions for an attempt
func (r *ExamRepository) GetAnsweredCount(attemptID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count).Error
	return count, err
}

// GetAnsweredCountWithContext gets answered count with context
func (r *ExamRepository) GetAnsweredCountWithContext(ctx context.Context, attemptID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count).Error
	return count, err
}

// GetMarkedForReviewCount gets count of marked questions
func (r *ExamRepository) GetMarkedForReviewCount(ctx context.Context, attemptID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.StudentAnswer{}).
		Where("attempt_id = ? AND is_marked = ?", attemptID, true).Count(&count).Error
	return count, err
}

// ============================================
// STUDENT ANSWERS
// ============================================

// SaveAnswer saves a student answer
func (r *ExamRepository) SaveAnswer(answer *models.StudentAnswer) error {
	return r.db.Create(answer).Error
}

// UpdateAnswer updates a student answer
func (r *ExamRepository) UpdateAnswer(answer *models.StudentAnswer) error {
	return r.db.Save(answer).Error
}

// FindAnswer finds a student answer by attempt and question
func (r *ExamRepository) FindAnswer(attemptID, questionID string) (*models.StudentAnswer, error) {
	var ans models.StudentAnswer
	err := r.db.Where("attempt_id = ? AND question_id = ?", attemptID, questionID).First(&ans).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ans, nil
}

// FindAnswersByAttempt finds all answers for an attempt
func (r *ExamRepository) FindAnswersByAttempt(attemptID string) ([]models.StudentAnswer, error) {
	var answers []models.StudentAnswer
	err := r.db.Where("attempt_id = ?", attemptID).Find(&answers).Error
	return answers, err
}

// FindAnswersByAttemptWithContext finds answers with context
func (r *ExamRepository) FindAnswersByAttemptWithContext(ctx context.Context, attemptID string) ([]models.StudentAnswer, error) {
	var answers []models.StudentAnswer
	err := r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).Find(&answers).Error
	return answers, err
}

// ============================================
// OFFLINE ANSWERS
// ============================================

// SaveOfflineAnswer saves an offline answer
func (r *ExamRepository) SaveOfflineAnswer(answer *models.OfflineAnswer) error {
	return r.db.Create(answer).Error
}

// SaveOfflineAnswers saves multiple offline answers
func (r *ExamRepository) SaveOfflineAnswers(ctx context.Context, answers []models.OfflineAnswer) error {
	return r.db.WithContext(ctx).Create(&answers).Error
}

// FindOfflineAnswers finds offline answers for a student and exam
func (r *ExamRepository) FindOfflineAnswers(studentID, examID string) ([]models.OfflineAnswer, error) {
	var answers []models.OfflineAnswer
	err := r.db.Where("student_id = ? AND exam_id = ? AND synced_at IS NULL", studentID, examID).
		Find(&answers).Error
	return answers, err
}

// FindOfflineAnswersByAttempt finds offline answers by attempt
func (r *ExamRepository) FindOfflineAnswersByAttempt(ctx context.Context, attemptID string) ([]models.OfflineAnswer, error) {
	var answers []models.OfflineAnswer
	err := r.db.WithContext(ctx).
		Where("attempt_id = ? AND synced_at IS NULL", attemptID).
		Find(&answers).Error
	return answers, err
}

// MarkOfflineAnswersSynced marks offline answers as synced
func (r *ExamRepository) MarkOfflineAnswersSynced(ids []string) error {
	now := time.Now()
	return r.db.Model(&models.OfflineAnswer{}).
		Where("id IN ?", ids).
		Update("synced_at", now).Error
}

// MarkOfflineAnswersSyncedWithContext marks offline answers as synced with context
func (r *ExamRepository) MarkOfflineAnswersSyncedWithContext(ctx context.Context, ids []string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.OfflineAnswer{}).
		Where("id IN ?", ids).
		Update("synced_at", now).Error
}

// ============================================
// RESULTS
// ============================================

// CreateResult creates a result
func (r *ExamRepository) CreateResult(result *models.Result) error {
	return r.db.Create(result).Error
}

// FindResultByExamAndStudent finds a result by exam and student
func (r *ExamRepository) FindResultByExamAndStudent(examID, studentID string) (*models.Result, error) {
	var res models.Result
	err := r.db.Where("exam_id = ? AND student_id = ?", examID, studentID).First(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

// FindResultByAttempt finds a result by attempt
func (r *ExamRepository) FindResultByAttempt(ctx context.Context, attemptID string) (*models.Result, error) {
	var result models.Result
	err := r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// FindResultsByStudent finds all results for a student
func (r *ExamRepository) FindResultsByStudent(ctx context.Context, studentID string) ([]models.Result, error) {
	var results []models.Result
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Order("created_at DESC").Find(&results).Error
	return results, err
}

// FindResultsByStudentAndTerm finds results by student and term
func (r *ExamRepository) FindResultsByStudentAndTerm(ctx context.Context, studentID, termID, sessionID string) ([]models.Result, error) {
	var results []models.Result
	err := r.db.WithContext(ctx).
		Table("results").
		Joins("JOIN exams ON exams.id = results.exam_id").
		Where("results.student_id = ? AND exams.term_id = ? AND exams.session_id = ?", studentID, termID, sessionID).
		Find(&results).Error
	return results, err
}

// FindResultsByClassAndTerm finds results for a class and term
func (r *ExamRepository) FindResultsByClassAndTerm(ctx context.Context, classID, termID string) ([]models.Result, error) {
	var results []models.Result
	err := r.db.WithContext(ctx).
		Table("results").
		Joins("JOIN exams ON exams.id = results.exam_id").
		Joins("JOIN students ON students.id = results.student_id").
		Where("students.class_id = ? AND exams.term_id = ?", classID, termID).
		Find(&results).Error
	return results, err
}

// GetStudentExamResult gets a student's exam result
func (r *ExamRepository) GetStudentExamResult(ctx context.Context, studentID, examID string) (*models.Result, error) {
	var result models.Result
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND exam_id = ?", studentID, examID).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// ============================================
// EXAM ASSIGNMENTS
// ============================================

// CreateAssignment creates an exam assignment
func (r *ExamRepository) CreateAssignment(assignment *models.ExamAssignment) error {
	return r.db.Create(assignment).Error
}

// CreateAssignmentWithContext creates an assignment with context
func (r *ExamRepository) CreateAssignmentWithContext(ctx context.Context, assignment *models.ExamAssignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

// FindAssignmentsByExam finds assignments by exam
func (r *ExamRepository) FindAssignmentsByExam(examID string) ([]models.ExamAssignment, error) {
	var assignments []models.ExamAssignment
	err := r.db.Where("exam_id = ?", examID).Find(&assignments).Error
	return assignments, err
}

// FindAssignmentsByExamWithContext finds assignments with context
func (r *ExamRepository) FindAssignmentsByExamWithContext(ctx context.Context, examID string) ([]models.ExamAssignment, error) {
	var assignments []models.ExamAssignment
	err := r.db.WithContext(ctx).Where("exam_id = ?", examID).Find(&assignments).Error
	return assignments, err
}

// FindAssignmentsByStudent finds assignments by student
func (r *ExamRepository) FindAssignmentsByStudent(ctx context.Context, studentID string) ([]models.ExamAssignment, error) {
	var assignments []models.ExamAssignment
	err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Find(&assignments).Error
	return assignments, err
}

// FindAssignmentsByClass finds assignments by class
func (r *ExamRepository) FindAssignmentsByClass(ctx context.Context, classID string) ([]models.ExamAssignment, error) {
	var assignments []models.ExamAssignment
	err := r.db.WithContext(ctx).Where("class_id = ?", classID).Find(&assignments).Error
	return assignments, err
}

// ============================================
// PRACTICE SESSIONS
// ============================================

// CreatePracticeSession creates a practice session
func (r *ExamRepository) CreatePracticeSession(session *models.PracticeSession) error {
	return r.db.Create(session).Error
}

// FindPracticeSession finds a practice session by ID
func (r *ExamRepository) FindPracticeSession(ctx context.Context, id string) (*models.PracticeSession, error) {
	var sess models.PracticeSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sess).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sess, nil
}

// UpdatePracticeSession updates a practice session
func (r *ExamRepository) UpdatePracticeSession(session *models.PracticeSession) error {
	return r.db.Save(session).Error
}

// GetRandomQuestions gets random questions for practice
func (r *ExamRepository) GetRandomQuestions(ctx context.Context, subjectID string, limit int) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).Where("subject_id = ? AND deleted_at IS NULL", subjectID).
		Order("RANDOM()").Limit(limit).Find(&questions).Error
	return questions, err
}

// ============================================
// PROCTORING
// ============================================

// CreateProctoringSession creates a proctoring session
func (r *ExamRepository) CreateProctoringSession(session *models.ProctoringSession) error {
	return r.db.Create(session).Error
}

// FindProctoringByAttempt finds proctoring by attempt
func (r *ExamRepository) FindProctoringByAttempt(ctx context.Context, attemptID string) (*models.ProctoringSession, error) {
	var sess models.ProctoringSession
	err := r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).First(&sess).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sess, nil
}

// UpdateProctoringSession updates a proctoring session
func (r *ExamRepository) UpdateProctoringSession(session *models.ProctoringSession) error {
	return r.db.Save(session).Error
}

// CreateViolation creates a proctoring violation
func (r *ExamRepository) CreateViolation(violation *models.ProctoringViolation) error {
	return r.db.Create(violation).Error
}

// GetViolationCount gets the count of violations for a proctoring session
func (r *ExamRepository) GetViolationCount(ctx context.Context, proctoringID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ProctoringViolation{}).
		Where("proctoring_id = ?", proctoringID).Count(&count).Error
	return count, err
}

// GetViolationsByAttempt gets violations by attempt
func (r *ExamRepository) GetViolationsByAttempt(ctx context.Context, attemptID string) ([]models.ProctoringViolation, error) {
	var violations []models.ProctoringViolation
	err := r.db.WithContext(ctx).
		Where("attempt_id = ?", attemptID).
		Order("timestamp DESC").
		Find(&violations).Error
	return violations, err
}

// ============================================
// RESULTS & PERFORMANCE - CLASS/TEACHER
// ============================================

// GetExamResultsByClass gets exam results for a class
func (r *ExamRepository) GetExamResultsByClass(ctx context.Context, examID, classID string, page, limit int) ([]models.ExamAttempt, int64, error) {
	var attempts []models.ExamAttempt
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.ExamAttempt{}).
		Joins("JOIN students ON students.id = exam_attempts.student_id").
		Joins("JOIN classes ON classes.id = students.class_id").
		Where("exam_attempts.exam_id = ? AND classes.id = ?", examID, classID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).
		Order("exam_attempts.score DESC").
		Find(&attempts).Error

	return attempts, total, err
}

// GetExamRankings gets exam rankings
func (r *ExamRepository) GetExamRankings(ctx context.Context, examID string, limit int) ([]models.ExamAttempt, error) {
	var attempts []models.ExamAttempt
	err := r.db.WithContext(ctx).
		Where("exam_id = ? AND status = ? AND score IS NOT NULL", examID, models.AttemptStatusCompleted).
		Order("score DESC, created_at ASC").
		Limit(limit).
		Find(&attempts).Error
	return attempts, err
}

// GetExamStatistics gets exam statistics
func (r *ExamRepository) GetExamStatistics(ctx context.Context, examID string) (map[string]interface{}, error) {
	var totalAttempts, completedCount, passedCount, failedCount int64
	var avgScore float64
	var inProgressCount int64

	// Total attempts
	r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
		Where("exam_id = ?", examID).Count(&totalAttempts)

	// Completed count
	r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
		Where("exam_id = ? AND status = ?", examID, models.AttemptStatusCompleted).Count(&completedCount)

	// In progress count
	r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
		Where("exam_id = ? AND status = ?", examID, models.AttemptStatusInProgress).Count(&inProgressCount)

	// Get exam for pass mark
	exam, err := r.FindExamByIDWithContext(ctx, examID)
	if err == nil && exam != nil {
		r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
			Where("exam_id = ? AND status = ? AND score >= ?", examID, models.AttemptStatusCompleted, exam.PassMark).
			Count(&passedCount)
		failedCount = completedCount - passedCount
	}

	// Average score
	r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
		Where("exam_id = ? AND status = ?", examID, models.AttemptStatusCompleted).
		Select("COALESCE(AVG(score), 0)").Row().Scan(&avgScore)

	// Grade distribution
	gradeDist := make(map[string]int)
	if exam != nil {
		var results []struct {
			Score int
		}
		r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
			Where("exam_id = ? AND status = ?", examID, models.AttemptStatusCompleted).
			Select("score").Find(&results)

		for _, result := range results {
			percentage := float64(result.Score) / float64(exam.TotalMarks) * 100
			grade := getGradeFromPercentage(percentage)
			gradeDist[grade]++
		}
	}

	passRate := 0.0
	if completedCount > 0 {
		passRate = float64(passedCount) / float64(completedCount) * 100
	}

	return map[string]interface{}{
		"total_attempts":     totalAttempts,
		"completed_count":    completedCount,
		"in_progress_count":  inProgressCount,
		"passed_count":       passedCount,
		"failed_count":       failedCount,
		"average_score":      avgScore,
		"pass_rate":          passRate,
		"grade_distribution": gradeDist,
	}, nil
}

// ============================================
// TEACHER VIEWS
// ============================================

// GetTeacherSubjectResults gets subject results for a teacher
func (r *ExamRepository) GetTeacherSubjectResults(ctx context.Context, subjectID, classID string) ([]models.ExamAttempt, error) {
	var attempts []models.ExamAttempt
	err := r.db.WithContext(ctx).
		Model(&models.ExamAttempt{}).
		Joins("JOIN exams ON exams.id = exam_attempts.exam_id").
		Joins("JOIN students ON students.id = exam_attempts.student_id").
		Joins("JOIN classes ON classes.id = students.class_id").
		Where("exams.subject_id = ? AND classes.id = ?", subjectID, classID).
		Where("exam_attempts.status = ?", models.AttemptStatusCompleted).
		Order("exam_attempts.created_at DESC").
		Find(&attempts).Error
	return attempts, err
}

// GetTeacherClassResults gets class results for a teacher
func (r *ExamRepository) GetTeacherClassResults(ctx context.Context, classID, subjectID string) ([]models.ExamAttempt, error) {
	var attempts []models.ExamAttempt
	query := r.db.WithContext(ctx).
		Model(&models.ExamAttempt{}).
		Joins("JOIN exams ON exams.id = exam_attempts.exam_id").
		Joins("JOIN students ON students.id = exam_attempts.student_id").
		Joins("JOIN classes ON classes.id = students.class_id").
		Where("classes.id = ?", classID).
		Where("exam_attempts.status = ?", models.AttemptStatusCompleted)

	if subjectID != "" {
		query = query.Where("exams.subject_id = ?", subjectID)
	}

	err := query.Order("exam_attempts.created_at DESC").
		Find(&attempts).Error
	return attempts, err
}

// GetUpcomingExams gets upcoming exams for a teacher
func (r *ExamRepository) GetUpcomingExams(ctx context.Context, teacherID string) ([]models.Exam, error) {
	var exams []models.Exam
	now := time.Now()

	err := r.db.WithContext(ctx).
		Model(&models.Exam{}).
		Where("start_time > ?", now).
		Where("is_active = ?", true).
		Order("start_time ASC").
		Limit(10).
		Find(&exams).Error
	return exams, err
}

// GetRecentActivities gets recent activities for a teacher
func (r *ExamRepository) GetRecentActivities(ctx context.Context, teacherID string, limit int) ([]models.ExamAttempt, error) {
	var attempts []models.ExamAttempt
	err := r.db.WithContext(ctx).
		Model(&models.ExamAttempt{}).
		Joins("JOIN exams ON exams.id = exam_attempts.exam_id").
		Where("exam_attempts.status = ?", models.AttemptStatusCompleted).
		Order("exam_attempts.updated_at DESC").
		Limit(limit).
		Find(&attempts).Error
	return attempts, err
}

// ============================================
// SCHOOL PERFORMANCE OVERVIEW
// ============================================

// GetSchoolPerformanceOverview gets school performance overview
func (r *ExamRepository) GetSchoolPerformanceOverview(ctx context.Context, schoolID string) (map[string]interface{}, error) {
	var totalStudents, totalTeachers, totalExams int64
	var overallPassRate float64

	// Total students
	r.db.WithContext(ctx).Model(&models.Student{}).
		Where("school_id = ? AND deleted_at IS NULL", schoolID).
		Count(&totalStudents)

	// Total teachers
	r.db.WithContext(ctx).Model(&models.User{}).
		Where("role = ? AND deleted_at IS NULL", "teacher").
		Count(&totalTeachers)

	// Total exams
	r.db.WithContext(ctx).Model(&models.Exam{}).
		Where("deleted_at IS NULL").Count(&totalExams)

	// Overall pass rate
	var totalCompleted int64
	r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
		Joins("JOIN exams ON exams.id = exam_attempts.exam_id").
		Where("exam_attempts.status = ?", models.AttemptStatusCompleted).
		Count(&totalCompleted)

	if totalCompleted > 0 {
		var passedCount int64
		r.db.WithContext(ctx).Model(&models.ExamAttempt{}).
			Joins("JOIN exams ON exams.id = exam_attempts.exam_id").
			Where("exam_attempts.status = ?", models.AttemptStatusCompleted).
			Where("exam_attempts.score >= exams.pass_mark").
			Count(&passedCount)
		overallPassRate = float64(passedCount) / float64(totalCompleted) * 100
	}

	return map[string]interface{}{
		"total_students":    totalStudents,
		"total_teachers":    totalTeachers,
		"total_exams":       totalExams,
		"overall_pass_rate": overallPassRate,
	}, nil
}

// ============================================
// CURRENT SESSION & TERM
// ============================================

// GetCurrentSessionAndTerm gets current session and term for a school
func (r *ExamRepository) GetCurrentSessionAndTerm(ctx context.Context, schoolID string) (*CurrentAcademicContext, error) {
	var session models.AcademicSession
	err := r.db.WithContext(ctx).
		Where("school_id = ? AND is_current = ? AND is_active = ?", schoolID, true, true).
		First(&session).Error
	if err != nil {
		return nil, err
	}

	var term models.Term
	err = r.db.WithContext(ctx).
		Where("session_id = ? AND is_current = ? AND is_active = ?", session.ID, true, true).
		First(&term).Error
	if err != nil {
		return nil, err
	}

	return &CurrentAcademicContext{
		SessionID:   session.ID,
		SessionName: session.Name,
		TermID:      term.ID,
		TermName:    term.Name,
		TermNumber:  term.TermNumber,
	}, nil
}

// ============================================
// STUDENT HELPERS
// ============================================

// GetStudentByID gets a student by ID
// func (r *ExamRepository) GetStudentByID(ctx context.Context, studentID string) (*models.Student, error) {
// 	var student models.Student
// 	err := r.db.WithContext(ctx).
// 		Where("id = ? AND deleted_at IS NULL", studentID).
// 		First(&student).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &student, nil
// }

// GetStudentByID - Updated to preload User
func (r *ExamRepository) GetStudentByID(ctx context.Context, studentID string) (*models.Student, error) {
    var student models.Student
    err := r.db.WithContext(ctx).
        Preload("User").  // ✅ ADD THIS - loads the User data
        Where("id = ? AND deleted_at IS NULL", studentID).
        First(&student).Error
    if err != nil {
        return nil, err
    }
    return &student, nil
}

// GetStudentsByClass gets students by class
func (r *ExamRepository) GetStudentsByClass(ctx context.Context, classID string) ([]models.Student, error) {
	var students []models.Student
	err := r.db.WithContext(ctx).
		Where("class_id = ? AND is_active = ? AND deleted_at IS NULL", classID, true).
		Find(&students).Error
	return students, err
}

// GetStudentsBySchool gets students by school
func (r *ExamRepository) GetStudentsBySchool(ctx context.Context, schoolID string) ([]models.Student, error) {
	var students []models.Student
	err := r.db.WithContext(ctx).
		Where("school_id = ? AND is_active = ? AND deleted_at IS NULL", schoolID, true).
		Find(&students).Error
	return students, err
}

// GetSubjectsByClassAndTerm gets subjects for a class and term
func (r *ExamRepository) GetSubjectsByClassAndTerm(ctx context.Context, classID, termID string) ([]models.Subject, error) {
	var subjects []models.Subject
	err := r.db.WithContext(ctx).
		Table("subjects").
		Joins("JOIN exams ON exams.subject_id = subjects.id").
		Where("exams.class_id = ? AND exams.term_id = ?", classID, termID).
		Where("subjects.deleted_at IS NULL").
		Distinct("subjects.id, subjects.name, subjects.code, subjects.is_active, subjects.created_at, subjects.updated_at").
		Find(&subjects).Error
	return subjects, err
}

// ============================================
// HELPER - Grade Calculation
// ============================================

func getGradeFromPercentage(percentage float64) string {
	switch {
	case percentage >= 70:
		return "A"
	case percentage >= 60:
		return "B"
	case percentage >= 50:
		return "C"
	case percentage >= 45:
		return "D"
	case percentage >= 40:
		return "E"
	default:
		return "F"
	}
}
// ============================================
// HELPER - Grade Calculation
// ============================================
// ============================================
// STUDENT HELPERS - ON ExamRepository
// ============================================

// GetStudentByUserID gets student by user ID
func (r *ExamRepository) GetStudentByUserID(ctx context.Context, userID string) (*models.Student, error) {
    var student models.Student
    err := r.db.WithContext(ctx).
        Preload("User").
        Where("user_id = ? AND deleted_at IS NULL", userID).
        First(&student).Error
    if err != nil {
        return nil, err
    }
    return &student, nil
}

// GetExamsByStudentUserID gets exams for a student by user ID
func (r *ExamRepository) GetExamsByStudentUserID(ctx context.Context, userID string) ([]models.Exam, error) {
    // First get the student
    student, err := r.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    return r.GetExamsByStudentAndClass(ctx, student.ID, student.ClassID)
}



