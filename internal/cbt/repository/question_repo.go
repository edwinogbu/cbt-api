package repository

import (
	"cbt-api/internal/models"
	"context"
	"errors"
	"fmt"
	"time"
	"strings"    
	"cbt-api/internal/cbt/dto" 


	"gorm.io/gorm"
)

// NO LOCAL generateID() function - we use models.GenerateID()

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// ============================================
// CRUD OPERATIONS
// ============================================

// func (r *QuestionRepository) Create(ctx context.Context, question *models.QuestionBank) error {
// 	if question == nil {
// 		return errors.New("question cannot be nil")
// 	}
// 	return r.db.WithContext(ctx).Create(question).Error
// }

// ============================================
// CRUD OPERATIONS
// ============================================

func (r *QuestionRepository) Create(ctx context.Context, question *models.QuestionBank) error {
	if question == nil {
		return errors.New("question cannot be nil")
	}

	// ============================================
	// VALIDATE ALL FOREIGN KEY IDs EXIST
	// ============================================
	
	// Validate School exists
	if question.SchoolID != "" {
		var school models.School
		if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", question.SchoolID).First(&school).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("school with ID %s not found", question.SchoolID)
			}
			return fmt.Errorf("failed to validate school: %w", err)
		}
	}

	// Validate Session exists
	if question.SessionID != "" {
		var session models.AcademicSession
		if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", question.SessionID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("session with ID %s not found", question.SessionID)
			}
			return fmt.Errorf("failed to validate session: %w", err)
		}
	}

	// Validate Term exists
	if question.TermID != "" {
		var term models.Term
		if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", question.TermID).First(&term).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("term with ID %s not found", question.TermID)
			}
			return fmt.Errorf("failed to validate term: %w", err)
		}
	}

	// Validate Class Level exists
	if question.ClassLevelID != "" {
		var classLevel models.ClassLevel
		if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", question.ClassLevelID).First(&classLevel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("class level with ID %s not found", question.ClassLevelID)
			}
			return fmt.Errorf("failed to validate class level: %w", err)
		}
	}

	// Validate Class exists
	if question.ClassID != "" {
		var class models.Class
		if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", question.ClassID).First(&class).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("class with ID %s not found", question.ClassID)
			}
			return fmt.Errorf("failed to validate class: %w", err)
		}
	}

	// Validate Subject exists
	if question.SubjectID != "" {
		var subject models.Subject
		if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", question.SubjectID).First(&subject).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("subject with ID %s not found", question.SubjectID)
			}
			return fmt.Errorf("failed to validate subject: %w", err)
		}
	}

	// Validate Exam Type
	if question.ExamType != "" {
		validExamTypes := map[string]bool{
			"weekly_test": true,
			"mid_term":    true,
			"main_exam":   true,
			"practice":    true,
		}
		if !validExamTypes[question.ExamType] {
			return fmt.Errorf("invalid exam_type: %s (must be one of: weekly_test, mid_term, main_exam, practice)", question.ExamType)
		}
	}

	// ============================================
	// CREATE THE QUESTION
	// ============================================
	
	return r.db.WithContext(ctx).Create(question).Error
}

func (r *QuestionRepository) FindByID(ctx context.Context, id string) (*models.QuestionBank, error) {
	var q models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&q).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("question with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to find question: %w", err)
	}
	return &q, nil
}

func (r *QuestionRepository) Update(ctx context.Context, question *models.QuestionBank) error {
	if question == nil {
		return errors.New("question cannot be nil")
	}
	return r.db.WithContext(ctx).Save(question).Error
}

func (r *QuestionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionTagMapping{}).Error; err != nil {
			return fmt.Errorf("failed to delete question tag mappings: %w", err)
		}
		if err := tx.Where("question_id = ?", id).Delete(&models.ExamQuestion{}).Error; err != nil {
			return fmt.Errorf("failed to delete exam questions: %w", err)
		}
		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
			return fmt.Errorf("failed to delete question attachments: %w", err)
		}
		if err := tx.Where("id = ?", id).Delete(&models.QuestionBank{}).Error; err != nil {
			return fmt.Errorf("failed to delete question: %w", err)
		}
		return nil
	})
}

// ============================================
// QUERY / FILTER OPERATIONS
// ============================================

func (r *QuestionRepository) ListBySubject(ctx context.Context, subjectID string, page, limit int) ([]models.QuestionBank, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	var questions []models.QuestionBank
	var total int64

	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Where("subject_id = ? AND deleted_at IS NULL", subjectID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count questions: %w", err)
	}

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list questions: %w", err)
	}

	return questions, total, nil
}

func (r *QuestionRepository) Filter(ctx context.Context, params map[string]interface{}, page, limit int) ([]models.QuestionBank, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var questions []models.QuestionBank
	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}
	if schoolID, ok := params["school_id"]; ok && schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	if sessionID, ok := params["session_id"]; ok && sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if termID, ok := params["term_id"]; ok && termID != "" {
		query = query.Where("term_id = ?", termID)
	}
	if classLevelID, ok := params["class_level_id"]; ok && classLevelID != "" {
		query = query.Where("class_level_id = ?", classLevelID)
	}
	if classID, ok := params["class_id"]; ok && classID != "" {
		query = query.Where("class_id = ?", classID)
	}
	if examType, ok := params["exam_type"]; ok && examType != "" {
		query = query.Where("exam_type = ?", examType)
	}

	if topic, ok := params["topic"]; ok && topic != "" {
		query = query.Where("topic ILIKE ?", "%"+topic.(string)+"%")
	}
	if difficulty, ok := params["difficulty"]; ok && difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}
	if bloomLevel, ok := params["bloom_level"]; ok && bloomLevel != "" {
		query = query.Where("bloom_level = ?", bloomLevel)
	}
	if questionType, ok := params["question_type"]; ok && questionType != "" {
		query = query.Where("question_type = ?", questionType)
	}
	if status, ok := params["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if curriculumType, ok := params["curriculum_type"]; ok && curriculumType != "" {
		query = query.Where("curriculum_type = ?", curriculumType)
	}
	if sourceType, ok := params["source_type"]; ok && sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}
	if externalID, ok := params["external_id"]; ok && externalID != "" {
		query = query.Where("external_id = ?", externalID)
	}

	if search, ok := params["search"]; ok && search != "" {
		searchStr := "%" + escapeWildcards(search.(string)) + "%"
		query = query.Where("question_text ILIKE ?", searchStr)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count filtered questions: %w", err)
	}

	offset := (page - 1) * limit
	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to filter questions: %w", err)
	}

	return questions, total, nil
}

func escapeWildcards(s string) string {
	result := ""
	for _, c := range s {
		if c == '%' || c == '_' {
			result += "\\" + string(c)
		} else {
			result += string(c)
		}
	}
	return result
}

func (r *QuestionRepository) FindByTag(ctx context.Context, tagName string, page, limit int) ([]models.QuestionBank, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var questions []models.QuestionBank
	query := r.db.WithContext(ctx).
		Joins("JOIN question_tag_mappings ON question_tag_mappings.question_id = question_bank.id").
		Joins("JOIN tags ON tags.id = question_tag_mappings.tag_id").
		Where("tags.name = ? AND question_bank.deleted_at IS NULL", tagName)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count questions by tag: %w", err)
	}

	offset := (page - 1) * limit
	err := query.
		Offset(offset).
		Limit(limit).
		Order("question_bank.created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find questions by tag: %w", err)
	}

	return questions, total, nil
}

func (r *QuestionRepository) FindByExternalID(ctx context.Context, schoolID, sessionID, termID, classID, externalID string) (*models.QuestionBank, error) {
	if externalID == "" {
		return nil, errors.New("external ID cannot be empty")
	}

	var q models.QuestionBank
	query := r.db.WithContext(ctx).
		Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)

	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if termID != "" {
		query = query.Where("term_id = ?", termID)
	}
	if classID != "" {
		query = query.Where("class_id = ?", classID)
	}

	err := query.First(&q).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find question by external ID: %w", err)
	}
	return &q, nil
}

// ============================================
// CONTEXT-AWARE QUERY METHODS
// ============================================

func (r *QuestionRepository) FindByTerm(ctx context.Context, subjectID, termID string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("subject_id = ? AND term_id = ? AND deleted_at IS NULL", subjectID, termID).
		Order("created_at DESC").
		Find(&questions).Error
	return questions, err
}

func (r *QuestionRepository) FindBySession(ctx context.Context, subjectID, sessionID string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("subject_id = ? AND session_id = ? AND deleted_at IS NULL", subjectID, sessionID).
		Order("created_at DESC").
		Find(&questions).Error
	return questions, err
}

func (r *QuestionRepository) FindByClass(ctx context.Context, classID string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("class_id = ? AND deleted_at IS NULL", classID).
		Order("created_at DESC").
		Find(&questions).Error
	return questions, err
}

func (r *QuestionRepository) FindByClassLevel(ctx context.Context, classLevelID string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("class_level_id = ? AND deleted_at IS NULL", classLevelID).
		Order("created_at DESC").
		Find(&questions).Error
	return questions, err
}

func (r *QuestionRepository) FindByExamType(ctx context.Context, examType string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("exam_type = ? AND deleted_at IS NULL", examType).
		Order("created_at DESC").
		Find(&questions).Error
	return questions, err
}

func (r *QuestionRepository) FindBySchoolAndSession(ctx context.Context, schoolID, sessionID string, page, limit int) ([]models.QuestionBank, int64, error) {
	var questions []models.QuestionBank
	var total int64

	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Where("school_id = ? AND session_id = ? AND deleted_at IS NULL", schoolID, sessionID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
	return questions, total, err
}

func (r *QuestionRepository) FindBySchoolAndTerm(ctx context.Context, schoolID, termID string, page, limit int) ([]models.QuestionBank, int64, error) {
	var questions []models.QuestionBank
	var total int64

	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Where("school_id = ? AND term_id = ? AND deleted_at IS NULL", schoolID, termID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
	return questions, total, err
}

func (r *QuestionRepository) GetQuestionsByClassAndExamType(ctx context.Context, classID string, examType string) ([]models.QuestionBank, error) {
	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("class_id = ? AND exam_type = ? AND deleted_at IS NULL", classID, examType).
		Order("created_at DESC").
		Find(&questions).Error
	return questions, err
}

func (r *QuestionRepository) CountQuestionsByTerm(ctx context.Context, termID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Where("term_id = ? AND deleted_at IS NULL", termID).
		Count(&count).Error
	return count, err
}

func (r *QuestionRepository) CountQuestionsByExamType(ctx context.Context, examType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Where("exam_type = ? AND deleted_at IS NULL", examType).
		Count(&count).Error
	return count, err
}

// ============================================
// SUMMARY & STATISTICS OPERATIONS
// ============================================

func (r *QuestionRepository) GetQuestionContextSummary(ctx context.Context, subjectID string) ([]QuestionTermSummary, error) {
	var results []QuestionTermSummary

	query := `
		SELECT 
			t.id as term_id,
			t.name as term_name,
			t.term_number,
			s.name as session_name,
			COUNT(q.id) as question_count,
			SUM(CASE WHEN q.exam_type = 'weekly_test' THEN 1 ELSE 0 END) as weekly_test_count,
			SUM(CASE WHEN q.exam_type = 'mid_term' THEN 1 ELSE 0 END) as mid_term_count,
			SUM(CASE WHEN q.exam_type = 'main_exam' THEN 1 ELSE 0 END) as main_exam_count,
			SUM(CASE WHEN q.exam_type = 'practice' THEN 1 ELSE 0 END) as practice_count
		FROM question_bank q
		JOIN terms t ON t.id = q.term_id
		JOIN academic_sessions s ON s.id = q.session_id
		WHERE q.subject_id = ? 
			AND q.deleted_at IS NULL
		GROUP BY t.id, t.name, t.term_number, s.name
		ORDER BY t.term_number ASC
	`

	err := r.db.WithContext(ctx).Raw(query, subjectID).Scan(&results).Error
	return results, err
}

func (r *QuestionRepository) GetStatisticsByExamType(ctx context.Context, subjectID string) (map[string]int64, error) {
	var results []struct {
		ExamType string
		Count    int64
	}

	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Select("exam_type, COUNT(*) as count").
		Where("deleted_at IS NULL")

	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}

	err := query.Group("exam_type").Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics by exam type: %w", err)
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.ExamType] = r.Count
	}

	examTypes := []string{"weekly_test", "mid_term", "main_exam", "practice"}
	for _, et := range examTypes {
		if _, ok := stats[et]; !ok {
			stats[et] = 0
		}
	}

	return stats, nil
}

// ============================================
// BULK / BATCH OPERATIONS
// ============================================

func (r *QuestionRepository) BulkCreate(ctx context.Context, questions []models.QuestionBank) error {
	if len(questions) == 0 {
		return errors.New("no questions to create")
	}
	return r.db.WithContext(ctx).CreateInBatches(questions, 100).Error
}

func (r *QuestionRepository) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errors.New("no question IDs provided")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionTagMapping{}).Error; err != nil {
			return fmt.Errorf("failed to delete question tag mappings: %w", err)
		}
		if err := tx.Where("question_id IN ?", ids).Delete(&models.ExamQuestion{}).Error; err != nil {
			return fmt.Errorf("failed to delete exam questions: %w", err)
		}
		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
			return fmt.Errorf("failed to delete question attachments: %w", err)
		}
		if err := tx.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error; err != nil {
			return fmt.Errorf("failed to delete questions: %w", err)
		}
		return nil
	})
}

func (r *QuestionRepository) BulkUpdateStatus(ctx context.Context, ids []string, status string) error {
	if len(ids) == 0 {
		return errors.New("no question IDs provided")
	}
	if status == "" {
		return errors.New("status cannot be empty")
	}

	result := r.db.WithContext(ctx).
		Model(&models.QuestionBank{}).
		Where("id IN ?", ids).
		Update("status", models.QuestionStatus(status))

	if result.Error != nil {
		return fmt.Errorf("failed to update status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("no questions found to update")
	}

	return nil
}

// ============================================
// VERSIONING OPERATIONS - USING models.GenerateID()
// ============================================

func (r *QuestionRepository) CreateNewVersion(ctx context.Context, old *models.QuestionBank, updates map[string]interface{}) (string, error) {
	if old == nil {
		return "", errors.New("old question cannot be nil")
	}
	if len(updates) == 0 {
		return "", errors.New("no updates provided")
	}

	newQ := *old
	newQ.ID = models.GenerateID() // ✅ USING models.GenerateID()
	newQ.Version = old.Version + 1
	newQ.ParentID = &old.ID
	newQ.CreatedAt = time.Now()
	newQ.UpdatedAt = time.Now()

	for k, v := range updates {
		if err := r.applyUpdate(&newQ, k, v); err != nil {
			return "", fmt.Errorf("failed to apply update for field %s: %w", k, err)
		}
	}

	err := r.db.WithContext(ctx).Create(&newQ).Error
	if err != nil {
		return "", fmt.Errorf("failed to create new version: %w", err)
	}

	return newQ.ID, nil
}

func (r *QuestionRepository) applyUpdate(q *models.QuestionBank, key string, value interface{}) error {
	switch key {
	case "topic":
		if v, ok := value.(string); ok {
			q.Topic = v
		}
	case "sub_topic":
		if v, ok := value.(string); ok {
			q.SubTopic = v
		}
	case "learning_objective":
		if v, ok := value.(string); ok {
			q.LearningObjective = v
		}
	case "question_text":
		if v, ok := value.(string); ok {
			q.QuestionText = v
		}
	case "question_type":
		if v, ok := value.(string); ok {
			q.QuestionType = models.QuestionType(v)
		}
	case "difficulty":
		if v, ok := value.(string); ok {
			q.Difficulty = models.DifficultyLevel(v)
		}
	case "bloom_level":
		if v, ok := value.(string); ok {
			q.BloomLevel = models.BloomTaxonomy(v)
		}
	case "exam_type":
		if v, ok := value.(string); ok {
			q.ExamType = v
		}
	case "school_id":
		if v, ok := value.(string); ok {
			q.SchoolID = v
		}
	case "session_id":
		if v, ok := value.(string); ok {
			q.SessionID = v
		}
	case "term_id":
		if v, ok := value.(string); ok {
			q.TermID = v
		}
	case "class_level_id":
		if v, ok := value.(string); ok {
			q.ClassLevelID = v
		}
	case "class_id":
		if v, ok := value.(string); ok {
			q.ClassID = v
		}
	case "subject_id":
		if v, ok := value.(string); ok {
			q.SubjectID = v
		}
	case "options":
		if v, ok := value.(models.OptionStorage); ok {
			q.Options = v
		} else {
			return fmt.Errorf("options must be of type OptionStorage, got %T", value)
		}
	case "correct_option_keys":
		if v, ok := value.([]string); ok {
			q.CorrectOptionKeys = v
		}
	case "correct_answer":
		if v, ok := value.(string); ok {
			q.CorrectAnswer = v
		}
	case "rubric":
		if v, ok := value.(models.RubricStorage); ok {
			q.Rubric = v
		} else {
			return fmt.Errorf("rubric must be of type RubricStorage, got %T", value)
		}
	case "tags":
		if v, ok := value.(models.TagStorage); ok {
			q.Tags = v
		} else {
			return fmt.Errorf("tags must be of type TagStorage, got %T", value)
		}
	case "explanation":
		if v, ok := value.(string); ok {
			q.Explanation = v
		}
	case "marks":
		if v, ok := value.(int); ok {
			q.Marks = v
		}
	case "negative_marks":
		if v, ok := value.(float64); ok {
			q.NegativeMarks = v
		}
	case "time_limit_seconds":
		if v, ok := value.(*int); ok {
			q.TimeLimitSeconds = v
		} else if v, ok := value.(int); ok {
			q.TimeLimitSeconds = &v
		}
	case "order":
		if v, ok := value.(int); ok {
			q.Order = v
		}
	case "is_required":
		if v, ok := value.(bool); ok {
			q.IsRequired = v
		}
	case "updated_by":
		if v, ok := value.(string); ok {
			q.UpdatedBy = v
		}
	case "curriculum_type":
		if v, ok := value.(string); ok {
			q.CurriculumType = v
		}
	case "source_type":
		if v, ok := value.(string); ok {
			q.SourceType = v
		}
	case "status":
		if v, ok := value.(string); ok {
			q.Status = models.QuestionStatus(v)
		}
	default:
		return fmt.Errorf("unknown field: %s", key)
	}
	return nil
}

// ============================================
// TAG OPERATIONS - USING models.GenerateID()
// ============================================

func (r *QuestionRepository) CreateTag(ctx context.Context, tag *models.Tag) error {
	if tag == nil {
		return errors.New("tag cannot be nil")
	}
	return r.db.WithContext(ctx).Create(tag).Error
}

func (r *QuestionRepository) FindTagByName(ctx context.Context, name string) (*models.Tag, error) {
	if name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	var tag models.Tag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find tag by name: %w", err)
	}
	return &tag, nil
}

func (r *QuestionRepository) ListTags(ctx context.Context) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("name ASC").
		Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

func (r *QuestionRepository) ListTagsPaginated(ctx context.Context, page, limit int) ([]models.Tag, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var tags []models.Tag
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Tag{}).Where("deleted_at IS NULL")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tags: %w", err)
	}

	offset := (page - 1) * limit
	err := query.
		Offset(offset).
		Limit(limit).
		Order("name ASC").
		Find(&tags).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, total, nil
}

func (r *QuestionRepository) AttachTags(ctx context.Context, questionID string, tagIDs []string) error {
	if questionID == "" {
		return errors.New("question ID cannot be empty")
	}
	if len(tagIDs) == 0 {
		return nil
	}

	for _, tid := range tagIDs {
		mapping := models.QuestionTagMapping{
			ID:         models.GenerateID(), // ✅ USING models.GenerateID()
			QuestionID: questionID,
			TagID:      tid,
		}
		if err := r.db.WithContext(ctx).Create(&mapping).Error; err != nil {
			return fmt.Errorf("failed to attach tag %s to question %s: %w", tid, questionID, err)
		}
	}
	return nil
}

func (r *QuestionRepository) DetachTags(ctx context.Context, questionID string, tagIDs []string) error {
	if questionID == "" {
		return errors.New("question ID cannot be empty")
	}
	if len(tagIDs) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Where("question_id = ? AND tag_id IN ?", questionID, tagIDs).
		Delete(&models.QuestionTagMapping{})

	if result.Error != nil {
		return fmt.Errorf("failed to detach tags: %w", result.Error)
	}
	return nil
}

// ============================================
// STATISTICS OPERATIONS
// ============================================

func (r *QuestionRepository) GetStatistics(ctx context.Context, subjectID string) (map[string]interface{}, error) {
	var total int64
	var published, draft, archived int64

	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total questions: %w", err)
	}

	if err := query.Where("status = ?", models.QuestionStatusPublished).Count(&published).Error; err != nil {
		return nil, fmt.Errorf("failed to count published questions: %w", err)
	}
	if err := query.Where("status = ?", models.QuestionStatusDraft).Count(&draft).Error; err != nil {
		return nil, fmt.Errorf("failed to count draft questions: %w", err)
	}
	if err := query.Where("status = ?", models.QuestionStatusArchived).Count(&archived).Error; err != nil {
		return nil, fmt.Errorf("failed to count archived questions: %w", err)
	}

	var avgMarks float64
	if err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Select("COALESCE(AVG(marks), 0)").
		Where("deleted_at IS NULL").
		Row().Scan(&avgMarks); err != nil {
		return nil, fmt.Errorf("failed to calculate average marks: %w", err)
	}

	examTypeStats, err := r.GetStatisticsByExamType(ctx, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exam type statistics: %w", err)
	}

	return map[string]interface{}{
		"total_questions": total,
		"published_count": published,
		"draft_count":     draft,
		"archived_count":  archived,
		"average_marks":   avgMarks,
		"by_exam_type":    examTypeStats,
	}, nil
}

func (r *QuestionRepository) GetDetailedStatistics(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

	for key, val := range filters {
		if val != nil && val != "" {
			query = query.Where(key+" = ?", val)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count questions: %w", err)
	}

	var published, draft, archived int64
	query.Where("status = ?", models.QuestionStatusPublished).Count(&published)
	query.Where("status = ?", models.QuestionStatusDraft).Count(&draft)
	query.Where("status = ?", models.QuestionStatusArchived).Count(&archived)

	var avgMarks float64
	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

	type DifficultyCount struct {
		Difficulty string
		Count      int
	}
	var difficultyCounts []DifficultyCount
	if err := query.Select("difficulty, count(*) as count").
		Group("difficulty").
		Scan(&difficultyCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get difficulty breakdown: %w", err)
	}

	byDifficulty := make(map[string]int)
	for _, dc := range difficultyCounts {
		byDifficulty[dc.Difficulty] = dc.Count
	}

	type TypeCount struct {
		QuestionType string
		Count        int
	}
	var typeCounts []TypeCount
	if err := query.Select("question_type, count(*) as count").
		Group("question_type").
		Scan(&typeCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get question type breakdown: %w", err)
	}

	byType := make(map[string]int)
	for _, tc := range typeCounts {
		byType[tc.QuestionType] = tc.Count
	}

	type ExamTypeCount struct {
		ExamType string
		Count    int
	}
	var examTypeCounts []ExamTypeCount
	if err := query.Select("exam_type, count(*) as count").
		Group("exam_type").
		Scan(&examTypeCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get exam type breakdown: %w", err)
	}

	byExamType := make(map[string]int)
	for _, etc := range examTypeCounts {
		byExamType[etc.ExamType] = etc.Count
	}

	return map[string]interface{}{
		"total":            total,
		"published":        published,
		"draft":            draft,
		"archived":         archived,
		"average_marks":    avgMarks,
		"by_difficulty":    byDifficulty,
		"by_question_type": byType,
		"by_exam_type":     byExamType,
	}, nil
}

// ============================================
// ADDITIONAL HELPER FUNCTIONS
// ============================================

func (r *QuestionRepository) FindByQuestionText(ctx context.Context, text string, limit int) ([]models.QuestionBank, error) {
	if text == "" {
		return nil, errors.New("search text cannot be empty")
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var questions []models.QuestionBank
	err := r.db.WithContext(ctx).
		Where("question_text ILIKE ? AND deleted_at IS NULL", "%"+escapeWildcards(text)+"%").
		Limit(limit).
		Order("created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to search questions: %w", err)
	}
	return questions, nil
}

func (r *QuestionRepository) CountQuestionsBySubject(ctx context.Context, subjectID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Where("subject_id = ? AND deleted_at IS NULL", subjectID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count questions by subject: %w", err)
	}
	return count, nil
}

func (r *QuestionRepository) GetQuestionsByStatus(ctx context.Context, status models.QuestionStatus, page, limit int) ([]models.QuestionBank, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var questions []models.QuestionBank
	var total int64

	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Where("status = ? AND deleted_at IS NULL", status)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count questions by status: %w", err)
	}

	offset := (page - 1) * limit
	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get questions by status: %w", err)
	}

	return questions, total, nil
}

func (r *QuestionRepository) GetRecentQuestions(ctx context.Context, subjectID string, limit int) ([]models.QuestionBank, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var questions []models.QuestionBank
	query := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit)

	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}

	err := query.Find(&questions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recent questions: %w", err)
	}
	return questions, nil
}




// GetSubjectByID gets a subject by ID
func (r *QuestionRepository) GetSubjectByID(ctx context.Context, id string) (*models.Subject, error) {
    var subject models.Subject
    err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&subject).Error
    if err != nil {
        return nil, err
    }
    return &subject, nil
}

// FindByIDWithContext finds a question by ID with context
func (r *QuestionRepository) FindByIDWithContext(ctx context.Context, id string) (*models.QuestionBank, error) {
    var question models.QuestionBank
    err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&question).Error
    if err != nil {
        return nil, err
    }
    return &question, nil
}

// ============================================
// FUNCTION TO GET DB INSTANCE
// ============================================

func (r *QuestionRepository) GetDB() *gorm.DB {
	return r.db
}

// ============================================
// SUMMARY STRUCTS
// ============================================

type QuestionTermSummary struct {
	TermID          string `json:"term_id"`
	TermName        string `json:"term_name"`
	TermNumber      int    `json:"term_number"`
	SessionName     string `json:"session_name"`
	QuestionCount   int    `json:"question_count"`
	WeeklyTestCount int    `json:"weekly_test_count"`
	MidTermCount    int    `json:"mid_term_count"`
	MainExamCount   int    `json:"main_exam_count"`
	PracticeCount   int    `json:"practice_count"`
}



// ============================================
// QUESTION VALIDATION METHODS - ADD TO question_repository.go
// ============================================

//  validates that questions are suitable for an exam
// Checks: exam_type, term_id, session_id, subject_id, class_id, and status
func (r *QuestionRepository) ValidateQuestionsForExam(ctx context.Context, exam *models.Exam, questionIDs []string) error {
    if len(questionIDs) == 0 {
        return errors.New("no questions selected")
    }

    // Get all questions in one batch query
    var questions []models.QuestionBank
    err := r.db.WithContext(ctx).
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
        if exam.ExamType != "practice" && q.Status != models.QuestionStatusPublished {
            validationErrors = append(validationErrors,
                fmt.Sprintf("Question %s is not published (status: %s)",
                    q.ID[:8], q.Status))
        }
    }

    if len(validationErrors) > 0 {
        return fmt.Errorf("question validation failed:\n%s", strings.Join(validationErrors, "\n"))
    }

    return nil
}

//  validates the number of questions for an exam type
func (r *QuestionRepository) ValidateQuestionCountForExam(examType string, count int) error {
    if count == 0 {
        return errors.New("no questions selected")
    }

    minQ, maxQ := dto.GetQuestionCountRange(examType)
    if count < minQ {
        return fmt.Errorf("exam requires at least %d questions, got %d (exam type: %s)", minQ, count, examType)
    }
    if count > maxQ {
        return fmt.Errorf("exam can have at most %d questions, got %d (exam type: %s)", maxQ, count, examType)
    }
    return nil
}

//  finds questions by multiple IDs
func (r *QuestionRepository) FindByIDs(ctx context.Context, ids []string) ([]models.QuestionBank, error) {
    if len(ids) == 0 {
        return []models.QuestionBank{}, nil
    }
    var questions []models.QuestionBank
    err := r.db.WithContext(ctx).
        Where("id IN ? AND deleted_at IS NULL", ids).
        Find(&questions).Error
    return questions, err
}


// package repository

// import (
// 	"cbt-api/internal/models"
// 	"context"
// 	"errors"
// 	"fmt"
// 	"time"

// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// // generateID creates a new UUID string
// func generateID() string {
// 	return uuid.New().String()
// }

// type QuestionRepository struct {
// 	db *gorm.DB
// }

// func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
// 	return &QuestionRepository{db: db}
// }

// // ============================================
// // CRUD OPERATIONS
// // ============================================

// func (r *QuestionRepository) Create(ctx context.Context, question *models.QuestionBank) error {
// 	if question == nil {
// 		return errors.New("question cannot be nil")
// 	}
// 	return r.db.WithContext(ctx).Create(question).Error
// }

// func (r *QuestionRepository) FindByID(ctx context.Context, id string) (*models.QuestionBank, error) {
// 	var q models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("id = ? AND deleted_at IS NULL", id).
// 		First(&q).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, fmt.Errorf("question with ID %s not found", id)
// 		}
// 		return nil, fmt.Errorf("failed to find question: %w", err)
// 	}
// 	return &q, nil
// }

// func (r *QuestionRepository) Update(ctx context.Context, question *models.QuestionBank) error {
// 	if question == nil {
// 		return errors.New("question cannot be nil")
// 	}
// 	return r.db.WithContext(ctx).Save(question).Error
// }

// func (r *QuestionRepository) Delete(ctx context.Context, id string) error {
// 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// 		}
// 		if err := tx.Where("question_id = ?", id).Delete(&models.ExamQuestion{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete exam questions: %w", err)
// 		}
// 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete question attachments: %w", err)
// 		}
// 		if err := tx.Where("id = ?", id).Delete(&models.QuestionBank{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete question: %w", err)
// 		}
// 		return nil
// 	})
// }

// // ============================================
// // QUERY / FILTER OPERATIONS
// // ============================================

// func (r *QuestionRepository) ListBySubject(ctx context.Context, subjectID string, page, limit int) ([]models.QuestionBank, int64, error) {
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 || limit > 100 {
// 		limit = 20
// 	}

// 	offset := (page - 1) * limit
// 	var questions []models.QuestionBank
// 	var total int64

// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Where("subject_id = ? AND deleted_at IS NULL", subjectID)

// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, fmt.Errorf("failed to count questions: %w", err)
// 	}

// 	err := query.
// 		Offset(offset).
// 		Limit(limit).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to list questions: %w", err)
// 	}

// 	return questions, total, nil
// }

// func (r *QuestionRepository) Filter(ctx context.Context, params map[string]interface{}, page, limit int) ([]models.QuestionBank, int64, error) {
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 || limit > 100 {
// 		limit = 20
// 	}

// 	var questions []models.QuestionBank
// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// 	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
// 		query = query.Where("subject_id = ?", subjectID)
// 	}
// 	if schoolID, ok := params["school_id"]; ok && schoolID != "" {
// 		query = query.Where("school_id = ?", schoolID)
// 	}
// 	if sessionID, ok := params["session_id"]; ok && sessionID != "" {
// 		query = query.Where("session_id = ?", sessionID)
// 	}
// 	if termID, ok := params["term_id"]; ok && termID != "" {
// 		query = query.Where("term_id = ?", termID)
// 	}
// 	if classLevelID, ok := params["class_level_id"]; ok && classLevelID != "" {
// 		query = query.Where("class_level_id = ?", classLevelID)
// 	}
// 	if classID, ok := params["class_id"]; ok && classID != "" {
// 		query = query.Where("class_id = ?", classID)
// 	}
// 	if examType, ok := params["exam_type"]; ok && examType != "" {
// 		query = query.Where("exam_type = ?", examType)
// 	}

// 	if topic, ok := params["topic"]; ok && topic != "" {
// 		query = query.Where("topic ILIKE ?", "%"+topic.(string)+"%")
// 	}
// 	if difficulty, ok := params["difficulty"]; ok && difficulty != "" {
// 		query = query.Where("difficulty = ?", difficulty)
// 	}
// 	if bloomLevel, ok := params["bloom_level"]; ok && bloomLevel != "" {
// 		query = query.Where("bloom_level = ?", bloomLevel)
// 	}
// 	if questionType, ok := params["question_type"]; ok && questionType != "" {
// 		query = query.Where("question_type = ?", questionType)
// 	}
// 	if status, ok := params["status"]; ok && status != "" {
// 		query = query.Where("status = ?", status)
// 	}
// 	if curriculumType, ok := params["curriculum_type"]; ok && curriculumType != "" {
// 		query = query.Where("curriculum_type = ?", curriculumType)
// 	}
// 	if sourceType, ok := params["source_type"]; ok && sourceType != "" {
// 		query = query.Where("source_type = ?", sourceType)
// 	}
// 	if externalID, ok := params["external_id"]; ok && externalID != "" {
// 		query = query.Where("external_id = ?", externalID)
// 	}

// 	if search, ok := params["search"]; ok && search != "" {
// 		searchStr := "%" + escapeWildcards(search.(string)) + "%"
// 		query = query.Where("question_text ILIKE ?", searchStr)
// 	}

// 	var total int64
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, fmt.Errorf("failed to count filtered questions: %w", err)
// 	}

// 	offset := (page - 1) * limit
// 	err := query.
// 		Offset(offset).
// 		Limit(limit).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to filter questions: %w", err)
// 	}

// 	return questions, total, nil
// }

// func escapeWildcards(s string) string {
// 	result := ""
// 	for _, c := range s {
// 		if c == '%' || c == '_' {
// 			result += "\\" + string(c)
// 		} else {
// 			result += string(c)
// 		}
// 	}
// 	return result
// }

// func (r *QuestionRepository) FindByTag(ctx context.Context, tagName string, page, limit int) ([]models.QuestionBank, int64, error) {
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 || limit > 100 {
// 		limit = 20
// 	}

// 	var questions []models.QuestionBank
// 	query := r.db.WithContext(ctx).
// 		Joins("JOIN question_tag_mappings ON question_tag_mappings.question_id = question_bank.id").
// 		Joins("JOIN tags ON tags.id = question_tag_mappings.tag_id").
// 		Where("tags.name = ? AND question_bank.deleted_at IS NULL", tagName)

// 	var total int64
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, fmt.Errorf("failed to count questions by tag: %w", err)
// 	}

// 	offset := (page - 1) * limit
// 	err := query.
// 		Offset(offset).
// 		Limit(limit).
// 		Order("question_bank.created_at DESC").
// 		Find(&questions).Error
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to find questions by tag: %w", err)
// 	}

// 	return questions, total, nil
// }

// func (r *QuestionRepository) FindByExternalID(ctx context.Context, schoolID, sessionID, termID, classID, externalID string) (*models.QuestionBank, error) {
// 	if externalID == "" {
// 		return nil, errors.New("external ID cannot be empty")
// 	}

// 	var q models.QuestionBank
// 	query := r.db.WithContext(ctx).
// 		Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)

// 	if sessionID != "" {
// 		query = query.Where("session_id = ?", sessionID)
// 	}
// 	if termID != "" {
// 		query = query.Where("term_id = ?", termID)
// 	}
// 	if classID != "" {
// 		query = query.Where("class_id = ?", classID)
// 	}

// 	err := query.First(&q).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, nil
// 		}
// 		return nil, fmt.Errorf("failed to find question by external ID: %w", err)
// 	}
// 	return &q, nil
// }

// // ============================================
// // CONTEXT-AWARE QUERY METHODS
// // ============================================

// func (r *QuestionRepository) FindByTerm(ctx context.Context, subjectID, termID string) ([]models.QuestionBank, error) {
// 	var questions []models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("subject_id = ? AND term_id = ? AND deleted_at IS NULL", subjectID, termID).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	return questions, err
// }

// func (r *QuestionRepository) FindBySession(ctx context.Context, subjectID, sessionID string) ([]models.QuestionBank, error) {
// 	var questions []models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("subject_id = ? AND session_id = ? AND deleted_at IS NULL", subjectID, sessionID).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	return questions, err
// }

// func (r *QuestionRepository) FindByClass(ctx context.Context, classID string) ([]models.QuestionBank, error) {
// 	var questions []models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("class_id = ? AND deleted_at IS NULL", classID).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	return questions, err
// }

// func (r *QuestionRepository) FindByClassLevel(ctx context.Context, classLevelID string) ([]models.QuestionBank, error) {
// 	var questions []models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("class_level_id = ? AND deleted_at IS NULL", classLevelID).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	return questions, err
// }

// func (r *QuestionRepository) FindByExamType(ctx context.Context, examType string) ([]models.QuestionBank, error) {
// 	var questions []models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("exam_type = ? AND deleted_at IS NULL", examType).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	return questions, err
// }

// func (r *QuestionRepository) FindBySchoolAndSession(ctx context.Context, schoolID, sessionID string, page, limit int) ([]models.QuestionBank, int64, error) {
// 	var questions []models.QuestionBank
// 	var total int64

// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Where("school_id = ? AND session_id = ? AND deleted_at IS NULL", schoolID, sessionID)

// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	offset := (page - 1) * limit
// 	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
// 	return questions, total, err
// }

// func (r *QuestionRepository) FindBySchoolAndTerm(ctx context.Context, schoolID, termID string, page, limit int) ([]models.QuestionBank, int64, error) {
// 	var questions []models.QuestionBank
// 	var total int64

// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Where("school_id = ? AND term_id = ? AND deleted_at IS NULL", schoolID, termID)

// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	offset := (page - 1) * limit
// 	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
// 	return questions, total, err
// }

// func (r *QuestionRepository) GetQuestionsByClassAndExamType(ctx context.Context, classID string, examType string) ([]models.QuestionBank, error) {
// 	var questions []models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("class_id = ? AND exam_type = ? AND deleted_at IS NULL", classID, examType).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	return questions, err
// }

// func (r *QuestionRepository) CountQuestionsByTerm(ctx context.Context, termID string) (int64, error) {
// 	var count int64
// 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Where("term_id = ? AND deleted_at IS NULL", termID).
// 		Count(&count).Error
// 	return count, err
// }

// func (r *QuestionRepository) CountQuestionsByExamType(ctx context.Context, examType string) (int64, error) {
// 	var count int64
// 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Where("exam_type = ? AND deleted_at IS NULL", examType).
// 		Count(&count).Error
// 	return count, err
// }

// // ============================================
// // SUMMARY & STATISTICS OPERATIONS
// // ============================================

// func (r *QuestionRepository) GetQuestionContextSummary(ctx context.Context, subjectID string) ([]QuestionTermSummary, error) {
// 	var results []QuestionTermSummary

// 	query := `
// 		SELECT 
// 			t.id as term_id,
// 			t.name as term_name,
// 			t.term_number,
// 			s.name as session_name,
// 			COUNT(q.id) as question_count,
// 			SUM(CASE WHEN q.exam_type = 'weekly_test' THEN 1 ELSE 0 END) as weekly_test_count,
// 			SUM(CASE WHEN q.exam_type = 'mid_term' THEN 1 ELSE 0 END) as mid_term_count,
// 			SUM(CASE WHEN q.exam_type = 'main_exam' THEN 1 ELSE 0 END) as main_exam_count,
// 			SUM(CASE WHEN q.exam_type = 'practice' THEN 1 ELSE 0 END) as practice_count
// 		FROM question_bank q
// 		JOIN terms t ON t.id = q.term_id
// 		JOIN academic_sessions s ON s.id = q.session_id
// 		WHERE q.subject_id = ? 
// 			AND q.deleted_at IS NULL
// 		GROUP BY t.id, t.name, t.term_number, s.name
// 		ORDER BY t.term_number ASC
// 	`

// 	err := r.db.WithContext(ctx).Raw(query, subjectID).Scan(&results).Error
// 	return results, err
// }

// func (r *QuestionRepository) GetStatisticsByExamType(ctx context.Context, subjectID string) (map[string]int64, error) {
// 	var results []struct {
// 		ExamType string
// 		Count    int64
// 	}

// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Select("exam_type, COUNT(*) as count").
// 		Where("deleted_at IS NULL")

// 	if subjectID != "" {
// 		query = query.Where("subject_id = ?", subjectID)
// 	}

// 	err := query.Group("exam_type").Scan(&results).Error
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get statistics by exam type: %w", err)
// 	}

// 	stats := make(map[string]int64)
// 	for _, r := range results {
// 		stats[r.ExamType] = r.Count
// 	}

// 	examTypes := []string{"weekly_test", "mid_term", "main_exam", "practice"}
// 	for _, et := range examTypes {
// 		if _, ok := stats[et]; !ok {
// 			stats[et] = 0
// 		}
// 	}

// 	return stats, nil
// }

// // ============================================
// // BULK / BATCH OPERATIONS
// // ============================================

// func (r *QuestionRepository) BulkCreate(ctx context.Context, questions []models.QuestionBank) error {
// 	if len(questions) == 0 {
// 		return errors.New("no questions to create")
// 	}
// 	return r.db.WithContext(ctx).CreateInBatches(questions, 100).Error
// }

// func (r *QuestionRepository) BulkDelete(ctx context.Context, ids []string) error {
// 	if len(ids) == 0 {
// 		return errors.New("no question IDs provided")
// 	}

// 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// 		}
// 		if err := tx.Where("question_id IN ?", ids).Delete(&models.ExamQuestion{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete exam questions: %w", err)
// 		}
// 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete question attachments: %w", err)
// 		}
// 		if err := tx.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error; err != nil {
// 			return fmt.Errorf("failed to delete questions: %w", err)
// 		}
// 		return nil
// 	})
// }

// func (r *QuestionRepository) BulkUpdateStatus(ctx context.Context, ids []string, status string) error {
// 	if len(ids) == 0 {
// 		return errors.New("no question IDs provided")
// 	}
// 	if status == "" {
// 		return errors.New("status cannot be empty")
// 	}

// 	result := r.db.WithContext(ctx).
// 		Model(&models.QuestionBank{}).
// 		Where("id IN ?", ids).
// 		Update("status", models.QuestionStatus(status))

// 	if result.Error != nil {
// 		return fmt.Errorf("failed to update status: %w", result.Error)
// 	}

// 	if result.RowsAffected == 0 {
// 		return errors.New("no questions found to update")
// 	}

// 	return nil
// }

// // ============================================
// // VERSIONING OPERATIONS
// // ============================================

// func (r *QuestionRepository) CreateNewVersion(ctx context.Context, old *models.QuestionBank, updates map[string]interface{}) (string, error) {
// 	if old == nil {
// 		return "", errors.New("old question cannot be nil")
// 	}
// 	if len(updates) == 0 {
// 		return "", errors.New("no updates provided")
// 	}

// 	newQ := *old
// 	newQ.ID = generateID()
// 	newQ.Version = old.Version + 1
// 	newQ.ParentID = &old.ID
// 	newQ.CreatedAt = time.Now()
// 	newQ.UpdatedAt = time.Now()

// 	for k, v := range updates {
// 		if err := r.applyUpdate(&newQ, k, v); err != nil {
// 			return "", fmt.Errorf("failed to apply update for field %s: %w", k, err)
// 		}
// 	}

// 	err := r.db.WithContext(ctx).Create(&newQ).Error
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create new version: %w", err)
// 	}

// 	return newQ.ID, nil
// }

// func (r *QuestionRepository) applyUpdate(q *models.QuestionBank, key string, value interface{}) error {
// 	switch key {
// 	case "topic":
// 		if v, ok := value.(string); ok {
// 			q.Topic = v
// 		}
// 	case "sub_topic":
// 		if v, ok := value.(string); ok {
// 			q.SubTopic = v
// 		}
// 	case "learning_objective":
// 		if v, ok := value.(string); ok {
// 			q.LearningObjective = v
// 		}
// 	case "question_text":
// 		if v, ok := value.(string); ok {
// 			q.QuestionText = v
// 		}
// 	case "question_type":
// 		if v, ok := value.(string); ok {
// 			q.QuestionType = models.QuestionType(v)
// 		}
// 	case "difficulty":
// 		if v, ok := value.(string); ok {
// 			q.Difficulty = models.DifficultyLevel(v)
// 		}
// 	case "bloom_level":
// 		if v, ok := value.(string); ok {
// 			q.BloomLevel = models.BloomTaxonomy(v)
// 		}
// 	case "exam_type":
// 		if v, ok := value.(string); ok {
// 			q.ExamType = v
// 		}
// 	case "school_id":
// 		if v, ok := value.(string); ok {
// 			q.SchoolID = v
// 		}
// 	case "session_id":
// 		if v, ok := value.(string); ok {
// 			q.SessionID = v
// 		}
// 	case "term_id":
// 		if v, ok := value.(string); ok {
// 			q.TermID = v
// 		}
// 	case "class_level_id":
// 		if v, ok := value.(string); ok {
// 			q.ClassLevelID = v
// 		}
// 	case "class_id":
// 		if v, ok := value.(string); ok {
// 			q.ClassID = v
// 		}
// 	case "subject_id":
// 		if v, ok := value.(string); ok {
// 			q.SubjectID = v
// 		}
// 	case "options":
// 		if v, ok := value.(models.OptionStorage); ok {
// 			q.Options = v
// 		} else {
// 			return fmt.Errorf("options must be of type OptionStorage, got %T", value)
// 		}
// 	case "correct_option_keys":
// 		if v, ok := value.([]string); ok {
// 			q.CorrectOptionKeys = v
// 		}
// 	case "correct_answer":
// 		if v, ok := value.(string); ok {
// 			q.CorrectAnswer = v
// 		}
// 	case "rubric":
// 		if v, ok := value.(models.RubricStorage); ok {
// 			q.Rubric = v
// 		} else {
// 			return fmt.Errorf("rubric must be of type RubricStorage, got %T", value)
// 		}
// 	case "tags":
// 		if v, ok := value.(models.TagStorage); ok {
// 			q.Tags = v
// 		} else {
// 			return fmt.Errorf("tags must be of type TagStorage, got %T", value)
// 		}
// 	case "explanation":
// 		if v, ok := value.(string); ok {
// 			q.Explanation = v
// 		}
// 	case "marks":
// 		if v, ok := value.(int); ok {
// 			q.Marks = v
// 		}
// 	case "negative_marks":
// 		if v, ok := value.(float64); ok {
// 			q.NegativeMarks = v
// 		}
// 	case "time_limit_seconds":
// 		if v, ok := value.(*int); ok {
// 			q.TimeLimitSeconds = v
// 		} else if v, ok := value.(int); ok {
// 			q.TimeLimitSeconds = &v
// 		}
// 	case "order":
// 		if v, ok := value.(int); ok {
// 			q.Order = v
// 		}
// 	case "is_required":
// 		if v, ok := value.(bool); ok {
// 			q.IsRequired = v
// 		}
// 	case "updated_by":
// 		if v, ok := value.(string); ok {
// 			q.UpdatedBy = v
// 		}
// 	case "curriculum_type":
// 		if v, ok := value.(string); ok {
// 			q.CurriculumType = v
// 		}
// 	case "source_type":
// 		if v, ok := value.(string); ok {
// 			q.SourceType = v
// 		}
// 	case "status":
// 		if v, ok := value.(string); ok {
// 			q.Status = models.QuestionStatus(v)
// 		}
// 	default:
// 		return fmt.Errorf("unknown field: %s", key)
// 	}
// 	return nil
// }

// // ============================================
// // TAG OPERATIONS
// // ============================================

// func (r *QuestionRepository) CreateTag(ctx context.Context, tag *models.Tag) error {
// 	if tag == nil {
// 		return errors.New("tag cannot be nil")
// 	}
// 	return r.db.WithContext(ctx).Create(tag).Error
// }

// func (r *QuestionRepository) FindTagByName(ctx context.Context, name string) (*models.Tag, error) {
// 	if name == "" {
// 		return nil, errors.New("tag name cannot be empty")
// 	}

// 	var tag models.Tag
// 	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, nil
// 		}
// 		return nil, fmt.Errorf("failed to find tag by name: %w", err)
// 	}
// 	return &tag, nil
// }

// func (r *QuestionRepository) ListTags(ctx context.Context) ([]models.Tag, error) {
// 	var tags []models.Tag
// 	err := r.db.WithContext(ctx).
// 		Where("deleted_at IS NULL").
// 		Order("name ASC").
// 		Find(&tags).Error
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to list tags: %w", err)
// 	}
// 	return tags, nil
// }

// func (r *QuestionRepository) ListTagsPaginated(ctx context.Context, page, limit int) ([]models.Tag, int64, error) {
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 || limit > 100 {
// 		limit = 20
// 	}

// 	var tags []models.Tag
// 	var total int64

// 	query := r.db.WithContext(ctx).Model(&models.Tag{}).Where("deleted_at IS NULL")

// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, fmt.Errorf("failed to count tags: %w", err)
// 	}

// 	offset := (page - 1) * limit
// 	err := query.
// 		Offset(offset).
// 		Limit(limit).
// 		Order("name ASC").
// 		Find(&tags).Error
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
// 	}

// 	return tags, total, nil
// }

// func (r *QuestionRepository) AttachTags(ctx context.Context, questionID string, tagIDs []string) error {
// 	if questionID == "" {
// 		return errors.New("question ID cannot be empty")
// 	}
// 	if len(tagIDs) == 0 {
// 		return nil
// 	}

// 	for _, tid := range tagIDs {
// 		mapping := models.QuestionTagMapping{
// 			ID:         generateID(),
// 			QuestionID: questionID,
// 			TagID:      tid,
// 		}
// 		if err := r.db.WithContext(ctx).Create(&mapping).Error; err != nil {
// 			return fmt.Errorf("failed to attach tag %s to question %s: %w", tid, questionID, err)
// 		}
// 	}
// 	return nil
// }

// func (r *QuestionRepository) DetachTags(ctx context.Context, questionID string, tagIDs []string) error {
// 	if questionID == "" {
// 		return errors.New("question ID cannot be empty")
// 	}
// 	if len(tagIDs) == 0 {
// 		return nil
// 	}

// 	result := r.db.WithContext(ctx).
// 		Where("question_id = ? AND tag_id IN ?", questionID, tagIDs).
// 		Delete(&models.QuestionTagMapping{})

// 	if result.Error != nil {
// 		return fmt.Errorf("failed to detach tags: %w", result.Error)
// 	}
// 	return nil
// }

// // ============================================
// // STATISTICS OPERATIONS
// // ============================================

// func (r *QuestionRepository) GetStatistics(ctx context.Context, subjectID string) (map[string]interface{}, error) {
// 	var total int64
// 	var published, draft, archived int64

// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
// 	if subjectID != "" {
// 		query = query.Where("subject_id = ?", subjectID)
// 	}

// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, fmt.Errorf("failed to count total questions: %w", err)
// 	}

// 	if err := query.Where("status = ?", models.QuestionStatusPublished).Count(&published).Error; err != nil {
// 		return nil, fmt.Errorf("failed to count published questions: %w", err)
// 	}
// 	if err := query.Where("status = ?", models.QuestionStatusDraft).Count(&draft).Error; err != nil {
// 		return nil, fmt.Errorf("failed to count draft questions: %w", err)
// 	}
// 	if err := query.Where("status = ?", models.QuestionStatusArchived).Count(&archived).Error; err != nil {
// 		return nil, fmt.Errorf("failed to count archived questions: %w", err)
// 	}

// 	var avgMarks float64
// 	if err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Select("COALESCE(AVG(marks), 0)").
// 		Where("deleted_at IS NULL").
// 		Row().Scan(&avgMarks); err != nil {
// 		return nil, fmt.Errorf("failed to calculate average marks: %w", err)
// 	}

// 	examTypeStats, err := r.GetStatisticsByExamType(ctx, subjectID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get exam type statistics: %w", err)
// 	}

// 	return map[string]interface{}{
// 		"total_questions": total,
// 		"published_count": published,
// 		"draft_count":     draft,
// 		"archived_count":  archived,
// 		"average_marks":   avgMarks,
// 		"by_exam_type":    examTypeStats,
// 	}, nil
// }

// func (r *QuestionRepository) GetDetailedStatistics(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// 	for key, val := range filters {
// 		if val != nil && val != "" {
// 			query = query.Where(key+" = ?", val)
// 		}
// 	}

// 	var total int64
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, fmt.Errorf("failed to count questions: %w", err)
// 	}

// 	var published, draft, archived int64
// 	query.Where("status = ?", models.QuestionStatusPublished).Count(&published)
// 	query.Where("status = ?", models.QuestionStatusDraft).Count(&draft)
// 	query.Where("status = ?", models.QuestionStatusArchived).Count(&archived)

// 	var avgMarks float64
// 	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

// 	type DifficultyCount struct {
// 		Difficulty string
// 		Count      int
// 	}
// 	var difficultyCounts []DifficultyCount
// 	if err := query.Select("difficulty, count(*) as count").
// 		Group("difficulty").
// 		Scan(&difficultyCounts).Error; err != nil {
// 		return nil, fmt.Errorf("failed to get difficulty breakdown: %w", err)
// 	}

// 	byDifficulty := make(map[string]int)
// 	for _, dc := range difficultyCounts {
// 		byDifficulty[dc.Difficulty] = dc.Count
// 	}

// 	type TypeCount struct {
// 		QuestionType string
// 		Count        int
// 	}
// 	var typeCounts []TypeCount
// 	if err := query.Select("question_type, count(*) as count").
// 		Group("question_type").
// 		Scan(&typeCounts).Error; err != nil {
// 		return nil, fmt.Errorf("failed to get question type breakdown: %w", err)
// 	}

// 	byType := make(map[string]int)
// 	for _, tc := range typeCounts {
// 		byType[tc.QuestionType] = tc.Count
// 	}

// 	type ExamTypeCount struct {
// 		ExamType string
// 		Count    int
// 	}
// 	var examTypeCounts []ExamTypeCount
// 	if err := query.Select("exam_type, count(*) as count").
// 		Group("exam_type").
// 		Scan(&examTypeCounts).Error; err != nil {
// 		return nil, fmt.Errorf("failed to get exam type breakdown: %w", err)
// 	}

// 	byExamType := make(map[string]int)
// 	for _, etc := range examTypeCounts {
// 		byExamType[etc.ExamType] = etc.Count
// 	}

// 	return map[string]interface{}{
// 		"total":            total,
// 		"published":        published,
// 		"draft":            draft,
// 		"archived":         archived,
// 		"average_marks":    avgMarks,
// 		"by_difficulty":    byDifficulty,
// 		"by_question_type": byType,
// 		"by_exam_type":     byExamType,
// 	}, nil
// }

// // ============================================
// // ADDITIONAL HELPER FUNCTIONS
// // ============================================

// func (r *QuestionRepository) FindByQuestionText(ctx context.Context, text string, limit int) ([]models.QuestionBank, error) {
// 	if text == "" {
// 		return nil, errors.New("search text cannot be empty")
// 	}
// 	if limit < 1 || limit > 100 {
// 		limit = 20
// 	}

// 	var questions []models.QuestionBank
// 	err := r.db.WithContext(ctx).
// 		Where("question_text ILIKE ? AND deleted_at IS NULL", "%"+escapeWildcards(text)+"%").
// 		Limit(limit).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to search questions: %w", err)
// 	}
// 	return questions, nil
// }

// func (r *QuestionRepository) CountQuestionsBySubject(ctx context.Context, subjectID string) (int64, error) {
// 	var count int64
// 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Where("subject_id = ? AND deleted_at IS NULL", subjectID).
// 		Count(&count).Error
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to count questions by subject: %w", err)
// 	}
// 	return count, nil
// }

// func (r *QuestionRepository) GetQuestionsByStatus(ctx context.Context, status models.QuestionStatus, page, limit int) ([]models.QuestionBank, int64, error) {
// 	if page < 1 {
// 		page = 1
// 	}
// 	if limit < 1 || limit > 100 {
// 		limit = 20
// 	}

// 	var questions []models.QuestionBank
// 	var total int64

// 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// 		Where("status = ? AND deleted_at IS NULL", status)

// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, fmt.Errorf("failed to count questions by status: %w", err)
// 	}

// 	offset := (page - 1) * limit
// 	err := query.
// 		Offset(offset).
// 		Limit(limit).
// 		Order("created_at DESC").
// 		Find(&questions).Error
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to get questions by status: %w", err)
// 	}

// 	return questions, total, nil
// }

// func (r *QuestionRepository) GetRecentQuestions(ctx context.Context, subjectID string, limit int) ([]models.QuestionBank, error) {
// 	if limit < 1 || limit > 50 {
// 		limit = 10
// 	}

// 	var questions []models.QuestionBank
// 	query := r.db.WithContext(ctx).
// 		Where("deleted_at IS NULL").
// 		Order("created_at DESC").
// 		Limit(limit)

// 	if subjectID != "" {
// 		query = query.Where("subject_id = ?", subjectID)
// 	}

// 	err := query.Find(&questions).Error
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get recent questions: %w", err)
// 	}
// 	return questions, nil
// }

// // ============================================
// // FUNCTION TO GET DB INSTANCE
// // ============================================

// func (r *QuestionRepository) GetDB() *gorm.DB {
// 	return r.db
// }

// // ============================================
// // SUMMARY STRUCTS
// // ============================================

// type QuestionTermSummary struct {
// 	TermID          string `json:"term_id"`
// 	TermName        string `json:"term_name"`
// 	TermNumber      int    `json:"term_number"`
// 	SessionName     string `json:"session_name"`
// 	QuestionCount   int    `json:"question_count"`
// 	WeeklyTestCount int    `json:"weekly_test_count"`
// 	MidTermCount    int    `json:"mid_term_count"`
// 	MainExamCount   int    `json:"main_exam_count"`
// 	PracticeCount   int    `json:"practice_count"`
// }



// // package repository

// // import (
// // 	"cbt-api/internal/models"
// // 	"context"
// // 	"errors"
// // 	"fmt"
// // 	"time"
// // 	// "github.com/google/uuid"  
// // 	"gorm.io/gorm"
// // )

// // type QuestionRepository struct {
// // 	db *gorm.DB
// // }

// // func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
// // 	return &QuestionRepository{db: db}
// // }

// // // ============================================
// // // CRUD OPERATIONS
// // // ============================================

// // func (r *QuestionRepository) Create(ctx context.Context, question *models.QuestionBank) error {
// // 	if question == nil {
// // 		return errors.New("question cannot be nil")
// // 	}
// // 	return r.db.WithContext(ctx).Create(question).Error
// // }

// // func (r *QuestionRepository) FindByID(ctx context.Context, id string) (*models.QuestionBank, error) {
// // 	var q models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("id = ? AND deleted_at IS NULL", id).
// // 		First(&q).Error
// // 	if err != nil {
// // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // 			return nil, fmt.Errorf("question with ID %s not found", id)
// // 		}
// // 		return nil, fmt.Errorf("failed to find question: %w", err)
// // 	}
// // 	return &q, nil
// // }

// // func (r *QuestionRepository) Update(ctx context.Context, question *models.QuestionBank) error {
// // 	if question == nil {
// // 		return errors.New("question cannot be nil")
// // 	}
// // 	return r.db.WithContext(ctx).Save(question).Error
// // }

// // func (r *QuestionRepository) Delete(ctx context.Context, id string) error {
// // 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// // 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// // 		}
// // 		if err := tx.Where("question_id = ?", id).Delete(&models.ExamQuestion{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete exam questions: %w", err)
// // 		}
// // 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete question attachments: %w", err)
// // 		}
// // 		if err := tx.Where("id = ?", id).Delete(&models.QuestionBank{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete question: %w", err)
// // 		}
// // 		return nil
// // 	})
// // }

// // // ============================================
// // // QUERY / FILTER OPERATIONS
// // // ============================================

// // func (r *QuestionRepository) ListBySubject(ctx context.Context, subjectID string, page, limit int) ([]models.QuestionBank, int64, error) {
// // 	if page < 1 {
// // 		page = 1
// // 	}
// // 	if limit < 1 || limit > 100 {
// // 		limit = 20
// // 	}

// // 	offset := (page - 1) * limit
// // 	var questions []models.QuestionBank
// // 	var total int64

// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Where("subject_id = ? AND deleted_at IS NULL", subjectID)

// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, fmt.Errorf("failed to count questions: %w", err)
// // 	}

// // 	err := query.
// // 		Offset(offset).
// // 		Limit(limit).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	if err != nil {
// // 		return nil, 0, fmt.Errorf("failed to list questions: %w", err)
// // 	}

// // 	return questions, total, nil
// // }

// // func (r *QuestionRepository) Filter(ctx context.Context, params map[string]interface{}, page, limit int) ([]models.QuestionBank, int64, error) {
// // 	if page < 1 {
// // 		page = 1
// // 	}
// // 	if limit < 1 || limit > 100 {
// // 		limit = 20
// // 	}

// // 	var questions []models.QuestionBank
// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// // 	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
// // 		query = query.Where("subject_id = ?", subjectID)
// // 	}
// // 	if schoolID, ok := params["school_id"]; ok && schoolID != "" {
// // 		query = query.Where("school_id = ?", schoolID)
// // 	}
// // 	if sessionID, ok := params["session_id"]; ok && sessionID != "" {
// // 		query = query.Where("session_id = ?", sessionID)
// // 	}
// // 	if termID, ok := params["term_id"]; ok && termID != "" {
// // 		query = query.Where("term_id = ?", termID)
// // 	}
// // 	if classLevelID, ok := params["class_level_id"]; ok && classLevelID != "" {
// // 		query = query.Where("class_level_id = ?", classLevelID)
// // 	}
// // 	if classID, ok := params["class_id"]; ok && classID != "" {
// // 		query = query.Where("class_id = ?", classID)
// // 	}
// // 	if examType, ok := params["exam_type"]; ok && examType != "" {
// // 		query = query.Where("exam_type = ?", examType)
// // 	}

// // 	if topic, ok := params["topic"]; ok && topic != "" {
// // 		query = query.Where("topic ILIKE ?", "%"+topic.(string)+"%")
// // 	}
// // 	if difficulty, ok := params["difficulty"]; ok && difficulty != "" {
// // 		query = query.Where("difficulty = ?", difficulty)
// // 	}
// // 	if bloomLevel, ok := params["bloom_level"]; ok && bloomLevel != "" {
// // 		query = query.Where("bloom_level = ?", bloomLevel)
// // 	}
// // 	if questionType, ok := params["question_type"]; ok && questionType != "" {
// // 		query = query.Where("question_type = ?", questionType)
// // 	}
// // 	if status, ok := params["status"]; ok && status != "" {
// // 		query = query.Where("status = ?", status)
// // 	}
// // 	if curriculumType, ok := params["curriculum_type"]; ok && curriculumType != "" {
// // 		query = query.Where("curriculum_type = ?", curriculumType)
// // 	}
// // 	if sourceType, ok := params["source_type"]; ok && sourceType != "" {
// // 		query = query.Where("source_type = ?", sourceType)
// // 	}
// // 	if externalID, ok := params["external_id"]; ok && externalID != "" {
// // 		query = query.Where("external_id = ?", externalID)
// // 	}

// // 	if search, ok := params["search"]; ok && search != "" {
// // 		searchStr := "%" + escapeWildcards(search.(string)) + "%"
// // 		query = query.Where("question_text ILIKE ?", searchStr)
// // 	}

// // 	var total int64
// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, fmt.Errorf("failed to count filtered questions: %w", err)
// // 	}

// // 	offset := (page - 1) * limit
// // 	err := query.
// // 		Offset(offset).
// // 		Limit(limit).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	if err != nil {
// // 		return nil, 0, fmt.Errorf("failed to filter questions: %w", err)
// // 	}

// // 	return questions, total, nil
// // }

// // func escapeWildcards(s string) string {
// // 	result := ""
// // 	for _, c := range s {
// // 		if c == '%' || c == '_' {
// // 			result += "\\" + string(c)
// // 		} else {
// // 			result += string(c)
// // 		}
// // 	}
// // 	return result
// // }

// // func (r *QuestionRepository) FindByTag(ctx context.Context, tagName string, page, limit int) ([]models.QuestionBank, int64, error) {
// // 	if page < 1 {
// // 		page = 1
// // 	}
// // 	if limit < 1 || limit > 100 {
// // 		limit = 20
// // 	}

// // 	var questions []models.QuestionBank
// // 	query := r.db.WithContext(ctx).
// // 		Joins("JOIN question_tag_mappings ON question_tag_mappings.question_id = question_bank.id").
// // 		Joins("JOIN tags ON tags.id = question_tag_mappings.tag_id").
// // 		Where("tags.name = ? AND question_bank.deleted_at IS NULL", tagName)

// // 	var total int64
// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, fmt.Errorf("failed to count questions by tag: %w", err)
// // 	}

// // 	offset := (page - 1) * limit
// // 	err := query.
// // 		Offset(offset).
// // 		Limit(limit).
// // 		Order("question_bank.created_at DESC").
// // 		Find(&questions).Error
// // 	if err != nil {
// // 		return nil, 0, fmt.Errorf("failed to find questions by tag: %w", err)
// // 	}

// // 	return questions, total, nil
// // }

// // func (r *QuestionRepository) FindByExternalID(ctx context.Context, schoolID, sessionID, termID, classID, externalID string) (*models.QuestionBank, error) {
// // 	if externalID == "" {
// // 		return nil, errors.New("external ID cannot be empty")
// // 	}

// // 	var q models.QuestionBank
// // 	query := r.db.WithContext(ctx).
// // 		Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)

// // 	if sessionID != "" {
// // 		query = query.Where("session_id = ?", sessionID)
// // 	}
// // 	if termID != "" {
// // 		query = query.Where("term_id = ?", termID)
// // 	}
// // 	if classID != "" {
// // 		query = query.Where("class_id = ?", classID)
// // 	}

// // 	err := query.First(&q).Error
// // 	if err != nil {
// // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // 			return nil, nil
// // 		}
// // 		return nil, fmt.Errorf("failed to find question by external ID: %w", err)
// // 	}
// // 	return &q, nil
// // }

// // // ============================================
// // // CONTEXT-AWARE QUERY METHODS
// // // ============================================

// // func (r *QuestionRepository) FindByTerm(ctx context.Context, subjectID, termID string) ([]models.QuestionBank, error) {
// // 	var questions []models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("subject_id = ? AND term_id = ? AND deleted_at IS NULL", subjectID, termID).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	return questions, err
// // }

// // func (r *QuestionRepository) FindBySession(ctx context.Context, subjectID, sessionID string) ([]models.QuestionBank, error) {
// // 	var questions []models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("subject_id = ? AND session_id = ? AND deleted_at IS NULL", subjectID, sessionID).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	return questions, err
// // }

// // func (r *QuestionRepository) FindByClass(ctx context.Context, classID string) ([]models.QuestionBank, error) {
// // 	var questions []models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("class_id = ? AND deleted_at IS NULL", classID).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	return questions, err
// // }

// // func (r *QuestionRepository) FindByClassLevel(ctx context.Context, classLevelID string) ([]models.QuestionBank, error) {
// // 	var questions []models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("class_level_id = ? AND deleted_at IS NULL", classLevelID).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	return questions, err
// // }

// // func (r *QuestionRepository) FindByExamType(ctx context.Context, examType string) ([]models.QuestionBank, error) {
// // 	var questions []models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("exam_type = ? AND deleted_at IS NULL", examType).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	return questions, err
// // }

// // func (r *QuestionRepository) FindBySchoolAndSession(ctx context.Context, schoolID, sessionID string, page, limit int) ([]models.QuestionBank, int64, error) {
// // 	var questions []models.QuestionBank
// // 	var total int64

// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Where("school_id = ? AND session_id = ? AND deleted_at IS NULL", schoolID, sessionID)

// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, err
// // 	}

// // 	offset := (page - 1) * limit
// // 	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
// // 	return questions, total, err
// // }

// // func (r *QuestionRepository) FindBySchoolAndTerm(ctx context.Context, schoolID, termID string, page, limit int) ([]models.QuestionBank, int64, error) {
// // 	var questions []models.QuestionBank
// // 	var total int64

// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Where("school_id = ? AND term_id = ? AND deleted_at IS NULL", schoolID, termID)

// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, err
// // 	}

// // 	offset := (page - 1) * limit
// // 	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
// // 	return questions, total, err
// // }

// // func (r *QuestionRepository) GetQuestionsByClassAndExamType(ctx context.Context, classID string, examType string) ([]models.QuestionBank, error) {
// // 	var questions []models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("class_id = ? AND exam_type = ? AND deleted_at IS NULL", classID, examType).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	return questions, err
// // }

// // func (r *QuestionRepository) CountQuestionsByTerm(ctx context.Context, termID string) (int64, error) {
// // 	var count int64
// // 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Where("term_id = ? AND deleted_at IS NULL", termID).
// // 		Count(&count).Error
// // 	return count, err
// // }

// // func (r *QuestionRepository) CountQuestionsByExamType(ctx context.Context, examType string) (int64, error) {
// // 	var count int64
// // 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Where("exam_type = ? AND deleted_at IS NULL", examType).
// // 		Count(&count).Error
// // 	return count, err
// // }

// // // ============================================
// // // SUMMARY & STATISTICS OPERATIONS
// // // ============================================

// // func (r *QuestionRepository) GetQuestionContextSummary(ctx context.Context, subjectID string) ([]QuestionTermSummary, error) {
// // 	var results []QuestionTermSummary

// // 	query := `
// // 		SELECT 
// // 			t.id as term_id,
// // 			t.name as term_name,
// // 			t.term_number,
// // 			s.name as session_name,
// // 			COUNT(q.id) as question_count,
// // 			SUM(CASE WHEN q.exam_type = 'weekly_test' THEN 1 ELSE 0 END) as weekly_test_count,
// // 			SUM(CASE WHEN q.exam_type = 'mid_term' THEN 1 ELSE 0 END) as mid_term_count,
// // 			SUM(CASE WHEN q.exam_type = 'main_exam' THEN 1 ELSE 0 END) as main_exam_count,
// // 			SUM(CASE WHEN q.exam_type = 'practice' THEN 1 ELSE 0 END) as practice_count
// // 		FROM question_bank q
// // 		JOIN terms t ON t.id = q.term_id
// // 		JOIN academic_sessions s ON s.id = q.session_id
// // 		WHERE q.subject_id = ? 
// // 			AND q.deleted_at IS NULL
// // 		GROUP BY t.id, t.name, t.term_number, s.name
// // 		ORDER BY t.term_number ASC
// // 	`

// // 	err := r.db.WithContext(ctx).Raw(query, subjectID).Scan(&results).Error
// // 	return results, err
// // }

// // func (r *QuestionRepository) GetStatisticsByExamType(ctx context.Context, subjectID string) (map[string]int64, error) {
// // 	var results []struct {
// // 		ExamType string
// // 		Count    int64
// // 	}

// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Select("exam_type, COUNT(*) as count").
// // 		Where("deleted_at IS NULL")

// // 	if subjectID != "" {
// // 		query = query.Where("subject_id = ?", subjectID)
// // 	}

// // 	err := query.Group("exam_type").Scan(&results).Error
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to get statistics by exam type: %w", err)
// // 	}

// // 	stats := make(map[string]int64)
// // 	for _, r := range results {
// // 		stats[r.ExamType] = r.Count
// // 	}

// // 	examTypes := []string{"weekly_test", "mid_term", "main_exam", "practice"}
// // 	for _, et := range examTypes {
// // 		if _, ok := stats[et]; !ok {
// // 			stats[et] = 0
// // 		}
// // 	}

// // 	return stats, nil
// // }

// // // ============================================
// // // BULK / BATCH OPERATIONS
// // // ============================================

// // func (r *QuestionRepository) BulkCreate(ctx context.Context, questions []models.QuestionBank) error {
// // 	if len(questions) == 0 {
// // 		return errors.New("no questions to create")
// // 	}
// // 	return r.db.WithContext(ctx).CreateInBatches(questions, 100).Error
// // }

// // func (r *QuestionRepository) BulkDelete(ctx context.Context, ids []string) error {
// // 	if len(ids) == 0 {
// // 		return errors.New("no question IDs provided")
// // 	}

// // 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// // 		}
// // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.ExamQuestion{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete exam questions: %w", err)
// // 		}
// // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete question attachments: %w", err)
// // 		}
// // 		if err := tx.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error; err != nil {
// // 			return fmt.Errorf("failed to delete questions: %w", err)
// // 		}
// // 		return nil
// // 	})
// // }

// // func (r *QuestionRepository) BulkUpdateStatus(ctx context.Context, ids []string, status string) error {
// // 	if len(ids) == 0 {
// // 		return errors.New("no question IDs provided")
// // 	}
// // 	if status == "" {
// // 		return errors.New("status cannot be empty")
// // 	}

// // 	result := r.db.WithContext(ctx).
// // 		Model(&models.QuestionBank{}).
// // 		Where("id IN ?", ids).
// // 		Update("status", models.QuestionStatus(status))

// // 	if result.Error != nil {
// // 		return fmt.Errorf("failed to update status: %w", result.Error)
// // 	}

// // 	if result.RowsAffected == 0 {
// // 		return errors.New("no questions found to update")
// // 	}

// // 	return nil
// // }

// // // ============================================
// // // VERSIONING OPERATIONS
// // // ============================================

// // func (r *QuestionRepository) CreateNewVersion(ctx context.Context, old *models.QuestionBank, updates map[string]interface{}) (string, error) {
// // 	if old == nil {
// // 		return "", errors.New("old question cannot be nil")
// // 	}
// // 	if len(updates) == 0 {
// // 		return "", errors.New("no updates provided")
// // 	}

// // 	newQ := *old
// // 	newQ.ID = generateID()
// // 	newQ.Version = old.Version + 1
// // 	newQ.ParentID = &old.ID
// // 	newQ.CreatedAt = time.Now()
// // 	newQ.UpdatedAt = time.Now()

// // 	for k, v := range updates {
// // 		if err := r.applyUpdate(&newQ, k, v); err != nil {
// // 			return "", fmt.Errorf("failed to apply update for field %s: %w", k, err)
// // 		}
// // 	}

// // 	err := r.db.WithContext(ctx).Create(&newQ).Error
// // 	if err != nil {
// // 		return "", fmt.Errorf("failed to create new version: %w", err)
// // 	}

// // 	return newQ.ID, nil
// // }

// // func (r *QuestionRepository) applyUpdate(q *models.QuestionBank, key string, value interface{}) error {
// // 	switch key {
// // 	case "topic":
// // 		if v, ok := value.(string); ok {
// // 			q.Topic = v
// // 		}
// // 	case "sub_topic":
// // 		if v, ok := value.(string); ok {
// // 			q.SubTopic = v
// // 		}
// // 	case "learning_objective":
// // 		if v, ok := value.(string); ok {
// // 			q.LearningObjective = v
// // 		}
// // 	case "question_text":
// // 		if v, ok := value.(string); ok {
// // 			q.QuestionText = v
// // 		}
// // 	case "question_type":
// // 		if v, ok := value.(string); ok {
// // 			q.QuestionType = models.QuestionType(v)
// // 		}
// // 	case "difficulty":
// // 		if v, ok := value.(string); ok {
// // 			q.Difficulty = models.DifficultyLevel(v)
// // 		}
// // 	case "bloom_level":
// // 		if v, ok := value.(string); ok {
// // 			q.BloomLevel = models.BloomTaxonomy(v)
// // 		}
// // 	case "exam_type":
// // 		if v, ok := value.(string); ok {
// // 			q.ExamType = v
// // 		}
// // 	case "school_id":
// // 		if v, ok := value.(string); ok {
// // 			q.SchoolID = v
// // 		}
// // 	case "session_id":
// // 		if v, ok := value.(string); ok {
// // 			q.SessionID = v
// // 		}
// // 	case "term_id":
// // 		if v, ok := value.(string); ok {
// // 			q.TermID = v
// // 		}
// // 	case "class_level_id":
// // 		if v, ok := value.(string); ok {
// // 			q.ClassLevelID = v
// // 		}
// // 	case "class_id":
// // 		if v, ok := value.(string); ok {
// // 			q.ClassID = v
// // 		}
// // 	case "subject_id":
// // 		if v, ok := value.(string); ok {
// // 			q.SubjectID = v
// // 		}
// // 	case "options":
// // 		if v, ok := value.(models.OptionStorage); ok {
// // 			q.Options = v
// // 		} else {
// // 			return fmt.Errorf("options must be of type OptionStorage, got %T", value)
// // 		}
// // 	case "correct_option_keys":
// // 		if v, ok := value.([]string); ok {
// // 			q.CorrectOptionKeys = v
// // 		}
// // 	case "correct_answer":
// // 		if v, ok := value.(string); ok {
// // 			q.CorrectAnswer = v
// // 		}
// // 	case "rubric":
// // 		if v, ok := value.(models.RubricStorage); ok {
// // 			q.Rubric = v
// // 		} else {
// // 			return fmt.Errorf("rubric must be of type RubricStorage, got %T", value)
// // 		}
// // 	case "tags":
// // 		if v, ok := value.(models.TagStorage); ok {
// // 			q.Tags = v
// // 		} else {
// // 			return fmt.Errorf("tags must be of type TagStorage, got %T", value)
// // 		}
// // 	case "explanation":
// // 		if v, ok := value.(string); ok {
// // 			q.Explanation = v
// // 		}
// // 	case "marks":
// // 		if v, ok := value.(int); ok {
// // 			q.Marks = v
// // 		}
// // 	case "negative_marks":
// // 		if v, ok := value.(float64); ok {
// // 			q.NegativeMarks = v
// // 		}
// // 	case "time_limit_seconds":
// // 		if v, ok := value.(*int); ok {
// // 			q.TimeLimitSeconds = v
// // 		} else if v, ok := value.(int); ok {
// // 			q.TimeLimitSeconds = &v
// // 		}
// // 	case "order":
// // 		if v, ok := value.(int); ok {
// // 			q.Order = v
// // 		}
// // 	case "is_required":
// // 		if v, ok := value.(bool); ok {
// // 			q.IsRequired = v
// // 		}
// // 	case "updated_by":
// // 		if v, ok := value.(string); ok {
// // 			q.UpdatedBy = v
// // 		}
// // 	case "curriculum_type":
// // 		if v, ok := value.(string); ok {
// // 			q.CurriculumType = v
// // 		}
// // 	case "source_type":
// // 		if v, ok := value.(string); ok {
// // 			q.SourceType = v
// // 		}
// // 	case "status":
// // 		if v, ok := value.(string); ok {
// // 			q.Status = models.QuestionStatus(v)
// // 		}
// // 	default:
// // 		return fmt.Errorf("unknown field: %s", key)
// // 	}
// // 	return nil
// // }

// // // ============================================
// // // TAG OPERATIONS
// // // ============================================

// // func (r *QuestionRepository) CreateTag(ctx context.Context, tag *models.Tag) error {
// // 	if tag == nil {
// // 		return errors.New("tag cannot be nil")
// // 	}
// // 	return r.db.WithContext(ctx).Create(tag).Error
// // }

// // func (r *QuestionRepository) FindTagByName(ctx context.Context, name string) (*models.Tag, error) {
// // 	if name == "" {
// // 		return nil, errors.New("tag name cannot be empty")
// // 	}

// // 	var tag models.Tag
// // 	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
// // 	if err != nil {
// // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // 			return nil, nil
// // 		}
// // 		return nil, fmt.Errorf("failed to find tag by name: %w", err)
// // 	}
// // 	return &tag, nil
// // }

// // func (r *QuestionRepository) ListTags(ctx context.Context) ([]models.Tag, error) {
// // 	var tags []models.Tag
// // 	err := r.db.WithContext(ctx).
// // 		Where("deleted_at IS NULL").
// // 		Order("name ASC").
// // 		Find(&tags).Error
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to list tags: %w", err)
// // 	}
// // 	return tags, nil
// // }

// // func (r *QuestionRepository) ListTagsPaginated(ctx context.Context, page, limit int) ([]models.Tag, int64, error) {
// // 	if page < 1 {
// // 		page = 1
// // 	}
// // 	if limit < 1 || limit > 100 {
// // 		limit = 20
// // 	}

// // 	var tags []models.Tag
// // 	var total int64

// // 	query := r.db.WithContext(ctx).Model(&models.Tag{}).Where("deleted_at IS NULL")

// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, fmt.Errorf("failed to count tags: %w", err)
// // 	}

// // 	offset := (page - 1) * limit
// // 	err := query.
// // 		Offset(offset).
// // 		Limit(limit).
// // 		Order("name ASC").
// // 		Find(&tags).Error
// // 	if err != nil {
// // 		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
// // 	}

// // 	return tags, total, nil
// // }

// // func (r *QuestionRepository) AttachTags(ctx context.Context, questionID string, tagIDs []string) error {
// // 	if questionID == "" {
// // 		return errors.New("question ID cannot be empty")
// // 	}
// // 	if len(tagIDs) == 0 {
// // 		return nil
// // 	}

// // 	for _, tid := range tagIDs {
// // 		mapping := models.QuestionTagMapping{
// // 			ID:         generateID(),
// // 			QuestionID: questionID,
// // 			TagID:      tid,
// // 		}
// // 		if err := r.db.WithContext(ctx).Create(&mapping).Error; err != nil {
// // 			return fmt.Errorf("failed to attach tag %s to question %s: %w", tid, questionID, err)
// // 		}
// // 	}
// // 	return nil
// // }

// // func (r *QuestionRepository) DetachTags(ctx context.Context, questionID string, tagIDs []string) error {
// // 	if questionID == "" {
// // 		return errors.New("question ID cannot be empty")
// // 	}
// // 	if len(tagIDs) == 0 {
// // 		return nil
// // 	}

// // 	result := r.db.WithContext(ctx).
// // 		Where("question_id = ? AND tag_id IN ?", questionID, tagIDs).
// // 		Delete(&models.QuestionTagMapping{})

// // 	if result.Error != nil {
// // 		return fmt.Errorf("failed to detach tags: %w", result.Error)
// // 	}
// // 	return nil
// // }

// // // ============================================
// // // STATISTICS OPERATIONS
// // // ============================================

// // func (r *QuestionRepository) GetStatistics(ctx context.Context, subjectID string) (map[string]interface{}, error) {
// // 	var total int64
// // 	var published, draft, archived int64

// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
// // 	if subjectID != "" {
// // 		query = query.Where("subject_id = ?", subjectID)
// // 	}

// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to count total questions: %w", err)
// // 	}

// // 	if err := query.Where("status = ?", models.QuestionStatusPublished).Count(&published).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to count published questions: %w", err)
// // 	}
// // 	if err := query.Where("status = ?", models.QuestionStatusDraft).Count(&draft).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to count draft questions: %w", err)
// // 	}
// // 	if err := query.Where("status = ?", models.QuestionStatusArchived).Count(&archived).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to count archived questions: %w", err)
// // 	}

// // 	var avgMarks float64
// // 	if err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Select("COALESCE(AVG(marks), 0)").
// // 		Where("deleted_at IS NULL").
// // 		Row().Scan(&avgMarks); err != nil {
// // 		return nil, fmt.Errorf("failed to calculate average marks: %w", err)
// // 	}

// // 	examTypeStats, err := r.GetStatisticsByExamType(ctx, subjectID)
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to get exam type statistics: %w", err)
// // 	}

// // 	return map[string]interface{}{
// // 		"total_questions": total,
// // 		"published_count": published,
// // 		"draft_count":     draft,
// // 		"archived_count":  archived,
// // 		"average_marks":   avgMarks,
// // 		"by_exam_type":    examTypeStats,
// // 	}, nil
// // }

// // func (r *QuestionRepository) GetDetailedStatistics(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// // 	for key, val := range filters {
// // 		if val != nil && val != "" {
// // 			query = query.Where(key+" = ?", val)
// // 		}
// // 	}

// // 	var total int64
// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to count questions: %w", err)
// // 	}

// // 	var published, draft, archived int64
// // 	query.Where("status = ?", models.QuestionStatusPublished).Count(&published)
// // 	query.Where("status = ?", models.QuestionStatusDraft).Count(&draft)
// // 	query.Where("status = ?", models.QuestionStatusArchived).Count(&archived)

// // 	var avgMarks float64
// // 	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

// // 	type DifficultyCount struct {
// // 		Difficulty string
// // 		Count      int
// // 	}
// // 	var difficultyCounts []DifficultyCount
// // 	if err := query.Select("difficulty, count(*) as count").
// // 		Group("difficulty").
// // 		Scan(&difficultyCounts).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to get difficulty breakdown: %w", err)
// // 	}

// // 	byDifficulty := make(map[string]int)
// // 	for _, dc := range difficultyCounts {
// // 		byDifficulty[dc.Difficulty] = dc.Count
// // 	}

// // 	type TypeCount struct {
// // 		QuestionType string
// // 		Count        int
// // 	}
// // 	var typeCounts []TypeCount
// // 	if err := query.Select("question_type, count(*) as count").
// // 		Group("question_type").
// // 		Scan(&typeCounts).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to get question type breakdown: %w", err)
// // 	}

// // 	byType := make(map[string]int)
// // 	for _, tc := range typeCounts {
// // 		byType[tc.QuestionType] = tc.Count
// // 	}

// // 	type ExamTypeCount struct {
// // 		ExamType string
// // 		Count    int
// // 	}
// // 	var examTypeCounts []ExamTypeCount
// // 	if err := query.Select("exam_type, count(*) as count").
// // 		Group("exam_type").
// // 		Scan(&examTypeCounts).Error; err != nil {
// // 		return nil, fmt.Errorf("failed to get exam type breakdown: %w", err)
// // 	}

// // 	byExamType := make(map[string]int)
// // 	for _, etc := range examTypeCounts {
// // 		byExamType[etc.ExamType] = etc.Count
// // 	}

// // 	return map[string]interface{}{
// // 		"total":            total,
// // 		"published":        published,
// // 		"draft":            draft,
// // 		"archived":         archived,
// // 		"average_marks":    avgMarks,
// // 		"by_difficulty":    byDifficulty,
// // 		"by_question_type": byType,
// // 		"by_exam_type":     byExamType,
// // 	}, nil
// // }

// // // ============================================
// // // ADDITIONAL HELPER FUNCTIONS
// // // ============================================

// // func (r *QuestionRepository) FindByQuestionText(ctx context.Context, text string, limit int) ([]models.QuestionBank, error) {
// // 	if text == "" {
// // 		return nil, errors.New("search text cannot be empty")
// // 	}
// // 	if limit < 1 || limit > 100 {
// // 		limit = 20
// // 	}

// // 	var questions []models.QuestionBank
// // 	err := r.db.WithContext(ctx).
// // 		Where("question_text ILIKE ? AND deleted_at IS NULL", "%"+escapeWildcards(text)+"%").
// // 		Limit(limit).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to search questions: %w", err)
// // 	}
// // 	return questions, nil
// // }

// // func (r *QuestionRepository) CountQuestionsBySubject(ctx context.Context, subjectID string) (int64, error) {
// // 	var count int64
// // 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Where("subject_id = ? AND deleted_at IS NULL", subjectID).
// // 		Count(&count).Error
// // 	if err != nil {
// // 		return 0, fmt.Errorf("failed to count questions by subject: %w", err)
// // 	}
// // 	return count, nil
// // }

// // func (r *QuestionRepository) GetQuestionsByStatus(ctx context.Context, status models.QuestionStatus, page, limit int) ([]models.QuestionBank, int64, error) {
// // 	if page < 1 {
// // 		page = 1
// // 	}
// // 	if limit < 1 || limit > 100 {
// // 		limit = 20
// // 	}

// // 	var questions []models.QuestionBank
// // 	var total int64

// // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // 		Where("status = ? AND deleted_at IS NULL", status)

// // 	if err := query.Count(&total).Error; err != nil {
// // 		return nil, 0, fmt.Errorf("failed to count questions by status: %w", err)
// // 	}

// // 	offset := (page - 1) * limit
// // 	err := query.
// // 		Offset(offset).
// // 		Limit(limit).
// // 		Order("created_at DESC").
// // 		Find(&questions).Error
// // 	if err != nil {
// // 		return nil, 0, fmt.Errorf("failed to get questions by status: %w", err)
// // 	}

// // 	return questions, total, nil
// // }

// // func (r *QuestionRepository) GetRecentQuestions(ctx context.Context, subjectID string, limit int) ([]models.QuestionBank, error) {
// // 	if limit < 1 || limit > 50 {
// // 		limit = 10
// // 	}

// // 	var questions []models.QuestionBank
// // 	query := r.db.WithContext(ctx).
// // 		Where("deleted_at IS NULL").
// // 		Order("created_at DESC").
// // 		Limit(limit)

// // 	if subjectID != "" {
// // 		query = query.Where("subject_id = ?", subjectID)
// // 	}

// // 	err := query.Find(&questions).Error
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to get recent questions: %w", err)
// // 	}
// // 	return questions, nil
// // }

// // // ============================================
// // // FUNCTION TO GET DB INSTANCE
// // // ============================================

// // func (r *QuestionRepository) GetDB() *gorm.DB {
// // 	return r.db
// // }

// // // ============================================
// // // SUMMARY STRUCTS
// // // ============================================

// // type QuestionTermSummary struct {
// // 	TermID          string `json:"term_id"`
// // 	TermName        string `json:"term_name"`
// // 	TermNumber      int    `json:"term_number"`
// // 	SessionName     string `json:"session_name"`
// // 	QuestionCount   int    `json:"question_count"`
// // 	WeeklyTestCount int    `json:"weekly_test_count"`
// // 	MidTermCount    int    `json:"mid_term_count"`
// // 	MainExamCount   int    `json:"main_exam_count"`
// // 	PracticeCount   int    `json:"practice_count"`
// // }




// // // package repository

// // // import (
// // // 	"cbt-api/internal/models"
// // // 	"context"
// // // 	"errors"
// // // 	"fmt"
// // // 	"time"

// // // 	"github.com/google/uuid"
// // // 	"gorm.io/gorm"
// // // )

// // // type QuestionRepository struct {
// // // 	db *gorm.DB
// // // }

// // // func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
// // // 	return &QuestionRepository{db: db}
// // // }

// // // // ============================================
// // // // CRUD OPERATIONS
// // // // ============================================

// // // func (r *QuestionRepository) Create(ctx context.Context, question *models.QuestionBank) error {
// // // 	if question == nil {
// // // 		return errors.New("question cannot be nil")
// // // 	}
// // // 	return r.db.WithContext(ctx).Create(question).Error
// // // }

// // // func (r *QuestionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.QuestionBank, error) {
// // // 	var q models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("id = ? AND deleted_at IS NULL", id).
// // // 		First(&q).Error
// // // 	if err != nil {
// // // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // // 			return nil, fmt.Errorf("question with ID %s not found", id)
// // // 		}
// // // 		return nil, fmt.Errorf("failed to find question: %w", err)
// // // 	}
// // // 	return &q, nil
// // // }

// // // func (r *QuestionRepository) Update(ctx context.Context, question *models.QuestionBank) error {
// // // 	if question == nil {
// // // 		return errors.New("question cannot be nil")
// // // 	}
// // // 	return r.db.WithContext(ctx).Save(question).Error
// // // }

// // // func (r *QuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
// // // 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// // // 		// 1. Delete question tag mappings
// // // 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// // // 		}

// // // 		// 2. Delete exam questions
// // // 		if err := tx.Where("question_id = ?", id).Delete(&models.ExamQuestion{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete exam questions: %w", err)
// // // 		}

// // // 		// 3. Delete question attachments
// // // 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete question attachments: %w", err)
// // // 		}

// // // 		// 4. Soft delete the question
// // // 		if err := tx.Where("id = ?", id).Delete(&models.QuestionBank{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete question: %w", err)
// // // 		}

// // // 		return nil
// // // 	})
// // // }

// // // // ============================================
// // // // QUERY / FILTER OPERATIONS
// // // // ============================================

// // // func (r *QuestionRepository) ListBySubject(ctx context.Context, subjectID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
// // // 	if page < 1 {
// // // 		page = 1
// // // 	}
// // // 	if limit < 1 || limit > 100 {
// // // 		limit = 20
// // // 	}

// // // 	offset := (page - 1) * limit
// // // 	var questions []models.QuestionBank
// // // 	var total int64

// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Where("subject_id = ? AND deleted_at IS NULL", subjectID)

// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to count questions: %w", err)
// // // 	}

// // // 	err := query.
// // // 		Offset(offset).
// // // 		Limit(limit).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	if err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to list questions: %w", err)
// // // 	}

// // // 	return questions, total, nil
// // // }

// // // func (r *QuestionRepository) Filter(ctx context.Context, params map[string]interface{}, page, limit int) ([]models.QuestionBank, int64, error) {
// // // 	if page < 1 {
// // // 		page = 1
// // // 	}
// // // 	if limit < 1 || limit > 100 {
// // // 		limit = 20
// // // 	}

// // // 	var questions []models.QuestionBank
// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// // // 	// ============================================================
// // // 	// ACADEMIC CONTEXT FILTERS - ALL 7 FIELDS
// // // 	// ============================================================
// // // 	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
// // // 		query = query.Where("subject_id = ?", subjectID)
// // // 	}
// // // 	if schoolID, ok := params["school_id"]; ok && schoolID != "" {
// // // 		query = query.Where("school_id = ?", schoolID)
// // // 	}
// // // 	if sessionID, ok := params["session_id"]; ok && sessionID != "" {
// // // 		query = query.Where("session_id = ?", sessionID)
// // // 	}
// // // 	if termID, ok := params["term_id"]; ok && termID != "" {
// // // 		query = query.Where("term_id = ?", termID)
// // // 	}
// // // 	if classLevelID, ok := params["class_level_id"]; ok && classLevelID != "" {
// // // 		query = query.Where("class_level_id = ?", classLevelID)
// // // 	}
// // // 	if classID, ok := params["class_id"]; ok && classID != "" {
// // // 		query = query.Where("class_id = ?", classID)
// // // 	}
// // // 	if examType, ok := params["exam_type"]; ok && examType != "" {
// // // 		query = query.Where("exam_type = ?", examType)
// // // 	}

// // // 	// ============================================================
// // // 	// CONTENT FILTERS
// // // 	// ============================================================
// // // 	if topic, ok := params["topic"]; ok && topic != "" {
// // // 		query = query.Where("topic ILIKE ?", "%"+topic.(string)+"%")
// // // 	}
// // // 	if difficulty, ok := params["difficulty"]; ok && difficulty != "" {
// // // 		query = query.Where("difficulty = ?", difficulty)
// // // 	}
// // // 	if bloomLevel, ok := params["bloom_level"]; ok && bloomLevel != "" {
// // // 		query = query.Where("bloom_level = ?", bloomLevel)
// // // 	}
// // // 	if questionType, ok := params["question_type"]; ok && questionType != "" {
// // // 		query = query.Where("question_type = ?", questionType)
// // // 	}
// // // 	if status, ok := params["status"]; ok && status != "" {
// // // 		query = query.Where("status = ?", status)
// // // 	}
// // // 	if curriculumType, ok := params["curriculum_type"]; ok && curriculumType != "" {
// // // 		query = query.Where("curriculum_type = ?", curriculumType)
// // // 	}
// // // 	if sourceType, ok := params["source_type"]; ok && sourceType != "" {
// // // 		query = query.Where("source_type = ?", sourceType)
// // // 	}
// // // 	if externalID, ok := params["external_id"]; ok && externalID != "" {
// // // 		query = query.Where("external_id = ?", externalID)
// // // 	}

// // // 	// ============================================================
// // // 	// SEARCH
// // // 	// ============================================================
// // // 	if search, ok := params["search"]; ok && search != "" {
// // // 		searchStr := "%" + escapeWildcards(search.(string)) + "%"
// // // 		query = query.Where("question_text ILIKE ?", searchStr)
// // // 	}

// // // 	var total int64
// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to count filtered questions: %w", err)
// // // 	}

// // // 	offset := (page - 1) * limit
// // // 	err := query.
// // // 		Offset(offset).
// // // 		Limit(limit).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	if err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to filter questions: %w", err)
// // // 	}

// // // 	return questions, total, nil
// // // }

// // // // escapeWildcards escapes wildcard characters in search strings
// // // func escapeWildcards(s string) string {
// // // 	result := ""
// // // 	for _, c := range s {
// // // 		if c == '%' || c == '_' {
// // // 			result += "\\" + string(c)
// // // 		} else {
// // // 			result += string(c)
// // // 		}
// // // 	}
// // // 	return result
// // // }

// // // func (r *QuestionRepository) FindByTag(ctx context.Context, tagName string, page, limit int) ([]models.QuestionBank, int64, error) {
// // // 	if page < 1 {
// // // 		page = 1
// // // 	}
// // // 	if limit < 1 || limit > 100 {
// // // 		limit = 20
// // // 	}

// // // 	var questions []models.QuestionBank
// // // 	query := r.db.WithContext(ctx).
// // // 		Joins("JOIN question_tag_mappings ON question_tag_mappings.question_id = question_bank.id").
// // // 		Joins("JOIN tags ON tags.id = question_tag_mappings.tag_id").
// // // 		Where("tags.name = ? AND question_bank.deleted_at IS NULL", tagName)

// // // 	var total int64
// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to count questions by tag: %w", err)
// // // 	}

// // // 	offset := (page - 1) * limit
// // // 	err := query.
// // // 		Offset(offset).
// // // 		Limit(limit).
// // // 		Order("question_bank.created_at DESC").
// // // 		Find(&questions).Error
// // // 	if err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to find questions by tag: %w", err)
// // // 	}

// // // 	return questions, total, nil
// // // }

// // // func (r *QuestionRepository) FindByExternalID(ctx context.Context, schoolID, sessionID, termID, classID uuid.UUID, externalID string) (*models.QuestionBank, error) {
// // // 	if externalID == "" {
// // // 		return nil, errors.New("external ID cannot be empty")
// // // 	}

// // // 	var q models.QuestionBank
// // // 	query := r.db.WithContext(ctx).
// // // 		Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)

// // // 	if sessionID != uuid.Nil {
// // // 		query = query.Where("session_id = ?", sessionID)
// // // 	}
// // // 	if termID != uuid.Nil {
// // // 		query = query.Where("term_id = ?", termID)
// // // 	}
// // // 	if classID != uuid.Nil {
// // // 		query = query.Where("class_id = ?", classID)
// // // 	}

// // // 	err := query.First(&q).Error
// // // 	if err != nil {
// // // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // // 			return nil, nil
// // // 		}
// // // 		return nil, fmt.Errorf("failed to find question by external ID: %w", err)
// // // 	}
// // // 	return &q, nil
// // // }

// // // // ============================================
// // // // CONTEXT-AWARE QUERY METHODS - NEW & UPDATED
// // // // ============================================

// // // // FindByTerm - Get questions for a specific term
// // // func (r *QuestionRepository) FindByTerm(ctx context.Context, subjectID, termID uuid.UUID) ([]models.QuestionBank, error) {
// // // 	var questions []models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("subject_id = ? AND term_id = ? AND deleted_at IS NULL", subjectID, termID).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	return questions, err
// // // }

// // // // FindBySession - Get questions for a specific session
// // // func (r *QuestionRepository) FindBySession(ctx context.Context, subjectID, sessionID uuid.UUID) ([]models.QuestionBank, error) {
// // // 	var questions []models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("subject_id = ? AND session_id = ? AND deleted_at IS NULL", subjectID, sessionID).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	return questions, err
// // // }

// // // // FindByClass - Get questions for a specific class (NEW)
// // // func (r *QuestionRepository) FindByClass(ctx context.Context, classID uuid.UUID) ([]models.QuestionBank, error) {
// // // 	var questions []models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("class_id = ? AND deleted_at IS NULL", classID).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	return questions, err
// // // }

// // // // FindByClassLevel - Get questions for a specific class level (NEW)
// // // func (r *QuestionRepository) FindByClassLevel(ctx context.Context, classLevelID uuid.UUID) ([]models.QuestionBank, error) {
// // // 	var questions []models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("class_level_id = ? AND deleted_at IS NULL", classLevelID).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	return questions, err
// // // }

// // // // FindByExamType - Get questions by exam type (NEW)
// // // func (r *QuestionRepository) FindByExamType(ctx context.Context, examType string) ([]models.QuestionBank, error) {
// // // 	var questions []models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("exam_type = ? AND deleted_at IS NULL", examType).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	return questions, err
// // // }

// // // // FindBySchoolAndSession - Get questions by school and session
// // // func (r *QuestionRepository) FindBySchoolAndSession(ctx context.Context, schoolID, sessionID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
// // // 	var questions []models.QuestionBank
// // // 	var total int64

// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Where("school_id = ? AND session_id = ? AND deleted_at IS NULL", schoolID, sessionID)

// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, 0, err
// // // 	}

// // // 	offset := (page - 1) * limit
// // // 	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
// // // 	return questions, total, err
// // // }

// // // // FindBySchoolAndTerm - Get questions by school and term (NEW)
// // // func (r *QuestionRepository) FindBySchoolAndTerm(ctx context.Context, schoolID, termID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
// // // 	var questions []models.QuestionBank
// // // 	var total int64

// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Where("school_id = ? AND term_id = ? AND deleted_at IS NULL", schoolID, termID)

// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, 0, err
// // // 	}

// // // 	offset := (page - 1) * limit
// // // 	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
// // // 	return questions, total, err
// // // }

// // // // GetQuestionsByClassAndExamType - Get questions by class and exam type (NEW)
// // // func (r *QuestionRepository) GetQuestionsByClassAndExamType(ctx context.Context, classID uuid.UUID, examType string) ([]models.QuestionBank, error) {
// // // 	var questions []models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("class_id = ? AND exam_type = ? AND deleted_at IS NULL", classID, examType).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	return questions, err
// // // }

// // // // CountQuestionsByTerm - Count questions in a term
// // // func (r *QuestionRepository) CountQuestionsByTerm(ctx context.Context, termID uuid.UUID) (int64, error) {
// // // 	var count int64
// // // 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Where("term_id = ? AND deleted_at IS NULL", termID).
// // // 		Count(&count).Error
// // // 	return count, err
// // // }

// // // // CountQuestionsByExamType - Count questions by exam type (NEW)
// // // func (r *QuestionRepository) CountQuestionsByExamType(ctx context.Context, examType string) (int64, error) {
// // // 	var count int64
// // // 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Where("exam_type = ? AND deleted_at IS NULL", examType).
// // // 		Count(&count).Error
// // // 	return count, err
// // // }

// // // // ============================================
// // // // SUMMARY & STATISTICS OPERATIONS
// // // // ============================================

// // // // GetQuestionContextSummary - Get summary grouped by term with exam type breakdown (UPDATED)
// // // func (r *QuestionRepository) GetQuestionContextSummary(ctx context.Context, subjectID uuid.UUID) ([]QuestionTermSummary, error) {
// // // 	var results []QuestionTermSummary

// // // 	query := `
// // // 		SELECT 
// // // 			t.id as term_id,
// // // 			t.name as term_name,
// // // 			t.term_number,
// // // 			s.name as session_name,
// // // 			COUNT(q.id) as question_count,
// // // 			SUM(CASE WHEN q.exam_type = 'weekly_test' THEN 1 ELSE 0 END) as weekly_test_count,
// // // 			SUM(CASE WHEN q.exam_type = 'mid_term' THEN 1 ELSE 0 END) as mid_term_count,
// // // 			SUM(CASE WHEN q.exam_type = 'main_exam' THEN 1 ELSE 0 END) as main_exam_count,
// // // 			SUM(CASE WHEN q.exam_type = 'practice' THEN 1 ELSE 0 END) as practice_count
// // // 		FROM question_bank q
// // // 		JOIN terms t ON t.id = q.term_id
// // // 		JOIN academic_sessions s ON s.id = q.session_id
// // // 		WHERE q.subject_id = ? 
// // // 			AND q.deleted_at IS NULL
// // // 		GROUP BY t.id, t.name, t.term_number, s.name
// // // 		ORDER BY t.term_number ASC
// // // 	`

// // // 	err := r.db.WithContext(ctx).Raw(query, subjectID).Scan(&results).Error
// // // 	return results, err
// // // }

// // // // GetStatisticsByExamType - Get statistics grouped by exam type (NEW)
// // // func (r *QuestionRepository) GetStatisticsByExamType(ctx context.Context, subjectID uuid.UUID) (map[string]int64, error) {
// // // 	var results []struct {
// // // 		ExamType string
// // // 		Count    int64
// // // 	}

// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Select("exam_type, COUNT(*) as count").
// // // 		Where("deleted_at IS NULL")

// // // 	if subjectID != uuid.Nil {
// // // 		query = query.Where("subject_id = ?", subjectID)
// // // 	}

// // // 	err := query.Group("exam_type").Scan(&results).Error
// // // 	if err != nil {
// // // 		return nil, fmt.Errorf("failed to get statistics by exam type: %w", err)
// // // 	}

// // // 	stats := make(map[string]int64)
// // // 	for _, r := range results {
// // // 		stats[r.ExamType] = r.Count
// // // 	}

// // // 	// Ensure all exam types are present
// // // 	examTypes := []string{"weekly_test", "mid_term", "main_exam", "practice"}
// // // 	for _, et := range examTypes {
// // // 		if _, ok := stats[et]; !ok {
// // // 			stats[et] = 0
// // // 		}
// // // 	}

// // // 	return stats, nil
// // // }

// // // // ============================================
// // // // BULK / BATCH OPERATIONS
// // // // ============================================

// // // func (r *QuestionRepository) BulkCreate(ctx context.Context, questions []models.QuestionBank) error {
// // // 	if len(questions) == 0 {
// // // 		return errors.New("no questions to create")
// // // 	}
// // // 	return r.db.WithContext(ctx).CreateInBatches(questions, 100).Error
// // // }

// // // func (r *QuestionRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
// // // 	if len(ids) == 0 {
// // // 		return errors.New("no question IDs provided")
// // // 	}

// // // 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// // // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// // // 		}
// // // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.ExamQuestion{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete exam questions: %w", err)
// // // 		}
// // // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete question attachments: %w", err)
// // // 		}
// // // 		if err := tx.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error; err != nil {
// // // 			return fmt.Errorf("failed to delete questions: %w", err)
// // // 		}
// // // 		return nil
// // // 	})
// // // }

// // // func (r *QuestionRepository) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status string) error {
// // // 	if len(ids) == 0 {
// // // 		return errors.New("no question IDs provided")
// // // 	}
// // // 	if status == "" {
// // // 		return errors.New("status cannot be empty")
// // // 	}

// // // 	result := r.db.WithContext(ctx).
// // // 		Model(&models.QuestionBank{}).
// // // 		Where("id IN ?", ids).
// // // 		Update("status", models.QuestionStatus(status))

// // // 	if result.Error != nil {
// // // 		return fmt.Errorf("failed to update status: %w", result.Error)
// // // 	}

// // // 	if result.RowsAffected == 0 {
// // // 		return errors.New("no questions found to update")
// // // 	}

// // // 	return nil
// // // }

// // // // ============================================
// // // // VERSIONING OPERATIONS
// // // // ============================================

// // // func (r *QuestionRepository) CreateNewVersion(ctx context.Context, old *models.QuestionBank, updates map[string]interface{}) (uuid.UUID, error) {
// // // 	if old == nil {
// // // 		return uuid.Nil, errors.New("old question cannot be nil")
// // // 	}
// // // 	if len(updates) == 0 {
// // // 		return uuid.Nil, errors.New("no updates provided")
// // // 	}

// // // 	newQ := *old
// // // 	newQ.ID = uuid.New()
// // // 	newQ.Version = old.Version + 1
// // // 	newQ.ParentID = &old.ID
// // // 	newQ.CreatedAt = time.Now()
// // // 	newQ.UpdatedAt = time.Now()

// // // 	for k, v := range updates {
// // // 		if err := r.applyUpdate(&newQ, k, v); err != nil {
// // // 			return uuid.Nil, fmt.Errorf("failed to apply update for field %s: %w", k, err)
// // // 		}
// // // 	}

// // // 	err := r.db.WithContext(ctx).Create(&newQ).Error
// // // 	if err != nil {
// // // 		return uuid.Nil, fmt.Errorf("failed to create new version: %w", err)
// // // 	}

// // // 	return newQ.ID, nil
// // // }

// // // func (r *QuestionRepository) applyUpdate(q *models.QuestionBank, key string, value interface{}) error {
// // // 	switch key {
// // // 	case "topic":
// // // 		if v, ok := value.(string); ok {
// // // 			q.Topic = v
// // // 		}
// // // 	case "sub_topic":
// // // 		if v, ok := value.(string); ok {
// // // 			q.SubTopic = v
// // // 		}
// // // 	case "learning_objective":
// // // 		if v, ok := value.(string); ok {
// // // 			q.LearningObjective = v
// // // 		}
// // // 	case "question_text":
// // // 		if v, ok := value.(string); ok {
// // // 			q.QuestionText = v
// // // 		}
// // // 	case "question_type":
// // // 		if v, ok := value.(string); ok {
// // // 			q.QuestionType = models.QuestionType(v)
// // // 		}
// // // 	case "difficulty":
// // // 		if v, ok := value.(string); ok {
// // // 			q.Difficulty = models.DifficultyLevel(v)
// // // 		}
// // // 	case "bloom_level":
// // // 		if v, ok := value.(string); ok {
// // // 			q.BloomLevel = models.BloomTaxonomy(v)
// // // 		}
// // // 	case "exam_type":
// // // 		if v, ok := value.(string); ok {
// // // 			q.ExamType = v
// // // 		}
// // // 	case "school_id":
// // // 		if v, ok := value.(uuid.UUID); ok {
// // // 			q.SchoolID = v
// // // 		}
// // // 	case "session_id":
// // // 		if v, ok := value.(uuid.UUID); ok {
// // // 			q.SessionID = v
// // // 		}
// // // 	case "term_id":
// // // 		if v, ok := value.(uuid.UUID); ok {
// // // 			q.TermID = v
// // // 		}
// // // 	case "class_level_id":
// // // 		if v, ok := value.(uuid.UUID); ok {
// // // 			q.ClassLevelID = v
// // // 		}
// // // 	case "class_id":
// // // 		if v, ok := value.(uuid.UUID); ok {
// // // 			q.ClassID = v
// // // 		}
// // // 	case "subject_id":
// // // 		if v, ok := value.(uuid.UUID); ok {
// // // 			q.SubjectID = v
// // // 		}
// // // 	case "options":
// // // 		if v, ok := value.(models.OptionStorage); ok {
// // // 			q.Options = v
// // // 		} else {
// // // 			return fmt.Errorf("options must be of type OptionStorage, got %T", value)
// // // 		}
// // // 	case "correct_option_keys":
// // // 		if v, ok := value.([]string); ok {
// // // 			q.CorrectOptionKeys = v
// // // 		}
// // // 	case "correct_answer":
// // // 		if v, ok := value.(string); ok {
// // // 			q.CorrectAnswer = v
// // // 		}
// // // 	case "rubric":
// // // 		if v, ok := value.(models.RubricStorage); ok {
// // // 			q.Rubric = v
// // // 		} else {
// // // 			return fmt.Errorf("rubric must be of type RubricStorage, got %T", value)
// // // 		}
// // // 	case "tags":
// // // 		if v, ok := value.(models.TagStorage); ok {
// // // 			q.Tags = v
// // // 		} else {
// // // 			return fmt.Errorf("tags must be of type TagStorage, got %T", value)
// // // 		}
// // // 	case "explanation":
// // // 		if v, ok := value.(string); ok {
// // // 			q.Explanation = v
// // // 		}
// // // 	case "marks":
// // // 		if v, ok := value.(int); ok {
// // // 			q.Marks = v
// // // 		}
// // // 	case "negative_marks":
// // // 		if v, ok := value.(float64); ok {
// // // 			q.NegativeMarks = v
// // // 		}
// // // 	case "time_limit_seconds":
// // // 		if v, ok := value.(*int); ok {
// // // 			q.TimeLimitSeconds = v
// // // 		} else if v, ok := value.(int); ok {
// // // 			q.TimeLimitSeconds = &v
// // // 		}
// // // 	case "order":
// // // 		if v, ok := value.(int); ok {
// // // 			q.Order = v
// // // 		}
// // // 	case "is_required":
// // // 		if v, ok := value.(bool); ok {
// // // 			q.IsRequired = v
// // // 		}
// // // 	case "updated_by":
// // // 		if v, ok := value.(uuid.UUID); ok {
// // // 			q.UpdatedBy = v
// // // 		}
// // // 	case "curriculum_type":
// // // 		if v, ok := value.(string); ok {
// // // 			q.CurriculumType = v
// // // 		}
// // // 	case "source_type":
// // // 		if v, ok := value.(string); ok {
// // // 			q.SourceType = v
// // // 		}
// // // 	case "status":
// // // 		if v, ok := value.(string); ok {
// // // 			q.Status = models.QuestionStatus(v)
// // // 		}
// // // 	default:
// // // 		return fmt.Errorf("unknown field: %s", key)
// // // 	}
// // // 	return nil
// // // }

// // // // ============================================
// // // // TAG OPERATIONS
// // // // ============================================

// // // func (r *QuestionRepository) CreateTag(ctx context.Context, tag *models.Tag) error {
// // // 	if tag == nil {
// // // 		return errors.New("tag cannot be nil")
// // // 	}
// // // 	return r.db.WithContext(ctx).Create(tag).Error
// // // }

// // // func (r *QuestionRepository) FindTagByName(ctx context.Context, name string) (*models.Tag, error) {
// // // 	if name == "" {
// // // 		return nil, errors.New("tag name cannot be empty")
// // // 	}

// // // 	var tag models.Tag
// // // 	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
// // // 	if err != nil {
// // // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // // 			return nil, nil
// // // 		}
// // // 		return nil, fmt.Errorf("failed to find tag by name: %w", err)
// // // 	}
// // // 	return &tag, nil
// // // }

// // // func (r *QuestionRepository) ListTags(ctx context.Context) ([]models.Tag, error) {
// // // 	var tags []models.Tag
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("deleted_at IS NULL").
// // // 		Order("name ASC").
// // // 		Find(&tags).Error
// // // 	if err != nil {
// // // 		return nil, fmt.Errorf("failed to list tags: %w", err)
// // // 	}
// // // 	return tags, nil
// // // }

// // // func (r *QuestionRepository) ListTagsPaginated(ctx context.Context, page, limit int) ([]models.Tag, int64, error) {
// // // 	if page < 1 {
// // // 		page = 1
// // // 	}
// // // 	if limit < 1 || limit > 100 {
// // // 		limit = 20
// // // 	}

// // // 	var tags []models.Tag
// // // 	var total int64

// // // 	query := r.db.WithContext(ctx).Model(&models.Tag{}).Where("deleted_at IS NULL")

// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to count tags: %w", err)
// // // 	}

// // // 	offset := (page - 1) * limit
// // // 	err := query.
// // // 		Offset(offset).
// // // 		Limit(limit).
// // // 		Order("name ASC").
// // // 		Find(&tags).Error
// // // 	if err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
// // // 	}

// // // 	return tags, total, nil
// // // }

// // // func (r *QuestionRepository) AttachTags(ctx context.Context, questionID uuid.UUID, tagIDs []uuid.UUID) error {
// // // 	if questionID == uuid.Nil {
// // // 		return errors.New("question ID cannot be nil")
// // // 	}
// // // 	if len(tagIDs) == 0 {
// // // 		return nil
// // // 	}

// // // 	for _, tid := range tagIDs {
// // // 		mapping := models.QuestionTagMapping{
// // // 			ID:         uuid.New(),
// // // 			QuestionID: questionID,
// // // 			TagID:      tid,
// // // 		}
// // // 		if err := r.db.WithContext(ctx).Create(&mapping).Error; err != nil {
// // // 			return fmt.Errorf("failed to attach tag %s to question %s: %w", tid, questionID, err)
// // // 		}
// // // 	}
// // // 	return nil
// // // }

// // // func (r *QuestionRepository) DetachTags(ctx context.Context, questionID uuid.UUID, tagIDs []uuid.UUID) error {
// // // 	if questionID == uuid.Nil {
// // // 		return errors.New("question ID cannot be nil")
// // // 	}
// // // 	if len(tagIDs) == 0 {
// // // 		return nil
// // // 	}

// // // 	result := r.db.WithContext(ctx).
// // // 		Where("question_id = ? AND tag_id IN ?", questionID, tagIDs).
// // // 		Delete(&models.QuestionTagMapping{})

// // // 	if result.Error != nil {
// // // 		return fmt.Errorf("failed to detach tags: %w", result.Error)
// // // 	}
// // // 	return nil
// // // }

// // // // ============================================
// // // // STATISTICS OPERATIONS
// // // // ============================================

// // // func (r *QuestionRepository) GetStatistics(ctx context.Context, subjectID uuid.UUID) (map[string]interface{}, error) {
// // // 	var total int64
// // // 	var published, draft, archived int64

// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
// // // 	if subjectID != uuid.Nil {
// // // 		query = query.Where("subject_id = ?", subjectID)
// // // 	}

// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to count total questions: %w", err)
// // // 	}

// // // 	if err := query.Where("status = ?", models.QuestionStatusPublished).Count(&published).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to count published questions: %w", err)
// // // 	}
// // // 	if err := query.Where("status = ?", models.QuestionStatusDraft).Count(&draft).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to count draft questions: %w", err)
// // // 	}
// // // 	if err := query.Where("status = ?", models.QuestionStatusArchived).Count(&archived).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to count archived questions: %w", err)
// // // 	}

// // // 	var avgMarks float64
// // // 	if err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Select("COALESCE(AVG(marks), 0)").
// // // 		Where("deleted_at IS NULL").
// // // 		Row().Scan(&avgMarks); err != nil {
// // // 		return nil, fmt.Errorf("failed to calculate average marks: %w", err)
// // // 	}

// // // 	// Get exam type breakdown
// // // 	examTypeStats, err := r.GetStatisticsByExamType(ctx, subjectID)
// // // 	if err != nil {
// // // 		return nil, fmt.Errorf("failed to get exam type statistics: %w", err)
// // // 	}

// // // 	return map[string]interface{}{
// // // 		"total_questions":       total,
// // // 		"published_count":       published,
// // // 		"draft_count":           draft,
// // // 		"archived_count":        archived,
// // // 		"average_marks":         avgMarks,
// // // 		"by_exam_type":          examTypeStats,
// // // 	}, nil
// // // }

// // // func (r *QuestionRepository) GetDetailedStatistics(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// // // 	for key, val := range filters {
// // // 		if val != nil && val != "" {
// // // 			query = query.Where(key+" = ?", val)
// // // 		}
// // // 	}

// // // 	var total int64
// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to count questions: %w", err)
// // // 	}

// // // 	var published, draft, archived int64
// // // 	query.Where("status = ?", models.QuestionStatusPublished).Count(&published)
// // // 	query.Where("status = ?", models.QuestionStatusDraft).Count(&draft)
// // // 	query.Where("status = ?", models.QuestionStatusArchived).Count(&archived)

// // // 	var avgMarks float64
// // // 	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

// // // 	// Breakdown by difficulty
// // // 	type DifficultyCount struct {
// // // 		Difficulty string
// // // 		Count      int
// // // 	}
// // // 	var difficultyCounts []DifficultyCount
// // // 	if err := query.Select("difficulty, count(*) as count").
// // // 		Group("difficulty").
// // // 		Scan(&difficultyCounts).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to get difficulty breakdown: %w", err)
// // // 	}

// // // 	byDifficulty := make(map[string]int)
// // // 	for _, dc := range difficultyCounts {
// // // 		byDifficulty[dc.Difficulty] = dc.Count
// // // 	}

// // // 	// Breakdown by question type
// // // 	type TypeCount struct {
// // // 		QuestionType string
// // // 		Count        int
// // // 	}
// // // 	var typeCounts []TypeCount
// // // 	if err := query.Select("question_type, count(*) as count").
// // // 		Group("question_type").
// // // 		Scan(&typeCounts).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to get question type breakdown: %w", err)
// // // 	}

// // // 	byType := make(map[string]int)
// // // 	for _, tc := range typeCounts {
// // // 		byType[tc.QuestionType] = tc.Count
// // // 	}

// // // 	// Breakdown by exam type
// // // 	type ExamTypeCount struct {
// // // 		ExamType string
// // // 		Count    int
// // // 	}
// // // 	var examTypeCounts []ExamTypeCount
// // // 	if err := query.Select("exam_type, count(*) as count").
// // // 		Group("exam_type").
// // // 		Scan(&examTypeCounts).Error; err != nil {
// // // 		return nil, fmt.Errorf("failed to get exam type breakdown: %w", err)
// // // 	}

// // // 	byExamType := make(map[string]int)
// // // 	for _, etc := range examTypeCounts {
// // // 		byExamType[etc.ExamType] = etc.Count
// // // 	}

// // // 	return map[string]interface{}{
// // // 		"total":              total,
// // // 		"published":          published,
// // // 		"draft":              draft,
// // // 		"archived":           archived,
// // // 		"average_marks":      avgMarks,
// // // 		"by_difficulty":      byDifficulty,
// // // 		"by_question_type":   byType,
// // // 		"by_exam_type":       byExamType,
// // // 	}, nil
// // // }

// // // // ============================================
// // // // ADDITIONAL HELPER FUNCTIONS
// // // // ============================================

// // // func (r *QuestionRepository) FindByQuestionText(ctx context.Context, text string, limit int) ([]models.QuestionBank, error) {
// // // 	if text == "" {
// // // 		return nil, errors.New("search text cannot be empty")
// // // 	}
// // // 	if limit < 1 || limit > 100 {
// // // 		limit = 20
// // // 	}

// // // 	var questions []models.QuestionBank
// // // 	err := r.db.WithContext(ctx).
// // // 		Where("question_text ILIKE ? AND deleted_at IS NULL", "%"+escapeWildcards(text)+"%").
// // // 		Limit(limit).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	if err != nil {
// // // 		return nil, fmt.Errorf("failed to search questions: %w", err)
// // // 	}
// // // 	return questions, nil
// // // }

// // // func (r *QuestionRepository) CountQuestionsBySubject(ctx context.Context, subjectID uuid.UUID) (int64, error) {
// // // 	var count int64
// // // 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Where("subject_id = ? AND deleted_at IS NULL", subjectID).
// // // 		Count(&count).Error
// // // 	if err != nil {
// // // 		return 0, fmt.Errorf("failed to count questions by subject: %w", err)
// // // 	}
// // // 	return count, nil
// // // }

// // // func (r *QuestionRepository) GetQuestionsByStatus(ctx context.Context, status models.QuestionStatus, page, limit int) ([]models.QuestionBank, int64, error) {
// // // 	if page < 1 {
// // // 		page = 1
// // // 	}
// // // 	if limit < 1 || limit > 100 {
// // // 		limit = 20
// // // 	}

// // // 	var questions []models.QuestionBank
// // // 	var total int64

// // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // 		Where("status = ? AND deleted_at IS NULL", status)

// // // 	if err := query.Count(&total).Error; err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to count questions by status: %w", err)
// // // 	}

// // // 	offset := (page - 1) * limit
// // // 	err := query.
// // // 		Offset(offset).
// // // 		Limit(limit).
// // // 		Order("created_at DESC").
// // // 		Find(&questions).Error
// // // 	if err != nil {
// // // 		return nil, 0, fmt.Errorf("failed to get questions by status: %w", err)
// // // 	}

// // // 	return questions, total, nil
// // // }

// // // func (r *QuestionRepository) GetRecentQuestions(ctx context.Context, subjectID uuid.UUID, limit int) ([]models.QuestionBank, error) {
// // // 	if limit < 1 || limit > 50 {
// // // 		limit = 10
// // // 	}

// // // 	var questions []models.QuestionBank
// // // 	query := r.db.WithContext(ctx).
// // // 		Where("deleted_at IS NULL").
// // // 		Order("created_at DESC").
// // // 		Limit(limit)

// // // 	if subjectID != uuid.Nil {
// // // 		query = query.Where("subject_id = ?", subjectID)
// // // 	}

// // // 	err := query.Find(&questions).Error
// // // 	if err != nil {
// // // 		return nil, fmt.Errorf("failed to get recent questions: %w", err)
// // // 	}
// // // 	return questions, nil
// // // }

// // // // ============================================
// // // // FUNCTION TO GET DB INSTANCE
// // // // ============================================

// // // func (r *QuestionRepository) GetDB() *gorm.DB {
// // // 	return r.db
// // // }

// // // // ============================================
// // // // SUMMARY STRUCTS
// // // // ============================================

// // // // QuestionTermSummary - Summary grouped by term with exam type breakdown (UPDATED)
// // // type QuestionTermSummary struct {
// // // 	TermID          string `json:"term_id"`
// // // 	TermName        string `json:"term_name"`
// // // 	TermNumber      int    `json:"term_number"`
// // // 	SessionName     string `json:"session_name"`
// // // 	QuestionCount   int    `json:"question_count"`
// // // 	WeeklyTestCount int    `json:"weekly_test_count"`
// // // 	MidTermCount    int    `json:"mid_term_count"`
// // // 	MainExamCount   int    `json:"main_exam_count"`
// // // 	PracticeCount   int    `json:"practice_count"`
// // // }




// // // // package repository

// // // // import (
// // // // 	"cbt-api/internal/models"
// // // // 	"context"
// // // // 	"errors"
// // // // 	"fmt"
// // // // 	"time"

// // // // 	"github.com/google/uuid"
// // // // 	"gorm.io/gorm"
// // // // )

// // // // type QuestionRepository struct {
// // // // 	db *gorm.DB
// // // // }

// // // // func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
// // // // 	return &QuestionRepository{db: db}
// // // // }

// // // // // ============================================
// // // // // CRUD OPERATIONS - FIXED WITH CONTEXT
// // // // // ============================================

// // // // func (r *QuestionRepository) Create(ctx context.Context, question *models.QuestionBank) error {
// // // // 	if question == nil {
// // // // 		return errors.New("question cannot be nil")
// // // // 	}
// // // // 	return r.db.WithContext(ctx).Create(question).Error
// // // // }

// // // // func (r *QuestionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.QuestionBank, error) {
// // // // 	var q models.QuestionBank
// // // // 	err := r.db.WithContext(ctx).
// // // // 		Where("id = ? AND deleted_at IS NULL", id).
// // // // 		First(&q).Error
// // // // 	if err != nil {
// // // // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // // // 			return nil, fmt.Errorf("question with ID %s not found", id)
// // // // 		}
// // // // 		return nil, fmt.Errorf("failed to find question: %w", err)
// // // // 	}
// // // // 	return &q, nil
// // // // }

// // // // func (r *QuestionRepository) Update(ctx context.Context, question *models.QuestionBank) error {
// // // // 	if question == nil {
// // // // 		return errors.New("question cannot be nil")
// // // // 	}
// // // // 	return r.db.WithContext(ctx).Save(question).Error
// // // // }

// // // // func (r *QuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
// // // // 	// Start transaction to handle cascading deletes
// // // // 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// // // // 		// 1. Delete question tag mappings
// // // // 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// // // // 		}

// // // // 		// 2. Delete exam questions
// // // // 		if err := tx.Where("question_id = ?", id).Delete(&models.ExamQuestion{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete exam questions: %w", err)
// // // // 		}

// // // // 		// 3. Delete question attachments
// // // // 		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete question attachments: %w", err)
// // // // 		}

// // // // 		// 4. Soft delete the question
// // // // 		if err := tx.Where("id = ?", id).Delete(&models.QuestionBank{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete question: %w", err)
// // // // 		}

// // // // 		return nil
// // // // 	})
// // // // }

// // // // // ============================================
// // // // // QUERY / FILTER OPERATIONS - FIXED
// // // // // ============================================

// // // // func (r *QuestionRepository) ListBySubject(ctx context.Context, subjectID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
// // // // 	if page < 1 {
// // // // 		page = 1
// // // // 	}
// // // // 	if limit < 1 || limit > 100 {
// // // // 		limit = 20
// // // // 	}
	
// // // // 	offset := (page - 1) * limit
// // // // 	var questions []models.QuestionBank
// // // // 	var total int64

// // // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // // 		Where("subject_id = ? AND deleted_at IS NULL", subjectID)

// // // // 	if err := query.Count(&total).Error; err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to count questions: %w", err)
// // // // 	}

// // // // 	err := query.
// // // // 		Offset(offset).
// // // // 		Limit(limit).
// // // // 		Order("created_at DESC").
// // // // 		Find(&questions).Error
// // // // 	if err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to list questions: %w", err)
// // // // 	}

// // // // 	return questions, total, nil
// // // // }

// // // // func (r *QuestionRepository) Filter(ctx context.Context, params map[string]interface{}, page, limit int) ([]models.QuestionBank, int64, error) {
// // // // 	if page < 1 {
// // // // 		page = 1
// // // // 	}
// // // // 	if limit < 1 || limit > 100 {
// // // // 		limit = 20
// // // // 	}

// // // // 	var questions []models.QuestionBank
// // // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// // // // 	// Apply filters with parameterized queries to prevent SQL injection
// // // // 	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
// // // // 		query = query.Where("subject_id = ?", subjectID)
// // // // 	}
// // // // 	if schoolID, ok := params["school_id"]; ok && schoolID != "" {
// // // // 		query = query.Where("school_id = ?", schoolID)
// // // // 	}
// // // // 	if classLevelID, ok := params["class_level_id"]; ok && classLevelID != "" {
// // // // 		query = query.Where("class_level_id = ?", classLevelID)
// // // // 	}
// // // // 	if sessionID, ok := params["session_id"]; ok && sessionID != "" {
// // // // 		query = query.Where("session_id = ?", sessionID)
// // // // 	}
// // // // 	if termID, ok := params["term_id"]; ok && termID != "" {
// // // // 		query = query.Where("term_id = ?", termID)
// // // // 	}
// // // // 	if topic, ok := params["topic"]; ok && topic != "" {
// // // // 		query = query.Where("topic ILIKE ?", "%"+topic.(string)+"%")
// // // // 	}
// // // // 	if difficulty, ok := params["difficulty"]; ok && difficulty != "" {
// // // // 		query = query.Where("difficulty = ?", difficulty)
// // // // 	}
// // // // 	if bloomLevel, ok := params["bloom_level"]; ok && bloomLevel != "" {
// // // // 		query = query.Where("bloom_level = ?", bloomLevel)
// // // // 	}
// // // // 	if questionType, ok := params["question_type"]; ok && questionType != "" {
// // // // 		query = query.Where("question_type = ?", questionType)
// // // // 	}
// // // // 	if status, ok := params["status"]; ok && status != "" {
// // // // 		query = query.Where("status = ?", status)
// // // // 	}
// // // // 	if curriculumType, ok := params["curriculum_type"]; ok && curriculumType != "" {
// // // // 		query = query.Where("curriculum_type = ?", curriculumType)
// // // // 	}
// // // // 	if sourceType, ok := params["source_type"]; ok && sourceType != "" {
// // // // 		query = query.Where("source_type = ?", sourceType)
// // // // 	}
// // // // 	if externalID, ok := params["external_id"]; ok && externalID != "" {
// // // // 		query = query.Where("external_id = ?", externalID)
// // // // 	}
	
// // // // 	// Safe search with escaped wildcards
// // // // 	if search, ok := params["search"]; ok && search != "" {
// // // // 		searchStr := "%" + escapeWildcards(search.(string)) + "%"
// // // // 		query = query.Where("question_text ILIKE ?", searchStr)
// // // // 	}

// // // // 	var total int64
// // // // 	if err := query.Count(&total).Error; err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to count filtered questions: %w", err)
// // // // 	}

// // // // 	offset := (page - 1) * limit
// // // // 	err := query.
// // // // 		Offset(offset).
// // // // 		Limit(limit).
// // // // 		Order("created_at DESC").
// // // // 		Find(&questions).Error
// // // // 	if err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to filter questions: %w", err)
// // // // 	}

// // // // 	return questions, total, nil
// // // // }

// // // // // escapeWildcards escapes wildcard characters in search strings
// // // // func escapeWildcards(s string) string {
// // // // 	// Replace % and _ with escaped versions
// // // // 	result := ""
// // // // 	for _, c := range s {
// // // // 		if c == '%' || c == '_' {
// // // // 			result += "\\" + string(c)
// // // // 		} else {
// // // // 			result += string(c)
// // // // 		}
// // // // 	}
// // // // 	return result
// // // // }

// // // // func (r *QuestionRepository) FindByTag(ctx context.Context, tagName string, page, limit int) ([]models.QuestionBank, int64, error) {
// // // // 	if page < 1 {
// // // // 		page = 1
// // // // 	}
// // // // 	if limit < 1 || limit > 100 {
// // // // 		limit = 20
// // // // 	}

// // // // 	var questions []models.QuestionBank
// // // // 	query := r.db.WithContext(ctx).
// // // // 		Joins("JOIN question_tag_mappings ON question_tag_mappings.question_id = question_bank.id").
// // // // 		Joins("JOIN tags ON tags.id = question_tag_mappings.tag_id").
// // // // 		Where("tags.name = ? AND question_bank.deleted_at IS NULL", tagName)

// // // // 	var total int64
// // // // 	if err := query.Count(&total).Error; err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to count questions by tag: %w", err)
// // // // 	}

// // // // 	offset := (page - 1) * limit
// // // // 	err := query.
// // // // 		Offset(offset).
// // // // 		Limit(limit).
// // // // 		Order("question_bank.created_at DESC").
// // // // 		Find(&questions).Error
// // // // 	if err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to find questions by tag: %w", err)
// // // // 	}

// // // // 	return questions, total, nil
// // // // }

// // // // func (r *QuestionRepository) FindByExternalID(ctx context.Context, schoolID, sessionID uuid.UUID, externalID string) (*models.QuestionBank, error) {
// // // // 	if externalID == "" {
// // // // 		return nil, errors.New("external ID cannot be empty")
// // // // 	}

// // // // 	var q models.QuestionBank
// // // // 	query := r.db.WithContext(ctx).
// // // // 		Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)
	
// // // // 	if sessionID != uuid.Nil {
// // // // 		query = query.Where("session_id = ?", sessionID)
// // // // 	} else {
// // // // 		query = query.Where("session_id IS NULL")
// // // // 	}

// // // // 	err := query.First(&q).Error
// // // // 	if err != nil {
// // // // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // // // 			return nil, nil // Not found is not an error
// // // // 		}
// // // // 		return nil, fmt.Errorf("failed to find question by external ID: %w", err)
// // // // 	}
// // // // 	return &q, nil
// // // // }

// // // // // ============================================
// // // // // BULK / BATCH OPERATIONS - FIXED WITH CONTEXT
// // // // // ============================================

// // // // func (r *QuestionRepository) BulkCreate(ctx context.Context, questions []models.QuestionBank) error {
// // // // 	if len(questions) == 0 {
// // // // 		return errors.New("no questions to create")
// // // // 	}
	
// // // // 	// Create in batches of 100 for performance
// // // // 	return r.db.WithContext(ctx).CreateInBatches(questions, 100).Error
// // // // }

// // // // func (r *QuestionRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
// // // // 	if len(ids) == 0 {
// // // // 		return errors.New("no question IDs provided")
// // // // 	}

// // // // 	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
// // // // 		// Delete related records first
// // // // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionTagMapping{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete question tag mappings: %w", err)
// // // // 		}
// // // // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.ExamQuestion{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete exam questions: %w", err)
// // // // 		}
// // // // 		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete question attachments: %w", err)
// // // // 		}
		
// // // // 		// Soft delete questions
// // // // 		if err := tx.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error; err != nil {
// // // // 			return fmt.Errorf("failed to delete questions: %w", err)
// // // // 		}
// // // // 		return nil
// // // // 	})
// // // // }

// // // // func (r *QuestionRepository) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status string) error {
// // // // 	if len(ids) == 0 {
// // // // 		return errors.New("no question IDs provided")
// // // // 	}
// // // // 	if status == "" {
// // // // 		return errors.New("status cannot be empty")
// // // // 	}

// // // // 	result := r.db.WithContext(ctx).
// // // // 		Model(&models.QuestionBank{}).
// // // // 		Where("id IN ?", ids).
// // // // 		Update("status", models.QuestionStatus(status))

// // // // 	if result.Error != nil {
// // // // 		return fmt.Errorf("failed to update status: %w", result.Error)
// // // // 	}
	
// // // // 	if result.RowsAffected == 0 {
// // // // 		return errors.New("no questions found to update")
// // // // 	}
	
// // // // 	return nil
// // // // }

// // // // // ============================================
// // // // // VERSIONING OPERATIONS - FIXED
// // // // // ============================================

// // // // func (r *QuestionRepository) CreateNewVersion(ctx context.Context, old *models.QuestionBank, updates map[string]interface{}) (uuid.UUID, error) {
// // // // 	if old == nil {
// // // // 		return uuid.Nil, errors.New("old question cannot be nil")
// // // // 	}
// // // // 	if len(updates) == 0 {
// // // // 		return uuid.Nil, errors.New("no updates provided")
// // // // 	}

// // // // 	// Create new version with incremented version
// // // // 	newQ := *old
// // // // 	newQ.ID = uuid.New()
// // // // 	newQ.Version = old.Version + 1
// // // // 	newQ.ParentID = &old.ID
// // // // 	newQ.CreatedAt = time.Now()
// // // // 	newQ.UpdatedAt = time.Now()

// // // // 	// Apply updates with proper type assertions
// // // // 	for k, v := range updates {
// // // // 		if err := r.applyUpdate(&newQ, k, v); err != nil {
// // // // 			return uuid.Nil, fmt.Errorf("failed to apply update for field %s: %w", k, err)
// // // // 		}
// // // // 	}

// // // // 	err := r.db.WithContext(ctx).Create(&newQ).Error
// // // // 	if err != nil {
// // // // 		return uuid.Nil, fmt.Errorf("failed to create new version: %w", err)
// // // // 	}
	
// // // // 	return newQ.ID, nil
// // // // }

// // // // func (r *QuestionRepository) applyUpdate(q *models.QuestionBank, key string, value interface{}) error {
// // // // 	switch key {
// // // // 	case "topic":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.Topic = v
// // // // 		}
// // // // 	case "sub_topic":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.SubTopic = v
// // // // 		}
// // // // 	case "learning_objective":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.LearningObjective = v
// // // // 		}
// // // // 	case "question_text":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.QuestionText = v
// // // // 		}
// // // // 	case "question_type":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.QuestionType = models.QuestionType(v)
// // // // 		}
// // // // 	case "difficulty":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.Difficulty = models.DifficultyLevel(v)
// // // // 		}
// // // // 	case "bloom_level":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.BloomLevel = models.BloomTaxonomy(v)
// // // // 		}
// // // // 	case "options":
// // // // 		if v, ok := value.(models.OptionStorage); ok {
// // // // 			q.Options = v
// // // // 		} else {
// // // // 			return fmt.Errorf("options must be of type OptionStorage, got %T", value)
// // // // 		}
// // // // 	case "correct_option_keys":
// // // // 		if v, ok := value.([]string); ok {
// // // // 			q.CorrectOptionKeys = v
// // // // 		}
// // // // 	case "correct_answer":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.CorrectAnswer = v
// // // // 		}
// // // // 	case "rubric":
// // // // 		if v, ok := value.(models.RubricStorage); ok {
// // // // 			q.Rubric = v
// // // // 		} else {
// // // // 			return fmt.Errorf("rubric must be of type RubricStorage, got %T", value)
// // // // 		}
// // // // 	case "tags":
// // // // 		if v, ok := value.(models.TagStorage); ok {
// // // // 			q.Tags = v
// // // // 		} else {
// // // // 			return fmt.Errorf("tags must be of type TagStorage, got %T", value)
// // // // 		}
// // // // 	case "explanation":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.Explanation = v
// // // // 		}
// // // // 	case "marks":
// // // // 		if v, ok := value.(int); ok {
// // // // 			q.Marks = v
// // // // 		}
// // // // 	case "negative_marks":
// // // // 		if v, ok := value.(float64); ok {
// // // // 			q.NegativeMarks = v
// // // // 		}
// // // // 	case "time_limit_seconds":
// // // // 		if v, ok := value.(*int); ok {
// // // // 			q.TimeLimitSeconds = v
// // // // 		} else if v, ok := value.(int); ok {
// // // // 			q.TimeLimitSeconds = &v
// // // // 		}
// // // // 	case "order":
// // // // 		if v, ok := value.(int); ok {
// // // // 			q.Order = v
// // // // 		}
// // // // 	case "is_required":
// // // // 		if v, ok := value.(bool); ok {
// // // // 			q.IsRequired = v
// // // // 		}
// // // // 	case "updated_by":
// // // // 		if v, ok := value.(uuid.UUID); ok {
// // // // 			q.UpdatedBy = v
// // // // 		}
// // // // 	case "curriculum_type":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.CurriculumType = v
// // // // 		}
// // // // 	case "source_type":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.SourceType = v
// // // // 		}
// // // // 	case "status":
// // // // 		if v, ok := value.(string); ok {
// // // // 			q.Status = models.QuestionStatus(v)
// // // // 		}
// // // // 	default:
// // // // 		return fmt.Errorf("unknown field: %s", key)
// // // // 	}
// // // // 	return nil
// // // // }

// // // // // ============================================
// // // // // TAG OPERATIONS - FIXED
// // // // // ============================================

// // // // func (r *QuestionRepository) CreateTag(ctx context.Context, tag *models.Tag) error {
// // // // 	if tag == nil {
// // // // 		return errors.New("tag cannot be nil")
// // // // 	}
// // // // 	return r.db.WithContext(ctx).Create(tag).Error
// // // // }

// // // // func (r *QuestionRepository) FindTagByName(ctx context.Context, name string) (*models.Tag, error) {
// // // // 	if name == "" {
// // // // 		return nil, errors.New("tag name cannot be empty")
// // // // 	}

// // // // 	var tag models.Tag
// // // // 	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
// // // // 	if err != nil {
// // // // 		if errors.Is(err, gorm.ErrRecordNotFound) {
// // // // 			return nil, nil
// // // // 		}
// // // // 		return nil, fmt.Errorf("failed to find tag by name: %w", err)
// // // // 	}
// // // // 	return &tag, nil
// // // // }

// // // // func (r *QuestionRepository) ListTags(ctx context.Context) ([]models.Tag, error) {
// // // // 	var tags []models.Tag
// // // // 	err := r.db.WithContext(ctx).
// // // // 		Where("deleted_at IS NULL").
// // // // 		Order("name ASC").
// // // // 		Find(&tags).Error
// // // // 	if err != nil {
// // // // 		return nil, fmt.Errorf("failed to list tags: %w", err)
// // // // 	}
// // // // 	return tags, nil
// // // // }

// // // // func (r *QuestionRepository) ListTagsPaginated(ctx context.Context, page, limit int) ([]models.Tag, int64, error) {
// // // // 	if page < 1 {
// // // // 		page = 1
// // // // 	}
// // // // 	if limit < 1 || limit > 100 {
// // // // 		limit = 20
// // // // 	}

// // // // 	var tags []models.Tag
// // // // 	var total int64
	
// // // // 	query := r.db.WithContext(ctx).Model(&models.Tag{}).Where("deleted_at IS NULL")
	
// // // // 	if err := query.Count(&total).Error; err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to count tags: %w", err)
// // // // 	}

// // // // 	offset := (page - 1) * limit
// // // // 	err := query.
// // // // 		Offset(offset).
// // // // 		Limit(limit).
// // // // 		Order("name ASC").
// // // // 		Find(&tags).Error
// // // // 	if err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
// // // // 	}

// // // // 	return tags, total, nil
// // // // }

// // // // func (r *QuestionRepository) AttachTags(ctx context.Context, questionID uuid.UUID, tagIDs []uuid.UUID) error {
// // // // 	if questionID == uuid.Nil {
// // // // 		return errors.New("question ID cannot be nil")
// // // // 	}
// // // // 	if len(tagIDs) == 0 {
// // // // 		return nil
// // // // 	}

// // // // 	for _, tid := range tagIDs {
// // // // 		mapping := models.QuestionTagMapping{
// // // // 			ID:         uuid.New(),
// // // // 			QuestionID: questionID,
// // // // 			TagID:      tid,
// // // // 		}
// // // // 		if err := r.db.WithContext(ctx).Create(&mapping).Error; err != nil {
// // // // 			return fmt.Errorf("failed to attach tag %s to question %s: %w", tid, questionID, err)
// // // // 		}
// // // // 	}
// // // // 	return nil
// // // // }

// // // // func (r *QuestionRepository) DetachTags(ctx context.Context, questionID uuid.UUID, tagIDs []uuid.UUID) error {
// // // // 	if questionID == uuid.Nil {
// // // // 		return errors.New("question ID cannot be nil")
// // // // 	}
// // // // 	if len(tagIDs) == 0 {
// // // // 		return nil
// // // // 	}

// // // // 	result := r.db.WithContext(ctx).
// // // // 		Where("question_id = ? AND tag_id IN ?", questionID, tagIDs).
// // // // 		Delete(&models.QuestionTagMapping{})
	
// // // // 	if result.Error != nil {
// // // // 		return fmt.Errorf("failed to detach tags: %w", result.Error)
// // // // 	}
// // // // 	return nil
// // // // }

// // // // // ============================================
// // // // // STATISTICS OPERATIONS - FIXED
// // // // // ============================================

// // // // func (r *QuestionRepository) GetStatistics(ctx context.Context, subjectID uuid.UUID) (map[string]interface{}, error) {
// // // // 	var total int64
// // // // 	var published, draft, archived int64

// // // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
// // // // 	if subjectID != uuid.Nil {
// // // // 		query = query.Where("subject_id = ?", subjectID)
// // // // 	}

// // // // 	// Count total
// // // // 	if err := query.Count(&total).Error; err != nil {
// // // // 		return nil, fmt.Errorf("failed to count total questions: %w", err)
// // // // 	}

// // // // 	// Count by status
// // // // 	if err := query.Where("status = ?", models.QuestionStatusPublished).Count(&published).Error; err != nil {
// // // // 		return nil, fmt.Errorf("failed to count published questions: %w", err)
// // // // 	}
// // // // 	if err := query.Where("status = ?", models.QuestionStatusDraft).Count(&draft).Error; err != nil {
// // // // 		return nil, fmt.Errorf("failed to count draft questions: %w", err)
// // // // 	}
// // // // 	if err := query.Where("status = ?", models.QuestionStatusArchived).Count(&archived).Error; err != nil {
// // // // 		return nil, fmt.Errorf("failed to count archived questions: %w", err)
// // // // 	}

// // // // 	// Calculate average marks
// // // // 	var avgMarks float64
// // // // 	if err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // // 		Select("COALESCE(AVG(marks), 0)").
// // // // 		Where("deleted_at IS NULL").
// // // // 		Row().Scan(&avgMarks); err != nil {
// // // // 		return nil, fmt.Errorf("failed to calculate average marks: %w", err)
// // // // 	}

// // // // 	return map[string]interface{}{
// // // // 		"total_questions": total,
// // // // 		"published_count": published,
// // // // 		"draft_count":     draft,
// // // // 		"archived_count":  archived,
// // // // 		"average_marks":   avgMarks,
// // // // 	}, nil
// // // // }

// // // // func (r *QuestionRepository) GetDetailedStatistics(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
// // // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
	
// // // // 	// Apply filters
// // // // 	for key, val := range filters {
// // // // 		if val != nil && val != "" {
// // // // 			query = query.Where(key+" = ?", val)
// // // // 		}
// // // // 	}

// // // // 	var total int64
// // // // 	if err := query.Count(&total).Error; err != nil {
// // // // 		return nil, fmt.Errorf("failed to count questions: %w", err)
// // // // 	}

// // // // 	// Count by status
// // // // 	var published, draft, archived int64
// // // // 	query.Where("status = ?", models.QuestionStatusPublished).Count(&published)
// // // // 	query.Where("status = ?", models.QuestionStatusDraft).Count(&draft)
// // // // 	query.Where("status = ?", models.QuestionStatusArchived).Count(&archived)

// // // // 	// Calculate average marks
// // // // 	var avgMarks float64
// // // // 	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

// // // // 	// Get breakdown by difficulty
// // // // 	type DifficultyCount struct {
// // // // 		Difficulty string
// // // // 		Count      int
// // // // 	}
// // // // 	var difficultyCounts []DifficultyCount
// // // // 	if err := query.Select("difficulty, count(*) as count").
// // // // 		Group("difficulty").
// // // // 		Scan(&difficultyCounts).Error; err != nil {
// // // // 		return nil, fmt.Errorf("failed to get difficulty breakdown: %w", err)
// // // // 	}
	
// // // // 	byDifficulty := make(map[string]int)
// // // // 	for _, dc := range difficultyCounts {
// // // // 		byDifficulty[dc.Difficulty] = dc.Count
// // // // 	}

// // // // 	// Get breakdown by question type
// // // // 	type TypeCount struct {
// // // // 		QuestionType string
// // // // 		Count        int
// // // // 	}
// // // // 	var typeCounts []TypeCount
// // // // 	if err := query.Select("question_type, count(*) as count").
// // // // 		Group("question_type").
// // // // 		Scan(&typeCounts).Error; err != nil {
// // // // 		return nil, fmt.Errorf("failed to get question type breakdown: %w", err)
// // // // 	}
	
// // // // 	byType := make(map[string]int)
// // // // 	for _, tc := range typeCounts {
// // // // 		byType[tc.QuestionType] = tc.Count
// // // // 	}

// // // // 	return map[string]interface{}{
// // // // 		"total":          total,
// // // // 		"published":      published,
// // // // 		"draft":          draft,
// // // // 		"archived":       archived,
// // // // 		"average_marks":  avgMarks,
// // // // 		"by_difficulty":  byDifficulty,
// // // // 		"by_question_type": byType,
// // // // 	}, nil
// // // // }

// // // // // ============================================
// // // // // ADDITIONAL HELPER FUNCTIONS
// // // // // ============================================

// // // // func (r *QuestionRepository) FindByQuestionText(ctx context.Context, text string, limit int) ([]models.QuestionBank, error) {
// // // // 	if text == "" {
// // // // 		return nil, errors.New("search text cannot be empty")
// // // // 	}
// // // // 	if limit < 1 || limit > 100 {
// // // // 		limit = 20
// // // // 	}

// // // // 	var questions []models.QuestionBank
// // // // 	err := r.db.WithContext(ctx).
// // // // 		Where("question_text ILIKE ? AND deleted_at IS NULL", "%"+escapeWildcards(text)+"%").
// // // // 		Limit(limit).
// // // // 		Order("created_at DESC").
// // // // 		Find(&questions).Error
// // // // 	if err != nil {
// // // // 		return nil, fmt.Errorf("failed to search questions: %w", err)
// // // // 	}
// // // // 	return questions, nil
// // // // }

// // // // func (r *QuestionRepository) CountQuestionsBySubject(ctx context.Context, subjectID uuid.UUID) (int64, error) {
// // // // 	var count int64
// // // // 	err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // // 		Where("subject_id = ? AND deleted_at IS NULL", subjectID).
// // // // 		Count(&count).Error
// // // // 	if err != nil {
// // // // 		return 0, fmt.Errorf("failed to count questions by subject: %w", err)
// // // // 	}
// // // // 	return count, nil
// // // // }

// // // // func (r *QuestionRepository) GetQuestionsByStatus(ctx context.Context, status models.QuestionStatus, page, limit int) ([]models.QuestionBank, int64, error) {
// // // // 	if page < 1 {
// // // // 		page = 1
// // // // 	}
// // // // 	if limit < 1 || limit > 100 {
// // // // 		limit = 20
// // // // 	}

// // // // 	var questions []models.QuestionBank
// // // // 	var total int64

// // // // 	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // // 		Where("status = ? AND deleted_at IS NULL", status)

// // // // 	if err := query.Count(&total).Error; err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to count questions by status: %w", err)
// // // // 	}

// // // // 	offset := (page - 1) * limit
// // // // 	err := query.
// // // // 		Offset(offset).
// // // // 		Limit(limit).
// // // // 		Order("created_at DESC").
// // // // 		Find(&questions).Error
// // // // 	if err != nil {
// // // // 		return nil, 0, fmt.Errorf("failed to get questions by status: %w", err)
// // // // 	}

// // // // 	return questions, total, nil
// // // // }

// // // // func (r *QuestionRepository) GetRecentQuestions(ctx context.Context, subjectID uuid.UUID, limit int) ([]models.QuestionBank, error) {
// // // // 	if limit < 1 || limit > 50 {
// // // // 		limit = 10
// // // // 	}

// // // // 	var questions []models.QuestionBank
// // // // 	query := r.db.WithContext(ctx).
// // // // 		Where("deleted_at IS NULL").
// // // // 		Order("created_at DESC").
// // // // 		Limit(limit)

// // // // 	if subjectID != uuid.Nil {
// // // // 		query = query.Where("subject_id = ?", subjectID)
// // // // 	}

// // // // 	err := query.Find(&questions).Error
// // // // 	if err != nil {
// // // // 		return nil, fmt.Errorf("failed to get recent questions: %w", err)
// // // // 	}
// // // // 	return questions, nil
// // // // }

// // // // // ============================================
// // // // // FUNCTION TO GET DB INSTANCE
// // // // // ============================================

// // // // func (r *QuestionRepository) GetDB() *gorm.DB {
// // // // 	return r.db
// // // // }


// // // // // ============================================
// // // // // NEW CONTEXT-AWARE QUERY METHODS
// // // // // ============================================

// // // // // FindByTerm - Get questions for a specific term
// // // // func (r *QuestionRepository) FindByTerm(ctx context.Context, subjectID, termID uuid.UUID) ([]models.QuestionBank, error) {
// // // //     var questions []models.QuestionBank
// // // //     err := r.db.WithContext(ctx).
// // // //         Where("subject_id = ? AND term_id = ? AND deleted_at IS NULL", subjectID, termID).
// // // //         Order("created_at DESC").
// // // //         Find(&questions).Error
// // // //     return questions, err
// // // // }

// // // // // FindBySession - Get questions for a specific session
// // // // func (r *QuestionRepository) FindBySession(ctx context.Context, subjectID, sessionID uuid.UUID) ([]models.QuestionBank, error) {
// // // //     var questions []models.QuestionBank
// // // //     err := r.db.WithContext(ctx).
// // // //         Where("subject_id = ? AND session_id = ? AND deleted_at IS NULL", subjectID, sessionID).
// // // //         Order("created_at DESC").
// // // //         Find(&questions).Error
// // // //     return questions, err
// // // // }

// // // // // FindBySchoolAndSession - Get questions by school and session
// // // // func (r *QuestionRepository) FindBySchoolAndSession(ctx context.Context, schoolID, sessionID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
// // // //     var questions []models.QuestionBank
// // // //     var total int64
    
// // // //     query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // //         Where("school_id = ? AND session_id = ? AND deleted_at IS NULL", schoolID, sessionID)
    
// // // //     if err := query.Count(&total).Error; err != nil {
// // // //         return nil, 0, err
// // // //     }
    
// // // //     offset := (page - 1) * limit
// // // //     err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&questions).Error
// // // //     return questions, total, err
// // // // }

// // // // // CountQuestionsByTerm - Count questions in a term
// // // // func (r *QuestionRepository) CountQuestionsByTerm(ctx context.Context, termID uuid.UUID) (int64, error) {
// // // //     var count int64
// // // //     err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
// // // //         Where("term_id = ? AND deleted_at IS NULL", termID).
// // // //         Count(&count).Error
// // // //     return count, err
// // // // }

// // // // // GetQuestionContextSummary - Get summary grouped by term
// // // // func (r *QuestionRepository) GetQuestionContextSummary(ctx context.Context, subjectID uuid.UUID) ([]QuestionTermSummary, error) {
// // // //     var results []QuestionTermSummary
    
// // // //     query := `
// // // //         SELECT 
// // // //             t.id as term_id,
// // // //             t.name as term_name,
// // // //             t.term_number,
// // // //             s.name as session_name,
// // // //             COUNT(q.id) as question_count
// // // //         FROM question_bank q
// // // //         JOIN terms t ON t.id = q.term_id
// // // //         JOIN academic_sessions s ON s.id = q.session_id
// // // //         WHERE q.subject_id = ? 
// // // //             AND q.deleted_at IS NULL
// // // //         GROUP BY t.id, t.name, t.term_number, s.name
// // // //         ORDER BY t.term_number ASC
// // // //     `
    
// // // //     err := r.db.WithContext(ctx).Raw(query, subjectID).Scan(&results).Error
// // // //     return results, err
// // // // }

// // // // // QuestionTermSummary - Summary struct
// // // // type QuestionTermSummary struct {
// // // //     TermID        string `json:"term_id"`
// // // //     TermName      string `json:"term_name"`
// // // //     TermNumber    int    `json:"term_number"`
// // // //     SessionName   string `json:"session_name"`
// // // //     QuestionCount int    `json:"question_count"`
// // // // }