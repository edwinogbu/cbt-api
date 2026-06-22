package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"cbt-api/internal/ai/providers"
)

type Generator struct {
	router *providers.Router
}

func NewGenerator(router *providers.Router) *Generator {
	return &Generator{router: router}
}

// Generate returns a list of raw questions (as engine.Question) before validation.
func (g *Generator) Generate(ctx context.Context, req ExamContext) ([]Question, error) {
	prompt := BuildGenerationPrompt(req)
	raw, err := g.router.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}
	var questions []Question
	if err := json.Unmarshal([]byte(raw), &questions); err != nil {
		return nil, fmt.Errorf("failed to parse AI output: %w", err)
	}
	// Assign default values if missing
	for i := range questions {
		if questions[i].Marks == 0 {
			questions[i].Marks = 1
		}
		if questions[i].Difficulty == "" {
			questions[i].Difficulty = req.Difficulty
		}
		if questions[i].BloomLevel == "" {
			questions[i].BloomLevel = req.BloomLevel
		}
		if questions[i].Topic == "" {
			questions[i].Topic = req.Topic
		}
		if questions[i].Type == "" {
			questions[i].Type = "mcq"
		}
	}
	return questions, nil
}



// package engine

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"

// 	"cbt-api/internal/ai/providers"
// )

// type Generator struct {
// 	router *providers.Router
// }

// func NewGenerator(router *providers.Router) *Generator {
// 	return &Generator{router: router}
// }

// // Generate returns a list of raw questions (as engine.Question) before validation.
// func (g *Generator) Generate(ctx context.Context, req ExamContext) ([]Question, error) {
// 	prompt := BuildGenerationPrompt(req)
// 	raw, err := g.router.Generate(ctx, prompt)
// 	if err != nil {
// 		return nil, fmt.Errorf("AI generation failed: %w", err)
// 	}
// 	var questions []Question
// 	if err := json.Unmarshal([]byte(raw), &questions); err != nil {
// 		return nil, fmt.Errorf("failed to parse AI output: %w", err)
// 	}
// 	// Assign default values if missing
// 	for i := range questions {
// 		if questions[i].Marks == 0 {
// 			questions[i].Marks = 1
// 		}
// 		if questions[i].Difficulty == "" {
// 			questions[i].Difficulty = req.Difficulty
// 		}
// 		if questions[i].BloomLevel == "" {
// 			questions[i].BloomLevel = req.BloomLevel
// 		}
// 		if questions[i].Topic == "" {
// 			questions[i].Topic = req.Topic
// 		}
// 		if questions[i].Type == "" {
// 			questions[i].Type = "mcq"
// 		}
// 	}
// 	return questions, nil
// }