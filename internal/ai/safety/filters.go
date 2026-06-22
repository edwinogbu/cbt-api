package safety

import (
	"regexp"
	"strings"
)

// FilterBadWords replaces profanity with ***.
func FilterBadWords(text string) string {
	badWords := []string{"badword", "profane", "idiot", "stupid"}
	for _, word := range badWords {
		re := regexp.MustCompile(`(?i)\b` + word + `\b`)
		text = re.ReplaceAllString(text, "***")
	}
	return text
}

// SanitizeText removes excess whitespace and special characters.
func SanitizeText(text string) string {
	// Remove multiple spaces
	text = strings.Join(strings.Fields(text), " ")
	// Remove control characters
	re := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	text = re.ReplaceAllString(text, "")
	return text
}

// RemoveExcessWhitespace removes extra spaces, tabs, and newlines.
func RemoveExcessWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}


// package safety

// import (
// 	"regexp"
// 	"strings"
// )

// // FilterBadWords replaces profanity with ***.
// func FilterBadWords(text string) string {
// 	badWords := []string{"badword", "profane", "idiot", "stupid"}
// 	for _, word := range badWords {
// 		re := regexp.MustCompile(`(?i)\b` + word + `\b`)
// 		text = re.ReplaceAllString(text, "***")
// 	}
// 	return text
// }

// // SanitizeText removes excess whitespace and special characters.
// func SanitizeText(text string) string {
// 	// Remove multiple spaces
// 	text = strings.Join(strings.Fields(text), " ")
// 	// Remove control characters
// 	re := regexp.MustCompile(`[\x00-\x1F\x7F]`)
// 	text = re.ReplaceAllString(text, "")
// 	return text
// }


// // package safety

// // import (
// // 	"regexp"
// // 	"strings"
// // )

// // // FilterBadWords replaces profanity with ***.
// // func FilterBadWords(text string) string {
// // 	badWords := []string{"badword", "profane"}
// // 	for _, word := range badWords {
// // 		re := regexp.MustCompile(`(?i)\b` + word + `\b`)
// // 		text = re.ReplaceAllString(text, "***")
// // 	}
// // 	return text
// // }

// // // SanitizeText removes any unwanted characters.
// // func SanitizeText(text string) string {
// // 	// Remove excess whitespace
// // 	text = strings.Join(strings.Fields(text), " ")
// // 	return text
// // }