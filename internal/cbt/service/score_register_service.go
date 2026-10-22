package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/grading"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// builtinComponentLabels names a register column for an ExamType when
// no AssessmentScheme is configured for this scope - display labels
// only, matching the broad-sheet templates' own column names. Never
// used for grading/weighting math (that stays exactly
// grading.DefaultAssessmentMode's existing unweighted-pooled behavior,
// unchanged) - purely so an unconfigured school's register still shows
// friendly column headers instead of raw exam_type strings.
var builtinComponentLabels = map[string]string{
	string(models.ExamTypeContinuousAssessment): "CA",
	string(models.ExamTypeAssignment):           "Assignment",
	string(models.ExamTypeTest):                 "Test",
	string(models.ExamTypePractical):            "Practical",
	string(models.ExamTypeOral):                 "Oral",
	string(models.ExamTypeMainExam):              "Exam",
}

// scoreComponent is one register column, resolved internally - the
// Exam it reads/writes against, whether it targets TheoryScore instead
// of TotalScore, and whether it's read-only (a real CBT exam with
// questions attached, no theory portion - the register may only ever
// display, never edit, an online-attempt-derived score).
type scoreComponent struct {
	Code       string
	Name       string
	Exam       models.Exam
	IsTheory   bool
	ReadOnly   bool
}

// resolveScoreComponents is the single place that decides what columns
// the Student Score Register shows for one class/subject/term/session,
// and - critically - what each column reads/writes against. Shared by
// GetScoreRegister and SaveScoreRegister so a save can never target a
// column the read path wouldn't also show, and a combined CBT+theory
// exam (models.Exam.TheoryMaxMarks) always produces two columns: the
// read-only objective one and an editable theory one.
func (s *ExamService) resolveScoreComponents(ctx context.Context, classID, subjectID, termID, sessionID string) ([]scoreComponent, error) {
	var exams []models.Exam
	if err := s.db.WithContext(ctx).
		Where("class_id = ? AND subject_id = ? AND term_id = ? AND session_id = ?", classID, subjectID, termID, sessionID).
		Order("created_at ASC").
		Find(&exams).Error; err != nil {
		return nil, err
	}
	if len(exams) == 0 {
		return nil, nil
	}

	class, err := s.getClassByID(ctx, classID)
	if err != nil {
		return nil, err
	}
	scheme := s.gradingSvc.ResolveScheme(ctx, class.SchoolID, class.ClassLevelID, subjectID)
	labelFor := func(examType string) string {
		if scheme.Source != grading.DefaultAssessmentMode {
			for _, c := range scheme.Components {
				if examTypeInComponent(c.ExamTypes, examType) {
					return c.Name
				}
			}
		}
		if name, ok := builtinComponentLabels[examType]; ok {
			return name
		}
		return examType
	}

	var components []scoreComponent
	for _, exam := range exams {
		var questionCount int64
		s.db.WithContext(ctx).Model(&models.ExamQuestion{}).Where("exam_id = ?", exam.ID).Count(&questionCount)
		hasQuestions := questionCount > 0

		components = append(components, scoreComponent{
			Code: exam.ExamType, Name: labelFor(exam.ExamType), Exam: exam,
			ReadOnly: hasQuestions,
		})
		if exam.TheoryMaxMarks != nil {
			components = append(components, scoreComponent{
				Code: exam.ExamType + "_theory", Name: labelFor(exam.ExamType) + " (Theory)", Exam: exam,
				IsTheory: true, ReadOnly: false,
			})
		}
	}
	return components, nil
}

func examTypeInComponent(types []string, examType string) bool {
	for _, t := range types {
		if t == examType {
			return true
		}
	}
	return false
}

// GetScoreRegister is the Subject Teacher Score Register (broad-sheet
// template A): every enrolled student's row, every configured
// component's cell (nil score = not yet entered, never a 0 - see
// models.Result's own doc comment), and the authoritative
// total/percentage/grade computed through the exact same
// buildSubjectTermlyResult/grading engine the report card uses - never
// a second calculation.
func (s *ExamService) GetScoreRegister(ctx context.Context, schoolID string, isPlatformOwner bool, classID, subjectID, termID, sessionID string) (*dto.ScoreRegisterResponse, error) {
	if err := s.authorizeClassTenant(ctx, classID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	class, err := s.getClassByID(ctx, classID)
	if err != nil {
		return nil, err
	}
	subject, err := s.questionRepo.GetSubjectByID(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	components, err := s.resolveScoreComponents(ctx, classID, subjectID, termID, sessionID)
	if err != nil {
		return nil, err
	}

	var students []models.Student
	if err := s.db.WithContext(ctx).Preload("User").
		Where("class_id = ? AND is_active = ? AND deleted_at IS NULL", classID, true).
		Find(&students).Error; err != nil {
		return nil, err
	}
	sort.Slice(students, func(i, j int) bool { return students[i].AdmissionNo < students[j].AdmissionNo })

	allResults, err := s.examRepo.FindResultsByClassAndTerm(ctx, classID, termID)
	if err != nil {
		return nil, err
	}
	resultsByStudentExam := make(map[string]models.Result, len(allResults))
	resultsByStudent := make(map[string][]models.Result, len(students))
	examByID := make(map[string]models.Exam, len(components))
	for _, c := range components {
		examByID[c.Exam.ID] = c.Exam
	}
	for _, r := range allResults {
		if _, relevant := examByID[r.ExamID]; !relevant {
			continue
		}
		resultsByStudentExam[r.StudentID+"|"+r.ExamID] = r
		resultsByStudent[r.StudentID] = append(resultsByStudent[r.StudentID], r)
	}

	assignment, _ := s.findTeacherSubjectAssignment(ctx, classID, subjectID, sessionID)
	submission, _ := s.findSubjectResultSubmission(ctx, class.SchoolID, classID, subjectID, termID, sessionID)

	resp := &dto.ScoreRegisterResponse{
		ClassID: classID, ClassName: models.GetClassDisplayName(class),
		SubjectID: subjectID, SubjectName: subject.Name,
		TermID: termID, SessionID: sessionID,
		StudentsEnrolled: len(students),
	}
	resp.Components = make([]dto.ScoreComponentDef, len(components))
	for i, c := range components {
		maxMarks := c.Exam.TotalMarks
		if c.IsTheory && c.Exam.TheoryMaxMarks != nil {
			maxMarks = *c.Exam.TheoryMaxMarks
		}
		resp.Components[i] = dto.ScoreComponentDef{Code: c.Code, Name: c.Name, ExamID: c.Exam.ID, MaxMarks: maxMarks, ExamType: c.Exam.ExamType}
	}
	if assignment != nil {
		resp.TeacherID = assignment.TeacherID
		var teacher models.User
		if err := s.db.WithContext(ctx).Select("first_name", "last_name").First(&teacher, "id = ?", assignment.TeacherID).Error; err == nil {
			resp.TeacherName = strings.TrimSpace(teacher.FirstName + " " + teacher.LastName)
		}
	}
	if submission != nil {
		resp.SubmissionStatus = submission.Status
		resp.IsLocked = submission.IsLocked()
	} else {
		resp.SubmissionStatus = "draft"
	}

	var remarkRows []models.ScoreRegisterRemark
	remarksByStudent := map[string]string{}
	if err := s.db.WithContext(ctx).
		Where("class_id = ? AND subject_id = ? AND term_id = ? AND session_id = ?", classID, subjectID, termID, sessionID).
		Find(&remarkRows).Error; err == nil {
		for _, r := range remarkRows {
			remarksByStudent[r.StudentID] = r.Remark
		}
	}

	resultsComplete := 0
	for _, st := range students {
		row := dto.ScoreRegisterRow{
			StudentID: st.ID, AdmissionNumber: st.AdmissionNo,
			StudentName: strings.TrimSpace(st.User.FirstName + " " + st.User.LastName),
		}
		if remark, ok := remarksByStudent[st.ID]; ok {
			row.Remark = &remark
		}
		row.Cells = make([]dto.ScoreCell, len(components))
		for i, c := range components {
			cell := dto.ScoreCell{ComponentCode: c.Code, ReadOnly: c.ReadOnly}
			if c.IsTheory {
				cell.MaxMarks = 0
				if c.Exam.TheoryMaxMarks != nil {
					cell.MaxMarks = *c.Exam.TheoryMaxMarks
				}
			} else {
				cell.MaxMarks = c.Exam.TotalMarks
			}
			if r, ok := resultsByStudentExam[st.ID+"|"+c.Exam.ID]; ok {
				if c.IsTheory {
					cell.Score = r.TheoryScore
				} else {
					v := r.TotalScore
					cell.Score = &v
				}
			}
			row.Cells[i] = cell
		}

		assessment, err := s.buildSubjectTermlyResult(ctx, st.ID, subjectID, class.SchoolID, class.ClassLevelID, resultsByStudent[st.ID])
		if err == nil && (assessment.TotalMarks > 0 || len(assessment.Exams) > 0) {
			total, maxTotal, pct := assessment.TotalScore, assessment.TotalMarks, assessment.Percentage
			row.Total, row.MaxTotal, row.Percentage, row.Grade = &total, maxTotal, &pct, assessment.Grade
			row.IsComplete = assessment.IsComplete
			if assessment.IsComplete {
				resultsComplete++
			}
		}
		resp.Rows = append(resp.Rows, row)
	}
	resp.ResultsComplete = resultsComplete
	return resp, nil
}

// findTeacherSubjectAssignment/findSubjectResultSubmission are small,
// "nil if none" lookups - neither condition is an error the caller
// needs to see (no assignment yet, or still in an unsubmitted draft,
// are both ordinary states, not failures).
func (s *ExamService) findTeacherSubjectAssignment(ctx context.Context, classID, subjectID, sessionID string) (*models.TeacherSubjectAssignment, error) {
	var a models.TeacherSubjectAssignment
	err := s.db.WithContext(ctx).
		Where("class_id = ? AND subject_id = ? AND session_id = ? AND is_active = ?", classID, subjectID, sessionID, true).
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *ExamService) findSubjectResultSubmission(ctx context.Context, schoolID, classID, subjectID, termID, sessionID string) (*models.SubjectResultSubmission, error) {
	var sub models.SubjectResultSubmission
	err := s.db.WithContext(ctx).
		Where("school_id = ? AND class_id = ? AND subject_id = ? AND term_id = ? AND session_id = ?", schoolID, classID, subjectID, termID, sessionID).
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// SaveScoreRegister batch-saves every edited cell of ONE component
// (one register column) - mirrors the attendance register's own
// roster -> edit-buffer -> one-batch-save shape. A score of nil clears
// the cell (deletes the Result row for a plain component, or just
// nils TheoryScore on a combined exam's row, so the row's objective
// score survives). Rejects: a read-only (CBT-sourced) component, a
// score above the component's configured max, and - unless
// CorrectionReason is given - any save once the scope's
// SubjectResultSubmission has been reviewed or approved (see
// models.SubjectResultSubmission.IsLocked).
func (s *ExamService) SaveScoreRegister(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.SaveScoreRegisterRequest) (*dto.SaveScoreRegisterResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	class, err := s.getClassByID(ctx, req.ClassID)
	if err != nil {
		return nil, err
	}

	submission, _ := s.findSubjectResultSubmission(ctx, class.SchoolID, req.ClassID, req.SubjectID, req.TermID, req.SessionID)
	locked := submission != nil && submission.IsLocked()
	if locked && strings.TrimSpace(req.CorrectionReason) == "" {
		return nil, errors.New("these results have already been reviewed/approved - a correction_reason is required to edit them now")
	}

	components, err := s.resolveScoreComponents(ctx, req.ClassID, req.SubjectID, req.TermID, req.SessionID)
	if err != nil {
		return nil, err
	}
	var target *scoreComponent
	for i := range components {
		if components[i].Code == req.ComponentCode {
			target = &components[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("no such component %q for this class/subject/term/session - create its Exam first", req.ComponentCode)
	}
	if target.ReadOnly {
		return nil, fmt.Errorf("component %q is sourced from a CBT exam attempt and cannot be edited directly", req.ComponentCode)
	}
	maxMarks := target.Exam.TotalMarks
	if target.IsTheory && target.Exam.TheoryMaxMarks != nil {
		maxMarks = *target.Exam.TheoryMaxMarks
	}

	students, err := s.examRepo.GetStudentsByClass(ctx, req.ClassID)
	if err != nil {
		return nil, err
	}
	inClass := make(map[string]bool, len(students))
	for _, st := range students {
		inClass[st.ID] = true
	}

	resp := &dto.SaveScoreRegisterResponse{Saved: []dto.SaveScoreRegisterResult{}, Failed: []dto.SaveScoreRegisterResult{}}
	savedCount := 0
	for _, entry := range req.Entries {
		if !inClass[entry.StudentID] {
			resp.Failed = append(resp.Failed, dto.SaveScoreRegisterResult{StudentID: entry.StudentID, Error: "student is not an active member of this class"})
			continue
		}
		if entry.Score != nil && (*entry.Score < 0 || *entry.Score > maxMarks) {
			resp.Failed = append(resp.Failed, dto.SaveScoreRegisterResult{StudentID: entry.StudentID, Error: fmt.Sprintf("score must be between 0 and %d", maxMarks)})
			continue
		}
		if err := s.saveOneScoreCell(ctx, entry.StudentID, target, entry.Score); err != nil {
			resp.Failed = append(resp.Failed, dto.SaveScoreRegisterResult{StudentID: entry.StudentID, Error: err.Error()})
			continue
		}
		resp.Saved = append(resp.Saved, dto.SaveScoreRegisterResult{StudentID: entry.StudentID})
		savedCount++
	}

	if savedCount > 0 {
		action := "score_saved"
		if locked {
			action = "corrected"
		}
		s.writeSubjectResultAudit(ctx, &models.SubjectResultAuditLog{
			SchoolID: class.SchoolID, ActorID: actorID, Action: action,
			ClassID: req.ClassID, SubjectID: req.SubjectID, TermID: req.TermID, SessionID: req.SessionID,
			Notes: fmt.Sprintf("component=%s saved=%d reason=%s", req.ComponentCode, savedCount, req.CorrectionReason),
		})
	}
	return resp, nil
}

// SaveScoreRegisterRemark upserts one student's teacher remark
// (template A's "Teacher remark" column) - subject to the exact same
// lock/correction-reason rule as a score cell, since a remark is still
// part of the subject result record a reviewer has already signed off
// on. An empty Remark clears it rather than deleting the row, so the
// student keeps a (now-empty) remark history entry rather than
// silently reverting to "no remark ever written".
func (s *ExamService) SaveScoreRegisterRemark(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.SaveScoreRegisterRemarkRequest) error {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return err
	}
	class, err := s.getClassByID(ctx, req.ClassID)
	if err != nil {
		return err
	}
	submission, _ := s.findSubjectResultSubmission(ctx, class.SchoolID, req.ClassID, req.SubjectID, req.TermID, req.SessionID)
	if submission != nil && submission.IsLocked() && strings.TrimSpace(req.CorrectionReason) == "" {
		return errors.New("these results have already been reviewed/approved - a correction_reason is required to edit them now")
	}

	var existing models.ScoreRegisterRemark
	err = s.db.WithContext(ctx).
		Where("student_id = ? AND subject_id = ? AND term_id = ? AND session_id = ?", req.StudentID, req.SubjectID, req.TermID, req.SessionID).
		First(&existing).Error
	switch {
	case err == nil:
		existing.Remark = req.Remark
		err = s.db.WithContext(ctx).Save(&existing).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		err = s.db.WithContext(ctx).Create(&models.ScoreRegisterRemark{
			SchoolID: class.SchoolID, StudentID: req.StudentID, ClassID: req.ClassID,
			SubjectID: req.SubjectID, TermID: req.TermID, SessionID: req.SessionID, Remark: req.Remark,
		}).Error
	}
	if err != nil {
		return err
	}

	action := "remark_saved"
	if submission != nil && submission.IsLocked() {
		action = "corrected"
	}
	s.writeSubjectResultAudit(ctx, &models.SubjectResultAuditLog{
		SchoolID: class.SchoolID, ActorID: actorID, Action: action,
		ClassID: req.ClassID, SubjectID: req.SubjectID, TermID: req.TermID, SessionID: req.SessionID,
		StudentID: &req.StudentID, Notes: "teacher remark updated reason=" + req.CorrectionReason,
	})
	return nil
}

// saveOneScoreCell writes (or clears) one student's one cell. A plain
// (non-theory) component with a nil score deletes the Result row
// entirely - no row means "not recorded", never a stored 0 (see
// models.Result's own doc comment) - while a theory cell only nils
// TheoryScore on the existing row, since that row's objective
// CBT-derived TotalScore must survive a theory-score clear.
func (s *ExamService) saveOneScoreCell(ctx context.Context, studentID string, comp *scoreComponent, score *int) error {
	existing, err := s.examRepo.FindResultByExamAndStudent(comp.Exam.ID, studentID)
	if err != nil {
		return err
	}

	if score == nil {
		if existing == nil {
			return nil
		}
		if comp.IsTheory {
			existing.TheoryScore = nil
			return s.examRepo.UpdateResult(existing)
		}
		return s.db.WithContext(ctx).Delete(existing).Error
	}

	if existing == nil {
		if comp.IsTheory {
			// A theory score with no CBT attempt yet recorded for this
			// exam - create the row now; the objective TotalScore stays
			// its zero value until a real attempt is graded (or never,
			// for a paper-only exam that also happens to carry a
			// theory-portion configuration, an unusual but not invalid
			// setup).
			existing = &models.Result{ID: models.GenerateID(), StudentID: studentID, ExamID: comp.Exam.ID}
			existing.TheoryScore = score
			return s.examRepo.CreateResult(existing)
		}
		pct := 0.0
		if comp.Exam.TotalMarks > 0 {
			pct = float64(*score) / float64(comp.Exam.TotalMarks) * 100
		}
		grade, _, _ := grading.Builtin(pct)
		existing = &models.Result{
			ID: models.GenerateID(), StudentID: studentID, ExamID: comp.Exam.ID,
			TotalScore: *score, Percentage: pct, Grade: grade, Passed: pct >= grading.BuiltinPassMark, FullyGraded: true,
		}
		return s.examRepo.CreateResult(existing)
	}

	if comp.IsTheory {
		existing.TheoryScore = score
		return s.examRepo.UpdateResult(existing)
	}
	existing.TotalScore = *score
	if comp.Exam.TotalMarks > 0 {
		existing.Percentage = float64(*score) / float64(comp.Exam.TotalMarks) * 100
	}
	existing.Grade, _, _ = grading.Builtin(existing.Percentage)
	existing.Passed = existing.Percentage >= grading.BuiltinPassMark
	existing.FullyGraded = true
	return s.examRepo.UpdateResult(existing)
}

// writeSubjectResultAudit mirrors writeReportCardAudit exactly (same
// fire-and-forget-on-failure design, same tamper-evident content hash,
// same microsecond truncation to match what Postgres's timestamptz
// column actually stores and returns - see that function's own doc
// comment for why untruncated nanosecond precision would make every
// freshly-written row immediately fail its own integrity check).
func (s *ExamService) writeSubjectResultAudit(ctx context.Context, log *models.SubjectResultAuditLog) {
	log.ID = models.GenerateID()
	log.CreatedAt = time.Now().Truncate(time.Microsecond)
	log.ContentHash = subjectResultAuditContentHash(
		log.SchoolID, log.ActorID, log.Action, log.ClassID, log.SubjectID,
		log.TermID, log.SessionID, strDeref(log.StudentID), log.Notes, log.CreatedAt,
	)
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		fmt.Printf("failed to write subject-result audit log: %v\n", err)
	}
}

// subjectResultAuditContentHash mirrors reportCardAuditContentHash
// exactly (internal/cbt/service/report_card_approval_service.go) -
// same tamper-evidence design, a sibling table rather than a shared
// hash function only because the two log different column sets.
func subjectResultAuditContentHash(schoolID, actorID, action, classID, subjectID, termID, sessionID, studentID, notes string, createdAt time.Time) string {
	h := sha256.New()
	for _, part := range []string{schoolID, actorID, action, classID, subjectID, termID, sessionID, studentID, notes} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	h.Write([]byte(createdAt.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))
}
