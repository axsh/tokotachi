package converter

import (
	"testing"

	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
)

func TestExtractTemplateVars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "extracts module_path and feature_name",
			input: "{{module_path}}/{{feature_name}}",
			want:  []string{"feature_name", "module_path"},
		},
		{
			name:  "extracts single var from base_dir",
			input: "features/{{feature_name}}",
			want:  []string{"feature_name"},
		},
		{
			name:  "no template vars",
			input: "no-template-vars",
			want:  nil,
		},
		{
			name:  "deduplicates repeated vars",
			input: "{{a}}/{{b}}/{{a}}",
			want:  []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTemplateVars(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("ExtractTemplateVars(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractTemplateVars(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParamCollector(t *testing.T) {
	t.Run("collects unique params and returns sorted", func(t *testing.T) {
		pc := NewParamCollector()
		pc.Add("feature_name")
		pc.Add("module_path")
		pc.Add("feature_name") // duplicate

		names := pc.Names()
		want := []string{"feature_name", "module_path"}

		if len(names) != len(want) {
			t.Fatalf("Names() = %v, want %v", names, want)
		}
		for i := range names {
			if names[i] != want[i] {
				t.Errorf("Names()[%d] = %q, want %q", i, names[i], want[i])
			}
		}
	})

	t.Run("empty collector returns nil", func(t *testing.T) {
		pc := NewParamCollector()
		names := pc.Names()
		if len(names) != 0 {
			t.Errorf("Names() = %v, want empty", names)
		}
	})
}

func TestMergeParams(t *testing.T) {
	tests := []struct {
		name       string
		defined    []catalog.TemplateParam
		discovered []string
		wantNames  []string
		wantLen    int
	}{
		{
			name: "exact match keeps scaffold.yaml definitions",
			defined: []catalog.TemplateParam{
				{Name: "feature_name", Description: "Feature name", Required: true, Default: "myprog"},
				{Name: "module_path", Description: "Go module path", Required: true, Default: "github.com/org/app"},
			},
			discovered: []string{"feature_name", "module_path"},
			wantNames:  []string{"feature_name", "module_path"},
			wantLen:    2,
		},
		{
			name: "discovered but not defined adds auto param",
			defined: []catalog.TemplateParam{
				{Name: "module_path", Description: "Go module path", Required: true},
			},
			discovered: []string{"feature_name", "module_path"},
			wantNames:  []string{"feature_name", "module_path"},
			wantLen:    2,
		},
		{
			name: "defined but not discovered keeps param",
			defined: []catalog.TemplateParam{
				{Name: "feature_name", Description: "Feature name", Required: true, Default: "myprog"},
				{Name: "module_path", Description: "Go module path", Required: true},
				{Name: "extra_param", Description: "Extra", Required: false},
			},
			discovered: []string{"feature_name", "module_path"},
			wantNames:  []string{"extra_param", "feature_name", "module_path"},
			wantLen:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeParams(tt.defined, tt.discovered)
			if len(got) != tt.wantLen {
				t.Fatalf("MergeParams() returned %d params, want %d: %+v", len(got), tt.wantLen, got)
			}
			for i, wantName := range tt.wantNames {
				if got[i].Name != wantName {
					t.Errorf("MergeParams()[%d].Name = %q, want %q", i, got[i].Name, wantName)
				}
			}
		})
	}

	t.Run("auto-added param has required=true", func(t *testing.T) {
		defined := []catalog.TemplateParam{
			{Name: "module_path", Description: "Go module path", Required: true},
		}
		got := MergeParams(defined, []string{"feature_name", "module_path"})

		// feature_name should be auto-added with Required=true
		for _, p := range got {
			if p.Name == "feature_name" {
				if !p.Required {
					t.Errorf("auto-added param 'feature_name' should have Required=true")
				}
				if p.Description != "" {
					t.Errorf("auto-added param 'feature_name' should have empty Description, got %q", p.Description)
				}
				return
			}
		}
		t.Errorf("'feature_name' not found in merged params")
	})

	t.Run("preserves description and default from defined", func(t *testing.T) {
		defined := []catalog.TemplateParam{
			{Name: "module_path", Description: "Go module path", Required: true, Default: "github.com/org/app"},
		}
		got := MergeParams(defined, []string{"module_path"})

		if got[0].Description != "Go module path" {
			t.Errorf("Description = %q, want %q", got[0].Description, "Go module path")
		}
		if got[0].Default != "github.com/org/app" {
			t.Errorf("Default = %q, want %q", got[0].Default, "github.com/org/app")
		}
	})

	t.Run("auto-added param has default ValueSpec", func(t *testing.T) {
		maxBytes := 512
		defined := []catalog.TemplateParam{
			{
				Name:        "module_path",
				Description: "Go module path",
				Required:    true,
				ValueSpec: &catalog.ValueSpec{
					Type: "string",
					Length: &catalog.LengthSpec{
						MaxBytes: &maxBytes,
					},
				},
			},
		}
		got := MergeParams(defined, []string{"feature_name", "module_path"})

		// feature_name should be auto-added with default ValueSpec.
		var featureParam *catalog.TemplateParam
		var moduleParam *catalog.TemplateParam
		for i := range got {
			switch got[i].Name {
			case "feature_name":
				featureParam = &got[i]
			case "module_path":
				moduleParam = &got[i]
			}
		}

		if featureParam == nil {
			t.Fatalf("'feature_name' not found in merged params")
		}
		if featureParam.ValueSpec == nil {
			t.Fatalf("auto-added param 'feature_name' should have ValueSpec, got nil")
		}
		if featureParam.ValueSpec.Type != "string" {
			t.Errorf("auto-added ValueSpec.Type = %q, want %q", featureParam.ValueSpec.Type, "string")
		}
		if featureParam.ValueSpec.Length == nil || featureParam.ValueSpec.Length.MaxBytes == nil {
			t.Fatalf("auto-added ValueSpec.Length.MaxBytes should not be nil")
		}
		if *featureParam.ValueSpec.Length.MaxBytes != 256 {
			t.Errorf("auto-added ValueSpec.Length.MaxBytes = %d, want 256", *featureParam.ValueSpec.Length.MaxBytes)
		}

		// module_path should preserve its original ValueSpec.
		if moduleParam == nil {
			t.Fatalf("'module_path' not found in merged params")
		}
		if moduleParam.ValueSpec == nil {
			t.Fatalf("'module_path' should preserve its ValueSpec, got nil")
		}
		if moduleParam.ValueSpec.Length == nil || moduleParam.ValueSpec.Length.MaxBytes == nil {
			t.Fatalf("'module_path' ValueSpec.Length.MaxBytes should not be nil")
		}
		if *moduleParam.ValueSpec.Length.MaxBytes != 512 {
			t.Errorf("'module_path' ValueSpec.Length.MaxBytes = %d, want 512 (should be preserved)", *moduleParam.ValueSpec.Length.MaxBytes)
		}
	})

	t.Run("preserves existing ValueSpec from defined", func(t *testing.T) {
		maxBytes := 512
		defined := []catalog.TemplateParam{
			{
				Name:     "module_path",
				Required: true,
				ValueSpec: &catalog.ValueSpec{
					Type: "string",
					Length: &catalog.LengthSpec{
						MaxBytes: &maxBytes,
					},
					Format: &catalog.FormatSpec{
						Pattern: "^[a-zA-Z0-9._/-]+$",
					},
				},
			},
		}
		got := MergeParams(defined, []string{"module_path"})

		if len(got) != 1 {
			t.Fatalf("MergeParams() returned %d params, want 1", len(got))
		}
		vs := got[0].ValueSpec
		if vs == nil {
			t.Fatalf("ValueSpec should be preserved, got nil")
		}
		if vs.Length == nil || vs.Length.MaxBytes == nil || *vs.Length.MaxBytes != 512 {
			t.Errorf("ValueSpec.Length.MaxBytes should be 512")
		}
		if vs.Format == nil || vs.Format.Pattern != "^[a-zA-Z0-9._/-]+$" {
			t.Errorf("ValueSpec.Format.Pattern should be preserved")
		}
	})
}
