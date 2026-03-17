package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/axsh/tokotachi/features/release-note/internal/llm"
	"github.com/axsh/tokotachi/features/release-note/internal/scanner"
)

const branchSummarySystemPrompt = `あなたはリリースノートの作成者です。以下の仕様書ファイルの内容を読み、
ユーザー（プログラムの利用者）が受ける影響に着目して、変更を以下の3カテゴリに分類してください:

(1)【新規】: 新しい機能、新しい設定などの登場
(2)【変更】: 既存の機能・設定がどう変わるのか（Before → After）
(3)【削除】: 廃止される機能、設定など

各カテゴリごとに箇条書きで記述し、該当しないカテゴリは省略してください。
ユーザーにとっての「差分」を簡潔に表現してください。`

const consolidateSystemPrompt = `以下は複数の変更の要約です。これらを統合し、最終的なリリースノートを作成してください。

統合ルール:
- 中間状態を除去し、最終状態のみ記述する（例: 「AがBになった」「BがCになった」→「AがCになった」）
- 同じ項目への重複した変更は最終状態のみ記述する
- 削除と追加が同名の場合は「新しい挙動になった」等に統合する
- 関連項目はグルーピングしてまとめる
- 「結局最終的にどうなったのか」に着目すること
- 冗長な変更履歴の列挙ではなく、利用者にとって分かりやすい最終差分を記述すること

出力フォーマット:
(1)【新規】【変更】【削除】のカテゴリに分類して箇条書きで記述してください。
該当しないカテゴリは省略してください。`

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
