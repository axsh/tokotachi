package llm

import "context"

// Provider is the common interface for LLM access.
// Each provider (OpenAI, Google, Anthropic) implements this interface.
type Provider interface {
	// Summarize sends systemPrompt and userContent to the LLM
	// and returns the generated summary text.
	Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error)
}
