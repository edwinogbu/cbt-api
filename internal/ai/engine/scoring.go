package engine

// ScoreQuestion returns a quality score (0-100) based on heuristics.
func ScoreQuestion(q Question) int {
	score := 100
	// Clarity: penalize short or unclear questions
	if len(q.QuestionText) < 20 {
		score -= 20
	} else if len(q.QuestionText) < 40 {
		score -= 10
	}
	// Options quality for MCQ
	if q.Type == "mcq" {
		if len(q.Options) < 3 {
			score -= 20
		} else if len(q.Options) < 4 {
			score -= 10
		}
	}
	// Explanation quality
	if q.Explanation == "" {
		score -= 15
	} else if len(q.Explanation) < 20 {
		score -= 5
	}
	// Rubric for essay
	if q.Type == "essay" && len(q.Rubric) == 0 {
		score -= 20
	}
	// Difficulty consistency
	if q.Difficulty == "" {
		score -= 10
	}
	// Bloom level
	if q.BloomLevel == "" {
		score -= 10
	}
	if score < 0 {
		score = 0
	}
	return score
}

// ScoreBatch returns scores for all questions.
func ScoreBatch(questions []Question) []int {
	scores := make([]int, len(questions))
	for i, q := range questions {
		scores[i] = ScoreQuestion(q)
	}
	return scores
}



// package validation

// import "cbt-api/internal/ai/engine"

// // ScoreQuestion returns a quality score (0-100) based on heuristics.
// func ScoreQuestion(q engine.Question) int {
// 	score := 100
// 	// Clarity: penalize short or unclear questions
// 	if len(q.QuestionText) < 20 {
// 		score -= 20
// 	} else if len(q.QuestionText) < 40 {
// 		score -= 10
// 	}
// 	// Options quality for MCQ
// 	if q.Type == "mcq" {
// 		if len(q.Options) < 3 {
// 			score -= 20
// 		} else if len(q.Options) < 4 {
// 			score -= 10
// 		}
// 	}
// 	// Explanation quality
// 	if q.Explanation == "" {
// 		score -= 15
// 	} else if len(q.Explanation) < 20 {
// 		score -= 5
// 	}
// 	// Rubric for essay
// 	if q.Type == "essay" && len(q.Rubric) == 0 {
// 		score -= 20
// 	}
// 	// Difficulty consistency
// 	if q.Difficulty == "" {
// 		score -= 10
// 	}
// 	// Bloom level
// 	if q.BloomLevel == "" {
// 		score -= 10
// 	}
// 	if score < 0 {
// 		score = 0
// 	}
// 	return score
// }

// // ScoreBatch returns scores for all questions.
// func ScoreBatch(questions []engine.Question) []int {
// 	scores := make([]int, len(questions))
// 	for i, q := range questions {
// 		scores[i] = ScoreQuestion(q)
// 	}
// 	return scores
// }




// // package validation

// // import "cbt-api/internal/ai/engine"

// // // ScoreQuestion returns a quality score (0-100) based on heuristics.
// // func ScoreQuestion(q engine.Question) int {
// // 	score := 100
// // 	// Clarity: penalize short or unclear questions
// // 	if len(q.QuestionText) < 20 {
// // 		score -= 20
// // 	} else if len(q.QuestionText) < 40 {
// // 		score -= 10
// // 	}
// // 	// Options quality for MCQ
// // 	if q.Type == "mcq" {
// // 		if len(q.Options) < 3 {
// // 			score -= 20
// // 		} else if len(q.Options) < 4 {
// // 			score -= 10
// // 		}
// // 	}
// // 	// Explanation quality
// // 	if q.Explanation == "" {
// // 		score -= 15
// // 	} else if len(q.Explanation) < 20 {
// // 		score -= 5
// // 	}
// // 	// Rubric for essay
// // 	if q.Type == "essay" && len(q.Rubric) == 0 {
// // 		score -= 20
// // 	}
// // 	// Difficulty consistency
// // 	if q.Difficulty == "" {
// // 		score -= 10
// // 	}
// // 	// Bloom level
// // 	if q.BloomLevel == "" {
// // 		score -= 10
// // 	}
// // 	if score < 0 {
// // 		score = 0
// // 	}
// // 	return score
// // }

// // // ScoreBatch returns scores for all questions.
// // func ScoreBatch(questions []engine.Question) []int {
// // 	scores := make([]int, len(questions))
// // 	for i, q := range questions {
// // 		scores[i] = ScoreQuestion(q)
// // 	}
// // 	return scores
// // }