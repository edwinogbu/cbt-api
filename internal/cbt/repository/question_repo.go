package repository

import (
	"cbt-api/internal/models"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// ============================================
// CRUD OPERATIONS - FIXED WITH CONTEXT
// ============================================

func (r *QuestionRepository) Create(ctx context.Context, question *models.QuestionBank) error {
	if question == nil {
		return errors.New("question cannot be nil")
	}
	return r.db.WithContext(ctx).Create(question).Error
}

func (r *QuestionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.QuestionBank, error) {
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

func (r *QuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Start transaction to handle cascading deletes
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Delete question tag mappings
		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionTagMapping{}).Error; err != nil {
			return fmt.Errorf("failed to delete question tag mappings: %w", err)
		}

		// 2. Delete exam questions
		if err := tx.Where("question_id = ?", id).Delete(&models.ExamQuestion{}).Error; err != nil {
			return fmt.Errorf("failed to delete exam questions: %w", err)
		}

		// 3. Delete question attachments
		if err := tx.Where("question_id = ?", id).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
			return fmt.Errorf("failed to delete question attachments: %w", err)
		}

		// 4. Soft delete the question
		if err := tx.Where("id = ?", id).Delete(&models.QuestionBank{}).Error; err != nil {
			return fmt.Errorf("failed to delete question: %w", err)
		}

		return nil
	})
}

// ============================================
// QUERY / FILTER OPERATIONS - FIXED
// ============================================

func (r *QuestionRepository) ListBySubject(ctx context.Context, subjectID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
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

	// Apply filters with parameterized queries to prevent SQL injection
	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}
	if schoolID, ok := params["school_id"]; ok && schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	if classLevelID, ok := params["class_level_id"]; ok && classLevelID != "" {
		query = query.Where("class_level_id = ?", classLevelID)
	}
	if sessionID, ok := params["session_id"]; ok && sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if termID, ok := params["term_id"]; ok && termID != "" {
		query = query.Where("term_id = ?", termID)
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
	
	// Safe search with escaped wildcards
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

// escapeWildcards escapes wildcard characters in search strings
func escapeWildcards(s string) string {
	// Replace % and _ with escaped versions
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

func (r *QuestionRepository) FindByExternalID(ctx context.Context, schoolID, sessionID uuid.UUID, externalID string) (*models.QuestionBank, error) {
	if externalID == "" {
		return nil, errors.New("external ID cannot be empty")
	}

	var q models.QuestionBank
	query := r.db.WithContext(ctx).
		Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)
	
	if sessionID != uuid.Nil {
		query = query.Where("session_id = ?", sessionID)
	} else {
		query = query.Where("session_id IS NULL")
	}

	err := query.First(&q).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("failed to find question by external ID: %w", err)
	}
	return &q, nil
}

// ============================================
// BULK / BATCH OPERATIONS - FIXED WITH CONTEXT
// ============================================

func (r *QuestionRepository) BulkCreate(ctx context.Context, questions []models.QuestionBank) error {
	if len(questions) == 0 {
		return errors.New("no questions to create")
	}
	
	// Create in batches of 100 for performance
	return r.db.WithContext(ctx).CreateInBatches(questions, 100).Error
}

func (r *QuestionRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return errors.New("no question IDs provided")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete related records first
		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionTagMapping{}).Error; err != nil {
			return fmt.Errorf("failed to delete question tag mappings: %w", err)
		}
		if err := tx.Where("question_id IN ?", ids).Delete(&models.ExamQuestion{}).Error; err != nil {
			return fmt.Errorf("failed to delete exam questions: %w", err)
		}
		if err := tx.Where("question_id IN ?", ids).Delete(&models.QuestionBankAttachment{}).Error; err != nil {
			return fmt.Errorf("failed to delete question attachments: %w", err)
		}
		
		// Soft delete questions
		if err := tx.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error; err != nil {
			return fmt.Errorf("failed to delete questions: %w", err)
		}
		return nil
	})
}

func (r *QuestionRepository) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status string) error {
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
// VERSIONING OPERATIONS - FIXED
// ============================================

func (r *QuestionRepository) CreateNewVersion(ctx context.Context, old *models.QuestionBank, updates map[string]interface{}) (uuid.UUID, error) {
	if old == nil {
		return uuid.Nil, errors.New("old question cannot be nil")
	}
	if len(updates) == 0 {
		return uuid.Nil, errors.New("no updates provided")
	}

	// Create new version with incremented version
	newQ := *old
	newQ.ID = uuid.New()
	newQ.Version = old.Version + 1
	newQ.ParentID = &old.ID
	newQ.CreatedAt = time.Now()
	newQ.UpdatedAt = time.Now()

	// Apply updates with proper type assertions
	for k, v := range updates {
		if err := r.applyUpdate(&newQ, k, v); err != nil {
			return uuid.Nil, fmt.Errorf("failed to apply update for field %s: %w", k, err)
		}
	}

	err := r.db.WithContext(ctx).Create(&newQ).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create new version: %w", err)
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
		if v, ok := value.(uuid.UUID); ok {
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
// TAG OPERATIONS - FIXED
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

func (r *QuestionRepository) AttachTags(ctx context.Context, questionID uuid.UUID, tagIDs []uuid.UUID) error {
	if questionID == uuid.Nil {
		return errors.New("question ID cannot be nil")
	}
	if len(tagIDs) == 0 {
		return nil
	}

	for _, tid := range tagIDs {
		mapping := models.QuestionTagMapping{
			ID:         uuid.New(),
			QuestionID: questionID,
			TagID:      tid,
		}
		if err := r.db.WithContext(ctx).Create(&mapping).Error; err != nil {
			return fmt.Errorf("failed to attach tag %s to question %s: %w", tid, questionID, err)
		}
	}
	return nil
}

func (r *QuestionRepository) DetachTags(ctx context.Context, questionID uuid.UUID, tagIDs []uuid.UUID) error {
	if questionID == uuid.Nil {
		return errors.New("question ID cannot be nil")
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
// STATISTICS OPERATIONS - FIXED
// ============================================

func (r *QuestionRepository) GetStatistics(ctx context.Context, subjectID uuid.UUID) (map[string]interface{}, error) {
	var total int64
	var published, draft, archived int64

	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
	if subjectID != uuid.Nil {
		query = query.Where("subject_id = ?", subjectID)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total questions: %w", err)
	}

	// Count by status
	if err := query.Where("status = ?", models.QuestionStatusPublished).Count(&published).Error; err != nil {
		return nil, fmt.Errorf("failed to count published questions: %w", err)
	}
	if err := query.Where("status = ?", models.QuestionStatusDraft).Count(&draft).Error; err != nil {
		return nil, fmt.Errorf("failed to count draft questions: %w", err)
	}
	if err := query.Where("status = ?", models.QuestionStatusArchived).Count(&archived).Error; err != nil {
		return nil, fmt.Errorf("failed to count archived questions: %w", err)
	}

	// Calculate average marks
	var avgMarks float64
	if err := r.db.WithContext(ctx).Model(&models.QuestionBank{}).
		Select("COALESCE(AVG(marks), 0)").
		Where("deleted_at IS NULL").
		Row().Scan(&avgMarks); err != nil {
		return nil, fmt.Errorf("failed to calculate average marks: %w", err)
	}

	return map[string]interface{}{
		"total_questions": total,
		"published_count": published,
		"draft_count":     draft,
		"archived_count":  archived,
		"average_marks":   avgMarks,
	}, nil
}

func (r *QuestionRepository) GetDetailedStatistics(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	query := r.db.WithContext(ctx).Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
	
	// Apply filters
	for key, val := range filters {
		if val != nil && val != "" {
			query = query.Where(key+" = ?", val)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count questions: %w", err)
	}

	// Count by status
	var published, draft, archived int64
	query.Where("status = ?", models.QuestionStatusPublished).Count(&published)
	query.Where("status = ?", models.QuestionStatusDraft).Count(&draft)
	query.Where("status = ?", models.QuestionStatusArchived).Count(&archived)

	// Calculate average marks
	var avgMarks float64
	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

	// Get breakdown by difficulty
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

	// Get breakdown by question type
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

	return map[string]interface{}{
		"total":          total,
		"published":      published,
		"draft":          draft,
		"archived":       archived,
		"average_marks":  avgMarks,
		"by_difficulty":  byDifficulty,
		"by_question_type": byType,
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

func (r *QuestionRepository) CountQuestionsBySubject(ctx context.Context, subjectID uuid.UUID) (int64, error) {
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

func (r *QuestionRepository) GetRecentQuestions(ctx context.Context, subjectID uuid.UUID, limit int) ([]models.QuestionBank, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var questions []models.QuestionBank
	query := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit)

	if subjectID != uuid.Nil {
		query = query.Where("subject_id = ?", subjectID)
	}

	err := query.Find(&questions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recent questions: %w", err)
	}
	return questions, nil
}

// ============================================
// FUNCTION TO GET DB INSTANCE
// ============================================

func (r *QuestionRepository) GetDB() *gorm.DB {
	return r.db
}




// package repository

// import (
// 	"cbt-api/internal/models"
// 	"time"

// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// type QuestionRepository struct {
// 	db *gorm.DB
// }

// func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
// 	return &QuestionRepository{db: db}
// }

// // ============================================
// // CRUD – unchanged
// // ============================================

// func (r *QuestionRepository) Create(question *models.QuestionBank) error {
// 	return r.db.Create(question).Error
// }

// func (r *QuestionRepository) FindByID(id uuid.UUID) (*models.QuestionBank, error) {
// 	var q models.QuestionBank
// 	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&q).Error
// 	return &q, err
// }

// func (r *QuestionRepository) Update(question *models.QuestionBank) error {
// 	return r.db.Save(question).Error
// }

// func (r *QuestionRepository) Delete(id uuid.UUID) error {
// 	return r.db.Where("id = ?", id).Delete(&models.QuestionBank{}).Error
// }

// // ============================================
// // Query / Filter – extended with new fields
// // ============================================

// func (r *QuestionRepository) ListBySubject(subjectID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
// 	var questions []models.QuestionBank
// 	offset := (page - 1) * limit
// 	var total int64
// 	query := r.db.Model(&models.QuestionBank{}).Where("subject_id = ? AND deleted_at IS NULL", subjectID)
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}
// 	err := query.Offset(offset).Limit(limit).Find(&questions).Error
// 	return questions, total, err
// }

// func (r *QuestionRepository) Filter(params map[string]interface{}, page, limit int) ([]models.QuestionBank, int64, error) {
// 	var questions []models.QuestionBank
// 	query := r.db.Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

// 	// Existing fields
// 	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
// 		query = query.Where("subject_id = ?", subjectID)
// 	}
// 	if topic, ok := params["topic"]; ok && topic != "" {
// 		query = query.Where("topic = ?", topic)
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
// 	if search, ok := params["search"]; ok && search != "" {
// 		query = query.Where("question_text ILIKE ?", "%"+search.(string)+"%")
// 	}

// 	// New fields
// 	if schoolID, ok := params["school_id"]; ok && schoolID != "" {
// 		query = query.Where("school_id = ?", schoolID)
// 	}
// 	if classLevelID, ok := params["class_level_id"]; ok && classLevelID != "" {
// 		query = query.Where("class_level_id = ?", classLevelID)
// 	}
// 	if sessionID, ok := params["session_id"]; ok && sessionID != "" {
// 		query = query.Where("session_id = ?", sessionID)
// 	}
// 	if termID, ok := params["term_id"]; ok && termID != "" {
// 		query = query.Where("term_id = ?", termID)
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

// 	var total int64
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}
// 	offset := (page - 1) * limit
// 	err := query.Offset(offset).Limit(limit).Find(&questions).Error
// 	return questions, total, err
// }

// func (r *QuestionRepository) FindByTag(tagName string, page, limit int) ([]models.QuestionBank, int64, error) {
// 	var questions []models.QuestionBank
// 	query := r.db.Joins("JOIN question_tag_mappings ON question_tag_mappings.question_id = question_bank.id").
// 		Joins("JOIN tags ON tags.id = question_tag_mappings.tag_id").
// 		Where("tags.name = ? AND question_bank.deleted_at IS NULL", tagName)
// 	var total int64
// 	if err := query.Count(&total).Error; err != nil {
// 		return nil, 0, err
// 	}
// 	offset := (page - 1) * limit
// 	err := query.Offset(offset).Limit(limit).Find(&questions).Error
// 	return questions, total, err
// }

// // ============================================
// // Bulk / Batch – unchanged
// // ============================================

// func (r *QuestionRepository) BulkCreate(questions []models.QuestionBank) error {
// 	return r.db.CreateInBatches(questions, 100).Error
// }

// func (r *QuestionRepository) BulkDelete(ids []uuid.UUID) error {
// 	return r.db.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error
// }

// func (r *QuestionRepository) BulkUpdateStatus(ids []uuid.UUID, status string) error {
// 	return r.db.Model(&models.QuestionBank{}).Where("id IN ?", ids).Update("status", status).Error
// }

// // ============================================
// // Statistics – extended
// // ============================================

// func (r *QuestionRepository) GetStatistics(subjectID uuid.UUID) (map[string]interface{}, error) {
// 	var total int64
// 	var published, draft, archived int64
// 	query := r.db.Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
// 	if subjectID != uuid.Nil {
// 		query = query.Where("subject_id = ?", subjectID)
// 	}
// 	query.Count(&total)
// 	query.Where("status = ?", "published").Count(&published)
// 	query.Where("status = ?", "draft").Count(&draft)
// 	query.Where("status = ?", "archived").Count(&archived)

// 	var avgMarks float64
// 	r.db.Model(&models.QuestionBank{}).Select("COALESCE(AVG(marks), 0)").Where("deleted_at IS NULL").Row().Scan(&avgMarks)

// 	stats := map[string]interface{}{
// 		"total_questions": total,
// 		"published_count": published,
// 		"draft_count":     draft,
// 		"archived_count":  archived,
// 		"average_marks":   avgMarks,
// 	}
// 	return stats, nil
// }

// // GetDetailedStatistics – new, with filters
// func (r *QuestionRepository) GetDetailedStatistics(filters map[string]interface{}) (map[string]interface{}, error) {
// 	query := r.db.Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
// 	for key, val := range filters {
// 		if val != nil && val != "" {
// 			query = query.Where(key+" = ?", val)
// 		}
// 	}
// 	var total int64
// 	query.Count(&total)

// 	var published, draft, archived int64
// 	query.Where("status = ?", "published").Count(&published)
// 	query.Where("status = ?", "draft").Count(&draft)
// 	query.Where("status = ?", "archived").Count(&archived)

// 	var avgMarks float64
// 	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

// 	// Breakdown by difficulty
// 	var byDifficulty map[string]int
// 	query.Select("difficulty, count(*) as count").Group("difficulty").Scan(&byDifficulty) // simplified

// 	// Placeholder for more breakdowns

// 	return map[string]interface{}{
// 		"total":          total,
// 		"published":      published,
// 		"draft":          draft,
// 		"archived":       archived,
// 		"average_marks":  avgMarks,
// 		"by_difficulty":  byDifficulty,
// 	}, nil
// }

// // ============================================
// // Tag operations – unchanged
// // ============================================

// func (r *QuestionRepository) CreateTag(tag *models.Tag) error {
// 	return r.db.Create(tag).Error
// }

// func (r *QuestionRepository) FindTagByName(name string) (*models.Tag, error) {
// 	var tag models.Tag
// 	err := r.db.Where("name = ?", name).First(&tag).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &tag, nil
// }

// func (r *QuestionRepository) ListTags() ([]models.Tag, error) {
// 	var tags []models.Tag
// 	err := r.db.Find(&tags).Error
// 	return tags, err
// }

// func (r *QuestionRepository) AttachTags(questionID uuid.UUID, tagIDs []uuid.UUID) error {
// 	for _, tid := range tagIDs {
// 		mapping := models.QuestionTagMapping{
// 			ID:         uuid.New(),
// 			QuestionID: questionID,
// 			TagID:      tid,
// 		}
// 		if err := r.db.Create(&mapping).Error; err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (r *QuestionRepository) DetachTags(questionID uuid.UUID, tagIDs []uuid.UUID) error {
// 	return r.db.Where("question_id = ? AND tag_id IN ?", questionID, tagIDs).Delete(&models.QuestionTagMapping{}).Error
// }

// // ============================================
// // NEW METHODS FOR BULK IMPORT & VERSIONING
// // ============================================

// // FindByExternalID – idempotency check
// func (r *QuestionRepository) FindByExternalID(schoolID, sessionID uuid.UUID, externalID string) (*models.QuestionBank, error) {
// 	var q models.QuestionBank
// 	query := r.db.Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)
// 	if sessionID != uuid.Nil {
// 		query = query.Where("session_id = ?", sessionID)
// 	} else {
// 		query = query.Where("session_id IS NULL")
// 	}
// 	err := query.First(&q).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &q, nil
// }

// // CreateNewVersion – duplicates question with incremented version and links to parent
// func (r *QuestionRepository) CreateNewVersion(old *models.QuestionBank, updates map[string]interface{}) (uuid.UUID, error) {
// 	newQ := *old
// 	newQ.ID = uuid.New()
// 	newQ.Version = old.Version + 1
// 	newQ.ParentID = &old.ID
// 	newQ.CreatedAt = time.Now()
// 	newQ.UpdatedAt = time.Now()

// 	// Apply updates
// 	for k, v := range updates {
// 		switch k {
// 		case "topic":
// 			newQ.Topic = v.(string)
// 		case "sub_topic":
// 			newQ.SubTopic = v.(string)
// 		case "learning_objective":
// 			newQ.LearningObjective = v.(string)
// 		case "question_text":
// 			newQ.QuestionText = v.(string)
// 		case "question_type":
// 			newQ.QuestionType = models.QuestionType(v.(string))
// 		case "difficulty":
// 			newQ.Difficulty = models.DifficultyLevel(v.(string))
// 		case "bloom_level":
// 			newQ.BloomLevel = models.BloomTaxonomy(v.(string))
// 		case "options":
// 			newQ.Options = v.(models.JSONMap)
// 		case "correct_option_keys":
// 			newQ.CorrectOptionKeys = v.([]string)
// 		case "rubric":
// 			newQ.Rubric = v.(models.JSONMap)
// 		case "explanation":
// 			newQ.Explanation = v.(string)
// 		case "marks":
// 			newQ.Marks = v.(int)
// 		case "negative_marks":
// 			newQ.NegativeMarks = v.(float64)
// 		case "time_limit_seconds":
// 			if v != nil {
// 				t := v.(int)
// 				newQ.TimeLimitSeconds = &t
// 			} else {
// 				newQ.TimeLimitSeconds = nil
// 			}
// 		case "order":
// 			newQ.Order = v.(int)
// 		case "is_required":
// 			newQ.IsRequired = v.(bool)
// 		case "updated_by":
// 			newQ.UpdatedBy = v.(uuid.UUID)
// 		case "tags":
// 			newQ.Tags = v.(models.JSONMap)
// 		case "curriculum_type":
// 			newQ.CurriculumType = v.(string)
// 		case "source_type":
// 			newQ.SourceType = v.(string)
// 		case "status":
// 			newQ.Status = models.QuestionStatus(v.(string))
// 		}
// 	}
// 	err := r.db.Create(&newQ).Error
// 	return newQ.ID, err
// }



