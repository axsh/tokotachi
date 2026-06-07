# 004-scaffold-branch-env-and-default

> **Source Specification**: [004-scaffold-branch-env-and-default.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/prompts/phases/000-foundation/branches/feat-arch-memory/ideas/004-scaffold-branch-env-and-default.md)

## Goal Description
テンプレートダウンロード機能において、リポジトリのブランチ名を環境変数 `TT_CONTENT_BRANCH` または `--branch` コマンドフラグから指定（オーバーライド）できるようにし、指定がない場合はデフォルトの `"main"` にフォールバックする仕組みを実装します。

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| デフォルトブランチ名 `"main"` の定義 | Proposed Changes > [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go) |
| 環境変数 `TT_CONTENT_BRANCH` によるオーバーライド | Proposed Changes > [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go) |
| コマンド引数 `--branch` による指定 | Proposed Changes > [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go) / [scaffold.go (cmd)](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/features/tt/cmd/scaffold.go) |
| 優先順位（コマンドフラグ > 環境変数 > デフォルト `"main"`） | Proposed Changes > [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go) / [scaffold_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold_test.go) |

## Proposed Changes

### pkg/scaffold

#### [MODIFY] [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go)
*   **Description**: 定数 `defaultRepoBranch`、および `resolveRepoBranch` ヘルパー関数の追加。`RunOptions` 構造体に `RepoBranch` を追加し、各関数でのブランチ解決と `github.Client` へのセット処理を適用します。また、`List` 関数の引数シグネチャを修正します。
*   **Technical Design**:
    ```go
    // defaultRepoBranch 定数
    const defaultRepoBranch = "main"

    // RunOptions へのフィールド追加
    type RunOptions struct {
        // ...
        RepoBranch string // Template repository branch
    }

    // ブランチ解決用ヘルパー関数
    func resolveRepoBranch(specifiedBranch string) string {
        if specifiedBranch != "" {
            return specifiedBranch
        }
        if envBranch := os.Getenv("TT_CONTENT_BRANCH"); envBranch != "" {
            return envBranch
        }
        return defaultRepoBranch
    }
    ```
*   **Logic**:
    - `Run` 関数の冒頭で `opts.RepoBranch = resolveRepoBranch(opts.RepoBranch)` を適用。
    - `Apply` 関数の冒頭で `opts.RepoBranch = resolveRepoBranch(opts.RepoBranch)` を適用。
    - `List` 関数の引数シグネチャを以下のように更新：
      ```go
      func List(repoURL string, repoBranch string, repoRoot string, filterCategory string) ([]ScaffoldEntry, error)
      ```
      また、冒頭で `repoBranch = resolveRepoBranch(repoBranch)` を適用。
    - `github.NewClient` を呼び出して作成した `downloader` などのクライアントに対し、`downloader.Branch = repoBranch` (または `opts.RepoBranch`) のように解決したブランチ名をセットします。

#### [MODIFY] [scaffold_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold_test.go)
*   **Description**: `resolveRepoBranch` ヘルパー関数の挙動（環境変数なし、環境変数あり、フラグ競合）を検証するテーブル駆動テスト `TestResolveRepoBranch` を追加します。
*   **Technical Design**:
    ```go
    func TestResolveRepoBranch(t *testing.T) {
        tests := []struct {
            name            string
            specifiedBranch string
            envBranch       string
            expectedBranch  string
        }{
            {
                name:            "Default fallback when no flag and no env",
                specifiedBranch: "",
                envBranch:       "",
                expectedBranch:  "main",
            },
            {
                name:            "Env override when no flag",
                specifiedBranch: "",
                envBranch:       "develop",
                expectedBranch:  "develop",
            },
            {
                name:            "Flag takes precedence over env",
                specifiedBranch: "feature/test",
                envBranch:       "develop",
                expectedBranch:  "feature/test",
            },
        }

        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                if tt.envBranch != "" {
                    t.Setenv("TT_CONTENT_BRANCH", tt.envBranch)
                } else {
                    os.Unsetenv("TT_CONTENT_BRANCH")
                }
                actual := resolveRepoBranch(tt.specifiedBranch)
                assert.Equal(t, tt.expectedBranch, actual)
            })
        }
    }
    ```

### features/tt/cmd

#### [MODIFY] [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/features/tt/cmd/scaffold.go)
*   **Description**: コマンドラインフラグ `--branch` を定義し、`RunOptions` および `List` 関数の引数にブランチ指定を渡すよう修正します。
*   **Technical Design**:
    ```go
    // 新規フラグ変数の定義
    var scaffoldFlagBranch string

    // init 関数内でのフラグバインド
    scaffoldCmd.Flags().StringVar(&scaffoldFlagBranch, "branch", "", "Override the default template repository branch")

    // runScaffold 内の変更
    opts := scaffold.RunOptions{
        // ...
        RepoBranch: scaffoldFlagBranch,
    }

    // List 呼び出しの変更
    entries, err := scaffold.List(scaffoldFlagRepo, scaffoldFlagBranch, repoRoot, filterCategory)
    ```

## Step-by-Step Implementation Guide

1.  **[x] [TDD] 単体テストの追加**:
    *   `pkg/scaffold/scaffold_test.go` に `TestResolveRepoBranch` テスト関数を追加します。
    *   この時点では `resolveRepoBranch` 関数や `RunOptions.RepoBranch` が未定義のため、ビルド/テストを実行してコンパイルエラー（Red）になることを確認します。
2.  **[x] [scaffold.go] 定数、構造体、ヘルパー関数の実装**:
    *   `pkg/scaffold/scaffold.go` に `defaultRepoBranch` 定数、`resolveRepoBranch` ヘルパー関数、および `RunOptions` の `RepoBranch` フィールドを追加します。
3.  **[x] [scaffold.go] 各機能への適用とリファクタリング**:
    *   `Run`、`Apply` 関数の冒頭でブランチ解決を行い、作成する `github.Client` に対し `downloader.Branch = opts.RepoBranch` のように設定します。
    *   `List` 関数の引数シグネチャを修正し、`List` 内部でのブランチ解決およびクライアントへのセット処理を追加します。
4.  **[x] [cmd/scaffold.go] コマラインフラグの追加と適用**:
    *   `features/tt/cmd/scaffold.go` を修正し、`--branch` フラグをバインドします。
    *   `List` および `RunOptions` にブランチフラグの値を渡すように修正します。
5.  **[x] 単体テストの実行と確認**:
    *   単体テストを実行し、テストがすべて成功すること（Green）を確認します。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ビルドおよび単体テストを実行します。
    ```bash
    ./scripts/process/build.sh --backend-only
    ```

2.  **Integration Tests**:
    統合テストを実行し、リグレッションがないか確認します。
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories common
    ```

### テスト項目設計のセルフレビュー
1. **網羅性の検証**: 環境変数なし、環境変数あり、環境変数とフラグの両方ありの各組み合わせで、適切な優先順位で解決されることを単体テスト `TestResolveRepoBranch` で網羅しています。
2. **証拠の十分性**: テストにおいて実際の環境変数の設定（`t.Setenv`）と引数の入力を用いて、戻り値が想定されるブランチ名と厳密に一致するか（`assert.Equal`）を検証するため、十分な証拠が得られます。
3. **総合判定**: 単体テストの成功に加え、統合テストの実行によって既存の処理全体が壊れていないことを確認します。

## Documentation

なし。
