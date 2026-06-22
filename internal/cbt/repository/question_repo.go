package repository

import (
	"cbt-api/internal/models"
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
// CRUD – unchanged
// ============================================

func (r *QuestionRepository) Create(question *models.QuestionBank) error {
	return r.db.Create(question).Error
}

func (r *QuestionRepository) FindByID(id uuid.UUID) (*models.QuestionBank, error) {
	var q models.QuestionBank
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&q).Error
	return &q, err
}

func (r *QuestionRepository) Update(question *models.QuestionBank) error {
	return r.db.Save(question).Error
}

func (r *QuestionRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.QuestionBank{}).Error
}

// ============================================
// Query / Filter – extended with new fields
// ============================================

func (r *QuestionRepository) ListBySubject(subjectID uuid.UUID, page, limit int) ([]models.QuestionBank, int64, error) {
	var questions []models.QuestionBank
	offset := (page - 1) * limit
	var total int64
	query := r.db.Model(&models.QuestionBank{}).Where("subject_id = ? AND deleted_at IS NULL", subjectID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Offset(offset).Limit(limit).Find(&questions).Error
	return questions, total, err
}

func (r *QuestionRepository) Filter(params map[string]interface{}, page, limit int) ([]models.QuestionBank, int64, error) {
	var questions []models.QuestionBank
	query := r.db.Model(&models.QuestionBank{}).Where("deleted_at IS NULL")

	// Existing fields
	if subjectID, ok := params["subject_id"]; ok && subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}
	if topic, ok := params["topic"]; ok && topic != "" {
		query = query.Where("topic = ?", topic)
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
	if search, ok := params["search"]; ok && search != "" {
		query = query.Where("question_text ILIKE ?", "%"+search.(string)+"%")
	}

	// New fields
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
	if curriculumType, ok := params["curriculum_type"]; ok && curriculumType != "" {
		query = query.Where("curriculum_type = ?", curriculumType)
	}
	if sourceType, ok := params["source_type"]; ok && sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}
	if externalID, ok := params["external_id"]; ok && externalID != "" {
		query = query.Where("external_id = ?", externalID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Find(&questions).Error
	return questions, total, err
}

func (r *QuestionRepository) FindByTag(tagName string, page, limit int) ([]models.QuestionBank, int64, error) {
	var questions []models.QuestionBank
	query := r.db.Joins("JOIN question_tag_mappings ON question_tag_mappings.question_id = question_bank.id").
		Joins("JOIN tags ON tags.id = question_tag_mappings.tag_id").
		Where("tags.name = ? AND question_bank.deleted_at IS NULL", tagName)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Find(&questions).Error
	return questions, total, err
}

// ============================================
// Bulk / Batch – unchanged
// ============================================

func (r *QuestionRepository) BulkCreate(questions []models.QuestionBank) error {
	return r.db.CreateInBatches(questions, 100).Error
}

func (r *QuestionRepository) BulkDelete(ids []uuid.UUID) error {
	return r.db.Where("id IN ?", ids).Delete(&models.QuestionBank{}).Error
}

func (r *QuestionRepository) BulkUpdateStatus(ids []uuid.UUID, status string) error {
	return r.db.Model(&models.QuestionBank{}).Where("id IN ?", ids).Update("status", status).Error
}

// ============================================
// Statistics – extended
// ============================================

func (r *QuestionRepository) GetStatistics(subjectID uuid.UUID) (map[string]interface{}, error) {
	var total int64
	var published, draft, archived int64
	query := r.db.Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
	if subjectID != uuid.Nil {
		query = query.Where("subject_id = ?", subjectID)
	}
	query.Count(&total)
	query.Where("status = ?", "published").Count(&published)
	query.Where("status = ?", "draft").Count(&draft)
	query.Where("status = ?", "archived").Count(&archived)

	var avgMarks float64
	r.db.Model(&models.QuestionBank{}).Select("COALESCE(AVG(marks), 0)").Where("deleted_at IS NULL").Row().Scan(&avgMarks)

	stats := map[string]interface{}{
		"total_questions": total,
		"published_count": published,
		"draft_count":     draft,
		"archived_count":  archived,
		"average_marks":   avgMarks,
	}
	return stats, nil
}

// GetDetailedStatistics – new, with filters
func (r *QuestionRepository) GetDetailedStatistics(filters map[string]interface{}) (map[string]interface{}, error) {
	query := r.db.Model(&models.QuestionBank{}).Where("deleted_at IS NULL")
	for key, val := range filters {
		if val != nil && val != "" {
			query = query.Where(key+" = ?", val)
		}
	}
	var total int64
	query.Count(&total)

	var published, draft, archived int64
	query.Where("status = ?", "published").Count(&published)
	query.Where("status = ?", "draft").Count(&draft)
	query.Where("status = ?", "archived").Count(&archived)

	var avgMarks float64
	query.Select("COALESCE(AVG(marks), 0)").Row().Scan(&avgMarks)

	// Breakdown by difficulty
	var byDifficulty map[string]int
	query.Select("difficulty, count(*) as count").Group("difficulty").Scan(&byDifficulty) // simplified

	// Placeholder for more breakdowns

	return map[string]interface{}{
		"total":          total,
		"published":      published,
		"draft":          draft,
		"archived":       archived,
		"average_marks":  avgMarks,
		"by_difficulty":  byDifficulty,
	}, nil
}

// ============================================
// Tag operations – unchanged
// ============================================

func (r *QuestionRepository) CreateTag(tag *models.Tag) error {
	return r.db.Create(tag).Error
}

func (r *QuestionRepository) FindTagByName(name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *QuestionRepository) ListTags() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Find(&tags).Error
	return tags, err
}

func (r *QuestionRepository) AttachTags(questionID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tid := range tagIDs {
		mapping := models.QuestionTagMapping{
			ID:         uuid.New(),
			QuestionID: questionID,
			TagID:      tid,
		}
		if err := r.db.Create(&mapping).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *QuestionRepository) DetachTags(questionID uuid.UUID, tagIDs []uuid.UUID) error {
	return r.db.Where("question_id = ? AND tag_id IN ?", questionID, tagIDs).Delete(&models.QuestionTagMapping{}).Error
}

// ============================================
// NEW METHODS FOR BULK IMPORT & VERSIONING
// ============================================

// FindByExternalID – idempotency check
func (r *QuestionRepository) FindByExternalID(schoolID, sessionID uuid.UUID, externalID string) (*models.QuestionBank, error) {
	var q models.QuestionBank
	query := r.db.Where("school_id = ? AND external_id = ? AND deleted_at IS NULL", schoolID, externalID)
	if sessionID != uuid.Nil {
		query = query.Where("session_id = ?", sessionID)
	} else {
		query = query.Where("session_id IS NULL")
	}
	err := query.First(&q).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// CreateNewVersion – duplicates question with incremented version and links to parent
func (r *QuestionRepository) CreateNewVersion(old *models.QuestionBank, updates map[string]interface{}) (uuid.UUID, error) {
	newQ := *old
	newQ.ID = uuid.New()
	newQ.Version = old.Version + 1
	newQ.ParentID = &old.ID
	newQ.CreatedAt = time.Now()
	newQ.UpdatedAt = time.Now()

	// Apply updates
	for k, v := range updates {
		switch k {
		case "topic":
			newQ.Topic = v.(string)
		case "sub_topic":
			newQ.SubTopic = v.(string)
		case "learning_objective":
			newQ.LearningObjective = v.(string)
		case "question_text":
			newQ.QuestionText = v.(string)
		case "question_type":
			newQ.QuestionType = models.QuestionType(v.(string))
		case "difficulty":
			newQ.Difficulty = models.DifficultyLevel(v.(string))
		case "bloom_level":
			newQ.BloomLevel = models.BloomTaxonomy(v.(string))
		case "options":
			newQ.Options = v.(models.JSONMap)
		case "correct_option_keys":
			newQ.CorrectOptionKeys = v.([]string)
		case "rubric":
			newQ.Rubric = v.(models.JSONMap)
		case "explanation":
			newQ.Explanation = v.(string)
		case "marks":
			newQ.Marks = v.(int)
		case "negative_marks":
			newQ.NegativeMarks = v.(float64)
		case "time_limit_seconds":
			if v != nil {
				t := v.(int)
				newQ.TimeLimitSeconds = &t
			} else {
				newQ.TimeLimitSeconds = nil
			}
		case "order":
			newQ.Order = v.(int)
		case "is_required":
			newQ.IsRequired = v.(bool)
		case "updated_by":
			newQ.UpdatedBy = v.(uuid.UUID)
		case "tags":
			newQ.Tags = v.(models.JSONMap)
		case "curriculum_type":
			newQ.CurriculumType = v.(string)
		case "source_type":
			newQ.SourceType = v.(string)
		case "status":
			newQ.Status = models.QuestionStatus(v.(string))
		}
	}
	err := r.db.Create(&newQ).Error
	return newQ.ID, err
}



