package engine

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// ValidateBatch returns validation results for all questions.
func (v *Validator) ValidateBatch(questions []Question) []ValidationResult {
	results := make([]ValidationResult, len(questions))
	for i, q := range questions {
		errs := ValidateQuestion(q)
		score := ScoreQuestion(q)
		status := "rejected"
		if len(errs) == 0 && score >= 70 {
			status = "approved"
		} else if len(errs) == 0 && score >= 50 {
			status = "needs_refinement"
		}
		results[i] = ValidationResult{
			Valid:  len(errs) == 0,
			Errors: errs,
			Score:  score,
			Status: status,
		}
	}
	return results
}


// package validation

// import "cbt-api/internal/ai/engine"

// type Validator struct{}

// func NewValidator() *Validator {
// 	return &Validator{}
// }

// // ValidateBatch returns validation results for all questions.
// func (v *Validator) ValidateBatch(questions []engine.Question) []engine.ValidationResult {
// 	results := make([]engine.ValidationResult, len(questions))
// 	for i, q := range questions {
// 		errs := ValidateQuestion(q)
// 		score := ScoreQuestion(q)
// 		status := "rejected"
// 		if len(errs) == 0 && score >= 70 {
// 			status = "approved"
// 		} else if len(errs) == 0 && score >= 50 {
// 			status = "needs_refinement"
// 		}
// 		results[i] = engine.ValidationResult{
// 			Valid:  len(errs) == 0,
// 			Errors: errs,
// 			Score:  score,
// 			Status: status,
// 		}
// 	}
// 	return results
// }


// // package validation

// // import "cbt-api/internal/ai/engine"

// // type Validator struct{}

// // func NewValidator() *Validator {
// // 	return &Validator{}
// // }

// // // ValidateBatch returns only valid questions and their scores.
// // func (v *Validator) ValidateBatch(questions []engine.Question) []engine.ValidationResult {
// // 	results := make([]engine.ValidationResult, len(questions))
// // 	for i, q := range questions {
// // 		errs := ValidateQuestion(q)
// // 		score := ScoreQuestion(q)
// // 		status := "rejected"
// // 		if len(errs) == 0 && score >= 70 {
// // 			status = "approved"
// // 		} else if len(errs) == 0 && score >= 50 {
// // 			status = "needs_refinement"
// // 		}
// // 		results[i] = engine.ValidationResult{
// // 			Valid:  len(errs) == 0,
// // 			Errors: errs,
// // 			Score:  score,
// // 			Status: status,
// // 		}
// // 	}
// // 	return results
// // }