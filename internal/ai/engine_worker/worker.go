package engine_worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"cbt-api/internal/ai/engine"
	"cbt-api/internal/ai/queue"
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"
	// "cbt-api/internal/repository"
	"cbt-api/internal/cbt/repository"


	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIWorker struct {
	queue   queue.Queue
	engine  *engine.Engine
	db      *gorm.DB
	qRepo   *repository.QuestionRepository
	subRepo *repository.SubjectRepository
}

func NewAIWorker(q queue.Queue, e *engine.Engine, db *gorm.DB) *AIWorker {
	qRepo := repository.NewQuestionRepository(db)
	subRepo := repository.NewSubjectRepository(db)
	return &AIWorker{
		queue:   q,
		engine:  e,
		db:      db,
		qRepo:   qRepo,
		subRepo: subRepo,
	}
}

// Start begins polling the queue and processing jobs.
func (w *AIWorker) Start() {
	log.Println("AI Worker started with Exam Engine...")
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		jobStr, err := w.queue.Pop(ctx, "ai_jobs")
		cancel()
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(jobStr), &payload); err != nil {
			log.Printf("Invalid job: %v", err)
			continue
		}
		jobType, _ := payload["type"].(string)
		switch jobType {
		case "generate":
			w.processGenerateJob(payload)
		case "extract":
			w.processExtractJob(payload)
		default:
			log.Printf("Unknown job type: %s", jobType)
		}
	}
}

func (w *AIWorker) processGenerateJob(payload map[string]interface{}) {
	jobIDStr, _ := payload["job_id"].(string)
	jobID, _ := uuid.Parse(jobIDStr)

	var req dto.AIGenerateQuestionsRequest
	b, _ := json.Marshal(payload["request"])
	json.Unmarshal(b, &req)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(w.engine.Config().TimeoutSeconds)*time.Second)
	defer cancel()

	// Build exam context
	examCtx := engine.ExamContext{
		SchoolID:          req.SchoolID,
		ClassLevelID:      req.ClassLevelID,
		SubjectID:         req.SubjectID,
		Topic:             req.Topic,
		NumberOfQuestions: req.NumberOfQuestions,
		Difficulty:        req.Difficulty,
		BloomLevel:        req.BloomLevel,
		CurriculumType:    req.CurriculumType,
		SourceText:        req.SourceText,
		Keywords:          req.Keywords,
	}

	questions, results, err := w.engine.GenerateExam(ctx, examCtx)
	if err != nil {
		w.updateJobStatus(jobID, "failed", err.Error())
		return
	}

	// Save valid questions
	saved := 0
	for i, q := range questions {
		if results[i].Status == "approved" || results[i].Status == "needs_refinement" {
			question := models.QuestionBank{
				ID:                uuid.New(),
				SchoolID:          uuid.MustParse(req.SchoolID),
				ClassLevelID:      uuid.MustParse(req.ClassLevelID),
				SubjectID:         uuid.MustParse(req.SubjectID),
				CurriculumType:    req.CurriculumType,
				SourceType:        "ai_generated",
				Topic:             req.Topic,
				QuestionText:      q.QuestionText,
				QuestionType:      models.QuestionType(q.Type),
				Difficulty:        models.DifficultyLevel(q.Difficulty),
				BloomLevel:        models.BloomTaxonomy(q.BloomLevel),
				Options:           convertOptionsToMap(q.Options),
				CorrectOptionKeys: q.CorrectOptionKeys,
				Explanation:       q.Explanation,
				Marks:             q.Marks,
				Status:            models.QuestionStatusDraft,
				Version:           1,
			}
			if err := w.qRepo.Create(&question); err != nil {
				log.Printf("Failed to save question: %v", err)
				continue
			}
			saved++
		}
	}

	msg := "Saved " + string(rune(saved)) + " questions"
	w.updateJobStatus(jobID, "completed", msg)
}

func (w *AIWorker) processExtractJob(payload map[string]interface{}) {
	jobIDStr, _ := payload["job_id"].(string)
	jobID, _ := uuid.Parse(jobIDStr)

	text, _ := payload["text"].(string)
	schoolID, _ := payload["school"].(string)
	classLevelID, _ := payload["class"].(string)
	subjectID, _ := payload["subject"].(string)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(w.engine.Config().TimeoutSeconds)*time.Second)
	defer cancel()

	examCtx := engine.ExamContext{
		SchoolID:          schoolID,
		ClassLevelID:      classLevelID,
		SubjectID:         subjectID,
		Topic:             "Extracted from text",
		NumberOfQuestions: 5,
		Difficulty:        "medium",
		BloomLevel:        "apply",
		SourceText:        text,
	}

	questions, results, err := w.engine.GenerateExam(ctx, examCtx)
	if err != nil {
		w.updateJobStatus(jobID, "failed", err.Error())
		return
	}

	saved := 0
	for i, q := range questions {
		if results[i].Status == "approved" || results[i].Status == "needs_refinement" {
			question := models.QuestionBank{
				ID:                uuid.New(),
				SchoolID:          uuid.MustParse(schoolID),
				ClassLevelID:      uuid.MustParse(classLevelID),
				SubjectID:         uuid.MustParse(subjectID),
				SourceType:        "extracted",
				QuestionText:      q.QuestionText,
				QuestionType:      models.QuestionType(q.Type),
				Difficulty:        models.DifficultyLevel(q.Difficulty),
				BloomLevel:        models.BloomTaxonomy(q.BloomLevel),
				Options:           convertOptionsToMap(q.Options),
				CorrectOptionKeys: q.CorrectOptionKeys,
				Explanation:       q.Explanation,
				Marks:             q.Marks,
				Status:            models.QuestionStatusDraft,
				Version:           1,
			}
			if err := w.qRepo.Create(&question); err != nil {
				log.Printf("Failed to save extracted question: %v", err)
				continue
			}
			saved++
		}
	}

	msg := "Extracted and saved " + string(rune(saved)) + " questions"
	w.updateJobStatus(jobID, "completed", msg)
}

func (w *AIWorker) updateJobStatus(jobID uuid.UUID, status, message string) {
	w.db.Model(&models.AIQuestionGenerationJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":        status,
			"error_message": message,
			"completed_at":  func() *time.Time { t := time.Now(); return &t }(),
		})
}

func convertOptionsToMap(opts []engine.Option) models.JSONMap {
	if len(opts) == 0 {
		return nil
	}
	arr := make([]map[string]string, len(opts))
	for i, o := range opts {
		arr[i] = map[string]string{"key": o.Key, "text": o.Text}
	}
	return models.JSONMap{"options": arr}
}