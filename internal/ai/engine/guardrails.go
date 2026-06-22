package engine

import (
	"strings"
)

// CheckCurriculum ensures topic matches subject.
func CheckCurriculum(q Question, subjectID, topic string) bool {
	text := strings.ToLower(q.QuestionText + " " + q.Topic)
	if topic != "" && strings.Contains(text, strings.ToLower(topic)) {
		return true
	}
	if subjectID != "" && strings.Contains(text, strings.ToLower(subjectID)) {
		return true
	}
	return false
}

// HasProfanity checks for inappropriate language.
func HasProfanity(text string) bool {
	profanityList := []string{"profane", "offensive", "badword", "idiot", "stupid"}
	lower := strings.ToLower(text)
	for _, word := range profanityList {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

// IsDuplicate checks if two questions are identical (or near-identical).
func IsDuplicate(a, b Question) bool {
	if a.QuestionText == b.QuestionText {
		return true
	}
	// Could add more advanced similarity check (e.g., Jaccard)
	return false
}

// IsTopicAppropriate checks if the topic is within allowed subjects.
func IsTopicAppropriate(topic string, allowedSubjects []string) bool {
	if len(allowedSubjects) == 0 {
		return true
	}
	lower := strings.ToLower(topic)
	for _, s := range allowedSubjects {
		if strings.Contains(lower, strings.ToLower(s)) {
			return true
		}
	}
	return false
}


// package safety

// import (
// 	"strings"

// 	"cbt-api/internal/ai/engine"
// )

// // CheckCurriculum ensures topic matches subject.
// func CheckCurriculum(q engine.Question, subjectID, topic string) bool {
// 	text := strings.ToLower(q.QuestionText + " " + q.Topic)
// 	if topic != "" && strings.Contains(text, strings.ToLower(topic)) {
// 		return true
// 	}
// 	if subjectID != "" && strings.Contains(text, strings.ToLower(subjectID)) {
// 		return true
// 	}
// 	return false
// }

// // HasProfanity checks for inappropriate language.
// func HasProfanity(text string) bool {
// 	profanityList := []string{"profane", "offensive", "badword", "idiot", "stupid"}
// 	lower := strings.ToLower(text)
// 	for _, word := range profanityList {
// 		if strings.Contains(lower, word) {
// 			return true
// 		}
// 	}
// 	return false
// }

// // IsDuplicate checks if two questions are identical (or near-identical).
// func IsDuplicate(a, b engine.Question) bool {
// 	if a.QuestionText == b.QuestionText {
// 		return true
// 	}
// 	// Could add more advanced similarity check (e.g., Jaccard)
// 	return false
// }


// // package safety

// // import (
// // 	"strings"

// // 	"cbt-api/internal/ai/engine"
// // )

// // // CheckCurriculum ensures topic matches subject.
// // func CheckCurriculum(q engine.Question, subjectID, topic string) bool {
// // 	// Simple keyword check – can be enhanced with embeddings
// // 	text := strings.ToLower(q.QuestionText + " " + q.Topic)
// // 	if topic != "" && strings.Contains(text, strings.ToLower(topic)) {
// // 		return true
// // 	}
// // 	if subjectID != "" && strings.Contains(text, strings.ToLower(subjectID)) {
// // 		return true
// // 	}
// // 	return false
// // }

// // // HasProfanity checks for inappropriate language.
// // func HasProfanity(text string) bool {
// // 	profanityList := []string{"profane", "offensive", "badword"}
// // 	lower := strings.ToLower(text)
// // 	for _, word := range profanityList {
// // 		if strings.Contains(lower, word) {
// // 			return true
// // 		}
// // 	}
// // 	return false
// // }

// // // IsTopicAppropriate checks if the topic is within allowed subjects.
// // func IsTopicAppropriate(topic string, allowedSubjects []string) bool {
// // 	if len(allowedSubjects) == 0 {
// // 		return true
// // 	}
// // 	lower := strings.ToLower(topic)
// // 	for _, s := range allowedSubjects {
// // 		if strings.Contains(lower, strings.ToLower(s)) {
// // 			return true
// // 		}
// // 	}
// // 	return false
// // }