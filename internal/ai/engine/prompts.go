package engine

import (
	"fmt"
	"strings"
)

// BuildGenerationPrompt creates the prompt for generating questions.
func BuildGenerationPrompt(ctx ExamContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`You are an expert CBT question generator.

Generate %d questions for:
Subject: %s
Topic: %s
Difficulty: %s
Bloom's Level: %s
Curriculum: %s

`, ctx.NumberOfQuestions, ctx.SubjectID, ctx.Topic, ctx.Difficulty, ctx.BloomLevel, ctx.CurriculumType))

	if ctx.SubTopic != "" {
		sb.WriteString(fmt.Sprintf("Sub-topic: %s\n", ctx.SubTopic))
	}
	if len(ctx.Keywords) > 0 {
		sb.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(ctx.Keywords, ", ")))
	}
	if ctx.SourceText != "" {
		sb.WriteString(fmt.Sprintf("Base questions on this text: %s\n", ctx.SourceText))
	}

	sb.WriteString(`
Return **ONLY** valid JSON array of objects with these fields:
- question_text (string)
- options (array of {key, text}) – required for mcq, optional for others
- correct_option_keys (array of strings) – required for mcq
- explanation (string)
- marks (int)
- type (string: "mcq", "essay", "fill_blank")
- difficulty (string: "easy", "medium", "hard", "expert")
- bloom_level (string: "remember", "understand", "apply", "analyse", "evaluate", "create")
- topic (string) – use the provided topic
- sub_topic (string) – optional
- learning_objective (string) – clear learning objective
- rubric (array of {criteria, marks}) – required for essay
- tags (array of strings)

Rules:
- For MCQ, provide exactly 4 options (A, B, C, D) with one correct key.
- For true/false, use options A:True, B:False.
- For essay, provide a clear prompt and a rubric with at least 2 criteria.
- Ensure difficulty and Bloom level match the request exactly.
- No markdown, no extra text. Only the JSON array.`)

	return sb.String()
}

// BuildRefinementPrompt creates a prompt for improving a weak question.
func BuildRefinementPrompt(q Question, errors []string) string {
	return fmt.Sprintf(`
The following question failed validation:
Question: %s
Errors: %s

Please rewrite the question to fix all errors.
Return **ONLY** valid JSON with the same structure as the original.
Make sure the question is clear, educationally sound, and matches the difficulty level.
`, q.QuestionText, strings.Join(errors, "; "))
}

// BuildCritiquePrompt asks AI to evaluate a question.
func BuildCritiquePrompt(q Question) string {
	return fmt.Sprintf(`
Evaluate this exam question for quality, clarity, and educational value.
Provide feedback on:
1. Clarity of the question
2. Appropriateness of difficulty
3. Correctness of the answer
4. Suggestions for improvement

Question: %s
Correct answer: %v
Difficulty: %s
Bloom level: %s

Return ONLY JSON with fields: clarity_score, difficulty_match, answer_correctness, suggestions.
`, q.QuestionText, q.CorrectOptionKeys, q.Difficulty, q.BloomLevel)
}




// package engine

// import (
// 	"fmt"
// 	"strings"
// )

// // BuildGenerationPrompt creates the prompt for generating questions.
// func BuildGenerationPrompt(ctx ExamContext) string {
// 	var sb strings.Builder
// 	sb.WriteString(fmt.Sprintf(`You are an expert CBT question generator.

// Generate %d questions for:
// Subject: %s
// Topic: %s
// Difficulty: %s
// Bloom's Level: %s
// Curriculum: %s

// `, ctx.NumberOfQuestions, ctx.SubjectID, ctx.Topic, ctx.Difficulty, ctx.BloomLevel, ctx.CurriculumType))

// 	if ctx.SubTopic != "" {
// 		sb.WriteString(fmt.Sprintf("Sub-topic: %s\n", ctx.SubTopic))
// 	}
// 	if len(ctx.Keywords) > 0 {
// 		sb.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(ctx.Keywords, ", ")))
// 	}
// 	if ctx.SourceText != "" {
// 		sb.WriteString(fmt.Sprintf("Base questions on this text: %s\n", ctx.SourceText))
// 	}

// 	sb.WriteString(`
// Return **ONLY** valid JSON array of objects with these fields:
// - question_text (string)
// - options (array of {key, text}) – required for mcq, optional for others
// - correct_option_keys (array of strings) – required for mcq
// - explanation (string)
// - marks (int)
// - type (string: "mcq", "essay", "fill_blank")
// - difficulty (string: "easy", "medium", "hard", "expert")
// - bloom_level (string: "remember", "understand", "apply", "analyse", "evaluate", "create")
// - topic (string) – use the provided topic
// - sub_topic (string) – optional
// - learning_objective (string) – clear learning objective
// - rubric (array of {criteria, marks}) – required for essay
// - tags (array of strings)

// Rules:
// - For MCQ, provide exactly 4 options (A, B, C, D) with one correct key.
// - For true/false, use options A:True, B:False.
// - For essay, provide a clear prompt and a rubric with at least 2 criteria.
// - Ensure difficulty and Bloom level match the request exactly.
// - No markdown, no extra text. Only the JSON array.`)

// 	return sb.String()
// }

// // BuildRefinementPrompt creates a prompt for improving a weak question.
// func BuildRefinementPrompt(q Question, errors []string) string {
// 	return fmt.Sprintf(`
// The following question failed validation:
// Question: %s
// Errors: %s

// Please rewrite the question to fix all errors.
// Return **ONLY** valid JSON with the same structure as the original.
// Make sure the question is clear, educationally sound, and matches the difficulty level.
// `, q.QuestionText, strings.Join(errors, "; "))
// }

// // BuildCritiquePrompt asks AI to evaluate a question.
// func BuildCritiquePrompt(q Question) string {
// 	return fmt.Sprintf(`
// Evaluate this exam question for quality, clarity, and educational value.
// Provide feedback on:
// 1. Clarity of the question
// 2. Appropriateness of difficulty
// 3. Correctness of the answer
// 4. Suggestions for improvement

// Question: %s
// Correct answer: %v
// Difficulty: %s
// Bloom level: %s

// Return ONLY JSON with fields: clarity_score, difficulty_match, answer_correctness, suggestions.
// `, q.QuestionText, q.CorrectOptionKeys, q.Difficulty, q.BloomLevel)
// }