package plan_test

import (
	"testing"

	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/axsh/tokotachi/pkg/matrix"
	"github.com/axsh/tokotachi/pkg/plan"
	"github.com/stretchr/testify/assert"
)

func TestBuildPlan_UpOnly(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature:       "feat-a",
		OS:            detect.OSLinux,
		Editor:        detect.EditorCursor,
		ContainerMode: matrix.ContainerDockerLocal,
		Up:            true,
	})
	assert.True(t, p.ShouldStartContainer)
	assert.False(t, p.ShouldOpenEditor)
}

func TestBuildPlan_UpWithEditor(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature:       "feat-a",
		OS:            detect.OSMacOS,
		Editor:        detect.EditorCursor,
		ContainerMode: matrix.ContainerDevContainer,
		Up:            true,
		EditorOpen:    true,
	})
	assert.True(t, p.ShouldStartContainer)
	assert.True(t, p.ShouldOpenEditor)
	assert.True(t, p.TryDevcontainerAttach)
}

func TestBuildPlan_OpenWithAttach(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature:       "feat-a",
		OS:            detect.OSMacOS,
		Editor:        detect.EditorCursor,
		ContainerMode: matrix.ContainerDevContainer,
		EditorOpen:    true,
		Attach:        true,
	})
	assert.True(t, p.ShouldOpenEditor)
	assert.True(t, p.TryDevcontainerAttach)
}

func TestBuildPlan_OpenWithoutAttach(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature:       "feat-a",
		OS:            detect.OSMacOS,
		Editor:        detect.EditorCursor,
		ContainerMode: matrix.ContainerDevContainer,
		EditorOpen:    true,
		Attach:        false,
	})
	assert.True(t, p.ShouldOpenEditor)
	assert.False(t, p.TryDevcontainerAttach)
}

func TestBuildPlan_OpenAG_NoDevcontainer(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature:       "feat-a",
		OS:            detect.OSMacOS,
		Editor:        detect.EditorAG,
		ContainerMode: matrix.ContainerDockerLocal,
		EditorOpen:    true,
		Attach:        true,
	})
	assert.True(t, p.ShouldOpenEditor)
	assert.False(t, p.TryDevcontainerAttach)
}

func TestBuildPlan_Down(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature: "feat-a",
		Down:    true,
	})
	assert.True(t, p.ShouldStopContainer)
	assert.False(t, p.ShouldOpenEditor)
}

func TestBuildPlan_SSH(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature:       "feat-a",
		OS:            detect.OSLinux,
		Editor:        detect.EditorVSCode,
		ContainerMode: matrix.ContainerDockerSSH,
		Up:            true,
		SSH:           true,
	})
	assert.True(t, p.SSHMode)
	assert.True(t, p.ShouldStartContainer)
}

func TestBuildPlan_Shell(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature: "feat-a",
		Shell:   true,
	})
	assert.True(t, p.ShouldOpenShell)
}

func TestBuildPlan_Exec(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature: "feat-a",
		Exec:    []string{"go", "test", "./..."},
	})
	assert.Equal(t, []string{"go", "test", "./..."}, p.ExecCommand)
}

func TestBuildPlan_Status(t *testing.T) {
	p := plan.Build(plan.Input{
		Feature: "feat-a",
		Status:  true,
	})
	assert.True(t, p.ShouldShowStatus)
}
