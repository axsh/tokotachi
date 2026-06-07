package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator validates a NotifyPayload against the JSON schema.
type Validator struct {
	schema *jsonschema.Schema
}

// NewValidator creates a new Validator.
// schemasDir is the absolute or relative path to the directory containing
// agent-notify-payload.schema.json.
func NewValidator(schemasDir string) (*Validator, error) {
	schemaPath := filepath.Join(schemasDir, "agent-notify-payload.schema.json")

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	c := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	if err := c.AddResource("agent-notify-payload.schema.json", doc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := c.Compile("agent-notify-payload.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return &Validator{schema: schema}, nil
}

// Validate validates the raw JSON bytes against the schema.
// Returns nil if valid, or an error with details of violations.
func (v *Validator) Validate(data []byte) error {
	var inst any
	if err := json.Unmarshal(data, &inst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return v.schema.Validate(inst)
}
