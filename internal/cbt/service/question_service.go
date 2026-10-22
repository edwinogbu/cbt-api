package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	academicRepo "cbt-api/internal/academic/repository"
	"cbt-api/internal/ai/engine"
	"cbt-api/internal/ai/queue"
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/repository"
	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// QuestionService handles all question business logic
type QuestionService struct {
	qRepo     *repository.QuestionRepository
	subRepo   *repository.SubjectRepository
	classRepo *academicRepo.ClassRepository
	userRepo  *academicRepo.UserRepository
	db        *gorm.DB
	queue     queue.Queue
	engine    *engine.Engine
}

// NewQuestionService creates a new QuestionService instance
func NewQuestionService(qRepo *repository.QuestionRepository, subRepo *repository.SubjectRepository, db *gorm.DB, queue queue.Queue, engine *engine.Engine) *QuestionService {
	return &QuestionService{
		qRepo:     qRepo,
		subRepo:   subRepo,
		classRepo: academicRepo.NewClassRepository(db),
		userRepo:  academicRepo.NewUserRepository(db),
		db:        db,
		queue:     queue,
		engine:    engine,
	}
}

// ============================================
// CRUD OPERATIONS
// ============================================

// CreateQuestion creates a new question with full validation
// CreateQuestion creates a question scoped to schoolID (the caller's own
// school, forced server-side - see effectiveSchoolID in
// internal/academic/service/tenant.go; req.SchoolID is honored only for
// an authenticated platform_owner). Before this fix, authorizeQuestionWrite
// gave admin/super_admin an unconditional bypass with no tenant check at
// all, so any admin could create a question under ANY school's
// school_id just by naming it in the request body.
func (s *QuestionService) CreateQuestion(ctx context.Context, req *dto.CreateQuestionRequest, userID, schoolID string, isPlatformOwner bool) (*dto.QuestionBankResponse, error) {
	if err := s.validateCreateQuestionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID

	if err := s.authorizeQuestionWrite(userID, req.SchoolID, req.ClassID); err != nil {
		return nil, err
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

// GetQuestion retrieves a single question by ID, scoped to the caller's
// own school (see questionAuthorizeSchoolOwnership). Previously took no
// school parameter at all - any authenticated role could read any
// school's question (question text, correct answer, explanation) just
// by knowing or enumerating its ID.
func (s *QuestionService) GetQuestion(ctx context.Context, id, schoolID string, isPlatformOwner bool) (*dto.QuestionBankResponse, error) {
	q, err := s.qRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get question: %w", err)
	}
	if err := questionAuthorizeSchoolOwnership(q.SchoolID, schoolID, isPlatformOwner); err != nil {
		return nil, fmt.Errorf("failed to get question: %w", errors.New("question not found"))
	}
	return s.toResponseWithSubject(ctx, q), nil
}

// UpdateQuestion updates an existing question with version control,
// scoped to the caller's own school (see questionAuthorizeSchoolOwnership).
// req.SchoolID is no longer honored at all - letting a caller move a
// question to a different school_id via update was itself a tenant-
// isolation violation regardless of who was asking (see CreateQuestion).
func (s *QuestionService) UpdateQuestion(ctx context.Context, id string, req *dto.UpdateQuestionRequest, userID, schoolID string, isPlatformOwner bool) (*dto.QuestionBankResponse, error) {
	q, err := s.qRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find question: %w", err)
	}

	if err := questionAuthorizeSchoolOwnership(q.SchoolID, schoolID, isPlatformOwner); err != nil {
		return nil, fmt.Errorf("failed to find question: %w", errors.New("question not found"))
	}

	if err := s.checkQuestionOwnership(q, userID); err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

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

// DeleteQuestion soft-deletes a question with cascading, scoped to the
// caller's own school (see questionAuthorizeSchoolOwnership).
func (s *QuestionService) DeleteQuestion(ctx context.Context, id string, userID, schoolID string, isPlatformOwner bool) error {
	q, err := s.qRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find question: %w", err)
	}

	if err := questionAuthorizeSchoolOwnership(q.SchoolID, schoolID, isPlatformOwner); err != nil {
		return fmt.Errorf("failed to find question: %w", errors.New("question not found"))
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

func (s *QuestionService) ListQuestions(ctx context.Context, subjectID, schoolID string, page, limit int) ([]dto.QuestionBankResponse, int64, error) {
	qs, total, err := s.qRepo.ListBySubject(ctx, subjectID, schoolID, page, limit)
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

func (s *QuestionService) GetStatistics(ctx context.Context, subjectID, schoolID string) (map[string]interface{}, error) {
	stats, err := s.qRepo.GetStatistics(ctx, subjectID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return stats, nil
}

// ============================================
// BULK OPERATIONS
// ============================================

// BulkCreateQuestionsFromJSON creates a batch of questions, each forced
// to schoolID (the caller's own school - see questionEffectiveSchoolID;
// a per-item SchoolID in the request body is honored only for an
// authenticated platform_owner). Before this fix, each item's raw
// qReq.SchoolID went straight into authorizeQuestionWrite, which trusted
// admin/super_admin platform-wide - so an admin could import a batch
// under any other school's school_id just by naming it in the body.
func (s *QuestionService) BulkCreateQuestionsFromJSON(ctx context.Context, req *dto.BulkCreateQuestionRequest, userID, schoolID string, isPlatformOwner bool) ([]dto.QuestionBankResponse, error) {
	if len(req.Questions) == 0 {
		return nil, errors.New("no questions provided")
	}

	effectiveID := questionEffectiveSchoolID("", schoolID, isPlatformOwner)
	if !isPlatformOwner && effectiveID == "" {
		return nil, errQuestionSchoolScopeRequired
	}

	// Authorize each distinct (school, class) pair referenced in this batch up
	// front, before opening the write transaction, so a batch spanning a class
	// the caller doesn't own fails fast without partially authorizing others.
	authorizedClasses := make(map[string]bool)
	for idx := range req.Questions {
		if !isPlatformOwner {
			req.Questions[idx].SchoolID = effectiveID
		} else if req.Questions[idx].SchoolID == "" {
			req.Questions[idx].SchoolID = effectiveID
		}
		qReq := req.Questions[idx]
		key := qReq.SchoolID + "|" + qReq.ClassID
		if authorizedClasses[key] {
			continue
		}
		if err := s.authorizeQuestionWrite(userID, qReq.SchoolID, qReq.ClassID); err != nil {
			return nil, fmt.Errorf("question %d: %w", idx+1, err)
		}
		authorizedClasses[key] = true
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

// allowedQuestionIDs filters ids down to the ones the caller may act on:
// the same per-question checks DeleteQuestion/UpdateQuestion already
// apply (school ownership, then creator ownership), applied one ID at a
// time so a caller can't bulk-mutate another school's (or another
// teacher's) questions just by listing their UUIDs. IDs that fail either
// check, or don't exist, are silently dropped rather than erroring the
// whole batch - the same policy BulkDeleteClasses uses - so the caller's
// own valid IDs still go through.
func (s *QuestionService) allowedQuestionIDs(ctx context.Context, ids []string, userID, schoolID string, isPlatformOwner bool) []string {
	allowed := make([]string, 0, len(ids))
	for _, id := range ids {
		q, err := s.qRepo.FindByID(ctx, id)
		if err != nil {
			continue
		}
		if err := questionAuthorizeSchoolOwnership(q.SchoolID, schoolID, isPlatformOwner); err != nil {
			continue
		}
		if err := s.checkQuestionOwnership(q, userID); err != nil {
			continue
		}
		allowed = append(allowed, id)
	}
	return allowed
}

// BulkDelete was previously unscoped: it deleted every ID in the request
// with no school or ownership check at all, letting any authenticated
// teacher/admin delete another school's (or another teacher's) questions
// just by supplying their UUIDs. Now mirrors DeleteQuestion's per-question
// checks before deleting, and reports back how many were actually
// deleted rather than a flat success regardless of what was dropped.
func (s *QuestionService) BulkDelete(ctx context.Context, req *dto.BulkDeleteRequest, userID, schoolID string, isPlatformOwner bool) (int64, error) {
	if len(req.QuestionIDs) == 0 {
		return 0, errors.New("no question IDs provided")
	}

	allowed := s.allowedQuestionIDs(ctx, req.QuestionIDs, userID, schoolID, isPlatformOwner)
	if len(allowed) == 0 {
		return 0, nil
	}

	if err := s.qRepo.BulkDelete(ctx, allowed); err != nil {
		return 0, fmt.Errorf("failed to delete questions: %w", err)
	}
	return int64(len(allowed)), nil
}

// BulkUpdateStatus had the same gap as BulkDelete - no school or
// ownership check at all. Now mirrors UpdateQuestion's per-question
// checks and reports back how many were actually updated.
func (s *QuestionService) BulkUpdateStatus(ctx context.Context, ids []string, status, userID, schoolID string, isPlatformOwner bool) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("no question IDs provided")
	}

	if err := s.validateStatus(status); err != nil {
		return 0, err
	}

	allowed := s.allowedQuestionIDs(ctx, ids, userID, schoolID, isPlatformOwner)
	if len(allowed) == 0 {
		return 0, nil
	}

	if err := s.qRepo.BulkUpdateStatus(ctx, allowed, status); err != nil {
		return 0, fmt.Errorf("failed to update statuses: %w", err)
	}
	return int64(len(allowed)), nil
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

func (s *QuestionService) BulkUploadFromFile(ctx context.Context, file io.Reader, format string, req *dto.BulkUploadRequest, userID, schoolID string, isPlatformOwner bool) (*dto.BulkUploadResponse, error) {
	// Validate request
	if err := s.validateBulkUploadRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID

	if err := s.authorizeQuestionWrite(userID, req.SchoolID, req.ClassID); err != nil {
		return nil, err
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
	const maxImportRows = 1000
	if len(importItems) > maxImportRows {
		return nil, fmt.Errorf("validation failed: too many questions in file: found %d, maximum %d per upload", len(importItems), maxImportRows)
	}

	// Prepare response
	resp := &dto.BulkUploadResponse{
		TotalProcessed: len(importItems),
		SuccessCount:   0,
		FailedCount:    0,
		Errors:         []string{},
		Warnings:       []string{},
	}

	// Pre-fetch existing question text in this exact subject+class scope,
	// to warn (not block) on likely duplicates against the question bank
	// already saved by any teacher - a cheap, scope-bounded query since a
	// class+subject's question count is small relative to the whole bank.
	existingNormalized := make(map[string]bool)
	if req.SubjectID != "" && req.ClassID != "" {
		var existingTexts []string
		s.db.WithContext(ctx).Model(&models.QuestionBank{}).
			Where("subject_id = ? AND class_id = ? AND deleted_at IS NULL", req.SubjectID, req.ClassID).
			Pluck("question_text", &existingTexts)
		for _, t := range existingTexts {
			existingNormalized[normalizeQuestionText(t)] = true
		}
	}

	// Detect exact duplicates WITHIN this file (same normalized text seen
	// twice) - these are skipped rather than saved twice; everything else
	// is reported but never silently discarded.
	seenInFile := make(map[string]int) // normalized text -> first row number seen

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

			normalized := normalizeQuestionText(item.QuestionText)
			if firstRow, dup := seenInFile[normalized]; dup {
				resp.DuplicateCount++
				resp.Warnings = append(resp.Warnings, fmt.Sprintf("Row %d: skipped - duplicates row %d in this file", rowNumber, firstRow))
				continue
			}
			seenInFile[normalized] = rowNumber
			if existingNormalized[normalized] {
				resp.Warnings = append(resp.Warnings, fmt.Sprintf("Row %d: saved, but an existing question in this class/subject looks identical or very similar - review for an unintended duplicate", rowNumber))
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

// PreviewBulkUpload parses, validates, and detects duplicates in an
// uploaded file EXACTLY as BulkUploadFromFile does, but never writes to
// the database - it returns the candidate rows (each mapped to the exact
// shape POST /questions/bulk expects) for the teacher to review, edit, or
// exclude before explicitly confirming the save via that endpoint.
func (s *QuestionService) PreviewBulkUpload(ctx context.Context, file io.Reader, format string, req *dto.BulkUploadRequest, userID, schoolID string, isPlatformOwner bool) (*dto.BulkUploadPreviewResponse, error) {
	if err := s.validateBulkUploadRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID
	if err := s.authorizeQuestionWrite(userID, req.SchoolID, req.ClassID); err != nil {
		return nil, err
	}

	format, err := dto.GetFormatFromRequest(format, req.File.Filename)
	if err != nil {
		return nil, err
	}
	parser, err := dto.NewParserFactory().GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("unsupported format '%s': %w", format, err)
	}
	importItems, err := parser.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s file: %w", format, err)
	}
	if len(importItems) == 0 {
		return nil, errors.New("no valid questions found in file")
	}
	const maxImportRows = 1000
	if len(importItems) > maxImportRows {
		return nil, fmt.Errorf("validation failed: too many questions in file: found %d, maximum %d per upload", len(importItems), maxImportRows)
	}

	existingNormalized := make(map[string]bool)
	if req.SubjectID != "" && req.ClassID != "" {
		var existingTexts []string
		s.db.WithContext(ctx).Model(&models.QuestionBank{}).
			Where("subject_id = ? AND class_id = ? AND deleted_at IS NULL", req.SubjectID, req.ClassID).
			Pluck("question_text", &existingTexts)
		for _, t := range existingTexts {
			existingNormalized[normalizeQuestionText(t)] = true
		}
	}

	resp := &dto.BulkUploadPreviewResponse{
		TotalDetected: len(importItems),
		Persisted:     false,
	}
	seenInFile := make(map[string]int)

	for idx, item := range importItems {
		rowNumber := idx + 1

		if err := s.validateImportItem(&item); err != nil {
			resp.InvalidCount++
			resp.Rows = append(resp.Rows, dto.ImportPreviewRow{RowNumber: rowNumber, Status: "invalid", Reason: err.Error()})
			continue
		}

		normalized := normalizeQuestionText(item.QuestionText)
		if firstRow, dup := seenInFile[normalized]; dup {
			resp.DuplicateCount++
			resp.Rows = append(resp.Rows, dto.ImportPreviewRow{
				RowNumber: rowNumber,
				Status:    "duplicate",
				Reason:    fmt.Sprintf("duplicates row %d in this file", firstRow),
			})
			continue
		}
		seenInFile[normalized] = rowNumber

		question := item.ToCreateQuestionRequest(req)
		row := dto.ImportPreviewRow{RowNumber: rowNumber, Status: "valid", Question: &question}
		if existingNormalized[normalized] {
			row.Status = "duplicate"
			row.Reason = "looks identical or very similar to an existing question already saved for this class/subject - review before confirming"
			resp.DuplicateCount++
		} else {
			resp.ValidCount++
		}
		resp.Rows = append(resp.Rows, row)
	}

	return resp, nil
}

// ============================================
// VALIDATION HELPERS FOR BULK UPLOAD
// ============================================

// normalizeQuestionText collapses whitespace and case so near-identical
// formatting (extra spaces, line breaks, capitalization) doesn't defeat
// duplicate detection, without doing a fuzzy/similarity match.
func normalizeQuestionText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

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
			// None of the bulk-import parsers (CSV, Excel, DOCX, TXT,
			// JSON) ever populate a structured Rubric - no import format
			// has a column/field for it. Every essay row in every
			// format's own official downloadable template therefore
			// failed this exact check, 100% of the time, for every
			// school: confirmed live by uploading GET
			// /questions/templates/csv's own unmodified output back
			// through this endpoint. Every bulk-import format's
			// Explanation column is already documented and used as the
			// essay's model answer (see the CSV template's own sample
			// row), so reusing it as a single default rubric criterion
			// unblocks essay questions in bulk import entirely, using
			// only data the teacher already provided - never a
			// fabricated grading criterion. A teacher can still add more
			// criteria or edit this one afterward via the question edit
			// screen.
			if item.Explanation != "" {
				item.Rubric = []dto.RubricCriteria{{Criteria: item.Explanation, Marks: item.Marks}}
			} else {
				return errors.New("rubric is required for essay questions - add a model-answer/explanation to the imported row, or add a rubric after import")
			}
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

func (s *QuestionService) GenerateQuestionsWithAI(ctx context.Context, req *dto.AIGenerateQuestionsRequest, userID, schoolID string, isPlatformOwner bool) (*dto.AIQuestionGenerationResponse, error) {
	if err := s.validateAIGenerateRequest(req); err != nil {
		return nil, err
	}

	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID

	if err := s.authorizeQuestionWrite(userID, req.SchoolID, req.ClassID); err != nil {
		return nil, err
	}

	job := &models.AIQuestionGenerationJob{
		ID:                models.GenerateID(),
		UserID:            userID,
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
		"user_id": userID,
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

func (s *QuestionService) ExtractQuestionsFromText(ctx context.Context, req *dto.ExtractTextQuestionsRequest, userID, schoolID string, isPlatformOwner bool) (*dto.AIQuestionGenerationResponse, error) {
	if err := s.validateExtractRequest(req); err != nil {
		return nil, err
	}

	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID

	if err := s.authorizeQuestionWrite(userID, req.SchoolID, req.ClassID); err != nil {
		return nil, err
	}

	job := &models.AIQuestionGenerationJob{
		ID:         models.GenerateID(),
		UserID:     userID,
		SubjectID:  req.SubjectID,
		SourceText: req.Text,
		Status:     "queued",
	}

	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	payload := map[string]interface{}{
		"job_id":        job.ID,
		"type":          "extract",
		"text":          req.Text,
		"user_id":       userID,
		"school":        req.SchoolID,
		"session":       req.SessionID,
		"term":          req.TermID,
		"class_level":   req.ClassLevelID,
		"class":         req.ClassID,
		"subject":       req.SubjectID,
		"exam_type":     req.ExamType,
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

// GetJobStatus is scoped to the job's own creator - AIQuestionGenerationJob
// has no school_id column at all (it predates the multi-tenant isolation
// work), so the caller's own user_id is the only ownership check
// available here. Before this fix, any authenticated caller could poll
// ANY job's status/error message by guessing its UUID, with no check of
// any kind - a cross-tenant IDOR, same shape as the question-bank ones
// fixed earlier, just keyed on user rather than school. platform_owner
// still bypasses it, matching every other ownership check in this file.
func (s *QuestionService) GetJobStatus(ctx context.Context, jobID, userID string, isPlatformOwner bool) (*dto.AIJobStatusResponse, error) {
	var job models.AIQuestionGenerationJob
	if err := s.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if !isPlatformOwner && job.UserID != userID {
		return nil, errors.New("job not found")
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
// SYNCHRONOUS, REVIEW-FIRST AI GENERATION
//
// Unlike GenerateQuestionsWithAI/ExtractQuestionsFromText above (which
// enqueue a job that auto-saves to the question bank once the worker
// finishes), these never write to the database. They return drafts for the
// teacher to review/edit; nothing reaches the question bank until the
// teacher explicitly saves the approved drafts via BulkCreateQuestionsFromJSON.
// ============================================

// ExtractLessonStructure extracts structured lesson/topic data from raw text
// for teacher review (Stage C). It does not persist anything.
func (s *QuestionService) ExtractLessonStructure(ctx context.Context, req *dto.ExtractLessonRequest, userID string) (*dto.ExtractLessonResponse, error) {
	// class_id isn't known yet at this stage of the workflow (context
	// selection may not have reached class level), so only role is checked
	// here; the later generate/save steps re-check against the specific
	// class the teacher ultimately picks via authorizeQuestionWrite.
	if err := s.authorizeRoleOnly(userID); err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(req.Text)
	if len(trimmed) < 50 {
		return nil, errors.New("text is too short to extract meaningful lesson structure (minimum 50 characters)")
	}
	const maxExtractChars = 50000
	if len(trimmed) > maxExtractChars {
		trimmed = trimmed[:maxExtractChars]
	}

	prompt := engine.BuildLessonExtractionPrompt(trimmed)
	raw, err := s.engine.Router().Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI lesson extraction failed: %w", err)
	}

	var lessons []dto.ExtractedLessonItem
	if err := json.Unmarshal([]byte(raw), &lessons); err != nil {
		return nil, fmt.Errorf("AI returned an unparsable response for lesson extraction: %w", err)
	}

	resp := &dto.ExtractLessonResponse{Lessons: lessons}
	if len(lessons) == 0 {
		resp.Warning = "No lesson structure could be identified in the supplied text. Try uploading a more detailed lesson plan or scheme of work, or enter topics manually."
	}
	return resp, nil
}

// GenerateGroundedQuestions generates draft questions grounded in
// teacher-approved lesson content, for teacher review (Stage D). It never
// persists anything - the teacher must save approved drafts separately via
// POST /questions/bulk.
func (s *QuestionService) GenerateGroundedQuestions(ctx context.Context, req *dto.GenerateGroundedQuestionsRequest, userID, schoolID string, isPlatformOwner bool) (*dto.GenerateGroundedQuestionsResponse, error) {
	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID

	if err := s.authorizeQuestionWrite(userID, req.SchoolID, req.ClassID); err != nil {
		return nil, err
	}

	// Build one (difficulty, count) bucket per entry in DifficultyDistribution,
	// or a single bucket from the flat Difficulty/NumberOfQuestions fields.
	type bucket struct {
		difficulty string
		count      int
	}
	var buckets []bucket
	if len(req.DifficultyDistribution) > 0 {
		for difficulty, count := range req.DifficultyDistribution {
			if count <= 0 {
				continue
			}
			if err := s.validateDifficulty(difficulty); err != nil {
				return nil, fmt.Errorf("difficulty_distribution: %w", err)
			}
			buckets = append(buckets, bucket{difficulty: difficulty, count: count})
		}
		if len(buckets) == 0 {
			return nil, errors.New("difficulty_distribution must contain at least one positive count")
		}
	} else {
		count := req.NumberOfQuestions
		if count < 1 {
			count = 5
		}
		difficulty := req.Difficulty
		if difficulty == "" {
			difficulty = "medium"
		}
		buckets = []bucket{{difficulty: difficulty, count: count}}
	}

	bloomLevel := req.BloomLevel
	if bloomLevel == "" {
		bloomLevel = "understand"
	}

	requestedTotal := 0
	var allQuestions []engine.Question
	var allStatuses []string
	for _, b := range buckets {
		requestedTotal += b.count
		examCtx := engine.ExamContext{
			SchoolID:          req.SchoolID,
			ClassLevelID:      req.ClassLevelID,
			SubjectID:         req.SubjectID,
			Topic:             req.Topic,
			SubTopic:          req.SubTopic,
			NumberOfQuestions: b.count,
			Difficulty:        b.difficulty,
			BloomLevel:        bloomLevel,
			CurriculumType:    req.CurriculumType,
			SourceText:        req.SourceText,
			Keywords:          req.Keywords,
			ExistingQuestions: allQuestions, // dedupe against questions already generated in this request
		}

		questions, results, err := s.engine.GenerateExam(ctx, examCtx)
		if err != nil {
			return nil, fmt.Errorf("AI question generation failed: %w", err)
		}
		for i, q := range questions {
			status := "approved"
			if i < len(results) {
				status = results[i].Status
			}
			allQuestions = append(allQuestions, q)
			allStatuses = append(allStatuses, status)
		}
	}

	drafts := make([]dto.GeneratedQuestionDraft, 0, len(allQuestions))
	for idx, q := range allQuestions {
		status := allStatuses[idx]
		drafts = append(drafts, dto.GeneratedQuestionDraft{
			QuestionText:      q.QuestionText,
			QuestionType:      mapEngineQuestionType(q.Type, q.CorrectOptionKeys),
			Options:           convertEngineOptions(q.Options),
			CorrectOptionKeys: q.CorrectOptionKeys,
			Explanation:       q.Explanation,
			Marks:             q.Marks,
			Difficulty:        q.Difficulty,
			BloomLevel:        q.BloomLevel,
			Topic:             q.Topic,
			SubTopic:          q.SubTopic,
			LearningObjective: q.LearningObjective,
			Rubric:            convertEngineRubric(q.Rubric),
			ValidationStatus:  status,
		})
	}

	resp := &dto.GenerateGroundedQuestionsResponse{
		Questions:      drafts,
		RequestedCount: requestedTotal,
		GeneratedCount: len(drafts),
		Persisted:      false,
	}
	if len(drafts) == 0 {
		resp.Warning = "AI generation produced no usable questions. This can happen when the source text is too short or off-topic, or when no AI provider is currently available. Try more detailed source material, or create questions manually."
	} else if len(drafts) < requestedTotal {
		resp.Warning = fmt.Sprintf("Only %d of the requested %d questions passed generation and validation. Review the questions below, or generate more to make up the difference.", len(drafts), requestedTotal)
	}
	return resp, nil
}

// authorizeRoleOnly checks that the caller is a teacher/admin/super_admin/
// platform_owner without requiring a specific class (used at workflow
// stages before a class has been selected yet). platform_owner is
// included here for the same reason every other ownership check in this
// file exempts it: it is the one role deliberately meant to act across
// every tenant rather than being confined to one school's staff.
func (s *QuestionService) authorizeRoleOnly(userID string) error {
	if userID == "" {
		return errors.New("unauthorized: user not authenticated")
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("unauthorized: user not found")
	}
	switch user.Role {
	case models.RoleAdmin, models.RoleSuperAdmin, models.RoleTeacher, models.RolePlatformOwner:
		return nil
	default:
		return errors.New("unauthorized: only teachers and admins may use the question-generation workflow")
	}
}

func mapEngineQuestionType(engineType string, correctKeys []string) string {
	switch engineType {
	case "mcq":
		if len(correctKeys) > 1 {
			return dto.QuestionTypeMultiple
		}
		return dto.QuestionTypeSingle
	case "true_false":
		return dto.QuestionTypeTrueFalse
	case "essay":
		return dto.QuestionTypeEssay
	case "fill_blank":
		return dto.QuestionTypeFillBlank
	default:
		return dto.QuestionTypeSingle
	}
}

func convertEngineOptions(opts []engine.Option) []dto.QuestionOption {
	if len(opts) == 0 {
		return nil
	}
	out := make([]dto.QuestionOption, len(opts))
	for i, o := range opts {
		out[i] = dto.QuestionOption{Key: o.Key, Text: o.Text}
	}
	return out
}

func convertEngineRubric(rubric []engine.Rubric) []dto.RubricCriteria {
	if len(rubric) == 0 {
		return nil
	}
	out := make([]dto.RubricCriteria, len(rubric))
	for i, r := range rubric {
		out[i] = dto.RubricCriteria{Criteria: r.Criteria, Marks: r.Marks}
	}
	return out
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

// GetQuestionsByTerm is scoped to schoolID - it previously called
// FindByTerm with no school check at all, so any authenticated caller
// could read any other school's questions for a term_id it doesn't own.
func (s *QuestionService) GetQuestionsByTerm(ctx context.Context, subjectID, termID, schoolID string, isPlatformOwner bool) ([]dto.QuestionContextResponse, error) {
	if schoolID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}
	questions, err := s.qRepo.FindByTerm(ctx, subjectID, termID, schoolID)
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

// GetQuestionsBySession is scoped to schoolID - same fix as
// GetQuestionsByTerm.
func (s *QuestionService) GetQuestionsBySession(ctx context.Context, subjectID, sessionID, schoolID string, isPlatformOwner bool) ([]dto.QuestionContextResponse, error) {
	if schoolID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}
	questions, err := s.qRepo.FindBySession(ctx, subjectID, sessionID, schoolID)
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

// GetQuestionsByClass is scoped to schoolID - same fix as
// GetQuestionsByTerm.
func (s *QuestionService) GetQuestionsByClass(ctx context.Context, classID, schoolID string, isPlatformOwner bool) ([]dto.QuestionContextResponse, error) {
	if schoolID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}
	questions, err := s.qRepo.FindByClass(ctx, classID, schoolID)
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

// GetQuestionContextSummary is scoped to schoolID - subject is a global
// table shared by every school, so subjectID alone previously let any
// authenticated caller see every school's term-by-term question counts
// (and the questions themselves, via the unscoped FindByTerm below) for
// it.
func (s *QuestionService) GetQuestionContextSummary(ctx context.Context, subjectID, schoolID string, isPlatformOwner bool) (*dto.QuestionContextSummaryResponse, error) {
	if schoolID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}

	subject, err := s.subRepo.FindByID(subjectID)
	if err != nil {
		return nil, fmt.Errorf("subject not found: %w", err)
	}

	summaries, err := s.qRepo.GetQuestionContextSummary(ctx, subjectID, schoolID)
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
		questions, err := s.qRepo.FindByTerm(ctx, subjectID, summary.TermID, schoolID)
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

// GetQuestionsWithContext is scoped to schoolID - req.SchoolID previously
// came straight from the client with no override, so any authenticated
// caller could read another school's question bank just by naming its
// school_id in the request body.
func (s *QuestionService) GetQuestionsWithContext(ctx context.Context, req *dto.FilterQuestionsWithContextRequest, schoolID string, isPlatformOwner bool) (*dto.QuestionListWithContextResponse, error) {
	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID

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

// GetQuestionsForExam is scoped to schoolID - it previously queried with
// no school filter at all, so any authenticated caller could pull another
// school's questions for exam creation just by naming its term_id/
// session_id.
func (s *QuestionService) GetQuestionsForExam(ctx context.Context, subjectID, termID, sessionID, schoolID string, isPlatformOwner bool) ([]dto.QuestionBankResponse, error) {
	if schoolID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}

	var questions []models.QuestionBank
	query := s.db.WithContext(ctx).
		Where("subject_id = ? AND term_id = ? AND session_id = ? AND deleted_at IS NULL",
			subjectID, termID, sessionID)
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}
	err := query.Order("created_at DESC").Find(&questions).Error
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

// errQuestionSchoolScopeRequired mirrors
// internal/academic/service.ErrSchoolScopeRequired: a non-platform-owner
// caller with no school_id of their own (e.g. a legacy account seeded
// with none) has no school to scope a question-bank write to.
var errQuestionSchoolScopeRequired = errors.New("no school scope for this account")

// questionEffectiveSchoolID mirrors
// internal/academic/service.effectiveSchoolID (duplicated locally since
// that helper is unexported in a different package, same as
// ExamService's own authorizeExamTenant not reusing it either): an
// authenticated platform_owner's client-supplied schoolID is honored as
// given; every other caller's is always forced to ctxSchoolID, so a
// client-supplied school_id can never by itself grant write access to
// another school's question bank.
func questionEffectiveSchoolID(clientSuppliedSchoolID, ctxSchoolID string, isPlatformOwner bool) string {
	if isPlatformOwner {
		return clientSuppliedSchoolID
	}
	return ctxSchoolID
}

// questionAuthorizeSchoolOwnership mirrors
// internal/academic/service.authorizeSchoolOwnership for QuestionBank's
// own direct SchoolID column.
func questionAuthorizeSchoolOwnership(recordSchoolID, ctxSchoolID string, isPlatformOwner bool) error {
	if isPlatformOwner {
		return nil
	}
	if ctxSchoolID == "" || recordSchoolID != ctxSchoolID {
		return errors.New("not found")
	}
	return nil
}

// authorizeQuestionWrite verifies the authenticated caller is actually allowed
// to create/import a question under the given school/class, instead of trusting
// the school_id/class_id supplied in the request body. schoolID here must
// already be the caller's own resolved school (questionEffectiveSchoolID),
// never a raw client-supplied value - admins and super admins are trusted
// for THAT school, not platform-wide (matching the Phase 1/2 multi-tenant
// model: only platform_owner is meant to act across every school); teachers
// must be the assigned teacher of the class (the same ownership check
// already used throughout teacher_service.go via Class.TeacherID).
func (s *QuestionService) authorizeQuestionWrite(userID, schoolID, classID string) error {
	if userID == "" {
		return errors.New("unauthorized: user not authenticated")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("unauthorized: user not found")
	}

	switch user.Role {
	case models.RolePlatformOwner, models.RoleAdmin, models.RoleSuperAdmin:
		return nil
	case models.RoleTeacher:
		// fall through to the class-ownership check below
	default:
		return errors.New("unauthorized: only teachers and admins may manage the question bank")
	}

	if classID == "" {
		return errors.New("unauthorized: class_id is required")
	}
	class, err := s.classRepo.FindByID(classID)
	if err != nil {
		return errors.New("unauthorized: class not found")
	}
	if class.TeacherID == nil || *class.TeacherID != userID {
		return errors.New("unauthorized: you are not the assigned teacher for this class")
	}
	if schoolID != "" && class.SchoolID != schoolID {
		return errors.New("unauthorized: class does not belong to the specified school")
	}

	return nil
}

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
// GetQuestionsGrouped is scoped to schoolID - req.SchoolID previously came
// straight from the client with no override, so any authenticated user
// could read another school's entire question bank by passing its
// school_id. The caller's own resolved schoolID (questionEffectiveSchoolID)
// always replaces whatever req.SchoolID held before it reaches the filter
// or the response context.
func (s *QuestionService) GetQuestionsGrouped(ctx context.Context, req *dto.FilterQuestionsGroupedRequest, schoolID string, isPlatformOwner bool) (*dto.QuestionGroupsResponse, error) {
	effectiveID := questionEffectiveSchoolID(req.SchoolID, schoolID, isPlatformOwner)
	if effectiveID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}
	req.SchoolID = effectiveID

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
// GetAllQuestionsGrouped is scoped to schoolID - it previously ran
// s.db.Find(&questions) with no filter of any kind, so GET
// /questions/all-grouped returned literally every school's entire
// question bank (draft, published, and archived alike) to any
// authenticated caller. An empty schoolID is only honored for an
// authenticated platform_owner explicitly requesting every school;
// everyone else is forced to their own school and rejected outright if
// they have none.
func (s *QuestionService) GetAllQuestionsGrouped(ctx context.Context, schoolID string, isPlatformOwner bool) (*dto.QuestionGroupsResponse, error) {
	if schoolID == "" && !isPlatformOwner {
		return nil, errQuestionSchoolScopeRequired
	}

	// 1. Fetch ALL questions - NO status filter, NO deleted filter
	var questions []models.QuestionBank

	query := s.db.WithContext(ctx).Order("created_at DESC")
	if schoolID != "" {
		query = query.Where("school_id = ?", schoolID)
	}

	// Get ALL questions directly from database (including draft, published, archived)
	err := query.Find(&questions).Error

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
// GetQuestionsByTermWithGrouping is scoped to schoolID - it previously
// called FindByTerm with no school check at all, then derived req.SchoolID
// from the first matching question's OWN school (never verified against
// the caller), so a valid termID belonging to a different school leaked
// straight through. The term's own school (via its academic session) must
// match the caller's school before any question for it is returned.
func (s *QuestionService) GetQuestionsByTermWithGrouping(ctx context.Context, subjectID, termID, schoolID string, isPlatformOwner bool) (*dto.QuestionGroupsResponse, error) {
	var term models.Term
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", termID).First(&term).Error; err != nil {
		return nil, errors.New("term not found")
	}
	var session models.AcademicSession
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", term.SessionID).First(&session).Error; err != nil {
		return nil, errors.New("term not found")
	}
	if err := questionAuthorizeSchoolOwnership(session.SchoolID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}

	req := &dto.FilterQuestionsGroupedRequest{
		SubjectID: subjectID,
		TermID:    termID,
		SchoolID:  session.SchoolID,
	}

	questions, err := s.qRepo.FindByTerm(ctx, subjectID, termID, session.SchoolID)
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
	req.ClassID = firstQ.ClassID
	req.SessionID = firstQ.SessionID

	return s.GetQuestionsGrouped(ctx, req, schoolID, isPlatformOwner)
}

// GetQuestionsBySessionWithGrouping is scoped to schoolID - same fix as
// GetQuestionsByTermWithGrouping: the session's own school must match the
// caller's before any question for it is returned, instead of trusting
// the first matching question's own (unverified) school.
func (s *QuestionService) GetQuestionsBySessionWithGrouping(ctx context.Context, subjectID, sessionID, schoolID string, isPlatformOwner bool) (*dto.QuestionGroupsResponse, error) {
	var session models.AcademicSession
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", sessionID).First(&session).Error; err != nil {
		return nil, errors.New("session not found")
	}
	if err := questionAuthorizeSchoolOwnership(session.SchoolID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}

	req := &dto.FilterQuestionsGroupedRequest{
		SubjectID: subjectID,
		SessionID: sessionID,
		SchoolID:  session.SchoolID,
	}

	questions, err := s.qRepo.FindBySession(ctx, subjectID, sessionID, session.SchoolID)
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
	req.ClassID = firstQ.ClassID
	req.TermID = firstQ.TermID

	return s.GetQuestionsGrouped(ctx, req, schoolID, isPlatformOwner)
}


