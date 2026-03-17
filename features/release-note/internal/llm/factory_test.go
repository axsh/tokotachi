package llm_test

import (
	"errors"
	"testing"

	"github.com/axsh/tokotachi/features/release-note/internal/llm"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		wantErr      error
	}{
		{
			name:         "openai returns no error",
			providerName: "openai",
			wantErr:      nil,
		},
		{
			name:         "google returns ErrNotImplemented",
			providerName: "google",
			wantErr:      llm.ErrNotImplemented,
		},
		{
			name:         "anthropic returns ErrNotImplemented",
			providerName: "anthropic",
			wantErr:      llm.ErrNotImplemented,
		},
		{
			name:         "unknown returns ErrUnknownProvider",
			providerName: "unknown-provider",
			wantErr:      llm.ErrUnknownProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := llm.NewProvider(tt.providerName, "test-api-key", "test-model")

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if provider != nil {
					t.Fatal("expected nil provider on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if provider == nil {
					t.Fatal("expected non-nil provider")
				}
			}
		})
	}
}
