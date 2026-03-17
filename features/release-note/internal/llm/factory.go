package llm

import (
	"fmt"

	"github.com/axsh/tokotachi/features/release-note/internal/llm/anthropic"
	"github.com/axsh/tokotachi/features/release-note/internal/llm/google"
	"github.com/axsh/tokotachi/features/release-note/internal/llm/openai"
)

// NewProvider creates a Provider instance for the given provider name.
// Supported: "openai". TODO: "google", "anthropic".
func NewProvider(providerName string, apiKey string, model string) (Provider, error) {
	switch providerName {
	case "openai":
		return openai.New(apiKey, model), nil
	case "google":
		_ = google.New(apiKey, model)
		return nil, fmt.Errorf("google (Gemini): %w", ErrNotImplemented)
	case "anthropic":
		_ = anthropic.New(apiKey, model)
		return nil, fmt.Errorf("anthropic: %w", ErrNotImplemented)
	default:
		return nil, fmt.Errorf("%s: %w", providerName, ErrUnknownProvider)
	}
}
