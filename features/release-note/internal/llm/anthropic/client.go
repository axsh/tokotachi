package anthropic

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotImplemented is returned when this provider is not yet implemented.
var ErrNotImplemented = errors.New("provider not implemented")

// Client is a placeholder for Anthropic provider.
// TODO: Implement Anthropic API integration.
type Client struct {
	apiKey string
	model  string
}

// New creates a new Anthropic client placeholder.
func New(apiKey string, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
	}
}

// Summarize is not yet implemented for Anthropic.
func (c *Client) Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error) {
	return "", fmt.Errorf("anthropic: %w", ErrNotImplemented)
}
