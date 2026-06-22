package service

import (
    "cbt-api/internal/cbt/dto"
    "cbt-api/internal/cbt/repository"
    "cbt-api/internal/models"
    "context"
    "errors"
    // "fmt"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

type ExamService struct {
    examRepo     *repository.ExamRepository
    questionRepo *repository.QuestionRepository
    db           *gorm.DB
}

func NewExamService(examRepo *repository.ExamRepository, questionRepo *repository.QuestionRepository, db *gorm.DB) *ExamService {
    return &ExamService{
        examRepo:     examRepo,
        questionRepo: questionRepo,
        db:           db,
    }
}

// ============================================
// EXAM ATTEMPT & TAKING
// ============================================

func (s *ExamService) StartExam(ctx context.Context, req *dto.StartExamRequest) (*dto.StartExamResponse, error) {
    examID, err := uuid.Parse(req.ExamID)
    if err != nil {
        return nil, errors.New("invalid exam ID")
    }
    studentID, err := uuid.Parse(req.StudentID)
    if err != nil {
        return nil, errors.New("invalid student ID")
    }

    // Check for existing active attempt
    existing, _ := s.examRepo.FindActiveAttempt(studentID, examID)
    if existing != nil {
        return s.buildStartExamResponse(existing)
    }

    // Load exam and its questions
    // exam, questions, err := s.examRepo.FindExamWithQuestions(examID)
    // if err != nil {
    //     return nil, errors.New("exam not found")
    // }
	// Load exam and its questions
	exam, _, err := s.examRepo.FindExamWithQuestions(examID)
	if err != nil {
		return nil, errors.New("exam not found")
	}

    // Validate exam schedule
    now := time.Now()
    if exam.StartTime != nil && now.Before(*exam.StartTime) {
        return nil, errors.New("exam has not started yet")
    }
    if exam.EndTime != nil && now.After(*exam.EndTime) {
        return nil, errors.New("exam has already ended")
    }

    // Create attempt
    attempt := &models.ExamAttempt{
        ID:         uuid.New(),
        StudentID:  studentID,
        ExamID:     examID,
        StartTime:  now,
        Status:     "in_progress",
    }
    if err := s.examRepo.CreateAttempt(attempt); err != nil {
        return nil, err
    }

    // Create proctoring session if needed (assume exam has a boolean field EnableProctoring; if not, skip)
    // For now, we skip because your Exam model does not have that field.
    // proctoringID := uuid.New()
    // if exam.EnableProctoring { ... }

    return s.buildStartExamResponse(attempt)
}

func (s *ExamService) buildStartExamResponse(attempt *models.ExamAttempt) (*dto.StartExamResponse, error) {
    exam, questions, err := s.examRepo.FindExamWithQuestions(attempt.ExamID)
    if err != nil {
        return nil, err
    }
    totalQ := len(questions)
    answered, _ := s.examRepo.GetAnsweredCount(attempt.ID)
    timeRemaining := calculateRemainingTime(attempt.StartTime, exam.DurationMinutes)

    examDetail := &dto.ExamDetailResponse{
        ID:               exam.ID.String(),
        Title:            exam.Title,
        SubjectID:        exam.SubjectID.String(),
        DurationMinutes:  exam.DurationMinutes,
        TotalMarks:       exam.TotalMarks,
        PassMark:         exam.PassMark,
        Instructions:     exam.Instructions,
        StartTime:        exam.StartTime,
        EndTime:          exam.EndTime,
        ShuffleQuestions: exam.ShuffleQuestions,
        ShuffleOptions:   exam.ShuffleOptions,
    }

    var qResponses []dto.QuestionResponse
    for i, q := range questions {
        qResponses = append(qResponses, dto.QuestionResponse{
            ID:           q.ID.String(),
            QuestionText: q.QuestionText,
            OptionA:      extractOption(q.Options, "A"),
            OptionB:      extractOption(q.Options, "B"),
            OptionC:      extractOption(q.Options, "C"),
            OptionD:      extractOption(q.Options, "D"),
            Marks:        q.Marks,
            SortOrder:    i + 1,
        })
    }

    attemptResp := &dto.ExamAttemptResponse{
        ID:             attempt.ID.String(),
        StudentID:      attempt.StudentID.String(),
        ExamID:         attempt.ExamID.String(),
        StartTime:      attempt.StartTime,
        Status:         attempt.Status,
        TimeRemaining:  timeRemaining,
        AnsweredCount:  int(answered),
        TotalQuestions: totalQ,
        CreatedAt:      attempt.CreatedAt,
    }

    proctoringID := ""
    proctoring, _ := s.examRepo.FindProctoringByAttempt(attempt.ID)
    if proctoring != nil {
        proctoringID = proctoring.ID.String()
    }

    return &dto.StartExamResponse{
        Attempt:      attemptResp,
        Exam:         examDetail,
        Questions:    qResponses,
        ProctoringID: proctoringID,
    }, nil
}

func (s *ExamService) SaveAnswer(ctx context.Context, req *dto.SaveAnswerRequest) (*dto.SaveAnswerResponse, error) {
    attemptID, err := uuid.Parse(req.AttemptID)
    if err != nil {
        return nil, errors.New("invalid attempt ID")
    }
    questionID, err := uuid.Parse(req.QuestionID)
    if err != nil {
        return nil, errors.New("invalid question ID")
    }

    attempt, err := s.examRepo.FindAttemptByID(attemptID)
    if err != nil {
        return nil, errors.New("attempt not found")
    }
    if attempt.Status != "in_progress" {
        return nil, errors.New("exam already submitted")
    }

    exam, err := s.examRepo.FindExamByID(attempt.ExamID)
    if err != nil {
        return nil, err
    }
    if time.Since(attempt.StartTime).Minutes() > float64(exam.DurationMinutes) {
        return nil, errors.New("time limit exceeded")
    }

    q, err := s.questionRepo.FindByID(questionID)
    if err != nil {
        return nil, errors.New("question not found")
    }

    isCorrect := (req.SelectedAnswer == q.CorrectAnswer)

    // Upsert answer
    existing, _ := s.examRepo.FindAnswer(attemptID, questionID)
    if existing != nil {
        existing.SelectedAnswer = req.SelectedAnswer
        existing.IsCorrect = isCorrect
        existing.TimeSpent = req.TimeSpent
        if err := s.examRepo.UpdateAnswer(existing); err != nil {
            return nil, err
        }
    } else {
        answer := &models.StudentAnswer{
            ID:             uuid.New(),
            AttemptID:      attemptID,
            QuestionID:     questionID,
            SelectedAnswer: req.SelectedAnswer,
            IsCorrect:      isCorrect,
            TimeSpent:      req.TimeSpent,
        }
        if err := s.examRepo.SaveAnswer(answer); err != nil {
            return nil, err
        }
    }

    return &dto.SaveAnswerResponse{
        IsCorrect:     isCorrect,
        CorrectAnswer: q.CorrectAnswer,
        Explanation:   q.Explanation,
    }, nil
}

func (s *ExamService) SubmitExam(ctx context.Context, req *dto.SubmitExamRequest) (*dto.SubmitExamResponse, error) {
    attemptID, err := uuid.Parse(req.AttemptID)
    if err != nil {
        return nil, errors.New("invalid attempt ID")
    }

    attempt, err := s.examRepo.FindAttemptByID(attemptID)
    if err != nil {
        return nil, errors.New("attempt not found")
    }
    if attempt.Status != "in_progress" {
        return nil, errors.New("exam already submitted")
    }

    tx := s.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    answers, err := s.examRepo.FindAnswersByAttempt(attemptID)
    if err != nil {
        tx.Rollback()
        return nil, err
    }

    // Calculate total score by fetching each question's marks
    var totalScore int
    for _, ans := range answers {
        if ans.IsCorrect {
            q, err := s.questionRepo.FindByID(ans.QuestionID)
            if err != nil {
                tx.Rollback()
                return nil, err
            }
            totalScore += q.Marks
        }
    }

    now := time.Now()
    attempt.Status = "submitted"
    attempt.EndTime = &now
    attempt.Score = &totalScore
    if err := s.examRepo.UpdateAttempt(attempt); err != nil {
        tx.Rollback()
        return nil, err
    }

    exam, err := s.examRepo.FindExamByID(attempt.ExamID)
    if err != nil {
        tx.Rollback()
        return nil, err
    }

    percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
    passed := percentage >= float64(exam.PassMark)
    grade := calculateGrade(percentage)

    result := &models.Result{
        ID:         uuid.New(),
        ExamID:     attempt.ExamID,
        StudentID:  attempt.StudentID,
        TotalScore: totalScore,
        Percentage: percentage,
        Grade:      grade,
        Remarks:    "",
        PublishedAt: &now,
    }
    if err := s.examRepo.CreateResult(result); err != nil {
        tx.Rollback()
        return nil, err
    }

    if err := tx.Commit().Error; err != nil {
        return nil, err
    }

    return &dto.SubmitExamResponse{
        AttemptID:  attemptID.String(),
        Score:      totalScore,
        TotalMarks: exam.TotalMarks,
        Percentage: percentage,
        Grade:      grade,
        Passed:     passed,
        ResultID:   result.ID.String(),
    }, nil
}

// ============================================
// PRACTICE SESSION
// ============================================

func (s *ExamService) StartPractice(ctx context.Context, studentID, subjectID uuid.UUID, questionCount int) (*dto.PracticeSessionResponse, error) {
    questions, err := s.examRepo.GetRandomQuestions(subjectID, questionCount)
    if err != nil {
        return nil, err
    }
    session := &models.PracticeSession{
        ID:             uuid.New(),
        StudentID:      studentID,
        SubjectID:      subjectID,
        TotalQuestions: len(questions),
        Status:         "in_progress",
        StartedAt:      time.Now(),
    }
    if err := s.examRepo.CreatePracticeSession(session); err != nil {
        return nil, err
    }
    // You may want to store the question IDs in session.QuestionIDs (JSON field)
    // For simplicity, we skip storing them now.
    return s.toPracticeResponse(session), nil
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func calculateRemainingTime(start time.Time, durationMinutes int) int {
    elapsed := int(time.Since(start).Minutes())
    remaining := durationMinutes - elapsed
    if remaining < 0 {
        return 0
    }
    return remaining
}

func calculateGrade(percentage float64) string {
    switch {
    case percentage >= 70:
        return "A"
    case percentage >= 60:
        return "B"
    case percentage >= 50:
        return "C"
    case percentage >= 45:
        return "D"
    default:
        return "F"
    }
}

func (s *ExamService) toPracticeResponse(session *models.PracticeSession) *dto.PracticeSessionResponse {
    return &dto.PracticeSessionResponse{
        ID:             session.ID.String(),
        SubjectID:      session.SubjectID.String(),
        TotalQuestions: session.TotalQuestions,
        Answered:       session.Answered,
        Score:          session.Score,
        Status:         session.Status,
        StartedAt:      session.StartedAt,
        CompletedAt:    session.CompletedAt,
    }
}

// extractOption extracts option value from Options map (assuming options are stored as {"A":"text", ...})
func extractOption(opts models.JSONMap, key string) string {
    if opts == nil {
        return ""
    }
    if val, ok := opts[key]; ok {
        if s, ok := val.(string); ok {
            return s
        }
    }
    return ""
}






// package service

// import (
//     "cbt-api/internal/cbt/dto"
//     "cbt-api/internal/cbt/repository"
//     "cbt-api/internal/models"
//     "context"
//     "errors"
//     "time"

//     "github.com/google/uuid"
//     "gorm.io/gorm"
// )

// type ExamService struct {
//     examRepo     *repository.ExamRepository
//     questionRepo *repository.QuestionRepository
//     db           *gorm.DB
// }

// func NewExamService(examRepo *repository.ExamRepository, questionRepo *repository.QuestionRepository, db *gorm.DB) *ExamService {
//     return &ExamService{
//         examRepo:     examRepo,
//         questionRepo: questionRepo,
//         db:           db,
//     }
// }

// // StartExam begins a new exam attempt
// func (s *ExamService) StartExam(ctx context.Context, req *dto.StartExamRequest) (*dto.StartExamResponse, error) {
//     examID, err := uuid.Parse(req.ExamID)
//     if err != nil {
//         return nil, errors.New("invalid exam ID")
//     }
//     studentID, err := uuid.Parse(req.StudentID)
//     if err != nil {
//         return nil, errors.New("invalid student ID")
//     }

//     // Check for existing active attempt
//     existing, _ := s.examRepo.FindActiveAttempt(studentID, examID)
//     if existing != nil {
//         return s.buildStartExamResponse(existing)
//     }

//     // Load exam and its questions
//     exam, questions, err := s.examRepo.FindExamWithQuestions(examID)
//     if err != nil {
//         return nil, errors.New("exam not found")
//     }

//     // Validate exam schedule
//     now := time.Now()
//     if exam.StartTime != nil && now.Before(*exam.StartTime) {
//         return nil, errors.New("exam has not started yet")
//     }
//     if exam.EndTime != nil && now.After(*exam.EndTime) {
//         return nil, errors.New("exam has already ended")
//     }

//     // Create attempt
//     attempt := &models.ExamAttempt{
//         ID:         uuid.New(),
//         StudentID:  studentID,
//         ExamID:     examID,
//         StartTime:  now,
//         Status:     "in_progress",
//     }
//     if err := s.examRepo.CreateAttempt(attempt); err != nil {
//         return nil, err
//     }

//     // Optional proctoring (if exam has proctoring enabled; adjust model if needed)
//     // Since your Exam model does not have EnableProctoring, we skip that part.
//     // If you want proctoring, add a field to the Exam model.

//     return s.buildStartExamResponse(attempt)
// }

// // buildStartExamResponse constructs the response for starting an exam
// func (s *ExamService) buildStartExamResponse(attempt *models.ExamAttempt) (*dto.StartExamResponse, error) {
//     exam, questions, err := s.examRepo.FindExamWithQuestions(attempt.ExamID)
//     if err != nil {
//         return nil, err
//     }
//     totalQ := len(questions)
//     answered, _ := s.examRepo.GetAnsweredCount(attempt.ID)
//     timeRemaining := calculateRemainingTime(attempt.StartTime, exam.DurationMinutes)

//     examDetail := &dto.ExamDetailResponse{
//         ID:               exam.ID.String(),
//         Title:            exam.Title,
//         SubjectID:        exam.SubjectID.String(),
//         DurationMinutes:  exam.DurationMinutes,
//         TotalMarks:       exam.TotalMarks,
//         PassMark:         exam.PassMark,
//         Instructions:     exam.Instructions,
//         StartTime:        exam.StartTime,
//         EndTime:          exam.EndTime,
//         ShuffleQuestions: exam.ShuffleQuestions,
//         ShuffleOptions:   exam.ShuffleOptions,
//     }

//     var qResponses []dto.QuestionResponse
//     for i, q := range questions {
//         qResponses = append(qResponses, dto.QuestionResponse{
//             ID:           q.ID.String(),
//             QuestionText: q.QuestionText,
//             OptionA:      q.OptionA,
//             OptionB:      q.OptionB,
//             OptionC:      q.OptionC,
//             OptionD:      q.OptionD,
//             Marks:        q.Marks,
//             SortOrder:    i + 1,
//         })
//     }

//     attemptResp := &dto.ExamAttemptResponse{
//         ID:             attempt.ID.String(),
//         StudentID:      attempt.StudentID.String(),
//         ExamID:         attempt.ExamID.String(),
//         StartTime:      attempt.StartTime,
//         Status:         attempt.Status,
//         TimeRemaining:  timeRemaining,
//         AnsweredCount:  int(answered),
//         TotalQuestions: totalQ,
//         CreatedAt:      attempt.CreatedAt,
//     }

//     return &dto.StartExamResponse{
//         Attempt:   attemptResp,
//         Exam:      examDetail,
//         Questions: qResponses,
//     }, nil
// }

// // SaveAnswer stores a student's answer
// func (s *ExamService) SaveAnswer(ctx context.Context, req *dto.SaveAnswerRequest) (*dto.SaveAnswerResponse, error) {
//     attemptID, err := uuid.Parse(req.AttemptID)
//     if err != nil {
//         return nil, errors.New("invalid attempt ID")
//     }
//     questionID, err := uuid.Parse(req.QuestionID)
//     if err != nil {
//         return nil, errors.New("invalid question ID")
//     }

//     attempt, err := s.examRepo.FindAttemptByID(attemptID)
//     if err != nil {
//         return nil, errors.New("attempt not found")
//     }
//     if attempt.Status != "in_progress" {
//         return nil, errors.New("exam already submitted")
//     }

//     // Check time limit
//     exam, err := s.examRepo.FindExamByID(attempt.ExamID)
//     if err != nil {
//         return nil, err
//     }
//     if time.Since(attempt.StartTime).Minutes() > float64(exam.DurationMinutes) {
//         return nil, errors.New("time limit exceeded")
//     }

//     // Load question
//     q, err := s.questionRepo.FindByID(questionID)
//     if err != nil {
//         return nil, errors.New("question not found")
//     }

//     isCorrect := (req.SelectedAnswer == q.CorrectAnswer)

//     // Upsert answer
//     existing, _ := s.examRepo.FindAnswer(attemptID, questionID)
//     if existing != nil {
//         existing.SelectedAnswer = req.SelectedAnswer
//         existing.IsCorrect = isCorrect
//         existing.TimeSpent = req.TimeSpent
//         if err := s.examRepo.UpdateAnswer(existing); err != nil {
//             return nil, err
//         }
//     } else {
//         answer := &models.StudentAnswer{
//             ID:             uuid.New(),
//             AttemptID:      attemptID,
//             QuestionID:     questionID,
//             SelectedAnswer: req.SelectedAnswer,
//             IsCorrect:      isCorrect,
//             TimeSpent:      req.TimeSpent,
//         }
//         if err := s.examRepo.SaveAnswer(answer); err != nil {
//             return nil, err
//         }
//     }

//     return &dto.SaveAnswerResponse{
//         IsCorrect:     isCorrect,
//         CorrectAnswer: q.CorrectAnswer,
//         Explanation:   q.Explanation,
//     }, nil
// }

// // SubmitExam finalizes the exam and calculates the score
// func (s *ExamService) SubmitExam(ctx context.Context, req *dto.SubmitExamRequest) (*dto.SubmitExamResponse, error) {
//     attemptID, err := uuid.Parse(req.AttemptID)
//     if err != nil {
//         return nil, errors.New("invalid attempt ID")
//     }

//     attempt, err := s.examRepo.FindAttemptByID(attemptID)
//     if err != nil {
//         return nil, errors.New("attempt not found")
//     }
//     if attempt.Status != "in_progress" {
//         return nil, errors.New("exam already submitted")
//     }

//     tx := s.db.Begin()
//     defer func() {
//         if r := recover(); r != nil {
//             tx.Rollback()
//         }
//     }()

//     // Calculate total score based on correct answers
//     answers, err := s.examRepo.FindAnswersByAttempt(attemptID)
//     if err != nil {
//         tx.Rollback()
//         return nil, err
//     }
//     totalScore := 0
//     for _, ans := range answers {
//         if ans.IsCorrect {
//             // You need to get marks per question – either store marks in StudentAnswer or fetch question.
//             // Simpler: fetch question marks for each correct answer.
//             q, err := s.questionRepo.FindByID(ans.QuestionID)
//             if err != nil {
//                 tx.Rollback()
//                 return nil, err
//             }
//             totalScore += q.Marks
//         }
//     }

//     now := time.Now()
//     attempt.Status = "submitted"
//     attempt.EndTime = &now
//     attempt.Score = &totalScore
//     if err := s.examRepo.UpdateAttempt(attempt); err != nil {
//         tx.Rollback()
//         return nil, err
//     }

//     exam, err := s.examRepo.FindExamByID(attempt.ExamID)
//     if err != nil {
//         tx.Rollback()
//         return nil, err
//     }

//     percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
//     passed := percentage >= float64(exam.PassMark)
//     grade := calculateGrade(percentage)

//     result := &models.Result{
//         ID:          uuid.New(),
//         ExamID:      attempt.ExamID,
//         StudentID:   attempt.StudentID,
//         TotalScore:  totalScore,
//         Percentage:  percentage,
//         Grade:       grade,
//         Remarks:     "",
//         PublishedAt: &now,
//     }
//     if err := s.examRepo.CreateResult(result); err != nil {
//         tx.Rollback()
//         return nil, err
//     }

//     if err := tx.Commit().Error; err != nil {
//         return nil, err
//     }

//     return &dto.SubmitExamResponse{
//         AttemptID:  attemptID.String(),
//         Score:      totalScore,
//         TotalMarks: exam.TotalMarks,
//         Percentage: percentage,
//         Grade:      grade,
//         Passed:     passed,
//         ResultID:   result.ID.String(),
//     }, nil
// }

// // StartPractice creates a practice session
// func (s *ExamService) StartPractice(ctx context.Context, studentID, subjectID uuid.UUID, questionCount int) (*dto.PracticeSessionResponse, error) {
//     questions, err := s.examRepo.GetRandomQuestions(subjectID, questionCount)
//     if err != nil {
//         return nil, err
//     }
//     session := &models.PracticeSession{
//         ID:             uuid.New(),
//         StudentID:      studentID,
//         SubjectID:      subjectID,
//         TotalQuestions: len(questions),
//         Status:         "in_progress",
//         StartedAt:      time.Now(),
//     }
//     if err := s.examRepo.CreatePracticeSession(session); err != nil {
//         return nil, err
//     }
//     return s.toPracticeResponse(session), nil
// }

// // Helper functions
// func calculateRemainingTime(start time.Time, durationMinutes int) int {
//     elapsed := int(time.Since(start).Minutes())
//     remaining := durationMinutes - elapsed
//     if remaining < 0 {
//         return 0
//     }
//     return remaining
// }

// func calculateGrade(percentage float64) string {
//     switch {
//     case percentage >= 70:
//         return "A"
//     case percentage >= 60:
//         return "B"
//     case percentage >= 50:
//         return "C"
//     case percentage >= 45:
//         return "D"
//     default:
//         return "F"
//     }
// }

// func (s *ExamService) toPracticeResponse(session *models.PracticeSession) *dto.PracticeSessionResponse {
//     return &dto.PracticeSessionResponse{
//         ID:             session.ID.String(),
//         SubjectID:      session.SubjectID.String(),
//         TotalQuestions: session.TotalQuestions,
//         Answered:       session.Answered,
//         Score:          session.Score,
//         Status:         session.Status,
//         StartedAt:      session.StartedAt,
//         CompletedAt:    session.CompletedAt,
//     }
// }



// // package service

// // import (
// //     "cbt-api/internal/cbt/dto"
// //     "cbt-api/internal/cbt/repository"
// //     "cbt-api/internal/models"
// //     "context"
// //     "errors"
// //     "fmt"
// //     "time"

// //     "github.com/google/uuid"
// //     "gorm.io/gorm"
// // )

// // type ExamService struct {
// //     examRepo     *repository.ExamRepository
// //     questionRepo *repository.QuestionRepository
// //     db           *gorm.DB // for transactions
// // }

// // func NewExamService(examRepo *repository.ExamRepository, questionRepo *repository.QuestionRepository, db *gorm.DB) *ExamService {
// //     return &ExamService{
// //         examRepo:     examRepo,
// //         questionRepo: questionRepo,
// //         db:           db,
// //     }
// // }

// // // ============================================
// // // EXAM ATTEMPT & TAKING
// // // ============================================

// // func (s *ExamService) StartExam(ctx context.Context, req *dto.StartExamRequest) (*dto.StartExamResponse, error) {
// //     examID, err := uuid.Parse(req.ExamID)
// //     if err != nil {
// //         return nil, errors.New("invalid exam ID")
// //     }
// //     studentID, err := uuid.Parse(req.StudentID)
// //     if err != nil {
// //         return nil, errors.New("invalid student ID")
// //     }

// //     // Check if already an active attempt
// //     existing, _ := s.examRepo.FindActiveAttempt(studentID, examID)
// //     if existing != nil {
// //         // Return existing attempt
// //         return s.buildStartExamResponse(existing)
// //     }

// //     // Load exam and its questions
// //     exam, questions, err := s.examRepo.FindExamWithQuestions(examID)
// //     if err != nil {
// //         return nil, errors.New("exam not found")
// //     }

// //     // Validate exam schedule (if start_time/end_time set)
// //     now := time.Now()
// //     if exam.StartTime != nil && now.Before(*exam.StartTime) {
// //         return nil, errors.New("exam has not started yet")
// //     }
// //     if exam.EndTime != nil && now.After(*exam.EndTime) {
// //         return nil, errors.New("exam has already ended")
// //     }

// //     // Create attempt
// //     attempt := &models.ExamAttempt{
// //         ID:         uuid.New(),
// //         StudentID:  studentID,
// //         ExamID:     examID,
// //         StartTime:  now,
// //         Status:     "in_progress",
// //     }
// //     if err := s.examRepo.CreateAttempt(attempt); err != nil {
// //         return nil, err
// //     }

// //     // Create proctoring session if needed
// //     proctoringID := uuid.New()
// //     if exam.EnableProctoring {
// //         proctoring := &models.ProctoringSession{
// //             ID:        proctoringID,
// //             AttemptID: attempt.ID,
// //             Status:    "active",
// //             StartedAt: now,
// //         }
// //         s.examRepo.CreateProctoringSession(proctoring)
// //     }

// //     return s.buildStartExamResponse(attempt)
// // }

// // func (s *ExamService) buildStartExamResponse(attempt *models.ExamAttempt) (*dto.StartExamResponse, error) {
// //     exam, questions, err := s.examRepo.FindExamWithQuestions(attempt.ExamID)
// //     if err != nil {
// //         return nil, err
// //     }
// //     totalQ := len(questions)
// //     answered, _ := s.examRepo.GetAnsweredCount(attempt.ID)
// //     timeRemaining := calculateRemainingTime(attempt.StartTime, exam.DurationMinutes)

// //     examDetail := &dto.ExamDetailResponse{
// //         ID:              exam.ID.String(),
// //         Title:           exam.Title,
// //         SubjectID:       exam.SubjectID.String(),
// //         DurationMinutes: exam.DurationMinutes,
// //         TotalMarks:      exam.TotalMarks,
// //         PassMark:        exam.PassMark,
// //         Instructions:    exam.Instructions,
// //         StartTime:       exam.StartTime,
// //         EndTime:         exam.EndTime,
// //         ShuffleQuestions: exam.ShuffleQuestions,
// //         ShuffleOptions:   exam.ShuffleOptions,
// //     }

// //     var qResponses []dto.QuestionResponse
// //     for i, q := range questions {
// //         qResponses = append(qResponses, dto.QuestionResponse{
// //             ID:           q.ID.String(),
// //             QuestionText: q.QuestionText,
// //             OptionA:      q.OptionA,
// //             OptionB:      q.OptionB,
// //             OptionC:      q.OptionC,
// //             OptionD:      q.OptionD,
// //             Marks:        q.Marks,
// //             SortOrder:    i + 1,
// //         })
// //     }

// //     attemptResp := &dto.ExamAttemptResponse{
// //         ID:            attempt.ID.String(),
// //         StudentID:     attempt.StudentID.String(),
// //         ExamID:        attempt.ExamID.String(),
// //         StartTime:     attempt.StartTime,
// //         Status:        attempt.Status,
// //         TimeRemaining: timeRemaining,
// //         AnsweredCount: int(answered),
// //         TotalQuestions: totalQ,
// //         CreatedAt:     attempt.CreatedAt,
// //     }

// //     proctoringID := ""
// //     proctoring, _ := s.examRepo.FindProctoringByAttempt(attempt.ID)
// //     if proctoring != nil {
// //         proctoringID = proctoring.ID.String()
// //     }

// //     return &dto.StartExamResponse{
// //         Attempt:      attemptResp,
// //         Exam:         examDetail,
// //         Questions:    qResponses,
// //         ProctoringID: proctoringID,
// //     }, nil
// // }

// // func (s *ExamService) SaveAnswer(ctx context.Context, req *dto.SaveAnswerRequest) (*dto.SaveAnswerResponse, error) {
// //     attemptID, err := uuid.Parse(req.AttemptID)
// //     if err != nil {
// //         return nil, errors.New("invalid attempt ID")
// //     }
// //     questionID, err := uuid.Parse(req.QuestionID)
// //     if err != nil {
// //         return nil, errors.New("invalid question ID")
// //     }

// //     attempt, err := s.examRepo.FindAttemptByID(attemptID)
// //     if err != nil {
// //         return nil, errors.New("attempt not found")
// //     }
// //     if attempt.Status != "in_progress" {
// //         return nil, errors.New("exam already submitted")
// //     }

// //     // Check time limit
// //     exam, err := s.examRepo.FindExamByID(attempt.ExamID)
// //     if err != nil {
// //         return nil, err
// //     }
// //     if time.Since(attempt.StartTime).Minutes() > float64(exam.DurationMinutes) {
// //         return nil, errors.New("time limit exceeded")
// //     }

// //     // Load question to verify answer
// //     q, err := s.questionRepo.FindByID(questionID)
// //     if err != nil {
// //         return nil, errors.New("question not found")
// //     }

// //     isCorrect := (req.SelectedAnswer == q.CorrectAnswer)
// //     // Calculate marks earned
// //     var marksEarned int
// //     if isCorrect {
// //         marksEarned = q.Marks
// //     }

// //     // Upsert answer
// //     existing, _ := s.examRepo.FindAnswer(attemptID, questionID)
// //     if existing != nil {
// //         existing.SelectedAnswer = req.SelectedAnswer
// //         existing.IsCorrect = isCorrect
// //         existing.TimeSpent = req.TimeSpent
// //         if err := s.examRepo.UpdateAnswer(existing); err != nil {
// //             return nil, err
// //         }
// //     } else {
// //         answer := &models.StudentAnswer{
// //             ID:             uuid.New(),
// //             AttemptID:      attemptID,
// //             QuestionID:     questionID,
// //             SelectedAnswer: req.SelectedAnswer,
// //             IsCorrect:      isCorrect,
// //             TimeSpent:      req.TimeSpent,
// //         }
// //         if err := s.examRepo.SaveAnswer(answer); err != nil {
// //             return nil, err
// //         }
// //     }

// //     return &dto.SaveAnswerResponse{
// //         IsCorrect:     isCorrect,
// //         CorrectAnswer: q.CorrectAnswer,
// //         Explanation:   q.Explanation,
// //     }, nil
// // }

// // func (s *ExamService) SubmitExam(ctx context.Context, req *dto.SubmitExamRequest) (*dto.SubmitExamResponse, error) {
// //     attemptID, err := uuid.Parse(req.AttemptID)
// //     if err != nil {
// //         return nil, errors.New("invalid attempt ID")
// //     }

// //     attempt, err := s.examRepo.FindAttemptByID(attemptID)
// //     if err != nil {
// //         return nil, errors.New("attempt not found")
// //     }
// //     if attempt.Status != "in_progress" {
// //         return nil, errors.New("exam already submitted")
// //     }

// //     // Start a transaction
// //     tx := s.db.Begin()
// //     defer func() {
// //         if r := recover(); r != nil {
// //             tx.Rollback()
// //         }
// //     }()

// //     // Calculate score
// //     answers, err := s.examRepo.FindAnswersByAttempt(attemptID)
// //     if err != nil {
// //         tx.Rollback()
// //         return nil, err
// //     }
// //     var totalScore int
// //     for _, ans := range answers {
// //         totalScore += ans.MarksEarned // assuming MarksEarned field exists in StudentAnswer
// //     }

// //     now := time.Now()
// //     attempt.Status = "submitted"
// //     attempt.EndTime = &now
// //     attempt.Score = &totalScore
// //     if err := s.examRepo.UpdateAttempt(attempt); err != nil {
// //         tx.Rollback()
// //         return nil, err
// //     }

// //     exam, err := s.examRepo.FindExamByID(attempt.ExamID)
// //     if err != nil {
// //         tx.Rollback()
// //         return nil, err
// //     }

// //     percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
// //     passed := percentage >= float64(exam.PassMark)
// //     grade := calculateGrade(percentage)

// //     result := &models.Result{
// //         ID:         uuid.New(),
// //         ExamID:     attempt.ExamID,
// //         StudentID:  attempt.StudentID,
// //         Score:      totalScore,
// //         Percentage: percentage,
// //         Grade:      grade,
// //         Status:     "published",
// //         PublishedAt: now,
// //     }
// //     if err := s.examRepo.CreateResult(result); err != nil {
// //         tx.Rollback()
// //         return nil, err
// //     }

// //     if err := tx.Commit().Error; err != nil {
// //         return nil, err
// //     }

// //     return &dto.SubmitExamResponse{
// //         AttemptID:  attemptID.String(),
// //         Score:      totalScore,
// //         TotalMarks: exam.TotalMarks,
// //         Percentage: percentage,
// //         Grade:      grade,
// //         Passed:     passed,
// //         ResultID:   result.ID.String(),
// //     }, nil
// // }

// // // ============================================
// // // PRACTICE SESSION
// // // ============================================

// // func (s *ExamService) StartPractice(ctx context.Context, studentID, subjectID uuid.UUID, questionCount int) (*dto.PracticeSessionResponse, error) {
// //     questions, err := s.examRepo.GetRandomQuestions(subjectID, questionCount)
// //     if err != nil {
// //         return nil, err
// //     }
// //     session := &models.PracticeSession{
// //         ID:             uuid.New(),
// //         StudentID:      studentID,
// //         SubjectID:      subjectID,
// //         TotalQuestions: len(questions),
// //         Status:         "in_progress",
// //         StartedAt:      time.Now(),
// //     }
// //     if err := s.examRepo.CreatePracticeSession(session); err != nil {
// //         return nil, err
// //     }
// //     // TODO: store the questions for the session (a join table PracticeSessionQuestion)
// //     return s.toPracticeResponse(session), nil
// // }

// // // Helper functions
// // func calculateRemainingTime(start time.Time, durationMinutes int) int {
// //     elapsed := int(time.Since(start).Minutes())
// //     remaining := durationMinutes - elapsed
// //     if remaining < 0 {
// //         return 0
// //     }
// //     return remaining
// // }

// // func calculateGrade(percentage float64) string {
// //     switch {
// //     case percentage >= 70:
// //         return "A"
// //     case percentage >= 60:
// //         return "B"
// //     case percentage >= 50:
// //         return "C"
// //     case percentage >= 45:
// //         return "D"
// //     default:
// //         return "F"
// //     }
// // }

// // func (s *ExamService) toPracticeResponse(session *models.PracticeSession) *dto.PracticeSessionResponse {
// //     return &dto.PracticeSessionResponse{
// //         ID:             session.ID.String(),
// //         SubjectID:      session.SubjectID.String(),
// //         TotalQuestions: session.TotalQuestions,
// //         Answered:       session.Answered,
// //         Score:          session.Score,
// //         Status:         session.Status,
// //         StartedAt:      session.StartedAt,
// //         CompletedAt:    session.CompletedAt,
// //     }
// // }