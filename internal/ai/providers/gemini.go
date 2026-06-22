package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"
)

type GeminiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiProvider() *GeminiProvider {
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-pro"
	}
	return &GeminiProvider{
		apiKey: os.Getenv("GEMINI_API_KEY"),
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) Available(ctx context.Context) bool {
	return p.apiKey != "" && len(p.apiKey) > 10
}

func (p *GeminiProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("GEMINI_API_KEY not set")
	}
	
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.3,
			"maxOutputTokens": 2000,
		},
	}
	
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	
	url := "https://generativelanguage.googleapis.com/v1/models/" + p.model + ":generateContent?key=" + p.apiKey
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("gemini API error: " + resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", errors.New("no candidates")
	}
	
	candidate := candidates[0].(map[string]interface{})
	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return "", errors.New("no content")
	}
	
	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", errors.New("empty parts")
	}
	
	part := parts[0].(map[string]interface{})
	text, ok := part["text"].(string)
	if !ok || text == "" {
		return "", errors.New("empty text")
	}
	
	return text, nil
}



// package providers

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"net/http"
// 	"os"
// 	"time"
// )

// type GeminiProvider struct {
// 	apiKey string
// 	client *http.Client
// }

// func NewGeminiProvider() *GeminiProvider {
// 	return &GeminiProvider{
// 		apiKey: os.Getenv("GEMINI_API_KEY"),
// 		client: &http.Client{Timeout: 60 * time.Second},
// 	}
// }

// func (p *GeminiProvider) Name() string { return "gemini" }

// func (p *GeminiProvider) Available(ctx context.Context) bool {
// 	return p.apiKey != ""
// }

// func (p *GeminiProvider) Generate(ctx context.Context, prompt string) (string, error) {
// 	if p.apiKey == "" {
// 		return "", errors.New("GEMINI_API_KEY not set")
// 	}
// 	body := map[string]interface{}{
// 		"contents": []map[string]interface{}{
// 			{
// 				"parts": []map[string]string{{"text": prompt}},
// 			},
// 		},
// 		"generationConfig": map[string]interface{}{
// 			"temperature": 0.3,
// 		},
// 	}
// 	b, err := json.Marshal(body)
// 	if err != nil {
// 		return "", err
// 	}
// 	url := "https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent?key=" + p.apiKey
// 	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(b))
// 	if err != nil {
// 		return "", err
// 	}
// 	req.Header.Set("Content-Type", "application/json")

// 	resp, err := p.client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return "", errors.New("gemini API error: " + resp.Status)
// 	}

// 	var result map[string]interface{}
// 	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// 		return "", err
// 	}
// 	candidates, _ := result["candidates"].([]interface{})
// 	if len(candidates) == 0 {
// 		return "", errors.New("no candidates")
// 	}
// 	candidate := candidates[0].(map[string]interface{})
// 	content, _ := candidate["content"].(map[string]interface{})
// 	parts, _ := content["parts"].([]interface{})
// 	if len(parts) == 0 {
// 		return "", errors.New("empty parts")
// 	}
// 	part := parts[0].(map[string]interface{})
// 	text, _ := part["text"].(string)
// 	if text == "" {
// 		return "", errors.New("empty text")
// 	}
// 	return text, nil
// }



// // package providers

// // import (
// // 	"bytes"
// // 	"context"
// // 	"encoding/json"
// // 	"errors"
// // 	"net/http"
// // 	"os"
// // 	"time"
// // )

// // type GeminiProvider struct {
// // 	apiKey string
// // 	client *http.Client
// // }

// // func NewGeminiProvider() *GeminiProvider {
// // 	return &GeminiProvider{
// // 		apiKey: os.Getenv("GEMINI_API_KEY"),
// // 		client: &http.Client{Timeout: 60 * time.Second},
// // 	}
// // }

// // func (p *GeminiProvider) Name() string { return "gemini" }

// // func (p *GeminiProvider) Available(ctx context.Context) bool {
// // 	return p.apiKey != ""
// // }

// // func (p *GeminiProvider) Generate(ctx context.Context, prompt string) (string, error) {
// // 	if p.apiKey == "" {
// // 		return "", errors.New("GEMINI_API_KEY not set")
// // 	}
// // 	body := map[string]interface{}{
// // 		"contents": []map[string]interface{}{
// // 			{
// // 				"parts": []map[string]string{{"text": prompt}},
// // 			},
// // 		},
// // 		"generationConfig": map[string]interface{}{
// // 			"temperature": 0.3,
// // 		},
// // 	}
// // 	b, err := json.Marshal(body)
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	url := "https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent?key=" + p.apiKey
// // 	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(b))
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	req.Header.Set("Content-Type", "application/json")

// // 	resp, err := p.client.Do(req)
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	defer resp.Body.Close()

// // 	if resp.StatusCode != http.StatusOK {
// // 		return "", errors.New("gemini API error: " + resp.Status)
// // 	}

// // 	var result map[string]interface{}
// // 	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// // 		return "", err
// // 	}
// // 	candidates, _ := result["candidates"].([]interface{})
// // 	if len(candidates) == 0 {
// // 		return "", errors.New("no candidates")
// // 	}
// // 	candidate := candidates[0].(map[string]interface{})
// // 	content, _ := candidate["content"].(map[string]interface{})
// // 	parts, _ := content["parts"].([]interface{})
// // 	if len(parts) == 0 {
// // 		return "", errors.New("empty parts")
// // 	}
// // 	part := parts[0].(map[string]interface{})
// // 	text, _ := part["text"].(string)
// // 	if text == "" {
// // 		return "", errors.New("empty text")
// // 	}
// // 	return text, nil
// // }



// // // package providers

// // // import (
// // // 	"context"
// // // 	"os"
// // // )

// // // // GeminiProvider is a stub for Google Gemini.
// // // type GeminiProvider struct {
// // // 	apiKey string
// // // }

// // // func NewGeminiProvider() *GeminiProvider {
// // // 	return &GeminiProvider{
// // // 		apiKey: os.Getenv("GEMINI_API_KEY"),
// // // 	}
// // // }

// // // func (p *GeminiProvider) Name() string { return "gemini" }

// // // func (p *GeminiProvider) Available(ctx context.Context) bool {
// // // 	return p.apiKey != ""
// // // }

// // // func (p *GeminiProvider) Generate(ctx context.Context, prompt string) (string, error) {
// // // 	// TODO: Implement Gemini API call
// // // 	return "", nil
// // // }