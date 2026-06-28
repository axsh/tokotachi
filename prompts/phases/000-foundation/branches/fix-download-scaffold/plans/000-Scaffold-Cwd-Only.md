# 000-Scaffold-Cwd-Only

> **Source Specification**: [000-Scaffold-Cwd-Only.md](file:///c:/Users/yamya/myprog/tokotachi/work/fix-download-scaffold/prompts/phases/000-foundation/branches/fix-download-scaffold/ideas/000-Scaffold-Cwd-Only.md)

## Goal Description

`tt scaffold` コマンド実行時にリポジトリルートを決定する際、自動で親ディレクトリを遡って `git rev-parse --show-toplevel` により Git ルートを検出する処理を廃止します。
代わりに、他の `tt` サブコマンドと一貫性を持たせ、実行時のカレントディレクトリ（CWD）をデフォルトのルートとして使用するように変更します。これにより、ホームディレクトリなどに意図しない `.git` が存在していた場合に `tt scaffold` を誤実行してシステムフォルダへのアクセス権限エラー（`Access is denied`）でクラッシュする問題を防ぎます。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `tt scaffold` 実行時に `--root` オプションが指定されていない場合、Git ルートの自動探索を行わず、デフォルトでカレントディレクトリをルートとする | Proposed Changes > [scaffold.go](file://features/tt/cmd/scaffold.go) |
| 明示的に `--root` オプションが指定された場合は、指定されたパスを基準にする | Proposed Changes > [scaffold.go](file://features/tt/cmd/scaffold.go) |

## Proposed Changes

### CLI Cmd (Go)

#### [NEW] [scaffold_test.go](file://features/tt/cmd/scaffold_test.go)
*   **Description**: `resolveRepoRoot` 関数の挙動をアサートする単体テストを追加します。
*   **Technical Design**:
    ```go
    package cmd

    import (
    	"os"
    	"testing"
    	"github.com/stretchr/testify/assert"
    	"github.com/stretchr/testify/require"
    )

    func TestResolveRepoRoot(t *testing.T) {
    	// 1. --root が指定された場合、指定されたパスがそのまま返る
    	assert.Equal(t, "/some/root/path", resolveRepoRoot("/some/root/path"))

    	// 2. --root が空文字列の場合、カレントディレクトリが返る
    	wd, err := os.Getwd()
    	require.NoError(t, err)
    	assert.Equal(t, wd, resolveRepoRoot(""))
    }
    ```

---

#### [MODIFY] [scaffold.go](file://features/tt/cmd/scaffold.go)
*   **Description**: `resolveRepoRoot` 関数から `git rev-parse --show-toplevel` の呼び出しを削除し、直接 CWD を返すように修正します。
*   **Technical Design**:
    ```go
    func resolveRepoRoot(rootPath string) string {
    	if rootPath != "" {
    		return rootPath
    	}
    	wd, err := os.Getwd()
    	if err != nil {
    		return "."
    	}
    	return wd
    }
    ```

---

## Step-by-Step Implementation Guide

1.  **単体テストの追加 (TDD: Failed First)**:
    *   新規ファイル `features/tt/cmd/scaffold_test.go` を作成し、`TestResolveRepoRoot` を追加します。
    *   この時点では `resolveRepoRoot` は古い実装（Gitルート自動探索あり）なので、テストが失敗するか、またはローカルの Git 環境に依存した不安定な挙動になる可能性があります（特に CWD に Git ルート以外の場所がある場合）。

2.  **ロジックの修正**:
    *   `features/tt/cmd/scaffold.go` の `resolveRepoRoot` を修正して `git rev-parse` 処理を削除し、常に `os.Getwd()` を返すようにします。

3.  **単体テストの検証**:
    *   ビルドおよび単体テストを実行して、新規テストケースを含むすべての単体テストが正常に通過することを確認します。
    *   `./scripts/process/build.sh --skip-frontend --skip-etc` を実行します。

4.  **最終検証**:
    *   後述の `Verification Plan` に従い、ビルドおよび統合テストを実行します。

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ビルドおよび全単体テストを実行します。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    `scaffold` はファイルの配置等を行うため、統合テスト（`template` カテゴリ）を実行してリグレッションがないことを確認します。
    ```bash
    ./scripts/process/integration_test.sh --categories "template"
    ```

3.  **E2E Tests (新規/追加)**:
    本修正は内部のルートパス特定ロジック（`resolveRepoRoot`）の改善であり、新規機能の追加ではありません。また、既存の `template` カテゴリ統合テストによって scaffold 機能全体の検証が自動化されているため、新規 E2E テストの追加は不要と判断します。
