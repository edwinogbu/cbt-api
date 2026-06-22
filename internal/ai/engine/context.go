package engine

import "errors"

// NewExamContext creates a new exam context with defaults.
func NewExamContext() ExamContext {
	return ExamContext{
		NumberOfQuestions: 5,
		Difficulty:        "medium",
		BloomLevel:        "apply",
		CurriculumType:    "WAEC",
	}
}

// Validate validates the context before generation.
func (c ExamContext) Validate() error {
	if c.SubjectID == "" {
		return errors.New("subject_id is required")
	}
	if c.Topic == "" {
		return errors.New("topic is required")
	}
	if c.NumberOfQuestions < 1 || c.NumberOfQuestions > 100 {
		return errors.New("number_of_questions must be between 1 and 100")
	}
	if c.Difficulty == "" {
		return errors.New("difficulty is required")
	}
	if c.BloomLevel == "" {
		return errors.New("bloom_level is required")
	}
	return nil
}

// WithSchool sets the school ID.
func (c ExamContext) WithSchool(schoolID string) ExamContext {
	c.SchoolID = schoolID
	return c
}

// WithClass sets the class level ID.
func (c ExamContext) WithClass(classLevelID string) ExamContext {
	c.ClassLevelID = classLevelID
	return c
}



// package engine

// // NewExamContext creates a new exam context with defaults.
// func NewExamContext() ExamContext {
// 	return ExamContext{
// 		NumberOfQuestions: 5,
// 		Difficulty:        "medium",
// 		BloomLevel:        "apply",
// 		CurriculumType:    "WAEC",
// 	}
// }

// // Validate validates the context before generation.
// func (c ExamContext) Validate() error {
// 	if c.SubjectID == "" {
// 		return ErrMissingSubject
// 	}
// 	if c.Topic == "" {
// 		return ErrMissingTopic
// 	}
// 	if c.NumberOfQuestions < 1 || c.NumberOfQuestions > 100 {
// 		return ErrInvalidQuestionCount
// 	}
// 	return nil
// }

// // Predefined errors.
// var (
// 	ErrMissingSubject       = &ContextError{Field: "subject_id"}
// 	ErrMissingTopic         = &ContextError{Field: "topic"}
// 	ErrInvalidQuestionCount = &ContextError{Field: "number_of_questions"}
// )

// type ContextError struct {
// 	Field string
// }

// func (e *ContextError) Error() string {
// 	return "missing or invalid: " + e.Field
// }