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

	values, err := CollectOptionValues(options, provided, strings.NewReader(""), nil)
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

	values, err := CollectOptionValues(options, provided, strings.NewReader(""), nil)
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

	values, err := CollectOptionValues(options, provided, reader, &output)
	require.NoError(t, err)
	assert.Equal(t, "my-feature", values["Name"])
	assert.Contains(t, output.String(), "Name")
}

func TestCollectOptionValues_RequiredMissing(t *testing.T) {
	options := []Option{
		{Name: "Name", Description: "Feature name", Required: true},
	}
	provided := map[string]string{}

	// Empty input for required field
	reader := strings.NewReader("\n")
	var output strings.Builder

	_, err := CollectOptionValues(options, provided, reader, &output)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

func TestCollectOptionValues_NoOptions(t *testing.T) {
	values, err := CollectOptionValues(nil, nil, strings.NewReader(""), nil)
	require.NoError(t, err)
	assert.Empty(t, values)
}
