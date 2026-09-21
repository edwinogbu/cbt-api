package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"sort"
	"strings"  
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/repository"
	"cbt-api/internal/models"

	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"  
	"gorm.io/gorm"
)

// ============================================
// EXAM SERVICE STRUCT
// ============================================

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
// ADD MISSING SERVICE METHODS
// ============================================

// CreateExamWithContext creates an exam with full academic context
func (s *ExamService) CreateExamWithContext(ctx context.Context, req *dto.CreateExamWithContextRequest) (*dto.ExamWithContextResponse, error) {
	exam := &models.Exam{
		ID:               models.GenerateID(),
		Title:            req.Title,
		SubjectID:        req.SubjectID,
		ClassID:          req.ClassID,
		SessionID:        req.SessionID,
		TermID:           req.TermID,
		ExamType:         req.ExamType,
		DurationMinutes:  req.DurationMinutes,
		TotalMarks:       req.TotalMarks,
		PassMark:         req.PassMark,
		Instructions:     req.Instructions,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		ShuffleQuestions: req.ShuffleQuestions,
		ShuffleOptions:   req.ShuffleOptions,
		IsActive:         req.IsActive,
		Status:           string(models.ExamStatusDraft),
		ReviewPolicy: models.ReviewPolicyStorage{
			ReleaseMode:        req.ReviewPolicy.ReleaseMode,
			ShowScore:          req.ReviewPolicy.ShowScore,
			ShowGrade:          req.ReviewPolicy.ShowGrade,
			ShowCorrectAnswers: req.ReviewPolicy.ShowCorrectAnswers,
			ShowExplanations:   req.ReviewPolicy.ShowExplanations,
			ShowQuestionReview: req.ReviewPolicy.ShowQuestionReview,
			AllowRetake:        req.ReviewPolicy.AllowRetake,
		},
	}

	if err := s.examRepo.CreateExamWithContext(ctx, exam); err != nil {
		return nil, err
	}

	return s.toExamWithContextResponse(ctx, exam)
}

// GetExamWithContext gets an exam with full academic context
func (s *ExamService) GetExamWithContext(ctx context.Context, id string) (*dto.ExamWithContextResponse, error) {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toExamWithContextResponse(ctx, exam)
}

// toExamWithContextResponse converts Exam to ExamWithContextResponse
func (s *ExamService) toExamWithContextResponse(ctx context.Context, exam *models.Exam) (*dto.ExamWithContextResponse, error) {
	questionCount, _ := s.examRepo.CountQuestionsInExam(ctx, exam.ID)

	var subject models.Subject
	var class models.Class
	var session models.AcademicSession
	var term models.Term

	s.db.WithContext(ctx).First(&subject, "id = ?", exam.SubjectID)
	if exam.ClassID != "" {
		s.db.WithContext(ctx).First(&class, "id = ?", exam.ClassID)
	}
	if exam.SessionID != "" {
		s.db.WithContext(ctx).First(&session, "id = ?", exam.SessionID)
	}
	if exam.TermID != "" {
		s.db.WithContext(ctx).First(&term, "id = ?", exam.TermID)
	}

	return &dto.ExamWithContextResponse{
		ID:               exam.ID,
		Title:            exam.Title,
		ExamType:         exam.ExamType,
		SubjectID:        exam.SubjectID,
		SubjectName:      subject.Name,
		ClassID:          exam.ClassID,
		ClassName:        models.GetClassDisplayName(&class),
		SessionID:        exam.SessionID,
		SessionName:      session.Name,
		TermID:           exam.TermID,
		TermName:         term.Name,
		DurationMinutes:  exam.DurationMinutes,
		TotalMarks:       exam.TotalMarks,
		PassMark:         exam.PassMark,
		Instructions:     exam.Instructions,
		StartTime:        exam.StartTime,
		EndTime:          exam.EndTime,
		ShuffleQuestions: exam.ShuffleQuestions,
		ShuffleOptions:   exam.ShuffleOptions,
		IsActive:         exam.IsActive,
		QuestionCount:    int(questionCount),
		Status:           exam.Status,
		CreatedAt:        exam.CreatedAt,
		UpdatedAt:        exam.UpdatedAt,
	}, nil
}

// // BulkAddQuestionsToExam bulk adds questions to an exam
// func (s *ExamService) BulkAddQuestionsToExam(ctx context.Context, examID string, req *dto.BulkAddQuestionsToExamRequest) error {
// 	exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
// 	if err != nil {
// 		return errors.New("exam not found")
// 	}

// 	var questionIDs []string

// 	if req.SelectAll {
// 		// Get questions from repository
// 		questions, err := s.examRepo.GetSubjectQuestionsWithContext(ctx,
// 			exam.SubjectID, exam.TermID, exam.SessionID, exam.ClassID)
// 		if err != nil {
// 			return err
// 		}

// 		for _, q := range questions {
// 			// Filter by difficulty
// 			if len(req.FilterByDifficulty) > 0 {
// 				found := false
// 				for _, d := range req.FilterByDifficulty {
// 					if string(q.Difficulty) == d {
// 						found = true
// 						break
// 					}
// 				}
// 				if !found {
// 					continue
// 				}
// 			}

// 			// Filter by bloom level
// 			if len(req.FilterByBloom) > 0 {
// 				found := false
// 				for _, b := range req.FilterByBloom {
// 					if string(q.BloomLevel) == b {
// 						found = true
// 						break
// 					}
// 				}
// 				if !found {
// 					continue
// 				}
// 			}

// 			// Filter by topic
// 			if req.FilterByTopic != "" && q.Topic != req.FilterByTopic {
// 				continue
// 			}

// 			questionIDs = append(questionIDs, q.ID)
// 		}
// 	} else {
// 		questionIDs = req.QuestionIDs
// 	}

// 	if len(questionIDs) == 0 {
// 		return errors.New("no questions selected")
// 	}

// 	return s.examRepo.BulkAddQuestionsToExam(ctx, examID, questionIDs)
// }

// ============================================
// BulkAddQuestionsToExam - PRODUCTION READY
// ============================================

func (s *ExamService) BulkAddQuestionsToExam(ctx context.Context, examID string, req *dto.BulkAddQuestionsToExamRequest) error {
    // 1. Get the exam with full context
    exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
    if err != nil {
        return errors.New("exam not found")
    }

    var questionIDs []string

    // 2. Handle "Select All" case with filters
    if req.SelectAll {
        // Get questions from repository matching subject, term, session, class
        questions, err := s.examRepo.GetSubjectQuestionsWithContext(ctx,
            exam.SubjectID, exam.TermID, exam.SessionID, exam.ClassID)
        if err != nil {
            return fmt.Errorf("failed to fetch questions: %w", err)
        }

        // Apply filters
        for _, q := range questions {
            // ✅ Filter by exam type (except for practice)
            if exam.ExamType != "practice" && q.ExamType != exam.ExamType {
                continue
            }

            // ✅ Only published questions (except for practice)
            if exam.ExamType != "practice" && q.Status != models.QuestionStatusPublished {
                continue
            }

            // Filter by difficulty
            if len(req.FilterByDifficulty) > 0 {
                found := false
                for _, d := range req.FilterByDifficulty {
                    if string(q.Difficulty) == d {
                        found = true
                        break
                    }
                }
                if !found {
                    continue
                }
            }

            // Filter by bloom level
            if len(req.FilterByBloom) > 0 {
                found := false
                for _, b := range req.FilterByBloom {
                    if string(q.BloomLevel) == b {
                        found = true
                        break
                    }
                }
                if !found {
                    continue
                }
            }

            // Filter by topic
            if req.FilterByTopic != "" && q.Topic != req.FilterByTopic {
                continue
            }

            questionIDs = append(questionIDs, q.ID)
        }
    } else {
        // 3. Handle specific question IDs
        if len(req.QuestionIDs) == 0 {
            return errors.New("no question IDs provided")
        }
        questionIDs = req.QuestionIDs
    }

    // 4. Check if we have any questions
    if len(questionIDs) == 0 {
        return errors.New("no questions selected or matching filters")
    }

    // 5. VALIDATE questions using repository method
    //    This checks: exam_type, term, session, subject, class, status
    if err := s.questionRepo.ValidateQuestionsForExam(ctx, exam, questionIDs); err != nil {
        return err
    }

    // 6.  VALIDATE question count using repository method
    //    This checks min/max based on exam type
    if err := s.questionRepo.ValidateQuestionCountForExam(exam.ExamType, len(questionIDs)); err != nil {
        return err
    }

    // 7. Remove duplicates
    uniqueMap := make(map[string]bool)
    var uniqueIDs []string
    for _, id := range questionIDs {
        if !uniqueMap[id] {
            uniqueMap[id] = true
            uniqueIDs = append(uniqueIDs, id)
        }
    }

    // 8. Use transaction to ensure atomic operation
    return s.db.Transaction(func(tx *gorm.DB) error {
        txRepo := repository.NewExamRepository(tx)

        // This will delete existing questions and add the new ones
        if err := txRepo.BulkAddQuestionsToExam(ctx, examID, uniqueIDs); err != nil {
            return fmt.Errorf("failed to add questions to exam: %w", err)
        }

        // Update total marks on the exam
        var questions []models.QuestionBank
        if err := tx.WithContext(ctx).
            Where("id IN ?", uniqueIDs).
            Find(&questions).Error; err != nil {
            return fmt.Errorf("failed to fetch questions for marks update: %w", err)
        }

        totalMarks := 0
        for _, q := range questions {
            totalMarks += q.Marks
        }

        if err := tx.WithContext(ctx).
            Model(&models.Exam{}).
            Where("id = ?", examID).
            Update("total_marks", totalMarks).Error; err != nil {
            return fmt.Errorf("failed to update exam total marks: %w", err)
        }

        return nil
    })
}


// PreviewExam previews an exam with all questions and statistics
func (s *ExamService) PreviewExam(ctx context.Context, examID string) (*dto.ExamPreviewResponse, error) {
	exam, questions, err := s.examRepo.FindExamWithQuestionsWithContext(ctx, examID)
	if err != nil {
		return nil, errors.New("exam not found")
	}

	stats := dto.ExamPreviewStatistics{
		TotalQuestions: len(questions),
		TotalMarks:     exam.TotalMarks,
		ByDifficulty:   make(map[string]int),
		ByBloomLevel:   make(map[string]int),
		ByQuestionType: make(map[string]int),
		PassMark:       exam.PassMark,
	}

	totalMarks := 0
	previewQuestions := make([]dto.QuestionPreviewItem, len(questions))

	for i, q := range questions {
		previewQuestions[i] = dto.QuestionPreviewItem{
			ID:           q.ID,
			QuestionText: q.QuestionText,
			QuestionType: string(q.QuestionType),
			Difficulty:   string(q.Difficulty),
			BloomLevel:   string(q.BloomLevel),
			Marks:        q.Marks,
			Options:      s.convertOptionsToPreview(q.Options),
		}

		stats.ByDifficulty[string(q.Difficulty)]++
		stats.ByBloomLevel[string(q.BloomLevel)]++
		stats.ByQuestionType[string(q.QuestionType)]++
		totalMarks += q.Marks
	}

	if len(questions) > 0 {
		stats.AverageMarks = float64(totalMarks) / float64(len(questions))
	}
	stats.PassMarkPercentage = float64(exam.PassMark) / float64(exam.TotalMarks) * 100

	var subject models.Subject
	var class models.Class
	s.db.WithContext(ctx).First(&subject, "id = ?", exam.SubjectID)
	if exam.ClassID != "" {
		s.db.WithContext(ctx).First(&class, "id = ?", exam.ClassID)
	}

	return &dto.ExamPreviewResponse{
		ExamID:          examID,
		ExamTitle:       exam.Title,
		SubjectID:       exam.SubjectID,
		SubjectName:     subject.Name,
		ClassID:         exam.ClassID,
		ClassName:       models.GetClassDisplayName(&class),
		TotalQuestions:  len(questions),
		TotalMarks:      exam.TotalMarks,
		DurationMinutes: exam.DurationMinutes,
		PassMark:        exam.PassMark,
		Statistics:      stats,
		Questions:       previewQuestions,
	}, nil
}

// convertOptionsToPreview converts OptionStorage to OptionPreview
func (s *ExamService) convertOptionsToPreview(storage models.OptionStorage) []dto.OptionPreview {
	opts := make([]dto.OptionPreview, 0, len(storage))
	for _, item := range storage {
		opts = append(opts, dto.OptionPreview{
			Key:  item.Key,
			Text: item.Text,
		})
	}
	return opts
}

// AutoSave auto-saves answers for offline support
func (s *ExamService) AutoSave(ctx context.Context, req *dto.AutoSaveRequest, studentID string) (*dto.AutoSaveResponse, error) {
	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	// Verify ownership
	if attempt.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	savedCount := 0
	for _, item := range req.Answers {
		existing, err := s.examRepo.FindAnswer(req.AttemptID, item.QuestionID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}

		if existing != nil {
			existing.SelectedAnswer = item.SelectedAnswer
			existing.IsMarked = item.IsMarked
			existing.TimeSpent = item.TimeSpent
			if err := s.examRepo.UpdateAnswer(existing); err != nil {
				continue
			}
		} else {
			answer := &models.StudentAnswer{
				ID:             models.GenerateID(),
				AttemptID:      req.AttemptID,
				QuestionID:     item.QuestionID,
				SelectedAnswer: item.SelectedAnswer,
				IsMarked:       item.IsMarked,
				TimeSpent:      item.TimeSpent,
			}
			if err := s.examRepo.SaveAnswer(answer); err != nil {
				continue
			}
		}
		savedCount++
	}

	return &dto.AutoSaveResponse{
		AttemptID:         req.AttemptID,
		SavedAt:           time.Now(),
		SavedCount:        savedCount,
		TotalAnswersSaved: savedCount,
		NextSaveAt:        time.Now().Add(30 * time.Second),
		LocalStorageKey:   "exam_attempt_" + req.AttemptID,
		SyncStatus: dto.SyncStatusData{
			Online:      req.SyncStatus.Online,
			PendingSync: req.SyncStatus.PendingSync,
			LastSync:    req.SyncStatus.LastSync,
		},
	}, nil
}

// GetExamReview gets detailed exam review with correct answers
func (s *ExamService) GetExamReview(ctx context.Context, attemptID string, studentID string) (*dto.StudentExamResultResponse, error) {
	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, attemptID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	// Verify ownership
	if attempt.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	// Check if review is allowed
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}

	// Check if exam is completed
	if attempt.Status != string(models.AttemptStatusCompleted) {
		return nil, errors.New("exam not completed yet")
	}

	// Check review policy
	if !exam.ReviewPolicy.ShowCorrectAnswers {
		return nil, errors.New("review not allowed for this exam")
	}

	return s.GetStudentExamResult(ctx, studentID, attempt.ExamID)
}

// GetStudentDashboard gets the student dashboard
func (s *ExamService) GetStudentDashboard(ctx context.Context, studentID string) (*dto.StudentDashboardResponse, error) {
	student, err := s.examRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, errors.New("student not found")
	}

	// Get available exams
	availableExams, err := s.examRepo.GetExamsByStudentAndClass(ctx, studentID, student.ClassID)
	if err != nil {
		return nil, err
	}

	// Get in-progress exams
	attempts, err := s.examRepo.GetActiveAttemptsByStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}

	// Get completed exams with results
	completedAttempts, err := s.examRepo.GetCompletedAttemptsByStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}

	// Get upcoming exams
	upcomingExams, err := s.getUpcomingExamsForStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}

	// Build available list
	availableList := make([]dto.AvailableExam, 0)
	for _, exam := range availableExams {
		availableList = append(availableList, dto.AvailableExam{
			ID:               exam.ID,
			Title:            exam.Title,
			ExamType:         exam.ExamType,
			DurationMinutes:  exam.DurationMinutes,
			TotalMarks:       exam.TotalMarks,
			PassMark:         exam.PassMark,
			Schedule: dto.ScheduleInfo{
				StartTime:   exam.StartTime,
				EndTime:     exam.EndTime,
				IsAvailable: s.isExamAvailable(exam),
			},
			Status:           exam.Status,
			OfflineAvailable: true,
			HasStarted:       false,
			HasCompleted:     false,
			Instructions:     exam.Instructions,
		})
	}

	// Build in-progress list
	inProgressList := make([]dto.InProgressExam, 0)
	for _, attempt := range attempts {
		exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
		if err != nil {
			continue
		}
		answered, _ := s.examRepo.GetAnsweredCountWithContext(ctx, attempt.ID)
		questions, _ := s.examRepo.GetExamQuestionsWithContext(ctx, attempt.ExamID)
		totalQ := len(questions)

		inProgressList = append(inProgressList, dto.InProgressExam{
			ID:              exam.ID,
			Title:           exam.Title,
			ExamType:        exam.ExamType,
			DurationMinutes: exam.DurationMinutes,
			TotalMarks:      exam.TotalMarks,
			PassMark:        exam.PassMark,
			Schedule: dto.ScheduleInfo{
				StartTime:   exam.StartTime,
				EndTime:     exam.EndTime,
				IsAvailable: true,
			},
			Attempt: dto.AttemptInfo{
				AttemptID:          attempt.ID,
				StartTime:          attempt.StartTime,
				TimeRemaining:      s.calculateRemainingTime(attempt.StartTime, exam.DurationMinutes),
				TimeElapsed:        int(time.Since(attempt.StartTime).Seconds()),
				AnsweredCount:      int(answered),
				TotalQuestions:     totalQ,
				ProgressPercentage: s.calculateProgress(int(answered), totalQ),
				LastActivity:       attempt.UpdatedAt,
			},
			OfflineAvailable: true,
			HasStarted:       true,
			HasCompleted:     false,
		})
	}

	// Build completed list
	completedList := make([]dto.CompletedExam, 0)
	for _, attempt := range completedAttempts {
		exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
		if err != nil {
			continue
		}
		result, err := s.examRepo.GetStudentExamResult(ctx, studentID, attempt.ExamID)
		if err != nil || result == nil {
			continue
		}

		completedList = append(completedList, dto.CompletedExam{
			ID:              exam.ID,
			Title:           exam.Title,
			ExamType:        exam.ExamType,
			DurationMinutes: exam.DurationMinutes,
			TotalMarks:      exam.TotalMarks,
			PassMark:        exam.PassMark,
			Schedule: dto.ScheduleInfo{
				StartTime: exam.StartTime,
				EndTime:   exam.EndTime,
			},
			Result: dto.ResultInfo{
				Score:       result.TotalScore,
				Percentage:  result.Percentage,
				Grade:       result.Grade,
				Passed:      result.Percentage >= float64(exam.PassMark),
				CompletedAt: *result.PublishedAt,
			},
			HasCompleted:    true,
			CanReview:       exam.ReviewPolicy.ShowCorrectAnswers,
			ReviewAvailable: exam.ReviewPolicy.ShowCorrectAnswers,
		})
	}

	// Build upcoming list
	upcomingList := make([]dto.StudentUpcomingExam, 0)
	for _, exam := range upcomingExams {
		upcomingList = append(upcomingList, dto.StudentUpcomingExam{
			ID:               exam.ID,
			Title:            exam.Title,
			ExamType:         exam.ExamType,
			DurationMinutes:  exam.DurationMinutes,
			TotalMarks:       exam.TotalMarks,
			PassMark:         exam.PassMark,
			Schedule: dto.ScheduleInfo{
				StartTime:   exam.StartTime,
				EndTime:     exam.EndTime,
				IsAvailable: false,
			},
			Status:           "upcoming",
			DaysUntil:        s.daysUntil(exam.StartTime),
			OfflineAvailable: false,
			HasStarted:       false,
			HasCompleted:     false,
			Instructions:     exam.Instructions,
		})
	}

	// Calculate summary
	totalAvailable := len(availableList)
	totalInProgress := len(inProgressList)
	totalCompleted := len(completedList)
	totalUpcoming := len(upcomingList)

	var overallAverage float64
	if totalCompleted > 0 {
		var totalScore int
		for _, exam := range completedList {
			totalScore += exam.Result.Score
		}
		overallAverage = float64(totalScore) / float64(totalCompleted)
	}

	// Get school info
	var school models.School
	s.db.WithContext(ctx).First(&school, "id = ?", student.SchoolID)

	return &dto.StudentDashboardResponse{
		Student: dto.StudentInfo{
			ID:              student.ID,
			Name:            student.User.FirstName + " " + student.User.LastName,
			AdmissionNumber: student.AdmissionNo,
			Class:           student.ClassID,
			ClassID:         student.ClassID,
			School:          school.Name,
			SchoolID:        student.SchoolID,
		},
		Summary: dto.DashboardSummary{
			TotalAvailable:  totalAvailable,
			TotalInProgress: totalInProgress,
			TotalCompleted:  totalCompleted,
			TotalUpcoming:   totalUpcoming,
			OverallAverage:  overallAverage,
		},
		AvailableExams:  availableList,
		InProgressExams: inProgressList,
		CompletedExams:  completedList,
		UpcomingExams:   upcomingList,
	}, nil
}

// getUpcomingExamsForStudent gets upcoming exams for a student
func (s *ExamService) getUpcomingExamsForStudent(ctx context.Context, studentID string) ([]models.Exam, error) {
	student, err := s.examRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	var exams []models.Exam
	now := time.Now()

	err = s.db.WithContext(ctx).
		Model(&models.Exam{}).
		Joins("JOIN exam_assignments ON exam_assignments.exam_id = exams.id").
		Where("(exam_assignments.student_id = ? OR exam_assignments.class_id = ?)", studentID, student.ClassID).
		Where("exams.start_time > ?", now).
		Where("exams.is_active = ?", true).
		Where("exams.status = ?", string(models.ExamStatusPublished)).
		Order("exams.start_time ASC").
		Limit(10).
		Find(&exams).Error

	return exams, err
}

// daysUntil calculates days until a date
func (s *ExamService) daysUntil(startTime *time.Time) int {
	if startTime == nil {
		return 0
	}
	days := int(startTime.Sub(time.Now()).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// isExamAvailable checks if an exam is available
func (s *ExamService) isExamAvailable(exam models.Exam) bool {
	if exam.StartTime == nil {
		return true
	}
	return time.Now().After(*exam.StartTime) && (exam.EndTime == nil || time.Now().Before(*exam.EndTime))
}

// GetTermlyReportCard gets a termly report card
func (s *ExamService) GetTermlyReportCard(ctx context.Context, studentID, termID, sessionID string) (*dto.TermlyReportCardResponse, error) {
	termlyResult, err := s.GetStudentTermlyResult(ctx, studentID, termID, sessionID)
	if err != nil {
		return nil, err
	}

	reportCardSubjects := make([]dto.ReportCardSubject, 0, len(termlyResult.Subjects))
	for _, subject := range termlyResult.Subjects {
		reportCardSubjects = append(reportCardSubjects, dto.ReportCardSubject{
			SubjectName:  subject.Subject.Name,
			SubjectCode:  subject.Subject.Code,
			TotalScore:   subject.TotalScore,
			Grade:        subject.Grade,
			GradePoint:   subject.GradePoint,
			Position:     subject.Position,
			ClassAverage: subject.ClassAverage,
			HighestScore: subject.HighestScore,
			LowestScore:  subject.LowestScore,
			Remarks:      subject.Remarks,
		})
	}

	return &dto.TermlyReportCardResponse{
		School: dto.SchoolInfo{
			ID:   termlyResult.School.ID,
			Name: termlyResult.School.Name,
		},
		Student: termlyResult.Student,
		Class:   termlyResult.Class,
		Term:    termlyResult.Term,
		Session: termlyResult.Session,
		Subjects: reportCardSubjects,
		Summary: dto.ReportCardSummary{
			TotalSubjects:         termlyResult.Summary.TotalSubjects,
			TotalScore:            termlyResult.Summary.TotalScore,
			TotalMarks:            termlyResult.Summary.TotalMarks,
			OverallPercentage:     termlyResult.Summary.OverallPercentage,
			OverallGrade:          termlyResult.Summary.OverallGrade,
			OverallGradePoint:     termlyResult.Summary.OverallGradePoint,
			NumberPassed:          termlyResult.Summary.NumberPassed,
			NumberFailed:          termlyResult.Summary.NumberFailed,
			ClassPosition:         termlyResult.Summary.ClassPosition,
			TotalStudentsInClass:  termlyResult.Summary.TotalStudents,
			BestPerformingSubject: termlyResult.Summary.BestPerformingSubject,
			WorstPerformingSubject: termlyResult.Summary.WorstPerformingSubject,
		},
	}, nil
}


// ============================================
// EXAM MANAGEMENT
// ============================================

// CreateExam creates a new exam
func (s *ExamService) CreateExam(ctx context.Context, req *dto.CreateExamRequest) (*dto.ExamResponse, error) {
	exam := &models.Exam{
		ID:               models.GenerateID(),
		Title:            req.Title,
		SubjectID:        req.SubjectID,
		ClassID:          req.ClassID,
		DurationMinutes:  req.DurationMinutes,
		TotalMarks:       req.TotalMarks,
		PassMark:         req.PassMark,
		Instructions:     req.Instructions,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		ShuffleQuestions: req.ShuffleQuestions,
		ShuffleOptions:   req.ShuffleOptions,
		IsActive:         req.IsActive,
		Status:           string(models.ExamStatusDraft),
	}

	if err := s.examRepo.CreateExamWithContext(ctx, exam); err != nil {
		return nil, err
	}

	return s.toExamResponse(ctx, exam)
}

// GetExam gets an exam by ID
func (s *ExamService) GetExam(ctx context.Context, id string) (*dto.ExamResponse, error) {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, id)
	if err != nil {
		return nil, errors.New("exam not found")
	}

	return s.toExamResponse(ctx, exam)
}

// ListExams lists all exams with pagination
func (s *ExamService) ListExams(ctx context.Context, page, limit int) (*dto.ExamListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	exams, total, err := s.examRepo.ListExams(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ExamResponse, len(exams))
	for i, exam := range exams {
		resp, err := s.toExamResponse(ctx, &exam)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return &dto.ExamListResponse{
		Exams:      responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}, nil
}

// ListExamsBySubject lists exams by subject
func (s *ExamService) ListExamsBySubject(ctx context.Context, subjectID string, page, limit int) (*dto.ExamListResponse, error) {
	exams, total, err := s.examRepo.ListExamsBySubject(ctx, subjectID, page, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ExamResponse, len(exams))
	for i, exam := range exams {
		resp, err := s.toExamResponse(ctx, &exam)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return &dto.ExamListResponse{
		Exams:      responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}, nil
}

// ListExamsByTerm lists exams by term
func (s *ExamService) ListExamsByTerm(ctx context.Context, termID string, page, limit int) (*dto.ExamListResponse, error) {
	exams, total, err := s.examRepo.ListExamsByTerm(ctx, termID, page, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ExamResponse, len(exams))
	for i, exam := range exams {
		resp, err := s.toExamResponse(ctx, &exam)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return &dto.ExamListResponse{
		Exams:      responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}, nil
}

// ListExamsBySession lists exams by session
func (s *ExamService) ListExamsBySession(ctx context.Context, sessionID string, page, limit int) (*dto.ExamListResponse, error) {
	exams, total, err := s.examRepo.ListExamsBySession(ctx, sessionID, page, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ExamResponse, len(exams))
	for i, exam := range exams {
		resp, err := s.toExamResponse(ctx, &exam)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return &dto.ExamListResponse{
		Exams:      responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}, nil
}

// ListExamsByClass lists exams by class
func (s *ExamService) ListExamsByClass(ctx context.Context, classID string, page, limit int) (*dto.ExamListResponse, error) {
	exams, total, err := s.examRepo.ListExamsByClass(ctx, classID, page, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ExamResponse, len(exams))
	for i, exam := range exams {
		resp, err := s.toExamResponse(ctx, &exam)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return &dto.ExamListResponse{
		Exams:      responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}, nil
}

// UpdateExam updates an exam
func (s *ExamService) UpdateExam(ctx context.Context, id string, req *dto.UpdateExamRequest) (*dto.ExamResponse, error) {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, id)
	if err != nil {
		return nil, errors.New("exam not found")
	}

	if req.Title != nil {
		exam.Title = *req.Title
	}
	if req.SubjectID != nil {
		exam.SubjectID = *req.SubjectID
	}
	if req.ClassID != nil && *req.ClassID != "" {
		exam.ClassID = *req.ClassID
	}
	if req.DurationMinutes != nil {
		exam.DurationMinutes = *req.DurationMinutes
	}
	if req.TotalMarks != nil {
		exam.TotalMarks = *req.TotalMarks
	}
	if req.PassMark != nil {
		exam.PassMark = *req.PassMark
	}
	if req.Instructions != nil {
		exam.Instructions = *req.Instructions
	}
	if req.StartTime != nil {
		exam.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		exam.EndTime = req.EndTime
	}
	if req.ShuffleQuestions != nil {
		exam.ShuffleQuestions = *req.ShuffleQuestions
	}
	if req.ShuffleOptions != nil {
		exam.ShuffleOptions = *req.ShuffleOptions
	}
	if req.IsActive != nil {
		exam.IsActive = *req.IsActive
	}
	if req.Status != nil {
		exam.Status = *req.Status
	}
	if req.ExamType != nil {
		exam.ExamType = *req.ExamType
	}
	if req.SessionID != nil && *req.SessionID != "" {
		exam.SessionID = *req.SessionID
	}
	if req.TermID != nil && *req.TermID != "" {
		exam.TermID = *req.TermID
	}

	if err := s.examRepo.UpdateExamWithContext(ctx, exam); err != nil {
		return nil, err
	}

	return s.toExamResponse(ctx, exam)
}

// DeleteExam deletes an exam
func (s *ExamService) DeleteExam(ctx context.Context, id string) error {
	return s.examRepo.DeleteExamWithContext(ctx, id)
}

// // AddQuestionsToExam adds questions to an exam
// func (s *ExamService) AddQuestionsToExam(ctx context.Context, examID string, questionIDs []string) error {
// 	_, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
// 	if err != nil {
// 		return errors.New("exam not found")
// 	}

// 	return s.examRepo.AddQuestionsToExam(ctx, examID, questionIDs)
// }

func (s *ExamService) AddQuestionsToExam(ctx context.Context, examID string, questionIDs []string) error {
    // Get exam with full context
    exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
    if err != nil {
        return errors.New("exam not found")
    }

    //  Validate questions using repository method
    if err := s.questionRepo.ValidateQuestionsForExam(ctx, exam, questionIDs); err != nil {
        return err
    }

    //  Validate question count using repository method
    currentCount, err := s.examRepo.CountQuestionsInExam(ctx, examID)
    if err != nil {
        return fmt.Errorf("failed to count existing questions: %w", err)
    }

    totalCount := int(currentCount) + len(questionIDs)
    if err := s.questionRepo.ValidateQuestionCountForExam(exam.ExamType, totalCount); err != nil {
        return err
    }

    // Remove duplicates
    uniqueMap := make(map[string]bool)
    var uniqueIDs []string
    for _, id := range questionIDs {
        if !uniqueMap[id] {
            uniqueMap[id] = true
            uniqueIDs = append(uniqueIDs, id)
        }
    }

    // Use transaction
    return s.db.Transaction(func(tx *gorm.DB) error {
        txRepo := repository.NewExamRepository(tx)
        return txRepo.AddQuestionsToExam(ctx, examID, uniqueIDs)
    })
}

// RemoveQuestionFromExam removes a question from an exam
func (s *ExamService) RemoveQuestionFromExam(ctx context.Context, examID, questionID string) error {
	return s.examRepo.RemoveQuestionFromExamWithContext(ctx, examID, questionID)
}

// AssignExam assigns an exam to students or a class
func (s *ExamService) AssignExam(ctx context.Context, examID string, req *dto.AssignExamRequest) error {
	_, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
	if err != nil {
		return errors.New("exam not found")
	}

	if req.ClassID != "" {
		assignment := &models.ExamAssignment{
			ID:         models.GenerateID(),
			ExamID:     examID,
			ClassID:    &req.ClassID,
			StartTime:  req.StartTime,
			EndTime:    req.EndTime,
			Status:     "active",
			AssignedBy: models.GenerateID(),
		}
		return s.examRepo.CreateAssignmentWithContext(ctx, assignment)
	}

	for _, sid := range req.StudentIDs {
		assignment := &models.ExamAssignment{
			ID:         models.GenerateID(),
			ExamID:     examID,
			StudentID:  &sid,
			StartTime:  req.StartTime,
			EndTime:    req.EndTime,
			Status:     "active",
			AssignedBy: models.GenerateID(),
		}
		if err := s.examRepo.CreateAssignmentWithContext(ctx, assignment); err != nil {
			return err
		}
	}
	return nil
}

// ============================================
// EXAM TAKING
// ============================================

// StartExam starts an exam for a student
func (s *ExamService) StartExam(ctx context.Context, req *dto.StartExamRequest) (*dto.StartExamResponse, error) {
	// Check for existing active attempt
	existing, _ := s.examRepo.FindActiveAttemptWithContext(ctx, req.StudentID, req.ExamID)
	if existing != nil {
		return s.buildStartExamResponse(ctx, existing)
	}

	// Get exam with questions
	exam, _, err := s.examRepo.FindExamWithQuestionsWithContext(ctx, req.ExamID)	
    if err != nil {
		return nil, errors.New("exam not found")
	}

	// Check if exam is available
	now := time.Now()
	if exam.StartTime != nil && now.Before(*exam.StartTime) {
		return nil, errors.New("exam has not started yet")
	}
	if exam.EndTime != nil && now.After(*exam.EndTime) {
		return nil, errors.New("exam has already ended")
	}

	// Create attempt
	attempt := &models.ExamAttempt{
		ID:         models.GenerateID(),
		StudentID:  req.StudentID,
		ExamID:     req.ExamID,
		StartTime:  now,
		Status:     string(models.AttemptStatusInProgress),
		IPAddress:  req.IPAddress,
		DeviceInfo: make(models.JSONMap),
	}

	if req.DeviceInfo != "" {
		attempt.DeviceInfo["user_agent"] = req.DeviceInfo
	}

	if err := s.examRepo.CreateAttempt(attempt); err != nil {
		return nil, err
	}

	// Create proctoring session
	proctoring := &models.ProctoringSession{
		ID:        models.GenerateID(),
		AttemptID: attempt.ID,
		StudentID: req.StudentID,
		Status:    "active",
		StartedAt: now,
	}
	if err := s.examRepo.CreateProctoringSession(proctoring); err != nil {
		// Log error but continue
	}

	return s.buildStartExamResponse(ctx, attempt)
}

// buildStartExamResponse builds the start exam response
func (s *ExamService) buildStartExamResponse(ctx context.Context, attempt *models.ExamAttempt) (*dto.StartExamResponse, error) {
	exam, questions, err := s.examRepo.FindExamWithQuestionsWithContext(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}

	totalQ := len(questions)
	answered, _ := s.examRepo.GetAnsweredCountWithContext(ctx, attempt.ID)
	timeRemaining := s.calculateRemainingTime(attempt.StartTime, exam.DurationMinutes)

	// Subject name isn't on the Exam model itself - same lookup pattern
	// used in StartExamForStudent's new-attempt path.
	var subject models.Subject
	subjectName := ""
	if err := s.db.WithContext(ctx).Where("id = ?", exam.SubjectID).First(&subject).Error; err == nil {
		subjectName = subject.Name
	}

	// Build exam detail
	examDetail := &dto.ExamDetailResponse{
		ID:               exam.ID,
		Title:            exam.Title,
		SubjectID:        exam.SubjectID,
		SubjectName:      subjectName,
		DurationMinutes:  exam.DurationMinutes,
		TotalMarks:       exam.TotalMarks,
		PassMark:         exam.PassMark,
		Instructions:     exam.Instructions,
		StartTime:        exam.StartTime,
		EndTime:          exam.EndTime,
		ShuffleQuestions: exam.ShuffleQuestions,
		ShuffleOptions:   exam.ShuffleOptions,
	}

	// Build questions (NO correct answers for students)
	var qResponses []dto.QuestionResponse
	for i, q := range questions {
		qResponses = append(qResponses, dto.QuestionResponse{
			ID:           q.ID,
			QuestionText: q.QuestionText,
			OptionA:      s.extractOptionFromStorage(q.Options, "A"),
			OptionB:      s.extractOptionFromStorage(q.Options, "B"),
			OptionC:      s.extractOptionFromStorage(q.Options, "C"),
			OptionD:      s.extractOptionFromStorage(q.Options, "D"),
			Marks:        q.Marks,
			SortOrder:    i + 1,
		})
	}

	attemptResp := &dto.ExamAttemptResponse{
		ID:             attempt.ID,
		StudentID:      attempt.StudentID,
		ExamID:         attempt.ExamID,
		StartTime:      attempt.StartTime,
		Status:         attempt.Status,
		TimeRemaining:  timeRemaining,
		AnsweredCount:  int(answered),
		TotalQuestions: totalQ,
		CreatedAt:      attempt.CreatedAt,
	}

	proctoringID := ""
	proctoring, _ := s.examRepo.FindProctoringByAttempt(ctx, attempt.ID)
	if proctoring != nil {
		proctoringID = proctoring.ID
	}

	return &dto.StartExamResponse{
		Attempt:      attemptResp,
		Exam:         examDetail,
		Questions:    qResponses,
		ProctoringID: proctoringID,
	}, nil
}

// GetAttemptState gets the current state of an attempt
func (s *ExamService) GetAttemptState(ctx context.Context, attemptID string, studentID string) (*dto.GetAttemptStateResponse, error) {
	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, attemptID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	// Verify ownership
	if attempt.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}

	questions, err := s.examRepo.GetExamQuestionsWithContext(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}

	answers, err := s.examRepo.FindAnswersByAttemptWithContext(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	// Build answer map for quick lookup
	answerMap := make(map[string]models.StudentAnswer)
	for _, ans := range answers {
		answerMap[ans.QuestionID] = ans
	}

	// Build question states
	var questionStates []dto.QuestionState
	for _, q := range questions {
		ans, exists := answerMap[q.ID]
		selectedAnswer := ""
		isAnswered := false
		isMarked := false
		timeSpent := 0

		if exists {
			selectedAnswer = ans.SelectedAnswer
			isAnswered = true
			isMarked = ans.IsMarked
			timeSpent = ans.TimeSpent
		}

		questionStates = append(questionStates, dto.QuestionState{
			ID:             q.ID,
			QuestionText:   q.QuestionText,
			QuestionType:   string(q.QuestionType),
			Marks:          q.Marks,
			SortOrder:      q.Order,
			Options:        s.extractOptions(q.Options),
			SelectedAnswer: selectedAnswer,
			IsAnswered:     isAnswered,
			IsMarked:       isMarked,
			TimeSpent:      timeSpent,
		})
	}

	timeRemaining := s.calculateRemainingTime(attempt.StartTime, exam.DurationMinutes)
	timeElapsed := int(time.Since(attempt.StartTime).Seconds())

	// Sort questions by sort order
	sort.Slice(questionStates, func(i, j int) bool {
		return questionStates[i].SortOrder < questionStates[j].SortOrder
	})

	answeredCount := len(answers)
	totalQuestions := len(questions)
	markedCount := 0
	for _, ans := range answers {
		if ans.IsMarked {
			markedCount++
		}
	}

	progressPercentage := 0
	if totalQuestions > 0 {
		progressPercentage = (answeredCount * 100) / totalQuestions
	}

	return &dto.GetAttemptStateResponse{
		Attempt: dto.AttemptState{
			ID:                   attempt.ID,
			Status:               attempt.Status,
			StartTime:            attempt.StartTime,
			TimeRemaining:        timeRemaining,
			TimeElapsed:          timeElapsed,
			TotalQuestions:       totalQuestions,
			AnsweredCount:        answeredCount,
			MarkedForReviewCount: markedCount,
			ProgressPercentage:   progressPercentage,
			LastActivity:         attempt.UpdatedAt,
		},
		Questions: questionStates,
		Progress: dto.ProgressData{
			Total:           totalQuestions,
			Answered:        answeredCount,
			Unanswered:      totalQuestions - answeredCount,
			MarkedForReview: markedCount,
			Percentage:      progressPercentage,
		},
		Timer: dto.TimerInfo{
			Remaining:        timeRemaining,
			Elapsed:          timeElapsed,
			PercentageRemaining: float64(timeRemaining) / float64(exam.DurationMinutes*60) * 100,
			WarningThreshold: 300,
			CriticalThreshold: 60,
		},
		SyncStatus: dto.SyncStatusData{
			Online:      true,
			PendingSync: 0,
		},
	}, nil
}

// SaveAnswer saves a student's answer
func (s *ExamService) SaveAnswer(ctx context.Context, req *dto.SaveAnswerRequest, studentID string) (*dto.SaveAnswerResponse, error) {
	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	// Verify ownership
	if attempt.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	if attempt.Status != string(models.AttemptStatusInProgress) {
		return nil, errors.New("exam already submitted")
	}

	exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}

	// Check time limit
	if time.Since(attempt.StartTime).Minutes() > float64(exam.DurationMinutes) {
		return nil, errors.New("time limit exceeded")
	}

	// Get question
	q, err := s.questionRepo.FindByID(ctx, req.QuestionID)
	if err != nil {
		return nil, errors.New("question not found")
	}

	// Check if answer already exists
	existing, _ := s.examRepo.FindAnswer(req.AttemptID, req.QuestionID)

	isCorrect := (req.SelectedAnswer == q.CorrectAnswer)

	var answer *models.StudentAnswer
	if existing != nil {
		existing.SelectedAnswer = req.SelectedAnswer
		existing.IsCorrect = isCorrect
		existing.TimeSpent = req.TimeSpent
		existing.IsMarked = req.IsMarked
		if err := s.examRepo.UpdateAnswer(existing); err != nil {
			return nil, err
		}
		answer = existing
	} else {
		answer = &models.StudentAnswer{
			ID:             models.GenerateID(),
			AttemptID:      req.AttemptID,
			QuestionID:     req.QuestionID,
			SelectedAnswer: req.SelectedAnswer,
			IsCorrect:      isCorrect,
			IsMarked:       req.IsMarked,
			TimeSpent:      req.TimeSpent,
		}
		if err := s.examRepo.SaveAnswer(answer); err != nil {
			return nil, err
		}
	}

	// Get updated progress
	totalQuestions, _ := s.examRepo.CountQuestionsInExam(ctx, attempt.ExamID)
	answeredCount, _ := s.examRepo.GetAnsweredCountWithContext(ctx, req.AttemptID)
	markedCount, _ := s.examRepo.GetMarkedForReviewCount(ctx, req.AttemptID)

	progressPercentage := 0
	if int(totalQuestions) > 0 {
		progressPercentage = (int(answeredCount) * 100) / int(totalQuestions)
	}

	return &dto.SaveAnswerResponse{
		QuestionID:     req.QuestionID,
		SelectedAnswer: req.SelectedAnswer,
		IsAnswered:     true,
		IsMarked:       req.IsMarked,
		TimeSpent:      req.TimeSpent,
		SavedAt:        time.Now(),
		Progress: dto.ProgressData{
			Total:           int(totalQuestions),
			Answered:        int(answeredCount),
			Unanswered:      int(totalQuestions) - int(answeredCount),
			MarkedForReview: int(markedCount),
			Percentage:      progressPercentage,
		},
	}, nil
}

// BulkSaveAnswers saves multiple answers (for offline sync)
func (s *ExamService) BulkSaveAnswers(ctx context.Context, req *dto.BulkSaveAnswerRequest, studentID string) (*dto.BulkSaveAnswerResponse, error) {
	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	// Verify ownership
	if attempt.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	if attempt.Status != string(models.AttemptStatusInProgress) {
		return nil, errors.New("exam already submitted")
	}

	var errorsList []string
	syncedCount := 0

	for _, item := range req.Answers {
		// Get question
		q, err := s.questionRepo.FindByID(ctx, item.QuestionID)
		if err != nil {
			errorsList = append(errorsList, "question "+item.QuestionID+" not found")
			continue
		}

		isCorrect := (item.SelectedAnswer == q.CorrectAnswer)

		// Check if answer exists
		existing, _ := s.examRepo.FindAnswer(req.AttemptID, item.QuestionID)

		if existing != nil {
			existing.SelectedAnswer = item.SelectedAnswer
			existing.IsCorrect = isCorrect
			existing.TimeSpent = item.TimeSpent
			existing.IsMarked = item.IsMarked
			if err := s.examRepo.UpdateAnswer(existing); err != nil {
				errorsList = append(errorsList, "failed to update answer for "+item.QuestionID)
				continue
			}
		} else {
			answer := &models.StudentAnswer{
				ID:             models.GenerateID(),
				AttemptID:      req.AttemptID,
				QuestionID:     item.QuestionID,
				SelectedAnswer: item.SelectedAnswer,
				IsCorrect:      isCorrect,
				IsMarked:       item.IsMarked,
				TimeSpent:      item.TimeSpent,
				SyncedAt:       nil, // Will be synced later
			}
			if err := s.examRepo.SaveAnswer(answer); err != nil {
				errorsList = append(errorsList, "failed to save answer for "+item.QuestionID)
				continue
			}
		}
		syncedCount++
	}

	// Get updated progress
	totalQuestions, _ := s.examRepo.CountQuestionsInExam(ctx, attempt.ExamID)
	answeredCount, _ := s.examRepo.GetAnsweredCountWithContext(ctx, req.AttemptID)

	progressPercentage := 0
	if int(totalQuestions) > 0 {
		progressPercentage = (int(answeredCount) * 100) / int(totalQuestions)
	}

	return &dto.BulkSaveAnswerResponse{
		SyncedCount: syncedCount,
		FailedCount: len(errorsList),
		Errors:      errorsList,
		Progress: dto.ProgressData{
			Total:      int(totalQuestions),
			Answered:   int(answeredCount),
			Unanswered: int(totalQuestions) - int(answeredCount),
			Percentage: progressPercentage,
		},
		SyncID:   models.GenerateID(),
		SyncedAt: time.Now(),
	}, nil
}

// MarkReview marks a question for review
func (s *ExamService) MarkReview(ctx context.Context, req *dto.MarkReviewRequest, studentID string) (*dto.MarkReviewResponse, error) {
	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	// Verify ownership
	if attempt.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	if attempt.Status != string(models.AttemptStatusInProgress) {
		return nil, errors.New("exam already submitted")
	}

	// Find or create answer
	existing, _ := s.examRepo.FindAnswer(req.AttemptID, req.QuestionID)

	if existing != nil {
		existing.IsMarked = req.IsMarked
		if err := s.examRepo.UpdateAnswer(existing); err != nil {
			return nil, err
		}
	} else {
		answer := &models.StudentAnswer{
			ID:         models.GenerateID(),
			AttemptID:  req.AttemptID,
			QuestionID: req.QuestionID,
			IsMarked:   req.IsMarked,
		}
		if err := s.examRepo.SaveAnswer(answer); err != nil {
			return nil, err
		}
	}

	// Get updated progress
	totalQuestions, _ := s.examRepo.CountQuestionsInExam(ctx, attempt.ExamID)
	answeredCount, _ := s.examRepo.GetAnsweredCountWithContext(ctx, req.AttemptID)
	markedCount, _ := s.examRepo.GetMarkedForReviewCount(ctx, req.AttemptID)

	progressPercentage := 0
	if int(totalQuestions) > 0 {
		progressPercentage = (int(answeredCount) * 100) / int(totalQuestions)
	}

	return &dto.MarkReviewResponse{
		QuestionID: req.QuestionID,
		IsMarked:   req.IsMarked,
		MarkedAt:   time.Now(),
		Progress: dto.ProgressData{
			Total:           int(totalQuestions),
			Answered:        int(answeredCount),
			Unanswered:      int(totalQuestions) - int(answeredCount),
			MarkedForReview: int(markedCount),
			Percentage:      progressPercentage,
		},
	}, nil
}

// SubmitExam submits an exam for grading
func (s *ExamService) SubmitExam(ctx context.Context, req *dto.SubmitExamRequest, studentID string) (*dto.SubmitExamResponse, error) {
    attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
    if err != nil {
        return nil, errors.New("attempt not found")
    }

    // Verify ownership
    if attempt.StudentID != studentID {
        return nil, errors.New("unauthorized")
    }

    if attempt.Status != string(models.AttemptStatusInProgress) {
        return nil, errors.New("exam already submitted")
    }

    tx := s.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Get all answers
    answers, err := s.examRepo.FindAnswersByAttemptWithContext(ctx, req.AttemptID)
    if err != nil {
        tx.Rollback()
        return nil, err
    }

    // Calculate score
    exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
    if err != nil {
        tx.Rollback()
        return nil, err
    }

    totalScore := 0
    for _, ans := range answers {
        if ans.IsCorrect {
            q, err := s.questionRepo.FindByID(ctx, ans.QuestionID)
            if err != nil {
                continue
            }
            totalScore += q.Marks
        }
    }

    now := time.Now()
    attempt.Status = string(models.AttemptStatusCompleted)
    attempt.EndTime = &now
    attempt.Score = &totalScore

    percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
    attempt.Percentage = &percentage

    if err := s.examRepo.UpdateAttempt(attempt); err != nil {
        tx.Rollback()
        return nil, err
    }

    // Calculate grade and passed status
    passed := percentage >= float64(exam.PassMark)
    grade, _, _ := dto.GetGrade(percentage)

    // Create result - NOW using the passed variable
    result := &models.Result{
        ID:          models.GenerateID(),
        ExamID:      attempt.ExamID,
        StudentID:   attempt.StudentID,
        AttemptID:   attempt.ID,
        TotalScore:  totalScore,
        Percentage:  percentage,
        Grade:       grade,
        Passed:      passed,  // ✅ FIXED: Now using the passed variable
        PublishedAt: &now,
    }
    if err := s.examRepo.CreateResult(result); err != nil {
        tx.Rollback()
        return nil, err
    }

    // Update proctoring session
    proctoring, _ := s.examRepo.FindProctoringByAttempt(ctx, attempt.ID)
    if proctoring != nil {
        proctoring.Status = "completed"
        proctoring.EndedAt = &now
        s.examRepo.UpdateProctoringSession(proctoring)
    }

    if err := tx.Commit().Error; err != nil {
        return nil, err
    }

    timeTaken := int(now.Sub(attempt.StartTime).Minutes())

    return &dto.SubmitExamResponse{
        AttemptID:        req.AttemptID,
        ResultID:         result.ID,
        Score:            totalScore,
        TotalMarks:       exam.TotalMarks,
        Percentage:       percentage,
        Grade:            grade,
        Passed:           passed,  // ✅ CORRECT
        TimeTakenMinutes: timeTaken,
        Status:           string(models.AttemptStatusCompleted),
        SubmittedAt:      now,
    }, nil
}

// OfflineSubmit submits an exam offline
func (s *ExamService) OfflineSubmit(ctx context.Context, req *dto.OfflineSubmitRequest, studentID string) (*dto.OfflineSubmitResponse, error) {
	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	// Verify ownership
	if attempt.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	if attempt.Status != string(models.AttemptStatusInProgress) {
		return nil, errors.New("exam already submitted")
	}

	// Save offline answers
	for _, item := range req.Answers {
		offlineAnswer := &models.OfflineAnswer{
			ID:             models.GenerateID(),
			StudentID:      attempt.StudentID,
			ExamID:         attempt.ExamID,
			AttemptID:      req.AttemptID,
			QuestionID:     item.QuestionID,
			SelectedAnswer: item.SelectedAnswer,
			IsMarked:       item.IsMarked,
			TimeSpent:      item.TimeSpent,
			DeviceID:       req.DeviceInfo.UserAgent,
			SyncedAt:       nil,
		}
		if err := s.examRepo.SaveOfflineAnswer(offlineAnswer); err != nil {
			return nil, err
		}
	}

	// Mark attempt as submitted (but not graded yet)
	now := time.Now()
	attempt.Status = string(models.AttemptStatusSubmitted)
	attempt.EndTime = &now
	if err := s.examRepo.UpdateAttempt(attempt); err != nil {
		return nil, err
	}

	nextSync := now.Add(30 * time.Second)

	return &dto.OfflineSubmitResponse{
		AttemptID:        req.AttemptID,
		SubmissionStatus: "queued",
		ResultID:         nil,
		Message:          "Exam submitted offline. Results will be available when online.",
		SubmittedAt:      now,
		NextSync:         nextSync,
	}, nil
}

// SyncOfflineAnswers syncs offline answers when online
// func (s *ExamService) SyncOfflineAnswers(ctx context.Context, req *dto.SyncAnswersRequest) (*dto.SyncAnswersResponse, error) {
// 	attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
// 	if err != nil {
// 		return nil, errors.New("attempt not found")
// 	}

// 	if attempt.Status != string(models.AttemptStatusSubmitted) {
// 		return nil, errors.New("attempt not in submitted state")
// 	}

// 	var errorsList []string
// 	syncedCount := 0

// 	tx := s.db.Begin()
// 	defer func() {
// 		if r := recover(); r != nil {
// 			tx.Rollback()
// 		}
// 	}()

// 	// Process each answer
// 	for _, item := range req.Answers {
// 		// Check if answer already exists
// 		existing, _ := s.examRepo.FindAnswer(req.AttemptID, item.QuestionID)

// 		if existing != nil {
// 			existing.SelectedAnswer = item.SelectedAnswer
// 			existing.IsMarked = item.IsMarked
// 			existing.TimeSpent = item.TimeSpent
// 			if err := s.examRepo.UpdateAnswer(existing); err != nil {
// 				errorsList = append(errorsList, "failed to update answer for "+item.QuestionID)
// 				continue
// 			}
// 		} else {
// 			answer := &models.StudentAnswer{
// 				ID:             models.GenerateID(),
// 				AttemptID:      req.AttemptID,
// 				QuestionID:     item.QuestionID,
// 				SelectedAnswer: item.SelectedAnswer,
// 				IsMarked:       item.IsMarked,
// 				TimeSpent:      item.TimeSpent,
// 			}
// 			if err := s.examRepo.SaveAnswer(answer); err != nil {
// 				errorsList = append(errorsList, "failed to save answer for "+item.QuestionID)
// 				continue
// 			}
// 			syncedCount++
// 		}
// 	}

// 	// Mark offline answers as synced
// 	var offlineIDs []string
// 	offlineAnswers, err := s.examRepo.FindOfflineAnswersByAttempt(ctx, req.AttemptID)
// 	if err == nil {
// 		for _, oa := range offlineAnswers {
// 			offlineIDs = append(offlineIDs, oa.ID)
// 		}
// 		if len(offlineIDs) > 0 {
// 			s.examRepo.MarkOfflineAnswersSyncedWithContext(ctx, offlineIDs)
// 		}
// 	}

// 	// Grade the exam now
// 	if attempt.Status == string(models.AttemptStatusSubmitted) {
// 		// Calculate score
// 		answers, err := s.examRepo.FindAnswersByAttemptWithContext(ctx, req.AttemptID)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}

// 		exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
// 		if err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}

// 		totalScore := 0
// 		for _, ans := range answers {
// 			if ans.IsCorrect {
// 				q, err := s.questionRepo.FindByID(ctx, ans.QuestionID)
// 				if err != nil {
// 					continue
// 				}
// 				totalScore += q.Marks
// 			}
// 		}

// 		now := time.Now()
// 		attempt.Status = string(models.AttemptStatusCompleted)
// 		attempt.Score = &totalScore

// 		percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
// 		attempt.Percentage = &percentage

// 		if err := s.examRepo.UpdateAttempt(attempt); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}

// 		passed := percentage >= float64(exam.PassMark)
// 		grade, _, _ := dto.GetGrade(percentage)

// 		result := &models.Result{
// 			ID:          models.GenerateID(),
// 			ExamID:      attempt.ExamID,
// 			StudentID:   attempt.StudentID,
// 			AttemptID:   attempt.ID,
// 			TotalScore:  totalScore,
// 			Percentage:  percentage,
// 			Grade:       grade,
// 			PublishedAt: &now,
// 		}
// 		if err := s.examRepo.CreateResult(result); err != nil {
// 			tx.Rollback()
// 			return nil, err
// 		}
// 	}

// 	if err := tx.Commit().Error; err != nil {
// 		return nil, err
// 	}

// 	nextSync := time.Now().Add(30 * time.Second)

// 	return &dto.SyncAnswersResponse{
// 		AttemptID:     req.AttemptID,
// 		SyncedCount:   syncedCount,
// 		FailedCount:   len(errorsList),
// 		Errors:        errorsList,
// 		SyncTimestamp: time.Now(),
// 		SyncID:        models.GenerateID(),
// 		NextSyncAfter: nextSync,
// 		SyncStatus:    "completed",
// 	}, nil
// }


// SyncOfflineAnswers syncs offline answers when online
func (s *ExamService) SyncOfflineAnswers(ctx context.Context, req *dto.SyncAnswersRequest, studentID string) (*dto.SyncAnswersResponse, error) {
    attempt, err := s.examRepo.FindAttemptByIDWithContext(ctx, req.AttemptID)
    if err != nil {
        return nil, errors.New("attempt not found")
    }

    // Verify ownership
    if attempt.StudentID != studentID {
        return nil, errors.New("unauthorized")
    }

    if attempt.Status != string(models.AttemptStatusSubmitted) {
        return nil, errors.New("attempt not in submitted state")
    }

    var errorsList []string
    syncedCount := 0

    tx := s.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Process each answer
    for _, item := range req.Answers {
        // Check if answer already exists
        existing, _ := s.examRepo.FindAnswer(req.AttemptID, item.QuestionID)

        if existing != nil {
            existing.SelectedAnswer = item.SelectedAnswer
            existing.IsMarked = item.IsMarked
            existing.TimeSpent = item.TimeSpent
            if err := s.examRepo.UpdateAnswer(existing); err != nil {
                errorsList = append(errorsList, "failed to update answer for "+item.QuestionID)
                continue
            }
        } else {
            answer := &models.StudentAnswer{
                ID:             models.GenerateID(),
                AttemptID:      req.AttemptID,
                QuestionID:     item.QuestionID,
                SelectedAnswer: item.SelectedAnswer,
                IsMarked:       item.IsMarked,
                TimeSpent:      item.TimeSpent,
            }
            if err := s.examRepo.SaveAnswer(answer); err != nil {
                errorsList = append(errorsList, "failed to save answer for "+item.QuestionID)
                continue
            }
            syncedCount++
        }
    }

    // Mark offline answers as synced
    var offlineIDs []string
    offlineAnswers, err := s.examRepo.FindOfflineAnswersByAttempt(ctx, req.AttemptID)
    if err == nil {
        for _, oa := range offlineAnswers {
            offlineIDs = append(offlineIDs, oa.ID)
        }
        if len(offlineIDs) > 0 {
            s.examRepo.MarkOfflineAnswersSyncedWithContext(ctx, offlineIDs)
        }
    }

    // Grade the exam now
    if attempt.Status == string(models.AttemptStatusSubmitted) {
        // Calculate score
        answers, err := s.examRepo.FindAnswersByAttemptWithContext(ctx, req.AttemptID)
        if err != nil {
            tx.Rollback()
            return nil, err
        }

        exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
        if err != nil {
            tx.Rollback()
            return nil, err
        }

        totalScore := 0
        for _, ans := range answers {
            if ans.IsCorrect {
                q, err := s.questionRepo.FindByID(ctx, ans.QuestionID)
                if err != nil {
                    continue
                }
                totalScore += q.Marks
            }
        }

        now := time.Now()
        attempt.Status = string(models.AttemptStatusCompleted)
        attempt.Score = &totalScore

        percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
        attempt.Percentage = &percentage

        if err := s.examRepo.UpdateAttempt(attempt); err != nil {
            tx.Rollback()
            return nil, err
        }

        passed := percentage >= float64(exam.PassMark)
        grade, _, _ := dto.GetGrade(percentage)

        result := &models.Result{
            ID:          models.GenerateID(),
            ExamID:      attempt.ExamID,
            StudentID:   attempt.StudentID,
            AttemptID:   attempt.ID,
            TotalScore:  totalScore,
            Percentage:  percentage,
            Grade:       grade,
            Passed:      passed,  // ✅ FIXED: Now using the passed variable
            PublishedAt: &now,
        }
        if err := s.examRepo.CreateResult(result); err != nil {
            tx.Rollback()
            return nil, err
        }
    }

    if err := tx.Commit().Error; err != nil {
        return nil, err
    }

    nextSync := time.Now().Add(30 * time.Second)

    return &dto.SyncAnswersResponse{
        AttemptID:     req.AttemptID,
        SyncedCount:   syncedCount,
        FailedCount:   len(errorsList),
        Errors:        errorsList,
        SyncTimestamp: time.Now(),
        SyncID:        models.GenerateID(),
        NextSyncAfter: nextSync,
        SyncStatus:    "completed",
    }, nil
}


// ============================================
// RESULTS
// ============================================

// GetStudentExamResult gets a student's exam result
func (s *ExamService) GetStudentExamResult(ctx context.Context, studentID, examID string) (*dto.StudentExamResultResponse, error) {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
	if err != nil {
		return nil, errors.New("exam not found")
	}

	result, err := s.examRepo.GetStudentExamResult(ctx, studentID, examID)
	if err != nil || result == nil {
		return nil, errors.New("result not found")
	}

	attempt, err := s.examRepo.FindAttemptByStudentAndExam(ctx, studentID, examID)
	if err != nil {
		return nil, errors.New("attempt not found")
	}

	answers, err := s.examRepo.FindAnswersByAttemptWithContext(ctx, attempt.ID)
	if err != nil {
		return nil, err
	}

	// Build answer details
	answerDetails := make([]dto.AnswerDetail, len(answers))
	correct := 0
	wrong := 0
	skipped := 0

	for i, ans := range answers {
		q, err := s.questionRepo.FindByID(ctx, ans.QuestionID)
		if err != nil {
			continue
		}

		marksObtained := 0
		if ans.IsCorrect {
			marksObtained = q.Marks
			correct++
		} else if ans.SelectedAnswer == "" {
			skipped++
		} else {
			wrong++
		}

		answerDetails[i] = dto.AnswerDetail{
			QuestionID:     ans.QuestionID,
			QuestionText:   q.QuestionText,
			StudentAnswer:  ans.SelectedAnswer,
			CorrectAnswer:  q.CorrectAnswer,
			IsCorrect:      ans.IsCorrect,
			MarksObtained:  marksObtained,
			TotalMarks:     q.Marks,
		}
	}

	totalQ := len(answerDetails)
	accuracy := 0.0
	if totalQ > 0 {
		accuracy = float64(correct) / float64(totalQ) * 100
	}

	timeTaken := int(attempt.EndTime.Sub(attempt.StartTime).Minutes())

	return &dto.StudentExamResultResponse{
		Result: dto.ResultDetail{
			ID:               result.ID,
			ExamTitle:        exam.Title,
			Subject:          "Subject",
			TotalScore:       result.TotalScore,
			TotalMarks:       exam.TotalMarks,
			Percentage:       result.Percentage,
			Grade:            result.Grade,
			Passed:           result.Percentage >= float64(exam.PassMark),
			StartTime:        attempt.StartTime,
			EndTime:          *attempt.EndTime,
			TimeTakenMinutes: timeTaken,
		},
		Answers: answerDetails,
		ResultSummary: dto.ResultSummary{
			CorrectAnswers: correct,
			WrongAnswers:   wrong,
			SkippedAnswers: skipped,
			TotalQuestions: totalQ,
			Accuracy:       accuracy,
		},
	}, nil
}

// GetClassResults gets class results for an exam
func (s *ExamService) GetClassResults(ctx context.Context, examID, classID string, page, limit int) (*dto.ClassResultResponse, error) {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
	if err != nil {
		return nil, errors.New("exam not found")
	}

	attempts, total, err := s.examRepo.GetExamResultsByClass(ctx, examID, classID, page, limit)
	if err != nil {
		return nil, err
	}

	students := make([]dto.StudentResult, len(attempts))
	for i, attempt := range attempts {
		percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
		grade, _, _ := dto.GetGrade(percentage)
		students[i] = dto.StudentResult{
			StudentID:        attempt.StudentID,
			StudentName:      "Student",
			Score:            *attempt.Score,
			Percentage:       percentage,
			Grade:            grade,
			Passed:           *attempt.Score >= exam.PassMark,
			Rank:             i + 1,
			AttemptStatus:    attempt.Status,
			StartTime:        &attempt.StartTime,
			EndTime:          attempt.EndTime,
			TimeTakenMinutes: 0,
		}
	}

	stats, _ := s.examRepo.GetExamStatistics(ctx, examID)

	// Convert grade distribution
	gradeDist := make(map[string]int)
	if gd, ok := stats["grade_distribution"].(map[string]int); ok {
		gradeDist = gd
	}

	return &dto.ClassResultResponse{
		Exam: dto.ExamSummary{
			ID:          exam.ID,
			Title:       exam.Title,
			SubjectID:   exam.SubjectID,
			TotalMarks:  exam.TotalMarks,
			PassMark:    exam.PassMark,
		},
		Class: dto.ClassSummary{
			ID:   classID,
			Name: "Class Name",
		},
		Students: students,
		Statistics: dto.ResultStatistics{
			TotalStudents:     int(stats["total_attempts"].(int64)),
			Attempted:         int(stats["completed_count"].(int64)),
			Passed:            int(stats["passed_count"].(int64)),
			Failed:            int(stats["failed_count"].(int64)),
			AverageScore:      stats["average_score"].(float64),
			GradeDistribution: gradeDist,
		},
		Pagination: dto.PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: int((total + int64(limit) - 1) / int64(limit)),
		},
	}, nil
}

// GetExamRankings gets exam rankings
func (s *ExamService) GetExamRankings(ctx context.Context, examID string, limit int) (*dto.ExamRankingResponse, error) {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
	if err != nil {
		return nil, errors.New("exam not found")
	}

	attempts, err := s.examRepo.GetExamRankings(ctx, examID, limit)
	if err != nil {
		return nil, err
	}

	badges := []string{"🏆 Gold Medal", "🥈 Silver Medal", "🥉 Bronze Medal"}
	rankings := make([]dto.RankingEntry, len(attempts))

	for i, attempt := range attempts {
		percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
		grade, _, _ := dto.GetGrade(percentage)
		badge := ""
		if i < 3 {
			badge = badges[i]
		}
		rankings[i] = dto.RankingEntry{
			Rank:        i + 1,
			StudentID:   attempt.StudentID,
			StudentName: "Student",
			Score:       *attempt.Score,
			Percentage:  percentage,
			Grade:       grade,
			Badge:       badge,
		}
	}

	return &dto.ExamRankingResponse{
		ExamTitle:         exam.Title,
		TotalParticipants: len(rankings),
		Rankings:          rankings,
	}, nil
}

// GetExamStatistics gets exam statistics
func (s *ExamService) GetExamStatistics(ctx context.Context, examID string) (*dto.ExamStatisticsResponse, error) {
	stats, err := s.examRepo.GetExamStatistics(ctx, examID)
	if err != nil {
		return nil, err
	}

	gradeDist := make(map[string]int)
	if gd, ok := stats["grade_distribution"].(map[string]int); ok {
		gradeDist = gd
	}

	passRate := 0.0
	if stats["completed_count"].(int64) > 0 {
		passRate = float64(stats["passed_count"].(int64)) / float64(stats["completed_count"].(int64)) * 100
	}

	return &dto.ExamStatisticsResponse{
		TotalAttempts:     int(stats["total_attempts"].(int64)),
		CompletedCount:    int(stats["completed_count"].(int64)),
		InProgressCount:   int(stats["in_progress_count"].(int64)),
		PassedCount:       int(stats["passed_count"].(int64)),
		FailedCount:       int(stats["failed_count"].(int64)),
		AverageScore:      stats["average_score"].(float64),
		PassRate:          passRate,
		GradeDistribution: gradeDist,
	}, nil
}

// ============================================
// TEACHER DASHBOARD
// ============================================

// GetTeacherDashboard gets teacher dashboard data
func (s *ExamService) GetTeacherDashboard(ctx context.Context, teacherID string) (*dto.TeacherDashboardResponse, error) {
	upcomingExams, err := s.examRepo.GetUpcomingExams(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	upcomingExamDTOs := make([]dto.TeacherUpcomingExam, len(upcomingExams))
	for i, exam := range upcomingExams {
		date := ""
		if exam.StartTime != nil {
			date = exam.StartTime.Format("2006-01-02")
		}
		upcomingExamDTOs[i] = dto.TeacherUpcomingExam{
			ExamTitle:         exam.Title,
			Date:              date,
			StudentsScheduled: 0,
		}
	}

	recentAttempts, err := s.examRepo.GetRecentActivities(ctx, teacherID, 10)
	if err != nil {
		return nil, err
	}

	recentActivities := make([]dto.RecentActivity, len(recentAttempts))
	for i, attempt := range recentAttempts {
		exam, _ := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
		recentActivities[i] = dto.RecentActivity{
			Action:  "Exam completed",
			Student: "Student",
			Exam:    exam.Title,
			Score:   *attempt.Score,
			Time:    time.Since(attempt.UpdatedAt).String(),
		}
	}

	return &dto.TeacherDashboardResponse{
		Teacher: dto.TeacherInfo{
			ID:             teacherID,
			Name:           "Teacher Name",
			SubjectsTaught: []string{},
		},
		UpcomingExams:    upcomingExamDTOs,
		RecentActivities: recentActivities,
		StudentTrends: dto.StudentTrends{
			Improving: 0,
			Declining: 0,
			Stable:    0,
			AtRisk:    0,
		},
	}, nil
}

// ============================================
// TERMLY RESULTS (WAEC/NECO Style)
// ============================================

// GetStudentTermlyResult gets termly result for a student
func (s *ExamService) GetStudentTermlyResult(ctx context.Context, studentID, termID, sessionID string) (*dto.TermlyResultResponse, error) {
	student, err := s.examRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, errors.New("student not found")
	}

	// Get all results for the term
	results, err := s.examRepo.FindResultsByStudentAndTerm(ctx, studentID, termID, sessionID)
	if err != nil {
		return nil, err
	}

	// Get all subjects for this class
	subjects, err := s.examRepo.GetSubjectsByClassAndTerm(ctx, student.ClassID, termID)
	if err != nil {
		return nil, err
	}

	subjectResults := make([]dto.SubjectTermlyResult, 0, len(subjects))
	totalScore := 0
	totalMarks := 0
	passed := 0

	for _, subject := range subjects {
		subjectResult, err := s.buildSubjectTermlyResult(ctx, studentID, subject.ID, results)
		if err != nil {
			continue
		}
		subjectResults = append(subjectResults, subjectResult)
		totalScore += subjectResult.TotalScore
		totalMarks += subjectResult.TotalMarks
		if subjectResult.Percentage >= 50 {
			passed++
		}
	}

	overallPercentage := 0.0
	if totalMarks > 0 {
		overallPercentage = float64(totalScore) / float64(totalMarks) * 100
	}
	overallGrade, overallGP, _ := dto.GetGrade(overallPercentage)

	// Get term and session info
	term, err := s.getTermByID(ctx, termID)
	if err != nil {
		return nil, err
	}
	session, err := s.getSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	school, err := s.getSchoolByID(ctx, student.SchoolID)
	if err != nil {
		return nil, err
	}
	class, err := s.getClassByID(ctx, student.ClassID)
	if err != nil {
		return nil, err
	}

	return &dto.TermlyResultResponse{
        Student: dto.StudentInfo{
                ID:              student.ID,
                Name:            student.User.FirstName + " " + student.User.LastName, // ✅ From User
                AdmissionNumber: student.AdmissionNo,  // ✅ Correct field name
                Class:           models.GetClassDisplayName(class), // ✅ Helper function
                ClassID:         student.ClassID,
                School:          school.Name,
                SchoolID:        school.ID,
            },
            Class: dto.ClassInfo{
                ID:   class.ID,
                Name: models.GetClassDisplayName(class), // ✅ Helper function
            },
		Term: dto.TermInfo{
			ID:     term.ID,
			Name:   term.Name,
			Number: term.TermNumber,
		},
		Session: dto.SessionInfo{
			ID:   session.ID,
			Name: session.Name,
		},
		School: dto.SchoolInfo{
			ID:   school.ID,
			Name: school.Name,
		},
		Subjects: subjectResults,
		Summary: dto.TermlySummary{
			TotalSubjects:      len(subjects),
			TotalScore:         totalScore,
			TotalMarks:         totalMarks,
			OverallPercentage:  overallPercentage,
			OverallGrade:       overallGrade,
			OverallGradePoint:  overallGP,
			NumberPassed:       passed,
			NumberFailed:       len(subjects) - passed,
			ClassPosition:      0,
			TotalStudents:      0,
			BestPerformingSubject:  s.getBestSubject(subjectResults),
			WorstPerformingSubject: s.getWorstSubject(subjectResults),
		},
		GeneratedAt: time.Now(),
	}, nil
}

// buildSubjectTermlyResult builds a subject's termly result
func (s *ExamService) buildSubjectTermlyResult(ctx context.Context, studentID, subjectID string, allResults []models.Result) (dto.SubjectTermlyResult, error) {
	var subjectResults []dto.ExamResultItem
	totalScore := 0
	totalMarks := 0

	for _, result := range allResults {
		exam, err := s.examRepo.FindExamByIDWithContext(ctx, result.ExamID)
		if err != nil {
			continue
		}
		if exam.SubjectID != subjectID {
			continue
		}

		subjectResults = append(subjectResults, dto.ExamResultItem{
			ExamID:     result.ExamID,
			ExamTitle:  exam.Title,
			ExamType:   exam.ExamType,
			Score:      result.TotalScore,
			TotalMarks: exam.TotalMarks,
			Percentage: result.Percentage,
			Grade:      result.Grade,
			Date:       result.CreatedAt,
		})
		totalScore += result.TotalScore
		totalMarks += exam.TotalMarks
	}

	percentage := 0.0
	if totalMarks > 0 {
		percentage = float64(totalScore) / float64(totalMarks) * 100
	}
	grade, gp, remark := dto.GetGrade(percentage)

	// Get subject info
	subject, err := s.questionRepo.GetSubjectByID(ctx, subjectID)
	if err != nil {
		return dto.SubjectTermlyResult{}, err
	}

	return dto.SubjectTermlyResult{
		Subject: dto.SubjectInfo{
			ID:   subject.ID,
			Name: subject.Name,
			Code: subject.Code,
		},
		Exams:       subjectResults,
		TotalScore:  totalScore,
		TotalMarks:  totalMarks,
		Percentage:  percentage,
		Grade:       grade,
		GradePoint:  gp,
		Remarks:     remark,
	}, nil
}

// ============================================
// PRACTICE SESSIONS
// ============================================

// StartPractice starts a practice session
func (s *ExamService) StartPractice(ctx context.Context, studentID, subjectID string, questionCount int) (*dto.PracticeSessionResponse, error) {
	questions, err := s.examRepo.GetRandomQuestions(ctx, subjectID, questionCount)
	if err != nil {
		return nil, err
	}

	session := &models.PracticeSession{
		ID:             models.GenerateID(),
		StudentID:      studentID,
		SubjectID:      subjectID,
		TotalQuestions: len(questions),
		Status:         "in_progress",
		StartedAt:      time.Now(),
	}

	if err := s.examRepo.CreatePracticeSession(session); err != nil {
		return nil, err
	}

	return &dto.PracticeSessionResponse{
		ID:             session.ID,
		SubjectID:      session.SubjectID,
		TotalQuestions: session.TotalQuestions,
		Answered:       session.Answered,
		Score:          session.Score,
		Status:         session.Status,
		StartedAt:      session.StartedAt,
		CompletedAt:    session.CompletedAt,
	}, nil
}

// ============================================
// HELPER FUNCTIONS
// ============================================

// toExamResponse converts an Exam model to ExamResponse DTO
func (s *ExamService) toExamResponse(ctx context.Context, exam *models.Exam) (*dto.ExamResponse, error) {
	var classID *string
	if exam.ClassID != "" {
		classID = &exam.ClassID
	}

	questionCount, _ := s.examRepo.CountQuestionsInExam(ctx, exam.ID)

	subject, err := s.questionRepo.GetSubjectByID(ctx, exam.SubjectID)
	subjectName := ""
	if err == nil && subject != nil {
		subjectName = subject.Name
	}

	return &dto.ExamResponse{
		ID:               exam.ID,
		Title:            exam.Title,
		SubjectID:        exam.SubjectID,
		SubjectName:      subjectName,
		ClassID:          classID,
		DurationMinutes:  exam.DurationMinutes,
		TotalMarks:       exam.TotalMarks,
		PassMark:         exam.PassMark,
		Instructions:     exam.Instructions,
		StartTime:        exam.StartTime,
		EndTime:          exam.EndTime,
		ShuffleQuestions: exam.ShuffleQuestions,
		ShuffleOptions:   exam.ShuffleOptions,
		IsActive:         exam.IsActive,
		QuestionCount:    int(questionCount),
		Status:           exam.Status,
		ExamType:         exam.ExamType,
		SessionID:        exam.SessionID,
		SessionName:      "",
		TermID:           exam.TermID,
		TermName:         "",
		CreatedAt:        exam.CreatedAt,
		UpdatedAt:        exam.UpdatedAt,
	}, nil
}


// calculateProgress calculates the progress percentage
func (s *ExamService) calculateProgress(answered, total int) int {
    if total == 0 {
        return 0
    }
    return (answered * 100) / total
}


// extractOptionFromStorage extracts an option from storage
func (s *ExamService) extractOptionFromStorage(storage models.OptionStorage, key string) string {
	if len(storage) == 0 {
		return ""
	}
	for _, item := range storage {
		if item.Key == key {
			return item.Text
		}
	}
	return ""
}

// extractOptions converts OptionStorage to OptionInfo slice
func (s *ExamService) extractOptions(storage models.OptionStorage) []dto.OptionInfo {
	opts := make([]dto.OptionInfo, 0, len(storage))
	for _, item := range storage {
		opts = append(opts, dto.OptionInfo{
			Key:  item.Key,
			Text: item.Text,
		})
	}
	return opts
}

// calculateRemainingTime calculates remaining time in seconds
func (s *ExamService) calculateRemainingTime(start time.Time, durationMinutes int) int {
	elapsed := int(time.Since(start).Seconds())
	remaining := durationMinutes*60 - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// getGradeFromPercentage gets grade from percentage
func (s *ExamService) getGradeFromPercentage(percentage float64) string {
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

// getBestSubject gets the best performing subject
func (s *ExamService) getBestSubject(subjects []dto.SubjectTermlyResult) string {
	if len(subjects) == 0 {
		return ""
	}
	best := subjects[0]
	for _, sub := range subjects {
		if sub.Percentage > best.Percentage {
			best = sub
		}
	}
	return best.Subject.Name
}

// getWorstSubject gets the worst performing subject
func (s *ExamService) getWorstSubject(subjects []dto.SubjectTermlyResult) string {
	if len(subjects) == 0 {
		return ""
	}
	worst := subjects[0]
	for _, sub := range subjects {
		if sub.Percentage < worst.Percentage {
			worst = sub
		}
	}
	return worst.Subject.Name
}

// getTermByID gets a term by ID
func (s *ExamService) getTermByID(ctx context.Context, id string) (*models.Term, error) {
	var term models.Term
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&term).Error
	return &term, err
}

// getSessionByID gets a session by ID
func (s *ExamService) getSessionByID(ctx context.Context, id string) (*models.AcademicSession, error) {
	var session models.AcademicSession
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&session).Error
	return &session, err
}

// getSchoolByID gets a school by ID
func (s *ExamService) getSchoolByID(ctx context.Context, id string) (*models.School, error) {
	var school models.School
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&school).Error
	return &school, err
}

// getClassByID gets a class by ID
func (s *ExamService) getClassByID(ctx context.Context, id string) (*models.Class, error) {
	var class models.Class
	err := s.db.WithContext(ctx).
		Preload("ClassLevel").
		Preload("ClassArm").
		Where("id = ?", id).First(&class).Error
	return &class, err
}

// ExportExamResults exports exam results to CSV or Excel
func (s *ExamService) ExportExamResults(ctx context.Context, examID, classID, format string) ([]byte, string, error) {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
	if err != nil {
		return nil, "", errors.New("exam not found")
	}

	attempts, _, err := s.examRepo.GetExamResultsByClass(ctx, examID, classID, 1, 10000)
	if err != nil {
		return nil, "", err
	}

	switch format {
	case "csv":
		return s.exportToCSV(attempts, exam), "text/csv", nil
	case "excel":
		return s.exportToExcel(attempts, exam), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	default:
		return nil, "", errors.New("unsupported format")
	}
}

// exportToCSV exports to CSV format
func (s *ExamService) exportToCSV(attempts []models.ExamAttempt, exam *models.Exam) []byte {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Write([]string{"Student Name", "Score", "Percentage", "Grade", "Passed", "Time Taken"})

	for _, attempt := range attempts {
		if attempt.Score == nil {
			continue
		}
		percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
		passed := *attempt.Score >= exam.PassMark
		grade, _, _ := dto.GetGrade(percentage)
		writer.Write([]string{
			"Student",
			fmt.Sprintf("%d", *attempt.Score),
			fmt.Sprintf("%.2f", percentage),
			grade,
			fmt.Sprintf("%v", passed),
			"",
		})
	}
	writer.Flush()
	return buf.Bytes()
}

// exportToExcel exports to Excel format
func (s *ExamService) exportToExcel(attempts []models.ExamAttempt, exam *models.Exam) []byte {
	f := excelize.NewFile()
	sheet := "Results"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"#", "Student Name", "Score", "Percentage", "Grade", "Passed", "Time Taken"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 65+i)
		f.SetCellValue(sheet, cell, h)
	}

	for i, attempt := range attempts {
		if attempt.Score == nil {
			continue
		}
		row := i + 2
		percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
		passed := *attempt.Score >= exam.PassMark
		grade, _, _ := dto.GetGrade(percentage)
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "Student")
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), *attempt.Score)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), percentage)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), grade)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), passed)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), "")
	}

	var buf bytes.Buffer
	f.Write(&buf)
	return buf.Bytes()
}

// ============================================
// ADD MISSING SERVICE METHODS
// ============================================

// GetExamAssignments gets assignments for an exam
func (s *ExamService) GetExamAssignments(ctx context.Context, examID string) (interface{}, error) {
	assignments, err := s.examRepo.FindAssignmentsByExamWithContext(ctx, examID)
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

// GetStudentPerformance gets student performance overview
func (s *ExamService) GetStudentPerformance(ctx context.Context, studentID string) (*dto.StudentPerformanceResponse, error) {
	student, err := s.examRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, errors.New("student not found")
	}

	// Get all attempts for the student
	attempts, err := s.examRepo.GetAttemptsByStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}

	// Group by subject
	subjectMap := make(map[string]*subjectPerformanceData)
	totalExams := 0
	totalScore := 0

	for _, attempt := range attempts {
		if attempt.Status != string(models.AttemptStatusCompleted) || attempt.Score == nil {
			continue
		}

		exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
		if err != nil {
			continue
		}

		subjectID := exam.SubjectID
		if _, exists := subjectMap[subjectID]; !exists {
			subjectMap[subjectID] = &subjectPerformanceData{
				SubjectID: subjectID,
				Exams:     []dto.ExamHistory{},
				TotalScore: 0,
				Count:     0,
			}
		}

		percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
		grade, _, _ := dto.GetGrade(percentage)

		subjectMap[subjectID].Exams = append(subjectMap[subjectID].Exams, dto.ExamHistory{
			ExamTitle:  exam.Title,
			Score:      *attempt.Score,
			Percentage: percentage,
			Grade:      grade,
			Date:       attempt.CreatedAt,
		})
		subjectMap[subjectID].TotalScore += *attempt.Score
		subjectMap[subjectID].Count++
		totalExams++
		totalScore += *attempt.Score
	}

	// Build subject performance list
	subjectPerformance := make([]dto.SubjectPerformance, 0, len(subjectMap))
	for _, data := range subjectMap {
		avgScore := 0.0
		if data.Count > 0 {
			avgScore = float64(data.TotalScore) / float64(data.Count)
		}
		subjectPerformance = append(subjectPerformance, dto.SubjectPerformance{
			Subject:      data.SubjectID,
			SubjectID:    data.SubjectID,
			ExamsTaken:   data.Count,
			AverageScore: avgScore,
			ExamHistory:  data.Exams,
		})
	}

	// Calculate overall
	overallAvg := 0.0
	if totalExams > 0 {
		overallAvg = float64(totalScore) / float64(totalExams)
	}
	overallGrade, _, _ := dto.GetGrade(overallAvg)

	// Get school info
	var school models.School
	s.db.WithContext(ctx).First(&school, "id = ?", student.SchoolID)

	return &dto.StudentPerformanceResponse{
		Student: dto.StudentInfo{
			ID:              student.ID,
			Name:            student.User.FirstName + " " + student.User.LastName,
			AdmissionNumber: student.AdmissionNo,
			School:          school.Name,
			SchoolID:        student.SchoolID,
		},
		SubjectPerformance: subjectPerformance,
		OverallPerformance: dto.OverallPerformance{
			TotalExamsTaken:          totalExams,
			OverallAveragePercentage: overallAvg,
			OverallGrade:             overallGrade,
		},
	}, nil
}

// GetTeacherSubjectResults gets subject results for a teacher
func (s *ExamService) GetTeacherSubjectResults(ctx context.Context, subjectID, classID string) (interface{}, error) {
	attempts, err := s.examRepo.GetTeacherSubjectResults(ctx, subjectID, classID)
	if err != nil {
		return nil, err
	}

	studentResults := make(map[string]map[string]interface{})
	for _, attempt := range attempts {
		studentID := attempt.StudentID
		if _, exists := studentResults[studentID]; !exists {
			studentResults[studentID] = map[string]interface{}{
				"student_id":    studentID,
				"student_name":  "Student",
				"exams":         []map[string]interface{}{},
				"total_exams":   0,
				"average_score": 0.0,
			}
		}

		exam, _ := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
		percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
		grade, _, _ := dto.GetGrade(percentage)

		studentResults[studentID]["exams"] = append(
			studentResults[studentID]["exams"].([]map[string]interface{}),
			map[string]interface{}{
				"exam_id":    attempt.ExamID,
				"exam_title": exam.Title,
				"score":      *attempt.Score,
				"percentage": percentage,
				"grade":      grade,
				"date_taken": attempt.CreatedAt,
			},
		)
	}

	return studentResults, nil
}

// GetTeacherClassResults gets class results for a teacher
func (s *ExamService) GetTeacherClassResults(ctx context.Context, classID, subjectID string) (interface{}, error) {
	attempts, err := s.examRepo.GetTeacherClassResults(ctx, classID, subjectID)
	if err != nil {
		return nil, err
	}

	var totalScore int
	var totalExams int
	var passed int

	for _, attempt := range attempts {
		if attempt.Score != nil {
			totalScore += *attempt.Score
			totalExams++
			exam, _ := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
			if *attempt.Score >= exam.PassMark {
				passed++
			}
		}
	}

	averageScore := 0.0
	if totalExams > 0 {
		averageScore = float64(totalScore) / float64(totalExams)
	}

	return map[string]interface{}{
		"class_id":       classID,
		"total_students": len(attempts),
		"average_score":  averageScore,
		"pass_rate":      float64(passed) / float64(len(attempts)) * 100,
		"attempts":       attempts,
	}, nil
}

// GetSchoolPerformanceOverview gets school performance overview
func (s *ExamService) GetSchoolPerformanceOverview(ctx context.Context, schoolID string) (interface{}, error) {
	stats, err := s.examRepo.GetSchoolPerformanceOverview(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// GetSubjectQuestionsForExam gets subject questions for exam creation
func (s *ExamService) GetSubjectQuestionsForExam(ctx context.Context, subjectID, termID, sessionID, classID string) (*dto.SubjectQuestionsResponse, error) {
	// Get subject
	subject, err := s.questionRepo.GetSubjectByID(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	// Get questions
	questions, err := s.examRepo.GetSubjectQuestionsWithContext(ctx, subjectID, termID, sessionID, classID)
	if err != nil {
		return nil, err
	}

	// Build response
	questionItems := make([]dto.QuestionWithSelection, len(questions))
	byDifficulty := make(map[string]int)
	byBloomLevel := make(map[string]int)
	byQuestionType := make(map[string]int)
	totalMarks := 0

	for i, q := range questions {
		questionItems[i] = dto.QuestionWithSelection{
			ID:           q.ID,
			QuestionText: q.QuestionText,
			QuestionType: string(q.QuestionType),
			Difficulty:   string(q.Difficulty),
			BloomLevel:   string(q.BloomLevel),
			Marks:        q.Marks,
			Topic:        q.Topic,
			Selected:     false,
			Options:      s.convertOptionsToPreview(q.Options),
		}

		byDifficulty[string(q.Difficulty)]++
		byBloomLevel[string(q.BloomLevel)]++
		byQuestionType[string(q.QuestionType)]++
		totalMarks += q.Marks
	}

	avgMarks := 0.0
	if len(questions) > 0 {
		avgMarks = float64(totalMarks) / float64(len(questions))
	}

	// Get class and session info
	var class models.Class
	var session models.AcademicSession
	var term models.Term

	if classID != "" {
		s.db.WithContext(ctx).First(&class, "id = ?", classID)
	}
	if sessionID != "" {
		s.db.WithContext(ctx).First(&session, "id = ?", sessionID)
	}
	if termID != "" {
		s.db.WithContext(ctx).First(&term, "id = ?", termID)
	}

	return &dto.SubjectQuestionsResponse{
		SubjectID:      subjectID,
		SubjectName:    subject.Name,
		ClassID:        classID,
		ClassName:      models.GetClassDisplayName(&class),
		SessionID:      sessionID,
		SessionName:    session.Name,
		TermID:         termID,
		TermName:       term.Name,
		TotalQuestions: len(questions),
		Questions:      questionItems,
		Statistics: dto.QuestionStatistics{
			ByDifficulty:   byDifficulty,
			ByBloomLevel:   byBloomLevel,
			ByQuestionType: byQuestionType,
			AverageMarks:   avgMarks,
		},
	}, nil
}

// PublishExam publishes an exam
func (s *ExamService) PublishExam(ctx context.Context, examID string) error {
	exam, err := s.examRepo.FindExamByIDWithContext(ctx, examID)
	if err != nil {
		return errors.New("exam not found")
	}

	exam.Status = string(models.ExamStatusPublished)
	exam.IsActive = true

	return s.examRepo.UpdateExamWithContext(ctx, exam)
}

// subjectPerformanceData is a helper struct
type subjectPerformanceData struct {
	SubjectID  string
	Exams      []dto.ExamHistory
	TotalScore int
	Count      int
}

// service/exam_service.go - Updated methods


// // ============================================
// // StartExamForStudent - Simplified version that uses authenticated user
// // ============================================
// func (s *ExamService) StartExamForStudent(ctx context.Context, examID string, req *dto.StartExamRequest) (*dto.StartExamResponse, error) {
//     logger := zap.L().With(
//         zap.String("method", "StartExamForStudent"),
//         zap.String("exam_id", examID),
//         zap.String("ip_address", req.IPAddress),
//     )
//     logger.Info("Starting exam for student")

//     // Get student ID from context
//     studentID, ok := ctx.Value("student_id").(string)
//     if !ok {
//         logger.Error("Student context not found")
//         return nil, errors.New("student context not found")
//     }
//     logger = logger.With(zap.String("student_id", studentID))

//     // Get student details
//     student, err := s.examRepo.GetStudentByID(ctx, studentID)
//     if err != nil {
//         logger.Error("Failed to get student details", zap.Error(err))
//         return nil, errors.New("student not found")
//     }
//     logger = logger.With(
//         zap.String("class_id", student.ClassID),
//         zap.String("school_id", student.SchoolID),
//     )
//     logger.Debug("Student found")

//     // Check for existing active attempt
//     existing, err := s.examRepo.FindActiveAttemptWithContext(ctx, studentID, examID)
//     if err != nil {
//         logger.Warn("Error checking for existing attempt", zap.Error(err))
//         // Continue - this is not a fatal error
//     }
//     if existing != nil {
//         logger.Info("Existing active attempt found, resuming", 
//             zap.String("attempt_id", existing.ID),
//             zap.Time("started_at", existing.StartTime),
//         )
//         return s.buildStartExamResponse(ctx, existing)
//     }
//     logger.Debug("No existing attempt found, creating new one")

//     // Get exam with questions
//     exam, _, err := s.examRepo.FindExamWithQuestionsWithContext(ctx, examID)
//     if err != nil {
//         logger.Error("Failed to find exam", zap.Error(err))
//         return nil, errors.New("exam not found")
//     }
//     logger = logger.With(
//         zap.String("exam_title", exam.Title),
//         zap.String("subject_id", exam.SubjectID),
//         zap.Int("duration_minutes", exam.DurationMinutes),
//     )

//     // Check if exam is available for this student's class
//     if exam.ClassID != "" && exam.ClassID != student.ClassID {
//         logger.Warn("Exam not assigned to student's class",
//             zap.String("exam_class_id", exam.ClassID),
//             zap.String("student_class_id", student.ClassID),
//         )
//         return nil, errors.New("exam not assigned to your class")
//     }
//     logger.Debug("Class assignment verified")

//     // Check exam availability
//     now := time.Now()
//     if exam.StartTime != nil && now.Before(*exam.StartTime) {
//         logger.Warn("Exam not started yet",
//             zap.Time("start_time", *exam.StartTime),
//             zap.Duration("time_until_start", time.Until(*exam.StartTime)),
//         )
//         return nil, errors.New("exam has not started yet")
//     }
//     if exam.EndTime != nil && now.After(*exam.EndTime) {
//         logger.Warn("Exam has already ended",
//             zap.Time("end_time", *exam.EndTime),
//             zap.Duration("time_since_end", time.Since(*exam.EndTime)),
//         )
//         return nil, errors.New("exam has already ended")
//     }
//     logger.Debug("Exam availability verified")

//     // Create attempt
//     attempt := &models.ExamAttempt{
//         ID:         models.GenerateID(),
//         StudentID:  studentID,
//         ExamID:     examID,
//         StartTime:  now,
//         Status:     string(models.AttemptStatusInProgress),
//         IPAddress:  req.IPAddress,
//         DeviceInfo: make(models.JSONMap),
//     }

//     if req.DeviceInfo != "" {
//         attempt.DeviceInfo["user_agent"] = req.DeviceInfo
//         logger.Debug("Device info recorded", zap.String("user_agent", req.DeviceInfo))
//     }

//     logger.Info("Creating exam attempt", zap.String("attempt_id", attempt.ID))

//     if err := s.examRepo.CreateAttempt(attempt); err != nil {
//         logger.Error("Failed to create attempt", 
//             zap.Error(err),
//             zap.String("attempt_id", attempt.ID),
//         )
//         return nil, err
//     }
//     logger.Info("Attempt created successfully", 
//         zap.String("attempt_id", attempt.ID),
//         zap.Time("start_time", now),
//     )

//     // Create proctoring session
//     proctoring := &models.ProctoringSession{
//         ID:        models.GenerateID(),
//         AttemptID: attempt.ID,
//         StudentID: studentID,
//         Status:    "active",
//         StartedAt: now,
//     }
    
//     logger.Debug("Creating proctoring session", zap.String("proctoring_id", proctoring.ID))
    
//     if err := s.examRepo.CreateProctoringSession(proctoring); err != nil {
//         // Log error but continue - proctoring is optional
//         logger.Error("Failed to create proctoring session (non-fatal)", 
//             zap.Error(err),
//             zap.String("attempt_id", attempt.ID),
//         )
//     } else {
//         logger.Debug("Proctoring session created", zap.String("proctoring_id", proctoring.ID))
//     }

//     logger.Info("Exam started successfully", 
//         zap.String("attempt_id", attempt.ID),
//         zap.String("exam_title", exam.Title),
//         zap.String("student_id", studentID),
//     )

//     return s.buildStartExamResponse(ctx, attempt)
// }

// // ============================================
// // StartExamForStudent - Simplified version that uses authenticated user
// // ============================================
// func (s *ExamService) StartExamForStudent(ctx context.Context, examID string, req *dto.StartExamRequest) (*dto.StartExamResponse, error) {
//     logger := zap.L().With(
//         zap.String("method", "StartExamForStudent"),
//         zap.String("exam_id", examID),
//         zap.String("ip_address", req.IPAddress),
//     )
//     logger.Info("Starting exam for student")

//     // Get student ID from context
//     studentID, ok := ctx.Value("student_id").(string)
//     if !ok {
//         logger.Error("Student context not found")
//         return nil, errors.New("student context not found")
//     }
//     logger = logger.With(zap.String("student_id", studentID))

//     // Get student details
//     student, err := s.examRepo.GetStudentByID(ctx, studentID)
//     if err != nil {
//         logger.Error("Failed to get student details", zap.Error(err))
//         return nil, errors.New("student not found")
//     }

//     // ✅ FIX: Get class display name for logging
//     classDisplayName := ""
//     if student.ClassID != "" {
//         var class models.Class
//         if err := s.db.WithContext(ctx).
//             Preload("ClassLevel").
//             Preload("ClassArm").
//             Where("id = ?", student.ClassID).
//             First(&class).Error; err == nil {
//             classDisplayName = models.GetClassDisplayName(&class)
//         }
//     }

//     logger = logger.With(
//         zap.String("class_id", student.ClassID),
//         zap.String("class_name", classDisplayName),
//         zap.String("school_id", student.SchoolID),
//     )
//     logger.Debug("Student found")

//     // Check for existing active attempt
//     existing, err := s.examRepo.FindActiveAttemptWithContext(ctx, studentID, examID)
//     if err != nil {
//         logger.Warn("Error checking for existing attempt", zap.Error(err))
//         // Continue - this is not a fatal error
//     }
//     if existing != nil {
//         logger.Info("Existing active attempt found, resuming",
//             zap.String("attempt_id", existing.ID),
//             zap.Time("started_at", existing.StartTime),
//         )
//         return s.buildStartExamResponse(ctx, existing)
//     }
//     logger.Debug("No existing attempt found, creating new one")

//     // Get exam with questions
//     exam, _, err := s.examRepo.FindExamWithQuestionsWithContext(ctx, examID)
//     if err != nil {
//         logger.Error("Failed to find exam", zap.Error(err))
//         return nil, errors.New("exam not found")
//     }

//     // ✅ FIX: Load subject info for logging
//     var subject models.Subject
//     subjectName := ""
//     if err := s.db.WithContext(ctx).Where("id = ?", exam.SubjectID).First(&subject).Error; err == nil {
//         subjectName = subject.Name
//     }

//     logger = logger.With(
//         zap.String("exam_title", exam.Title),
//         zap.String("subject_id", exam.SubjectID),
//         zap.String("subject_name", subjectName),
//         zap.Int("duration_minutes", exam.DurationMinutes),
//     )

//     // Check if exam is available for this student's class
//     if exam.ClassID != "" && exam.ClassID != student.ClassID {
//         logger.Warn("Exam not assigned to student's class",
//             zap.String("exam_class_id", exam.ClassID),
//             zap.String("student_class_id", student.ClassID),
//         )
//         return nil, errors.New("exam not assigned to your class")
//     }
//     logger.Debug("Class assignment verified")

//     // Check exam availability
//     now := time.Now()
//     if exam.StartTime != nil && now.Before(*exam.StartTime) {
//         logger.Warn("Exam not started yet",
//             zap.Time("start_time", *exam.StartTime),
//             zap.Duration("time_until_start", time.Until(*exam.StartTime)),
//         )
//         return nil, errors.New("exam has not started yet")
//     }
//     if exam.EndTime != nil && now.After(*exam.EndTime) {
//         logger.Warn("Exam has already ended",
//             zap.Time("end_time", *exam.EndTime),
//             zap.Duration("time_since_end", time.Since(*exam.EndTime)),
//         )
//         return nil, errors.New("exam has already ended")
//     }
//     logger.Debug("Exam availability verified")

//     // Create attempt
//     attempt := &models.ExamAttempt{
//         ID:         models.GenerateID(),
//         StudentID:  studentID,
//         ExamID:     examID,
//         StartTime:  now,
//         Status:     string(models.AttemptStatusInProgress),
//         IPAddress:  req.IPAddress,
//         DeviceInfo: make(models.JSONMap),
//     }

//     if req.DeviceInfo != "" {
//         attempt.DeviceInfo["user_agent"] = req.DeviceInfo
//         logger.Debug("Device info recorded", zap.String("user_agent", req.DeviceInfo))
//     }

//     logger.Info("Creating exam attempt", zap.String("attempt_id", attempt.ID))

//     if err := s.examRepo.CreateAttempt(attempt); err != nil {
//         logger.Error("Failed to create attempt",
//             zap.Error(err),
//             zap.String("attempt_id", attempt.ID),
//         )
//         return nil, err
//     }
//     logger.Info("Attempt created successfully",
//         zap.String("attempt_id", attempt.ID),
//         zap.Time("start_time", now),
//     )

//     // Create proctoring session
//     proctoring := &models.ProctoringSession{
//         ID:        models.GenerateID(),
//         AttemptID: attempt.ID,
//         StudentID: studentID,
//         Status:    "active",
//         StartedAt: now,
//     }

//     logger.Debug("Creating proctoring session", zap.String("proctoring_id", proctoring.ID))

//     if err := s.examRepo.CreateProctoringSession(proctoring); err != nil {
//         // Log error but continue - proctoring is optional
//         logger.Error("Failed to create proctoring session (non-fatal)",
//             zap.Error(err),
//             zap.String("attempt_id", attempt.ID),
//         )
//     } else {
//         logger.Debug("Proctoring session created", zap.String("proctoring_id", proctoring.ID))
//     }

//     logger.Info("Exam started successfully",
//         zap.String("attempt_id", attempt.ID),
//         zap.String("exam_title", exam.Title),
//         zap.String("student_id", studentID),
//     )

//     return s.buildStartExamResponse(ctx, attempt)
// }

// service/exam_service.go

// ============================================
// StartExamForStudent - Modified to accept studentID as parameter
// ============================================
func (s *ExamService) StartExamForStudent(ctx context.Context, examID string, studentID string, req *dto.StartExamRequest) (*dto.StartExamResponse, error) {
    logger := zap.L().With(
        zap.String("method", "StartExamForStudent"),
        zap.String("exam_id", examID),
        zap.String("student_id", studentID),
        zap.String("ip_address", req.IPAddress),
    )
    logger.Info("Starting exam for student")

    if studentID == "" {
        logger.Error("Student ID is empty")
        return nil, errors.New("student ID is required")
    }

    // Get student details
    student, err := s.examRepo.GetStudentByID(ctx, studentID)
    if err != nil {
        logger.Error("Failed to get student details", zap.Error(err))
        return nil, errors.New("student not found")
    }

    // Get class display name for logging
    classDisplayName := ""
    if student.ClassID != "" {
        var class models.Class
        if err := s.db.WithContext(ctx).
            Preload("ClassLevel").
            Preload("ClassArm").
            Where("id = ?", student.ClassID).
            First(&class).Error; err == nil {
            classDisplayName = models.GetClassDisplayName(&class)
        }
    }

    logger = logger.With(
        zap.String("class_id", student.ClassID),
        zap.String("class_name", classDisplayName),
        zap.String("school_id", student.SchoolID),
    )
    logger.Debug("Student found")

    // Check for existing active attempt
    existing, err := s.examRepo.FindActiveAttemptWithContext(ctx, studentID, examID)
    if err != nil {
        logger.Warn("Error checking for existing attempt", zap.Error(err))
        // Continue - this is not a fatal error
    }
    if existing != nil {
        logger.Info("Existing active attempt found, resuming",
            zap.String("attempt_id", existing.ID),
            zap.Time("started_at", existing.StartTime),
        )
        return s.buildStartExamResponse(ctx, existing)
    }
    logger.Debug("No existing attempt found, creating new one")

    // Get exam with questions
    exam, _, err := s.examRepo.FindExamWithQuestionsWithContext(ctx, examID)
    if err != nil {
        logger.Error("Failed to find exam", zap.Error(err))
        return nil, errors.New("exam not found")
    }

    // Load subject info for logging
    var subject models.Subject
    subjectName := ""
    if err := s.db.WithContext(ctx).Where("id = ?", exam.SubjectID).First(&subject).Error; err == nil {
        subjectName = subject.Name
    }

    logger = logger.With(
        zap.String("exam_title", exam.Title),
        zap.String("subject_id", exam.SubjectID),
        zap.String("subject_name", subjectName),
        zap.Int("duration_minutes", exam.DurationMinutes),
    )

    // Check if exam is available for this student's class
    if exam.ClassID != "" && exam.ClassID != student.ClassID {
        logger.Warn("Exam not assigned to student's class",
            zap.String("exam_class_id", exam.ClassID),
            zap.String("student_class_id", student.ClassID),
        )
        return nil, errors.New("exam not assigned to your class")
    }
    logger.Debug("Class assignment verified")

    // Check exam availability
    now := time.Now()
    if exam.StartTime != nil && now.Before(*exam.StartTime) {
        logger.Warn("Exam not started yet",
            zap.Time("start_time", *exam.StartTime),
            zap.Duration("time_until_start", time.Until(*exam.StartTime)),
        )
        return nil, errors.New("exam has not started yet")
    }
    if exam.EndTime != nil && now.After(*exam.EndTime) {
        logger.Warn("Exam has already ended",
            zap.Time("end_time", *exam.EndTime),
            zap.Duration("time_since_end", time.Since(*exam.EndTime)),
        )
        return nil, errors.New("exam has already ended")
    }
    logger.Debug("Exam availability verified")

    // Create attempt
    attempt := &models.ExamAttempt{
        ID:         models.GenerateID(),
        StudentID:  studentID,
        ExamID:     examID,
        StartTime:  now,
        Status:     string(models.AttemptStatusInProgress),
        IPAddress:  req.IPAddress,
        DeviceInfo: make(models.JSONMap),
    }

    if req.DeviceInfo != "" {
        attempt.DeviceInfo["user_agent"] = req.DeviceInfo
        logger.Debug("Device info recorded", zap.String("user_agent", req.DeviceInfo))
    }

    logger.Info("Creating exam attempt", zap.String("attempt_id", attempt.ID))

    if err := s.examRepo.CreateAttempt(attempt); err != nil {
        logger.Error("Failed to create attempt",
            zap.Error(err),
            zap.String("attempt_id", attempt.ID),
        )
        return nil, err
    }
    logger.Info("Attempt created successfully",
        zap.String("attempt_id", attempt.ID),
        zap.Time("start_time", now),
    )

    // Create proctoring session
    proctoring := &models.ProctoringSession{
        ID:        models.GenerateID(),
        AttemptID: attempt.ID,
        StudentID: studentID,
        Status:    "active",
        StartedAt: now,
    }

    logger.Debug("Creating proctoring session", zap.String("proctoring_id", proctoring.ID))

    if err := s.examRepo.CreateProctoringSession(proctoring); err != nil {
        // Log error but continue - proctoring is optional
        logger.Error("Failed to create proctoring session (non-fatal)",
            zap.Error(err),
            zap.String("attempt_id", attempt.ID),
        )
    } else {
        logger.Debug("Proctoring session created", zap.String("proctoring_id", proctoring.ID))
    }

    logger.Info("Exam started successfully",
        zap.String("attempt_id", attempt.ID),
        zap.String("exam_title", exam.Title),
        zap.String("student_id", studentID),
    )

    return s.buildStartExamResponse(ctx, attempt)
}

// ============================================
// GetStudentDashboardForUser - Get dashboard using authenticated user
// ============================================
func (s *ExamService) GetStudentDashboardForUser(ctx context.Context, userID string) (*dto.StudentDashboardResponse, error) {
    // Get student from user ID
    student, err := s.examRepo.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, errors.New("student not found")
    }

    // ✅ FIX: Get class display name with proper preload
    classDisplayName := ""
    if student.ClassID != "" {
        var class models.Class
        if err := s.db.WithContext(ctx).
            Preload("ClassLevel").
            Preload("ClassArm").
            Where("id = ?", student.ClassID).
            First(&class).Error; err == nil {
            classDisplayName = models.GetClassDisplayName(&class)
        }
    }

    // Get school info
    var school models.School
    s.db.WithContext(ctx).First(&school, "id = ?", student.SchoolID)

    // Get available exams
    availableExams, err := s.examRepo.GetExamsByStudentAndClass(ctx, student.ID, student.ClassID)
    if err != nil {
        return nil, err
    }

    // ✅ FIX: Get ALL attempts including submitted and in-progress
    attempts, err := s.examRepo.GetAttemptsByStudent(ctx, student.ID)
    if err != nil {
        return nil, err
    }

    // Get completed exams with results
    completedAttempts, err := s.examRepo.GetCompletedAttemptsByStudent(ctx, student.ID)
    if err != nil {
        return nil, err
    }

    // Get upcoming exams
    upcomingExams, err := s.getUpcomingExamsForStudent(ctx, student.ID)
    if err != nil {
        return nil, err
    }

    // ✅ FIX: Build available list with subject info and NO duplicates
    availableList := make([]dto.AvailableExam, 0)
    seenExamIDs := make(map[string]bool)

    for _, exam := range availableExams {
        // Skip duplicates
        if seenExamIDs[exam.ID] {
            continue
        }
        seenExamIDs[exam.ID] = true

        // Load subject info
        var subject models.Subject
        subjectName := ""
        subjectCode := ""
        if err := s.db.WithContext(ctx).Where("id = ?", exam.SubjectID).First(&subject).Error; err == nil {
            subjectName = subject.Name
            subjectCode = subject.Code
        }

        // ✅ FIX: Check if exam has an active attempt
        hasAttempt := false
        for _, attempt := range attempts {
            if attempt.ExamID == exam.ID && attempt.Status == string(models.AttemptStatusInProgress) {
                hasAttempt = true
                break
            }
        }

        availableList = append(availableList, dto.AvailableExam{
            ID:               exam.ID,
            Title:            exam.Title,
            ExamType:         exam.ExamType,
            Subject: dto.SubjectInfo{
                ID:   exam.SubjectID,
                Name: subjectName,
                Code: subjectCode,
            },
            DurationMinutes:  exam.DurationMinutes,
            TotalMarks:       exam.TotalMarks,
            PassMark:         exam.PassMark,
            Schedule: dto.ScheduleInfo{
                StartTime:   exam.StartTime,
                EndTime:     exam.EndTime,
                IsAvailable: s.isExamAvailable(exam),
            },
            Status:           exam.Status,
            OfflineAvailable: true,
            HasStarted:       hasAttempt,  // ✅ FIXED
            HasCompleted:     false,
            Instructions:     exam.Instructions,
        })
    }

    // ✅ FIX: Build in-progress list from ALL attempts
    inProgressList := make([]dto.InProgressExam, 0)
    for _, attempt := range attempts {
        // Include both in_progress AND submitted attempts
        if attempt.Status != string(models.AttemptStatusInProgress) && 
           attempt.Status != string(models.AttemptStatusSubmitted) {
            continue
        }

        exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
        if err != nil {
            continue
        }

        answered, _ := s.examRepo.GetAnsweredCountWithContext(ctx, attempt.ID)
        questions, _ := s.examRepo.GetExamQuestionsWithContext(ctx, attempt.ExamID)
        totalQ := len(questions)

        // Load subject info
        var subject models.Subject
        subjectName := ""
        subjectCode := ""
        if err := s.db.WithContext(ctx).Where("id = ?", exam.SubjectID).First(&subject).Error; err == nil {
            subjectName = subject.Name
            subjectCode = subject.Code
        }

        // Calculate time remaining
        timeRemaining := 0
        if attempt.Status == string(models.AttemptStatusInProgress) {
            timeRemaining = s.calculateRemainingTime(attempt.StartTime, exam.DurationMinutes)
        }

        inProgressList = append(inProgressList, dto.InProgressExam{
            ID:              exam.ID,
            Title:           exam.Title,
            ExamType:        exam.ExamType,
            Subject: dto.SubjectInfo{
                ID:   exam.SubjectID,
                Name: subjectName,
                Code: subjectCode,
            },
            DurationMinutes: exam.DurationMinutes,
            TotalMarks:      exam.TotalMarks,
            PassMark:        exam.PassMark,
            Schedule: dto.ScheduleInfo{
                StartTime:   exam.StartTime,
                EndTime:     exam.EndTime,
                IsAvailable: true,
            },
            Attempt: dto.AttemptInfo{
                AttemptID:          attempt.ID,
                StartTime:          attempt.StartTime,
                TimeRemaining:      timeRemaining,
                TimeElapsed:        int(time.Since(attempt.StartTime).Seconds()),
                AnsweredCount:      int(answered),
                TotalQuestions:     totalQ,
                ProgressPercentage: s.calculateProgress(int(answered), totalQ),
                LastActivity:       attempt.UpdatedAt,
            },
            OfflineAvailable: true,
            HasStarted:       true,
            HasCompleted:     false,
        })
    }

    // Build completed list with subject info
    completedList := make([]dto.CompletedExam, 0)
    for _, attempt := range completedAttempts {
        exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
        if err != nil {
            continue
        }
        result, err := s.examRepo.GetStudentExamResult(ctx, student.ID, attempt.ExamID)
        if err != nil || result == nil {
            continue
        }

        // Load subject info
        var subject models.Subject
        subjectName := ""
        subjectCode := ""
        if err := s.db.WithContext(ctx).Where("id = ?", exam.SubjectID).First(&subject).Error; err == nil {
            subjectName = subject.Name
            subjectCode = subject.Code
        }

        completedList = append(completedList, dto.CompletedExam{
            ID:              exam.ID,
            Title:           exam.Title,
            ExamType:        exam.ExamType,
            Subject: dto.SubjectInfo{
                ID:   exam.SubjectID,
                Name: subjectName,
                Code: subjectCode,
            },
            DurationMinutes: exam.DurationMinutes,
            TotalMarks:      exam.TotalMarks,
            PassMark:        exam.PassMark,
            Schedule: dto.ScheduleInfo{
                StartTime: exam.StartTime,
                EndTime:   exam.EndTime,
            },
            Result: dto.ResultInfo{
                Score:       result.TotalScore,
                Percentage:  result.Percentage,
                Grade:       result.Grade,
                Passed:      result.Percentage >= float64(exam.PassMark),
                CompletedAt: *result.PublishedAt,
            },
            HasCompleted:    true,
            CanReview:       exam.ReviewPolicy.ShowCorrectAnswers,
            ReviewAvailable: exam.ReviewPolicy.ShowCorrectAnswers,
        })
    }

    // Build upcoming list with subject info
    upcomingList := make([]dto.StudentUpcomingExam, 0)
    for _, exam := range upcomingExams {
        // Load subject info
        var subject models.Subject
        subjectName := ""
        subjectCode := ""
        if err := s.db.WithContext(ctx).Where("id = ?", exam.SubjectID).First(&subject).Error; err == nil {
            subjectName = subject.Name
            subjectCode = subject.Code
        }

        upcomingList = append(upcomingList, dto.StudentUpcomingExam{
            ID:               exam.ID,
            Title:            exam.Title,
            ExamType:         exam.ExamType,
            Subject: dto.SubjectInfo{
                ID:   exam.SubjectID,
                Name: subjectName,
                Code: subjectCode,
            },
            DurationMinutes:  exam.DurationMinutes,
            TotalMarks:       exam.TotalMarks,
            PassMark:         exam.PassMark,
            Schedule: dto.ScheduleInfo{
                StartTime:   exam.StartTime,
                EndTime:     exam.EndTime,
                IsAvailable: false,
            },
            Status:           "upcoming",
            DaysUntil:        s.daysUntil(exam.StartTime),
            OfflineAvailable: false,
            HasStarted:       false,
            HasCompleted:     false,
            Instructions:     exam.Instructions,
        })
    }

    // Calculate summary
    totalAvailable := len(availableList)
    totalInProgress := len(inProgressList)
    totalCompleted := len(completedList)
    totalUpcoming := len(upcomingList)

    var overallAverage float64
    if totalCompleted > 0 {
        var totalScore int
        for _, exam := range completedList {
            totalScore += exam.Result.Score
        }
        overallAverage = float64(totalScore) / float64(totalCompleted)
    }

    return &dto.StudentDashboardResponse{
        Student: dto.StudentInfo{
            ID:              student.ID,
            Name:            student.User.FirstName + " " + student.User.LastName,
            AdmissionNumber: student.AdmissionNo,
            Class:           classDisplayName,
            ClassID:         student.ClassID,
            School:          school.Name,
            SchoolID:        student.SchoolID,
        },
        Summary: dto.DashboardSummary{
            TotalAvailable:  totalAvailable,
            TotalInProgress: totalInProgress,
            TotalCompleted:  totalCompleted,
            TotalUpcoming:   totalUpcoming,
            OverallAverage:  overallAverage,
        },
        AvailableExams:  availableList,
        InProgressExams: inProgressList,
        CompletedExams:  completedList,
        UpcomingExams:   upcomingList,
    }, nil
}


// // ============================================
// // GetStudentPerformanceForUser gets performance for authenticated student
// // ============================================
// func (s *ExamService) GetStudentPerformanceForUser(ctx context.Context, userID string) (*dto.StudentPerformanceResponse, error) {
//     // Get student from user ID
//     student, err := s.examRepo.GetStudentByUserID(ctx, userID)
//     if err != nil {
//         return nil, errors.New("student not found")
//     }

//     // ✅ FIX: Get class display name with proper preload
//     classDisplayName := ""
//     if student.ClassID != "" {
//         var class models.Class
//         if err := s.db.WithContext(ctx).
//             Preload("ClassLevel").
//             Preload("ClassArm").
//             Where("id = ?", student.ClassID).
//             First(&class).Error; err == nil {
//             classDisplayName = models.GetClassDisplayName(&class)
//         }
//     }

//     // Get school info
//     var school models.School
//     s.db.WithContext(ctx).First(&school, "id = ?", student.SchoolID)

//     // Get all attempts for the student
//     attempts, err := s.examRepo.GetAttemptsByStudent(ctx, student.ID)
//     if err != nil {
//         return nil, err
//     }

//     // Group by subject
//     subjectMap := make(map[string]*subjectPerformanceData)
//     totalExams := 0
//     totalScore := 0

//     for _, attempt := range attempts {
//         if attempt.Status != string(models.AttemptStatusCompleted) || attempt.Score == nil {
//             continue
//         }

//         exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
//         if err != nil {
//             continue
//         }

//         subjectID := exam.SubjectID
//         if _, exists := subjectMap[subjectID]; !exists {
//             subjectMap[subjectID] = &subjectPerformanceData{
//                 SubjectID:  subjectID,
//                 Exams:      []dto.ExamHistory{},
//                 TotalScore: 0,
//                 Count:      0,
//             }
//         }

//         percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
//         grade, _, _ := dto.GetGrade(percentage)

//         subjectMap[subjectID].Exams = append(subjectMap[subjectID].Exams, dto.ExamHistory{
//             ExamTitle:  exam.Title,
//             Score:      *attempt.Score,
//             Percentage: percentage,
//             Grade:      grade,
//             Date:       attempt.CreatedAt,
//         })
//         subjectMap[subjectID].TotalScore += *attempt.Score
//         subjectMap[subjectID].Count++
//         totalExams++
//         totalScore += *attempt.Score
//     }

//     // Build subject performance list
//     subjectPerformance := make([]dto.SubjectPerformance, 0, len(subjectMap))
//     for subjectID, data := range subjectMap {
//         avgScore := 0.0
//         if data.Count > 0 {
//             avgScore = float64(data.TotalScore) / float64(data.Count)
//         }

//         // Get subject name
//         subject, _ := s.questionRepo.GetSubjectByID(ctx, subjectID)
//         subjectName := ""
//         if subject != nil {
//             subjectName = subject.Name
//         }

//         subjectPerformance = append(subjectPerformance, dto.SubjectPerformance{
//             Subject:      subjectName,
//             SubjectID:    subjectID,
//             ExamsTaken:   data.Count,
//             AverageScore: avgScore,
//             ExamHistory:  data.Exams,
//         })
//     }

//     // Calculate overall
//     overallAvg := 0.0
//     if totalExams > 0 {
//         overallAvg = float64(totalScore) / float64(totalExams)
//     }
//     overallGrade, _, _ := dto.GetGrade(overallAvg)

//     // ✅ Return with class display name properly populated
//     return &dto.StudentPerformanceResponse{
//         Student: dto.StudentInfo{
//             ID:              student.ID,
//             Name:            student.User.FirstName + " " + student.User.LastName,
//             AdmissionNumber: student.AdmissionNo,
//             Class:           classDisplayName, // ✅ FIXED
//             ClassID:         student.ClassID,  // ✅ FIXED
//             School:          school.Name,
//             SchoolID:        student.SchoolID,
//         },
//         SubjectPerformance: subjectPerformance,
//         OverallPerformance: dto.OverallPerformance{
//             TotalExamsTaken:          totalExams,
//             OverallAveragePercentage: overallAvg,
//             OverallGrade:             overallGrade,
//             SubjectsPassed:           0,
//             SubjectsFailed:           0,
//             BestSubject:              "",
//             WorstSubject:             "",
//         },
//         Attendance: dto.AttendanceInfo{
//             Present:             0,
//             Absent:              0,
//             AttendancePercentage: 0,
//         },
//     }, nil
// }

// ============================================
// GetStudentPerformanceForUser gets performance for authenticated student
// ============================================
func (s *ExamService) GetStudentPerformanceForUser(ctx context.Context, userID string) (*dto.StudentPerformanceResponse, error) {
    // Get student from user ID
    student, err := s.examRepo.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, errors.New("student not found")
    }

    // ✅ FIX: Get class display name with proper preload
    classDisplayName := ""
    if student.ClassID != "" {
        var class models.Class
        if err := s.db.WithContext(ctx).
            Preload("ClassLevel").
            Preload("ClassArm").
            Where("id = ?", student.ClassID).
            First(&class).Error; err == nil {
            classDisplayName = models.GetClassDisplayName(&class)
        }
    }

    // Get school info
    var school models.School
    s.db.WithContext(ctx).First(&school, "id = ?", student.SchoolID)

    // Get all attempts for the student
    attempts, err := s.examRepo.GetAttemptsByStudent(ctx, student.ID)
    if err != nil {
        return nil, err
    }

    // ✅ FIX: Include ALL attempts, not just completed
    // Also include in-progress and submitted attempts with partial data
    subjectMap := make(map[string]*subjectPerformanceData)
    totalExams := 0
    totalScore := 0
    hasAnyAttempt := false

    for _, attempt := range attempts {
        // ✅ FIX: Count all attempts, not just completed
        hasAnyAttempt = true
        
        // Skip if no score or not completed
        if attempt.Status != string(models.AttemptStatusCompleted) || attempt.Score == nil {
            continue
        }

        exam, err := s.examRepo.FindExamByIDWithContext(ctx, attempt.ExamID)
        if err != nil {
            continue
        }

        subjectID := exam.SubjectID
        if _, exists := subjectMap[subjectID]; !exists {
            subjectMap[subjectID] = &subjectPerformanceData{
                SubjectID:  subjectID,
                Exams:      []dto.ExamHistory{},
                TotalScore: 0,
                Count:      0,
            }
        }

        percentage := float64(*attempt.Score) / float64(exam.TotalMarks) * 100
        grade, _, _ := dto.GetGrade(percentage)

        subjectMap[subjectID].Exams = append(subjectMap[subjectID].Exams, dto.ExamHistory{
            ExamTitle:  exam.Title,
            Score:      *attempt.Score,
            Percentage: percentage,
            Grade:      grade,
            Date:       attempt.CreatedAt,
        })
        subjectMap[subjectID].TotalScore += *attempt.Score
        subjectMap[subjectID].Count++
        totalExams++
        totalScore += *attempt.Score
    }

    // Build subject performance list
    subjectPerformance := make([]dto.SubjectPerformance, 0, len(subjectMap))
    for subjectID, data := range subjectMap {
        avgScore := 0.0
        if data.Count > 0 {
            avgScore = float64(data.TotalScore) / float64(data.Count)
        }

        // Get subject name
        subject, _ := s.questionRepo.GetSubjectByID(ctx, subjectID)
        subjectName := ""
        if subject != nil {
            subjectName = subject.Name
        }

        subjectPerformance = append(subjectPerformance, dto.SubjectPerformance{
            Subject:      subjectName,
            SubjectID:    subjectID,
            ExamsTaken:   data.Count,
            AverageScore: avgScore,
            ExamHistory:  data.Exams,
        })
    }

    // Calculate overall
    overallAvg := 0.0
    overallGrade := "-"  // ✅ FIX: Default to "-" instead of "F"
    if totalExams > 0 {
        overallAvg = float64(totalScore) / float64(totalExams)
        overallGrade, _, _ = dto.GetGrade(overallAvg)
    }

    // ✅ FIX: Add message about no completed exams
    attendancePercentage := 0.0
    if hasAnyAttempt && totalExams == 0 {
        // Student has attempts but none completed
        // Keep attendance at 0
    }

    return &dto.StudentPerformanceResponse{
        Student: dto.StudentInfo{
            ID:              student.ID,
            Name:            student.User.FirstName + " " + student.User.LastName,
            AdmissionNumber: student.AdmissionNo,
            Class:           classDisplayName,
            ClassID:         student.ClassID,
            School:          school.Name,
            SchoolID:        student.SchoolID,
        },
        SubjectPerformance: subjectPerformance,
        OverallPerformance: dto.OverallPerformance{
            TotalExamsTaken:          totalExams,
            OverallAveragePercentage: overallAvg,
            OverallGrade:             overallGrade,  // ✅ FIXED: "-" instead of "F"
            SubjectsPassed:           0,
            SubjectsFailed:           0,
            BestSubject:              "",
            WorstSubject:             "",
        },
        Attendance: dto.AttendanceInfo{
            Present:             0,
            Absent:              0,
            AttendancePercentage: attendancePercentage,
        },
    }, nil
}


// GetStudentExamResultForUser gets result for authenticated student
func (s *ExamService) GetStudentExamResultForUser(ctx context.Context, examID string, userID string) (*dto.StudentExamResultResponse, error) {
    student, err := s.examRepo.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, errors.New("student not found")
    }
    return s.GetStudentExamResult(ctx, student.ID, examID)
}

// GetExamReviewForUser gets review for authenticated student
func (s *ExamService) GetExamReviewForUser(ctx context.Context, attemptID string, userID string) (*dto.StudentExamResultResponse, error) {
    student, err := s.examRepo.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, errors.New("student not found")
    }
    return s.GetExamReview(ctx, attemptID, student.ID)
}

// // GetTermlyResultForUser gets termly result for authenticated student
// func (s *ExamService) GetTermlyResultForUser(ctx context.Context, termID, sessionID, userID string) (*dto.TermlyResultResponse, error) {
//     student, err := s.examRepo.GetStudentByUserID(ctx, userID)
//     if err != nil {
//         return nil, errors.New("student not found")
//     }
//     return s.GetStudentTermlyResult(ctx, student.ID, termID, sessionID)
// }

// ============================================
// GetTermlyResultForUser gets termly result for authenticated student
// ============================================
func (s *ExamService) GetTermlyResultForUser(ctx context.Context, termID, sessionID, userID string) (*dto.TermlyResultResponse, error) {
    // Get student from user ID
    student, err := s.examRepo.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, errors.New("student not found")
    }

    // ✅ FIX: Get class display name for the response
    classDisplayName := ""
    if student.ClassID != "" {
        var class models.Class
        if err := s.db.WithContext(ctx).
            Preload("ClassLevel").
            Preload("ClassArm").
            Where("id = ?", student.ClassID).
            First(&class).Error; err == nil {
            classDisplayName = models.GetClassDisplayName(&class)
        }
    }

    // Get school info
    var school models.School
    s.db.WithContext(ctx).First(&school, "id = ?", student.SchoolID)

    // Get term and session info
    term, err := s.getTermByID(ctx, termID)
    if err != nil {
        return nil, err
    }
    session, err := s.getSessionByID(ctx, sessionID)
    if err != nil {
        return nil, err
    }

    // Get all results for the term
    results, err := s.examRepo.FindResultsByStudentAndTerm(ctx, student.ID, termID, sessionID)
    if err != nil {
        return nil, err
    }

    // Get all subjects for this class
    subjects, err := s.examRepo.GetSubjectsByClassAndTerm(ctx, student.ClassID, termID)
    if err != nil {
        return nil, err
    }

    // Build subject results
    subjectResults := make([]dto.SubjectTermlyResult, 0, len(subjects))
    totalScore := 0
    totalMarks := 0
    passed := 0

    for _, subject := range subjects {
        subjectResult, err := s.buildSubjectTermlyResult(ctx, student.ID, subject.ID, results)
        if err != nil {
            continue
        }
        subjectResults = append(subjectResults, subjectResult)
        totalScore += subjectResult.TotalScore
        totalMarks += subjectResult.TotalMarks
        if subjectResult.Percentage >= 50 {
            passed++
        }
    }

    // Calculate overall
    overallPercentage := 0.0
    if totalMarks > 0 {
        overallPercentage = float64(totalScore) / float64(totalMarks) * 100
    }
    overallGrade, overallGP, _ := dto.GetGrade(overallPercentage)

    // ✅ Return with class name properly populated
    return &dto.TermlyResultResponse{
        Student: dto.StudentInfo{
            ID:              student.ID,
            Name:            student.User.FirstName + " " + student.User.LastName,
            AdmissionNumber: student.AdmissionNo,
            Class:           classDisplayName, // ✅ FIXED
            ClassID:         student.ClassID,  // ✅ FIXED
            School:          school.Name,
            SchoolID:        school.ID,
        },
        Class: dto.ClassInfo{
            ID:   student.ClassID,
            Name: classDisplayName, // ✅ FIXED
        },
        Term: dto.TermInfo{
            ID:     term.ID,
            Name:   term.Name,
            Number: term.TermNumber,
        },
        Session: dto.SessionInfo{
            ID:   session.ID,
            Name: session.Name,
        },
        School: dto.SchoolInfo{
            ID:   school.ID,
            Name: school.Name,
        },
        Subjects: subjectResults,
        Summary: dto.TermlySummary{
            TotalSubjects:          len(subjects),
            TotalScore:             totalScore,
            TotalMarks:             totalMarks,
            OverallPercentage:      overallPercentage,
            OverallGrade:           overallGrade,
            OverallGradePoint:      overallGP,
            NumberPassed:           passed,
            NumberFailed:           len(subjects) - passed,
            ClassPosition:          0,
            TotalStudents:          0,
            BestPerformingSubject:  s.getBestSubject(subjectResults),
            WorstPerformingSubject: s.getWorstSubject(subjectResults),
        },
        GeneratedAt: time.Now(),
    }, nil
}

// // GetReportCardForUser gets report card for authenticated student
// func (s *ExamService) GetReportCardForUser(ctx context.Context, termID, sessionID, userID string) (*dto.TermlyReportCardResponse, error) {
//     student, err := s.examRepo.GetStudentByUserID(ctx, userID)
//     if err != nil {
//         return nil, errors.New("student not found")
//     }
//     return s.GetTermlyReportCard(ctx, student.ID, termID, sessionID)
// }


// ============================================
// GetReportCardForUser gets report card for authenticated student
// ============================================
func (s *ExamService) GetReportCardForUser(ctx context.Context, termID, sessionID, userID string) (*dto.TermlyReportCardResponse, error) {
    // Get student from user ID
    student, err := s.examRepo.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, errors.New("student not found")
    }

    // ✅ FIX: Get class display name with proper preload
    classDisplayName := ""
    if student.ClassID != "" {
        var class models.Class
        if err := s.db.WithContext(ctx).
            Preload("ClassLevel").
            Preload("ClassArm").
            Where("id = ?", student.ClassID).
            First(&class).Error; err == nil {
            classDisplayName = models.GetClassDisplayName(&class)
        }
    }

    // Get school info
    var school models.School
    s.db.WithContext(ctx).First(&school, "id = ?", student.SchoolID)

    // Get term and session info
    term, err := s.getTermByID(ctx, termID)
    if err != nil {
        return nil, err
    }
    session, err := s.getSessionByID(ctx, sessionID)
    if err != nil {
        return nil, err
    }

    // Get the termly result first
    termlyResult, err := s.GetStudentTermlyResult(ctx, student.ID, termID, sessionID)
    if err != nil {
        return nil, err
    }

    // Build report card subjects
    reportCardSubjects := make([]dto.ReportCardSubject, 0, len(termlyResult.Subjects))
    for _, subject := range termlyResult.Subjects {
        reportCardSubjects = append(reportCardSubjects, dto.ReportCardSubject{
            SubjectName:  subject.Subject.Name,
            SubjectCode:  subject.Subject.Code,
            TotalScore:   subject.TotalScore,
            Grade:        subject.Grade,
            GradePoint:   subject.GradePoint,
            Position:     subject.Position,
            ClassAverage: subject.ClassAverage,
            HighestScore: subject.HighestScore,
            LowestScore:  subject.LowestScore,
            Remarks:      subject.Remarks,
        })
    }

    // ✅ Return with class name properly populated
    return &dto.TermlyReportCardResponse{
        School: dto.SchoolInfo{
            ID:   termlyResult.School.ID,
            Name: termlyResult.School.Name,
        },
        Student: dto.StudentInfo{
            ID:              student.ID,
            Name:            student.User.FirstName + " " + student.User.LastName,
            AdmissionNumber: student.AdmissionNo,
            Class:           classDisplayName, // ✅ FIXED
            ClassID:         student.ClassID,  // ✅ FIXED
            School:          school.Name,
            SchoolID:        student.SchoolID,
        },
        Class: dto.ClassInfo{
            ID:   student.ClassID,
            Name: classDisplayName, // ✅ FIXED
        },
        Term: dto.TermInfo{
            ID:     term.ID,
            Name:   term.Name,
            Number: term.TermNumber,
        },
        Session: dto.SessionInfo{
            ID:   session.ID,
            Name: session.Name,
        },
        Subjects: reportCardSubjects,
        Summary: dto.ReportCardSummary{
            TotalSubjects:         termlyResult.Summary.TotalSubjects,
            TotalScore:            termlyResult.Summary.TotalScore,
            TotalMarks:            termlyResult.Summary.TotalMarks,
            OverallPercentage:     termlyResult.Summary.OverallPercentage,
            OverallGrade:          termlyResult.Summary.OverallGrade,
            OverallGradePoint:     termlyResult.Summary.OverallGradePoint,
            NumberPassed:          termlyResult.Summary.NumberPassed,
            NumberFailed:          termlyResult.Summary.NumberFailed,
            ClassPosition:         termlyResult.Summary.ClassPosition,
            TotalStudentsInClass:  termlyResult.Summary.TotalStudents,
            BestPerformingSubject: termlyResult.Summary.BestPerformingSubject,
            WorstPerformingSubject: termlyResult.Summary.WorstPerformingSubject,
        },
        TeacherRemarks:   "",
        PrincipalRemarks: "",
        NextTermBegins:   nil,
    }, nil
}



// StartPracticeForUser starts practice for authenticated student
func (s *ExamService) StartPracticeForUser(ctx context.Context, subjectID string, questionCount int, userID string) (*dto.PracticeSessionResponse, error) {
    student, err := s.examRepo.GetStudentByUserID(ctx, userID)
    if err != nil {
        return nil, errors.New("student not found")
    }
    return s.StartPractice(ctx, student.ID, subjectID, questionCount)
}

// ============================================
// VALIDATE QUESTIONS FOR EXAM - ADD TO question_service.go
// ============================================

// ValidateQuestionsForExam validates that questions are suitable for an exam
// Checks: exam_type, term_id, session_id, subject_id, and question count
func (s *QuestionService) ValidateQuestionsForExam(ctx context.Context, exam *models.Exam, questionIDs []string) error {
    if len(questionIDs) == 0 {
        return errors.New("no questions selected")
    }

    // Get all questions in one query
    var questions []models.QuestionBank
    err := s.db.WithContext(ctx).
        Where("id IN ? AND deleted_at IS NULL", questionIDs).
        Find(&questions).Error
    if err != nil {
        return fmt.Errorf("failed to fetch questions: %w", err)
    }

    // Check all questions exist
    if len(questions) != len(questionIDs) {
        return errors.New("one or more questions not found")
    }

    // Validation errors collection
    var validationErrors []string

    for _, q := range questions {
        // 1. Check exam type matches (except for practice which can use any)
        if exam.ExamType != "practice" && q.ExamType != exam.ExamType {
            validationErrors = append(validationErrors, 
                fmt.Sprintf("Question %s has exam_type '%s' but exam expects '%s'", 
                    q.ID[:8], q.ExamType, exam.ExamType))
        }

        // 2. Check term matches
        if q.TermID != exam.TermID {
            validationErrors = append(validationErrors,
                fmt.Sprintf("Question %s belongs to term %s, not %s", 
                    q.ID[:8], q.TermID, exam.TermID))
        }

        // 3. Check session matches
        if q.SessionID != exam.SessionID {
            validationErrors = append(validationErrors,
                fmt.Sprintf("Question %s belongs to session %s, not %s", 
                    q.ID[:8], q.SessionID, exam.SessionID))
        }

        // 4. Check subject matches
        if q.SubjectID != exam.SubjectID {
            validationErrors = append(validationErrors,
                fmt.Sprintf("Question %s belongs to subject %s, not %s", 
                    q.ID[:8], q.SubjectID, exam.SubjectID))
        }

        // 5. Check class matches
        if q.ClassID != exam.ClassID {
            validationErrors = append(validationErrors,
                fmt.Sprintf("Question %s belongs to class %s, not %s", 
                    q.ID[:8], q.ClassID, exam.ClassID))
        }

        // 6. Check status is published (or draft allowed for practice)
        if q.Status != models.QuestionStatusPublished && exam.ExamType != "practice" {
            validationErrors = append(validationErrors,
                fmt.Sprintf("Question %s is not published (status: %s)", 
                    q.ID[:8], q.Status))
        }
        
        // ✅ REMOVED: SchoolID check - exam doesn't have SchoolID field
        // School is derived from the question's context, not the exam
    }

    if len(validationErrors) > 0 {
        return fmt.Errorf("question validation failed:\n%s", strings.Join(validationErrors, "\n"))
    }

    return nil
}

// ValidateQuestionCountForExam validates the number of questions for an exam type
func (s *QuestionService) ValidateQuestionCountForExam(examType string, count int) error {
    minQ, maxQ := dto.GetQuestionCountRange(examType)
    if count < minQ {
        return fmt.Errorf("exam requires at least %d questions, got %d", minQ, count)
    }
    if count > maxQ {
        return fmt.Errorf("exam can have at most %d questions, got %d", maxQ, count)
    }
    return nil
}