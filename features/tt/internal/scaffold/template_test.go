package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTemplate_Simple(t *testing.T) {
	result, err := ProcessTemplate("Hello {{.Name}}", map[string]string{"Name": "world"})
	require.NoError(t, err)
	assert.Equal(t, "Hello world", result)
}

func TestProcessTemplate_NoVars(t *testing.T) {
	result, err := ProcessTemplate("Hello World", map[string]string{})
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result)
}

func TestProcessTemplate_MultipleVars(t *testing.T) {
	tmpl := "module {{.GoModule}}\nname: {{.Name}}"
	values := map[string]string{
		"GoModule": "github.com/example/foo",
		"Name":     "foo",
	}
	result, err := ProcessTemplate(tmpl, values)
	require.NoError(t, err)
	assert.Contains(t, result, "module github.com/example/foo")
	assert.Contains(t, result, "name: foo")
}

func TestProcessTemplatePath(t *testing.T) {
	result, err := ProcessTemplatePath("features/{{.Name}}", map[string]string{"Name": "my-feature"})
	require.NoError(t, err)
	assert.Equal(t, "features/my-feature", result)
}

func TestCollectOptionValues_AllProvided(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true},
		{Name: "GoModule", Description: "Go module path", Required: false, Default: "default-module"},
	}
	provided := map[string]string{"Name": "my-feature", "GoModule": "custom-module"}

	values, err := CollectOptionValues(options, provided, strings.NewReader(""), nil, false)
	require.NoError(t, err)
	assert.Equal(t, "my-feature", values["Name"])
	assert.Equal(t, "custom-module", values["GoModule"])
}

func TestCollectOptionValues_DefaultApplied(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true},
		{Name: "GoModule", Description: "Go module path", Required: false, Default: "default-module"},
	}
	provided := map[string]string{"Name": "my-feature"}

	// useDefaults=true: non-required options with defaults are auto-applied
	values, err := CollectOptionValues(options, provided, strings.NewReader(""), nil, true)
	require.NoError(t, err)
	assert.Equal(t, "my-feature", values["Name"])
	assert.Equal(t, "default-module", values["GoModule"])
}

func TestCollectOptionValues_InteractiveInput(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true},
	}
	provided := map[string]string{}

	// Simulate user typing "my-feature\n"
	reader := strings.NewReader("my-feature\n")
	var output strings.Builder

	values, err := CollectOptionValues(options, provided, reader, &output, false)
	require.NoError(t, err)
	assert.Equal(t, "my-feature", values["Name"])
	assert.Contains(t, output.String(), "Name")
}

func TestCollectOptionValues_RequiredMissing(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true},
	}
	provided := map[string]string{}

	// Empty input for required field without default
	reader := strings.NewReader("\n")
	var output strings.Builder

	_, err := CollectOptionValues(options, provided, reader, &output, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

func TestCollectOptionValues_NoOptions(t *testing.T) {
	values, err := CollectOptionValues(nil, nil, strings.NewReader(""), nil, false)
	require.NoError(t, err)
	assert.Empty(t, values)
}

func TestCollectOptionValues_InteractiveNonRequired(t *testing.T) {
	options := []Option{
		{Name: "GoModule", Description: "Go module path", Required: false, Default: "default-module"},
	}
	provided := map[string]string{}

	// useDefaults=false: non-required options are prompted interactively
	reader := strings.NewReader("custom-module\n")
	var output strings.Builder

	values, err := CollectOptionValues(options, provided, reader, &output, false)
	require.NoError(t, err)
	assert.Equal(t, "custom-module", values["GoModule"])
	assert.Contains(t, output.String(), "GoModule")
}

func TestCollectOptionValues_InteractiveEmptyEnterWithDefault(t *testing.T) {
	options := []Option{
		{Name: "GoModule", Description: "Go module path", Required: false, Default: "default-module"},
	}
	provided := map[string]string{}

	// Empty enter on non-required option with default -> default applied
	reader := strings.NewReader("\n")
	var output strings.Builder

	values, err := CollectOptionValues(options, provided, reader, &output, false)
	require.NoError(t, err)
	assert.Equal(t, "default-module", values["GoModule"])
}

func TestCollectOptionValues_RequiredWithDefaultEmptyEnter(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true, Default: "myprog"},
	}
	provided := map[string]string{}

	// Empty enter on required option with default -> default applied
	reader := strings.NewReader("\n")
	var output strings.Builder

	values, err := CollectOptionValues(options, provided, reader, &output, false)
	require.NoError(t, err)
	assert.Equal(t, "myprog", values["Name"])
}

func TestCollectOptionValues_RequiredNoDefaultEmptyEnter(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true},
	}
	provided := map[string]string{}

	// Empty enter on required option without default -> error
	reader := strings.NewReader("\n")
	var output strings.Builder

	_, err := CollectOptionValues(options, provided, reader, &output, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

func TestCollectOptionValues_UseDefaultsSkipsNonRequired(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true},
		{Name: "GoModule", Description: "Go module path", Required: false, Default: "default-module"},
	}
	provided := map[string]string{}

	// useDefaults=true: required options are prompted, non-required are auto-applied
	reader := strings.NewReader("my-feature\n")
	var output strings.Builder

	values, err := CollectOptionValues(options, provided, reader, &output, false)
	require.NoError(t, err)
	assert.Equal(t, "my-feature", values["Name"])

	// Now test with useDefaults=true
	reader2 := strings.NewReader("my-feature\n")
	var output2 strings.Builder

	values2, err := CollectOptionValues(options, provided, reader2, &output2, true)
	require.NoError(t, err)
	assert.Equal(t, "my-feature", values2["Name"])
	assert.Equal(t, "default-module", values2["GoModule"])
	// The prompt output should NOT contain GoModule when useDefaults=true
	assert.NotContains(t, output2.String(), "GoModule")
}

func TestCollectOptionValues_PromptFormat(t *testing.T) {
	options := []Option{
		{Name: "GoModule", Description: "Go module path", Required: false, Default: "default-module"},
	}
	provided := map[string]string{}

	reader := strings.NewReader("val\n")
	var output strings.Builder

	_, err := CollectOptionValues(options, provided, reader, &output, false)
	require.NoError(t, err)
	// Prompt should display the default value hint
	assert.Contains(t, output.String(), "default-module")
}

func TestParseOptionOverrides(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "single key-value",
			args: []string{"feature_name=foobar"},
			want: map[string]string{"feature_name": "foobar"},
		},
		{
			name: "multiple key-values",
			args: []string{"feature_name=foo", "go_module=github.com/example"},
			want: map[string]string{"feature_name": "foo", "go_module": "github.com/example"},
		},
		{
			name: "empty args",
			args: nil,
			want: map[string]string{},
		},
		{
			name: "value with equals sign",
			args: []string{"key=val=ue"},
			want: map[string]string{"key": "val=ue"},
		},
		{
			name:    "missing equals sign",
			args:    []string{"invalid"},
			wantErr: true,
		},
		{
			name:    "empty key",
			args:    []string{"=value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOptionOverrides(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
