package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// scopeScheme/scopeScale compute the ScopeKey models.AssessmentScheme/
// models.GradingScale rely on to make a nullable (class_level_id[,
// subject_id]) pair enforceable via one ordinary composite unique
// index - see those fields' own doc comments for why a nullable column
// alone can't be made unique in Postgres. Deliberately simple and
// collision-free: "|" never appears in a UUID, so joining with it can
// never make two distinct (classLevel, subject) pairs collide.
func scopeScheme(classLevelID, subjectID string) string {
	return classLevelID + "|" + subjectID
}

func scopeScale(classLevelID string) string {
	if classLevelID == "" {
		return "school"
	}
	return classLevelID
}

// UpsertAssessmentScheme creates or replaces the AssessmentScheme at
// the given scope (school-wide if both ClassLevelID/SubjectID are
// empty) and its full set of components, validating that weights sum
// to 100 and no ExamType is claimed by more than one component - see
// models.AssessmentComponent's own doc comment for why that's required
// rather than DB-enforced. Replacing (not patching) a scheme's
// components on every call keeps this idempotent and avoids a stale
// component surviving an update that meant to remove it.
func (s *ExamService) UpsertAssessmentScheme(ctx context.Context, schoolID string, isPlatformOwner bool, targetSchoolID string, actorID string, req *dto.AssessmentSchemeRequest) (*dto.AssessmentSchemeResponse, error) {
	effSchool := examEffectiveSchoolID(targetSchoolID, schoolID, isPlatformOwner)
	if effSchool == "" {
		return nil, errors.New("school context is required")
	}
	if err := validateAssessmentComponents(req.Components); err != nil {
		return nil, err
	}
	missingPolicy := req.MissingPolicy
	if missingPolicy == "" {
		missingPolicy = "renormalize"
	}
	if missingPolicy != "renormalize" && missingPolicy != "zero" {
		return nil, errors.New("missing_policy must be 'renormalize' or 'zero'")
	}

	var scheme models.AssessmentScheme
	scopeKey := scopeScheme(req.ClassLevelID, req.SubjectID)
	err := s.db.WithContext(ctx).
		Where("school_id = ? AND scope_key = ?", effSchool, scopeKey).
		First(&scheme).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		scheme = models.AssessmentScheme{ID: models.GenerateID(), SchoolID: effSchool, ScopeKey: scopeKey}
	case err != nil:
		return nil, err
	}
	if req.ClassLevelID != "" {
		scheme.ClassLevelID = &req.ClassLevelID
	} else {
		scheme.ClassLevelID = nil
	}
	if req.SubjectID != "" {
		scheme.SubjectID = &req.SubjectID
	} else {
		scheme.SubjectID = nil
	}
	scheme.MissingPolicy = missingPolicy
	scheme.IsActive = true
	scheme.UpdatedBy = &actorID

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&scheme).Error; err != nil {
			return err
		}
		if err := tx.Where("scheme_id = ?", scheme.ID).Delete(&models.AssessmentComponent{}).Error; err != nil {
			return err
		}
		components := make([]models.AssessmentComponent, len(req.Components))
		for i, c := range req.Components {
			agg := c.Aggregation
			if agg == "" {
				agg = "pooled"
			}
			components[i] = models.AssessmentComponent{
				ID: models.GenerateID(), SchemeID: scheme.ID, Code: c.Code, Name: c.Name,
				Weight: c.Weight, ExamTypes: models.StringArray(c.ExamTypes), Aggregation: agg, SortOrder: c.SortOrder,
			}
		}
		return tx.Create(&components).Error
	})
	if err != nil {
		return nil, err
	}
	return s.getAssessmentSchemeResponse(ctx, scheme.ID)
}

func validateAssessmentComponents(components []dto.AssessmentComponentInput) error {
	var weightSum float64
	seenExamTypes := map[string]string{}
	seenCodes := map[string]bool{}
	for _, c := range components {
		if seenCodes[c.Code] {
			return fmt.Errorf("duplicate component code %q", c.Code)
		}
		seenCodes[c.Code] = true
		weightSum += c.Weight
		for _, et := range c.ExamTypes {
			if owner, ok := seenExamTypes[et]; ok {
				return fmt.Errorf("exam type %q is claimed by both component %q and %q", et, owner, c.Code)
			}
			seenExamTypes[et] = c.Code
		}
	}
	// Tolerate float rounding (e.g. 33.33 x 3) within 0.5 points.
	if weightSum < 99.5 || weightSum > 100.5 {
		return fmt.Errorf("component weights must sum to 100, got %.2f", weightSum)
	}
	return nil
}

func (s *ExamService) getAssessmentSchemeResponse(ctx context.Context, id string) (*dto.AssessmentSchemeResponse, error) {
	var scheme models.AssessmentScheme
	if err := s.db.WithContext(ctx).Preload("Components").First(&scheme, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return toAssessmentSchemeResponse(&scheme), nil
}

func toAssessmentSchemeResponse(scheme *models.AssessmentScheme) *dto.AssessmentSchemeResponse {
	resp := &dto.AssessmentSchemeResponse{
		ID: scheme.ID, SchoolID: scheme.SchoolID, MissingPolicy: scheme.MissingPolicy,
		IsActive: scheme.IsActive, CreatedAt: scheme.CreatedAt, UpdatedAt: scheme.UpdatedAt,
	}
	if scheme.ClassLevelID != nil {
		resp.ClassLevelID = *scheme.ClassLevelID
	}
	if scheme.SubjectID != nil {
		resp.SubjectID = *scheme.SubjectID
	}
	resp.Components = make([]dto.AssessmentComponentResponse, len(scheme.Components))
	for i, c := range scheme.Components {
		resp.Components[i] = dto.AssessmentComponentResponse{
			ID: c.ID, Code: c.Code, Name: c.Name, Weight: c.Weight,
			ExamTypes: []string(c.ExamTypes), Aggregation: c.Aggregation, SortOrder: c.SortOrder,
		}
	}
	return resp
}

// ListAssessmentSchemes returns every active scheme for a school,
// narrowed to one class level when provided.
func (s *ExamService) ListAssessmentSchemes(ctx context.Context, schoolID string, isPlatformOwner bool, targetSchoolID, classLevelID string) ([]dto.AssessmentSchemeResponse, error) {
	effSchool := examEffectiveSchoolID(targetSchoolID, schoolID, isPlatformOwner)
	if effSchool == "" {
		return nil, errors.New("school context is required")
	}
	q := s.db.WithContext(ctx).Preload("Components").Where("school_id = ? AND is_active = ?", effSchool, true)
	if classLevelID != "" {
		q = q.Where("class_level_id = ? OR class_level_id IS NULL", classLevelID)
	}
	var schemes []models.AssessmentScheme
	if err := q.Find(&schemes).Error; err != nil {
		return nil, err
	}
	out := make([]dto.AssessmentSchemeResponse, len(schemes))
	for i := range schemes {
		out[i] = *toAssessmentSchemeResponse(&schemes[i])
	}
	return out, nil
}

// DeactivateAssessmentScheme soft-disables a scheme (IsActive = false)
// rather than deleting it - grading.ResolveScheme only considers
// active schemes, so this immediately reverts that scope to the next
// broader one (or the built-in default), while the row and its past
// association with any already-approved report stays intact for audit
// purposes.
func (s *ExamService) DeactivateAssessmentScheme(ctx context.Context, schoolID string, isPlatformOwner bool, id string) error {
	q := s.db.WithContext(ctx).Model(&models.AssessmentScheme{}).Where("id = ?", id)
	if !isPlatformOwner {
		q = q.Where("school_id = ?", schoolID)
	}
	res := q.Update("is_active", false)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("assessment scheme not found")
	}
	return nil
}

// UpsertGradingScale mirrors UpsertAssessmentScheme for
// models.GradingScale/GradingScaleBand - see grading.ValidateBands for
// the band-set invariants enforced here.
func (s *ExamService) UpsertGradingScale(ctx context.Context, schoolID string, isPlatformOwner bool, targetSchoolID string, actorID string, req *dto.GradingScaleRequest) (*dto.GradingScaleResponse, error) {
	effSchool := examEffectiveSchoolID(targetSchoolID, schoolID, isPlatformOwner)
	if effSchool == "" {
		return nil, errors.New("school context is required")
	}
	bands := make([]gradingBand, len(req.Bands))
	for i, b := range req.Bands {
		bands[i] = gradingBand{Grade: b.Grade, MinPercent: b.MinPercent}
	}
	if err := validateBandSet(bands); err != nil {
		return nil, err
	}

	var scale models.GradingScale
	scopeKey := scopeScale(req.ClassLevelID)
	err := s.db.WithContext(ctx).
		Where("school_id = ? AND scope_key = ?", effSchool, scopeKey).
		First(&scale).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		scale = models.GradingScale{ID: models.GenerateID(), SchoolID: effSchool, ScopeKey: scopeKey}
	case err != nil:
		return nil, err
	}
	if req.ClassLevelID != "" {
		scale.ClassLevelID = &req.ClassLevelID
	} else {
		scale.ClassLevelID = nil
	}
	scale.Name = req.Name
	scale.PassMark = req.PassMark
	scale.IsActive = true
	scale.UpdatedBy = &actorID

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&scale).Error; err != nil {
			return err
		}
		if err := tx.Where("grading_scale_id = ?", scale.ID).Delete(&models.GradingScaleBand{}).Error; err != nil {
			return err
		}
		rows := make([]models.GradingScaleBand, len(req.Bands))
		for i, b := range req.Bands {
			rows[i] = models.GradingScaleBand{
				ID: models.GenerateID(), GradingScaleID: scale.ID, Grade: b.Grade,
				MinPercent: b.MinPercent, GradePoint: b.GradePoint, Remark: b.Remark, SortOrder: b.SortOrder,
			}
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	return s.getGradingScaleResponse(ctx, scale.ID)
}

// gradingBand/validateBandSet is a thin, dto-independent re-expression
// of grading.ValidateBands(bands []grading.Band) - kept local to avoid
// this handler-facing package importing internal/grading's Band type
// just to validate a request shape.
type gradingBand struct {
	Grade      string
	MinPercent float64
}

func validateBandSet(bands []gradingBand) error {
	if len(bands) == 0 {
		return errors.New("a grading scale must have at least one band")
	}
	seen := map[float64]bool{}
	hasZero := false
	for _, b := range bands {
		if strings.TrimSpace(b.Grade) == "" {
			return errors.New("every band must have a grade label")
		}
		if seen[b.MinPercent] {
			return fmt.Errorf("duplicate band boundary at %.2f%% (grade %s)", b.MinPercent, b.Grade)
		}
		seen[b.MinPercent] = true
		if b.MinPercent == 0 {
			hasZero = true
		}
		if b.MinPercent < 0 || b.MinPercent > 100 {
			return fmt.Errorf("band boundary %.2f%% (grade %s) must be between 0 and 100", b.MinPercent, b.Grade)
		}
	}
	if !hasZero {
		return errors.New("a grading scale must have a band starting at 0% so every percentage resolves to a grade")
	}
	return nil
}

func (s *ExamService) getGradingScaleResponse(ctx context.Context, id string) (*dto.GradingScaleResponse, error) {
	var scale models.GradingScale
	if err := s.db.WithContext(ctx).Preload("Bands").First(&scale, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return toGradingScaleResponse(&scale), nil
}

func toGradingScaleResponse(scale *models.GradingScale) *dto.GradingScaleResponse {
	resp := &dto.GradingScaleResponse{
		ID: scale.ID, SchoolID: scale.SchoolID, Name: scale.Name, PassMark: scale.PassMark,
		IsActive: scale.IsActive, CreatedAt: scale.CreatedAt, UpdatedAt: scale.UpdatedAt,
	}
	if scale.ClassLevelID != nil {
		resp.ClassLevelID = *scale.ClassLevelID
	}
	resp.Bands = make([]dto.GradingScaleBandResponse, len(scale.Bands))
	for i, b := range scale.Bands {
		resp.Bands[i] = dto.GradingScaleBandResponse{
			ID: b.ID, Grade: b.Grade, MinPercent: b.MinPercent,
			GradePoint: b.GradePoint, Remark: b.Remark, SortOrder: b.SortOrder,
		}
	}
	return resp
}

// ListGradingScales returns every active grading scale for a school.
func (s *ExamService) ListGradingScales(ctx context.Context, schoolID string, isPlatformOwner bool, targetSchoolID string) ([]dto.GradingScaleResponse, error) {
	effSchool := examEffectiveSchoolID(targetSchoolID, schoolID, isPlatformOwner)
	if effSchool == "" {
		return nil, errors.New("school context is required")
	}
	var scales []models.GradingScale
	if err := s.db.WithContext(ctx).Preload("Bands").Where("school_id = ? AND is_active = ?", effSchool, true).Find(&scales).Error; err != nil {
		return nil, err
	}
	out := make([]dto.GradingScaleResponse, len(scales))
	for i := range scales {
		out[i] = *toGradingScaleResponse(&scales[i])
	}
	return out, nil
}

// DeactivateGradingScale mirrors DeactivateAssessmentScheme.
func (s *ExamService) DeactivateGradingScale(ctx context.Context, schoolID string, isPlatformOwner bool, id string) error {
	q := s.db.WithContext(ctx).Model(&models.GradingScale{}).Where("id = ?", id)
	if !isPlatformOwner {
		q = q.Where("school_id = ?", schoolID)
	}
	res := q.Update("is_active", false)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("grading scale not found")
	}
	return nil
}

// ============================================
// TEACHER SUBJECT ASSIGNMENTS
// ============================================

// CreateTeacherSubjectAssignment assigns a teacher to a subject within
// a class for a session - the record the Student Score Register, "my
// subjects", and the HOD/Examination Officer broad sheet all resolve
// against (see models.TeacherSubjectAssignment's own doc comment for
// why this is distinct from Class.TeacherID).
func (s *ExamService) CreateTeacherSubjectAssignment(ctx context.Context, schoolID string, isPlatformOwner bool, req *dto.CreateTeacherSubjectAssignmentRequest) (*dto.TeacherSubjectAssignmentResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	assignment := &models.TeacherSubjectAssignment{
		ID: models.GenerateID(), SchoolID: schoolID, TeacherID: req.TeacherID,
		SubjectID: req.SubjectID, ClassID: req.ClassID, SessionID: req.SessionID, IsActive: true,
	}
	if err := s.db.WithContext(ctx).Create(assignment).Error; err != nil {
		return nil, err
	}
	return s.toTeacherSubjectAssignmentResponse(ctx, assignment)
}

func (s *ExamService) toTeacherSubjectAssignmentResponse(ctx context.Context, a *models.TeacherSubjectAssignment) (*dto.TeacherSubjectAssignmentResponse, error) {
	resp := &dto.TeacherSubjectAssignmentResponse{
		ID: a.ID, TeacherID: a.TeacherID, SubjectID: a.SubjectID,
		ClassID: a.ClassID, SessionID: a.SessionID, IsActive: a.IsActive,
	}
	var teacher models.User
	if err := s.db.WithContext(ctx).Select("first_name", "last_name").First(&teacher, "id = ?", a.TeacherID).Error; err == nil {
		resp.TeacherName = strings.TrimSpace(teacher.FirstName + " " + teacher.LastName)
	}
	if subject, err := s.questionRepo.GetSubjectByID(ctx, a.SubjectID); err == nil {
		resp.SubjectName = subject.Name
	}
	if class, err := s.getClassByID(ctx, a.ClassID); err == nil {
		resp.ClassName = models.GetClassDisplayName(class)
	}
	return resp, nil
}

// ListTeacherSubjectAssignments lists a class's (or, with classID
// empty, every) assignment for a school, optionally narrowed to one
// teacher or session.
func (s *ExamService) ListTeacherSubjectAssignments(ctx context.Context, schoolID string, isPlatformOwner bool, classID, teacherID, sessionID string) ([]dto.TeacherSubjectAssignmentResponse, error) {
	q := s.db.WithContext(ctx).Where("is_active = ?", true)
	if !isPlatformOwner {
		q = q.Where("school_id = ?", schoolID)
	}
	if classID != "" {
		q = q.Where("class_id = ?", classID)
	}
	if teacherID != "" {
		q = q.Where("teacher_id = ?", teacherID)
	}
	if sessionID != "" {
		q = q.Where("session_id = ?", sessionID)
	}
	var rows []models.TeacherSubjectAssignment
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]dto.TeacherSubjectAssignmentResponse, len(rows))
	for i := range rows {
		resp, _ := s.toTeacherSubjectAssignmentResponse(ctx, &rows[i])
		out[i] = *resp
	}
	return out, nil
}

// DeleteTeacherSubjectAssignment deactivates one assignment.
func (s *ExamService) DeleteTeacherSubjectAssignment(ctx context.Context, schoolID string, isPlatformOwner bool, id string) error {
	q := s.db.WithContext(ctx).Model(&models.TeacherSubjectAssignment{}).Where("id = ?", id)
	if !isPlatformOwner {
		q = q.Where("school_id = ?", schoolID)
	}
	res := q.Update("is_active", false)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("assignment not found")
	}
	return nil
}

// GetMySubjectAssignments returns the authenticated teacher's own
// active assignments - the "my subjects" list the Student Score
// Register's class/subject picker is built from, mirroring
// classService.getClassesByTeacher's existing "my classes" pattern on
// the frontend but resolved at subject granularity.
func (s *ExamService) GetMySubjectAssignments(ctx context.Context, teacherID string, sessionID string) ([]dto.MySubjectAssignment, error) {
	q := s.db.WithContext(ctx).Where("teacher_id = ? AND is_active = ?", teacherID, true)
	if sessionID != "" {
		q = q.Where("session_id = ?", sessionID)
	}
	var rows []models.TeacherSubjectAssignment
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]dto.MySubjectAssignment, 0, len(rows))
	for _, a := range rows {
		item := dto.MySubjectAssignment{AssignmentID: a.ID, SubjectID: a.SubjectID, ClassID: a.ClassID, SessionID: a.SessionID}
		if subject, err := s.questionRepo.GetSubjectByID(ctx, a.SubjectID); err == nil {
			item.SubjectName, item.SubjectCode = subject.Name, subject.Code
		}
		if class, err := s.getClassByID(ctx, a.ClassID); err == nil {
			item.ClassName = models.GetClassDisplayName(class)
		}
		out = append(out, item)
	}
	return out, nil
}
