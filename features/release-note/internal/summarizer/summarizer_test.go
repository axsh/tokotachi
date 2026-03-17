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
		responses: []string{"[New] Feature A was added.\n[Changed] Feature B was changed."},
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

	if result != "[New] Feature A was added.\n[Changed] Feature B was changed." {
		t.Errorf("unexpected result: %s", result)
	}

	// Verify system prompt mentions the three categories
	if !strings.Contains(mock.lastSys, "New") {
		t.Error("system prompt should mention New")
	}
	if !strings.Contains(mock.lastSys, "Changed") {
		t.Error("system prompt should mention Changed")
	}
	if !strings.Contains(mock.lastSys, "Removed") {
		t.Error("system prompt should mention Removed")
	}
}

func TestConsolidate(t *testing.T) {
	mock := &mockProvider{
		responses: []string{"Final consolidated summary."},
	}

	s := summarizer.New(mock)

	summaries := []string{
		"[New] Feature A was added.",
		"[Changed] Feature B was changed.",
	}

	result, err := s.Consolidate(context.Background(), summaries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Final consolidated summary." {
		t.Errorf("unexpected result: %s", result)
	}

	// Verify system prompt mentions consolidation rules
	if !strings.Contains(mock.lastSys, "consolidat") || !strings.Contains(mock.lastSys, "final") {
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
