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

type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		model:  "claude-3-haiku-20240307",
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Available(ctx context.Context) bool {
	return p.apiKey != ""
}

func (p *AnthropicProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("ANTHROPIC_API_KEY not set")
	}
	body := map[string]interface{}{
		"model":         p.model,
		"messages":      []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":    1000,
		"temperature":   0.3,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("anthropic API error: " + resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	content, _ := result["content"].(string)
	if content == "" {
		return "", errors.New("empty content")
	}
	return content, nil
}


// package providers

// import (
// 	"context"
// 	"os"
// )

// // AnthropicProvider is a stub for Anthropic Claude.
// type AnthropicProvider struct {
// 	apiKey string
// }

// func NewAnthropicProvider() *AnthropicProvider {
// 	return &AnthropicProvider{
// 		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
// 	}
// }

// func (p *AnthropicProvider) Name() string { return "anthropic" }

// func (p *AnthropicProvider) Available(ctx context.Context) bool {
// 	return p.apiKey != ""
// }

// func (p *AnthropicProvider) Generate(ctx context.Context, prompt string) (string, error) {
// 	// TODO: Implement Anthropic API call
// 	return "", nil
// }