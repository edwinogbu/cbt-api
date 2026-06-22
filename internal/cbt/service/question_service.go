package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"cbt-api/internal/ai/engine"
	"cbt-api/internal/ai/queue"
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/repository"
	"cbt-api/internal/models"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type QuestionService struct {
	qRepo   *repository.QuestionRepository
	subRepo *repository.SubjectRepository
	db      *gorm.DB
	queue   queue.Queue
	engine  *engine.Engine
}

func NewQuestionService(qRepo *repository.QuestionRepository, subRepo *repository.SubjectRepository, db *gorm.DB, queue queue.Queue,
	engine *engine.Engine) *QuestionService {
	return &QuestionService{
		qRepo:   qRepo,
		subRepo: subRepo,
		db:      db,
		queue:   queue,
		engine:  engine,
	}
}

// ============================================
// CRUD
// ============================================

func (s *QuestionService) CreateQuestion(req *dto.CreateQuestionRequest, userID string) (*dto.QuestionBankResponse, error) {
	questionID := uuid.New()

	var schoolID, classLevelID uuid.UUID
	var classID, sessionID, termID *uuid.UUID
	if req.SchoolID != "" {
		schoolID = uuid.MustParse(req.SchoolID)
	}
	if req.ClassLevelID != "" {
		classLevelID = uuid.MustParse(req.ClassLevelID)
	}
	if req.ClassID != "" {
		u := uuid.MustParse(req.ClassID)
		classID = &u
	}
	if req.SessionID != "" {
		u := uuid.MustParse(req.SessionID)
		sessionID = &u
	}
	if req.TermID != "" {
		u := uuid.MustParse(req.TermID)
		termID = &u
	}
	
	// Parse user ID from token
	createdBy := uuid.Nil
	if userID != "" {
		createdBy = uuid.MustParse(userID)
	}

	optsJSON := convertOptionsToJSON(req.OptionsArray)
	if req.OptionsArray == nil && req.Options != nil {
		var arr []dto.QuestionOption
		for k, v := range req.Options {
			arr = append(arr, dto.QuestionOption{Key: k, Text: v})
		}
		optsJSON = convertOptionsToJSON(arr)
	}

	q := &models.QuestionBank{
		ID:                questionID,
		SchoolID:          schoolID,
		ClassLevelID:      classLevelID,
		ClassID:           classID,
		SessionID:         sessionID,
		TermID:            termID,
		CurriculumType:    req.CurriculumType,
		SourceType:        req.SourceType,
		ExternalID:        req.ExternalID,
		SubjectID:         uuid.MustParse(req.SubjectID),
		Topic:             req.Topic,
		SubTopic:          req.SubTopic,
		LearningObjective: req.LearningObjective,
		QuestionText:      req.QuestionText,
		QuestionType:      models.QuestionType(req.QuestionType),
		Difficulty:        models.DifficultyLevel(req.Difficulty),
		BloomLevel:        models.BloomTaxonomy(req.BloomLevel),
		Options:           optsJSON,
		CorrectAnswer:     req.CorrectAnswer,
		CorrectOptionKeys: req.CorrectOptionKeys,
		Rubric:            convertRubricToJSON(req.Rubric),
		Explanation:       req.Explanation,
		Marks:             req.Marks,
		NegativeMarks:     req.NegativeMarks,
		TimeLimitSeconds:  req.TimeLimitSeconds,
		Order:             req.Order,
		IsRequired:        req.IsRequired,
		Status:            models.QuestionStatusDraft,
		Version:           1,
		CreatedBy:         createdBy,
		UpdatedBy:         createdBy,
	}

	if err := s.qRepo.Create(q); err != nil {
		return nil, err
	}
	if len(req.Tags) > 0 {
		if err := s.attachTagsByNames(questionID, req.Tags); err != nil {
			return nil, err
		}
	}
	return s.toResponseWithSubject(q), nil
}

func (s *QuestionService) GetQuestion(id string) (*dto.QuestionBankResponse, error) {
	qID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid question ID")
	}
	q, err := s.qRepo.FindByID(qID)
	if err != nil {
		return nil, errors.New("question not found")
	}
	return s.toResponseWithSubject(q), nil
}

func (s *QuestionService) UpdateQuestion(id string, req *dto.UpdateQuestionRequest, userID string) (*dto.QuestionBankResponse, error) {
	qID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid question ID")
	}
	q, err := s.qRepo.FindByID(qID)
	if err != nil {
		return nil, errors.New("question not found")
	}

	updatedBy := uuid.Nil
	if userID != "" {
		updatedBy = uuid.MustParse(userID)
	}

	updates := make(map[string]interface{})
	if req.QuestionText != nil {
		updates["question_text"] = *req.QuestionText
	}
	if req.Options != nil {
		var arr []dto.QuestionOption
		for k, v := range req.Options {
			arr = append(arr, dto.QuestionOption{Key: k, Text: v})
		}
		updates["options"] = convertOptionsToJSON(arr)
	}
	if req.OptionsArray != nil {
		updates["options"] = convertOptionsToJSON(req.OptionsArray)
	}
	if req.CorrectAnswer != nil {
		updates["correct_answer"] = *req.CorrectAnswer
	}
	if req.CorrectOptionKeys != nil {
		updates["correct_option_keys"] = req.CorrectOptionKeys
	}
	if req.Rubric != nil {
		updates["rubric"] = convertRubricToJSON(req.Rubric)
	}
	if req.Explanation != nil {
		updates["explanation"] = *req.Explanation
	}
	if req.Marks != nil {
		updates["marks"] = *req.Marks
	}
	if req.Difficulty != nil {
		updates["difficulty"] = *req.Difficulty
	}
	if req.BloomLevel != nil {
		updates["bloom_level"] = *req.BloomLevel
	}
	if req.TimeLimitSeconds != nil {
		updates["time_limit_seconds"] = *req.TimeLimitSeconds
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Topic != nil {
		updates["topic"] = *req.Topic
	}
	if req.SubTopic != nil {
		updates["sub_topic"] = *req.SubTopic
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
	
	// Add updated_by
	updates["updated_by"] = updatedBy

	if len(updates) == 0 {
		return s.toResponseWithSubject(q), nil
	}

	newID, err := s.qRepo.CreateNewVersion(q, updates)
	if err != nil {
		return nil, err
	}
	newQ, err := s.qRepo.FindByID(newID)
	if err != nil {
		return nil, err
	}
	return s.toResponseWithSubject(newQ), nil
}

func (s *QuestionService) DeleteQuestion(id string) error {
	qID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid question ID")
	}
	return s.qRepo.Delete(qID)
}

func (s *QuestionService) ListQuestions(subjectID string, page, limit int) ([]dto.QuestionBankResponse, int64, error) {
	subj, err := uuid.Parse(subjectID)
	if err != nil {
		return nil, 0, errors.New("invalid subject ID")
	}
	qs, total, err := s.qRepo.ListBySubject(subj, page, limit)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]dto.QuestionBankResponse, 0, len(qs))
	for _, q := range qs {
		r := s.toResponseWithSubject(&q)
		resp = append(resp, *r)
	}
	return resp, total, nil
}

func (s *QuestionService) FilterQuestions(req *dto.FilterQuestionsRequest) ([]dto.QuestionBankResponse, int64, error) {
	params := map[string]interface{}{
		"subject_id":      req.SubjectID,
		"school_id":       req.SchoolID,
		"class_level_id":  req.ClassLevelID,
		"session_id":      req.SessionID,
		"term_id":         req.TermID,
		"topic":           req.Topic,
		"difficulty":      strings.Join(req.Difficulty, ","),
		"bloom_level":     strings.Join(req.BloomLevel, ","),
		"question_type":   strings.Join(req.QuestionType, ","),
		"status":          req.Status,
		"search":          req.Search,
	}
	qs, total, err := s.qRepo.Filter(params, req.Page, req.Limit)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]dto.QuestionBankResponse, 0, len(qs))
	for _, q := range qs {
		r := s.toResponseWithSubject(&q)
		resp = append(resp, *r)
	}
	return resp, total, nil
}

func (s *QuestionService) BulkDelete(req *dto.BulkDeleteRequest) error {
	ids := make([]uuid.UUID, len(req.QuestionIDs))
	for i, idStr := range req.QuestionIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return fmt.Errorf("invalid ID: %s", idStr)
		}
		ids[i] = id
	}
	return s.qRepo.BulkDelete(ids)
}

func (s *QuestionService) BulkUpdateStatus(ids []string, status string) error {
	uuids := make([]uuid.UUID, len(ids))
	for i, idStr := range ids {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return err
		}
		uuids[i] = id
	}
	return s.qRepo.BulkUpdateStatus(uuids, status)
}

func (s *QuestionService) CreateTag(req *dto.CreateTagRequest) (*dto.TagResponse, error) {
	tag := &models.Tag{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        strings.ReplaceAll(strings.ToLower(req.Name), " ", "-"),
		Description: req.Description,
	}
	if err := s.qRepo.CreateTag(tag); err != nil {
		return nil, err
	}
	return &dto.TagResponse{
		ID:          tag.ID.String(),
		Name:        tag.Name,
		Slug:        tag.Slug,
		Description: tag.Description,
		CreatedAt:   tag.CreatedAt,
	}, nil
}

func (s *QuestionService) ListTags() ([]dto.TagResponse, error) {
	tags, err := s.qRepo.ListTags()
	if err != nil {
		return nil, err
	}
	resp := make([]dto.TagResponse, len(tags))
	for i, t := range tags {
		resp[i] = dto.TagResponse{
			ID:          t.ID.String(),
			Name:        t.Name,
			Slug:        t.Slug,
			Description: t.Description,
			UsageCount:  t.UsageCount,
			CreatedAt:   t.CreatedAt,
		}
	}
	return resp, nil
}

func (s *QuestionService) GetStatistics(subjectID string) (map[string]interface{}, error) {
	var subj uuid.UUID
	if subjectID != "" {
		var err error
		subj, err = uuid.Parse(subjectID)
		if err != nil {
			return nil, errors.New("invalid subject ID")
		}
	}
	return s.qRepo.GetStatistics(subj)
}

// ============================================
// BULK CREATE FROM JSON
// ============================================

func (s *QuestionService) BulkCreateQuestionsFromJSON(req *dto.BulkCreateQuestionRequest, userID string) ([]dto.QuestionBankResponse, error) {
	if len(req.Questions) == 0 {
		return nil, errors.New("no questions provided")
	}

	createdBy := uuid.Nil
	if userID != "" {
		createdBy = uuid.MustParse(userID)
	}

	var responses []dto.QuestionBankResponse

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txQRepo := repository.NewQuestionRepository(tx)
		for _, qReq := range req.Questions {
			questionID := uuid.New()
			var optsJSON models.JSONMap
			if qReq.OptionsArray != nil {
				optsJSON = convertOptionsToJSON(qReq.OptionsArray)
			} else if qReq.Options != nil {
				var arr []dto.QuestionOption
				for k, v := range qReq.Options {
					arr = append(arr, dto.QuestionOption{Key: k, Text: v})
				}
				optsJSON = convertOptionsToJSON(arr)
			}
			schoolID, _ := uuid.Parse(qReq.SchoolID)
			classLevelID, _ := uuid.Parse(qReq.ClassLevelID)

			q := &models.QuestionBank{
				ID:                questionID,
				SchoolID:          schoolID,
				ClassLevelID:      classLevelID,
				ClassID:           parseOptionalUUID(qReq.ClassID),
				SessionID:         parseOptionalUUID(qReq.SessionID),
				TermID:            parseOptionalUUID(qReq.TermID),
				CurriculumType:    qReq.CurriculumType,
				SourceType:        qReq.SourceType,
				ExternalID:        qReq.ExternalID,
				SubjectID:         uuid.MustParse(qReq.SubjectID),
				Topic:             qReq.Topic,
				SubTopic:          qReq.SubTopic,
				LearningObjective: qReq.LearningObjective,
				QuestionText:      qReq.QuestionText,
				QuestionType:      models.QuestionType(qReq.QuestionType),
				Difficulty:        models.DifficultyLevel(qReq.Difficulty),
				BloomLevel:        models.BloomTaxonomy(qReq.BloomLevel),
				Options:           optsJSON,
				CorrectAnswer:     qReq.CorrectAnswer,
				CorrectOptionKeys: qReq.CorrectOptionKeys,
				Rubric:            convertRubricToJSON(qReq.Rubric),
				Explanation:       qReq.Explanation,
				Marks:             qReq.Marks,
				NegativeMarks:     qReq.NegativeMarks,
				TimeLimitSeconds:  qReq.TimeLimitSeconds,
				Order:             qReq.Order,
				IsRequired:        qReq.IsRequired,
				Status:            models.QuestionStatusDraft,
				Version:           1,
				CreatedBy:         createdBy,
				UpdatedBy:         createdBy,
			}
			if err := txQRepo.Create(q); err != nil {
				return fmt.Errorf("failed to create question: %w", err)
			}
			if len(qReq.Tags) > 0 {
				if err := s.attachTagsByNamesInTx(tx, questionID, qReq.Tags); err != nil {
					return err
				}
			}
			responses = append(responses, *s.toResponseLight(q))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i, resp := range responses {
		subj, _ := s.subRepo.FindByID(uuid.MustParse(resp.SubjectID))
		if subj != nil {
			responses[i].SubjectName = subj.Name
		}
	}
	return responses, nil
}

func parseOptionalUUID(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	u := uuid.MustParse(s)
	return &u
}

// ============================================
// BULK UPLOAD FROM FILE
// ============================================

func (s *QuestionService) BulkUploadFromFile(file io.Reader, format, subjectIDStr string, hasHeader bool, userID string) (*dto.BulkUploadResponse, error) {
	subjectID, err := uuid.Parse(subjectIDStr)
	if err != nil {
		return nil, errors.New("invalid subject_id")
	}

	createdBy := uuid.Nil
	if userID != "" {
		createdBy = uuid.MustParse(userID)
	}

	var rows []dto.CSVQuestionRow
	switch format {
	case "csv":
		rows, err = s.parseCSV(file, hasHeader)
	case "json":
		rows, err = s.parseJSON(file)
	case "excel":
		rows, err = s.parseExcel(file)
	default:
		return nil, errors.New("unsupported format, use csv, json, or excel")
	}
	if err != nil {
		return nil, err
	}

	resp := &dto.BulkUploadResponse{
		TotalProcessed: len(rows),
		Errors:         []string{},
	}

	for i, row := range rows {
		if row.QuestionText == "" || row.CorrectAnswer == "" {
			resp.FailedCount++
			resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: missing question text or correct answer", i+1))
			continue
		}
		opts := make(map[string]string)
		if row.OptionA != "" {
			opts["A"] = row.OptionA
		}
		if row.OptionB != "" {
			opts["B"] = row.OptionB
		}
		if row.OptionC != "" {
			opts["C"] = row.OptionC
		}
		if row.OptionD != "" {
			opts["D"] = row.OptionD
		}

		difficulty := models.DifficultyLevel(row.Difficulty)
		if difficulty == "" {
			difficulty = models.DifficultyMedium
		}
		bloom := models.BloomTaxonomy(row.BloomLevel)
		if bloom == "" {
			bloom = models.BloomRemember
		}
		qType := models.QuestionType(row.QuestionType)
		if qType == "" {
			qType = models.QuestionTypeSingle
		}
		marks := row.Marks
		if marks == 0 {
			marks = 1
		}

		var optsArr []dto.QuestionOption
		for k, v := range opts {
			optsArr = append(optsArr, dto.QuestionOption{Key: k, Text: v})
		}
		optsJSON := convertOptionsToJSON(optsArr)

		q := &models.QuestionBank{
			ID:            uuid.New(),
			SubjectID:     subjectID,
			Topic:         row.Topic,
			SubTopic:      row.SubTopic,
			QuestionText:  row.QuestionText,
			QuestionType:  qType,
			Difficulty:    difficulty,
			BloomLevel:    bloom,
			Options:       optsJSON,
			CorrectAnswer: row.CorrectAnswer,
			Explanation:   row.Explanation,
			Marks:         marks,
			Status:        models.QuestionStatusDraft,
			Version:       1,
			CreatedBy:     createdBy,
			UpdatedBy:     createdBy,
		}
		if err := s.qRepo.Create(q); err != nil {
			resp.FailedCount++
			resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: %v", i+1, err))
			continue
		}
		resp.SuccessCount++
	}
	return resp, nil
}

// ============================================
// Parsers
// ============================================

func (s *QuestionService) parseCSV(file io.Reader, hasHeader bool) ([]dto.CSVQuestionRow, error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.New("empty CSV")
	}
	start := 0
	if hasHeader {
		start = 1
	}
	var rows []dto.CSVQuestionRow
	for i := start; i < len(records); i++ {
		row := records[i]
		if len(row) < 14 {
			continue
		}
		rows = append(rows, dto.CSVQuestionRow{
			QuestionText:  row[0],
			OptionA:       row[1],
			OptionB:       row[2],
			OptionC:       row[3],
			OptionD:       row[4],
			CorrectAnswer: row[5],
			Explanation:   row[6],
			Marks:         parseInt(row[7]),
			Topic:         row[8],
			SubTopic:      row[9],
			Difficulty:    row[10],
			BloomLevel:    row[11],
			QuestionType:  row[12],
			SubjectID:     row[13],
		})
	}
	if len(rows) == 0 {
		return nil, errors.New("no valid data rows found (expected 14 columns)")
	}
	return rows, nil
}

func (s *QuestionService) parseJSON(file io.Reader) ([]dto.CSVQuestionRow, error) {
	var importData dto.JSONQuestionImport
	if err := json.NewDecoder(file).Decode(&importData); err != nil {
		return nil, err
	}
	var rows []dto.CSVQuestionRow
	for _, q := range importData.Questions {
		rows = append(rows, dto.CSVQuestionRow{
			QuestionText:  q.QuestionText,
			OptionA:       q.OptionA,
			OptionB:       q.OptionB,
			OptionC:       q.OptionC,
			OptionD:       q.OptionD,
			CorrectAnswer: q.CorrectAnswer,
			Explanation:   q.Explanation,
			Marks:         q.Marks,
			Topic:         q.Topic,
			SubTopic:      q.SubTopic,
			Difficulty:    q.Difficulty,
			BloomLevel:    q.BloomLevel,
			QuestionType:  q.QuestionType,
			SubjectID:     q.SubjectID,
		})
	}
	return rows, nil
}

func (s *QuestionService) parseExcel(file io.Reader) ([]dto.CSVQuestionRow, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, errors.New("Excel file must have header + data")
	}
	var result []dto.CSVQuestionRow
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 14 {
			continue
		}
		result = append(result, dto.CSVQuestionRow{
			QuestionText:  row[0],
			OptionA:       row[1],
			OptionB:       row[2],
			OptionC:       row[3],
			OptionD:       row[4],
			CorrectAnswer: row[5],
			Explanation:   row[6],
			Marks:         parseInt(row[7]),
			Topic:         row[8],
			SubTopic:      row[9],
			Difficulty:    row[10],
			BloomLevel:    row[11],
			QuestionType:  row[12],
			SubjectID:     row[13],
		})
	}
	return result, nil
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// ============================================
// Helpers – Tag attachment
// ============================================

func (s *QuestionService) attachTagsByNames(questionID uuid.UUID, tagNames []string) error {
	for _, name := range tagNames {
		tag, err := s.qRepo.FindTagByName(name)
		if err != nil {
			tag = &models.Tag{
				ID:   uuid.New(),
				Name: name,
				Slug: strings.ReplaceAll(strings.ToLower(name), " ", "-"),
			}
			if err := s.qRepo.CreateTag(tag); err != nil {
				return err
			}
		}
		if err := s.qRepo.AttachTags(questionID, []uuid.UUID{tag.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (s *QuestionService) attachTagsByNamesInTx(tx *gorm.DB, questionID uuid.UUID, tagNames []string) error {
	for _, name := range tagNames {
		var tag models.Tag
		err := tx.Where("name = ?", name).First(&tag).Error
		if err != nil {
			tag = models.Tag{
				ID:   uuid.New(),
				Name: name,
				Slug: strings.ReplaceAll(strings.ToLower(name), " ", "-"),
			}
			if err := tx.Create(&tag).Error; err != nil {
				return err
			}
		}
		mapping := models.QuestionTagMapping{
			ID:         uuid.New(),
			QuestionID: questionID,
			TagID:      tag.ID,
		}
		if err := tx.Create(&mapping).Error; err != nil {
			return err
		}
	}
	return nil
}

// ============================================
// Response Builders
// ============================================

func (s *QuestionService) toQuestionBankResponse(q *models.QuestionBank) *dto.QuestionBankResponse {
	var opts []dto.QuestionOption
	if q.Options != nil {
		if jsonStr, ok := q.Options["_json"].(string); ok && jsonStr != "" {
			var arr []map[string]string
			if err := json.Unmarshal([]byte(jsonStr), &arr); err == nil {
				for _, item := range arr {
					opts = append(opts, dto.QuestionOption{
						Key:  item["key"],
						Text: item["text"],
					})
				}
			}
		} else if optArr, ok := q.Options["options"].([]interface{}); ok {
			for _, item := range optArr {
				if m, ok := item.(map[string]interface{}); ok {
					opts = append(opts, dto.QuestionOption{
						Key:  m["key"].(string),
						Text: m["text"].(string),
					})
				}
			}
		}
	}

	var rubric []dto.RubricCriteria
	if q.Rubric != nil {
		if jsonStr, ok := q.Rubric["_json"].(string); ok && jsonStr != "" {
			var arr []map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &arr); err == nil {
				for _, item := range arr {
					rubric = append(rubric, dto.RubricCriteria{
						Criteria: item["criteria"].(string),
						Marks:    int(item["marks"].(float64)),
					})
				}
			}
		} else if rubArr, ok := q.Rubric["rubric"].([]interface{}); ok {
			for _, item := range rubArr {
				if m, ok := item.(map[string]interface{}); ok {
					rubric = append(rubric, dto.RubricCriteria{
						Criteria: m["criteria"].(string),
						Marks:    int(m["marks"].(float64)),
					})
				}
			}
		}
	}

	var tags []string
	if q.Tags != nil {
		if jsonStr, ok := q.Tags["_json"].(string); ok && jsonStr != "" {
			json.Unmarshal([]byte(jsonStr), &tags)
		} else if tagArr, ok := q.Tags["tags"].([]interface{}); ok {
			for _, t := range tagArr {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
		}
	}

	return &dto.QuestionBankResponse{
		ID:                q.ID.String(),
		SubjectID:         q.SubjectID.String(),
		SubjectName:       "",
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
		CreatedBy:         q.CreatedBy.String(),
		CreatedByName:     "",
		SchoolID:          q.SchoolID.String(),
		ClassLevelID:      q.ClassLevelID.String(),
		ClassID:           nilToPtr(q.ClassID),
		SessionID:         nilToPtr(q.SessionID),
		TermID:            nilToPtr(q.TermID),
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

func nilToPtr(u *uuid.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

func (s *QuestionService) toResponseWithSubject(q *models.QuestionBank) *dto.QuestionBankResponse {
	resp := s.toQuestionBankResponse(q)
	subject, err := s.subRepo.FindByID(q.SubjectID)
	if err == nil && subject != nil {
		resp.SubjectName = subject.Name
	}
	return resp
}

func (s *QuestionService) toResponseLight(q *models.QuestionBank) *dto.QuestionBankResponse {
	return s.toQuestionBankResponse(q)
}

// ============================================
// Converters for JSON storage - FIXED
// ============================================

func convertOptionsToJSON(opts []dto.QuestionOption) models.JSONMap {
	if opts == nil || len(opts) == 0 {
		return models.JSONMap{}
	}
	arr := make([]map[string]string, len(opts))
	for i, o := range opts {
		arr[i] = map[string]string{"key": o.Key, "text": o.Text}
	}
	return models.JSONMap{
		"": arr,
	}
}

func convertRubricToJSON(rubric []dto.RubricCriteria) models.JSONMap {
	if rubric == nil || len(rubric) == 0 {
		return models.JSONMap{}
	}
	arr := make([]map[string]interface{}, len(rubric))
	for i, r := range rubric {
		arr[i] = map[string]interface{}{"criteria": r.Criteria, "marks": r.Marks}
	}
	return models.JSONMap{
		"": arr,
	}
}

func convertTagsToJSON(tags []string) models.JSONMap {
	if tags == nil || len(tags) == 0 {
		return models.JSONMap{}
	}
	return models.JSONMap{
		"": tags,
	}
}

// ============================================
// NEW: Bulk Import (Exact JSON)
// ============================================

func (s *QuestionService) BulkImportQuestions(req *dto.BulkQuestionImportRequest) ([]dto.QuestionBankResponse, error) {
	if len(req.Questions) == 0 {
		return nil, errors.New("no questions provided")
	}

	var responses []dto.QuestionBankResponse

	schoolID, err := uuid.Parse(req.SchoolID)
	if err != nil {
		return nil, errors.New("invalid school_id")
	}
	classLevelID, err := uuid.Parse(req.ClassLevelID)
	if err != nil {
		return nil, errors.New("invalid class_level_id")
	}
	var classID, sessionID, termID *uuid.UUID
	if req.ClassID != "" {
		u, err := uuid.Parse(req.ClassID)
		if err != nil {
			return nil, errors.New("invalid class_id")
		}
		classID = &u
	}
	if req.SessionID != "" {
		u, err := uuid.Parse(req.SessionID)
		if err != nil {
			return nil, errors.New("invalid session_id")
		}
		sessionID = &u
	}
	if req.TermID != "" {
		u, err := uuid.Parse(req.TermID)
		if err != nil {
			return nil, errors.New("invalid term_id")
		}
		termID = &u
	}
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, errors.New("invalid created_by")
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := repository.NewQuestionRepository(tx)

		for idx, item := range req.Questions {
			if err := validateQuestionItem(item); err != nil {
				return fmt.Errorf("question %d: %w", idx+1, err)
			}

			subjectID, err := uuid.Parse(item.SubjectID)
			if err != nil {
				return fmt.Errorf("question %d: invalid subject_id", idx+1)
			}

			var existing *models.QuestionBank
			if item.ExternalID != "" {
				var sessID uuid.UUID
				if sessionID != nil {
					sessID = *sessionID
				}
				existing, _ = txRepo.FindByExternalID(schoolID, sessID, item.ExternalID)
			}

			optsJSON := convertOptionsToJSON(item.Options)
			rubricJSON := convertRubricToJSON(item.Rubric)
			tagsJSON := convertTagsToJSON(item.Tags)

			if existing != nil {
				updates := map[string]interface{}{
					"topic":               item.Topic,
					"sub_topic":           item.SubTopic,
					"learning_objective":  item.LearningObjective,
					"question_text":       item.QuestionText,
					"question_type":       item.QuestionType,
					"difficulty":          item.Difficulty,
					"bloom_level":         item.BloomLevel,
					"options":             optsJSON,
					"correct_option_keys": item.CorrectOptionKeys,
					"rubric":              rubricJSON,
					"explanation":         item.Explanation,
					"marks":               item.Marks,
					"negative_marks":      item.NegativeMarks,
					"time_limit_seconds":  item.TimeLimitSeconds,
					"order":               item.Order,
					"is_required":         item.IsRequired,
					"updated_by":          createdBy,
					"tags":                tagsJSON,
					"status":              req.Status,
					"curriculum_type":     req.CurriculumType,
					"source_type":         req.SourceType,
				}
				newID, err := txRepo.CreateNewVersion(existing, updates)
				if err != nil {
					return fmt.Errorf("failed to update version for question %d: %w", idx+1, err)
				}
				existing, err = txRepo.FindByID(newID)
				if err != nil {
					return fmt.Errorf("failed to fetch updated question %d: %w", idx+1, err)
				}
				if len(item.Tags) > 0 {
					if err := s.attachTagsByNamesInTx(tx, existing.ID, item.Tags); err != nil {
						return fmt.Errorf("failed to attach tags for question %d: %w", idx+1, err)
					}
				}
			} else {
				q := &models.QuestionBank{
					ID:                uuid.New(),
					SchoolID:          schoolID,
					ClassLevelID:      classLevelID,
					ClassID:           classID,
					SessionID:         sessionID,
					TermID:            termID,
					CurriculumType:    req.CurriculumType,
					SourceType:        req.SourceType,
					ExternalID:        item.ExternalID,
					SubjectID:         subjectID,
					Topic:             item.Topic,
					SubTopic:          item.SubTopic,
					LearningObjective: item.LearningObjective,
					QuestionText:      item.QuestionText,
					QuestionType:      models.QuestionType(item.QuestionType),
					Difficulty:        models.DifficultyLevel(item.Difficulty),
					BloomLevel:        models.BloomTaxonomy(item.BloomLevel),
					Options:           optsJSON,
					CorrectOptionKeys: item.CorrectOptionKeys,
					Rubric:            rubricJSON,
					Explanation:       item.Explanation,
					Marks:             item.Marks,
					NegativeMarks:     item.NegativeMarks,
					TimeLimitSeconds:  &item.TimeLimitSeconds,
					Order:             item.Order,
					IsRequired:        item.IsRequired,
					Tags:              tagsJSON,
					Status:            models.QuestionStatus(req.Status),
					Version:           1,
					CreatedBy:         createdBy,
					UpdatedBy:         createdBy,
				}
				if err := txRepo.Create(q); err != nil {
					return fmt.Errorf("failed to create question %d: %w", idx+1, err)
				}
				if len(item.Tags) > 0 {
					if err := s.attachTagsByNamesInTx(tx, q.ID, item.Tags); err != nil {
						return fmt.Errorf("failed to attach tags for question %d: %w", idx+1, err)
					}
				}
				existing = q
			}

			resp := s.toQuestionBankResponse(existing)
			responses = append(responses, *resp)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	for i := range responses {
		subj, _ := s.subRepo.FindByID(uuid.MustParse(responses[i].SubjectID))
		if subj != nil {
			responses[i].SubjectName = subj.Name
		}
	}
	return responses, nil
}

func validateQuestionItem(item dto.QuestionImportItem) error {
	switch item.QuestionType {
	case "single_choice", "multiple_choice", "true_false":
		if len(item.Options) == 0 {
			return errors.New("MCQ/true_false must have options")
		}
		if len(item.CorrectOptionKeys) == 0 {
			return errors.New("MCQ/true_false must have correct option keys")
		}
		if item.Rubric != nil && len(item.Rubric) > 0 {
			return errors.New("MCQ/true_false cannot have rubric")
		}
	case "essay":
		if item.Options != nil && len(item.Options) > 0 {
			return errors.New("essay cannot have options")
		}
		if item.CorrectOptionKeys != nil && len(item.CorrectOptionKeys) > 0 {
			return errors.New("essay cannot have correct option keys")
		}
		if item.Rubric == nil || len(item.Rubric) == 0 {
			return errors.New("essay must have rubric")
		}
	case "fill_blank":
		// optional
	}
	return nil
}

// ============================================
// AI Methods
// ============================================

func (s *QuestionService) GenerateQuestionsWithAI(req *dto.AIGenerateQuestionsRequest) (*dto.AIQuestionGenerationResponse, error) {
	job := &models.AIQuestionGenerationJob{
		ID:                uuid.New(),
		UserID:            uuid.Nil,
		SubjectID:         uuid.MustParse(req.SubjectID),
		Topic:             req.Topic,
		NumberOfQuestions: req.NumberOfQuestions,
		Difficulty:        models.DifficultyLevel(req.Difficulty),
		BloomLevel:        models.BloomTaxonomy(req.BloomLevel),
		SourceText:        req.SourceText,
		Status:            "queued",
	}

	if err := s.db.Create(job).Error; err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"job_id":  job.ID,
		"type":    "generate",
		"request": req,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := s.queue.Push(ctx, "ai_jobs", string(data)); err != nil {
		return nil, err
	}

	return &dto.AIQuestionGenerationResponse{
		JobID:   job.ID.String(),
		Status:  "queued",
		Message: "Job enqueued successfully",
	}, nil
}

func (s *QuestionService) ExtractQuestionsFromText(req *dto.ExtractTextQuestionsRequest) (*dto.AIQuestionGenerationResponse, error) {
	job := &models.AIQuestionGenerationJob{
		ID:         uuid.New(),
		UserID:     uuid.Nil,
		SubjectID:  uuid.MustParse(req.SubjectID),
		SourceText: req.Text,
		Status:     "queued",
	}
	if err := s.db.Create(job).Error; err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"job_id":  job.ID,
		"type":    "extract",
		"text":    req.Text,
		"school":  req.SchoolID,
		"class":   req.ClassLevelID,
		"subject": req.SubjectID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := s.queue.Push(context.Background(), "ai_jobs", string(data)); err != nil {
		return nil, err
	}

	return &dto.AIQuestionGenerationResponse{
		JobID:   job.ID.String(),
		Status:  "queued",
		Message: "Extraction job enqueued",
	}, nil
}

func (s *QuestionService) GetJobStatus(jobID string) (*dto.AIJobStatusResponse, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return nil, errors.New("invalid job ID")
	}
	var job models.AIQuestionGenerationJob
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		return nil, errors.New("job not found")
	}
	return &dto.AIJobStatusResponse{
		JobID:        job.ID.String(),
		Status:       job.Status,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		CompletedAt:  job.CompletedAt,
	}, nil
}



// package service

// import (
// 	"context" // ✅ ADDED - required for queue operations
// 	"encoding/csv"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"strings"

// 	"cbt-api/internal/ai/engine"
// 	"cbt-api/internal/ai/queue"
// 	"cbt-api/internal/cbt/dto"
// 	"cbt-api/internal/cbt/repository"
// 	"cbt-api/internal/models"

// 	"github.com/google/uuid"
// 	"github.com/xuri/excelize/v2"
// 	"gorm.io/gorm"
// )

// type QuestionService struct {
// 	qRepo   *repository.QuestionRepository
// 	subRepo *repository.SubjectRepository
// 	db      *gorm.DB
// 	queue   queue.Queue
// 	engine  *engine.Engine
// }

// func NewQuestionService(qRepo *repository.QuestionRepository, subRepo *repository.SubjectRepository, db *gorm.DB, queue queue.Queue,
// 	engine *engine.Engine) *QuestionService {
// 	return &QuestionService{
// 		qRepo:   qRepo,
// 		subRepo: subRepo,
// 		db:      db,
// 		queue:   queue,
// 		engine:  engine,
// 	}
// }

// // ============================================
// // CRUD
// // ============================================

// func (s *QuestionService) CreateQuestion(req *dto.CreateQuestionRequest) (*dto.QuestionBankResponse, error) {
// 	questionID := uuid.New()

// 	// Parse new fields if provided, else use zero values
// 	var schoolID, classLevelID uuid.UUID
// 	var classID, sessionID, termID *uuid.UUID
// 	if req.SchoolID != "" {
// 		schoolID = uuid.MustParse(req.SchoolID)
// 	}
// 	if req.ClassLevelID != "" {
// 		classLevelID = uuid.MustParse(req.ClassLevelID)
// 	}
// 	if req.ClassID != "" {
// 		u := uuid.MustParse(req.ClassID)
// 		classID = &u
// 	}
// 	if req.SessionID != "" {
// 		u := uuid.MustParse(req.SessionID)
// 		sessionID = &u
// 	}
// 	if req.TermID != "" {
// 		u := uuid.MustParse(req.TermID)
// 		termID = &u
// 	}
// 	createdBy := uuid.Nil // will be set from context later

// 	// Convert options: if OptionsArray provided, use that, else fallback to legacy map
// 	optsJSON := convertOptionsToJSON(req.OptionsArray)
// 	if req.OptionsArray == nil && req.Options != nil {
// 		// Convert flat map to array format for consistency
// 		var arr []dto.QuestionOption
// 		for k, v := range req.Options {
// 			arr = append(arr, dto.QuestionOption{Key: k, Text: v})
// 		}
// 		optsJSON = convertOptionsToJSON(arr)
// 	}

// 	// Build the question
// 	q := &models.QuestionBank{
// 		ID:                questionID,
// 		SchoolID:          schoolID,
// 		ClassLevelID:      classLevelID,
// 		ClassID:           classID,
// 		SessionID:         sessionID,
// 		TermID:            termID,
// 		CurriculumType:    req.CurriculumType,
// 		SourceType:        req.SourceType,
// 		ExternalID:        req.ExternalID,
// 		SubjectID:         uuid.MustParse(req.SubjectID),
// 		Topic:             req.Topic,
// 		SubTopic:          req.SubTopic,
// 		LearningObjective: req.LearningObjective,
// 		QuestionText:      req.QuestionText,
// 		QuestionType:      models.QuestionType(req.QuestionType),
// 		Difficulty:        models.DifficultyLevel(req.Difficulty),
// 		BloomLevel:        models.BloomTaxonomy(req.BloomLevel),
// 		Options:           optsJSON,
// 		CorrectAnswer:     req.CorrectAnswer,
// 		CorrectOptionKeys: req.CorrectOptionKeys,
// 		Rubric:            convertRubricToJSON(req.Rubric),
// 		Explanation:       req.Explanation,
// 		Marks:             req.Marks,
// 		NegativeMarks:     req.NegativeMarks,
// 		TimeLimitSeconds:  req.TimeLimitSeconds,
// 		Order:             req.Order,
// 		IsRequired:        req.IsRequired,
// 		Status:            models.QuestionStatusDraft,
// 		Version:           1,
// 		CreatedBy:         createdBy,
// 		UpdatedBy:         createdBy,
// 	}

// 	if err := s.qRepo.Create(q); err != nil {
// 		return nil, err
// 	}
// 	if len(req.Tags) > 0 {
// 		if err := s.attachTagsByNames(questionID, req.Tags); err != nil {
// 			return nil, err
// 		}
// 	}
// 	return s.toResponseWithSubject(q), nil
// }

// func (s *QuestionService) GetQuestion(id string) (*dto.QuestionBankResponse, error) {
// 	qID, err := uuid.Parse(id)
// 	if err != nil {
// 		return nil, errors.New("invalid question ID")
// 	}
// 	q, err := s.qRepo.FindByID(qID)
// 	if err != nil {
// 		return nil, errors.New("question not found")
// 	}
// 	return s.toResponseWithSubject(q), nil
// }

// func (s *QuestionService) UpdateQuestion(id string, req *dto.UpdateQuestionRequest) (*dto.QuestionBankResponse, error) {
// 	qID, err := uuid.Parse(id)
// 	if err != nil {
// 		return nil, errors.New("invalid question ID")
// 	}
// 	q, err := s.qRepo.FindByID(qID)
// 	if err != nil {
// 		return nil, errors.New("question not found")
// 	}

// 	// Build updates map
// 	updates := make(map[string]interface{})
// 	if req.QuestionText != nil {
// 		updates["question_text"] = *req.QuestionText
// 	}
// 	if req.Options != nil {
// 		// convert flat to array
// 		var arr []dto.QuestionOption
// 		for k, v := range req.Options {
// 			arr = append(arr, dto.QuestionOption{Key: k, Text: v})
// 		}
// 		updates["options"] = convertOptionsToJSON(arr)
// 	}
// 	if req.OptionsArray != nil {
// 		updates["options"] = convertOptionsToJSON(req.OptionsArray)
// 	}
// 	if req.CorrectAnswer != nil {
// 		updates["correct_answer"] = *req.CorrectAnswer
// 	}
// 	if req.CorrectOptionKeys != nil {
// 		updates["correct_option_keys"] = req.CorrectOptionKeys
// 	}
// 	if req.Rubric != nil {
// 		updates["rubric"] = convertRubricToJSON(req.Rubric)
// 	}
// 	if req.Explanation != nil {
// 		updates["explanation"] = *req.Explanation
// 	}
// 	if req.Marks != nil {
// 		updates["marks"] = *req.Marks
// 	}
// 	if req.Difficulty != nil {
// 		updates["difficulty"] = *req.Difficulty
// 	}
// 	if req.BloomLevel != nil {
// 		updates["bloom_level"] = *req.BloomLevel
// 	}
// 	if req.TimeLimitSeconds != nil {
// 		updates["time_limit_seconds"] = *req.TimeLimitSeconds
// 	}
// 	if req.Status != nil {
// 		updates["status"] = *req.Status
// 	}
// 	if req.Topic != nil {
// 		updates["topic"] = *req.Topic
// 	}
// 	if req.SubTopic != nil {
// 		updates["sub_topic"] = *req.SubTopic
// 	}
// 	if req.CurriculumType != nil {
// 		updates["curriculum_type"] = *req.CurriculumType
// 	}
// 	if req.SourceType != nil {
// 		updates["source_type"] = *req.SourceType
// 	}
// 	if req.LearningObjective != nil {
// 		updates["learning_objective"] = *req.LearningObjective
// 	}
// 	if req.NegativeMarks != nil {
// 		updates["negative_marks"] = *req.NegativeMarks
// 	}
// 	if req.Order != nil {
// 		updates["order"] = *req.Order
// 	}
// 	if req.IsRequired != nil {
// 		updates["is_required"] = *req.IsRequired
// 	}

// 	if len(updates) == 0 {
// 		return s.toResponseWithSubject(q), nil
// 	}

// 	// Create new version with updates
// 	newID, err := s.qRepo.CreateNewVersion(q, updates)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// Fetch the new version
// 	newQ, err := s.qRepo.FindByID(newID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return s.toResponseWithSubject(newQ), nil
// }

// func (s *QuestionService) DeleteQuestion(id string) error {
// 	qID, err := uuid.Parse(id)
// 	if err != nil {
// 		return errors.New("invalid question ID")
// 	}
// 	return s.qRepo.Delete(qID)
// }

// func (s *QuestionService) ListQuestions(subjectID string, page, limit int) ([]dto.QuestionBankResponse, int64, error) {
// 	subj, err := uuid.Parse(subjectID)
// 	if err != nil {
// 		return nil, 0, errors.New("invalid subject ID")
// 	}
// 	qs, total, err := s.qRepo.ListBySubject(subj, page, limit)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	resp := make([]dto.QuestionBankResponse, 0, len(qs))
// 	for _, q := range qs {
// 		r := s.toResponseWithSubject(&q)
// 		resp = append(resp, *r)
// 	}
// 	return resp, total, nil
// }

// func (s *QuestionService) FilterQuestions(req *dto.FilterQuestionsRequest) ([]dto.QuestionBankResponse, int64, error) {
// 	params := map[string]interface{}{
// 		"subject_id":      req.SubjectID,
// 		"school_id":       req.SchoolID,
// 		"class_level_id":  req.ClassLevelID,
// 		"session_id":      req.SessionID,
// 		"term_id":         req.TermID,
// 		"topic":           req.Topic,
// 		"difficulty":      strings.Join(req.Difficulty, ","),
// 		"bloom_level":     strings.Join(req.BloomLevel, ","),
// 		"question_type":   strings.Join(req.QuestionType, ","),
// 		"status":          req.Status,
// 		"search":          req.Search,
// 	}
// 	qs, total, err := s.qRepo.Filter(params, req.Page, req.Limit)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	resp := make([]dto.QuestionBankResponse, 0, len(qs))
// 	for _, q := range qs {
// 		r := s.toResponseWithSubject(&q)
// 		resp = append(resp, *r)
// 	}
// 	return resp, total, nil
// }

// func (s *QuestionService) BulkDelete(req *dto.BulkDeleteRequest) error {
// 	ids := make([]uuid.UUID, len(req.QuestionIDs))
// 	for i, idStr := range req.QuestionIDs {
// 		id, err := uuid.Parse(idStr)
// 		if err != nil {
// 			return fmt.Errorf("invalid ID: %s", idStr)
// 		}
// 		ids[i] = id
// 	}
// 	return s.qRepo.BulkDelete(ids)
// }

// func (s *QuestionService) BulkUpdateStatus(ids []string, status string) error {
// 	uuids := make([]uuid.UUID, len(ids))
// 	for i, idStr := range ids {
// 		id, err := uuid.Parse(idStr)
// 		if err != nil {
// 			return err
// 		}
// 		uuids[i] = id
// 	}
// 	return s.qRepo.BulkUpdateStatus(uuids, status)
// }

// func (s *QuestionService) CreateTag(req *dto.CreateTagRequest) (*dto.TagResponse, error) {
// 	tag := &models.Tag{
// 		ID:          uuid.New(),
// 		Name:        req.Name,
// 		Slug:        strings.ReplaceAll(strings.ToLower(req.Name), " ", "-"),
// 		Description: req.Description,
// 	}
// 	if err := s.qRepo.CreateTag(tag); err != nil {
// 		return nil, err
// 	}
// 	return &dto.TagResponse{
// 		ID:          tag.ID.String(),
// 		Name:        tag.Name,
// 		Slug:        tag.Slug,
// 		Description: tag.Description,
// 		CreatedAt:   tag.CreatedAt,
// 	}, nil
// }

// func (s *QuestionService) ListTags() ([]dto.TagResponse, error) {
// 	tags, err := s.qRepo.ListTags()
// 	if err != nil {
// 		return nil, err
// 	}
// 	resp := make([]dto.TagResponse, len(tags))
// 	for i, t := range tags {
// 		resp[i] = dto.TagResponse{
// 			ID:          t.ID.String(),
// 			Name:        t.Name,
// 			Slug:        t.Slug,
// 			Description: t.Description,
// 			UsageCount:  t.UsageCount,
// 			CreatedAt:   t.CreatedAt,
// 		}
// 	}
// 	return resp, nil
// }

// func (s *QuestionService) GetStatistics(subjectID string) (map[string]interface{}, error) {
// 	var subj uuid.UUID
// 	if subjectID != "" {
// 		var err error
// 		subj, err = uuid.Parse(subjectID)
// 		if err != nil {
// 			return nil, errors.New("invalid subject ID")
// 		}
// 	}
// 	return s.qRepo.GetStatistics(subj)
// }

// // ============================================
// // BULK CREATE FROM JSON (existing) – now updated
// // ============================================

// func (s *QuestionService) BulkCreateQuestionsFromJSON(req *dto.BulkCreateQuestionRequest) ([]dto.QuestionBankResponse, error) {
// 	if len(req.Questions) == 0 {
// 		return nil, errors.New("no questions provided")
// 	}

// 	var responses []dto.QuestionBankResponse

// 	err := s.db.Transaction(func(tx *gorm.DB) error {
// 		txQRepo := repository.NewQuestionRepository(tx)
// 		for _, qReq := range req.Questions {
// 			questionID := uuid.New()
// 			var optsJSON models.JSONMap
// 			if qReq.OptionsArray != nil {
// 				optsJSON = convertOptionsToJSON(qReq.OptionsArray)
// 			} else if qReq.Options != nil {
// 				var arr []dto.QuestionOption
// 				for k, v := range qReq.Options {
// 					arr = append(arr, dto.QuestionOption{Key: k, Text: v})
// 				}
// 				optsJSON = convertOptionsToJSON(arr)
// 			}
// 			schoolID, _ := uuid.Parse(qReq.SchoolID)
// 			classLevelID, _ := uuid.Parse(qReq.ClassLevelID)

// 			q := &models.QuestionBank{
// 				ID:                questionID,
// 				SchoolID:          schoolID,
// 				ClassLevelID:      classLevelID,
// 				ClassID:           parseOptionalUUID(qReq.ClassID),
// 				SessionID:         parseOptionalUUID(qReq.SessionID),
// 				TermID:            parseOptionalUUID(qReq.TermID),
// 				CurriculumType:    qReq.CurriculumType,
// 				SourceType:        qReq.SourceType,
// 				ExternalID:        qReq.ExternalID,
// 				SubjectID:         uuid.MustParse(qReq.SubjectID),
// 				Topic:             qReq.Topic,
// 				SubTopic:          qReq.SubTopic,
// 				LearningObjective: qReq.LearningObjective,
// 				QuestionText:      qReq.QuestionText,
// 				QuestionType:      models.QuestionType(qReq.QuestionType),
// 				Difficulty:        models.DifficultyLevel(qReq.Difficulty),
// 				BloomLevel:        models.BloomTaxonomy(qReq.BloomLevel),
// 				Options:           optsJSON,
// 				CorrectAnswer:     qReq.CorrectAnswer,
// 				CorrectOptionKeys: qReq.CorrectOptionKeys,
// 				Rubric:            convertRubricToJSON(qReq.Rubric),
// 				Explanation:       qReq.Explanation,
// 				Marks:             qReq.Marks,
// 				NegativeMarks:     qReq.NegativeMarks,
// 				TimeLimitSeconds:  qReq.TimeLimitSeconds,
// 				Order:             qReq.Order,
// 				IsRequired:        qReq.IsRequired,
// 				Status:            models.QuestionStatusDraft,
// 				Version:           1,
// 				CreatedBy:         uuid.Nil,
// 			}
// 			if err := txQRepo.Create(q); err != nil {
// 				return fmt.Errorf("failed to create question: %w", err)
// 			}
// 			if len(qReq.Tags) > 0 {
// 				if err := s.attachTagsByNamesInTx(tx, questionID, qReq.Tags); err != nil {
// 					return err
// 				}
// 			}
// 			responses = append(responses, *s.toResponseLight(q))
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	for i, resp := range responses {
// 		subj, _ := s.subRepo.FindByID(uuid.MustParse(resp.SubjectID))
// 		if subj != nil {
// 			responses[i].SubjectName = subj.Name
// 		}
// 	}
// 	return responses, nil
// }

// // Helper to parse optional UUID
// func parseOptionalUUID(s string) *uuid.UUID {
// 	if s == "" {
// 		return nil
// 	}
// 	u := uuid.MustParse(s)
// 	return &u
// }

// // ============================================
// // BULK UPLOAD FROM FILE (no transaction)
// // ============================================

// func (s *QuestionService) BulkUploadFromFile(file io.Reader, format, subjectIDStr string, hasHeader bool) (*dto.BulkUploadResponse, error) {
// 	subjectID, err := uuid.Parse(subjectIDStr)
// 	if err != nil {
// 		return nil, errors.New("invalid subject_id")
// 	}

// 	var rows []dto.CSVQuestionRow
// 	switch format {
// 	case "csv":
// 		rows, err = s.parseCSV(file, hasHeader)
// 	case "json":
// 		rows, err = s.parseJSON(file)
// 	case "excel":
// 		rows, err = s.parseExcel(file)
// 	default:
// 		return nil, errors.New("unsupported format, use csv, json, or excel")
// 	}
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp := &dto.BulkUploadResponse{
// 		TotalProcessed: len(rows),
// 		Errors:         []string{},
// 	}

// 	for i, row := range rows {
// 		if row.QuestionText == "" || row.CorrectAnswer == "" {
// 			resp.FailedCount++
// 			resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: missing question text or correct answer", i+1))
// 			continue
// 		}
// 		opts := make(map[string]string)
// 		if row.OptionA != "" {
// 			opts["A"] = row.OptionA
// 		}
// 		if row.OptionB != "" {
// 			opts["B"] = row.OptionB
// 		}
// 		if row.OptionC != "" {
// 			opts["C"] = row.OptionC
// 		}
// 		if row.OptionD != "" {
// 			opts["D"] = row.OptionD
// 		}

// 		difficulty := models.DifficultyLevel(row.Difficulty)
// 		if difficulty == "" {
// 			difficulty = models.DifficultyMedium
// 		}
// 		bloom := models.BloomTaxonomy(row.BloomLevel)
// 		if bloom == "" {
// 			bloom = models.BloomRemember
// 		}
// 		qType := models.QuestionType(row.QuestionType)
// 		if qType == "" {
// 			qType = models.QuestionTypeSingle
// 		}
// 		marks := row.Marks
// 		if marks == 0 {
// 			marks = 1
// 		}

// 		var optsArr []dto.QuestionOption
// 		for k, v := range opts {
// 			optsArr = append(optsArr, dto.QuestionOption{Key: k, Text: v})
// 		}
// 		optsJSON := convertOptionsToJSON(optsArr)

// 		q := &models.QuestionBank{
// 			ID:            uuid.New(),
// 			SubjectID:     subjectID,
// 			Topic:         row.Topic,
// 			SubTopic:      row.SubTopic,
// 			QuestionText:  row.QuestionText,
// 			QuestionType:  qType,
// 			Difficulty:    difficulty,
// 			BloomLevel:    bloom,
// 			Options:       optsJSON,
// 			CorrectAnswer: row.CorrectAnswer,
// 			Explanation:   row.Explanation,
// 			Marks:         marks,
// 			Status:        models.QuestionStatusDraft,
// 			Version:       1,
// 			CreatedBy:     uuid.Nil,
// 		}
// 		if err := s.qRepo.Create(q); err != nil {
// 			resp.FailedCount++
// 			resp.Errors = append(resp.Errors, fmt.Sprintf("Row %d: %v", i+1, err))
// 			continue
// 		}
// 		resp.SuccessCount++
// 	}
// 	return resp, nil
// }

// // ============================================
// // Parsers – unchanged
// // ============================================

// func (s *QuestionService) parseCSV(file io.Reader, hasHeader bool) ([]dto.CSVQuestionRow, error) {
// 	reader := csv.NewReader(file)
// 	reader.FieldsPerRecord = -1
// 	reader.TrimLeadingSpace = true
// 	records, err := reader.ReadAll()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(records) == 0 {
// 		return nil, errors.New("empty CSV")
// 	}
// 	start := 0
// 	if hasHeader {
// 		start = 1
// 	}
// 	var rows []dto.CSVQuestionRow
// 	for i := start; i < len(records); i++ {
// 		row := records[i]
// 		if len(row) < 14 {
// 			continue
// 		}
// 		rows = append(rows, dto.CSVQuestionRow{
// 			QuestionText:  row[0],
// 			OptionA:       row[1],
// 			OptionB:       row[2],
// 			OptionC:       row[3],
// 			OptionD:       row[4],
// 			CorrectAnswer: row[5],
// 			Explanation:   row[6],
// 			Marks:         parseInt(row[7]),
// 			Topic:         row[8],
// 			SubTopic:      row[9],
// 			Difficulty:    row[10],
// 			BloomLevel:    row[11],
// 			QuestionType:  row[12],
// 			SubjectID:     row[13],
// 		})
// 	}
// 	if len(rows) == 0 {
// 		return nil, errors.New("no valid data rows found (expected 14 columns)")
// 	}
// 	return rows, nil
// }

// func (s *QuestionService) parseJSON(file io.Reader) ([]dto.CSVQuestionRow, error) {
// 	var importData dto.JSONQuestionImport
// 	if err := json.NewDecoder(file).Decode(&importData); err != nil {
// 		return nil, err
// 	}
// 	var rows []dto.CSVQuestionRow
// 	for _, q := range importData.Questions {
// 		rows = append(rows, dto.CSVQuestionRow{
// 			QuestionText:  q.QuestionText,
// 			OptionA:       q.OptionA,
// 			OptionB:       q.OptionB,
// 			OptionC:       q.OptionC,
// 			OptionD:       q.OptionD,
// 			CorrectAnswer: q.CorrectAnswer,
// 			Explanation:   q.Explanation,
// 			Marks:         q.Marks,
// 			Topic:         q.Topic,
// 			SubTopic:      q.SubTopic,
// 			Difficulty:    q.Difficulty,
// 			BloomLevel:    q.BloomLevel,
// 			QuestionType:  q.QuestionType,
// 			SubjectID:     q.SubjectID,
// 		})
// 	}
// 	return rows, nil
// }

// func (s *QuestionService) parseExcel(file io.Reader) ([]dto.CSVQuestionRow, error) {
// 	f, err := excelize.OpenReader(file)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	rows, err := f.GetRows(f.GetSheetName(0))
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(rows) < 2 {
// 		return nil, errors.New("Excel file must have header + data")
// 	}
// 	var result []dto.CSVQuestionRow
// 	for i := 1; i < len(rows); i++ {
// 		row := rows[i]
// 		if len(row) < 14 {
// 			continue
// 		}
// 		result = append(result, dto.CSVQuestionRow{
// 			QuestionText:  row[0],
// 			OptionA:       row[1],
// 			OptionB:       row[2],
// 			OptionC:       row[3],
// 			OptionD:       row[4],
// 			CorrectAnswer: row[5],
// 			Explanation:   row[6],
// 			Marks:         parseInt(row[7]),
// 			Topic:         row[8],
// 			SubTopic:      row[9],
// 			Difficulty:    row[10],
// 			BloomLevel:    row[11],
// 			QuestionType:  row[12],
// 			SubjectID:     row[13],
// 		})
// 	}
// 	return result, nil
// }

// func parseInt(s string) int {
// 	var i int
// 	fmt.Sscanf(s, "%d", &i)
// 	return i
// }

// // ============================================
// // Helpers – Tag attachment
// // ============================================

// func (s *QuestionService) attachTagsByNames(questionID uuid.UUID, tagNames []string) error {
// 	for _, name := range tagNames {
// 		tag, err := s.qRepo.FindTagByName(name)
// 		if err != nil {
// 			tag = &models.Tag{
// 				ID:   uuid.New(),
// 				Name: name,
// 				Slug: strings.ReplaceAll(strings.ToLower(name), " ", "-"),
// 			}
// 			if err := s.qRepo.CreateTag(tag); err != nil {
// 				return err
// 			}
// 		}
// 		if err := s.qRepo.AttachTags(questionID, []uuid.UUID{tag.ID}); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (s *QuestionService) attachTagsByNamesInTx(tx *gorm.DB, questionID uuid.UUID, tagNames []string) error {
// 	for _, name := range tagNames {
// 		var tag models.Tag
// 		err := tx.Where("name = ?", name).First(&tag).Error
// 		if err != nil {
// 			tag = models.Tag{
// 				ID:   uuid.New(),
// 				Name: name,
// 				Slug: strings.ReplaceAll(strings.ToLower(name), " ", "-"),
// 			}
// 			if err := tx.Create(&tag).Error; err != nil {
// 				return err
// 			}
// 		}
// 		mapping := models.QuestionTagMapping{
// 			ID:         uuid.New(),
// 			QuestionID: questionID,
// 			TagID:      tag.ID,
// 		}
// 		if err := tx.Create(&mapping).Error; err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// // ============================================
// // Response Builders – updated
// // ============================================

// // func (s *QuestionService) toQuestionBankResponse(q *models.QuestionBank) *dto.QuestionBankResponse {
// // 	// Extract options from JSONMap
// // 	var opts []dto.QuestionOption
// // 	if q.Options != nil {
// // 		if optArr, ok := q.Options["options"].([]interface{}); ok {
// // 			for _, item := range optArr {
// // 				if m, ok := item.(map[string]interface{}); ok {
// // 					opts = append(opts, dto.QuestionOption{
// // 						Key:  m["key"].(string),
// // 						Text: m["text"].(string),
// // 					})
// // 				}
// // 			}
// // 		}
// // 	}
// // 	var rubric []dto.RubricCriteria
// // 	if q.Rubric != nil {
// // 		if rubArr, ok := q.Rubric["rubric"].([]interface{}); ok {
// // 			for _, item := range rubArr {
// // 				if m, ok := item.(map[string]interface{}); ok {
// // 					rubric = append(rubric, dto.RubricCriteria{
// // 						Criteria: m["criteria"].(string),
// // 						Marks:    int(m["marks"].(float64)),
// // 					})
// // 				}
// // 			}
// // 		}
// // 	}
// // 	var tags []string
// // 	if q.Tags != nil {
// // 		if tagArr, ok := q.Tags["tags"].([]interface{}); ok {
// // 			for _, t := range tagArr {
// // 				if s, ok := t.(string); ok {
// // 					tags = append(tags, s)
// // 				}
// // 			}
// // 		}
// // 	}

// // 	// Convert UUIDs to strings with helper
// // 	return &dto.QuestionBankResponse{
// // 		ID:                q.ID.String(),
// // 		SubjectID:         q.SubjectID.String(),
// // 		SubjectName:       "",
// // 		Topic:             q.Topic,
// // 		SubTopic:          q.SubTopic,
// // 		QuestionText:      q.QuestionText,
// // 		QuestionType:      string(q.QuestionType),
// // 		Difficulty:        string(q.Difficulty),
// // 		BloomLevel:        string(q.BloomLevel),
// // 		Options:           opts,
// // 		CorrectAnswer:     q.CorrectAnswer,
// // 		Explanation:       q.Explanation,
// // 		Marks:             q.Marks,
// // 		TimeLimitSeconds:  q.TimeLimitSeconds,
// // 		Tags:              tags,
// // 		Status:            string(q.Status),
// // 		Version:           q.Version,
// // 		UsageCount:        q.UsageCount,
// // 		SuccessRate:       q.SuccessRate,
// // 		Attachments:       nil,
// // 		CreatedAt:         q.CreatedAt,
// // 		UpdatedAt:         q.UpdatedAt,
// // 		CreatedBy:         q.CreatedBy.String(),
// // 		CreatedByName:     "",
// // 		SchoolID:          q.SchoolID.String(),
// // 		ClassLevelID:      q.ClassLevelID.String(),
// // 		ClassID:           nilToPtr(q.ClassID),
// // 		SessionID:         nilToPtr(q.SessionID),
// // 		TermID:            nilToPtr(q.TermID),
// // 		CurriculumType:    q.CurriculumType,
// // 		SourceType:        q.SourceType,
// // 		ExternalID:        q.ExternalID,
// // 		LearningObjective: q.LearningObjective,
// // 		CorrectOptionKeys: q.CorrectOptionKeys,
// // 		Rubric:            rubric,
// // 		NegativeMarks:     q.NegativeMarks,
// // 		Order:             q.Order,
// // 		IsRequired:        q.IsRequired,
// // 	}
// // }


// func (s *QuestionService) toQuestionBankResponse(q *models.QuestionBank) *dto.QuestionBankResponse {
//     // Extract options from JSONMap - supports both old and new format
//     var opts []dto.QuestionOption
//     if q.Options != nil {
//         // Check for new format (_json key)
//         if jsonStr, ok := q.Options["_json"].(string); ok && jsonStr != "" {
//             var arr []map[string]string
//             if err := json.Unmarshal([]byte(jsonStr), &arr); err == nil {
//                 for _, item := range arr {
//                     opts = append(opts, dto.QuestionOption{
//                         Key:  item["key"],
//                         Text: item["text"],
//                     })
//                 }
//             }
//         } else if optArr, ok := q.Options["options"].([]interface{}); ok {
//             // Fallback for old format
//             for _, item := range optArr {
//                 if m, ok := item.(map[string]interface{}); ok {
//                     opts = append(opts, dto.QuestionOption{
//                         Key:  m["key"].(string),
//                         Text: m["text"].(string),
//                     })
//                 }
//             }
//         }
//     }

//     var rubric []dto.RubricCriteria
//     if q.Rubric != nil {
//         if jsonStr, ok := q.Rubric["_json"].(string); ok && jsonStr != "" {
//             var arr []map[string]interface{}
//             if err := json.Unmarshal([]byte(jsonStr), &arr); err == nil {
//                 for _, item := range arr {
//                     rubric = append(rubric, dto.RubricCriteria{
//                         Criteria: item["criteria"].(string),
//                         Marks:    int(item["marks"].(float64)),
//                     })
//                 }
//             }
//         } else if rubArr, ok := q.Rubric["rubric"].([]interface{}); ok {
//             for _, item := range rubArr {
//                 if m, ok := item.(map[string]interface{}); ok {
//                     rubric = append(rubric, dto.RubricCriteria{
//                         Criteria: m["criteria"].(string),
//                         Marks:    int(m["marks"].(float64)),
//                     })
//                 }
//             }
//         }
//     }

//     var tags []string
//     if q.Tags != nil {
//         if jsonStr, ok := q.Tags["_json"].(string); ok && jsonStr != "" {
//             json.Unmarshal([]byte(jsonStr), &tags)
//         } else if tagArr, ok := q.Tags["tags"].([]interface{}); ok {
//             for _, t := range tagArr {
//                 if s, ok := t.(string); ok {
//                     tags = append(tags, s)
//                 }
//             }
//         }
//     }

//     // Convert UUIDs to strings with helper
//     return &dto.QuestionBankResponse{
//         ID:                q.ID.String(),
//         SubjectID:         q.SubjectID.String(),
//         SubjectName:       "",
//         Topic:             q.Topic,
//         SubTopic:          q.SubTopic,
//         QuestionText:      q.QuestionText,
//         QuestionType:      string(q.QuestionType),
//         Difficulty:        string(q.Difficulty),
//         BloomLevel:        string(q.BloomLevel),
//         Options:           opts,
//         CorrectAnswer:     q.CorrectAnswer,
//         Explanation:       q.Explanation,
//         Marks:             q.Marks,
//         TimeLimitSeconds:  q.TimeLimitSeconds,
//         Tags:              tags,
//         Status:            string(q.Status),
//         Version:           q.Version,
//         UsageCount:        q.UsageCount,
//         SuccessRate:       q.SuccessRate,
//         Attachments:       nil,
//         CreatedAt:         q.CreatedAt,
//         UpdatedAt:         q.UpdatedAt,
//         CreatedBy:         q.CreatedBy.String(),
//         CreatedByName:     "",
//         SchoolID:          q.SchoolID.String(),
//         ClassLevelID:      q.ClassLevelID.String(),
//         ClassID:           nilToPtr(q.ClassID),
//         SessionID:         nilToPtr(q.SessionID),
//         TermID:            nilToPtr(q.TermID),
//         CurriculumType:    q.CurriculumType,
//         SourceType:        q.SourceType,
//         ExternalID:        q.ExternalID,
//         LearningObjective: q.LearningObjective,
//         CorrectOptionKeys: q.CorrectOptionKeys,
//         Rubric:            rubric,
//         NegativeMarks:     q.NegativeMarks,
//         Order:             q.Order,
//         IsRequired:        q.IsRequired,
//     }
// }



// func nilToPtr(u *uuid.UUID) *string {
// 	if u == nil {
// 		return nil
// 	}
// 	s := u.String()
// 	return &s
// }

// // toResponseWithSubject – adds subject name
// func (s *QuestionService) toResponseWithSubject(q *models.QuestionBank) *dto.QuestionBankResponse {
// 	resp := s.toQuestionBankResponse(q)
// 	subject, err := s.subRepo.FindByID(q.SubjectID)
// 	if err == nil && subject != nil {
// 		resp.SubjectName = subject.Name
// 	}
// 	return resp
// }

// // toResponseLight – for bulk operations (without subject name)
// func (s *QuestionService) toResponseLight(q *models.QuestionBank) *dto.QuestionBankResponse {
// 	return s.toQuestionBankResponse(q)
// }

// // ============================================
// // Converters for JSON storage
// // ============================================

// // func convertOptionsToJSON(opts []dto.QuestionOption) models.JSONMap {
// // 	if opts == nil {
// // 		return nil
// // 	}
// // 	arr := make([]map[string]string, len(opts))
// // 	for i, o := range opts {
// // 		arr[i] = map[string]string{"key": o.Key, "text": o.Text}
// // 	}
// // 	return models.JSONMap{"options": arr}
// // }

// // func convertOptionsToJSON(opts []dto.QuestionOption) models.Options {
// //     if opts == nil {
// //         return models.Options{}
// //     }
// //     result := make(models.Options, len(opts))
// //     for i, o := range opts {
// //         result[i] = models.QuestionOption{
// //             Key:  o.Key,
// //             Text: o.Text,
// //         }
// //     }
// //     return result
// // }

// // func convertOptionsToJSON(opts []dto.QuestionOption) models.JSONMap {
// //     if opts == nil {
// //         return models.JSONMap{}
// //     }
// //     arr := make([]map[string]string, len(opts))
// //     for i, o := range opts {
// //         arr[i] = map[string]string{"key": o.Key, "text": o.Text}
// //     }
// //     // Marshal to JSON string and store as JSONMap
// //     jsonBytes, _ := json.Marshal(arr)
// //     return models.JSONMap{"_raw": string(jsonBytes)}
// // }

// // func convertRubricToJSON(rubric []dto.RubricCriteria) models.JSONMap {
// // 	if rubric == nil {
// // 		return nil
// // 	}
// // 	arr := make([]map[string]interface{}, len(rubric))
// // 	for i, r := range rubric {
// // 		arr[i] = map[string]interface{}{"criteria": r.Criteria, "marks": r.Marks}
// // 	}
// // 	return models.JSONMap{"rubric": arr}
// // }

// // func convertTagsToJSON(tags []string) models.JSONMap {
// // 	if tags == nil {
// // 		return nil
// // 	}
// // 	return models.JSONMap{"tags": tags}
// // }

// // ============================================
// // Converters for JSON storage - FIXED
// // ============================================

// func convertOptionsToJSON(opts []dto.QuestionOption) models.JSONMap {
//     if opts == nil || len(opts) == 0 {
//         // Return empty array as JSON
//         return models.JSONMap{}
//     }
//     // Convert to array of maps directly (not wrapped in "options" key)
//     arr := make([]map[string]string, len(opts))
//     for i, o := range opts {
//         arr[i] = map[string]string{"key": o.Key, "text": o.Text}
//     }
//     // Store as JSON array directly
//     return models.JSONMap{
//         "": arr, // This will be marshaled as the array
//     }
// }

// func convertRubricToJSON(rubric []dto.RubricCriteria) models.JSONMap {
//     if rubric == nil || len(rubric) == 0 {
//         return models.JSONMap{}
//     }
//     arr := make([]map[string]interface{}, len(rubric))
//     for i, r := range rubric {
//         arr[i] = map[string]interface{}{"criteria": r.Criteria, "marks": r.Marks}
//     }
//     return models.JSONMap{
//         "": arr,
//     }
// }

// func convertTagsToJSON(tags []string) models.JSONMap {
//     if tags == nil || len(tags) == 0 {
//         return models.JSONMap{}
//     }
//     return models.JSONMap{
//         "": tags,
//     }
// }


// // ============================================
// // NEW: Bulk Import (Exact JSON)
// // ============================================

// func (s *QuestionService) BulkImportQuestions(req *dto.BulkQuestionImportRequest) ([]dto.QuestionBankResponse, error) {
// 	if len(req.Questions) == 0 {
// 		return nil, errors.New("no questions provided")
// 	}

// 	var responses []dto.QuestionBankResponse

// 	// Parse top-level UUIDs
// 	schoolID, err := uuid.Parse(req.SchoolID)
// 	if err != nil {
// 		return nil, errors.New("invalid school_id")
// 	}
// 	classLevelID, err := uuid.Parse(req.ClassLevelID)
// 	if err != nil {
// 		return nil, errors.New("invalid class_level_id")
// 	}
// 	var classID, sessionID, termID *uuid.UUID
// 	if req.ClassID != "" {
// 		u, err := uuid.Parse(req.ClassID)
// 		if err != nil {
// 			return nil, errors.New("invalid class_id")
// 		}
// 		classID = &u
// 	}
// 	if req.SessionID != "" {
// 		u, err := uuid.Parse(req.SessionID)
// 		if err != nil {
// 			return nil, errors.New("invalid session_id")
// 		}
// 		sessionID = &u
// 	}
// 	if req.TermID != "" {
// 		u, err := uuid.Parse(req.TermID)
// 		if err != nil {
// 			return nil, errors.New("invalid term_id")
// 		}
// 		termID = &u
// 	}
// 	createdBy, err := uuid.Parse(req.CreatedBy)
// 	if err != nil {
// 		return nil, errors.New("invalid created_by")
// 	}

// 	err = s.db.Transaction(func(tx *gorm.DB) error {
// 		txRepo := repository.NewQuestionRepository(tx)

// 		for idx, item := range req.Questions {
// 			// Validate question-type rules
// 			if err := validateQuestionItem(item); err != nil {
// 				return fmt.Errorf("question %d: %w", idx+1, err)
// 			}

// 			// Parse subject ID
// 			subjectID, err := uuid.Parse(item.SubjectID)
// 			if err != nil {
// 				return fmt.Errorf("question %d: invalid subject_id", idx+1)
// 			}

// 			// 1. Check idempotency via external_id
// 			var existing *models.QuestionBank
// 			if item.ExternalID != "" {
// 				var sessID uuid.UUID
// 				if sessionID != nil {
// 					sessID = *sessionID
// 				}
// 				existing, _ = txRepo.FindByExternalID(schoolID, sessID, item.ExternalID)
// 			}

// 			// Build common data
// 			optsJSON := convertOptionsToJSON(item.Options)
// 			rubricJSON := convertRubricToJSON(item.Rubric)
// 			tagsJSON := convertTagsToJSON(item.Tags)

// 			if existing != nil {
// 				// Update: create new version
// 				updates := map[string]interface{}{
// 					"topic":               item.Topic,
// 					"sub_topic":           item.SubTopic,
// 					"learning_objective":  item.LearningObjective,
// 					"question_text":       item.QuestionText,
// 					"question_type":       item.QuestionType,
// 					"difficulty":          item.Difficulty,
// 					"bloom_level":         item.BloomLevel,
// 					"options":             optsJSON,
// 					"correct_option_keys": item.CorrectOptionKeys,
// 					"rubric":              rubricJSON,
// 					"explanation":         item.Explanation,
// 					"marks":               item.Marks,
// 					"negative_marks":      item.NegativeMarks,
// 					"time_limit_seconds":  item.TimeLimitSeconds,
// 					"order":               item.Order,
// 					"is_required":         item.IsRequired,
// 					"updated_by":          createdBy,
// 					"tags":                tagsJSON,
// 					"status":              req.Status,
// 					"curriculum_type":     req.CurriculumType,
// 					"source_type":         req.SourceType,
// 				}
// 				newID, err := txRepo.CreateNewVersion(existing, updates)
// 				if err != nil {
// 					return fmt.Errorf("failed to update version for question %d: %w", idx+1, err)
// 				}
// 				existing, err = txRepo.FindByID(newID)
// 				if err != nil {
// 					return fmt.Errorf("failed to fetch updated question %d: %w", idx+1, err)
// 				}
// 				if len(item.Tags) > 0 {
// 					if err := s.attachTagsByNamesInTx(tx, existing.ID, item.Tags); err != nil {
// 						return fmt.Errorf("failed to attach tags for question %d: %w", idx+1, err)
// 					}
// 				}
// 			} else {
// 				// Create new
// 				q := &models.QuestionBank{
// 					ID:                uuid.New(),
// 					SchoolID:          schoolID,
// 					ClassLevelID:      classLevelID,
// 					ClassID:           classID,
// 					SessionID:         sessionID,
// 					TermID:            termID,
// 					CurriculumType:    req.CurriculumType,
// 					SourceType:        req.SourceType,
// 					ExternalID:        item.ExternalID,
// 					SubjectID:         subjectID,
// 					Topic:             item.Topic,
// 					SubTopic:          item.SubTopic,
// 					LearningObjective: item.LearningObjective,
// 					QuestionText:      item.QuestionText,
// 					QuestionType:      models.QuestionType(item.QuestionType),
// 					Difficulty:        models.DifficultyLevel(item.Difficulty),
// 					BloomLevel:        models.BloomTaxonomy(item.BloomLevel),
// 					Options:           optsJSON,
// 					CorrectOptionKeys: item.CorrectOptionKeys,
// 					Rubric:            rubricJSON,
// 					Explanation:       item.Explanation,
// 					Marks:             item.Marks,
// 					NegativeMarks:     item.NegativeMarks,
// 					TimeLimitSeconds:  &item.TimeLimitSeconds,
// 					Order:             item.Order,
// 					IsRequired:        item.IsRequired,
// 					Tags:              tagsJSON,
// 					Status:            models.QuestionStatus(req.Status),
// 					Version:           1,
// 					CreatedBy:         createdBy,
// 					UpdatedBy:         createdBy,
// 				}
// 				if err := txRepo.Create(q); err != nil {
// 					return fmt.Errorf("failed to create question %d: %w", idx+1, err)
// 				}
// 				if len(item.Tags) > 0 {
// 					if err := s.attachTagsByNamesInTx(tx, q.ID, item.Tags); err != nil {
// 						return fmt.Errorf("failed to attach tags for question %d: %w", idx+1, err)
// 					}
// 				}
// 				existing = q
// 			}

// 			// Build response
// 			resp := s.toQuestionBankResponse(existing)
// 			responses = append(responses, *resp)
// 		}
// 		return nil
// 	})

// 	if err != nil {
// 		return nil, err
// 	}

// 	// Fetch subject names for responses
// 	for i := range responses {
// 		subj, _ := s.subRepo.FindByID(uuid.MustParse(responses[i].SubjectID))
// 		if subj != nil {
// 			responses[i].SubjectName = subj.Name
// 		}
// 	}
// 	return responses, nil
// }

// func validateQuestionItem(item dto.QuestionImportItem) error {
// 	switch item.QuestionType {
// 	case "single_choice", "multiple_choice", "true_false":
// 		if len(item.Options) == 0 {
// 			return errors.New("MCQ/true_false must have options")
// 		}
// 		if len(item.CorrectOptionKeys) == 0 {
// 			return errors.New("MCQ/true_false must have correct option keys")
// 		}
// 		if item.Rubric != nil && len(item.Rubric) > 0 {
// 			return errors.New("MCQ/true_false cannot have rubric")
// 		}
// 	case "essay":
// 		if item.Options != nil && len(item.Options) > 0 {
// 			return errors.New("essay cannot have options")
// 		}
// 		if item.CorrectOptionKeys != nil && len(item.CorrectOptionKeys) > 0 {
// 			return errors.New("essay cannot have correct option keys")
// 		}
// 		if item.Rubric == nil || len(item.Rubric) == 0 {
// 			return errors.New("essay must have rubric")
// 		}
// 	case "fill_blank":
// 		// optional
// 	}
// 	return nil
// }

// // ============================================
// // AI Methods – Enqueue Jobs
// // ============================================

// func (s *QuestionService) GenerateQuestionsWithAI(req *dto.AIGenerateQuestionsRequest) (*dto.AIQuestionGenerationResponse, error) {
// 	// Create job record
// 	job := &models.AIQuestionGenerationJob{
// 		ID:                uuid.New(),
// 		UserID:            uuid.Nil,
// 		SubjectID:         uuid.MustParse(req.SubjectID),
// 		Topic:             req.Topic,
// 		NumberOfQuestions: req.NumberOfQuestions,
// 		Difficulty:        models.DifficultyLevel(req.Difficulty),
// 		BloomLevel:        models.BloomTaxonomy(req.BloomLevel),
// 		SourceText:        req.SourceText,
// 		Status:            "queued",
// 	}

// 	if err := s.db.Create(job).Error; err != nil {
// 		return nil, err
// 	}

// 	// Build payload for worker
// 	payload := map[string]interface{}{
// 		"job_id":  job.ID,
// 		"type":    "generate",
// 		"request": req,
// 	}
// 	data, err := json.Marshal(payload)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()
// 	if err := s.queue.Push(ctx, "ai_jobs", string(data)); err != nil {
// 		return nil, err
// 	}

// 	return &dto.AIQuestionGenerationResponse{
// 		JobID:   job.ID.String(),
// 		Status:  "queued",
// 		Message: "Job enqueued successfully",
// 	}, nil
// }

// func (s *QuestionService) ExtractQuestionsFromText(req *dto.ExtractTextQuestionsRequest) (*dto.AIQuestionGenerationResponse, error) {
// 	job := &models.AIQuestionGenerationJob{
// 		ID:         uuid.New(),
// 		UserID:     uuid.Nil,
// 		SubjectID:  uuid.MustParse(req.SubjectID),
// 		SourceText: req.Text,
// 		Status:     "queued",
// 	}
// 	if err := s.db.Create(job).Error; err != nil {
// 		return nil, err
// 	}

// 	payload := map[string]interface{}{
// 		"job_id":  job.ID,
// 		"type":    "extract",
// 		"text":    req.Text,
// 		"school":  req.SchoolID,
// 		"class":   req.ClassLevelID,
// 		"subject": req.SubjectID,
// 	}
// 	data, err := json.Marshal(payload)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if err := s.queue.Push(context.Background(), "ai_jobs", string(data)); err != nil {
// 		return nil, err
// 	}

// 	return &dto.AIQuestionGenerationResponse{
// 		JobID:   job.ID.String(),
// 		Status:  "queued",
// 		Message: "Extraction job enqueued",
// 	}, nil
// }

// func (s *QuestionService) GetJobStatus(jobID string) (*dto.AIJobStatusResponse, error) {
// 	id, err := uuid.Parse(jobID)
// 	if err != nil {
// 		return nil, errors.New("invalid job ID")
// 	}
// 	var job models.AIQuestionGenerationJob
// 	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
// 		return nil, errors.New("job not found")
// 	}
// 	return &dto.AIJobStatusResponse{
// 		JobID:        job.ID.String(),
// 		Status:       job.Status,
// 		ErrorMessage: job.ErrorMessage,
// 		CreatedAt:    job.CreatedAt,
// 		CompletedAt:  job.CompletedAt,
// 	}, nil
// }

