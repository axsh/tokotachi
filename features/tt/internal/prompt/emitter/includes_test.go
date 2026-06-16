package emitter

import (
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

func TestExtractIncludes_NilTarget(t *testing.T) {
	inc := ExtractIncludes(nil)
	if !inc.Policy || !inc.Capability || !inc.Procedure || inc.Subagent {
		t.Errorf("expected defaults (Policy=true, Capability=true, Procedure=true, Subagent=false), got %+v", inc)
	}
}

func TestExtractIncludes_NewIncludesKey(t *testing.T) {
	target := &manifest.Entity{
		ID: "test-target",
		Raw: map[string]any{
			"includes": map[string]any{
				"policy":     true,
				"capability": true,
				"procedure":  false,
				"subagent":   false,
			},
		},
	}
	inc := ExtractIncludes(target)
	if !inc.Policy {
		t.Error("expected Policy=true")
	}
	if !inc.Capability {
		t.Error("expected Capability=true")
	}
	if inc.Procedure {
		t.Error("expected Procedure=false")
	}
	if inc.Subagent {
		t.Error("expected Subagent=false")
	}
}

func TestExtractIncludes_LegacyCapabilitiesKey(t *testing.T) {
	target := &manifest.Entity{
		ID: "test-target",
		Raw: map[string]any{
			"capabilities": map[string]any{
				"rules":     true,
				"skills":    false,
				"workflows": true,
				"subagents": false,
			},
		},
	}
	inc := ExtractIncludes(target)
	if !inc.Policy {
		t.Error("expected Policy=true (mapped from rules)")
	}
	if inc.Capability {
		t.Error("expected Capability=false (mapped from skills)")
	}
	if !inc.Procedure {
		t.Error("expected Procedure=true (mapped from workflows)")
	}
	if inc.Subagent {
		t.Error("expected Subagent=false (mapped from subagents)")
	}
}

func TestExtractIncludes_PartialFlags(t *testing.T) {
	target := &manifest.Entity{
		ID: "test-target",
		Raw: map[string]any{
			"includes": map[string]any{
				"procedure": false,
			},
		},
	}
	inc := ExtractIncludes(target)
	if !inc.Policy {
		t.Error("expected Policy=true (default)")
	}
	if !inc.Capability {
		t.Error("expected Capability=true (default)")
	}
	if inc.Procedure {
		t.Error("expected Procedure=false")
	}
	if inc.Subagent {
		t.Error("expected Subagent=false (default)")
	}
}

func TestExtractIncludes_NoIncludesOrCapabilities(t *testing.T) {
	target := &manifest.Entity{
		ID:  "test-target",
		Raw: map[string]any{},
	}
	inc := ExtractIncludes(target)
	if !inc.Policy || !inc.Capability || !inc.Procedure || inc.Subagent {
		t.Errorf("expected defaults, got %+v", inc)
	}
}

func TestExtractIncludes_IncludesTakesPrecedenceOverCapabilities(t *testing.T) {
	target := &manifest.Entity{
		ID: "test-target",
		Raw: map[string]any{
			"includes": map[string]any{
				"procedure": true,
			},
			"capabilities": map[string]any{
				"workflows": false,
			},
		},
	}
	inc := ExtractIncludes(target)
	if !inc.Procedure {
		t.Error("expected Procedure=true (includes takes precedence over capabilities)")
	}
}
