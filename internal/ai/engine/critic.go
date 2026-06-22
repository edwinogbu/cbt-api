package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"cbt-api/internal/ai/providers"
)

// Critic evaluates question quality using AI.
type Critic struct {
	router *providers.Router
}

// NewCritic creates a new Critic instance.
func NewCritic(router *providers.Router) *Critic {
	return &Critic{router: router}
}

// Critique asks AI to evaluate a question and return detailed feedback.
func (c *Critic) Critique(ctx context.Context, q Question) (string, error) {
	prompt := BuildCritiquePrompt(q)
	raw, err := c.router.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("critique failed: %w", err)
	}
	return raw, nil
}

// ScoreCritique parses the critique output and returns a numeric score (0-100).
func (c *Critic) ScoreCritique(critique string) int {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(critique), &result); err != nil {
		return 0
	}
	if clarity, ok := result["clarity_score"].(float64); ok {
		return int(clarity * 10) // scale to 0-100
	}
	return 0
}

// EvaluateQuestion combines critique and scoring into one call.
func (c *Critic) EvaluateQuestion(ctx context.Context, q Question) (score int, feedback string, err error) {
	feedback, err = c.Critique(ctx, q)
	if err != nil {
		return 0, "", err
	}
	score = c.ScoreCritique(feedback)
	return score, feedback, nil
}

// ValidateCritiqueResponse checks if the critique is valid JSON.
func (c *Critic) ValidateCritiqueResponse(raw string) bool {
	var result map[string]interface{}
	return json.Unmarshal([]byte(raw), &result) == nil
}

// BatchCritique evaluates multiple questions.
func (c *Critic) BatchCritique(ctx context.Context, questions []Question) ([]string, []int, error) {
	feedbacks := make([]string, len(questions))
	scores := make([]int, len(questions))
	for i, q := range questions {
		feedback, err := c.Critique(ctx, q)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to critique question %d: %w", i, err)
		}
		feedbacks[i] = feedback
		scores[i] = c.ScoreCritique(feedback)
	}
	return feedbacks, scores, nil
}



// package refinement

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"

// 	"cbt-api/internal/ai/engine"
// 	"cbt-api/internal/ai/generation"
// 	"cbt-api/internal/ai/providers"
// )

// type Critic struct {
// 	router *providers.Router
// }

// func NewCritic(router *providers.Router) *Critic {
// 	return &Critic{router: router}
// }

// // Critique asks AI to evaluate a question and return feedback.
// func (c *Critic) Critique(ctx context.Context, q engine.Question) (string, error) {
// 	prompt := generation.BuildCritiquePrompt(q)
// 	raw, err := c.router.Generate(ctx, prompt)
// 	if err != nil {
// 		return "", err
// 	}
// 	return raw, nil
// }

// // ScoreCritique parses the critique output and returns a numeric score.
// func (c *Critic) ScoreCritique(critique string) int {
// 	// Simple parsing – in production, use structured output
// 	var result map[string]interface{}
// 	if err := json.Unmarshal([]byte(critique), &result); err != nil {
// 		return 0
// 	}
// 	if clarity, ok := result["clarity_score"].(float64); ok {
// 		return int(clarity * 10) // scale to 0-100
// 	}
// 	return 0
// }


// // package refinement

// // import (
// // 	"context"
// // 	"encoding/json"
// // 	"fmt"

// // 	"cbt-api/internal/ai/engine"
// // 	"cbt-api/internal/ai/providers"
// // )

// // type Critic struct {
// // 	router *providers.Router
// // }

// // func NewCritic(router *providers.Router) *Critic {
// // 	return &Critic{router: router}
// // }

// // // Critique asks AI to evaluate a question and return feedback.
// // func (c *Critic) Critique(ctx context.Context, q engine.Question) (string, error) {
// // 	prompt := fmt.Sprintf(`
// // Evaluate this exam question for quality, clarity, and educational value.
// // Provide feedback on:
// // 1. Clarity of the question
// // 2. Appropriateness of difficulty
// // 3. Correctness of the answer
// // 4. Suggestions for improvement

// // Question: %s
// // Correct answer: %v
// // Difficulty: %s
// // Bloom level: %s

// // Return ONLY JSON with fields: clarity_score, difficulty_match, answer_correctness, suggestions.
// // `, q.QuestionText, q.CorrectOptionKeys, q.Difficulty, q.BloomLevel)

// // 	raw, err := c.router.Generate(ctx, prompt)
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	return raw, nil
// // }