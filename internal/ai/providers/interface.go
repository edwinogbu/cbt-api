package providers

import "context"

// Provider defines the contract for AI providers
type Provider interface {
	// Name returns the provider name (openai, anthropic, gemini, fallback)
	Name() string
	// Available checks if the provider is available (API key set, etc.)
	Available(ctx context.Context) bool
	// Generate generates content from a prompt
	Generate(ctx context.Context, prompt string) (string, error)
}

// package providers

// import "context"

// // Client is the interface each AI provider must implement.
// type Client interface {
// 	// Generate sends a prompt and returns the response.
// 	Generate(ctx context.Context, prompt string) (string, error)
// 	// Name returns the provider's identifier.
// 	Name() string
// 	// Available checks if the provider is ready (key set, healthy).
// 	Available(ctx context.Context) bool
// }


// package providers

// import "context"

// // Client is the interface each AI provider must implement.
// type Client interface {
// 	// Generate sends a prompt and returns the response.
// 	Generate(ctx context.Context, prompt string) (string, error)
// 	// Name returns the provider's identifier.
// 	Name() string
// 	// Available checks if the provider is ready (key set, healthy).
// 	Available(ctx context.Context) bool
// }



// // package providers

// // import "context"

// // // Client is the interface each AI provider must implement.
// // type Client interface {
// // 	Generate(ctx context.Context, prompt string) (string, error)
// // 	Name() string
// // 	Available(ctx context.Context) bool
// // }