# 010-Intake-Processed-Robustness-And-Skill-Structure

> **Source Specification**: [010-Intake-Processed-Robustness-And-Skill-Structure.md](file:///prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/010-Intake-Processed-Robustness-And-Skill-Structure.md)

## Goal Description

intake processed コマンドのファイル不在時フォールバックを追加し、
systematize-far-knowledge スキルの Step 7 に正しいディレクトリ構造を明記する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: ファイル不在時も DB 更新を試みる | Proposed Changes > processed.go, agent_intake.go |
| R1: ErrNotFoundInPending 型付きエラー | Proposed Changes > processed.go |
| R1: ファイル不在 + DB 更新成功 = WARN + 正常完了 | Proposed Changes > agent_intake.go |
| R1: ファイル不在 + DB 更新失敗 = エラー | Proposed Changes > agent_intake.go |
| R2: Step 7 にサブディレクトリ構造を明記 | Proposed Changes > SKILL.md |

## Proposed Changes

### R1: intake パッケージ

#### [MODIFY] [processed_test.go](file:///features/tt/internal/agent/intake/processed_test.go)

*   **Description**: `ErrNotFoundInPending` の型判定テストを追加 (TDD: Red first)
*   **Technical Design**:
    *   既存の `"event not found in pending"` テストケース (L55-66) を拡張し、`errors.Is` で判定:
    ```go
    {
        name: "event not found returns ErrNotFoundInPending",
        setup: func(t *testing.T, varDir string) string {
            t.Helper()
            pendingDir := filepath.Join(varDir, "intake", "pending")
            require.NoError(t, os.MkdirAll(pendingDir, 0o755))
            return ""
        },
        eventID:       "E-NONEXISTENT",
        wantErr:       true,
        errSubstr:     "not found",
        wantSentinel:  true,  // new field
    },
    ```
    *   テストの検証部分に追加:
    ```go
    if tc.wantSentinel {
        assert.ErrorIs(t, err, ErrNotFoundInPending)
    }
    ```
*   **Logic**: `errors.Is` での判定が可能であることを単体テストで保証する

#### [MODIFY] [processed.go](file:///features/tt/internal/agent/intake/processed.go)

*   **Description**: `ErrNotFoundInPending` センチネルエラーを導入
*   **Technical Design**:
    *   import に `"errors"` を追加
    *   パッケージレベル変数を追加:
    ```go
    // ErrNotFoundInPending indicates the event file was not found in pending/.
    var ErrNotFoundInPending = errors.New("event not found in pending")
    ```
    *   L37-38 のエラー返却を変更:
    ```go
    // Before:
    return fmt.Errorf("event %s not found in pending", eventID)
    // After:
    return fmt.Errorf("%w: %s", ErrNotFoundInPending, eventID)
    ```
*   **Logic**: `errors.Is(err, ErrNotFoundInPending)` で呼び出し元がファイル不在を判別可能にする。既存テストの `errSubstr: "not found"` は引き続きマッチする。

---

### R1: cmd パッケージ

#### [MODIFY] [agent_intake.go](file:///features/tt/cmd/agent_intake.go)

*   **Description**: `runAgentIntakeProcessed` にファイル不在時のフォールバック分岐を追加
*   **Technical Design**:
    *   import に `"errors"` を追加
    *   `runAgentIntakeProcessed` を変更:
    ```go
    func runAgentIntakeProcessed(cmd *cobra.Command, args []string) error {
        varDir := filepath.Join("prompts", "memory", "var")
        eventID := args[0]

        moveErr := intake.MoveToProcessed(varDir, eventID)
        if moveErr != nil {
            if !errors.Is(moveErr, intake.ErrNotFoundInPending) {
                return fmt.Errorf("failed to move event to processed: %w", moveErr)
            }
            fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] %v (attempting DB-only update)\n", moveErr)
        }

        // Update index.db status
        dbPath := filepath.Join(varDir, "intake", "index.db")
        idx, err := storage.NewIndex(dbPath)
        if err != nil {
            if moveErr != nil {
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
*   **Logic**:
    *   `MoveToProcessed` がエラーを返した場合、`ErrNotFoundInPending` かどうかで分岐
    *   `ErrNotFoundInPending` の場合: WARN を出力し DB 更新に進む
    *   その他のエラー: 即座に return (ディスク障害等)
    *   DB 更新が失敗 + ファイルも不在の場合: エラーを返す (両方失敗は致命的)
    *   DB 更新が失敗 + ファイルは移動済み: WARN のみ

---

### R2: systematize-far-knowledge スキル

#### [MODIFY] [SKILL.md](file:///.agents/skills/systematize-far-knowledge/SKILL.md)

*   **Description**: Step 7 のスキル配置指示にサブディレクトリ構造と `status: current` を明記。これはソースファイルなので修正後 `prompt compile --apply` でデプロイする。
*   **Technical Design**:
    *   ソースファイル: `prompts/manifest/code_content/procedures/systematize-far-knowledge.md`
    *   L108-112 を以下に変更:
    ```markdown
    1. カテゴリの知識ファイルを確認
    2. capability スキーマに変換:
       - id には `__far-knowledge-` プレフィックスを付与
       - `user_visible: false` を設定
       - `manual_only: false` を設定
       - `status: current` を設定
    3. `prompts/memory/branches/<branch-package-id>/skills/<id>/SKILL.md` に配置
       - 重要: `<id>/SKILL.md` のサブディレクトリ構造にすること (フラットファイル不可)
       - `ScanBranchSkills()` がこの構造を期待する

    配置例:
    ```
    prompts/memory/branches/BR-xxx/skills/
      __far-knowledge-agent-record-branch-package/
        SKILL.md
      __far-knowledge-prompt-memory-architecture/
        SKILL.md
    ```
    ```

## Step-by-Step Implementation Guide

1.  **テストの追加 (TDD: Red)**:
    *   `processed_test.go` に `wantSentinel` フィールドと `errors.Is` 検証を追加
    *   テスト実行して `ErrNotFoundInPending` 未定義でコンパイルエラーを確認

2.  **R1a: processed.go の修正 (TDD: Green)**:
    *   `ErrNotFoundInPending` センチネルエラーを追加
    *   `MoveToProcessed` のエラー返却を `%w` で wrap
    *   テスト通過確認

3.  **R1b: agent_intake.go の修正**:
    *   `runAgentIntakeProcessed` にフォールバック分岐を追加
    *   import に `"errors"` を追加
    *   コンパイル確認

4.  **R2: SKILL.md の修正**:
    *   ソースファイルを修正
    *   `prompt compile --apply` でデプロイ

5.  **Verification Plan の実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "Intake|Processed"
    ```

### Manual Verification

1.  SKILL.md の diff レビュー (Step 7 の変更内容)
