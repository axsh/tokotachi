package summarizer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/release-note/internal/scanner"
	"github.com/axsh/tokotachi/features/release-note/internal/summarizer"
)

// mockProvider is a test double for llm.Provider.
type mockProvider struct {
	responses []string
	callCount int
	lastSys   string
	lastUser  string
}

func (m *mockProvider) Summarize(_ context.Context, systemPrompt string, userContent string) (string, error) {
	m.lastSys = systemPrompt
	m.lastUser = userContent
	idx := m.callCount
	m.callCount++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "default response", nil
}

func TestSummarizeBranch(t *testing.T) {
	mock := &mockProvider{
		responses: []string{"【新規】Feature A was added.\n【変更】Feature B was changed."},
	}

	s := summarizer.New(mock)

	spec := scanner.BranchSpec{
		BranchName: "feat-test",
		PhaseName:  "000-foundation",
		FolderPath: "/tmp/test",
		Files:      []string{}, // Empty files list; content passed directly
	}

	result, err := s.SummarizeBranch(context.Background(), spec, "# Test Spec\nSome content here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "【新規】Feature A was added.\n【変更】Feature B was changed." {
		t.Errorf("unexpected result: %s", result)
	}

	// Verify system prompt mentions the three categories
	if !strings.Contains(mock.lastSys, "新規") {
		t.Error("system prompt should mention 新規")
	}
	if !strings.Contains(mock.lastSys, "変更") {
		t.Error("system prompt should mention 変更")
	}
	if !strings.Contains(mock.lastSys, "削除") {
		t.Error("system prompt should mention 削除")
	}
}

func TestConsolidate(t *testing.T) {
	mock := &mockProvider{
		responses: []string{"Final consolidated summary."},
	}

	s := summarizer.New(mock)

	summaries := []string{
		"【新規】Feature A was added.",
		"【変更】Feature B was changed.",
	}

	result, err := s.Consolidate(context.Background(), summaries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Final consolidated summary." {
		t.Errorf("unexpected result: %s", result)
	}

	// Verify system prompt mentions consolidation rules
	if !strings.Contains(mock.lastSys, "統合") || !strings.Contains(mock.lastSys, "最終") {
		t.Error("system prompt should mention consolidation rules")
	}
}

func TestConsolidate_SingleSummary(t *testing.T) {
	mock := &mockProvider{
		responses: []string{"Single summary result."},
	}

	s := summarizer.New(mock)

	summaries := []string{"Only one branch summary."}

	result, err := s.Consolidate(context.Background(), summaries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
	_ = result
}
