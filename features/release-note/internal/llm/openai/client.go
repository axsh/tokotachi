package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultBaseURL = "https://api.openai.com"

// Client implements llm.Provider for OpenAI Chat Completions API.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// New creates a new OpenAI client with default settings.
func New(apiKey string, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: http.DefaultClient,
		baseURL:    defaultBaseURL,
	}
}

// NewWithHTTPClient creates a new OpenAI client with a custom HTTP client
// and base URL. Useful for testing with mock servers.
func NewWithHTTPClient(apiKey string, model string, httpClient *http.Client, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// chatRequest represents the request body for Chat Completions API.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage represents a single message in the chat.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse represents the response from Chat Completions API.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

// chatChoice represents a single choice in the response.
type chatChoice struct {
	Message chatMessage `json:"message"`
}

// apiError represents an error from the OpenAI API.
type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Summarize sends a chat completion request to OpenAI and returns
// the generated text from the first choice.
func (c *Client) Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp chatResponse
		if json.Unmarshal(respBytes, &errResp) == nil && errResp.Error != nil {
			return "", fmt.Errorf("OpenAI API error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("OpenAI API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
