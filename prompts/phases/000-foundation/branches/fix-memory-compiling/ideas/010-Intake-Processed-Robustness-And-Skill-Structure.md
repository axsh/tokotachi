# 010-Intake-Processed-Robustness-And-Skill-Structure

## 背景 (Background)

009-Agent-Tooling-Fixes で intake processed のインデックス同期を実装したが、
systematize-far-knowledge ワークフロー実行時に以下の2つの問題が発覚した:

1. **intake processed で「ファイルが既に pending にない」場合のハンドリング漏れ**:
   前回の processed 移動はインデックス更新なしの旧コードで実施されたため、ファイルは
   `processed/` に移動済みだが `index.db` の status は `pending` のままだった。
   新コードで再度 processed を実行しようとすると、`MoveToProcessed` が
   「event not found in pending」エラーを返し、DB 更新に到達せずに終了する。
   手動で `sqlite3` コマンドによる直接 UPDATE が必要になった。

2. **far-knowledge スキルのソースファイル構造が誤っていた**:
   `ScanBranchSkills()` は `branches/*/skills/<id>/SKILL.md` という
   サブディレクトリ構造を期待するが、systematize-far-knowledge スキルの
   Step 7 では「`prompts/memory/branches/<branch-package-id>/skills/` に配置」
   としか記述されておらず、フラットなファイル (`<id>.md`) で作成してしまった。
   結果として `prompt update` でスキルが検出されず、`.agents/skills/` に
   デプロイされなかった。

## 要件 (Requirements)

### R1: intake processed のファイル不在時のフォールバック (必須)

- `intake processed <event-id>` 実行時、ファイルが pending に存在しない場合でも
  `index.db` の status 更新を試みること
- ファイル移動が成功した場合も、従来通り DB 更新を実施すること
- 最終的な動作パターン:

| ファイル移動 | DB 更新 | 結果 |
|:---:|:---:|:---|
| 成功 | 成功 | 正常完了 |
| 成功 | 失敗 | WARN 出力 + 正常完了 |
| 失敗 (not found) | 成功 | WARN 出力 + 正常完了 |
| 失敗 (not found) | 失敗 | エラー |
| 失敗 (その他) | - | エラー |

- 「ファイルが pending にない」ケースは `ErrNotFoundInPending` として型付きエラーで判別する
- その他の移動エラー (例: ディスク障害) は即座にエラーを返す

### R2: systematize-far-knowledge スキル手順の修正 (必須)

- Step 7 のスキル配置指示に、サブディレクトリ構造 (`<id>/SKILL.md`) を明記する
- `ScanBranchSkills()` が期待する構造を具体例で示す
- 修正前: `prompts/memory/branches/<branch-package-id>/skills/ に配置`
- 修正後: `prompts/memory/branches/<branch-package-id>/skills/<id>/SKILL.md に配置`

## 実現方針 (Implementation Approach)

### R1: intake processed のフォールバック

`processed.go` に型付きエラーを導入し、`agent_intake.go` でエラー種別に応じた
分岐を実装する。

**processed.go の変更:**

```go
// ErrNotFoundInPending indicates the event file was not found in pending/.
var ErrNotFoundInPending = fmt.Errorf("event not found in pending")

func MoveToProcessed(varDir, eventID string) error {
    // ... (既存の WalkDir ロジック) ...
    if foundPath == "" {
        return fmt.Errorf("%w: %s", ErrNotFoundInPending, eventID)
    }
    // ... (以降は既存のまま) ...
}
```

**agent_intake.go の変更:**

```go
func runAgentIntakeProcessed(cmd *cobra.Command, args []string) error {
    varDir := filepath.Join("prompts", "memory", "var")
    eventID := args[0]

    moveErr := intake.MoveToProcessed(varDir, eventID)
    if moveErr != nil {
        if !errors.Is(moveErr, intake.ErrNotFoundInPending) {
            // ディスク障害等の致命的エラー
            return fmt.Errorf("failed to move event to processed: %w", moveErr)
        }
        // ファイルは既に移動済み、DB のみ更新を試みる
        fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] %v (attempting DB-only update)\n", moveErr)
    }

    // Update index.db status
    dbPath := filepath.Join(varDir, "intake", "index.db")
    idx, err := storage.NewIndex(dbPath)
    if err != nil {
        if moveErr != nil {
            // ファイルも DB も更新できない場合はエラー
            return fmt.Errorf("event %s: file not in pending and index unavailable: %w", eventID, err)
        }
        fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] Index update skipped: %v\n", err)
    } else {
        defer idx.Close()
        if updateErr := idx.UpdateStatus(eventID, "processed"); updateErr != nil {
            if moveErr != nil {
                return fmt.Errorf("event %s: file not in pending and index update failed: %w", eventID, updateErr)
            }
            fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] Index update failed: %v\n", updateErr)
        }
    }

    fmt.Printf("Event %s moved to processed\n", eventID)
    return nil
}
```

### R2: スキル手順の修正

`systematize-far-knowledge` の SKILL.md Step 7 を修正する。

修正箇所:
- `.agents/skills/systematize-far-knowledge/SKILL.md` L108-112
- 同ファイルが `.claude/skills/` と `.cursor/skills/` にもデプロイされているため、
  ソース修正後に `prompt compile --apply` を実行

修正内容:
```markdown
1. カテゴリの知識ファイルを確認
2. capability スキーマに変換:
   - id には `__far-knowledge-` プレフィックスを付与
   - `user_visible: false` を設定
   - `manual_only: false` を設定
   - `status: current` を設定
3. `prompts/memory/branches/<branch-package-id>/skills/<id>/SKILL.md` に配置
   - 重要: フラットなファイルではなく、`<id>/SKILL.md` のサブディレクトリ構造にすること
   - `ScanBranchSkills()` がこの構造を期待する

配置例:
prompts/memory/branches/BR-fix-memory-compiling-a378fee0/skills/
  __far-knowledge-agent-record-branch-package/
    SKILL.md
  __far-knowledge-prompt-memory-architecture/
    SKILL.md
```

## 検証シナリオ (Verification Scenarios)

### S1: ファイル移動済み・DB 未更新のケース

1. `tt agent record` でイベントを作成 (E-TEST-001)
2. `index.db` に pending として登録される
3. pending ディレクトリからファイルを手動で processed に移動 (DB は pending のまま)
4. `tt agent intake processed E-TEST-001` を実行
5. WARN メッセージが出力されるが、エラーにはならない
6. `tt agent intake list --status pending` に E-TEST-001 が表示されない
7. `tt agent intake list --status processed` に E-TEST-001 が表示される

### S2: 正常ケース (ファイル・DB 両方更新)

1. `tt agent record` でイベントを作成 (E-TEST-002)
2. `tt agent intake processed E-TEST-002` を実行
3. エラーもWARNも出力されない
4. ファイルが processed に移動している
5. `tt agent intake list --status processed` に E-TEST-002 が表示される

### S3: 存在しないイベントのケース

1. `tt agent intake processed E-NONEXISTENT` を実行
2. ファイルもDBにもないため、エラーが返る

### S4: スキル構造の検証

1. `prompts/memory/branches/<id>/skills/__far-knowledge-test/SKILL.md` を作成
2. `tt prompt compile --dry-run` を実行
3. `__far-knowledge-test` が出力に含まれる
4. フラットな `prompts/memory/branches/<id>/skills/__far-knowledge-test.md` を作成
5. `tt prompt compile --dry-run` を実行
6. 出力に含まれない (サブディレクトリ構造のみ検出される)

## テスト項目 (Testing for the Requirements)

### R1: intake processed フォールバック

- **単体テスト**: `processed_test.go` に `TestMoveToProcessed_NotFound` を追加
  - pending にファイルがない場合に `ErrNotFoundInPending` が返されることを検証
  - `errors.Is(err, ErrNotFoundInPending)` で判別できることを検証

- **単体テスト**: `cmd/agent_intake_test.go` に追加 (または既存テストを拡張)
  - ファイル不在時にDB更新が実行されることを検証

### R2: スキル手順

- **手動検証**: SKILL.md の diff レビュー

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```

2. tt 統合テスト:
   ```
   scripts/process/integration_test.sh --categories "tt" --specify "Intake|Processed|FarKnowledge"
   ```
