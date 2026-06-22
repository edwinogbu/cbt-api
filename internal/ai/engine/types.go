package engine

// import "time"

// Question is the internal representation used by the engine.
type Question struct {
	ID                string   `json:"id"`
	QuestionText      string   `json:"question_text"`
	Options           []Option `json:"options"`
	CorrectOptionKeys []string `json:"correct_option_keys"`
	Explanation       string   `json:"explanation"`
	Marks             int      `json:"marks"`
	Type              string   `json:"type"` // mcq, essay, fill_blank
	Difficulty        string   `json:"difficulty"`
	BloomLevel        string   `json:"bloom_level"`
	Topic             string   `json:"topic"`
	SubTopic          string   `json:"sub_topic"`
	LearningObjective string   `json:"learning_objective"`
	Rubric            []Rubric `json:"rubric,omitempty"`
	Tags              []string `json:"tags"`
}

// Option represents a multiple-choice option.
type Option struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

// Rubric defines criteria for essay grading.
type Rubric struct {
	Criteria string `json:"criteria"`
	Marks    int    `json:"marks"`
}

// ExamContext holds metadata for generation.
type ExamContext struct {
	SchoolID          string
	ClassLevelID      string
	SubjectID         string
	Topic             string
	SubTopic          string
	NumberOfQuestions int
	Difficulty        string
	BloomLevel        string
	CurriculumType    string
	SourceText        string
	Keywords          []string
	ExistingQuestions []Question // for deduplication
}

// ValidationResult holds validation status.
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors"`
	Score   int      `json:"score"`
	Status  string   `json:"status"` // approved, needs_refinement, rejected
}

// EngineConfig configures the engine behavior.
type EngineConfig struct {
	MaxRefinementCycles int `json:"max_refinement_cycles"` // default 2
	QualityThreshold    int `json:"quality_threshold"`     // default 70
	EnableSafety        bool `json:"enable_safety"`
	EnableScoring       bool `json:"enable_scoring"`
	TimeoutSeconds      int  `json:"timeout_seconds"`
}

// package engine

// import "time"

// // Question is the internal representation used by the engine.
// type Question struct {
// 	ID                string   `json:"id"`
// 	QuestionText      string   `json:"question_text"`
// 	Options           []Option `json:"options"`
// 	CorrectOptionKeys []string `json:"correct_option_keys"`
// 	Explanation       string   `json:"explanation"`
// 	Marks             int      `json:"marks"`
// 	Type              string   `json:"type"` // mcq, essay, fill_blank
// 	Difficulty        string   `json:"difficulty"`
// 	BloomLevel        string   `json:"bloom_level"`
// 	Topic             string   `json:"topic"`
// 	SubTopic          string   `json:"sub_topic"`
// 	LearningObjective string   `json:"learning_objective"`
// 	Rubric            []Rubric `json:"rubric,omitempty"`
// 	Tags              []string `json:"tags"`
// }

// // Option represents a multiple-choice option.
// type Option struct {
// 	Key  string `json:"key"`
// 	Text string `json:"text"`
// }

// // Rubric defines criteria for essay grading.
// type Rubric struct {
// 	Criteria string `json:"criteria"`
// 	Marks    int    `json:"marks"`
// }

// // ExamContext holds metadata for generation.
// type ExamContext struct {
// 	SchoolID          string
// 	ClassLevelID      string
// 	SubjectID         string
// 	Topic             string
// 	SubTopic          string
// 	NumberOfQuestions int
// 	Difficulty        string
// 	BloomLevel        string
// 	CurriculumType    string
// 	SourceText        string
// 	Keywords          []string
// 	ExistingQuestions []Question // for deduplication
// }

// // ValidationResult holds validation status.
// type ValidationResult struct {
// 	Valid   bool     `json:"valid"`
// 	Errors  []string `json:"errors"`
// 	Score   int      `json:"score"`
// 	Status  string   `json:"status"` // approved, needs_refinement, rejected
// }

// // EngineConfig configures the engine behavior.
// type EngineConfig struct {
// 	MaxRefinementCycles int `json:"max_refinement_cycles"` // default 2
// 	QualityThreshold    int `json:"quality_threshold"`     // default 70
// 	EnableSafety        bool `json:"enable_safety"`
// 	EnableScoring       bool `json:"enable_scoring"`
// 	TimeoutSeconds      int  `json:"timeout_seconds"`
// }



// // package engine

// // import "time"

// // // Question is the internal representation used by the engine.
// // type Question struct {
// // 	ID                string   `json:"id"`
// // 	QuestionText      string   `json:"question_text"`
// // 	Options           []Option `json:"options"`
// // 	CorrectOptionKeys []string `json:"correct_option_keys"`
// // 	Explanation       string   `json:"explanation"`
// // 	Marks             int      `json:"marks"`
// // 	Type              string   `json:"type"` // mcq, essay, fill_blank
// // 	Difficulty        string   `json:"difficulty"`
// // 	BloomLevel        string   `json:"bloom_level"`
// // 	Topic             string   `json:"topic"`
// // 	SubTopic          string   `json:"sub_topic"`
// // 	LearningObjective string   `json:"learning_objective"`
// // 	Rubric            []Rubric `json:"rubric,omitempty"`
// // 	Tags              []string `json:"tags"`
// // }

// // // Option represents a multiple-choice option.
// // type Option struct {
// // 	Key  string `json:"key"`
// // 	Text string `json:"text"`
// // }

// // // Rubric defines criteria for essay grading.
// // type Rubric struct {
// // 	Criteria string `json:"criteria"`
// // 	Marks    int    `json:"marks"`
// // }

// // // ExamContext holds metadata for generation.
// // type ExamContext struct {
// // 	SchoolID         string
// // 	ClassLevelID     string
// // 	SubjectID        string
// // 	Topic            string
// // 	SubTopic         string
// // 	NumberOfQuestions int
// // 	Difficulty       string
// // 	BloomLevel       string
// // 	CurriculumType   string
// // 	SourceText       string
// // 	Keywords         []string
// // 	ExistingQuestions []Question // for deduplication
// // }

// // // ValidationResult holds validation status.
// // type ValidationResult struct {
// // 	Valid   bool     `json:"valid"`
// // 	Errors  []string `json:"errors"`
// // 	Score   int      `json:"score"`
// // 	Status  string   `json:"status"` // approved, needs_refinement, rejected
// // }

// // // EngineConfig configures the engine behavior.
// // type EngineConfig struct {
// // 	MaxRefinementCycles int     `json:"max_refinement_cycles"` // default 2
// // 	QualityThreshold    int     `json:"quality_threshold"`     // default 70
// // 	EnableSafety        bool    `json:"enable_safety"`
// // 	EnableScoring       bool    `json:"enable_scoring"`
// // 	TimeoutSeconds      int     `json:"timeout_seconds"`
// // }