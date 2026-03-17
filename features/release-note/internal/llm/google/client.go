package google

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotImplemented is returned when this provider is not yet implemented.
var ErrNotImplemented = errors.New("provider not implemented")

// Client is a placeholder for Google Gemini provider.
// TODO: Implement Google Gemini API integration.
type Client struct {
	apiKey string
	model  string
}

// New creates a new Google Gemini client placeholder.
func New(apiKey string, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
	}
}

// Summarize is not yet implemented for Google Gemini.
func (c *Client) Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error) {
	return "", fmt.Errorf("google (Gemini): %w", ErrNotImplemented)
}
