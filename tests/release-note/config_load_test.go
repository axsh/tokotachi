package release_note_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigLoad_RealProjectFiles(t *testing.T) {
	configPath := filepath.Join(projectRoot(), "features", "release-note", "settings", "config.yaml")

	// Verify config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config.yaml not found at %s", configPath)
	}

	// Parse and validate structure
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.yaml: %v", err)
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse config.yaml: %v", err)
	}

	// Verify credentials_path exists and is non-empty
	credPath, ok := config["credentials_path"]
	if !ok {
		t.Fatal("config.yaml missing 'credentials_path' field")
	}
	if credPath == "" {
		t.Fatal("config.yaml 'credentials_path' is empty")
	}

	// Verify llm section exists
	llmSection, ok := config["llm"]
	if !ok {
		t.Fatal("config.yaml missing 'llm' section")
	}

	llmMap, ok := llmSection.(map[string]any)
	if !ok {
		t.Fatal("config.yaml 'llm' section is not a map")
	}

	// Verify provider is "openai"
	provider, ok := llmMap["provider"]
	if !ok {
		t.Fatal("config.yaml missing 'llm.provider'")
	}
	if provider != "openai" {
		t.Errorf("expected llm.provider 'openai', got '%v'", provider)
	}

	// Verify model is non-empty
	model, ok := llmMap["model"]
	if !ok {
		t.Fatal("config.yaml missing 'llm.model'")
	}
	if model == "" {
		t.Fatal("config.yaml 'llm.model' is empty")
	}
}

func TestConfigLoad_CredentialFileExists(t *testing.T) {
	configPath := filepath.Join(projectRoot(), "features", "release-note", "settings", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.yaml: %v", err)
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse config.yaml: %v", err)
	}

	credRelPath, ok := config["credentials_path"].(string)
	if !ok || credRelPath == "" {
		t.Fatal("credentials_path not found or empty in config.yaml")
	}

	// Resolve credential path relative to config directory
	settingsDir := filepath.Dir(configPath)
	credAbsPath := filepath.Join(settingsDir, credRelPath)

	// Verify credential file exists
	if _, err := os.Stat(credAbsPath); os.IsNotExist(err) {
		t.Fatalf("credential file not found at %s", credAbsPath)
	}

	// Parse and validate structure
	credData, err := os.ReadFile(credAbsPath)
	if err != nil {
		t.Fatalf("failed to read credential file: %v", err)
	}

	var creds map[string]any
	if err := yaml.Unmarshal(credData, &creds); err != nil {
		t.Fatalf("failed to parse credential file: %v", err)
	}

	// Verify llm.providers section exists with at least one entry
	llmSection, ok := creds["llm"]
	if !ok {
		t.Fatal("credential file missing 'llm' section")
	}

	llmMap, ok := llmSection.(map[string]any)
	if !ok {
		t.Fatal("credential file 'llm' section is not a map")
	}

	providers, ok := llmMap["providers"]
	if !ok {
		t.Fatal("credential file missing 'llm.providers'")
	}

	providersMap, ok := providers.(map[string]any)
	if !ok {
		t.Fatal("credential file 'llm.providers' is not a map")
	}

	if len(providersMap) == 0 {
		t.Fatal("credential file 'llm.providers' has no entries")
	}
}
