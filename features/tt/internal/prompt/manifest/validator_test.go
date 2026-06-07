package manifest

import (
	"testing"
)

func TestNewValidator(t *testing.T) {
	v, err := NewValidator("testdata/schemas")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}
	// Should have loaded policy, procedure, capability schemas
	if len(v.schemas) < 3 {
		t.Errorf("NewValidator() loaded %d schemas, want at least 3", len(v.schemas))
	}
}

func TestNewValidator_NonexistentDir(t *testing.T) {
	v, err := NewValidator("testdata/nonexistent")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	if len(v.schemas) != 0 {
		t.Errorf("NewValidator() loaded %d schemas from nonexistent dir, want 0", len(v.schemas))
	}
}

func TestValidateSchema_ValidPolicy(t *testing.T) {
	v, err := NewValidator("testdata/schemas")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	entity := &Entity{
		Kind:     "policy",
		ID:       "test",
		FilePath: "test.yaml",
		Raw: map[string]any{
			"apiVersion": "agent.meta/v1",
			"kind":       "policy",
			"id":         "test",
			"title":      "Test",
			"scope":      "project",
			"activation": map[string]any{"mode": "always"},
			"body":       "test body",
		},
	}

	errs := v.ValidateSchema(entity)
	if len(errs) > 0 {
		t.Errorf("ValidateSchema() got %d errors for valid policy: %v", len(errs), errs)
	}
}

func TestValidateSchema_InvalidPolicy(t *testing.T) {
	v, err := NewValidator("testdata/schemas")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	entity := &Entity{
		Kind:     "policy",
		ID:       "test",
		FilePath: "test.yaml",
		Raw: map[string]any{
			"apiVersion": "agent.meta/v1",
			"kind":       "policy",
			"id":         "test",
			"title":      "Test",
			// missing scope and activation and body
		},
	}

	errs := v.ValidateSchema(entity)
	if len(errs) == 0 {
		t.Error("ValidateSchema() expected errors for invalid policy")
	}
}

func TestValidateSchema_UnknownKind(t *testing.T) {
	v, err := NewValidator("testdata/schemas")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	entity := &Entity{
		Kind:     "unknown",
		ID:       "test",
		FilePath: "test.yaml",
		Raw:      map[string]any{"kind": "unknown"},
	}

	// unknown has no schema file, should return empty (skip with warning)
	errs := v.ValidateSchema(entity)
	if len(errs) != 0 {
		t.Errorf("ValidateSchema() expected 0 errors for missing schema kind, got %d", len(errs))
	}
}

func TestValidateIDUniqueness_NoDuplicates(t *testing.T) {
	entities := []*Entity{
		{ID: "policy-1", FilePath: "a.yaml"},
		{ID: "policy-2", FilePath: "b.yaml"},
	}
	memDocs := []*MemoryDoc{
		{ID: "arch-1", FilePath: "c.md"},
	}

	errs := ValidateIDUniqueness(entities, memDocs)
	if len(errs) > 0 {
		t.Errorf("ValidateIDUniqueness() got %d errors, want 0: %v", len(errs), errs)
	}
}

func TestValidateIDUniqueness_WithDuplicates(t *testing.T) {
	entities := []*Entity{
		{ID: "duplicate-id", FilePath: "a.yaml"},
		{ID: "duplicate-id", FilePath: "b.yaml"},
	}
	memDocs := []*MemoryDoc{}

	errs := ValidateIDUniqueness(entities, memDocs)
	if len(errs) == 0 {
		t.Error("ValidateIDUniqueness() expected errors for duplicate IDs")
	}
}

func TestValidateIDUniqueness_CrossEntityArch(t *testing.T) {
	entities := []*Entity{
		{ID: "shared-id", FilePath: "a.yaml"},
	}
	memDocs := []*MemoryDoc{
		{ID: "shared-id", FilePath: "b.md"},
	}

	errs := ValidateIDUniqueness(entities, memDocs)
	if len(errs) == 0 {
		t.Error("ValidateIDUniqueness() expected errors for entity-archDoc ID collision")
	}
}

func TestValidateReferences_ValidBodyFile(t *testing.T) {
	entities := []*Entity{
		{
			ID:       "test",
			FilePath: "testdata/valid/policies/test-policy.yaml",
			Raw:      map[string]any{},
		},
	}
	memDocs := []*MemoryDoc{}

	errs := ValidateReferences(entities, memDocs, ".")
	if len(errs) > 0 {
		t.Errorf("ValidateReferences() got %d errors, want 0: %v", len(errs), errs)
	}
}

func TestValidateSchema_FrontmatterRequired(t *testing.T) {
	v, err := NewValidator("testdata/schemas")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	// body is required under the new schema
	entity := &Entity{
		Kind:     "policy",
		ID:       "test",
		FilePath: "test.yaml",
		Raw: map[string]any{
			"apiVersion": "agent.meta/v1",
			"kind":       "policy",
			"id":         "test",
			"title":      "Test",
			"scope":      "project",
			"activation": map[string]any{"mode": "always"},
			// missing body
		},
	}

	errs := v.ValidateSchema(entity)
	if len(errs) == 0 {
		t.Error("ValidateSchema() expected errors for missing body")
	}
}

func TestValidateReferences_BadCapabilityRef(t *testing.T) {
	entities := []*Entity{
		{
			ID:       "proc-with-bad-cap",
			Kind:     "procedure",
			FilePath: "test.yaml",
			Raw:      map[string]any{"uses_capabilities": []any{"nonexistent-cap"}},
		},
	}
	memDocs := []*MemoryDoc{}

	errs := ValidateReferences(entities, memDocs, ".")
	if len(errs) == 0 {
		t.Error("ValidateReferences() expected errors for nonexistent capability reference")
	}
}

func TestValidateReferences_ValidCapabilityRef(t *testing.T) {
	entities := []*Entity{
		{
			ID:       "my-cap",
			Kind:     "capability",
			FilePath: "cap.yaml",
			Raw:      map[string]any{},
		},
		{
			ID:       "my-proc",
			Kind:     "procedure",
			FilePath: "proc.yaml",
			Raw:      map[string]any{"uses_capabilities": []any{"my-cap"}},
		},
	}
	memDocs := []*MemoryDoc{}

	errs := ValidateReferences(entities, memDocs, ".")
	if len(errs) > 0 {
		t.Errorf("ValidateReferences() got %d errors for valid cap ref: %v", len(errs), errs)
	}
}

func TestValidateReferences_BundleIncludes(t *testing.T) {
	entities := []*Entity{
		{
			ID:       "my-guard",
			Kind:     "guard",
			FilePath: "guard.yaml",
			Raw:      map[string]any{},
		},
		{
			ID:       "my-bundle",
			Kind:     "bundle",
			FilePath: "bundle.yaml",
			Raw:      map[string]any{"includes": []any{"my-guard"}},
		},
	}
	memDocs := []*MemoryDoc{}

	errs := ValidateReferences(entities, memDocs, ".")
	if len(errs) > 0 {
		t.Errorf("ValidateReferences() got %d errors for valid bundle includes: %v", len(errs), errs)
	}

	// Reference to non-existent entity
	entitiesWithBadRef := []*Entity{
		{
			ID:       "my-bundle",
			Kind:     "bundle",
			FilePath: "bundle.yaml",
			Raw:      map[string]any{"includes": []any{"nonexistent-guard"}},
		},
	}
	errsBad := ValidateReferences(entitiesWithBadRef, memDocs, ".")
	if len(errsBad) == 0 {
		t.Error("ValidateReferences() expected errors for nonexistent bundle includes")
	}
}

func TestValidateSchema_SafetyEntities(t *testing.T) {
	v, err := NewValidator("testdata/schemas")
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	guard := &Entity{
		Kind:     "guard",
		ID:       "my-guard",
		FilePath: "guard.yaml",
		Raw: map[string]any{
			"apiVersion":  "agent.meta/v1",
			"kind":        "guard",
			"id":          "my-guard",
			"title":       "My Guard",
			"description": "Test guard",
			"event":       "file_write",
			"action":      "deny",
			"paths":       []any{"*.go"},
		},
	}
	if errs := v.ValidateSchema(guard); len(errs) > 0 {
		t.Errorf("ValidateSchema() got errors for valid guard: %v", errs)
	}

	worker := &Entity{
		Kind:     "worker",
		ID:       "my-worker",
		FilePath: "worker.yaml",
		Raw: map[string]any{
			"apiVersion":  "agent.meta/v1",
			"kind":        "worker",
			"id":          "my-worker",
			"title":       "My Worker",
			"description": "Test worker",
			"capability":  "my-cap",
		},
	}
	if errs := v.ValidateSchema(worker); len(errs) > 0 {
		t.Errorf("ValidateSchema() got errors for valid worker: %v", errs)
	}

	bundle := &Entity{
		Kind:     "bundle",
		ID:       "my-bundle",
		FilePath: "bundle.yaml",
		Raw: map[string]any{
			"apiVersion":  "agent.meta/v1",
			"kind":        "bundle",
			"id":          "my-bundle",
			"title":       "My Bundle",
			"description": "Test bundle",
			"includes":    []any{"my-guard", "my-worker"},
		},
	}
	if errs := v.ValidateSchema(bundle); len(errs) > 0 {
		t.Errorf("ValidateSchema() got errors for valid bundle: %v", errs)
	}
}

