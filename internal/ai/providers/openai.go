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

type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		model:  "gpt-4o-mini",
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Available(ctx context.Context) bool {
	return p.apiKey != ""
}

func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("OPENAI_API_KEY not set")
	}
	body := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("openai API error: " + resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", errors.New("no choices")
	}
	choice := choices[0].(map[string]interface{})
	msg, _ := choice["message"].(map[string]interface{})
	content, _ := msg["content"].(string)
	if content == "" {
		return "", errors.New("empty content")
	}
	return content, nil
}



// package providers

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"net/http"
// 	"os"
// )

// type OpenAIProvider struct {
// 	apiKey string
// 	model  string
// }

// func NewOpenAIProvider() *OpenAIProvider {
// 	return &OpenAIProvider{
// 		apiKey: os.Getenv("OPENAI_API_KEY"),
// 		model:  "gpt-4o-mini",
// 	}
// }

// func (p *OpenAIProvider) Name() string { return "openai" }

// func (p *OpenAIProvider) Available(ctx context.Context) bool {
// 	return p.apiKey != ""
// }

// func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
// 	if p.apiKey == "" {
// 		return "", errors.New("OPENAI_API_KEY not set")
// 	}
// 	body := map[string]interface{}{
// 		"model": p.model,
// 		"messages": []map[string]string{
// 			{"role": "user", "content": prompt},
// 		},
// 		"temperature": 0.3,
// 	}
// 	b, err := json.Marshal(body)
// 	if err != nil {
// 		return "", err
// 	}
// 	req, err := http.NewRequestWithContext(ctx, "POST",
// 		"https://api.openai.com/v1/chat/completions",
// 		bytes.NewBuffer(b),
// 	)
// 	if err != nil {
// 		return "", err
// 	}
// 	req.Header.Set("Authorization", "Bearer "+p.apiKey)
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	var result map[string]interface{}
// 	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// 		return "", err
// 	}
// 	choices, ok := result["choices"].([]interface{})
// 	if !ok || len(choices) == 0 {
// 		return "", errors.New("no choices")
// 	}
// 	choice := choices[0].(map[string]interface{})
// 	msg, _ := choice["message"].(map[string]interface{})
// 	content, _ := msg["content"].(string)
// 	if content == "" {
// 		return "", errors.New("empty content")
// 	}
// 	return content, nil
// }


// // package providers

// // import (
// // 	"bytes"
// // 	"context"
// // 	"encoding/json"
// // 	"errors"
// // 	"net/http"
// // 	"os"
// // )

// // type OpenAIProvider struct {
// // 	apiKey string
// // 	model  string
// // }

// // func NewOpenAIProvider() *OpenAIProvider {
// // 	return &OpenAIProvider{
// // 		apiKey: os.Getenv("OPENAI_API_KEY"),
// // 		model:  "gpt-4o-mini",
// // 	}
// // }

// // func (p *OpenAIProvider) Name() string { return "openai" }

// // func (p *OpenAIProvider) Available(ctx context.Context) bool {
// // 	return p.apiKey != ""
// // }

// // func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
// // 	if p.apiKey == "" {
// // 		return "", errors.New("OPENAI_API_KEY not set")
// // 	}
// // 	body := map[string]interface{}{
// // 		"model": p.model,
// // 		"messages": []map[string]string{
// // 			{"role": "user", "content": prompt},
// // 		},
// // 		"temperature": 0.3,
// // 	}
// // 	b, err := json.Marshal(body)
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	req, err := http.NewRequestWithContext(ctx, "POST",
// // 		"https://api.openai.com/v1/chat/completions",
// // 		bytes.NewBuffer(b),
// // 	)
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	req.Header.Set("Authorization", "Bearer "+p.apiKey)
// // 	req.Header.Set("Content-Type", "application/json")

// // 	client := &http.Client{}
// // 	resp, err := client.Do(req)
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	defer resp.Body.Close()

// // 	var result map[string]interface{}
// // 	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// // 		return "", err
// // 	}
// // 	choices, ok := result["choices"].([]interface{})
// // 	if !ok || len(choices) == 0 {
// // 		return "", errors.New("no choices")
// // 	}
// // 	choice := choices[0].(map[string]interface{})
// // 	msg, _ := choice["message"].(map[string]interface{})
// // 	content, _ := msg["content"].(string)
// // 	if content == "" {
// // 		return "", errors.New("empty content")
// // 	}
// // 	return content, nil
// // }