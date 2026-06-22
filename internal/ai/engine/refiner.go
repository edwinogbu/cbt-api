package engine

import (
	"context"
	"encoding/json"

	"cbt-api/internal/ai/providers"
)

// Refiner improves weak questions using AI.
type Refiner struct {
	router *providers.Router
}

// NewRefiner creates a new Refiner instance.
func NewRefiner(router *providers.Router) *Refiner {
	return &Refiner{router: router}
}

// Improve takes a question with issues and asks AI to fix it.
func (r *Refiner) Improve(ctx context.Context, q Question, errors []string) (Question, error) {
	if len(errors) == 0 {
		return q, nil
	}
	prompt := BuildRefinementPrompt(q, errors)
	raw, err := r.router.Generate(ctx, prompt)
	if err != nil {
		return q, err
	}
	var improved Question
	if err := json.Unmarshal([]byte(raw), &improved); err != nil {
		return q, err
	}
	// Preserve original metadata if missing
	if improved.Topic == "" {
		improved.Topic = q.Topic
	}
	if improved.Difficulty == "" {
		improved.Difficulty = q.Difficulty
	}
	if improved.BloomLevel == "" {
		improved.BloomLevel = q.BloomLevel
	}
	if improved.Marks == 0 {
		improved.Marks = q.Marks
	}
	return improved, nil
}

// BatchImprove improves multiple questions.
func (r *Refiner) BatchImprove(ctx context.Context, questions []Question, errorsList [][]string) ([]Question, error) {
	improved := make([]Question, len(questions))
	for i, q := range questions {
		if len(errorsList) > i && len(errorsList[i]) > 0 {
			imp, err := r.Improve(ctx, q, errorsList[i])
			if err != nil {
				return nil, err
			}
			improved[i] = imp
		} else {
			improved[i] = q
		}
	}
	return improved, nil
}


// package refinement

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"

// 	"cbt-api/internal/ai/engine"
// 	"cbt-api/internal/ai/providers"
// )

// type Refiner struct {
// 	router *providers.Router
// }

// func NewRefiner(router *providers.Router) *Refiner {
// 	return &Refiner{router: router}
// }

// // Improve takes a question with issues and asks AI to fix it.
// func (r *Refiner) Improve(ctx context.Context, q engine.Question, errors []string) (engine.Question, error) {
// 	if len(errors) == 0 {
// 		return q, nil
// 	}
// 	prompt := engine.BuildRefinementPrompt(q, errors)
// 	raw, err := r.router.Generate(ctx, prompt)
// 	if err != nil {
// 		return q, err
// 	}
// 	var improved engine.Question
// 	if err := json.Unmarshal([]byte(raw), &improved); err != nil {
// 		return q, err
// 	}
// 	// Preserve original metadata if missing
// 	if improved.Topic == "" {
// 		improved.Topic = q.Topic
// 	}
// 	if improved.Difficulty == "" {
// 		improved.Difficulty = q.Difficulty
// 	}
// 	if improved.BloomLevel == "" {
// 		improved.BloomLevel = q.BloomLevel
// 	}
// 	if improved.Marks == 0 {
// 		improved.Marks = q.Marks
// 	}
// 	return improved, nil
// }



// // package refinement

// // import (
// // 	"context"
// // 	"encoding/json"
// // 	"fmt"

// // 	"cbt-api/internal/ai/engine"
// // 	"cbt-api/internal/ai/providers"
	
// // )

// // type Refiner struct {
// // 	router *providers.Router
// // }

// // func NewRefiner(router *providers.Router) *Refiner {
// // 	return &Refiner{router: router}
// // }

// // // Improve takes a question with issues and asks AI to fix it.
// // func (r *Refiner) Improve(ctx context.Context, q engine.Question, errors []string) (engine.Question, error) {
// // 	if len(errors) == 0 {
// // 		return q, nil
// // 	}
// // 	prompt := engine.BuildRefinementPrompt(q, errors)
// // 	raw, err := r.router.Generate(ctx, prompt)
// // 	if err != nil {
// // 		return q, err
// // 	}
// // 	var improved engine.Question
// // 	if err := json.Unmarshal([]byte(raw), &improved); err != nil {
// // 		return q, err
// // 	}
// // 	// Preserve original metadata if missing
// // 	if improved.Topic == "" {
// // 		improved.Topic = q.Topic
// // 	}
// // 	if improved.Difficulty == "" {
// // 		improved.Difficulty = q.Difficulty
// // 	}
// // 	if improved.BloomLevel == "" {
// // 		improved.BloomLevel = q.BloomLevel
// // 	}
// // 	if improved.Marks == 0 {
// // 		improved.Marks = q.Marks
// // 	}
// // 	return improved, nil
// // }


// // // package refinement

// // // import (
// // // 	"context"
// // // 	"encoding/json"
// // // 	"fmt"

// // // 	"cbt-api/internal/ai/engine"
// // // 	"cbt-api/internal/ai/generation"
// // // 	"cbt-api/internal/ai/providers"
// // // )

// // // type Refiner struct {
// // // 	router *providers.Router
// // // }

// // // func NewRefiner(router *providers.Router) *Refiner {
// // // 	return &Refiner{router: router}
// // // }

// // // // Improve takes a question with issues and asks AI to fix it.
// // // func (r *Refiner) Improve(ctx context.Context, q engine.Question, errors []string) (engine.Question, error) {
// // // 	if len(errors) == 0 {
// // // 		return q, nil
// // // 	}
// // // 	prompt := generation.BuildRefinementPrompt(q, errors)
// // // 	raw, err := r.router.Generate(ctx, prompt)
// // // 	if err != nil {
// // // 		return q, err
// // // 	}
// // // 	var improved engine.Question
// // // 	if err := json.Unmarshal([]byte(raw), &improved); err != nil {
// // // 		return q, err
// // // 	}
// // // 	// Preserve original metadata if missing
// // // 	if improved.Topic == "" {
// // // 		improved.Topic = q.Topic
// // // 	}
// // // 	if improved.Difficulty == "" {
// // // 		improved.Difficulty = q.Difficulty
// // // 	}
// // // 	if improved.BloomLevel == "" {
// // // 		improved.BloomLevel = q.BloomLevel
// // // 	}
// // // 	return improved, nil
// // // }