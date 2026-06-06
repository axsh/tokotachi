# 003-scaffold-repo-env-and-default

> **Source Specification**: [003-scaffold-repo-env-and-default.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/prompts/phases/000-foundation/branches/feat-arch-memory/ideas/003-scaffold-repo-env-and-default.md)

## Goal Description
テンプレートダウンロード機能におけるデフォルトリポジトリURLを `https://github.com/axsh/tokotachi` に変更し、環境変数 `TT_CONTENT_REPO` による指定（上書き）を可能にします。また、環境変数がない場合はデフォルトURLにフォールバックされるよう制御します。

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| デフォルトURLを `https://github.com/axsh/tokotachi` に変更 | Proposed Changes > [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go) |
| 環境変数 `TT_CONTENT_REPO` によるオーバーライド | Proposed Changes > [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go) |
| 優先順位（コマンドフラグ > 環境変数 > デフォルトURL） | Proposed Changes > [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go) |
| 自動化されたテスト（単体テスト・統合テスト）の実施 | Proposed Changes > [scaffold_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold_test.go) (NEW) / Verification Plan |

## Proposed Changes

### pkg/scaffold

#### [MODIFY] [scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go)
*   **Description**: デフォルトURL定数の変更、および優先順位に基づきURLを解決するヘルパー関数の追加。`Run`、`List`、`Apply` 各関数でこのヘルパー関数を使用するように修正します。
*   **Technical Design**:
    ```go
    // defaultRepoURL 定数の値を変更
    const defaultRepoURL = "https://github.com/axsh/tokotachi"

    // URL解決ヘルパー関数
    func resolveRepoURL(specifiedURL string) string {
        if specifiedURL != "" {
            return specifiedURL
        }
        if envURL := os.Getenv("TT_CONTENT_REPO"); envURL != "" {
            return envURL
        }
        return defaultRepoURL
    }
    ```
*   **Logic**:
    - `resolveRepoURL` 関数は、引数で渡された `specifiedURL`（コマンドラインフラグ由来）が空でない場合はそれをそのまま返します。
    - 空の場合は、環境変数 `TT_CONTENT_REPO` を確認し、設定されていればその値を返します。
    - 環境変数も空の場合は、定数 `defaultRepoURL`（`https://github.com/axsh/tokotachi`）を返します。
    - `Run`, `List`, `Apply` 関数の冒頭で `opts.RepoURL = resolveRepoURL(opts.RepoURL)` または `repoURL = resolveRepoURL(repoURL)` を呼び出し、解決されたURLで処理を続行します。

#### [NEW] [scaffold_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold_test.go)
*   **Description**: 新規作成する `scaffold.go` に対する単体テストファイル。`resolveRepoURL` 関数の挙動をテーブル駆動テストで検証します。
*   **Technical Design**:
    ```go
    package scaffold

    import (
        "os"
        "testing"

        "github.com/stretchr/testify/assert"
    )

    func TestResolveRepoURL(t *testing.T) {
        tests := []struct {
            name         string
            specifiedURL string
            envURL       string
            expectedURL  string
        }{
            {
                name:         "Default fallback when no flag and no env",
                specifiedURL: "",
                envURL:       "",
                expectedURL:  "https://github.com/axsh/tokotachi",
            },
            {
                name:         "Env override when no flag",
                specifiedURL: "",
                envURL:       "https://github.com/some-owner/some-repo",
                expectedURL:  "https://github.com/some-owner/some-repo",
            },
            {
                name:         "Flag takes precedence over env",
                specifiedURL: "https://github.com/flag-owner/flag-repo",
                envURL:       "https://github.com/some-owner/some-repo",
                expectedURL:  "https://github.com/flag-owner/flag-repo",
            },
        }

        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                if tt.envURL != "" {
                    t.Setenv("TT_CONTENT_REPO", tt.envURL)
                } else {
                    os.Unsetenv("TT_CONTENT_REPO")
                }
                actual := resolveRepoURL(tt.specifiedURL)
                assert.Equal(t, tt.expectedURL, actual)
            })
        }
    }
    ```

## Step-by-Step Implementation Guide

1.  **[TDD] 新規テストファイルの作成**:
    *   `pkg/scaffold/scaffold_test.go` を新規作成し、`TestResolveRepoURL` のテーブル駆動テストを実装します。
    *   この時点では `resolveRepoURL` が未実装（または未変更）のため、ビルド/テストが失敗すること（Red）を確認します。
2.  **[scaffold.go] 定数およびヘルパー関数の実装**:
    *   `pkg/scaffold/scaffold.go` 内の `defaultRepoURL` を `https://github.com/axsh/tokotachi` に変更します。
    *   `resolveRepoURL` ヘルパー関数を `scaffold.go` に追加します（必要に応じて `"os"` パッケージをインポートに追加します）。
3.  **[scaffold.go] 各機能への適用**:
    *   `Run`、`List`、`Apply` 関数（および依存パッケージの読み込みで `opts.RepoURL` を利用している箇所）の冒頭で `resolveRepoURL` を使用してリポジトリURLを解決するように修正します。
4.  **単体テストの実行と確認**:
    *   単体テストを実行し、テストがすべて成功すること（Green）を確認します。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ビルドおよび単体テストを実行します。
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests**:
    共通機能（common）に関する統合テストを実行し、リグレッションがないか確認します。
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc && ./scripts/process/integration_test.sh --categories common
    ```

### テスト項目設計のセルフレビュー
1. **網羅性の検証**: 環境変数なし、環境変数あり、環境変数とフラグの両方ありの各組み合わせで、適切な優先順位（フラグ > 環境変数 > デフォルトURL）で解決されることを単体テスト `TestResolveRepoURL` で網羅しています。
2. **証拠の十分性**: テストにおいて実際の環境変数の設定（`t.Setenv`）と引数の入力を用いて、戻り値が想定されるURLと厳密に一致するか（`assert.Equal`）を検証するため、十分な証拠が得られます。
3. **総合判定**: 単体テストの成功に加え、統合テスト（`common` カテゴリ）の実行によって既存のscaffold処理全体が正しく動作し、リグレッションが起きていないことを確認します。

## Documentation

なし。
