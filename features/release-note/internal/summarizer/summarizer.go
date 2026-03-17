package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/axsh/tokotachi/features/release-note/internal/llm"
	"github.com/axsh/tokotachi/features/release-note/internal/scanner"
)

const branchSummarySystemPrompt = `You are a release note author. Read the specification files below and
classify the changes into the following three categories, focusing on
the impact to the end user (the person using the program):

(1) [New]: New features, new settings, etc.
(2) [Changed]: How existing features/settings have changed (Before → After)
(3) [Removed]: Deprecated features, settings, etc.

List items as bullet points under each category. Omit any category that
has no items. Describe the "diff" from the user's perspective concisely.`

const consolidateSystemPrompt = `Below are summaries of multiple changes. Consolidate them into a final
release note.

Consolidation rules:
- Remove intermediate states and describe only the final state
  (e.g. "A became B" + "B became C" → "A became C")
- For duplicate changes to the same item, describe only the final state
- If a removal and an addition share the same name, consolidate into
  "behavior changed" or similar
- Group related items together
- Focus on "what is the final outcome"
- Provide a clear final diff for the user, not a verbose change history

Output format:
Classify items under (1) [New], (2) [Changed], (3) [Removed] as bullet
points. Omit any category that has no items.`

// Summarizer orchestrates LLM-based summarization.
type Summarizer struct {
	provider llm.Provider
}

// New creates a new Summarizer with the given LLM provider.
func New(provider llm.Provider) *Summarizer {
	return &Summarizer{provider: provider}
}

// SummarizeBranch takes a BranchSpec and the combined content of its files,
// then asks the LLM to produce a categorized summary.
func (s *Summarizer) SummarizeBranch(ctx context.Context, branch scanner.BranchSpec, content string) (string, error) {
	userContent := fmt.Sprintf("Branch: %s (Phase: %s)\n\n%s", branch.BranchName, branch.PhaseName, content)

	result, err := s.provider.Summarize(ctx, branchSummarySystemPrompt, userContent)
	if err != nil {
		return "", fmt.Errorf("failed to summarize branch %s: %w", branch.BranchName, err)
	}

	return result, nil
}

// Consolidate takes all per-branch summaries and produces a final
// integrated summary following the consolidation rules.
func (s *Summarizer) Consolidate(ctx context.Context, branchSummaries []string) (string, error) {
	combined := strings.Join(branchSummaries, "\n\n---\n\n")

	result, err := s.provider.Summarize(ctx, consolidateSystemPrompt, combined)
	if err != nil {
		return "", fmt.Errorf("failed to consolidate summaries: %w", err)
	}

	return result, nil
}
