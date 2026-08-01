package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"cbt-api/internal/ai/engine"
	"cbt-api/internal/ai/queue"
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/repository"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// QuestionService handles all question business logic
type QuestionService struct {
	qRepo   *repository.QuestionRepository
	subRepo *repository.SubjectRepository
	db      *gorm.DB
	queue   queue.Queue
	engine  *engine.Engine
}

// NewQuestionService creates a new QuestionService instance
func NewQuestionService(qRepo *repository.QuestionRepository, subRepo *repository.SubjectRepository, db *gorm.DB, queue queue.Queue, engine *engine.Engine) *QuestionService {
	return &QuestionService{
		qRepo:   qRepo,
		subRepo: subRepo,
		db:      db,
		queue:   queue,
		engine:  engine,
	}
}

// ============================================
// CRUD OPERATIONS
// ============================================

// CreateQuestion creates a new question with full validation
func (s *QuestionService) CreateQuestion(ctx context.Context, req *dto.CreateQuestionRequest, userID string) (*dto.QuestionBankResponse, error) {
	if err := s.validateCreateQuestionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	questionID := models.GenerateID()

	optsStorage := s.convertOptionsToStorage(req.OptionsArray)
	if req.OptionsArray == nil && req.Options != nil {
		optsStorage = s.convertOptionsMapToStorage(req.Options)
	}

	q := &models.QuestionBank{
		ID:                questionID,
		SchoolID:          req.SchoolID,
		SessionID:         req.SessionID,
		TermID:            req.TermID,
		ClassLevelID:      req.ClassLevelID,
		ClassID:           req.ClassID,
		SubjectID:         req.SubjectID,
		ExamType:          req.ExamType,
		Topic:             req.Topic,
		SubTopic:          req.SubTopic,
		LearningObjective: req.LearningObjective,
		QuestionText:      req.QuestionText,
		QuestionType:      models.QuestionType(req.QuestionType),
		Difficulty:        models.DifficultyLevel(req.Difficulty),
		BloomLevel:        models.BloomTaxonomy(req.BloomLevel),
		Options:           optsStorage,
		CorrectAnswer:     req.CorrectAnswer,
		CorrectOptionKeys: req.CorrectOptionKeys,
		Rubric:            s.convertRubricToStorage(req.Rubric),
		Explanation:       req.Explanation,
		Marks:             req.Marks,
		NegativeMarks:     req.NegativeMarks,
		TimeLimitSeconds:  req.TimeLimitSeconds,
		Order:             req.Order,
		IsRequired:        req.IsRequired,
		CurriculumType:    req.CurriculumType,
		SourceType:        req.SourceType,
		ExternalID:        req.ExternalID,
		Status:            models.QuestionStatusDraft,
		Version:           1,
		CreatedBy:         userID,
		UpdatedBy:         userID,
	}

	if err := s.qRepo.Create(ctx, q); err != nil {
		return nil, fmt.Errorf("failed to create question: %w", err)
	}

	if len(req.Tags) > 0 {
		if err := s.attachTagsByNames(ctx, questionID, req.Tags); err != nil {
			return nil, fmt.Errorf("failed to attach tags: %w", err)
		}
	}

	return s.toResponseWithSubject(ctx, q), nil
}

// GetQuestion retrieves a single question by ID
func (s *QuestionService) GetQuestion(ctx context.Context, id string) (*dto.QuestionBankResponse, error) {
	q, err := s.qRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get question: %w", err)
	}
	return s.toResponseWithSubject(ctx, q), nil
}

// UpdateQuestion updates an existing question with version control
func (s *QuestionService) UpdateQuestion(ctx context.Context, id string, req *dto.UpdateQuestionRequest, userID string) (*dto.QuestionBankResponse, error) {
	q, err := s.qRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find question: %w", err)
	}

	if err := s.checkQuestionOwnership(q, userID); err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.SchoolID != nil {
		updates["school_id"] = *req.SchoolID
	}
	if req.SessionID != nil {
		updates["session_id"] = *req.SessionID
	}
	if req.TermID != nil {
		updates["term_id"] = *req.TermID
	}
	if req.ClassLevelID != nil {
		updates["class_level_id"] = *req.ClassLevelID
	}
	if req.ClassID != nil {
		updates["class_id"] = *req.ClassID
	}
	if req.SubjectID != nil {
		updates["subject_id"] = *req.SubjectID
	}
	if req.ExamType != nil {
		updates["exam_type"] = *req.ExamType
	}
	if req.QuestionText != nil {
		updates["question_text"] = *req.QuestionText
	}
	if req.Topic != nil {
		updates["topic"] = *req.Topic
	}
	if req.SubTopic != nil {
		updates["sub_topic"] = *req.SubTopic
	}
	if req.CorrectAnswer != nil {
		updates["correct_answer"] = *req.CorrectAnswer
	}
	if req.Explanation != nil {
		updates["explanation"] = *req.Explanation
	}
	if req.Marks != nil {
		if err := s.validateMarks(*req.Marks); err != nil {
			return nil, err
		}
		updates["marks"] = *req.Marks
	}
	if req.Difficulty != nil {
		if err := s.validateDifficulty(*req.Difficulty); err != nil {
			return nil, err
		}
		updates["difficulty"] = *req.Difficulty
	}
	if req.BloomLevel != nil {
		if err := s.validateBloomLevel(*req.BloomLevel); err != nil {
			return nil, err
		}
		updates["bloom_level"] = *req.BloomLevel
	}
	if req.TimeLimitSeconds != nil {
		updates["time_limit_seconds"] = req.TimeLimitSeconds
	}
	if req.Status != nil {
		if err := s.validateStatus(*req.Status); err != nil {
			return nil, err
		}
		updates["status"] = *req.Status
	}
	if req.CurriculumType != nil {
		updates["curriculum_type"] = *req.CurriculumType
	}
	if req.SourceType != nil {
		updates["source_type"] = *req.SourceType
	}
	if req.LearningObjective != nil {
		updates["learning_objective"] = *req.LearningObjective
	}
	if req.NegativeMarks != nil {
		updates["negative_marks"] = *req.NegativeMarks
	}
	if req.Order != nil {
		updates["order"] = *req.Order
	}
	if req.IsRequired != nil {
		updates["is_required"] = *req.IsRequired
	}
	if req.Options != nil {
		updates["options"] = s.convertOptionsMapToStorage(req.Options)
	}
	if req.OptionsArray != nil {
		updates["options"] = s.convertOptionsToStorage(req.OptionsArray)
	}
	if req.CorrectOptionKeys != nil {
		updates["correct_option_keys"] = req.CorrectOptionKeys
	}
	if req.Rubric != nil {
		updates["rubric"] = s.convertRubricToStorage(req.Rubric)
	}
	updates["updated_by"] = userID

	if len(updates) == 0 {
		return s.toResponseWithSubject(ctx, q), nil
	}

	newID, err := s.qRepo.CreateNewVersion(ctx, q, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to create new version: %w", err)
	}

	newQ, err := s.qRepo.FindByID(ctx, newID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated question: %w", err)
	}

	return s.toResponseWithSubject(ctx, newQ), nil
}

// DeleteQuestion soft-deletes a question with cascading
func (s *QuestionService) DeleteQuestion(ctx context.Context, id string, userID string) error {
	q, err := s.qRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find question: %w", err)
	}

	if err := s.checkQuestionOwnership(q, userID); err != nil {
		return err
	}

	if err := s.qRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete question: %w", err)
	}

	return nil
}

// ============================================
// LISTING & FILTERING OPERATIONS
// ============================================

func (s *QuestionService) ListQuestions(ctx context.Context, subjectID string, page, limit int) ([]dto.QuestionBankResponse, int64, error) {
	qs, total, err := s.qRepo.ListBySubject(ctx, subjectID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list questions: %w", err)
	}

	resp := make([]dto.QuestionBankResponse, 0, len(qs))
	for i := range qs {
		r := s.toResponseWithSubject(ctx, &qs[i])
		resp = append(resp, *r)
	}
	return resp, total, nil
}

func (s *QuestionService) FilterQuestions(ctx context.Context, req *dto.FilterQuestionsRequest) ([]dto.QuestionBankResponse, int64, error) {
	params := map[string]interface{}{
		"subject_id":      req.SubjectID,
		"school_id":       req.SchoolID,
		"class_level_id":  req.ClassLevelID,
		"session_id":      req.SessionID,
		"term_id":         req.TermID,
		"class_id":        req.ClassID,
		"exam_type":       req.ExamType,
		"topic":           req.Topic,
		"difficulty":      strings.Join(req.Difficulty, ","),
		"bloom_level":     strings.Join(req.BloomLevel, ","),
		"question_type":   strings.Join(req.QuestionType, ","),
		"status":          req.Status,
		"search":          req.Search,
	}

	qs, total, err := s.qRepo.Filter(ctx, params, req.Page, req.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to filter questions: %w", err)
	}

	resp := make([]dto.QuestionBankResponse, 0, len(qs))
	for i := range qs {
		r := s.toResponseWithSubject(ctx, &qs[i])
		resp = append(resp, *r)
	}
	return resp, total, nil
}

func (s *QuestionService) GetStatistics(ctx context.Context, subjectID string) (map[string]interface{}, error) {
	stats, err := s.qRepo.GetStatistics(ctx, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return stats, nil
}

// ============================================
// BULK OPERATIONS
// ============================================

func (s *QuestionService) BulkCreateQuestionsFromJSON(ctx context.Context, req *dto.BulkCreateQuestionRequest, userID string) ([]dto.QuestionBankResponse, error) {
	if len(req.Questions) == 0 {
		return nil, errors.New("no questions provided")
	}

	var responses []dto.QuestionBankResponse

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txQRepo := repository.NewQuestionRepository(tx)

		for idx, qReq := range req.Questions {
			if err := s.validateCreateQuestionRequest(&qReq); err != nil {
				return fmt.Errorf("question %d validation failed: %w", idx+1, err)
			}

			questionID := models.GenerateID()

			optsStorage := s.convertOptionsToStorage(qReq.OptionsArray)
			if qReq.OptionsArray == nil && qReq.Options != nil {
				optsStorage = s.convertOptionsMapToStorage(qReq.Options)
			}

			q := &models.QuestionBank{
				ID:                questionID,
				SchoolID:          qReq.SchoolID,
				SessionID:         qReq.SessionID,
				TermID:            qReq.TermID,
				ClassLevelID:      qReq.ClassLevelID,
				ClassID:           qReq.ClassID,
				SubjectID:         qReq.SubjectID,
				ExamType:          qReq.ExamType,
				Topic:             qReq.Topic,
				SubTopic:          qReq.SubTopic,
				LearningObjective: qReq.LearningObjective,
				QuestionText:      qReq.QuestionText,
				QuestionType:      models.QuestionType(qReq.QuestionType),
				Difficulty:        models.DifficultyLevel(qReq.Difficulty),
				BloomLevel:        models.BloomTaxonomy(qReq.BloomLevel),
				Options:           optsStorage,
				CorrectAnswer:     qReq.CorrectAnswer,
				CorrectOptionKeys: qReq.CorrectOptionKeys,
				Rubric:            s.convertRubricToStorage(qReq.Rubric),
				Explanation:       qReq.Explanation,
				Marks:             qReq.Marks,
				NegativeMarks:     qReq.NegativeMarks,
				TimeLimitSeconds:  qReq.TimeLimitSeconds,
				Order:             qReq.Order,
				IsRequired:        qReq.IsRequired,
				CurriculumType:    qReq.CurriculumType,
				SourceType:        qReq.SourceType,
				ExternalID:        qReq.ExternalID,
				Status:            models.QuestionStatusDraft,
				Version:           1,
				CreatedBy:         userID,
				UpdatedBy:         userID,
			}

			if err := txQRepo.Create(ctx, q); err != nil {
				return fmt.Errorf("failed to create question %d: %w", idx+1, err)
			}

			if len(qReq.Tags) > 0 {
				if err := s.attachTagsByNamesInTx(ctx, tx, questionID, qReq.Tags); err != nil {
					return fmt.Errorf("failed to attach tags for question %d: %w", idx+1, err)
				}
			}

			resp, err := s.toResponseLight(ctx, q)
			if err != nil {
				return fmt.Errorf("failed to build response: %w", err)
			}
			responses = append(responses, *resp)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	for i := range responses {
		subj, _ := s.subRepo.FindByID(responses[i].SubjectID)
		if subj != nil {
			responses[i].SubjectName = subj.Name
		}
	}

	return responses, nil
}

func (s *QuestionService) BulkDelete(ctx context.Context, req *dto.BulkDeleteRequest) error {
	if len(req.QuestionIDs) == 0 {
		return errors.New("no question IDs provided")
	}

	if err := s.qRepo.BulkDelete(ctx, req.QuestionIDs); err != nil {
		return fmt.Errorf("failed to delete questions: %w", err)
	}
	return nil
}

func (s *QuestionService) BulkUpdateStatus(ctx context.Context, ids []string, status string) error {
	if len(ids) == 0 {
		return errors.New("no question IDs provided")
	}

	if err := s.validateStatus(status); err != nil {
		return err
	}

	if err := s.qRepo.BulkUpdateStatus(ctx, ids, status); err != nil {
		return fmt.Errorf("failed to update statuses: %w", err)
	}
	return nil
}

// ============================================
// TAG OPERATIONS
// ============================================

func (s *QuestionService) CreateTag(ctx context.Context, req *dto.CreateTagRequest) (*dto.TagResponse, error) {
	if req.Name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	existing, err := s.qRepo.FindTagByName(ctx, req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing tag: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("tag '%s' already exists", req.Name)
	}

	tag := &models.Tag{
		ID:          models.GenerateID(),
		Name:        req.Name,
		Slug:        strings.ReplaceAll(strings.ToLower(req.Name), " ", "-"),
		Description: req.Description,
	}

	if err := s.qRepo.CreateTag(ctx, tag); err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return &dto.TagResponse{
		ID:          tag.ID,
		Name:        tag.Name,
		Slug:        tag.Slug,
		Description: tag.Description,
		CreatedAt:   tag.CreatedAt,
	}, nil
}

func (s *QuestionService) ListTags(ctx context.Context, page, limit int) ([]dto.TagResponse, int64, error) {
	tags, total, err := s.qRepo.ListTagsPaginated(ctx, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
	}

	resp := make([]dto.TagResponse, len(tags))
	for i, t := range tags {
		resp[i] = dto.TagResponse{
			ID:          t.ID,
			Name:        t.Name,
			Slug:        t.Slug,
			Description: t.Description,
			UsageCount:  t.UsageCount,
			CreatedAt:   t.CreatedAt,
		}
	}
	return resp, total, nil
}

// ============================================
// BULK UPLOAD FROM FILE - UPDATED WITH DTO PARSERS
// ============================================

func (s *QuestionService) BulkUploadFromFile(ctx context.Context, file io.Reader, format string, req *dto.BulkUploadRequest, userID string) (*dto.BulkUploadResponse, error) {
	// Validate request
	if err := s.validateBulkUploadRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Determine format if auto
	var err error
	format, err = dto.GetFormatFromRequest(format, req.File.Filename)
	if err != nil {
		return nil, err
	}

	// Create parser factory and get appropriate parser
	factory := dto.NewParserFactory()
	parser, err := factory.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("unsupported format '%s': %w", format, err)
	}

	// Parse the file using the DTO parser
	importItems, err := parser.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s file: %w", format, err)
	}

	if len(importItems) == 0 {
		return nil, errors.New("no valid questions found in file")
	}

	// Prepare response
	resp := &dto.BulkUploadResponse{
		TotalProcessed: len(importItems),
		SuccessCount:   0,
		FailedCount:    0,
		Errors:         []string{},
	}

	// Process in batches for performance
	batchSize := 100

	for i := 0; i < len(importItems); i += batchSize {
		end := i + batchSize
		if end > len(importItems) {
			end = len(importItems)
		}
		batch := importItems[i:end]

		for idx, item := range batch {
			rowNumber := i + idx + 1

			// Validate the imported item
			if err := s.validateImportItem(&item); err != nil {
				resp.FailedCount++
				resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: %v", rowNumber, err))
				continue
			}

			// Build question model from import item
			q, err := s.buildQuestionFromImportItem(ctx, &item, req, userID)
			if err != nil {
				resp.FailedCount++
				resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: %v", rowNumber, err))
				continue
			}

			// Create the question
			if err := s.qRepo.Create(ctx, q); err != nil {
				resp.FailedCount++
				resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: failed to create question: %v", rowNumber, err))
				continue
			}

			// Attach tags if any
			if len(item.Tags) > 0 {
				if err := s.attachTagsByNames(ctx, q.ID, item.Tags); err != nil {
					// Don't fail the whole import, just log the error
					resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: question created but failed to attach tags: %v", rowNumber, err))
				}
			}

			resp.SuccessCount++
		}
	}

	return resp, nil
}

// ============================================
// VALIDATION HELPERS FOR BULK UPLOAD
// ============================================

func (s *QuestionService) validateBulkUploadRequest(req *dto.BulkUploadRequest) error {
	if req.SchoolID == "" {
		return errors.New("school_id is required")
	}
	if req.SessionID == "" {
		return errors.New("session_id is required")
	}
	if req.TermID == "" {
		return errors.New("term_id is required")
	}
	if req.ClassLevelID == "" {
		return errors.New("class_level_id is required")
	}
	if req.ClassID == "" {
		return errors.New("class_id is required")
	}
	if req.SubjectID == "" {
		return errors.New("subject_id is required")
	}
	if req.ExamType == "" {
		return errors.New("exam_type is required")
	}
	if req.File == nil {
		return errors.New("file is required")
	}
	return nil
}

func (s *QuestionService) validateImportItem(item *dto.QuestionImportItem) error {
	if item.QuestionText == "" {
		return errors.New("question text is required")
	}
	if item.Topic == "" {
		return errors.New("topic is required")
	}
	if item.Marks < 1 {
		return errors.New("marks must be at least 1")
	}

	if err := s.validateQuestionType(item.QuestionType); err != nil {
		return err
	}
	if err := s.validateDifficulty(item.Difficulty); err != nil {
		return err
	}
	if err := s.validateBloomLevel(item.BloomLevel); err != nil {
		return err
	}

	switch item.QuestionType {
	case dto.QuestionTypeSingle, dto.QuestionTypeMultiple, dto.QuestionTypeTrueFalse:
		if len(item.Options) == 0 {
			return errors.New("options are required for MCQ questions")
		}
		if len(item.CorrectOptionKeys) == 0 && item.CorrectAnswer == "" {
			return errors.New("correct answer is required for MCQ questions")
		}
	case dto.QuestionTypeEssay:
		if len(item.Rubric) == 0 {
			return errors.New("rubric is required for essay questions")
		}
		if len(item.Options) > 0 {
			return errors.New("essay questions cannot have options")
		}
	case dto.QuestionTypeFillBlank:
		if len(item.CorrectOptionKeys) == 0 && item.CorrectAnswer == "" {
			return errors.New("correct answer is required for fill in the blank questions")
		}
		if len(item.Options) > 0 {
			return errors.New("fill in the blank questions cannot have options")
		}
	}

	return nil
}

// ============================================
// QUESTION BUILDER FROM IMPORT ITEM
// ============================================

func (s *QuestionService) buildQuestionFromImportItem(ctx context.Context, item *dto.QuestionImportItem, req *dto.BulkUploadRequest, createdBy string) (*models.QuestionBank, error) {
	optsStorage := s.convertQuestionOptionsToStorage(item.Options)
	rubricStorage := s.convertRubricToStorage(item.Rubric)

	correctAnswer := item.CorrectAnswer
	correctOptionKeys := item.CorrectOptionKeys

	if len(correctOptionKeys) == 0 && correctAnswer != "" {
		correctOptionKeys = []string{strings.ToUpper(strings.TrimSpace(correctAnswer))}
	}

	var timeLimitSeconds *int
	if item.TimeLimitSeconds > 0 {
		timeLimitSeconds = &item.TimeLimitSeconds
	}

	q := &models.QuestionBank{
		ID:                models.GenerateID(),
		SchoolID:          req.SchoolID,
		SessionID:         req.SessionID,
		TermID:            req.TermID,
		ClassLevelID:      req.ClassLevelID,
		ClassID:           req.ClassID,
		SubjectID:         req.SubjectID,
		ExamType:          req.ExamType,
		Topic:             item.Topic,
		SubTopic:          item.SubTopic,
		LearningObjective: item.LearningObjective,
		QuestionText:      item.QuestionText,
		QuestionType:      models.QuestionType(item.QuestionType),
		Difficulty:        models.DifficultyLevel(item.Difficulty),
		BloomLevel:        models.BloomTaxonomy(item.BloomLevel),
		Options:           optsStorage,
		CorrectAnswer:     correctAnswer,
		CorrectOptionKeys: correctOptionKeys,
		Rubric:            rubricStorage,
		Explanation:       item.Explanation,
		Marks:             item.Marks,
		NegativeMarks:     item.NegativeMarks,
		TimeLimitSeconds:  timeLimitSeconds,
		Order:             item.Order,
		IsRequired:        item.IsRequired,
		CurriculumType:    req.CurriculumType,
		SourceType:        "upload",
		ExternalID:        item.ExternalID,
		Status:            models.QuestionStatusDraft,
		Version:           1,
		CreatedBy:         createdBy,
		UpdatedBy:         createdBy,
	}

	return q, nil
}

// ============================================
// CONVERTER FUNCTIONS FOR IMPORT ITEMS
// ============================================

func (s *QuestionService) convertQuestionOptionsToStorage(opts []dto.QuestionOption) models.OptionStorage {
	if opts == nil || len(opts) == 0 {
		return models.OptionStorage{}
	}

	storage := make(models.OptionStorage, len(opts))
	for i, opt := range opts {
		storage[i] = models.OptionItem{
			Key:  opt.Key,
			Text: opt.Text,
		}
	}
	return storage
}

// parseInt helper - converts string to int
func (s *QuestionService) parseInt(str string) int {
	if str == "" {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(str))
	if err != nil {
		return 0
	}
	return val
}

// ============================================
// AI METHODS
// ============================================

func (s *QuestionService) GenerateQuestionsWithAI(ctx context.Context, req *dto.AIGenerateQuestionsRequest) (*dto.AIQuestionGenerationResponse, error) {
	if err := s.validateAIGenerateRequest(req); err != nil {
		return nil, err
	}

	job := &models.AIQuestionGenerationJob{
		ID:                models.GenerateID(),
		UserID:            "",
		SubjectID:         req.SubjectID,
		Topic:             req.Topic,
		NumberOfQuestions: req.NumberOfQuestions,
		Difficulty:        models.DifficultyLevel(req.Difficulty),
		BloomLevel:        models.BloomTaxonomy(req.BloomLevel),
		SourceText:        req.SourceText,
		Status:            "queued",
	}

	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	payload := map[string]interface{}{
		"job_id":  job.ID,
		"type":    "generate",
		"request": req,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	if err := s.queue.Push(ctx, "ai_jobs", string(data)); err != nil {
		return nil, fmt.Errorf("failed to queue job: %w", err)
	}

	return &dto.AIQuestionGenerationResponse{
		JobID:   job.ID,
		Status:  "queued",
		Message: "Job enqueued successfully",
	}, nil
}

func (s *QuestionService) ExtractQuestionsFromText(ctx context.Context, req *dto.ExtractTextQuestionsRequest) (*dto.AIQuestionGenerationResponse, error) {
	if err := s.validateExtractRequest(req); err != nil {
		return nil, err
	}

	job := &models.AIQuestionGenerationJob{
		ID:         models.GenerateID(),
		UserID:     "",
		SubjectID:  req.SubjectID,
		SourceText: req.Text,
		Status:     "queued",
	}

	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	payload := map[string]interface{}{
		"job_id":    job.ID,
		"type":      "extract",
		"text":      req.Text,
		"school":    req.SchoolID,
		"session":   req.SessionID,
		"term":      req.TermID,
		"class":     req.ClassLevelID,
		"subject":   req.SubjectID,
		"exam_type": req.ExamType,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	if err := s.queue.Push(ctx, "ai_jobs", string(data)); err != nil {
		return nil, fmt.Errorf("failed to queue job: %w", err)
	}

	return &dto.AIQuestionGenerationResponse{
		JobID:   job.ID,
		Status:  "queued",
		Message: "Extraction job enqueued",
	}, nil
}

func (s *QuestionService) GetJobStatus(ctx context.Context, jobID string) (*dto.AIJobStatusResponse, error) {
	var job models.AIQuestionGenerationJob
	if err := s.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &dto.AIJobStatusResponse{
		JobID:        job.ID,
		Status:       job.Status,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		CompletedAt:  job.CompletedAt,
	}, nil
}

// ============================================
// CONTEXT-AWARE METHODS
// ============================================

func (s *QuestionService) GetCurrentAcademicContext(ctx context.Context, schoolID string) (*dto.CurrentAcademicContextResponse, error) {
	var session models.AcademicSession
	err := s.db.WithContext(ctx).
		Where("school_id = ? AND is_current = ? AND is_active = ?", schoolID, true, true).
		First(&session).Error
	if err != nil {
		return nil, errors.New("no current session found for this school")
	}

	var term models.Term
	err = s.db.WithContext(ctx).
		Where("session_id = ? AND is_current = ? AND is_active = ?", session.ID, true, true).
		First(&term).Error
	if err != nil {
		return nil, errors.New("no current term found for this session")
	}

	return &dto.CurrentAcademicContextResponse{
		SchoolID:    schoolID,
		SchoolName:  "",
		SessionID:   session.ID,
		SessionName: session.Name,
		SessionYear: "",
		TermID:      term.ID,
		TermName:    term.Name,
		TermNumber:  term.TermNumber,
		IsCurrent:   term.IsCurrent,
		IsActive:    term.IsActive,
	}, nil
}

func (s *QuestionService) GetQuestionsByTerm(ctx context.Context, subjectID, termID string) ([]dto.QuestionContextResponse, error) {
	questions, err := s.qRepo.FindByTerm(ctx, subjectID, termID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.QuestionContextResponse, len(questions))
	for i, q := range questions {
		resp, err := s.toContextResponse(ctx, &q)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return responses, nil
}

func (s *QuestionService) GetQuestionsBySession(ctx context.Context, subjectID, sessionID string) ([]dto.QuestionContextResponse, error) {
	questions, err := s.qRepo.FindBySession(ctx, subjectID, sessionID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.QuestionContextResponse, len(questions))
	for i, q := range questions {
		resp, err := s.toContextResponse(ctx, &q)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return responses, nil
}

func (s *QuestionService) GetQuestionsByClass(ctx context.Context, classID string) ([]dto.QuestionContextResponse, error) {
	questions, err := s.qRepo.FindByClass(ctx, classID)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.QuestionContextResponse, len(questions))
	for i, q := range questions {
		resp, err := s.toContextResponse(ctx, &q)
		if err != nil {
			continue
		}
		responses[i] = *resp
	}

	return responses, nil
}

func (s *QuestionService) GetQuestionContextSummary(ctx context.Context, subjectID string) (*dto.QuestionContextSummaryResponse, error) {
	subject, err := s.subRepo.FindByID(subjectID)
	if err != nil {
		return nil, fmt.Errorf("subject not found: %w", err)
	}

	summaries, err := s.qRepo.GetQuestionContextSummary(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	termGroups := make([]dto.QuestionTermGroupResponse, 0, len(summaries))
	totalQuestions := 0
	weeklyTestCount := 0
	midTermCount := 0
	mainExamCount := 0
	practiceCount := 0

	for _, summary := range summaries {
		questions, err := s.qRepo.FindByTerm(ctx, subjectID, summary.TermID)
		if err != nil {
			continue
		}

		questionResponses := make([]dto.QuestionContextResponse, len(questions))
		for i, q := range questions {
			resp, err := s.toContextResponse(ctx, &q)
			if err != nil {
				continue
			}
			questionResponses[i] = *resp
		}

		termGroups = append(termGroups, dto.QuestionTermGroupResponse{
			TermID:          summary.TermID,
			TermName:        summary.TermName,
			TermNumber:      summary.TermNumber,
			SessionName:     summary.SessionName,
			TotalCount:      summary.QuestionCount,
			WeeklyTestCount: summary.WeeklyTestCount,
			MidTermCount:    summary.MidTermCount,
			MainExamCount:   summary.MainExamCount,
			PracticeCount:   summary.PracticeCount,
			Questions:       questionResponses,
		})

		totalQuestions += summary.QuestionCount
		weeklyTestCount += summary.WeeklyTestCount
		midTermCount += summary.MidTermCount
		mainExamCount += summary.MainExamCount
		practiceCount += summary.PracticeCount
	}

	return &dto.QuestionContextSummaryResponse{
		SubjectID:       subjectID,
		SubjectName:     subject.Name,
		SchoolID:        "",
		SchoolName:      "",
		TotalQuestions:  totalQuestions,
		WeeklyTestCount: weeklyTestCount,
		MidTermCount:    midTermCount,
		MainExamCount:   mainExamCount,
		PracticeCount:   practiceCount,
		TermGroups:      termGroups,
		Topics:          []dto.TopicSummary{},
	}, nil
}

func (s *QuestionService) GetQuestionsWithContext(ctx context.Context, req *dto.FilterQuestionsWithContextRequest) (*dto.QuestionListWithContextResponse, error) {
	params := map[string]interface{}{
		"subject_id": req.SubjectID,
		"school_id":  req.SchoolID,
		"topic":      req.Topic,
		"status":     req.Status,
		"search":     req.Search,
	}

	if req.SessionID != "" {
		params["session_id"] = req.SessionID
	}
	if req.TermID != "" {
		params["term_id"] = req.TermID
	}
	if req.ClassLevelID != "" {
		params["class_level_id"] = req.ClassLevelID
	}
	if req.ClassID != "" {
		params["class_id"] = req.ClassID
	}
	if req.ExamType != "" {
		params["exam_type"] = req.ExamType
	}
	if len(req.Difficulty) > 0 {
		params["difficulty"] = strings.Join(req.Difficulty, ",")
	}
	if len(req.BloomLevel) > 0 {
		params["bloom_level"] = strings.Join(req.BloomLevel, ",")
	}
	if len(req.QuestionType) > 0 {
		params["question_type"] = strings.Join(req.QuestionType, ",")
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	questions, total, err := s.qRepo.Filter(ctx, params, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	questionResponses := make([]dto.QuestionBankResponse, len(questions))
	termNames := make(map[string]string)
	sessionNames := make(map[string]string)

	for i, q := range questions {
		questionResponses[i] = *s.toResponseWithSubject(ctx, &q)

		if q.TermID != "" {
			var term models.Term
			if err := s.db.WithContext(ctx).First(&term, "id = ?", q.TermID).Error; err == nil {
				termNames[q.TermID] = term.Name
			}
		}

		if q.SessionID != "" {
			var session models.AcademicSession
			if err := s.db.WithContext(ctx).First(&session, "id = ?", q.SessionID).Error; err == nil {
				sessionNames[q.SessionID] = session.Name
			}
		}
	}

	var currentTerm *dto.TermContext
	if req.IsCurrentTerm != nil && *req.IsCurrentTerm {
		ctx, err := s.GetCurrentAcademicContext(ctx, req.SchoolID)
		if err == nil {
			currentTerm = &dto.TermContext{
				TermID:      ctx.TermID,
				TermName:    ctx.TermName,
				TermNumber:  ctx.TermNumber,
				SessionID:   ctx.SessionID,
				SessionName: ctx.SessionName,
			}
		}
	}

	return &dto.QuestionListWithContextResponse{
		Questions:    questionResponses,
		Total:        total,
		Page:         req.Page,
		Limit:        req.Limit,
		TotalPages:   int((total + int64(req.Limit) - 1) / int64(req.Limit)),
		TermNames:    termNames,
		SessionNames: sessionNames,
		CurrentTerm:  currentTerm,
	}, nil
}

func (s *QuestionService) GetQuestionsForExam(ctx context.Context, subjectID, termID, sessionID string) ([]dto.QuestionBankResponse, error) {
	var questions []models.QuestionBank
	err := s.db.WithContext(ctx).
		Where("subject_id = ? AND term_id = ? AND session_id = ? AND deleted_at IS NULL",
			subjectID, termID, sessionID).
		Order("created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, err
	}

	responses := make([]dto.QuestionBankResponse, len(questions))
	for i, q := range questions {
		responses[i] = *s.toResponseWithSubject(ctx, &q)
	}

	return responses, nil
}

// ============================================
// VALIDATION FUNCTIONS
// ============================================

func (s *QuestionService) validateCreateQuestionRequest(req *dto.CreateQuestionRequest) error {
	if req.SchoolID == "" {
		return errors.New("school_id is required")
	}
	if req.SessionID == "" {
		return errors.New("session_id is required")
	}
	if req.TermID == "" {
		return errors.New("term_id is required")
	}
	if req.ClassLevelID == "" {
		return errors.New("class_level_id is required")
	}
	if req.ClassID == "" {
		return errors.New("class_id is required")
	}
	if req.SubjectID == "" {
		return errors.New("subject_id is required")
	}
	if req.ExamType == "" {
		return errors.New("exam_type is required")
	}
	if req.QuestionText == "" {
		return errors.New("question text is required")
	}
	if req.Topic == "" {
		return errors.New("topic is required")
	}
	if req.Marks < 1 {
		return errors.New("marks must be at least 1")
	}

	if err := s.validateQuestionType(req.QuestionType); err != nil {
		return err
	}
	if err := s.validateDifficulty(req.Difficulty); err != nil {
		return err
	}
	if err := s.validateBloomLevel(req.BloomLevel); err != nil {
		return err
	}

	switch req.QuestionType {
	case "single_choice", "multiple_choice", "true_false":
		if len(req.OptionsArray) == 0 && len(req.Options) == 0 {
			return errors.New("options are required for MCQ questions")
		}
		if len(req.CorrectOptionKeys) == 0 && req.CorrectAnswer == "" {
			return errors.New("correct answer is required for MCQ questions")
		}
		if req.Rubric != nil && len(req.Rubric) > 0 {
			return errors.New("MCQ questions cannot have rubric")
		}
	case "essay":
		if req.Rubric == nil || len(req.Rubric) == 0 {
			return errors.New("essay questions must have rubric")
		}
		if len(req.OptionsArray) > 0 || len(req.Options) > 0 {
			return errors.New("essay questions cannot have options")
		}
	}

	return nil
}

func (s *QuestionService) validateQuestionType(qType string) error {
	validTypes := []string{"single_choice", "multiple_choice", "true_false", "essay", "fill_blank"}
	for _, t := range validTypes {
		if t == qType {
			return nil
		}
	}
	return fmt.Errorf("invalid question type: %s", qType)
}

func (s *QuestionService) validateDifficulty(difficulty string) error {
	validDifficulties := []string{"easy", "medium", "hard", "expert"}
	for _, d := range validDifficulties {
		if d == difficulty {
			return nil
		}
	}
	return fmt.Errorf("invalid difficulty: %s", difficulty)
}

func (s *QuestionService) validateBloomLevel(level string) error {
	validLevels := []string{"remember", "understand", "apply", "analyse", "evaluate", "create"}
	for _, l := range validLevels {
		if l == level {
			return nil
		}
	}
	return fmt.Errorf("invalid bloom level: %s", level)
}

func (s *QuestionService) validateStatus(status string) error {
	validStatuses := []string{"draft", "published", "archived"}
	for _, st := range validStatuses {
		if st == status {
			return nil
		}
	}
	return fmt.Errorf("invalid status: %s", status)
}

func (s *QuestionService) validateMarks(marks int) error {
	if marks < 1 {
		return errors.New("marks must be at least 1")
	}
	return nil
}

func (s *QuestionService) validateAIGenerateRequest(req *dto.AIGenerateQuestionsRequest) error {
	if req.SchoolID == "" {
		return errors.New("school_id is required")
	}
	if req.SessionID == "" {
		return errors.New("session_id is required")
	}
	if req.TermID == "" {
		return errors.New("term_id is required")
	}
	if req.ClassLevelID == "" {
		return errors.New("class_level_id is required")
	}
	if req.ClassID == "" {
		return errors.New("class_id is required")
	}
	if req.SubjectID == "" {
		return errors.New("subject_id is required")
	}
	if req.ExamType == "" {
		return errors.New("exam_type is required")
	}
	if req.Topic == "" {
		return errors.New("topic is required")
	}
	if req.NumberOfQuestions < 1 || req.NumberOfQuestions > 100 {
		return errors.New("number_of_questions must be between 1 and 100")
	}
	if err := s.validateDifficulty(req.Difficulty); err != nil {
		return err
	}
	if err := s.validateBloomLevel(req.BloomLevel); err != nil {
		return err
	}
	return nil
}

func (s *QuestionService) validateExtractRequest(req *dto.ExtractTextQuestionsRequest) error {
	if req.SchoolID == "" {
		return errors.New("school_id is required")
	}
	if req.SessionID == "" {
		return errors.New("session_id is required")
	}
	if req.TermID == "" {
		return errors.New("term_id is required")
	}
	if req.ClassLevelID == "" {
		return errors.New("class_level_id is required")
	}
	if req.ClassID == "" {
		return errors.New("class_id is required")
	}
	if req.SubjectID == "" {
		return errors.New("subject_id is required")
	}
	if req.ExamType == "" {
		return errors.New("exam_type is required")
	}
	if req.Text == "" {
		return errors.New("text is required")
	}
	return nil
}

// ============================================
// PERMISSION FUNCTIONS
// ============================================

func (s *QuestionService) checkQuestionOwnership(q *models.QuestionBank, userID string) error {
	if userID == "" {
		return errors.New("user not authenticated")
	}

	if q.CreatedBy != userID {
		return errors.New("permission denied: you do not own this question")
	}

	return nil
}

// ============================================
// CONVERTER FUNCTIONS
// ============================================

func (s *QuestionService) convertOptionsToStorage(opts []dto.QuestionOption) models.OptionStorage {
	if opts == nil || len(opts) == 0 {
		return models.OptionStorage{}
	}

	storage := make(models.OptionStorage, len(opts))
	for i, opt := range opts {
		storage[i] = models.OptionItem{
			Key:  opt.Key,
			Text: opt.Text,
		}
	}
	return storage
}

func (s *QuestionService) convertOptionsMapToStorage(opts map[string]string) models.OptionStorage {
	if opts == nil || len(opts) == 0 {
		return models.OptionStorage{}
	}

	storage := make(models.OptionStorage, 0, len(opts))
	for key, text := range opts {
		storage = append(storage, models.OptionItem{
			Key:  key,
			Text: text,
		})
	}
	return storage
}

func (s *QuestionService) convertRubricToStorage(rubric []dto.RubricCriteria) models.RubricStorage {
	if rubric == nil || len(rubric) == 0 {
		return models.RubricStorage{}
	}

	storage := make(models.RubricStorage, len(rubric))
	for i, r := range rubric {
		storage[i] = models.RubricItem{
			Criteria: r.Criteria,
			Marks:    r.Marks,
		}
	}
	return storage
}

// ============================================
// RESPONSE BUILDERS
// ============================================

func (s *QuestionService) toQuestionBankResponse(q *models.QuestionBank) *dto.QuestionBankResponse {
	opts := make([]dto.QuestionOption, 0, len(q.Options))
	for _, item := range q.Options {
		opts = append(opts, dto.QuestionOption{
			Key:  item.Key,
			Text: item.Text,
		})
	}

	rubric := make([]dto.RubricCriteria, 0, len(q.Rubric))
	for _, item := range q.Rubric {
		rubric = append(rubric, dto.RubricCriteria{
			Criteria: item.Criteria,
			Marks:    item.Marks,
		})
	}

	tags := make([]string, 0, len(q.Tags))
	tags = append(tags, q.Tags...)

	return &dto.QuestionBankResponse{
		ID:                q.ID,
		SubjectID:         q.SubjectID,
		SubjectName:       "",
		SchoolID:          q.SchoolID,
		SessionID:         q.SessionID,
		TermID:            q.TermID,
		ClassLevelID:      q.ClassLevelID,
		ClassID:           q.ClassID,
		ExamType:          q.ExamType,
		Topic:             q.Topic,
		SubTopic:          q.SubTopic,
		QuestionText:      q.QuestionText,
		QuestionType:      string(q.QuestionType),
		Difficulty:        string(q.Difficulty),
		BloomLevel:        string(q.BloomLevel),
		Options:           opts,
		CorrectAnswer:     q.CorrectAnswer,
		Explanation:       q.Explanation,
		Marks:             q.Marks,
		TimeLimitSeconds:  q.TimeLimitSeconds,
		Tags:              tags,
		Status:            string(q.Status),
		Version:           q.Version,
		UsageCount:        q.UsageCount,
		SuccessRate:       q.SuccessRate,
		Attachments:       nil,
		CreatedAt:         q.CreatedAt,
		UpdatedAt:         q.UpdatedAt,
		CreatedBy:         q.CreatedBy,
		CreatedByName:     "",
		CurriculumType:    q.CurriculumType,
		SourceType:        q.SourceType,
		ExternalID:        q.ExternalID,
		LearningObjective: q.LearningObjective,
		CorrectOptionKeys: q.CorrectOptionKeys,
		Rubric:            rubric,
		NegativeMarks:     q.NegativeMarks,
		Order:             q.Order,
		IsRequired:        q.IsRequired,
	}
}

func (s *QuestionService) toResponseWithSubject(ctx context.Context, q *models.QuestionBank) *dto.QuestionBankResponse {
	resp := s.toQuestionBankResponse(q)

	subject, err := s.subRepo.FindByID(q.SubjectID)
	if err == nil && subject != nil {
		resp.SubjectName = subject.Name
	}

	return resp
}

func (s *QuestionService) toResponseLight(ctx context.Context, q *models.QuestionBank) (*dto.QuestionBankResponse, error) {
	resp := s.toQuestionBankResponse(q)

	subject, err := s.subRepo.FindByID(q.SubjectID)
	if err == nil && subject != nil {
		resp.SubjectName = subject.Name
	}

	var school models.School
	if err := s.db.WithContext(ctx).First(&school, "id = ?", q.SchoolID).Error; err == nil {
		resp.SchoolName = school.Name
	}

	var session models.AcademicSession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", q.SessionID).Error; err == nil {
		resp.SessionName = session.Name
	}

	var term models.Term
	if err := s.db.WithContext(ctx).First(&term, "id = ?", q.TermID).Error; err == nil {
		resp.TermName = term.Name
		resp.TermNumber = term.TermNumber
	}

	var class models.Class
	if err := s.db.WithContext(ctx).First(&class, "id = ?", q.ClassID).Error; err == nil {
		resp.ClassName = models.GetClassDisplayName(&class)
	}

	var classLevel models.ClassLevel
	if err := s.db.WithContext(ctx).First(&classLevel, "id = ?", q.ClassLevelID).Error; err == nil {
		resp.ClassLevel = classLevel.Name
	}

	return resp, nil
}

// toContextResponse converts a question to a context response
func (s *QuestionService) toContextResponse(ctx context.Context, q *models.QuestionBank) (*dto.QuestionContextResponse, error) {
	subject, err := s.subRepo.FindByID(q.SubjectID)
	subjectName := ""
	if err == nil && subject != nil {
		subjectName = subject.Name
	}

	termName := ""
	termNumber := 0
	isCurrentTerm := false
	if q.TermID != "" {
		var term models.Term
		if err := s.db.WithContext(ctx).First(&term, "id = ?", q.TermID).Error; err == nil {
			termName = term.Name
			termNumber = term.TermNumber
			isCurrentTerm = term.IsCurrent
		}
	}

	sessionName := ""
	isCurrentSession := false
	if q.SessionID != "" {
		var session models.AcademicSession
		if err := s.db.WithContext(ctx).First(&session, "id = ?", q.SessionID).Error; err == nil {
			sessionName = session.Name
			isCurrentSession = session.IsCurrent
		}
	}

	className := ""
	classLevelName := ""
	if q.ClassID != "" {
		var class models.Class
		if err := s.db.WithContext(ctx).First(&class, "id = ?", q.ClassID).Error; err == nil {
			className = models.GetClassDisplayName(&class)
		}
	}
	if q.ClassLevelID != "" {
		var classLevel models.ClassLevel
		if err := s.db.WithContext(ctx).First(&classLevel, "id = ?", q.ClassLevelID).Error; err == nil {
			classLevelName = classLevel.Name
		}
	}

	return &dto.QuestionContextResponse{
		QuestionID:        q.ID,
		QuestionText:      q.QuestionText,
		QuestionType:      string(q.QuestionType),
		Difficulty:        string(q.Difficulty),
		BloomLevel:        string(q.BloomLevel),
		Marks:             q.Marks,
		SchoolID:          q.SchoolID,
		SchoolName:        "",
		SessionID:         q.SessionID,
		SessionName:       sessionName,
		TermID:            q.TermID,
		TermName:          termName,
		TermNumber:        termNumber,
		ClassLevelID:      q.ClassLevelID,
		ClassLevelName:    classLevelName,
		ClassID:           q.ClassID,
		ClassName:         className,
		SubjectID:         q.SubjectID,
		SubjectName:       subjectName,
		ExamType:          q.ExamType,
		IsCurrentTerm:     isCurrentTerm,
		IsCurrentSession:  isCurrentSession,
		Topic:             q.Topic,
		CreatedAt:         q.CreatedAt,
	}, nil
}

// ============================================
// TAG ATTACHMENT FUNCTIONS
// ============================================

func (s *QuestionService) attachTagsByNames(ctx context.Context, questionID string, tagNames []string) error {
	for _, name := range tagNames {
		tag, err := s.qRepo.FindTagByName(ctx, name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to find tag '%s': %w", name, err)
		}

		if tag == nil {
			tag = &models.Tag{
				ID:   models.GenerateID(),
				Name: name,
				Slug: strings.ReplaceAll(strings.ToLower(name), " ", "-"),
			}
			if err := s.qRepo.CreateTag(ctx, tag); err != nil {
				return fmt.Errorf("failed to create tag '%s': %w", name, err)
			}
		}

		if err := s.qRepo.AttachTags(ctx, questionID, []string{tag.ID}); err != nil {
			return fmt.Errorf("failed to attach tag '%s': %w", name, err)
		}
	}
	return nil
}

func (s *QuestionService) attachTagsByNamesInTx(ctx context.Context, tx *gorm.DB, questionID string, tagNames []string) error {
	for _, name := range tagNames {
		var tag models.Tag
		err := tx.WithContext(ctx).Where("name = ?", name).First(&tag).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tag = models.Tag{
					ID:   models.GenerateID(),
					Name: name,
					Slug: strings.ReplaceAll(strings.ToLower(name), " ", "-"),
				}
				if err := tx.WithContext(ctx).Create(&tag).Error; err != nil {
					return fmt.Errorf("failed to create tag '%s': %w", name, err)
				}
			} else {
				return fmt.Errorf("failed to find tag '%s': %w", name, err)
			}
		}

		mapping := models.QuestionTagMapping{
			ID:         models.GenerateID(),
			QuestionID: questionID,
			TagID:      tag.ID,
		}
		if err := tx.WithContext(ctx).Create(&mapping).Error; err != nil {
			return fmt.Errorf("failed to attach tag '%s': %w", name, err)
		}
	}
	return nil
}


// ============================================================
// GROUPED QUESTION METHODS - ADD TO END OF question_service.go
// ============================================================

// GetQuestionsGrouped returns questions organized by exam_type -> question_type
func (s *QuestionService) GetQuestionsGrouped(ctx context.Context, req *dto.FilterQuestionsGroupedRequest) (*dto.QuestionGroupsResponse, error) {
	// 1. Fetch questions with full context
	params := map[string]interface{}{
		"subject_id": req.SubjectID,
		"school_id":  req.SchoolID,
		"topic":      req.Topic,
		"status":     req.Status,
		"search":     req.Search,
	}

	if req.SessionID != "" {
		params["session_id"] = req.SessionID
	}
	if req.TermID != "" {
		params["term_id"] = req.TermID
	}
	if req.ClassLevelID != "" {
		params["class_level_id"] = req.ClassLevelID
	}
	if req.ClassID != "" {
		params["class_id"] = req.ClassID
	}
	if req.ExamType != "" {
		params["exam_type"] = req.ExamType
	}
	if len(req.Difficulty) > 0 {
		params["difficulty"] = strings.Join(req.Difficulty, ",")
	}
	if len(req.BloomLevel) > 0 {
		params["bloom_level"] = strings.Join(req.BloomLevel, ",")
	}
	if len(req.QuestionType) > 0 {
		params["question_type"] = strings.Join(req.QuestionType, ",")
	}

	// Get all questions (no pagination for grouping)
	questions, _, err := s.qRepo.Filter(ctx, params, 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch questions: %w", err)
	}

	// 2. Get context data
	school, _ := s.getSchool(ctx, req.SchoolID)
	session, _ := s.getSession(ctx, req.SessionID)
	term, _ := s.getTerm(ctx, req.TermID)
	class, _ := s.getClass(ctx, req.ClassID)
	subject, _ := s.getSubject(ctx, req.SubjectID)

	// 3. Build response structure
	response := &dto.QuestionGroupsResponse{
		Status:  true,
		Message: "Questions loaded successfully",
		School: dto.SchoolContext{
			ID:   req.SchoolID,
			Name: school.Name,
		},
		Session: dto.SessionContext{
			ID:   req.SessionID,
			Name: session.Name,
		},
		Term: dto.TermContext{
			TermID:      req.TermID,
			TermName:    term.Name,
			TermNumber:  term.TermNumber,
			SessionID:   req.SessionID,
			SessionName: session.Name,
		},
		// Class: dto.ClassContext{
		// 	ID:   req.ClassID,
		// 	Name: class.Name,
		// },
		Class: dto.ClassContext{
			ID:   req.ClassID,
			Name: models.GetClassDisplayName(class),
		},
		Subject: dto.SubjectContext{
			ID:   req.SubjectID,
			Name: subject.Name,
		},
		QuestionGroups: make(map[string]dto.ExamGroup),
	}

	// 4. Group questions by exam_type -> question_type
	for _, q := range questions {
		examType := q.ExamType
		if examType == "" {
			examType = "practice"
		}

		if _, exists := response.QuestionGroups[examType]; !exists {
			response.QuestionGroups[examType] = dto.ExamGroup{
				TotalQuestions: 0,
				QuestionTypes:  make(map[string][]dto.QuestionItem),
			}
		}

		group := response.QuestionGroups[examType]
		questionType := string(q.QuestionType)
		if questionType == "" {
			questionType = "single_choice"
		}

		if _, exists := group.QuestionTypes[questionType]; !exists {
			group.QuestionTypes[questionType] = []dto.QuestionItem{}
		}

		item := s.convertToQuestionItem(&q)
		group.QuestionTypes[questionType] = append(group.QuestionTypes[questionType], item)
		group.TotalQuestions++
		response.QuestionGroups[examType] = group
	}

	return response, nil
}

// // GetAllQuestionsGrouped returns ALL questions organized by exam_type -> question_type
// // This method does NOT require any parameters - it fetches all questions from the database
// func (s *QuestionService) GetAllQuestionsGrouped(ctx context.Context) (*dto.QuestionGroupsResponse, error) {
// 	// 1. Fetch ALL questions (no filters)
// 	params := map[string]interface{}{
// 		"status": "published",
// 	}

// 	// Get all questions (no pagination for grouping)
// 	questions, _, err := s.qRepo.Filter(ctx, params, 1, 10000)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch questions: %w", err)
// 	}

// 	if len(questions) == 0 {
// 		return &dto.QuestionGroupsResponse{
// 			Status:         true,
// 			Message:        "No questions found",
// 			QuestionGroups: make(map[string]dto.ExamGroup),
// 		}, nil
// 	}

// 	// 2. Get context data from the first question
// 	firstQ := questions[0]
// 	school, _ := s.getSchool(ctx, firstQ.SchoolID)
// 	session, _ := s.getSession(ctx, firstQ.SessionID)
// 	term, _ := s.getTerm(ctx, firstQ.TermID)
// 	class, _ := s.getClass(ctx, firstQ.ClassID)
// 	subject, _ := s.getSubject(ctx, firstQ.SubjectID)

// 	// 3. Build response structure
// 	response := &dto.QuestionGroupsResponse{
// 		Status:  true,
// 		Message: "All questions loaded successfully",
// 		School: dto.SchoolContext{
// 			ID:   firstQ.SchoolID,
// 			Name: school.Name,
// 		},
// 		Session: dto.SessionContext{
// 			ID:   firstQ.SessionID,
// 			Name: session.Name,
// 		},
// 		Term: dto.TermContext{
// 			TermID:      firstQ.TermID,
// 			TermName:    term.Name,
// 			TermNumber:  term.TermNumber,
// 			SessionID:   firstQ.SessionID,
// 			SessionName: session.Name,
// 		},
// 		Class: dto.ClassContext{
// 			ID:   firstQ.ClassID,
// 			Name: models.GetClassDisplayName(class),
// 		},
// 		Subject: dto.SubjectContext{
// 			ID:   firstQ.SubjectID,
// 			Name: subject.Name,
// 		},
// 		QuestionGroups: make(map[string]dto.ExamGroup),
// 	}

// 	// 4. Group questions by exam_type -> question_type
// 	for _, q := range questions {
// 		examType := q.ExamType
// 		if examType == "" {
// 			examType = "practice"
// 		}

// 		if _, exists := response.QuestionGroups[examType]; !exists {
// 			response.QuestionGroups[examType] = dto.ExamGroup{
// 				TotalQuestions: 0,
// 				QuestionTypes:  make(map[string][]dto.QuestionItem),
// 			}
// 		}

// 		group := response.QuestionGroups[examType]
// 		questionType := string(q.QuestionType)
// 		if questionType == "" {
// 			questionType = "single_choice"
// 		}

// 		if _, exists := group.QuestionTypes[questionType]; !exists {
// 			group.QuestionTypes[questionType] = []dto.QuestionItem{}
// 		}

// 		item := s.convertToQuestionItem(&q)
// 		group.QuestionTypes[questionType] = append(group.QuestionTypes[questionType], item)
// 		group.TotalQuestions++
// 		response.QuestionGroups[examType] = group
// 	}

// 	return response, nil
// }

// // GetAllQuestionsGrouped returns ALL questions organized by exam_type -> question_type
// // This method does NOT require any parameters - it fetches all questions from the database
// func (s *QuestionService) GetAllQuestionsGrouped(ctx context.Context) (*dto.QuestionGroupsResponse, error) {
// 	// 1. Fetch ALL questions (no filters)
// 	params := map[string]interface{}{
// 		"status": "published",
// 	}

// 	// Get all questions (no pagination for grouping)
// 	questions, _, err := s.qRepo.Filter(ctx, params, 1, 10000)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch questions: %w", err)
// 	}

// 	if len(questions) == 0 {
// 		return &dto.QuestionGroupsResponse{
// 			Status:         true,
// 			Message:        "No questions found",
// 			QuestionGroups: make(map[string]dto.ExamGroup),
// 		}, nil
// 	}

// 	// 2. Get context data from the first question
// 	firstQ := questions[0]
// 	school, _ := s.getSchool(ctx, firstQ.SchoolID)
// 	session, _ := s.getSession(ctx, firstQ.SessionID)
// 	term, _ := s.getTerm(ctx, firstQ.TermID)
// 	class, _ := s.getClass(ctx, firstQ.ClassID)
// 	subject, _ := s.getSubject(ctx, firstQ.SubjectID)

// 	// 3. Build response structure
// 	response := &dto.QuestionGroupsResponse{
// 		Status:  true,
// 		Message: "All questions loaded successfully",
// 		School: dto.SchoolContext{
// 			ID:   firstQ.SchoolID,
// 			Name: school.Name,
// 		},
// 		Session: dto.SessionContext{
// 			ID:   firstQ.SessionID,
// 			Name: session.Name,
// 		},
// 		Term: dto.TermContext{
// 			TermID:      firstQ.TermID,
// 			TermName:    term.Name,
// 			TermNumber:  term.TermNumber,
// 			SessionID:   firstQ.SessionID,
// 			SessionName: session.Name,
// 		},
// 		Class: dto.ClassContext{
// 			ID:   firstQ.ClassID,
// 			Name: models.GetClassDisplayName(class),
// 		},
// 		Subject: dto.SubjectContext{
// 			ID:   firstQ.SubjectID,
// 			Name: subject.Name,
// 		},
// 		QuestionGroups: make(map[string]dto.ExamGroup),
// 	}

// 	// 4. Group questions by exam_type -> question_type
// 	for _, q := range questions {
// 		examType := q.ExamType
// 		if examType == "" {
// 			examType = "practice"
// 		}

// 		if _, exists := response.QuestionGroups[examType]; !exists {
// 			response.QuestionGroups[examType] = dto.ExamGroup{
// 				TotalQuestions: 0,
// 				QuestionTypes:  make(map[string][]dto.QuestionItem),
// 			}
// 		}

// 		group := response.QuestionGroups[examType]
// 		questionType := string(q.QuestionType)
// 		if questionType == "" {
// 			questionType = "single_choice"
// 		}

// 		if _, exists := group.QuestionTypes[questionType]; !exists {
// 			group.QuestionTypes[questionType] = []dto.QuestionItem{}
// 		}

// 		item := s.convertToQuestionItem(&q)
// 		group.QuestionTypes[questionType] = append(group.QuestionTypes[questionType], item)
// 		group.TotalQuestions++
// 		response.QuestionGroups[examType] = group
// 	}

// 	return response, nil
// }

// GetAllQuestionsGrouped returns ALL questions organized by exam_type -> question_type
// This method does NOT require any parameters - it fetches ALL questions from the database
func (s *QuestionService) GetAllQuestionsGrouped(ctx context.Context) (*dto.QuestionGroupsResponse, error) {
	// 1. Fetch ALL questions - NO status filter, NO deleted filter
	var questions []models.QuestionBank
	
	// Get ALL questions directly from database (including draft, published, archived)
	err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&questions).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to fetch questions: %w", err)
	}

	if len(questions) == 0 {
		return &dto.QuestionGroupsResponse{
			Status:         true,
			Message:        "No questions found in the database",
			School:         dto.SchoolContext{ID: "", Name: ""},
			Session:        dto.SessionContext{ID: "", Name: ""},
			Term:           dto.TermContext{},
			Class:          dto.ClassContext{ID: "", Name: ""},
			Subject:        dto.SubjectContext{ID: "", Name: ""},
			QuestionGroups: make(map[string]dto.ExamGroup),
		}, nil
	}

	// 2. Get context data from the first question
	firstQ := questions[0]
	school, _ := s.getSchool(ctx, firstQ.SchoolID)
	session, _ := s.getSession(ctx, firstQ.SessionID)
	term, _ := s.getTerm(ctx, firstQ.TermID)
	class, _ := s.getClass(ctx, firstQ.ClassID)
	subject, _ := s.getSubject(ctx, firstQ.SubjectID)

	// 3. Build response structure
	response := &dto.QuestionGroupsResponse{
		Status:  true,
		Message: fmt.Sprintf("All %d questions loaded successfully (includes draft, published, archived)", len(questions)),
		School: dto.SchoolContext{
			ID:   firstQ.SchoolID,
			Name: school.Name,
		},
		Session: dto.SessionContext{
			ID:   firstQ.SessionID,
			Name: session.Name,
		},
		Term: dto.TermContext{
			TermID:      firstQ.TermID,
			TermName:    term.Name,
			TermNumber:  term.TermNumber,
			SessionID:   firstQ.SessionID,
			SessionName: session.Name,
		},
		Class: dto.ClassContext{
			ID:   firstQ.ClassID,
			Name: models.GetClassDisplayName(class),
		},
		Subject: dto.SubjectContext{
			ID:   firstQ.SubjectID,
			Name: subject.Name,
		},
		QuestionGroups: make(map[string]dto.ExamGroup),
	}

	// 4. Group questions by exam_type -> question_type
	for _, q := range questions {
		examType := q.ExamType
		if examType == "" {
			examType = "practice"
		}

		if _, exists := response.QuestionGroups[examType]; !exists {
			response.QuestionGroups[examType] = dto.ExamGroup{
				TotalQuestions: 0,
				QuestionTypes:  make(map[string][]dto.QuestionItem),
			}
		}

		group := response.QuestionGroups[examType]
		questionType := string(q.QuestionType)
		if questionType == "" {
			questionType = "single_choice"
		}

		if _, exists := group.QuestionTypes[questionType]; !exists {
			group.QuestionTypes[questionType] = []dto.QuestionItem{}
		}

		item := s.convertToQuestionItem(&q)
		group.QuestionTypes[questionType] = append(group.QuestionTypes[questionType], item)
		group.TotalQuestions++
		response.QuestionGroups[examType] = group
	}

	return response, nil
}


// convertToQuestionItem converts a QuestionBank to QuestionItem
func (s *QuestionService) convertToQuestionItem(q *models.QuestionBank) dto.QuestionItem {
	item := dto.QuestionItem{
		QuestionID:        q.ID,
		QuestionText:      q.QuestionText,
		CorrectAnswer:     q.CorrectAnswer,
		CorrectOptionKeys: q.CorrectOptionKeys,
		Difficulty:        string(q.Difficulty),
		BloomLevel:        string(q.BloomLevel),
		Marks:             q.Marks,
		Topic:             q.Topic,
		SubTopic:          q.SubTopic,
		Explanation:       q.Explanation,
	}

	if len(q.Options) > 0 {
		item.Options = make(map[string]string)
		for _, opt := range q.Options {
			item.Options[opt.Key] = opt.Text
		}
	}

	if len(q.Rubric) > 0 {
		item.Rubric = make([]dto.RubricCriteria, len(q.Rubric))
		for i, r := range q.Rubric {
			item.Rubric[i] = dto.RubricCriteria{
				Criteria: r.Criteria,
				Marks:    r.Marks,
			}
		}
	}

	return item
}

// getSchool - Safe school fetcher
func (s *QuestionService) getSchool(ctx context.Context, id string) (*models.School, error) {
	if id == "" {
		return &models.School{ID: id, Name: "Unknown School"}, nil
	}
	var school models.School
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&school).Error
	if err != nil {
		return &models.School{ID: id, Name: "Unknown School"}, nil
	}
	return &school, nil
}

// getSession - Safe session fetcher
func (s *QuestionService) getSession(ctx context.Context, id string) (*models.AcademicSession, error) {
	if id == "" {
		return &models.AcademicSession{ID: id, Name: "Unknown Session"}, nil
	}
	var session models.AcademicSession
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&session).Error
	if err != nil {
		return &models.AcademicSession{ID: id, Name: "Unknown Session"}, nil
	}
	return &session, nil
}

// getTerm - Safe term fetcher
func (s *QuestionService) getTerm(ctx context.Context, id string) (*models.Term, error) {
	if id == "" {
		return &models.Term{ID: id, Name: "Unknown Term", TermNumber: 0}, nil
	}
	var term models.Term
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&term).Error
	if err != nil {
		return &models.Term{ID: id, Name: "Unknown Term", TermNumber: 0}, nil
	}
	return &term, nil
}

// getClass - Safe class fetcher
// func (s *QuestionService) getClass(ctx context.Context, id string) (*models.Class, error) {
// 	if id == "" {
// 		return &models.Class{ID: id, Name: "Unknown Class"}, nil
// 	}
// 	var class models.Class
// 	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&class).Error
// 	if err != nil {
// 		return &models.Class{ID: id, Name: "Unknown Class"}, nil
// 	}
// 	return &class, nil
// }

// getClass - Safe class fetcher
func (s *QuestionService) getClass(ctx context.Context, id string) (*models.Class, error) {
	if id == "" {
		return &models.Class{ID: id}, nil
	}
	var class models.Class
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&class).Error
	if err != nil {
		return &models.Class{ID: id}, nil
	}
	return &class, nil
}

// getSubject - Safe subject fetcher
func (s *QuestionService) getSubject(ctx context.Context, id string) (*models.Subject, error) {
	if id == "" {
		return &models.Subject{ID: id, Name: "Unknown Subject"}, nil
	}
	var subject models.Subject
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&subject).Error
	if err != nil {
		return &models.Subject{ID: id, Name: "Unknown Subject"}, nil
	}
	return &subject, nil
}

// GetQuestionsByTermWithGrouping returns questions for a term grouped by exam type
func (s *QuestionService) GetQuestionsByTermWithGrouping(ctx context.Context, subjectID, termID string) (*dto.QuestionGroupsResponse, error) {
	req := &dto.FilterQuestionsGroupedRequest{
		SubjectID: subjectID,
		TermID:    termID,
	}

	questions, err := s.qRepo.FindByTerm(ctx, subjectID, termID)
	if err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return &dto.QuestionGroupsResponse{
			Status:         true,
			Message:        "No questions found for this term",
			QuestionGroups: make(map[string]dto.ExamGroup),
		}, nil
	}

	firstQ := questions[0]
	req.SchoolID = firstQ.SchoolID
	req.ClassID = firstQ.ClassID
	req.SessionID = firstQ.SessionID

	return s.GetQuestionsGrouped(ctx, req)
}

// GetQuestionsBySessionWithGrouping returns questions for a session grouped by exam type
func (s *QuestionService) GetQuestionsBySessionWithGrouping(ctx context.Context, subjectID, sessionID string) (*dto.QuestionGroupsResponse, error) {
	req := &dto.FilterQuestionsGroupedRequest{
		SubjectID: subjectID,
		SessionID: sessionID,
	}

	questions, err := s.qRepo.FindBySession(ctx, subjectID, sessionID)
	if err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return &dto.QuestionGroupsResponse{
			Status:         true,
			Message:        "No questions found for this session",
			QuestionGroups: make(map[string]dto.ExamGroup),
		}, nil
	}

	firstQ := questions[0]
	req.SchoolID = firstQ.SchoolID
	req.ClassID = firstQ.ClassID
	req.TermID = firstQ.TermID

	return s.GetQuestionsGrouped(ctx, req)
}


