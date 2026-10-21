package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/grading"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// resolveLastTermOfSession finds the highest-TermNumber term in a
// session - "promotion examination results" are evaluated against the
// session's final term, the same real per-subject results the Main
// Termly report for that term already shows (GetStudentTermlyResult),
// not a separately invented computation.
func (s *ExamService) resolveLastTermOfSession(ctx context.Context, sessionID string) (*models.Term, error) {
	var term models.Term
	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("term_number DESC").
		First(&term).Error
	return &term, err
}

// GetPromotionReportCardForUser builds the authenticated student's
// Promotion Examination Report Card: the session's final-term results
// (the exact same shared grading engine as the Main Termly report),
// a promotion-policy evaluation that is clearly labeled a
// RECOMMENDATION ONLY, and the authorized final decision if one has
// been recorded (models.PromotionDecision) - "Pending authorized
// approval" otherwise. This function never copies the recommendation
// into FinalDecision; only RecordPromotionDecision, a separate,
// explicitly human-triggered write path, can set FinalDecision at all.
func (s *ExamService) GetPromotionReportCardForUser(ctx context.Context, sessionID, userID string) (*dto.PromotionReportCardResponse, error) {
	student, err := s.examRepo.GetStudentByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("student not found")
	}

	term, err := s.resolveLastTermOfSession(ctx, sessionID)
	if err != nil {
		return nil, errors.New("no term found for this session")
	}

	termlyResult, err := s.GetStudentTermlyResult(ctx, student.ID, term.ID, sessionID)
	if err != nil {
		return nil, err
	}

	classDisplayName, classTeacherName, classLevelID := s.resolveClassWithTeacher(ctx, student.ClassID)

	proposedNextClass := "Not configured"
	if classLevelID != "" {
		var currentLevel models.ClassLevel
		if err := s.db.WithContext(ctx).Where("id = ?", classLevelID).First(&currentLevel).Error; err == nil && currentLevel.PromotionTo != nil {
			var nextLevel models.ClassLevel
			if err := s.db.WithContext(ctx).Where("id = ?", *currentLevel.PromotionTo).First(&nextLevel).Error; err == nil {
				proposedNextClass = nextLevel.Name
			}
		}
	}

	reportSubjects := make([]dto.ReportCardSubject, 0, len(termlyResult.Subjects))
	standings := make([]grading.SubjectStanding, 0, len(termlyResult.Subjects))
	for _, subject := range termlyResult.Subjects {
		reportSubjects = append(reportSubjects, dto.ReportCardSubject{
			SubjectName: subject.Subject.Name, SubjectCode: subject.Subject.Code,
			TotalScore: subject.TotalScore, Percentage: subject.Percentage,
			Grade: subject.Grade, GradePoint: subject.GradePoint,
			Position: subject.Position, ClassAverage: subject.ClassAverage,
			HighestScore: subject.HighestScore, LowestScore: subject.LowestScore,
			Remarks: subject.Remarks, Components: subject.Components,
			IsComplete: subject.IsComplete, IsProvisional: subject.IsProvisional,
		})
		standings = append(standings, grading.SubjectStanding{
			SubjectID: subject.Subject.ID, Percentage: subject.Percentage, IsComplete: subject.IsComplete,
		})
	}

	policy := s.gradingSvc.ResolvePromotionPolicy(ctx, student.SchoolID, classLevelID)
	evaluation := grading.EvaluatePromotion(policy, termlyResult.Summary.OverallPercentage, standings)

	resultCompleteness := "Complete"
	if !evaluation.ResultsComplete {
		resultCompleteness = "Incomplete - one or more subjects pending"
	}

	policyChecks := "Overall threshold: " + formatPercent(policy.OverallThreshold)
	if len(policy.CoreSubjectIDs) > 0 {
		policyChecks += " - core-subject rules: school-configured"
	}

	finalDecision := ""
	if decision := s.getPromotionDecision(ctx, student.ID, sessionID); decision != nil {
		finalDecision = decision.Decision
		if decision.Notes != "" {
			finalDecision += " (" + decision.Notes + ")"
		}
	}

	return &dto.PromotionReportCardResponse{
		School:  termlyResult.School,
		Student: termlyResult.Student,
		Class:   dto.ClassInfo{ID: student.ClassID, Name: classDisplayName, ClassTeacher: classTeacherName},
		Session: termlyResult.Session,
		Subjects: reportSubjects,
		Summary: dto.ReportCardSummary{
			TotalSubjects: termlyResult.Summary.TotalSubjects, TotalScore: termlyResult.Summary.TotalScore,
			TotalMarks: termlyResult.Summary.TotalMarks, OverallPercentage: termlyResult.Summary.OverallPercentage,
			OverallGrade: termlyResult.Summary.OverallGrade, OverallGradePoint: termlyResult.Summary.OverallGradePoint,
			NumberPassed: termlyResult.Summary.NumberPassed, NumberFailed: termlyResult.Summary.NumberFailed,
			ClassPosition: termlyResult.Summary.ClassPosition, TotalStudentsInClass: termlyResult.Summary.TotalStudents,
		},
		ProposedNextClass:    proposedNextClass,
		PolicyChecksSummary:  policyChecks,
		ResultCompleteness:   resultCompleteness,
		SystemRecommendation: evaluation.Recommendation,
		FinalDecision:        finalDecision,
	}, nil
}

func formatPercent(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + "%"
}

// getPromotionDecision fetches the authorized final decision for one
// student's session, or nil if none has been recorded yet.
func (s *ExamService) getPromotionDecision(ctx context.Context, studentID, sessionID string) *models.PromotionDecision {
	var decision models.PromotionDecision
	if err := s.db.WithContext(ctx).
		Where("student_id = ? AND session_id = ?", studentID, sessionID).
		First(&decision).Error; err != nil {
		return nil
	}
	return &decision
}

// RecordPromotionDecision persists the authorized, final promotion
// decision for one student's session - the ONLY write path in this
// codebase that may set a PromotionDecision row. Never called from
// GetPromotionReportCardForUser or anywhere a recommendation is
// computed: a human (an admin/super_admin, per this method's own
// route gate) must explicitly call this themselves. schoolID must
// already be the caller's own resolved tenant.
func (s *ExamService) RecordPromotionDecision(ctx context.Context, schoolID, actorUserID string, req *dto.RecordPromotionDecisionRequest) error {
	student, err := s.examRepo.GetStudentByID(ctx, req.StudentID)
	if err != nil {
		return errors.New("student not found")
	}
	if student.SchoolID != schoolID {
		return errors.New("student does not belong to your school")
	}
	if req.Decision == "" {
		return errors.New("decision is required")
	}

	var decision models.PromotionDecision
	err = s.db.WithContext(ctx).
		Where("student_id = ? AND session_id = ?", req.StudentID, req.SessionID).
		First(&decision).Error
	if err == gorm.ErrRecordNotFound {
		decision = models.PromotionDecision{
			ID: models.GenerateID(), SchoolID: schoolID,
			StudentID: req.StudentID, SessionID: req.SessionID,
		}
	} else if err != nil {
		return err
	}

	decision.Decision = req.Decision
	decision.Notes = req.Notes
	decision.ProposedClassLevelID = req.ProposedClassLevelID
	decision.DecidedBy = actorUserID
	decision.DecidedAt = time.Now()
	return s.db.WithContext(ctx).Save(&decision).Error
}
