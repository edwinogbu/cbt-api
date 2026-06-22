package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"cbt-api/internal/ai/providers"
	// ✅ REMOVED: "cbt-api/internal/ai/validation"
)

// Engine is the main orchestrator for AI-powered exam generation.
type Engine struct {
	router    *providers.Router
	generator *Generator
	validator *Validator //Changed to local Validator
	refiner   *Refiner
	critic    *Critic
	config    EngineConfig
	mu        sync.RWMutex
}

// NewEngine creates a new exam intelligence engine.
func NewEngine(router *providers.Router, config EngineConfig) *Engine {
	return &Engine{
		router:    router,
		generator: NewGenerator(router),
		validator: NewValidator(), // Changed to local
		refiner:   NewRefiner(router),
		critic:    NewCritic(router),
		config:    config,
	}
}

// GenerateExam is the main entry point for question generation.
func (e *Engine) GenerateExam(ctx context.Context, req ExamContext) ([]Question, []ValidationResult, error) {
	// Validate context
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}
	if req.NumberOfQuestions < 1 {
		req.NumberOfQuestions = 5
	}
	if e.config.MaxRefinementCycles == 0 {
		e.config.MaxRefinementCycles = 2
	}
	if e.config.QualityThreshold == 0 {
		e.config.QualityThreshold = 70
	}

	// Apply timeout if configured
	if e.config.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.config.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	// 1. Generate raw questions
	questions, err := e.generator.Generate(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if len(questions) == 0 {
		return nil, nil, nil
	}

	// 2. Validate and score - using local validator
	results := e.validator.ValidateBatch(questions)

	// 3. Refinement loop for weak questions
	for cycle := 0; cycle < e.config.MaxRefinementCycles; cycle++ {
		improved := false
		for i, r := range results {
			if r.Status == "needs_refinement" {
				newQ, err := e.refiner.Improve(ctx, questions[i], r.Errors)
				if err == nil {
					questions[i] = newQ
					newResults := e.validator.ValidateBatch([]Question{newQ})
					results[i] = newResults[0]
					improved = true
				}
			}
		}
		if !improved {
			break
		}
	}

	// 4. Safety checks (optional) - using local functions from guardrails.go
	if e.config.EnableSafety {
		for i := range questions {
			if !CheckCurriculum(questions[i], req.SubjectID, req.Topic) {
				results[i].Status = "rejected"
				results[i].Errors = append(results[i].Errors, "topic mismatch")
			}
			if HasProfanity(questions[i].QuestionText) {
				results[i].Status = "rejected"
				results[i].Errors = append(results[i].Errors, "profanity detected")
			}
			// Check for duplicate questions
			for j, existing := range req.ExistingQuestions {
				if j != i && IsDuplicate(questions[i], existing) {
					results[i].Status = "rejected"
					results[i].Errors = append(results[i].Errors, "duplicate question")
					break
				}
			}
		}
	}

	// 5. Filter out rejected questions
	var finalQuestions []Question
	var finalResults []ValidationResult
	for i, r := range results {
		if r.Status != "rejected" {
			finalQuestions = append(finalQuestions, questions[i])
			finalResults = append(finalResults, r)
		}
	}

	log.Printf("Engine generated %d questions, accepted %d", len(questions), len(finalQuestions))
	return finalQuestions, finalResults, nil
}

// Config returns the engine configuration.
func (e *Engine) Config() EngineConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// UpdateConfig updates the engine configuration (thread-safe).
func (e *Engine) UpdateConfig(config EngineConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = config
}

// GetRefiner returns the refiner instance.
func (e *Engine) GetRefiner() *Refiner {
	return e.refiner
}

// GetCritic returns the critic instance.
func (e *Engine) GetCritic() *Critic {
	return e.critic
}


// package engine

// import (
// 	"context"
// 	"log"
// 	"sync"
// 	"time"

// 	"cbt-api/internal/ai/providers"
// 	"cbt-api/internal/ai/validation"
// 	// ✅ REMOVED: "cbt-api/internal/ai/safety"
// )

// // Engine is the main orchestrator for AI-powered exam generation.
// type Engine struct {
// 	router    *providers.Router
// 	generator *Generator
// 	validator *validation.Validator
// 	refiner   *Refiner
// 	critic    *Critic
// 	config    EngineConfig
// 	mu        sync.RWMutex
// }

// // NewEngine creates a new exam intelligence engine.
// func NewEngine(router *providers.Router, config EngineConfig) *Engine {
// 	return &Engine{
// 		router:    router,
// 		generator: NewGenerator(router),
// 		validator: validation.NewValidator(),
// 		refiner:   NewRefiner(router),
// 		critic:    NewCritic(router),
// 		config:    config,
// 	}
// }

// // GenerateExam is the main entry point for question generation.
// func (e *Engine) GenerateExam(ctx context.Context, req ExamContext) ([]Question, []ValidationResult, error) {
// 	// Validate context
// 	if err := req.Validate(); err != nil {
// 		return nil, nil, err
// 	}
// 	if req.NumberOfQuestions < 1 {
// 		req.NumberOfQuestions = 5
// 	}
// 	if e.config.MaxRefinementCycles == 0 {
// 		e.config.MaxRefinementCycles = 2
// 	}
// 	if e.config.QualityThreshold == 0 {
// 		e.config.QualityThreshold = 70
// 	}

// 	// Apply timeout if configured
// 	if e.config.TimeoutSeconds > 0 {
// 		var cancel context.CancelFunc
// 		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.config.TimeoutSeconds)*time.Second)
// 		defer cancel()
// 	}

// 	// 1. Generate raw questions
// 	questions, err := e.generator.Generate(ctx, req)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	if len(questions) == 0 {
// 		return nil, nil, nil
// 	}

// 	// 2. Validate and score
// 	results := e.validator.ValidateBatch(questions)

// 	// 3. Refinement loop for weak questions
// 	for cycle := 0; cycle < e.config.MaxRefinementCycles; cycle++ {
// 		improved := false
// 		for i, r := range results {
// 			if r.Status == "needs_refinement" {
// 				newQ, err := e.refiner.Improve(ctx, questions[i], r.Errors)
// 				if err == nil {
// 					questions[i] = newQ
// 					newResults := e.validator.ValidateBatch([]Question{newQ})
// 					results[i] = newResults[0]
// 					improved = true
// 				}
// 			}
// 		}
// 		if !improved {
// 			break
// 		}
// 	}

// 	// 4. Safety checks (optional) - using local functions from guardrails.go
// 	if e.config.EnableSafety {
// 		for i := range questions {
// 			if !CheckCurriculum(questions[i], req.SubjectID, req.Topic) {
// 				results[i].Status = "rejected"
// 				results[i].Errors = append(results[i].Errors, "topic mismatch")
// 			}
// 			if HasProfanity(questions[i].QuestionText) {
// 				results[i].Status = "rejected"
// 				results[i].Errors = append(results[i].Errors, "profanity detected")
// 			}
// 			// Check for duplicate questions
// 			for j, existing := range req.ExistingQuestions {
// 				if j != i && IsDuplicate(questions[i], existing) {
// 					results[i].Status = "rejected"
// 					results[i].Errors = append(results[i].Errors, "duplicate question")
// 					break
// 				}
// 			}
// 		}
// 	}

// 	// 5. Filter out rejected questions
// 	var finalQuestions []Question
// 	var finalResults []ValidationResult
// 	for i, r := range results {
// 		if r.Status != "rejected" {
// 			finalQuestions = append(finalQuestions, questions[i])
// 			finalResults = append(finalResults, r)
// 		}
// 	}

// 	log.Printf("Engine generated %d questions, accepted %d", len(questions), len(finalQuestions))
// 	return finalQuestions, finalResults, nil
// }

// // Config returns the engine configuration.
// func (e *Engine) Config() EngineConfig {
// 	e.mu.RLock()
// 	defer e.mu.RUnlock()
// 	return e.config
// }

// // UpdateConfig updates the engine configuration (thread-safe).
// func (e *Engine) UpdateConfig(config EngineConfig) {
// 	e.mu.Lock()
// 	defer e.mu.Unlock()
// 	e.config = config
// }

// // GetRefiner returns the refiner instance.
// func (e *Engine) GetRefiner() *Refiner {
// 	return e.refiner
// }

// // GetCritic returns the critic instance.
// func (e *Engine) GetCritic() *Critic {
// 	return e.critic
// }



// // package engine

// // import (
// // 	"context"
// // 	"log"
// // 	"sync"
// // 	"time"

// // 	"cbt-api/internal/ai/providers"
// // 	"cbt-api/internal/ai/safety"
// // 	"cbt-api/internal/ai/validation"
// // 	// ❌ REMOVED: "cbt-api/internal/ai/refinement"
// // 	// ❌ REMOVED: "cbt-api/internal/ai/generation"
// // )

// // // Engine is the main orchestrator for AI-powered exam generation.
// // type Engine struct {
// // 	router    *providers.Router
// // 	generator *Generator
// // 	validator *validation.Validator
// // 	refiner   *Refiner // ✅ Changed to local Refiner
// // 	critic    *Critic  // ✅ Added Critic
// // 	config    EngineConfig
// // 	mu        sync.RWMutex
// // }

// // // NewEngine creates a new exam intelligence engine.
// // func NewEngine(router *providers.Router, config EngineConfig) *Engine {
// // 	return &Engine{
// // 		router:    router,
// // 		generator: NewGenerator(router),
// // 		validator: validation.NewValidator(),
// // 		refiner:   NewRefiner(router), // ✅ Changed to local function
// // 		critic:    NewCritic(router),  // ✅ Added critic
// // 		config:    config,
// // 	}
// // }

// // // GenerateExam is the main entry point for question generation.
// // func (e *Engine) GenerateExam(ctx context.Context, req ExamContext) ([]Question, []ValidationResult, error) {
// // 	// Validate context
// // 	if err := req.Validate(); err != nil {
// // 		return nil, nil, err
// // 	}
// // 	if req.NumberOfQuestions < 1 {
// // 		req.NumberOfQuestions = 5
// // 	}
// // 	if e.config.MaxRefinementCycles == 0 {
// // 		e.config.MaxRefinementCycles = 2
// // 	}
// // 	if e.config.QualityThreshold == 0 {
// // 		e.config.QualityThreshold = 70
// // 	}

// // 	// Apply timeout if configured
// // 	if e.config.TimeoutSeconds > 0 {
// // 		var cancel context.CancelFunc
// // 		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.config.TimeoutSeconds)*time.Second)
// // 		defer cancel()
// // 	}

// // 	// 1. Generate raw questions
// // 	questions, err := e.generator.Generate(ctx, req)
// // 	if err != nil {
// // 		return nil, nil, err
// // 	}
// // 	if len(questions) == 0 {
// // 		return nil, nil, nil
// // 	}

// // 	// 2. Validate and score
// // 	results := e.validator.ValidateBatch(questions)

// // 	// 3. Refinement loop for weak questions
// // 	for cycle := 0; cycle < e.config.MaxRefinementCycles; cycle++ {
// // 		improved := false
// // 		for i, r := range results {
// // 			if r.Status == "needs_refinement" {
// // 				newQ, err := e.refiner.Improve(ctx, questions[i], r.Errors)
// // 				if err == nil {
// // 					questions[i] = newQ
// // 					newResults := e.validator.ValidateBatch([]Question{newQ})
// // 					results[i] = newResults[0]
// // 					improved = true
// // 				}
// // 			}
// // 		}
// // 		if !improved {
// // 			break
// // 		}
// // 	}

// // 	// 4. Safety checks (optional)
// // 	if e.config.EnableSafety {
// // 		for i := range questions {
// // 			if !safety.CheckCurriculum(questions[i], req.SubjectID, req.Topic) {
// // 				results[i].Status = "rejected"
// // 				results[i].Errors = append(results[i].Errors, "topic mismatch")
// // 			}
// // 			if safety.HasProfanity(questions[i].QuestionText) {
// // 				results[i].Status = "rejected"
// // 				results[i].Errors = append(results[i].Errors, "profanity detected")
// // 			}
// // 			// Check for duplicate questions
// // 			for j, existing := range req.ExistingQuestions {
// // 				if j != i && safety.IsDuplicate(questions[i], existing) {
// // 					results[i].Status = "rejected"
// // 					results[i].Errors = append(results[i].Errors, "duplicate question")
// // 					break
// // 				}
// // 			}
// // 		}
// // 	}

// // 	// 5. Filter out rejected questions
// // 	var finalQuestions []Question
// // 	var finalResults []ValidationResult
// // 	for i, r := range results {
// // 		if r.Status != "rejected" {
// // 			finalQuestions = append(finalQuestions, questions[i])
// // 			finalResults = append(finalResults, r)
// // 		}
// // 	}

// // 	log.Printf("Engine generated %d questions, accepted %d", len(questions), len(finalQuestions))
// // 	return finalQuestions, finalResults, nil
// // }

// // // Config returns the engine configuration.
// // func (e *Engine) Config() EngineConfig {
// // 	e.mu.RLock()
// // 	defer e.mu.RUnlock()
// // 	return e.config
// // }

// // // UpdateConfig updates the engine configuration (thread-safe).
// // func (e *Engine) UpdateConfig(config EngineConfig) {
// // 	e.mu.Lock()
// // 	defer e.mu.Unlock()
// // 	e.config = config
// // }

// // // GetRefiner returns the refiner instance.
// // func (e *Engine) GetRefiner() *Refiner {
// // 	return e.refiner
// // }

// // // GetCritic returns the critic instance.
// // func (e *Engine) GetCritic() *Critic {
// // 	return e.critic
// // }


// // // package engine

// // // import (
// // // 	"context"
// // // 	"log"
// // // 	"sync"
// // // 	"time"

// // // 	"cbt-api/internal/ai/providers"
// // // 	"cbt-api/internal/ai/refinement"
// // // 	"cbt-api/internal/ai/safety"
// // // 	"cbt-api/internal/ai/validation"
// // // 	// ❌ REMOVED: "cbt-api/internal/ai/generation"
// // // )

// // // // Engine is the main orchestrator for AI-powered exam generation.
// // // type Engine struct {
// // // 	router    *providers.Router
// // // 	generator *Generator // ✅ Changed to local Generator
// // // 	validator *validation.Validator
// // // 	refiner   *refinement.Refiner
// // // 	config    EngineConfig
// // // 	mu        sync.RWMutex
// // // }

// // // // NewEngine creates a new exam intelligence engine.
// // // func NewEngine(router *providers.Router, config EngineConfig) *Engine {
// // // 	return &Engine{
// // // 		router:    router,
// // // 		generator: NewGenerator(router), // ✅ Changed to local function
// // // 		validator: validation.NewValidator(),
// // // 		refiner:   refinement.NewRefiner(router),
// // // 		config:    config,
// // // 	}
// // // }

// // // // GenerateExam is the main entry point for question generation.
// // // func (e *Engine) GenerateExam(ctx context.Context, req ExamContext) ([]Question, []ValidationResult, error) {
// // // 	// Validate context
// // // 	if err := req.Validate(); err != nil {
// // // 		return nil, nil, err
// // // 	}
// // // 	if req.NumberOfQuestions < 1 {
// // // 		req.NumberOfQuestions = 5
// // // 	}
// // // 	if e.config.MaxRefinementCycles == 0 {
// // // 		e.config.MaxRefinementCycles = 2
// // // 	}
// // // 	if e.config.QualityThreshold == 0 {
// // // 		e.config.QualityThreshold = 70
// // // 	}

// // // 	// Apply timeout if configured
// // // 	if e.config.TimeoutSeconds > 0 {
// // // 		var cancel context.CancelFunc
// // // 		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.config.TimeoutSeconds)*time.Second)
// // // 		defer cancel()
// // // 	}

// // // 	// 1. Generate raw questions
// // // 	questions, err := e.generator.Generate(ctx, req)
// // // 	if err != nil {
// // // 		return nil, nil, err
// // // 	}
// // // 	if len(questions) == 0 {
// // // 		return nil, nil, nil
// // // 	}

// // // 	// 2. Validate and score
// // // 	results := e.validator.ValidateBatch(questions)

// // // 	// 3. Refinement loop for weak questions
// // // 	for cycle := 0; cycle < e.config.MaxRefinementCycles; cycle++ {
// // // 		improved := false
// // // 		for i, r := range results {
// // // 			if r.Status == "needs_refinement" {
// // // 				newQ, err := e.refiner.Improve(ctx, questions[i], r.Errors)
// // // 				if err == nil {
// // // 					questions[i] = newQ
// // // 					newResults := e.validator.ValidateBatch([]Question{newQ})
// // // 					results[i] = newResults[0]
// // // 					improved = true
// // // 				}
// // // 			}
// // // 		}
// // // 		if !improved {
// // // 			break
// // // 		}
// // // 	}

// // // 	// 4. Safety checks (optional)
// // // 	if e.config.EnableSafety {
// // // 		for i := range questions {
// // // 			if !safety.CheckCurriculum(questions[i], req.SubjectID, req.Topic) {
// // // 				results[i].Status = "rejected"
// // // 				results[i].Errors = append(results[i].Errors, "topic mismatch")
// // // 			}
// // // 			if safety.HasProfanity(questions[i].QuestionText) {
// // // 				results[i].Status = "rejected"
// // // 				results[i].Errors = append(results[i].Errors, "profanity detected")
// // // 			}
// // // 			// Check for duplicate questions
// // // 			for j, existing := range req.ExistingQuestions {
// // // 				if j != i && safety.IsDuplicate(questions[i], existing) {
// // // 					results[i].Status = "rejected"
// // // 					results[i].Errors = append(results[i].Errors, "duplicate question")
// // // 					break
// // // 				}
// // // 			}
// // // 		}
// // // 	}

// // // 	// 5. Filter out rejected questions
// // // 	var finalQuestions []Question
// // // 	var finalResults []ValidationResult
// // // 	for i, r := range results {
// // // 		if r.Status != "rejected" {
// // // 			finalQuestions = append(finalQuestions, questions[i])
// // // 			finalResults = append(finalResults, r)
// // // 		}
// // // 	}

// // // 	log.Printf("Engine generated %d questions, accepted %d", len(questions), len(finalQuestions))
// // // 	return finalQuestions, finalResults, nil
// // // }

// // // // Config returns the engine configuration.
// // // func (e *Engine) Config() EngineConfig {
// // // 	e.mu.RLock()
// // // 	defer e.mu.RUnlock()
// // // 	return e.config
// // // }

// // // // UpdateConfig updates the engine configuration (thread-safe).
// // // func (e *Engine) UpdateConfig(config EngineConfig) {
// // // 	e.mu.Lock()
// // // 	defer e.mu.Unlock()
// // // 	e.config = config
// // // }



// // // // package engine

// // // // import (
// // // // 	"context"
// // // // 	"log"
// // // // 	"sync"
// // // // 	"time"

// // // // 	"cbt-api/internal/ai/generation"
// // // // 	"cbt-api/internal/ai/providers"
// // // // 	"cbt-api/internal/ai/refinement"
// // // // 	"cbt-api/internal/ai/safety"
// // // // 	"cbt-api/internal/ai/validation"
// // // // )

// // // // // Engine is the main orchestrator for AI-powered exam generation.
// // // // type Engine struct {
// // // // 	router    *providers.Router
// // // // 	generator *generation.Generator
// // // // 	validator *validation.Validator
// // // // 	refiner   *refinement.Refiner
// // // // 	config    EngineConfig
// // // // 	mu        sync.RWMutex
// // // // }

// // // // // NewEngine creates a new exam intelligence engine.
// // // // func NewEngine(router *providers.Router, config EngineConfig) *Engine {
// // // // 	return &Engine{
// // // // 		router:    router,
// // // // 		generator: generation.NewGenerator(router),
// // // // 		validator: validation.NewValidator(),
// // // // 		refiner:   refinement.NewRefiner(router),
// // // // 		config:    config,
// // // // 	}
// // // // }

// // // // // GenerateExam is the main entry point for question generation.
// // // // func (e *Engine) GenerateExam(ctx context.Context, req ExamContext) ([]Question, []ValidationResult, error) {
// // // // 	// Validate context
// // // // 	if err := req.Validate(); err != nil {
// // // // 		return nil, nil, err
// // // // 	}
// // // // 	if req.NumberOfQuestions < 1 {
// // // // 		req.NumberOfQuestions = 5
// // // // 	}
// // // // 	if e.config.MaxRefinementCycles == 0 {
// // // // 		e.config.MaxRefinementCycles = 2
// // // // 	}
// // // // 	if e.config.QualityThreshold == 0 {
// // // // 		e.config.QualityThreshold = 70
// // // // 	}

// // // // 	// Apply timeout if configured
// // // // 	if e.config.TimeoutSeconds > 0 {
// // // // 		var cancel context.CancelFunc
// // // // 		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.config.TimeoutSeconds)*time.Second)
// // // // 		defer cancel()
// // // // 	}

// // // // 	// 1. Generate raw questions
// // // // 	questions, err := e.generator.Generate(ctx, req)
// // // // 	if err != nil {
// // // // 		return nil, nil, err
// // // // 	}
// // // // 	if len(questions) == 0 {
// // // // 		return nil, nil, nil
// // // // 	}

// // // // 	// 2. Validate and score
// // // // 	results := e.validator.ValidateBatch(questions)

// // // // 	// 3. Refinement loop for weak questions
// // // // 	for cycle := 0; cycle < e.config.MaxRefinementCycles; cycle++ {
// // // // 		improved := false
// // // // 		for i, r := range results {
// // // // 			if r.Status == "needs_refinement" {
// // // // 				newQ, err := e.refiner.Improve(ctx, questions[i], r.Errors)
// // // // 				if err == nil {
// // // // 					questions[i] = newQ
// // // // 					newResults := e.validator.ValidateBatch([]Question{newQ})
// // // // 					results[i] = newResults[0]
// // // // 					improved = true
// // // // 				}
// // // // 			}
// // // // 		}
// // // // 		if !improved {
// // // // 			break
// // // // 		}
// // // // 	}

// // // // 	// 4. Safety checks (optional)
// // // // 	if e.config.EnableSafety {
// // // // 		for i := range questions {
// // // // 			if !safety.CheckCurriculum(questions[i], req.SubjectID, req.Topic) {
// // // // 				results[i].Status = "rejected"
// // // // 				results[i].Errors = append(results[i].Errors, "topic mismatch")
// // // // 			}
// // // // 			if safety.HasProfanity(questions[i].QuestionText) {
// // // // 				results[i].Status = "rejected"
// // // // 				results[i].Errors = append(results[i].Errors, "profanity detected")
// // // // 			}
// // // // 			// Check for duplicate questions
// // // // 			for j, existing := range req.ExistingQuestions {
// // // // 				if j != i && safety.IsDuplicate(questions[i], existing) {
// // // // 					results[i].Status = "rejected"
// // // // 					results[i].Errors = append(results[i].Errors, "duplicate question")
// // // // 					break
// // // // 				}
// // // // 			}
// // // // 		}
// // // // 	}

// // // // 	// 5. Filter out rejected questions
// // // // 	var finalQuestions []Question
// // // // 	var finalResults []ValidationResult
// // // // 	for i, r := range results {
// // // // 		if r.Status != "rejected" {
// // // // 			finalQuestions = append(finalQuestions, questions[i])
// // // // 			finalResults = append(finalResults, r)
// // // // 		}
// // // // 	}

// // // // 	log.Printf("Engine generated %d questions, accepted %d", len(questions), len(finalQuestions))
// // // // 	return finalQuestions, finalResults, nil
// // // // }

// // // // // Config returns the engine configuration.
// // // // func (e *Engine) Config() EngineConfig {
// // // // 	e.mu.RLock()
// // // // 	defer e.mu.RUnlock()
// // // // 	return e.config
// // // // }

// // // // // UpdateConfig updates the engine configuration (thread-safe).
// // // // func (e *Engine) UpdateConfig(config EngineConfig) {
// // // // 	e.mu.Lock()
// // // // 	defer e.mu.Unlock()
// // // // 	e.config = config
// // // // }



// // // // // package engine

// // // // // import (
// // // // // 	"context"
// // // // // 	"log"
// // // // // 	"sync"
// // // // // 	"time"

// // // // // 	"cbt-api/internal/ai/generation"
// // // // // 	"cbt-api/internal/ai/providers"
// // // // // 	"cbt-api/internal/ai/refinement"
// // // // // 	"cbt-api/internal/ai/safety"
// // // // // 	"cbt-api/internal/ai/validation"
// // // // // )

// // // // // type Engine struct {
// // // // // 	router    *providers.Router
// // // // // 	generator *generation.Generator
// // // // // 	validator *validation.Validator
// // // // // 	refiner   *refinement.Refiner
// // // // // 	config    EngineConfig
// // // // // 	mu        sync.RWMutex
// // // // // }

// // // // // // NewEngine creates a new exam intelligence engine.
// // // // // func NewEngine(router *providers.Router, config EngineConfig) *Engine {
// // // // // 	return &Engine{
// // // // // 		router:    router,
// // // // // 		generator: generation.NewGenerator(router),
// // // // // 		validator: validation.NewValidator(),
// // // // // 		refiner:   refinement.NewRefiner(router),
// // // // // 		config:    config,
// // // // // 	}
// // // // // }

// // // // // // GenerateExam is the main entry point for question generation.
// // // // // func (e *Engine) GenerateExam(ctx context.Context, req ExamContext) ([]Question, []ValidationResult, error) {
// // // // // 	// Validate context
// // // // // 	if err := req.Validate(); err != nil {
// // // // // 		return nil, nil, err
// // // // // 	}
// // // // // 	if req.NumberOfQuestions < 1 {
// // // // // 		req.NumberOfQuestions = 5
// // // // // 	}
// // // // // 	if e.config.MaxRefinementCycles == 0 {
// // // // // 		e.config.MaxRefinementCycles = 2
// // // // // 	}
// // // // // 	if e.config.QualityThreshold == 0 {
// // // // // 		e.config.QualityThreshold = 70
// // // // // 	}

// // // // // 	// Apply timeout if configured
// // // // // 	if e.config.TimeoutSeconds > 0 {
// // // // // 		var cancel context.CancelFunc
// // // // // 		ctx, cancel = context.WithTimeout(ctx, time.Duration(e.config.TimeoutSeconds)*time.Second)
// // // // // 		defer cancel()
// // // // // 	}

// // // // // 	// 1. Generate raw questions
// // // // // 	questions, err := e.generator.Generate(ctx, req)
// // // // // 	if err != nil {
// // // // // 		return nil, nil, err
// // // // // 	}
// // // // // 	if len(questions) == 0 {
// // // // // 		return nil, nil, nil
// // // // // 	}

// // // // // 	// 2. Validate and score
// // // // // 	results := e.validator.ValidateBatch(questions)

// // // // // 	// 3. Refinement loop for weak questions
// // // // // 	for cycle := 0; cycle < e.config.MaxRefinementCycles; cycle++ {
// // // // // 		improved := false
// // // // // 		for i, r := range results {
// // // // // 			if r.Status == "needs_refinement" {
// // // // // 				newQ, err := e.refiner.Improve(ctx, questions[i], r.Errors)
// // // // // 				if err == nil {
// // // // // 					questions[i] = newQ
// // // // // 					newResults := e.validator.ValidateBatch([]Question{newQ})
// // // // // 					results[i] = newResults[0]
// // // // // 					improved = true
// // // // // 				}
// // // // // 			}
// // // // // 		}
// // // // // 		if !improved {
// // // // // 			break
// // // // // 		}
// // // // // 	}

// // // // // 	// 4. Safety checks (optional)
// // // // // 	if e.config.EnableSafety {
// // // // // 		for i := range questions {
// // // // // 			if !safety.CheckCurriculum(questions[i], req.SubjectID, req.Topic) {
// // // // // 				results[i].Status = "rejected"
// // // // // 				results[i].Errors = append(results[i].Errors, "topic mismatch")
// // // // // 			}
// // // // // 			if safety.HasProfanity(questions[i].QuestionText) {
// // // // // 				results[i].Status = "rejected"
// // // // // 				results[i].Errors = append(results[i].Errors, "profanity detected")
// // // // // 			}
// // // // // 		}
// // // // // 	}

// // // // // 	// 5. Filter out rejected questions
// // // // // 	var finalQuestions []Question
// // // // // 	var finalResults []ValidationResult
// // // // // 	for i, r := range results {
// // // // // 		if r.Status != "rejected" {
// // // // // 			finalQuestions = append(finalQuestions, questions[i])
// // // // // 			finalResults = append(finalResults, r)
// // // // // 		}
// // // // // 	}

// // // // // 	log.Printf("Engine generated %d questions, accepted %d", len(questions), len(finalQuestions))
// // // // // 	return finalQuestions, finalResults, nil
// // // // // }

// // // // // // Config returns the engine configuration.
// // // // // func (e *Engine) Config() EngineConfig {
// // // // // 	e.mu.RLock()
// // // // // 	defer e.mu.RUnlock()
// // // // // 	return e.config
// // // // // }

// // // // // // UpdateConfig updates the engine configuration (thread-safe).
// // // // // func (e *Engine) UpdateConfig(config EngineConfig) {
// // // // // 	e.mu.Lock()
// // // // // 	defer e.mu.Unlock()
// // // // // 	e.config = config
// // // // // }