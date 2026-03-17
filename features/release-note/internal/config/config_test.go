package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/release-note/internal/config"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()

	// Create credential.yaml
	secretsDir := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("failed to create secrets dir: %v", err)
	}
	credContent := `llm:
  providers:
    openai:
      api_key: "sk-test-key-123"
    google:
      api_key: "google-key-456"
`
	if err := os.WriteFile(filepath.Join(secretsDir, "credential.yaml"), []byte(credContent), 0o644); err != nil {
		t.Fatalf("failed to write credential.yaml: %v", err)
	}

	// Create config.yaml
	configContent := `credentials_path: "./secrets/credential.yaml"
llm:
  provider: "openai"
  model: "gpt-4.1"
`
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	cfg, apiKey, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "gpt-4.1" {
		t.Errorf("expected model 'gpt-4.1', got '%s'", cfg.LLM.Model)
	}
	if apiKey != "sk-test-key-123" {
		t.Errorf("expected api key 'sk-test-key-123', got '%s'", apiKey)
	}
}

func TestLoad_MissingCredentialsPath(t *testing.T) {
	dir := t.TempDir()
	configContent := `llm:
  provider: "openai"
  model: "gpt-4.1"
`
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	_, _, err := config.Load(configPath)
	if err == nil {
		t.Fatal("expected error for missing credentials_path, got nil")
	}
}

func TestLoad_ProviderNotInCredentials(t *testing.T) {
	dir := t.TempDir()

	secretsDir := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("failed to create secrets dir: %v", err)
	}
	credContent := `llm:
  providers:
    google:
      api_key: "google-key"
`
	if err := os.WriteFile(filepath.Join(secretsDir, "credential.yaml"), []byte(credContent), 0o644); err != nil {
		t.Fatalf("failed to write credential.yaml: %v", err)
	}

	configContent := `credentials_path: "./secrets/credential.yaml"
llm:
  provider: "openai"
  model: "gpt-4.1"
`
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	_, _, err := config.Load(configPath)
	if err == nil {
		t.Fatal("expected error for provider not in credentials, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	_, _, err := config.Load(configPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_EmptyAPIKey(t *testing.T) {
	dir := t.TempDir()

	secretsDir := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("failed to create secrets dir: %v", err)
	}
	credContent := `llm:
  providers:
    openai:
      api_key: ""
`
	if err := os.WriteFile(filepath.Join(secretsDir, "credential.yaml"), []byte(credContent), 0o644); err != nil {
		t.Fatalf("failed to write credential.yaml: %v", err)
	}

	configContent := `credentials_path: "./secrets/credential.yaml"
llm:
  provider: "openai"
  model: "gpt-4.1"
`
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	_, _, err := config.Load(configPath)
	if err == nil {
		t.Fatal("expected error for empty API key, got nil")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, _, err := config.Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}
