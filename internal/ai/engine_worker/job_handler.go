package engine_worker

import (
	"log"
)

// JobHandler processes different job types. This can be used to separate logic.
type JobHandler struct {
	worker *AIWorker
}

func NewJobHandler(worker *AIWorker) *JobHandler {
	return &JobHandler{worker: worker}
}

// HandleProcess handles a job payload.
func (h *JobHandler) HandleProcess(payload map[string]interface{}) {
	jobType, _ := payload["type"].(string)
	switch jobType {
	case "generate":
		h.handleGenerate(payload)
	case "extract":
		h.handleExtract(payload)
	default:
		log.Printf("Unknown job type in handler: %s", jobType)
	}
}

func (h *JobHandler) handleGenerate(payload map[string]interface{}) {
	h.worker.processGenerateJob(payload)
}

func (h *JobHandler) handleExtract(payload map[string]interface{}) {
	h.worker.processExtractJob(payload)
}


// package engine_worker

// import (
// 	"context"
// 	// "encoding/json"
// 	"log"

// 	"cbt-api/internal/ai/engine"
// 	"cbt-api/internal/cbt/dto"
// 	"cbt-api/internal/models"

// 	"github.com/google/uuid"
// )

// // JobHandler processes different job types. This can be used to separate logic.
// type JobHandler struct {
// 	worker *AIWorker
// }

// func NewJobHandler(worker *AIWorker) *JobHandler {
// 	return &JobHandler{worker: worker}
// }

// // HandleProcess handles a job payload.
// func (h *JobHandler) HandleProcess(payload map[string]interface{}) {
// 	jobType, _ := payload["type"].(string)
// 	switch jobType {
// 	case "generate":
// 		h.handleGenerate(payload)
// 	case "extract":
// 		h.handleExtract(payload)
// 	default:
// 		log.Printf("Unknown job type in handler: %s", jobType)
// 	}
// }

// func (h *JobHandler) handleGenerate(payload map[string]interface{}) {
// 	h.worker.processGenerateJob(payload)
// }

// func (h *JobHandler) handleExtract(payload map[string]interface{}) {
// 	h.worker.processExtractJob(payload)
// }




// // package worker

// // // This file can be used to handle different job types.
// // // The logic is already inside worker.go, but we can extract it for clarity.
// // // For now, it's empty, but we keep it as a placeholder.