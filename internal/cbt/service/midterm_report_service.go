package service

import (
	"context"
	"errors"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/grading"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// midTermExcludedExamTypes are exam types a Mid-Term Progress Report
// must never include - a progress snapshot is explicitly NOT a final
// result (see dto.MidTermSubjectResult's own doc comment), so any
// exam of a "this term is now finished" type is excluded even if one
// happens to already exist in the data (e.g. a teacher entered the
// main exam early). Continuous assessment, weekly tests, quizzes,
// assignments and mock exams are all still included - exactly the
// components a genuine progress snapshot is for.
var midTermExcludedExamTypes = map[string]bool{
	string(models.ExamTypeMainExam):    true,
	string(models.ExamTypeEndOfTerm):   true,
	string(models.ExamTypeFinalExam):   true,
	string(models.ExamTypePromotional): true,
}

// buildMidTermSubjectResult mirrors buildSubjectTermlyResult exactly,
// except it filters out midTermExcludedExamTypes before resolving the
// scheme and computing the assessment - the same shared grading engine
// (internal/grading), just given a narrower slice of exams. A school
// with no final-type exam entered yet for this subject sees identical
// output either way; this only changes behavior once a final exam
// result exists, which a mid-term snapshot must never reflect.
func (s *ExamService) buildMidTermSubjectResult(ctx context.Context, subjectID, schoolID, classLevelID string, allResults []models.Result) (dto.MidTermSubjectResult, error) {
	var scoredExams []grading.ScoredExam
	totalScore, totalMarks := 0, 0

	for _, result := range allResults {
		exam, err := s.examRepo.FindExamByIDWithContext(ctx, result.ExamID)
		if err != nil {
			continue
		}
		if exam.SubjectID != subjectID {
			continue
		}
		if midTermExcludedExamTypes[exam.ExamType] {
			continue
		}
		// Theory combining (Exam.TheoryMaxMarks/Result.TheoryScore) is a
		// main-exam concept in practice - main_exam is already excluded
		// above - but handled defensively here too, same rule as
		// buildSubjectTermlyResult, in case a school ever configures it
		// on an included exam type.
		examScore, examMarks := result.TotalScore, exam.TotalMarks
		if exam.TheoryMaxMarks != nil {
			if result.TheoryScore == nil {
				continue
			}
			examScore += *result.TheoryScore
			examMarks += *exam.TheoryMaxMarks
		}
		scoredExams = append(scoredExams, grading.ScoredExam{
			ExamType:    exam.ExamType,
			TotalMarks:  examMarks,
			TotalScore:  examScore,
			FullyGraded: result.FullyGraded,
		})
		totalScore += examScore
		totalMarks += examMarks
	}

	scheme := s.gradingSvc.ResolveScheme(ctx, schoolID, classLevelID, subjectID)
	assessment := grading.ComputeSubjectAssessment(scheme, scoredExams)

	subject, err := s.questionRepo.GetSubjectByID(ctx, subjectID)
	if err != nil {
		return dto.MidTermSubjectResult{}, err
	}

	components := make([]dto.SubjectComponentResult, len(assessment.Components))
	for i, c := range assessment.Components {
		components[i] = dto.SubjectComponentResult{
			Code: c.Code, Name: c.Name, Weight: c.Weight,
			Percentage: c.Percentage, HasResults: c.HasResults,
		}
	}

	return dto.MidTermSubjectResult{
		SubjectName: subject.Name,
		SubjectCode: subject.Code,
		Components:  components,
		TotalScore:  totalScore,
		TotalMarks:  totalMarks,
		Percentage:  assessment.TotalPercent,
		IsComplete:  assessment.IsComplete,
	}, nil
}

// GetMidTermProgressReportForUser builds the authenticated student's
// Mid-Term Progress Report Card - a snapshot of whatever CA/test
// components have real results so far this term, explicitly excluding
// any final-exam-type result (see midTermExcludedExamTypes) and never
// assigning a letter grade (dto.MidTermSubjectResult has no Grade
// field to assign one to).
func (s *ExamService) GetMidTermProgressReportForUser(ctx context.Context, termID, sessionID, userID string) (*dto.MidTermReportCardResponse, error) {
	student, err := s.examRepo.GetStudentByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	return s.GetMidTermProgressReport(ctx, student, termID, sessionID)
}

// GetMidTermProgressReport builds the Mid-Term Progress Report Card
// for an already-resolved student - the shared builder behind
// GetMidTermProgressReportForUser (student-self) and the batch/admin
// report-card paths (collectClassReportCards), the same split as
// GetTermlyReportCard/GetReportCardForUser.
func (s *ExamService) GetMidTermProgressReport(ctx context.Context, student *models.Student, termID, sessionID string) (*dto.MidTermReportCardResponse, error) {
	classDisplayName, classTeacherName, classLevelID := s.resolveClassWithTeacher(ctx, student.ClassID)

	school, err := s.getSchoolByID(ctx, student.SchoolID)
	if err != nil {
		return nil, err
	}
	term, err := s.getTermByID(ctx, termID)
	if err != nil {
		return nil, err
	}
	session, err := s.getSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	results, err := s.examRepo.FindResultsByStudentAndTerm(ctx, student.ID, termID, sessionID)
	if err != nil {
		return nil, err
	}
	subjects, err := s.examRepo.GetSubjectsByClassAndTerm(ctx, student.ClassID, termID)
	if err != nil {
		return nil, err
	}

	subjectResults := make([]dto.MidTermSubjectResult, 0, len(subjects))
	totalPercentSum, completeCount, passedCount := 0.0, 0, 0
	totalScoreSum, totalMarksSum := 0, 0
	for _, subject := range subjects {
		sr, err := s.buildMidTermSubjectResult(ctx, subject.ID, student.SchoolID, classLevelID, results)
		if err != nil {
			continue
		}
		subjectResults = append(subjectResults, sr)
		totalScoreSum += sr.TotalScore
		totalMarksSum += sr.TotalMarks
		if sr.IsComplete {
			completeCount++
			totalPercentSum += sr.Percentage
			if s.gradingSvc.ComputeGrade(ctx, student.SchoolID, classLevelID, sr.Percentage).Passed {
				passedCount++
			}
		}
	}
	overallPercentage := 0.0
	if completeCount > 0 {
		overallPercentage = totalPercentSum / float64(completeCount)
	}

	teacherComment, nextSteps := "", ""
	if remark := s.getReportCardRemark(ctx, student.ID, termID, sessionID); remark != nil {
		teacherComment = remark.MidTermComment
		nextSteps = remark.MidTermNextSteps
	}

	return &dto.MidTermReportCardResponse{
		School: dto.SchoolInfo{
			ID: school.ID, Name: school.Name, Address: school.Address,
			Logo: school.Logo, PrimaryColor: school.PrimaryColor,
		},
		Student: dto.StudentInfo{
			ID: student.ID, Name: student.User.FirstName + " " + student.User.LastName,
			AdmissionNumber: student.AdmissionNo, Class: classDisplayName,
			ClassID: student.ClassID, School: school.Name, SchoolID: student.SchoolID,
		},
		Class:   dto.ClassInfo{ID: student.ClassID, Name: classDisplayName, ClassTeacher: classTeacherName},
		Term:    dto.TermInfo{ID: term.ID, Name: term.Name, Number: term.TermNumber},
		Session: dto.SessionInfo{ID: session.ID, Name: session.Name},
		Subjects: subjectResults,
		Summary: dto.ReportCardSummary{
			TotalSubjects: len(subjects), TotalScore: totalScoreSum, TotalMarks: totalMarksSum,
			OverallPercentage: overallPercentage,
			NumberPassed:      passedCount, NumberFailed: completeCount - passedCount,
		},
		// nil until this class/term has had at least one day's register
		// taken - see GetTermlyReportCardForStudent's identical comment
		// on why this must stay nil rather than an all-zero struct.
		Attendance: func() *dto.MidTermAttendance {
			present, absent, open, aErr := s.attendanceSummary(ctx, student.ID, student.ClassID, termID)
			if aErr != nil || open == 0 {
				return nil
			}
			// DaysPossible is "school days so far this term" - exactly
			// what attendanceSummary's own openDays count already is
			// (every date with a register, which for a Mid-Term report
			// built partway through the term is naturally only the days
			// that have actually happened). HomeworkNote is left empty:
			// homework/assignment tracking is a separate, not-yet-built
			// feature this struct's field anticipates but this task
			// does not implement.
			return &dto.MidTermAttendance{DaysPossible: open, DaysPresent: present, DaysAbsent: absent}
		}(),
		ClassTeacherComment: teacherComment,
		NextSteps:           nextSteps,
	}, nil
}

// SetMidTermRemark records a class teacher's mid-term progress comment
// and next-steps note - separate from, and never overwriting, the
// Main Termly end-of-term TeacherRemark on the same
// ReportCardRemark row.
func (s *ExamService) SetMidTermRemark(ctx context.Context, schoolID, actorUserID string, req *dto.SetMidTermRemarkRequest) error {
	student, err := s.examRepo.GetStudentByID(ctx, req.StudentID)
	if err != nil {
		return errors.New("student not found")
	}
	if student.SchoolID != schoolID {
		return errors.New("student does not belong to your school")
	}

	var remark models.ReportCardRemark
	err = s.db.WithContext(ctx).
		Where("student_id = ? AND term_id = ? AND session_id = ?", req.StudentID, req.TermID, req.SessionID).
		First(&remark).Error
	if err == gorm.ErrRecordNotFound {
		remark = models.ReportCardRemark{
			ID: models.GenerateID(), SchoolID: schoolID,
			StudentID: req.StudentID, TermID: req.TermID, SessionID: req.SessionID,
		}
	} else if err != nil {
		return err
	}

	now := time.Now()
	remark.MidTermComment = req.Comment
	remark.MidTermNextSteps = req.NextSteps
	remark.MidTermSetBy = &actorUserID
	remark.MidTermSetAt = &now
	return s.db.WithContext(ctx).Save(&remark).Error
}
