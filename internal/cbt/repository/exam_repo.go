package repository

import (
    "cbt-api/internal/models"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "time"
)

type ExamRepository struct {
    db *gorm.DB
}

func NewExamRepository(db *gorm.DB) *ExamRepository {
    return &ExamRepository{db: db}
}

// ============================================
// EXAM CRUD
// ============================================

func (r *ExamRepository) CreateExam(exam *models.Exam) error {
    return r.db.Create(exam).Error
}

func (r *ExamRepository) FindExamByID(id uuid.UUID) (*models.Exam, error) {
    var exam models.Exam
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&exam).Error
    return &exam, err
}

func (r *ExamRepository) UpdateExam(exam *models.Exam) error {
    return r.db.Save(exam).Error
}

func (r *ExamRepository) DeleteExam(id uuid.UUID) error {
    return r.db.Where("id = ?", id).Delete(&models.Exam{}).Error
}

func (r *ExamRepository) ListExamsBySubject(subjectID uuid.UUID, page, limit int) ([]models.Exam, int64, error) {
    var exams []models.Exam
    query := r.db.Model(&models.Exam{}).Where("subject_id = ? AND deleted_at IS NULL", subjectID)
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    offset := (page - 1) * limit
    err := query.Offset(offset).Limit(limit).Find(&exams).Error
    return exams, total, err
}

// ============================================
// EXAM QUESTIONS (pivot table)
// ============================================

func (r *ExamRepository) AddQuestionToExam(examID, questionID uuid.UUID, sortOrder int) error {
    eq := models.ExamQuestion{
        ID:         uuid.New(),
        ExamID:     examID,
        QuestionID: questionID,
        SortOrder:  sortOrder,
    }
    return r.db.Create(&eq).Error
}

func (r *ExamRepository) RemoveQuestionFromExam(examID, questionID uuid.UUID) error {
    return r.db.Where("exam_id = ? AND question_id = ?", examID, questionID).Delete(&models.ExamQuestion{}).Error
}

// GetExamQuestions returns only the questions (without the exam object)
func (r *ExamRepository) GetExamQuestions(examID uuid.UUID) ([]models.QuestionBank, error) {
    var questions []models.QuestionBank
    err := r.db.Table("question_bank").
        Joins("JOIN exam_questions ON exam_questions.question_id = question_bank.id").
        Where("exam_questions.exam_id = ? AND question_bank.deleted_at IS NULL", examID).
        Order("exam_questions.sort_order ASC").
        Find(&questions).Error
    return questions, err
}

// FindExamWithQuestions returns the exam together with its questions
func (r *ExamRepository) FindExamWithQuestions(examID uuid.UUID) (*models.Exam, []models.QuestionBank, error) {
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

// ============================================
// EXAM ATTEMPTS
// ============================================

func (r *ExamRepository) CreateAttempt(attempt *models.ExamAttempt) error {
    return r.db.Create(attempt).Error
}

func (r *ExamRepository) FindAttemptByID(id uuid.UUID) (*models.ExamAttempt, error) {
    var attempt models.ExamAttempt
    err := r.db.Where("id = ?", id).First(&attempt).Error
    return &attempt, err
}

func (r *ExamRepository) FindActiveAttempt(studentID, examID uuid.UUID) (*models.ExamAttempt, error) {
    var attempt models.ExamAttempt
    err := r.db.Where("student_id = ? AND exam_id = ? AND status = ?",
        studentID, examID, "in_progress").First(&attempt).Error
    if err != nil {
        return nil, err
    }
    return &attempt, nil
}

// FindByStudentID retrieves all exam attempts for a given student ID.
func (r *ExamRepository) FindByStudentID(studentID string) ([]models.ExamAttempt, error) {
    id, err := uuid.Parse(studentID)
    if err != nil {
        return nil, err
    }
    var attempts []models.ExamAttempt
    err = r.db.Where("student_id = ?", id).Order("created_at DESC").Find(&attempts).Error
    return attempts, err
}

func (r *ExamRepository) UpdateAttempt(attempt *models.ExamAttempt) error {
    return r.db.Save(attempt).Error
}

func (r *ExamRepository) UpdateAttemptStatus(id uuid.UUID, status string) error {
    return r.db.Model(&models.ExamAttempt{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ExamRepository) GetAnsweredCount(attemptID uuid.UUID) (int64, error) {
    var count int64
    err := r.db.Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count).Error
    return count, err
}

// ============================================
// STUDENT ANSWERS
// ============================================

func (r *ExamRepository) SaveAnswer(answer *models.StudentAnswer) error {
    return r.db.Create(answer).Error
}

func (r *ExamRepository) UpdateAnswer(answer *models.StudentAnswer) error {
    return r.db.Save(answer).Error
}

func (r *ExamRepository) FindAnswer(attemptID, questionID uuid.UUID) (*models.StudentAnswer, error) {
    var ans models.StudentAnswer
    err := r.db.Where("attempt_id = ? AND question_id = ?", attemptID, questionID).First(&ans).Error
    return &ans, err
}

func (r *ExamRepository) FindAnswersByAttempt(attemptID uuid.UUID) ([]models.StudentAnswer, error) {
    var answers []models.StudentAnswer
    err := r.db.Where("attempt_id = ?", attemptID).Find(&answers).Error
    return answers, err
}

// ============================================
// RESULTS
// ============================================

func (r *ExamRepository) CreateResult(result *models.Result) error {
    return r.db.Create(result).Error
}

func (r *ExamRepository) FindResultByExamAndStudent(examID, studentID uuid.UUID) (*models.Result, error) {
    var res models.Result
    err := r.db.Where("exam_id = ? AND student_id = ?", examID, studentID).First(&res).Error
    return &res, err
}

func (r *ExamRepository) FindResultsByStudent(studentID uuid.UUID) ([]models.Result, error) {
    var results []models.Result
    err := r.db.Where("student_id = ?", studentID).Order("created_at DESC").Find(&results).Error
    return results, err
}

// ============================================
// EXAM ASSIGNMENTS
// ============================================

func (r *ExamRepository) CreateAssignment(assignment *models.ExamAssignment) error {
    return r.db.Create(assignment).Error
}

func (r *ExamRepository) FindAssignmentsByExam(examID uuid.UUID) ([]models.ExamAssignment, error) {
    var assignments []models.ExamAssignment
    err := r.db.Where("exam_id = ?", examID).Find(&assignments).Error
    return assignments, err
}

func (r *ExamRepository) FindAssignmentsByStudent(studentID uuid.UUID) ([]models.ExamAssignment, error) {
    var assignments []models.ExamAssignment
    err := r.db.Where("student_id = ?", studentID).Find(&assignments).Error
    return assignments, err
}

// ============================================
// PRACTICE SESSION
// ============================================

func (r *ExamRepository) CreatePracticeSession(session *models.PracticeSession) error {
    return r.db.Create(session).Error
}

func (r *ExamRepository) FindPracticeSession(id uuid.UUID) (*models.PracticeSession, error) {
    var sess models.PracticeSession
    err := r.db.Where("id = ?", id).First(&sess).Error
    return &sess, err
}

func (r *ExamRepository) UpdatePracticeSession(session *models.PracticeSession) error {
    return r.db.Save(session).Error
}

func (r *ExamRepository) GetRandomQuestions(subjectID uuid.UUID, limit int) ([]models.QuestionBank, error) {
    var questions []models.QuestionBank
    err := r.db.Where("subject_id = ? AND deleted_at IS NULL", subjectID).
        Order("RANDOM()").Limit(limit).Find(&questions).Error
    return questions, err
}

// ============================================
// PROCTORING
// ============================================

func (r *ExamRepository) CreateProctoringSession(session *models.ProctoringSession) error {
    return r.db.Create(session).Error
}

func (r *ExamRepository) FindProctoringByAttempt(attemptID uuid.UUID) (*models.ProctoringSession, error) {
    var sess models.ProctoringSession
    err := r.db.Where("attempt_id = ?", attemptID).First(&sess).Error
    return &sess, err
}

func (r *ExamRepository) UpdateProctoringSession(session *models.ProctoringSession) error {
    return r.db.Save(session).Error
}

func (r *ExamRepository) CreateViolation(violation *models.ProctoringViolation) error {
    return r.db.Create(violation).Error
}

func (r *ExamRepository) GetViolationCount(proctoringID uuid.UUID) (int64, error) {
    var count int64
    err := r.db.Model(&models.ProctoringViolation{}).Where("proctoring_id = ?", proctoringID).Count(&count).Error
    return count, err
}

// ============================================
// OFFLINE ANSWERS
// ============================================

func (r *ExamRepository) SaveOfflineAnswer(answer *models.OfflineAnswer) error {
    return r.db.Create(answer).Error
}

func (r *ExamRepository) FindOfflineAnswers(studentID, examID uuid.UUID) ([]models.OfflineAnswer, error) {
    var answers []models.OfflineAnswer
    err := r.db.Where("student_id = ? AND exam_id = ? AND synced_at IS NULL", studentID, examID).
        Find(&answers).Error
    return answers, err
}

func (r *ExamRepository) MarkOfflineAnswersSynced(ids []uuid.UUID) error {
    return r.db.Model(&models.OfflineAnswer{}).
        Where("id IN ?", ids).
        Update("synced_at", time.Now()).Error
}






// package repository

// import (
//     "cbt-api/internal/models"
//     "github.com/google/uuid"
//     "gorm.io/gorm"
//     "time"
// )

// type ExamRepository struct {
//     db *gorm.DB
// }

// func NewExamRepository(db *gorm.DB) *ExamRepository {
//     return &ExamRepository{db: db}
// }

// // ============================================
// // EXAM CRUD
// // ============================================

// func (r *ExamRepository) CreateExam(exam *models.Exam) error {
//     return r.db.Create(exam).Error
// }

// func (r *ExamRepository) FindExamByID(id uuid.UUID) (*models.Exam, error) {
//     var exam models.Exam
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&exam).Error
//     return &exam, err
// }

// func (r *ExamRepository) UpdateExam(exam *models.Exam) error {
//     return r.db.Save(exam).Error
// }

// func (r *ExamRepository) DeleteExam(id uuid.UUID) error {
//     return r.db.Where("id = ?", id).Delete(&models.Exam{}).Error
// }

// func (r *ExamRepository) ListExamsBySubject(subjectID uuid.UUID, page, limit int) ([]models.Exam, int64, error) {
//     var exams []models.Exam
//     query := r.db.Model(&models.Exam{}).Where("subject_id = ? AND deleted_at IS NULL", subjectID)
//     var total int64
//     if err := query.Count(&total).Error; err != nil {
//         return nil, 0, err
//     }
//     offset := (page - 1) * limit
//     err := query.Offset(offset).Limit(limit).Find(&exams).Error
//     return exams, total, err
// }

// // ============================================
// // EXAM QUESTIONS (pivot table)
// // ============================================

// func (r *ExamRepository) AddQuestionToExam(examID, questionID uuid.UUID, sortOrder int) error {
//     eq := models.ExamQuestion{
//         ID:         uuid.New(),
//         ExamID:     examID,
//         QuestionID: questionID,
//         SortOrder:  sortOrder,
//     }
//     return r.db.Create(&eq).Error
// }

// func (r *ExamRepository) RemoveQuestionFromExam(examID, questionID uuid.UUID) error {
//     return r.db.Where("exam_id = ? AND question_id = ?", examID, questionID).Delete(&models.ExamQuestion{}).Error
// }

// func (r *ExamRepository) GetExamQuestions(examID uuid.UUID) ([]models.QuestionBank, error) {
//     var questions []models.QuestionBank
//     err := r.db.Table("question_bank").
//         Joins("JOIN exam_questions ON exam_questions.question_id = question_bank.id").
//         Where("exam_questions.exam_id = ? AND question_bank.deleted_at IS NULL", examID).
//         Order("exam_questions.sort_order ASC").
//         Find(&questions).Error
//     return questions, err
// }

// // ============================================
// // EXAM ATTEMPTS
// // ============================================

// func (r *ExamRepository) CreateAttempt(attempt *models.ExamAttempt) error {
//     return r.db.Create(attempt).Error
// }

// func (r *ExamRepository) FindAttemptByID(id uuid.UUID) (*models.ExamAttempt, error) {
//     var attempt models.ExamAttempt
//     err := r.db.Where("id = ?", id).First(&attempt).Error
//     return &attempt, err
// }

// func (r *ExamRepository) FindActiveAttempt(studentID, examID uuid.UUID) (*models.ExamAttempt, error) {
//     var attempt models.ExamAttempt
//     err := r.db.Where("student_id = ? AND exam_id = ? AND status = ?",
//         studentID, examID, "in_progress").First(&attempt).Error
//     if err != nil {
//         return nil, err
//     }
//     return &attempt, nil
// }

// func (r *ExamRepository) UpdateAttempt(attempt *models.ExamAttempt) error {
//     return r.db.Save(attempt).Error
// }

// func (r *ExamRepository) UpdateAttemptStatus(id uuid.UUID, status string) error {
//     return r.db.Model(&models.ExamAttempt{}).Where("id = ?", id).Update("status", status).Error
// }

// func (r *ExamRepository) GetAnsweredCount(attemptID uuid.UUID) (int64, error) {
//     var count int64
//     err := r.db.Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count).Error
//     return count, err
// }

// // ============================================
// // STUDENT ANSWERS
// // ============================================

// func (r *ExamRepository) SaveAnswer(answer *models.StudentAnswer) error {
//     return r.db.Create(answer).Error
// }

// func (r *ExamRepository) UpdateAnswer(answer *models.StudentAnswer) error {
//     return r.db.Save(answer).Error
// }

// func (r *ExamRepository) FindAnswer(attemptID, questionID uuid.UUID) (*models.StudentAnswer, error) {
//     var ans models.StudentAnswer
//     err := r.db.Where("attempt_id = ? AND question_id = ?", attemptID, questionID).First(&ans).Error
//     return &ans, err
// }

// func (r *ExamRepository) FindAnswersByAttempt(attemptID uuid.UUID) ([]models.StudentAnswer, error) {
//     var answers []models.StudentAnswer
//     err := r.db.Where("attempt_id = ?", attemptID).Find(&answers).Error
//     return answers, err
// }

// // ============================================
// // RESULTS
// // ============================================

// func (r *ExamRepository) CreateResult(result *models.Result) error {
//     return r.db.Create(result).Error
// }

// func (r *ExamRepository) FindResultByExamAndStudent(examID, studentID uuid.UUID) (*models.Result, error) {
//     var res models.Result
//     err := r.db.Where("exam_id = ? AND student_id = ?", examID, studentID).First(&res).Error
//     return &res, err
// }

// func (r *ExamRepository) FindResultsByStudent(studentID uuid.UUID) ([]models.Result, error) {
//     var results []models.Result
//     err := r.db.Where("student_id = ?", studentID).Order("created_at DESC").Find(&results).Error
//     return results, err
// }

// // ============================================
// // EXAM ASSIGNMENTS
// // ============================================

// func (r *ExamRepository) CreateAssignment(assignment *models.ExamAssignment) error {
//     return r.db.Create(assignment).Error
// }

// func (r *ExamRepository) FindAssignmentsByExam(examID uuid.UUID) ([]models.ExamAssignment, error) {
//     var assignments []models.ExamAssignment
//     err := r.db.Where("exam_id = ?", examID).Find(&assignments).Error
//     return assignments, err
// }

// func (r *ExamRepository) FindAssignmentsByStudent(studentID uuid.UUID) ([]models.ExamAssignment, error) {
//     var assignments []models.ExamAssignment
//     err := r.db.Where("student_id = ?", studentID).Find(&assignments).Error
//     return assignments, err
// }

// // ============================================
// // PRACTICE SESSION
// // ============================================

// func (r *ExamRepository) CreatePracticeSession(session *models.PracticeSession) error {
//     return r.db.Create(session).Error
// }

// func (r *ExamRepository) FindPracticeSession(id uuid.UUID) (*models.PracticeSession, error) {
//     var sess models.PracticeSession
//     err := r.db.Where("id = ?", id).First(&sess).Error
//     return &sess, err
// }

// func (r *ExamRepository) UpdatePracticeSession(session *models.PracticeSession) error {
//     return r.db.Save(session).Error
// }

// func (r *ExamRepository) GetRandomQuestions(subjectID uuid.UUID, limit int) ([]models.QuestionBank, error) {
//     var questions []models.QuestionBank
//     err := r.db.Where("subject_id = ? AND deleted_at IS NULL", subjectID).
//         Order("RANDOM()").Limit(limit).Find(&questions).Error
//     return questions, err
// }

// // ============================================
// // PROCTORING
// // ============================================

// func (r *ExamRepository) CreateProctoringSession(session *models.ProctoringSession) error {
//     return r.db.Create(session).Error
// }

// func (r *ExamRepository) FindProctoringByAttempt(attemptID uuid.UUID) (*models.ProctoringSession, error) {
//     var sess models.ProctoringSession
//     err := r.db.Where("attempt_id = ?", attemptID).First(&sess).Error
//     return &sess, err
// }

// func (r *ExamRepository) UpdateProctoringSession(session *models.ProctoringSession) error {
//     return r.db.Save(session).Error
// }

// func (r *ExamRepository) CreateViolation(violation *models.ProctoringViolation) error {
//     return r.db.Create(violation).Error
// }

// func (r *ExamRepository) GetViolationCount(proctoringID uuid.UUID) (int64, error) {
//     var count int64
//     err := r.db.Model(&models.ProctoringViolation{}).Where("proctoring_id = ?", proctoringID).Count(&count).Error
//     return count, err
// }

// // ============================================
// // OFFLINE ANSWERS
// // ============================================

// func (r *ExamRepository) SaveOfflineAnswer(answer *models.OfflineAnswer) error {
//     return r.db.Create(answer).Error
// }

// func (r *ExamRepository) FindOfflineAnswers(studentID, examID uuid.UUID) ([]models.OfflineAnswer, error) {
//     var answers []models.OfflineAnswer
//     err := r.db.Where("student_id = ? AND exam_id = ? AND synced_at IS NULL", studentID, examID).
//         Find(&answers).Error
//     return answers, err
// }

// func (r *ExamRepository) MarkOfflineAnswersSynced(ids []uuid.UUID) error {
//     return r.db.Model(&models.OfflineAnswer{}).
//         Where("id IN ?", ids).
//         Update("synced_at", time.Now()).Error
// }


// // FindExamWithQuestions retrieves an exam together with its associated questions.
// func (r *ExamRepository) FindExamWithQuestions(examID uuid.UUID) (*models.Exam, []models.QuestionBank, error) {
//     var exam models.Exam
//     err := r.db.Where("id = ? AND deleted_at IS NULL", examID).First(&exam).Error
//     if err != nil {
//         return nil, nil, err
//     }
//     var questions []models.QuestionBank
//     err = r.db.Table("question_bank").
//         Joins("JOIN exam_questions ON exam_questions.question_id = question_bank.id").
//         Where("exam_questions.exam_id = ? AND question_bank.deleted_at IS NULL", examID).
//         Order("exam_questions.sort_order ASC").
//         Find(&questions).Error
//     if err != nil {
//         return nil, nil, err
//     }
//     return &exam, questions, nil
// }


// // package repository

// // import (
// //     "cbt-api/internal/models"
// //     "github.com/google/uuid"
// //     "gorm.io/gorm"
// //     "time"
// // )

// // type ExamRepository struct {
// //     db *gorm.DB
// // }

// // func NewExamRepository(db *gorm.DB) *ExamRepository {
// //     return &ExamRepository{db: db}
// // }

// // // Exam CRUD
// // func (r *ExamRepository) CreateExam(exam *models.Exam) error {
// //     return r.db.Create(exam).Error
// // }

// // func (r *ExamRepository) FindExamByID(id uuid.UUID) (*models.Exam, error) {
// //     var exam models.Exam
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&exam).Error
// //     return &exam, err
// // }

// // func (r *ExamRepository) UpdateExam(exam *models.Exam) error {
// //     return r.db.Save(exam).Error
// // }

// // func (r *ExamRepository) DeleteExam(id uuid.UUID) error {
// //     return r.db.Where("id = ?", id).Delete(&models.Exam{}).Error
// // }

// // // Get exam with its questions
// // func (r *ExamRepository) FindExamWithQuestions(examID uuid.UUID) (*models.Exam, []models.Question, error) {
// //     var exam models.Exam
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", examID).First(&exam).Error
// //     if err != nil {
// //         return nil, nil, err
// //     }
// //     var questions []models.Question
// //     err = r.db.Where("exam_id = ? AND deleted_at IS NULL", examID).
// //         Order("sort_order ASC").Find(&questions).Error
// //     return &exam, questions, err
// // }

// // func (r *ExamRepository) AddQuestionToExam(examID, questionID uuid.UUID, sortOrder int) error {
// //     // Because Question model already has ExamID, just update the question.
// //     return r.db.Model(&models.Question{}).Where("id = ?", questionID).
// //         Updates(map[string]interface{}{
// //             "exam_id":    examID,
// //             "sort_order": sortOrder,
// //         }).Error
// // }

// // func (r *ExamRepository) RemoveQuestionFromExam(examID, questionID uuid.UUID) error {
// //     return r.db.Model(&models.Question{}).Where("id = ?", questionID).
// //         Update("exam_id", nil).Error
// // }

// // // Attempts
// // func (r *ExamRepository) CreateAttempt(attempt *models.ExamAttempt) error {
// //     return r.db.Create(attempt).Error
// // }

// // func (r *ExamRepository) FindAttemptByID(id uuid.UUID) (*models.ExamAttempt, error) {
// //     var attempt models.ExamAttempt
// //     err := r.db.Where("id = ?", id).First(&attempt).Error
// //     return &attempt, err
// // }

// // func (r *ExamRepository) FindActiveAttempt(studentID, examID uuid.UUID) (*models.ExamAttempt, error) {
// //     var attempt models.ExamAttempt
// //     err := r.db.Where("student_id = ? AND exam_id = ? AND status = ?",
// //         studentID, examID, "in_progress").First(&attempt).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &attempt, nil
// // }

// // func (r *ExamRepository) UpdateAttempt(attempt *models.ExamAttempt) error {
// //     return r.db.Save(attempt).Error
// // }

// // func (r *ExamRepository) GetAnsweredCount(attemptID uuid.UUID) (int64, error) {
// //     var count int64
// //     err := r.db.Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count).Error
// //     return count, err
// // }

// // // Answers
// // func (r *ExamRepository) SaveAnswer(answer *models.StudentAnswer) error {
// //     return r.db.Create(answer).Error
// // }

// // func (r *ExamRepository) UpdateAnswer(answer *models.StudentAnswer) error {
// //     return r.db.Save(answer).Error
// // }

// // func (r *ExamRepository) FindAnswer(attemptID, questionID uuid.UUID) (*models.StudentAnswer, error) {
// //     var ans models.StudentAnswer
// //     err := r.db.Where("attempt_id = ? AND question_id = ?", attemptID, questionID).First(&ans).Error
// //     return &ans, err
// // }

// // func (r *ExamRepository) FindAnswersByAttempt(attemptID uuid.UUID) ([]models.StudentAnswer, error) {
// //     var answers []models.StudentAnswer
// //     err := r.db.Where("attempt_id = ?", attemptID).Find(&answers).Error
// //     return answers, err
// // }

// // // Results
// // func (r *ExamRepository) CreateResult(result *models.Result) error {
// //     return r.db.Create(result).Error
// // }

// // func (r *ExamRepository) FindResultByExamAndStudent(examID, studentID uuid.UUID) (*models.Result, error) {
// //     var res models.Result
// //     err := r.db.Where("exam_id = ? AND student_id = ?", examID, studentID).First(&res).Error
// //     return &res, err
// // }

// // // Practice sessions
// // func (r *ExamRepository) CreatePracticeSession(session *models.PracticeSession) error {
// //     return r.db.Create(session).Error
// // }

// // func (r *ExamRepository) FindPracticeSession(id uuid.UUID) (*models.PracticeSession, error) {
// //     var sess models.PracticeSession
// //     err := r.db.Where("id = ?", id).First(&sess).Error
// //     return &sess, err
// // }

// // func (r *ExamRepository) UpdatePracticeSession(session *models.PracticeSession) error {
// //     return r.db.Save(session).Error
// // }

// // func (r *ExamRepository) GetRandomQuestionsFromBank(subjectID uuid.UUID, limit int) ([]models.QuestionBank, error) {
// //     var questions []models.QuestionBank
// //     err := r.db.Where("subject_id = ? AND deleted_at IS NULL", subjectID).
// //         Order("RANDOM()").Limit(limit).Find(&questions).Error
// //     return questions, err
// // }

// // // Proctoring
// // func (r *ExamRepository) CreateProctoringSession(session *models.ProctoringSession) error {
// //     return r.db.Create(session).Error
// // }

// // func (r *ExamRepository) FindProctoringByAttempt(attemptID uuid.UUID) (*models.ProctoringSession, error) {
// //     var sess models.ProctoringSession
// //     err := r.db.Where("attempt_id = ?", attemptID).First(&sess).Error
// //     return &sess, err
// // }

// // func (r *ExamRepository) UpdateProctoringSession(session *models.ProctoringSession) error {
// //     return r.db.Save(session).Error
// // }

// // func (r *ExamRepository) CreateViolation(violation *models.ProctoringViolation) error {
// //     return r.db.Create(violation).Error
// // }

// // func (r *ExamRepository) GetViolationCount(proctoringID uuid.UUID) (int64, error) {
// //     var count int64
// //     err := r.db.Model(&models.ProctoringViolation{}).Where("proctoring_id = ?", proctoringID).Count(&count).Error
// //     return count, err
// // }

// // // Offline answers
// // func (r *ExamRepository) SaveOfflineAnswer(answer *models.OfflineAnswer) error {
// //     return r.db.Create(answer).Error
// // }

// // func (r *ExamRepository) FindOfflineAnswers(studentID, examID uuid.UUID) ([]models.OfflineAnswer, error) {
// //     var answers []models.OfflineAnswer
// //     err := r.db.Where("student_id = ? AND exam_id = ? AND synced_at IS NULL", studentID, examID).
// //         Find(&answers).Error
// //     return answers, err
// // }

// // func (r *ExamRepository) MarkOfflineAnswersSynced(ids []uuid.UUID) error {
// //     return r.db.Model(&models.OfflineAnswer{}).
// //         Where("id IN ?", ids).
// //         Update("synced_at", time.Now()).Error
// // }

// // // Assignments
// // func (r *ExamRepository) CreateAssignment(assignment *models.ExamAssignment) error {
// //     return r.db.Create(assignment).Error
// // }

// // func (r *ExamRepository) FindAssignmentsByExam(examID uuid.UUID) ([]models.ExamAssignment, error) {
// //     var assignments []models.ExamAssignment
// //     err := r.db.Where("exam_id = ?", examID).Find(&assignments).Error
// //     return assignments, err
// // }

// // func (r *ExamRepository) FindAssignmentsByStudent(studentID uuid.UUID) ([]models.ExamAssignment, error) {
// //     var assignments []models.ExamAssignment
// //     err := r.db.Where("student_id = ?", studentID).Find(&assignments).Error
// //     return assignments, err
// // }


// // package repository

// // import (
// //     "cbt-api/internal/models"
// //     "github.com/google/uuid"
// //     "gorm.io/gorm"
// //     "time"
// // )

// // type ExamRepository struct {
// //     db *gorm.DB
// // }

// // func NewExamRepository(db *gorm.DB) *ExamRepository {
// //     return &ExamRepository{db: db}
// // }

// // // ============================================
// // // EXAM CRUD
// // // ============================================

// // func (r *ExamRepository) CreateExam(exam *models.Exam) error {
// //     return r.db.Create(exam).Error
// // }

// // func (r *ExamRepository) FindExamByID(id uuid.UUID) (*models.Exam, error) {
// //     var exam models.Exam
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&exam).Error
// //     return &exam, err
// // }

// // func (r *ExamRepository) UpdateExam(exam *models.Exam) error {
// //     return r.db.Save(exam).Error
// // }

// // func (r *ExamRepository) DeleteExam(id uuid.UUID) error {
// //     return r.db.Where("id = ?", id).Delete(&models.Exam{}).Error
// // }

// // func (r *ExamRepository) ListExamsBySubject(subjectID uuid.UUID, page, limit int) ([]models.Exam, int64, error) {
// //     var exams []models.Exam
// //     query := r.db.Model(&models.Exam{}).Where("subject_id = ? AND deleted_at IS NULL", subjectID)
// //     var total int64
// //     if err := query.Count(&total).Error; err != nil {
// //         return nil, 0, err
// //     }
// //     offset := (page - 1) * limit
// //     err := query.Offset(offset).Limit(limit).Find(&exams).Error
// //     return exams, total, err
// // }

// // // ============================================
// // // EXAM QUESTIONS
// // // ============================================

// // func (r *ExamRepository) AddQuestionToExam(examID, questionID uuid.UUID, sortOrder int) error {
// //     eq := models.ExamQuestion{
// //         ExamID:     examID,
// //         QuestionID: questionID,
// //         SortOrder:  sortOrder,
// //     }
// //     return r.db.Create(&eq).Error
// // }

// // func (r *ExamRepository) RemoveQuestionFromExam(examID, questionID uuid.UUID) error {
// //     return r.db.Where("exam_id = ? AND question_id = ?", examID, questionID).Delete(&models.ExamQuestion{}).Error
// // }

// // func (r *ExamRepository) GetExamQuestions(examID uuid.UUID) ([]models.Question, error) {
// //     var questions []models.Question
// //     err := r.db.Table("questions").
// //         Joins("JOIN exam_questions ON exam_questions.question_id = questions.id").
// //         Where("exam_questions.exam_id = ? AND questions.deleted_at IS NULL", examID).
// //         Order("exam_questions.sort_order ASC").
// //         Find(&questions).Error
// //     return questions, err
// // }

// // // ============================================
// // // EXAM ATTEMPTS
// // // ============================================

// // func (r *ExamRepository) CreateAttempt(attempt *models.ExamAttempt) error {
// //     return r.db.Create(attempt).Error
// // }

// // func (r *ExamRepository) FindAttemptByID(id uuid.UUID) (*models.ExamAttempt, error) {
// //     var attempt models.ExamAttempt
// //     err := r.db.Where("id = ?", id).First(&attempt).Error
// //     return &attempt, err
// // }

// // func (r *ExamRepository) FindActiveAttempt(studentID, examID uuid.UUID) (*models.ExamAttempt, error) {
// //     var attempt models.ExamAttempt
// //     err := r.db.Where("student_id = ? AND exam_id = ? AND status = ?",
// //         studentID, examID, "in_progress").First(&attempt).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &attempt, nil
// // }

// // func (r *ExamRepository) UpdateAttempt(attempt *models.ExamAttempt) error {
// //     return r.db.Save(attempt).Error
// // }

// // func (r *ExamRepository) GetAnsweredCount(attemptID uuid.UUID) (int64, error) {
// //     var count int64
// //     err := r.db.Model(&models.StudentAnswer{}).Where("attempt_id = ?", attemptID).Count(&count).Error
// //     return count, err
// // }

// // // ============================================
// // // STUDENT ANSWERS
// // // ============================================

// // func (r *ExamRepository) SaveAnswer(answer *models.StudentAnswer) error {
// //     return r.db.Create(answer).Error
// // }

// // func (r *ExamRepository) FindAnswer(attemptID, questionID uuid.UUID) (*models.StudentAnswer, error) {
// //     var ans models.StudentAnswer
// //     err := r.db.Where("attempt_id = ? AND question_id = ?", attemptID, questionID).First(&ans).Error
// //     return &ans, err
// // }

// // func (r *ExamRepository) FindAnswersByAttempt(attemptID uuid.UUID) ([]models.StudentAnswer, error) {
// //     var answers []models.StudentAnswer
// //     err := r.db.Where("attempt_id = ?", attemptID).Find(&answers).Error
// //     return answers, err
// // }

// // // ============================================
// // // RESULTS
// // // ============================================

// // func (r *ExamRepository) CreateResult(result *models.Result) error {
// //     return r.db.Create(result).Error
// // }

// // func (r *ExamRepository) FindResultByExamAndStudent(examID, studentID uuid.UUID) (*models.Result, error) {
// //     var res models.Result
// //     err := r.db.Where("exam_id = ? AND student_id = ?", examID, studentID).First(&res).Error
// //     return &res, err
// // }

// // // ============================================
// // // EXAM ASSIGNMENTS
// // // ============================================

// // func (r *ExamRepository) CreateAssignment(assignment *models.ExamAssignment) error {
// //     return r.db.Create(assignment).Error
// // }

// // func (r *ExamRepository) FindAssignmentsByExam(examID uuid.UUID) ([]models.ExamAssignment, error) {
// //     var assignments []models.ExamAssignment
// //     err := r.db.Where("exam_id = ?", examID).Find(&assignments).Error
// //     return assignments, err
// // }

// // func (r *ExamRepository) FindAssignmentsByStudent(studentID uuid.UUID) ([]models.ExamAssignment, error) {
// //     var assignments []models.ExamAssignment
// //     err := r.db.Where("student_id = ?", studentID).Find(&assignments).Error
// //     return assignments, err
// // }

// // // ============================================
// // // PRACTICE SESSION
// // // ============================================

// // func (r *ExamRepository) CreatePracticeSession(session *models.PracticeSession) error {
// //     return r.db.Create(session).Error
// // }

// // func (r *ExamRepository) FindPracticeSession(id uuid.UUID) (*models.PracticeSession, error) {
// //     var sess models.PracticeSession
// //     err := r.db.Where("id = ?", id).First(&sess).Error
// //     return &sess, err
// // }

// // func (r *ExamRepository) UpdatePracticeSession(session *models.PracticeSession) error {
// //     return r.db.Save(session).Error
// // }

// // func (r *ExamRepository) GetRandomQuestions(subjectID uuid.UUID, limit int) ([]models.Question, error) {
// //     var questions []models.Question
// //     err := r.db.Where("subject_id = ? AND deleted_at IS NULL", subjectID).
// //         Order("RANDOM()").Limit(limit).Find(&questions).Error
// //     return questions, err
// // }

// // // ============================================
// // // PROCTORING
// // // ============================================

// // func (r *ExamRepository) CreateProctoringSession(session *models.ProctoringSession) error {
// //     return r.db.Create(session).Error
// // }

// // func (r *ExamRepository) FindProctoringByAttempt(attemptID uuid.UUID) (*models.ProctoringSession, error) {
// //     var sess models.ProctoringSession
// //     err := r.db.Where("attempt_id = ?", attemptID).First(&sess).Error
// //     return &sess, err
// // }

// // func (r *ExamRepository) UpdateProctoringSession(session *models.ProctoringSession) error {
// //     return r.db.Save(session).Error
// // }

// // func (r *ExamRepository) CreateViolation(violation *models.ProctoringViolation) error {
// //     return r.db.Create(violation).Error
// // }

// // func (r *ExamRepository) GetViolationCount(proctoringID uuid.UUID) (int64, error) {
// //     var count int64
// //     err := r.db.Model(&models.ProctoringViolation{}).Where("proctoring_id = ?", proctoringID).Count(&count).Error
// //     return count, err
// // }

// // // ============================================
// // // OFFLINE ANSWERS
// // // ============================================

// // func (r *ExamRepository) SaveOfflineAnswer(answer *models.OfflineAnswer) error {
// //     return r.db.Create(answer).Error
// // }

// // func (r *ExamRepository) FindOfflineAnswers(studentID, examID uuid.UUID) ([]models.OfflineAnswer, error) {
// //     var answers []models.OfflineAnswer
// //     err := r.db.Where("student_id = ? AND exam_id = ? AND synced_at IS NULL", studentID, examID).
// //         Find(&answers).Error
// //     return answers, err
// // }

// // func (r *ExamRepository) MarkOfflineAnswersSynced(ids []uuid.UUID) error {
// //     return r.db.Model(&models.OfflineAnswer{}).
// //         Where("id IN ?", ids).
// //         Update("synced_at", time.Now()).Error
// // }




// // package repository

// // import (
// //     "cbt-api/internal/models"
// //     "github.com/google/uuid"
// //     "gorm.io/gorm"
// //     "time"
// // )

// // type ExamRepository struct {
// //     db *gorm.DB
// // }

// // func NewExamRepository(db *gorm.DB) *ExamRepository {
// //     return &ExamRepository{db: db}
// // }

// // // ============================================
// // // EXAM OPERATIONS
// // // ============================================

// // func (r *ExamRepository) FindByID(id uuid.UUID) (*models.Exam, error) {
// //     var exam models.Exam
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&exam).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &exam, nil
// // }

// // func (r *ExamRepository) FindExamWithQuestions(examID uuid.UUID) (*models.Exam, []models.Question, error) {
// //     var exam models.Exam
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", examID).First(&exam).Error
// //     if err != nil {
// //         return nil, nil, err
// //     }
    
// //     var questions []models.Question
// //     err = r.db.Where("exam_id = ? AND deleted_at IS NULL", examID).
// //         Order("sort_order ASC").Find(&questions).Error
// //     if err != nil {
// //         return nil, nil, err
// //     }
    
// //     return &exam, questions, nil
// // }

// // func (r *ExamRepository) FindQuestionByID(id uuid.UUID) (*models.Question, error) {
// //     var question models.Question
// //     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&question).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &question, nil
// // }

// // // ============================================
// // // EXAM ATTEMPT OPERATIONS
// // // ============================================

// // func (r *ExamRepository) CreateAttempt(attempt *models.ExamAttempt) error {
// //     return r.db.Create(attempt).Error
// // }

// // func (r *ExamRepository) FindAttemptByID(id uuid.UUID) (*models.ExamAttempt, error) {
// //     var attempt models.ExamAttempt
// //     err := r.db.Where("id = ?", id).First(&attempt).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &attempt, nil
// // }

// // func (r *ExamRepository) FindActiveAttempt(studentID, examID uuid.UUID) (*models.ExamAttempt, error) {
// //     var attempt models.ExamAttempt
// //     err := r.db.Where("student_id = ? AND exam_id = ? AND status = ?", 
// //         studentID, examID, "in_progress").
// //         First(&attempt).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &attempt, nil
// // }

// // func (r *ExamRepository) FindAttemptsByStudent(studentID uuid.UUID) ([]models.ExamAttempt, error) {
// //     var attempts []models.ExamAttempt
// //     err := r.db.Where("student_id = ?", studentID).
// //         Order("created_at DESC").Find(&attempts).Error
// //     return attempts, err
// // }

// // func (r *ExamRepository) UpdateAttempt(attempt *models.ExamAttempt) error {
// //     return r.db.Save(attempt).Error
// // }

// // func (r *ExamRepository) UpdateAttemptStatus(id uuid.UUID, status string) error {
// //     return r.db.Model(&models.ExamAttempt{}).Where("id = ?", id).
// //         Update("status", status).Error
// // }

// // func (r *ExamRepository) GetAnsweredCount(attemptID uuid.UUID) (int64, error) {
// //     var count int64
// //     err := r.db.Model(&models.StudentAnswer{}).
// //         Where("attempt_id = ?", attemptID).
// //         Count(&count).Error
// //     return count, err
// // }

// // // ============================================
// // // STUDENT ANSWER OPERATIONS
// // // ============================================

// // func (r *ExamRepository) SaveAnswer(answer *models.StudentAnswer) error {
// //     return r.db.Create(answer).Error
// // }

// // func (r *ExamRepository) UpdateAnswer(answer *models.StudentAnswer) error {
// //     return r.db.Save(answer).Error
// // }

// // func (r *ExamRepository) FindAnswer(attemptID, questionID uuid.UUID) (*models.StudentAnswer, error) {
// //     var answer models.StudentAnswer
// //     err := r.db.Where("attempt_id = ? AND question_id = ?", attemptID, questionID).
// //         First(&answer).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &answer, nil
// // }

// // func (r *ExamRepository) FindAnswersByAttempt(attemptID uuid.UUID) ([]models.StudentAnswer, error) {
// //     var answers []models.StudentAnswer
// //     err := r.db.Where("attempt_id = ?", attemptID).Find(&answers).Error
// //     return answers, err
// // }

// // // ============================================
// // // RESULT OPERATIONS
// // // ============================================

// // func (r *ExamRepository) CreateResult(result *models.Result) error {
// //     return r.db.Create(result).Error
// // }

// // func (r *ExamRepository) FindResultByExamAndStudent(examID, studentID uuid.UUID) (*models.Result, error) {
// //     var result models.Result
// //     err := r.db.Where("exam_id = ? AND student_id = ?", examID, studentID).
// //         First(&result).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &result, nil
// // }

// // func (r *ExamRepository) FindResultsByStudent(studentID uuid.UUID) ([]models.Result, error) {
// //     var results []models.Result
// //     err := r.db.Where("student_id = ?", studentID).
// //         Order("created_at DESC").Find(&results).Error
// //     return results, err
// // }

// // // ============================================
// // // PRACTICE SESSION OPERATIONS
// // // ============================================

// // func (r *ExamRepository) CreatePracticeSession(session *models.PracticeSession) error {
// //     return r.db.Create(session).Error
// // }

// // func (r *ExamRepository) FindPracticeSession(id uuid.UUID) (*models.PracticeSession, error) {
// //     var session models.PracticeSession
// //     err := r.db.Where("id = ?", id).First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &session, nil
// // }

// // func (r *ExamRepository) UpdatePracticeSession(session *models.PracticeSession) error {
// //     return r.db.Save(session).Error
// // }

// // func (r *ExamRepository) GetRandomQuestions(subjectID uuid.UUID, limit int) ([]models.Question, error) {
// //     var questions []models.Question
// //     err := r.db.Where("subject_id = ? AND deleted_at IS NULL", subjectID).
// //         Order("RANDOM()").
// //         Limit(limit).
// //         Find(&questions).Error
// //     return questions, err
// // }

// // // ============================================
// // // PROCTORING OPERATIONS
// // // ============================================

// // func (r *ExamRepository) CreateProctoringSession(session *models.ProctoringSession) error {
// //     return r.db.Create(session).Error
// // }

// // func (r *ExamRepository) FindProctoringByAttempt(attemptID uuid.UUID) (*models.ProctoringSession, error) {
// //     var session models.ProctoringSession
// //     err := r.db.Where("attempt_id = ?", attemptID).First(&session).Error
// //     if err != nil {
// //         return nil, err
// //     }
// //     return &session, nil
// // }

// // func (r *ExamRepository) UpdateProctoringSession(session *models.ProctoringSession) error {
// //     return r.db.Save(session).Error
// // }

// // func (r *ExamRepository) CreateViolation(violation *models.ProctoringViolation) error {
// //     return r.db.Create(violation).Error
// // }

// // func (r *ExamRepository) GetViolationCount(proctoringID uuid.UUID) (int64, error) {
// //     var count int64
// //     err := r.db.Model(&models.ProctoringViolation{}).
// //         Where("proctoring_id = ?", proctoringID).
// //         Count(&count).Error
// //     return count, err
// // }

// // // ============================================
// // // OFFLINE ANSWER OPERATIONS
// // // ============================================

// // func (r *ExamRepository) SaveOfflineAnswer(answer *models.OfflineAnswer) error {
// //     return r.db.Create(answer).Error
// // }

// // func (r *ExamRepository) FindOfflineAnswers(studentID, examID uuid.UUID) ([]models.OfflineAnswer, error) {
// //     var answers []models.OfflineAnswer
// //     err := r.db.Where("student_id = ? AND exam_id = ? AND synced_at IS NULL", studentID, examID).
// //         Find(&answers).Error
// //     return answers, err
// // }

// // func (r *ExamRepository) MarkOfflineAnswersSynced(ids []uuid.UUID) error {
// //     return r.db.Model(&models.OfflineAnswer{}).
// //         Where("id IN ?", ids).
// //         Update("synced_at", time.Now()).Error
// // }

// // // ============================================
// // // EXAM ASSIGNMENT OPERATIONS
// // // ============================================

// // func (r *ExamRepository) CreateAssignment(assignment *models.ExamAssignment) error {
// //     return r.db.Create(assignment).Error
// // }

// // func (r *ExamRepository) FindAssignmentsByExam(examID uuid.UUID) ([]models.ExamAssignment, error) {
// //     var assignments []models.ExamAssignment
// //     err := r.db.Where("exam_id = ?", examID).Find(&assignments).Error
// //     return assignments, err
// // }

// // func (r *ExamRepository) FindAssignmentsByStudent(studentID uuid.UUID) ([]models.ExamAssignment, error) {
// //     var assignments []models.ExamAssignment
// //     err := r.db.Where("student_id = ?", studentID).Find(&assignments).Error
// //     return assignments, err
// // }