package providers

import (
	"context"
	"encoding/json"
)

// FallbackProvider is a mock that always works (used as last resort).
type FallbackProvider struct{}

func NewFallbackProvider() *FallbackProvider {
	return &FallbackProvider{}
}

func (p *FallbackProvider) Name() string { return "fallback" }

func (p *FallbackProvider) Available(ctx context.Context) bool { return true }

func (p *FallbackProvider) Generate(ctx context.Context, prompt string) (string, error) {
	// Return a generic mock question
	mock := []map[string]interface{}{
		{
			"question_text":       "What is the capital of France? (fallback)",
			"options":             []map[string]string{{"key": "A", "text": "Paris"}, {"key": "B", "text": "London"}},
			"correct_option_keys": []string{"A"},
			"explanation":         "Paris is the capital of France.",
			"marks":               1,
			"type":                "mcq",
			"difficulty":          "easy",
			"bloom_level":         "remember",
			"topic":               "Geography",
			"sub_topic":           "Capitals",
			"learning_objective":  "Recall the capital of France.",
			"tags":                []string{"geography", "capital"},
		},
	}
	b, _ := json.Marshal(mock)
	return string(b), nil
}


// package providers

// import (
// 	"context"
// 	"encoding/json"
// )

// // FallbackProvider is a mock that always works (used as last resort).
// type FallbackProvider struct{}

// func NewFallbackProvider() *FallbackProvider {
// 	return &FallbackProvider{}
// }

// func (p *FallbackProvider) Name() string { return "fallback" }

// func (p *FallbackProvider) Available(ctx context.Context) bool { return true }

// func (p *FallbackProvider) Generate(ctx context.Context, prompt string) (string, error) {
// 	// Return a generic mock question
// 	mock := []map[string]interface{}{
// 		{
// 			"question_text":       "What is the capital of France? (fallback)",
// 			"options":             []map[string]string{{"key": "A", "text": "Paris"}, {"key": "B", "text": "London"}},
// 			"correct_option_keys": []string{"A"},
// 			"explanation":         "Paris is the capital of France.",
// 			"marks":               1,
// 			"type":                "mcq",
// 			"difficulty":          "easy",
// 			"bloom_level":         "remember",
// 			"topic":               "Geography",
// 			"sub_topic":           "Capitals",
// 			"learning_objective":  "Recall the capital of France.",
// 			"tags":                []string{"geography", "capital"},
// 		},
// 	}
// 	b, _ := json.Marshal(mock)
// 	return string(b), nil
// }