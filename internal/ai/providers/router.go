package providers

import (
	"context"
	"errors"
	"log"
)

// Router manages multiple AI providers
type Router struct {
	providers []Provider
}

func NewRouter() *Router {
	return &Router{
		providers: []Provider{},
	}
}

func (r *Router) Register(p Provider) {
	r.providers = append(r.providers, p)
}

func (r *Router) List() []string {
	names := []string{}
	for _, p := range r.providers {
		names = append(names, p.Name())
	}
	return names
}

func (r *Router) Generate(ctx context.Context, prompt string) (string, error) {
	// Try all providers
	for _, p := range r.providers {
		if p.Available(ctx) {
			log.Printf("🔍 Using provider: %s", p.Name())
			result, err := p.Generate(ctx, prompt)
			if err == nil {
				log.Printf("✅ Provider %s succeeded", p.Name())
				return result, nil
			}
			log.Printf("⚠️ Provider %s failed: %v", p.Name(), err)
		} else {
			log.Printf("⏭️ Provider %s is unavailable", p.Name())
		}
	}
	return "", errors.New("no provider available")
}


// package providers

// import (
// 	"context"
// 	"log"
// 	"sync"
// )

// // Router manages multiple providers with fallback.
// type Router struct {
// 	mu        sync.RWMutex
// 	providers []Client
// }

// // NewRouter creates a router with the given providers in priority order.
// func NewRouter(providers ...Client) *Router {
// 	return &Router{providers: providers}
// }

// // Register adds a provider to the end.
// func (r *Router) Register(p Client) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	r.providers = append(r.providers, p)
// }

// // Generate tries providers in order until one succeeds.
// func (r *Router) Generate(ctx context.Context, prompt string) (string, error) {
// 	r.mu.RLock()
// 	defer r.mu.RUnlock()
// 	var lastErr error
// 	for _, p := range r.providers {
// 		if !p.Available(ctx) {
// 			log.Printf("Provider %s is unavailable, skipping", p.Name())
// 			continue
// 		}
// 		resp, err := p.Generate(ctx, prompt)
// 		if err == nil {
// 			log.Printf("Provider %s succeeded", p.Name())
// 			return resp, nil
// 		}
// 		lastErr = err
// 		log.Printf("Provider %s failed: %v", p.Name(), err)
// 	}
// 	if lastErr == nil {
// 		lastErr = context.DeadlineExceeded
// 	}
// 	return "", lastErr
// }

// // List returns all registered provider names.
// func (r *Router) List() []string {
// 	r.mu.RLock()
// 	defer r.mu.RUnlock()
// 	names := make([]string, len(r.providers))
// 	for i, p := range r.providers {
// 		names[i] = p.Name()
// 	}
// 	return names
// }


// // package providers

// // import (
// // 	"context"
// // 	"log"
// // 	"sync"
// // )

// // // Router manages multiple providers with fallback.
// // type Router struct {
// // 	mu        sync.RWMutex
// // 	providers []Client
// // }

// // // NewRouter creates a router with the given providers in priority order.
// // func NewRouter(providers ...Client) *Router {
// // 	return &Router{providers: providers}
// // }

// // // Register adds a provider to the end.
// // func (r *Router) Register(p Client) {
// // 	r.mu.Lock()
// // 	defer r.mu.Unlock()
// // 	r.providers = append(r.providers, p)
// // }

// // // Generate tries providers in order until one succeeds.
// // func (r *Router) Generate(ctx context.Context, prompt string) (string, error) {
// // 	r.mu.RLock()
// // 	defer r.mu.RUnlock()
// // 	var lastErr error
// // 	for _, p := range r.providers {
// // 		if !p.Available(ctx) {
// // 			log.Printf("Provider %s is unavailable, skipping", p.Name())
// // 			continue
// // 		}
// // 		resp, err := p.Generate(ctx, prompt)
// // 		if err == nil {
// // 			log.Printf("Provider %s succeeded", p.Name())
// // 			return resp, nil
// // 		}
// // 		lastErr = err
// // 		log.Printf("Provider %s failed: %v", p.Name(), err)
// // 	}
// // 	return "", lastErr
// // }

// // // List returns all registered provider names.
// // func (r *Router) List() []string {
// // 	r.mu.RLock()
// // 	defer r.mu.RUnlock()
// // 	names := make([]string, len(r.providers))
// // 	for i, p := range r.providers {
// // 		names[i] = p.Name()
// // 	}
// // 	return names
// // }



// // // package providers

// // // import (
// // // 	"context"
// // // 	"log"
// // // 	"sync"
// // // )

// // // // Router manages multiple providers with fallback.
// // // type Router struct {
// // // 	mu        sync.RWMutex
// // // 	providers []Client
// // // }

// // // // NewRouter creates a router with the given providers in priority order.
// // // func NewRouter(providers ...Client) *Router {
// // // 	return &Router{providers: providers}
// // // }

// // // // Register adds a provider to the end.
// // // func (r *Router) Register(p Client) {
// // // 	r.mu.Lock()
// // // 	defer r.mu.Unlock()
// // // 	r.providers = append(r.providers, p)
// // // }

// // // // Generate tries providers in order until one succeeds.
// // // func (r *Router) Generate(ctx context.Context, prompt string) (string, error) {
// // // 	r.mu.RLock()
// // // 	defer r.mu.RUnlock()
// // // 	var lastErr error
// // // 	for _, p := range r.providers {
// // // 		if !p.Available(ctx) {
// // // 			log.Printf("Provider %s is unavailable, skipping", p.Name())
// // // 			continue
// // // 		}
// // // 		resp, err := p.Generate(ctx, prompt)
// // // 		if err == nil {
// // // 			log.Printf("Provider %s succeeded", p.Name())
// // // 			return resp, nil
// // // 		}
// // // 		lastErr = err
// // // 		log.Printf("Provider %s failed: %v", p.Name(), err)
// // // 	}
// // // 	return "", lastErr
// // // }