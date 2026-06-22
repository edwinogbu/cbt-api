package engine

// ValidateQuestion checks a single question against educational rules.
func ValidateQuestion(q Question) []string {
	var errors []string
	if q.QuestionText == "" {
		errors = append(errors, "question_text is empty")
	}
	if q.Type == "" {
		errors = append(errors, "type is required")
	}
	switch q.Type {
	case "mcq", "true_false":
		if len(q.Options) < 2 {
			errors = append(errors, "mcq needs at least 2 options")
		}
		if len(q.CorrectOptionKeys) == 0 {
			errors = append(errors, "correct_option_keys missing")
		}
		keyMap := make(map[string]bool)
		for _, opt := range q.Options {
			keyMap[opt.Key] = true
		}
		for _, key := range q.CorrectOptionKeys {
			if !keyMap[key] {
				errors = append(errors, "correct key "+key+" not in options")
			}
		}
	case "essay":
		if len(q.Rubric) == 0 {
			errors = append(errors, "essay must have rubric")
		}
		if len(q.Options) > 0 {
			errors = append(errors, "essay should not have options")
		}
	case "fill_blank":
		if len(q.Options) > 0 {
			errors = append(errors, "fill_blank should not have options")
		}
	default:
		errors = append(errors, "invalid type: "+q.Type)
	}
	if q.Marks < 0 {
		errors = append(errors, "marks cannot be negative")
	}
	if q.Difficulty == "" {
		errors = append(errors, "difficulty is required")
	}
	if q.BloomLevel == "" {
		errors = append(errors, "bloom_level is required")
	}
	return errors
}


// package validation

// import "cbt-api/internal/ai/engine"

// // ValidateQuestion checks a single question against educational rules.
// func ValidateQuestion(q engine.Question) []string {
// 	var errors []string
// 	if q.QuestionText == "" {
// 		errors = append(errors, "question_text is empty")
// 	}
// 	if q.Type == "" {
// 		errors = append(errors, "type is required")
// 	}
// 	switch q.Type {
// 	case "mcq", "true_false":
// 		if len(q.Options) < 2 {
// 			errors = append(errors, "mcq needs at least 2 options")
// 		}
// 		if len(q.CorrectOptionKeys) == 0 {
// 			errors = append(errors, "correct_option_keys missing")
// 		}
// 		keyMap := make(map[string]bool)
// 		for _, opt := range q.Options {
// 			keyMap[opt.Key] = true
// 		}
// 		for _, key := range q.CorrectOptionKeys {
// 			if !keyMap[key] {
// 				errors = append(errors, "correct key "+key+" not in options")
// 			}
// 		}
// 	case "essay":
// 		if len(q.Rubric) == 0 {
// 			errors = append(errors, "essay must have rubric")
// 		}
// 		if len(q.Options) > 0 {
// 			errors = append(errors, "essay should not have options")
// 		}
// 	case "fill_blank":
// 		if len(q.Options) > 0 {
// 			errors = append(errors, "fill_blank should not have options")
// 		}
// 	default:
// 		errors = append(errors, "invalid type: "+q.Type)
// 	}
// 	if q.Marks < 0 {
// 		errors = append(errors, "marks cannot be negative")
// 	}
// 	if q.Difficulty == "" {
// 		errors = append(errors, "difficulty is required")
// 	}
// 	if q.BloomLevel == "" {
// 		errors = append(errors, "bloom_level is required")
// 	}
// 	return errors
// }


// // package validation

// // import "cbt-api/internal/ai/engine"

// // // ValidateQuestion checks a single question against educational rules.
// // func ValidateQuestion(q engine.Question) []string {
// // 	var errors []string
// // 	if q.QuestionText == "" {
// // 		errors = append(errors, "question_text is empty")
// // 	}
// // 	if q.Type == "" {
// // 		errors = append(errors, "type is required")
// // 	}
// // 	switch q.Type {
// // 	case "mcq", "true_false":
// // 		if len(q.Options) < 2 {
// // 			errors = append(errors, "mcq needs at least 2 options")
// // 		}
// // 		if len(q.CorrectOptionKeys) == 0 {
// // 			errors = append(errors, "correct_option_keys missing")
// // 		}
// // 		keyMap := make(map[string]bool)
// // 		for _, opt := range q.Options {
// // 			keyMap[opt.Key] = true
// // 		}
// // 		for _, key := range q.CorrectOptionKeys {
// // 			if !keyMap[key] {
// // 				errors = append(errors, "correct key "+key+" not in options")
// // 			}
// // 		}
// // 	case "essay":
// // 		if len(q.Rubric) == 0 {
// // 			errors = append(errors, "essay must have rubric")
// // 		}
// // 		if len(q.Options) > 0 {
// // 			errors = append(errors, "essay should not have options")
// // 		}
// // 	case "fill_blank":
// // 		if len(q.Options) > 0 {
// // 			errors = append(errors, "fill_blank should not have options")
// // 		}
// // 	default:
// // 		errors = append(errors, "invalid type: "+q.Type)
// // 	}
// // 	if q.Marks < 0 {
// // 		errors = append(errors, "marks cannot be negative")
// // 	}
// // 	if q.Difficulty == "" {
// // 		errors = append(errors, "difficulty is required")
// // 	}
// // 	if q.BloomLevel == "" {
// // 		errors = append(errors, "bloom_level is required")
// // 	}
// // 	return errors
// // }