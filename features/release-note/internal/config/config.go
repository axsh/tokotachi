package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration from config.yaml.
type Config struct {
	CredentialsPath string    `yaml:"credentials_path"`
	LLM             LLMConfig `yaml:"llm"`
}

// LLMConfig holds LLM provider and model settings.
type LLMConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

// Credentials represents the secrets from credential.yaml.
type Credentials struct {
	LLM LLMCredentials `yaml:"llm"`
}

// LLMCredentials holds provider-specific credentials.
type LLMCredentials struct {
	Providers map[string]ProviderCredential `yaml:"providers"`
}

// ProviderCredential holds the API key for a single provider.
type ProviderCredential struct {
	APIKey string `yaml:"api_key"`
}

// Load reads config.yaml from the given path and also loads
// the referenced credential.yaml. Returns the parsed Config
// and the resolved API key for the configured provider.
func Load(configPath string) (*Config, string, error) {
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.CredentialsPath == "" {
		return nil, "", fmt.Errorf("credentials_path is not set in config")
	}

	if cfg.LLM.Provider == "" {
		return nil, "", fmt.Errorf("llm.provider is not set in config")
	}

	// Resolve credential path relative to config file directory
	configDir := filepath.Dir(configPath)
	credPath := filepath.Join(configDir, cfg.CredentialsPath)

	// Read credentials file
	credData, err := os.ReadFile(credPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read credentials file: %w", err)
	}

	var creds Credentials
	if err := yaml.Unmarshal(credData, &creds); err != nil {
		return nil, "", fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// Look up the API key for the configured provider
	providerCred, ok := creds.LLM.Providers[cfg.LLM.Provider]
	if !ok {
		return nil, "", fmt.Errorf("provider %q not found in credentials", cfg.LLM.Provider)
	}

	if providerCred.APIKey == "" {
		return nil, "", fmt.Errorf("api_key is empty for provider %q", cfg.LLM.Provider)
	}

	return &cfg, providerCred.APIKey, nil
}
