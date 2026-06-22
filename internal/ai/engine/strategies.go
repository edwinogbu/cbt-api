package engine

import (
	"context"
	// "encoding/json"
)

// Strategy defines how to generate a specific question type.
type Strategy interface {
	Generate(ctx context.Context, req ExamContext) (Question, error)
}

// MCQStrategy generates multiple-choice questions.
type MCQStrategy struct {
	generator *Generator
}

func NewMCQStrategy(g *Generator) *MCQStrategy {
	return &MCQStrategy{generator: g}
}

func (s *MCQStrategy) Generate(ctx context.Context, req ExamContext) (Question, error) {
	questions, err := s.generator.Generate(ctx, req)
	if err != nil {
		return Question{}, err
	}
	for _, q := range questions {
		if q.Type == "mcq" {
			return q, nil
		}
	}
	return Question{}, nil
}

// EssayStrategy generates essay questions.
type EssayStrategy struct {
	generator *Generator
}

func NewEssayStrategy(g *Generator) *EssayStrategy {
	return &EssayStrategy{generator: g}
}

func (s *EssayStrategy) Generate(ctx context.Context, req ExamContext) (Question, error) {
	questions, err := s.generator.Generate(ctx, req)
	if err != nil {
		return Question{}, err
	}
	for _, q := range questions {
		if q.Type == "essay" {
			return q, nil
		}
	}
	return Question{}, nil
}


// package generation

// import (
// 	"cbt-api/internal/ai/engine"
// 	"encoding/json"
// )

// // Strategy defines how to generate a specific question type.
// type Strategy interface {
// 	// Generate returns a question based on context.
// 	Generate(ctx engine.ExamContext) (engine.Question, error)
// }

// // MCQStrategy generates multiple-choice questions.
// type MCQStrategy struct {
// 	generator *Generator
// }

// func NewMCQStrategy(g *Generator) *MCQStrategy {
// 	return &MCQStrategy{generator: g}
// }

// func (s *MCQStrategy) Generate(ctx engine.ExamContext) (engine.Question, error) {
// 	// We reuse the generator but filter only MCQ type.
// 	questions, err := s.generator.Generate(context.Background(), ctx)
// 	if err != nil {
// 		return engine.Question{}, err
// 	}
// 	for _, q := range questions {
// 		if q.Type == "mcq" {
// 			return q, nil
// 		}
// 	}
// 	return engine.Question{}, nil
// }

// // EssayStrategy generates essay questions.
// type EssayStrategy struct {
// 	generator *Generator
// }

// func NewEssayStrategy(g *Generator) *EssayStrategy {
// 	return &EssayStrategy{generator: g}
// }

// func (s *EssayStrategy) Generate(ctx engine.ExamContext) (engine.Question, error) {
// 	questions, err := s.generator.Generate(context.Background(), ctx)
// 	if err != nil {
// 		return engine.Question{}, err
// 	}
// 	for _, q := range questions {
// 		if q.Type == "essay" {
// 			return q, nil
// 		}
// 	}
// 	return engine.Question{}, nil
// }


// package generation

// import (
// 	"cbt-api/internal/ai/engine"
// 	"encoding/json"
// )

// // Strategy defines how to generate a specific question type.
// type Strategy interface {
// 	// Generate returns a question based on context.
// 	Generate(ctx engine.ExamContext) (engine.Question, error)
// }

// // MCQStrategy generates multiple-choice questions.
// type MCQStrategy struct {
// 	generator *Generator
// }

// func NewMCQStrategy(g *Generator) *MCQStrategy {
// 	return &MCQStrategy{generator: g}
// }

// func (s *MCQStrategy) Generate(ctx engine.ExamContext) (engine.Question, error) {
// 	// We reuse the generator but filter only MCQ type.
// 	questions, err := s.generator.Generate(context.Background(), ctx)
// 	if err != nil {
// 		return engine.Question{}, err
// 	}
// 	for _, q := range questions {
// 		if q.Type == "mcq" {
// 			return q, nil
// 		}
// 	}
// 	return engine.Question{}, nil
// }

// // EssayStrategy generates essay questions.
// type EssayStrategy struct {
// 	generator *Generator
// }

// func NewEssayStrategy(g *Generator) *EssayStrategy {
// 	return &EssayStrategy{generator: g}
// }

// func (s *EssayStrategy) Generate(ctx engine.ExamContext) (engine.Question, error) {
// 	questions, err := s.generator.Generate(context.Background(), ctx)
// 	if err != nil {
// 		return engine.Question{}, err
// 	}
// 	for _, q := range questions {
// 		if q.Type == "essay" {
// 			return q, nil
// 		}
// 	}
// 	return engine.Question{}, nil
// }



// // package generation

// // import "cbt-api/internal/ai/engine"

// // // Strategy defines how to generate a specific question type.
// // type Strategy interface {
// // 	BuildPrompt(topic, difficulty, bloom string) string
// // 	ParseResponse(raw string) (engine.Question, error)
// // }

// // // MCQStrategy generates multiple-choice questions.
// // type MCQStrategy struct{}

// // func (s *MCQStrategy) BuildPrompt(topic, difficulty, bloom string) string {
// // 	return BuildGenerationPrompt(engine.ExamContext{
// // 		Topic:      topic,
// // 		Difficulty: difficulty,
// // 		BloomLevel: bloom,
// // 	})
// // }

// // func (s *MCQStrategy) ParseResponse(raw string) (engine.Question, error) {
// // 	// reuse generator parse
// // 	return engine.Question{}, nil
// // }

// // // EssayStrategy generates essay questions.
// // type EssayStrategy struct{}

// // func (s *EssayStrategy) BuildPrompt(topic, difficulty, bloom string) string {
// // 	// ...
// // 	return ""
// // }

// // func (s *EssayStrategy) ParseResponse(raw string) (engine.Question, error) {
// // 	return engine.Question{}, nil
// // }