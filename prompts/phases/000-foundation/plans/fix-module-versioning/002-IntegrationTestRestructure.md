# 002-IntegrationTestRestructure

> **Source Specification**: [002-IntegrationTestRestructure.md](file://prompts/phases/000-foundation/ideas/fix-module-versioning/002-IntegrationTestRestructure.md)

## Goal Description

統合テストディレクトリ `tests/integration-test/` を `tests/tt/` に移動してtt専用カテゴリとし、新規に `tests/release-note/` カテゴリを作成してリリースノートプログラムの統合テストを追加する。既存テストの非破壊を保証する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: ttテストの移動 | Proposed Changes > tests/tt/ (全ファイル移動 + go.mod更新) |
| R2: integration-testディレクトリの削除 | Step-by-Step > Step 2 |
| R3: release-note統合テスト作成 | Proposed Changes > tests/release-note/ |
| R4: テスト項目（設定/探索/出力） | Proposed Changes > release-note テストファイル3つ |
| R5: 既存テストの非破壊 | Verification Plan > `--categories "tt"` |

## Proposed Changes

### tests/tt (既存テスト移動)

#### [MODIFY] [go.mod](file://tests/tt/go.mod)
*   **Description**: モジュール名を `tests/integration-test` から `tests/tt` に変更
*   **Technical Design**:
    ```go
    module github.com/axsh/tokotachi/tests/tt
    // (残りの依存は変更なし)
    ```

#### [MODIFY] [helpers_test.go](file://tests/tt/helpers_test.go)
*   **Description**: パス解決のコメントを更新。ロジック自体は変更不要。
*   **Logic**:
    *   `projectRoot()` は `runtime.Caller(0)` でファイルの絶対パスを取得し、2階層上に遡る
    *   `tests/tt/` も `tests/integration-test/` と同じ `tests/` の1階層下なので、2階層上 = プロジェクトルート → **パス解決ロジックの変更は不要**
    *   コメント `// helpers_test.go is in tests/integration-test/` → `// helpers_test.go is in tests/tt/` に更新

---

### tests/release-note (新規統合テスト)

#### [NEW] [go.mod](file://tests/release-note/go.mod)
*   **Description**: release-note統合テスト用の独立Goモジュール
*   **Technical Design**:
    ```go
    module github.com/axsh/tokotachi/tests/release-note

    go 1.24.0

    require gopkg.in/yaml.v3 v3.0.1
    ```

#### [NEW] [helpers_test.go](file://tests/release-note/helpers_test.go)
*   **Description**: 共通ヘルパー関数（プロジェクトルート解決）
*   **Technical Design**:
    ```go
    package release_note_test

    import (
        "fmt"
        "os"
        "path/filepath"
        "runtime"
        "testing"
    )

    // projectRoot returns the absolute path to the project root.
    // Derived from this file's location: tests/release-note/ -> 2 levels up.
    func projectRoot() string

    // TestMain runs all tests.
    func TestMain(m *testing.M) {
        os.Exit(m.Run())
    }
    ```

#### [NEW] [config_load_test.go](file://tests/release-note/config_load_test.go)
*   **Description**: 実プロジェクトの設定ファイルが正しく読み込めることを検証
*   **Technical Design**:
    ```go
    package release_note_test

    func TestConfigLoad_RealProjectFiles(t *testing.T)
    // 1. projectRoot() + "features/release-note/settings/config.yaml" のパスを構築
    // 2. ファイルが存在することを os.Stat() で確認
    // 3. YAML としてパースし、以下のフィールドが存在することを確認:
    //    - credentials_path が空でないこと
    //    - llm.provider が "openai" であること
    //    - llm.model が空でないこと

    func TestConfigLoad_CredentialFileExists(t *testing.T)
    // 1. config.yaml の credentials_path を読み取り
    // 2. settings/ からの相対パスで credential.yaml の存在を確認
    // 3. YAML としてパースし、llm.providers が少なくとも1つ存在することを確認
    // NOTE: API キーの値は検証しない（空でもパスする設計 — CI環境を考慮）
    ```
*   **Logic**:
    *   `gopkg.in/yaml.v3` を使用して YAML パース
    *   構造体は `map[string]any` で汎用的にパースし、フィールド存在チェックのみ行う
    *   API キーの実体値は検証しない（セキュリティ + CI環境対応）

#### [NEW] [scanner_test.go](file://tests/release-note/scanner_test.go)
*   **Description**: 実プロジェクトのフェーズ構造が正しく検出されることを検証
*   **Technical Design**:
    ```go
    package release_note_test

    func TestScanner_RealPhaseStructure(t *testing.T)
    // 1. projectRoot() + "prompts/phases/" のパスを構築
    // 2. ディレクトリ一覧を os.ReadDir() で取得
    // 3. 少なくとも "000-foundation" フォルダが存在することを確認
    // 4. "000-foundation/ideas/" サブディレクトリが存在することを確認

    func TestScanner_FindBranchFolder(t *testing.T)
    // 1. projectRoot() + "prompts/phases/" のパスを構築
    // 2. "000-foundation/ideas/fix-module-versioning/" の存在を確認
    // 3. そのフォルダ内に .md ファイルが少なくとも1つ存在することを確認
    ```
*   **Logic**:
    *   テスト用パッケージインポートは不要（`os` 標準ライブラリのみ使用）
    *   実プロジェクトのディレクトリ構造に依存するため、構造が変わった場合はテスト更新が必要

#### [NEW] [writer_test.go](file://tests/release-note/writer_test.go)
*   **Description**: リリースノート出力の統合テスト（一時ディレクトリに出力して検証）
*   **Technical Design**:
    ```go
    package release_note_test

    func TestWriter_GenerateReleaseNote(t *testing.T)
    // 1. t.TempDir() に一時出力ディレクトリを作成
    // 2. リリースノート内容を文字列として用意:
    //    "【新規】テスト機能が追加されました。\n【変更】設定形式が変更されました。"
    // 3. 以下のファイルを手動で作成:
    //    - {tmpDir}/latest.md:  "# Release Notes\n\n## What's New\n\n{内容}\n"
    //    - {tmpDir}/v1.0.0.md:  同上（アーカイブ）
    // 4. latest.md が存在し、"# Release Notes" ヘッダを含むことを確認
    // 5. v1.0.0.md が存在し、latest.md と同内容であることを確認

    func TestWriter_OutputDirectoryStructure(t *testing.T)
    // 1. projectRoot() + "releases/notes/" のパスを構築
    // 2. ディレクトリが存在することを確認
    // 3. "templates/" サブディレクトリが存在することを確認
    // 4. "templates/release-note.md.tmpl" ファイルが存在することを確認
    ```
*   **Logic**:
    *   `internal/writer` パッケージは直接インポートしない（独立モジュールのため）
    *   代わりに、同じ出力フォーマットを手動で作成して構造を検証
    *   実プロジェクトの `releases/notes/` ディレクトリ構造が正しいことも確認

## Step-by-Step Implementation Guide

### Step 1: ttテストの移動

1.  `tests/tt/` ディレクトリを作成
2.  `tests/integration-test/` の全ファイルを `tests/tt/` にコピー
3.  `tests/tt/go.mod` のモジュール名を `github.com/axsh/tokotachi/tests/tt` に更新
4.  `tests/tt/helpers_test.go` のコメントを `tests/tt/` に更新

### Step 2: 旧ディレクトリの削除

5.  `tests/integration-test/` ディレクトリを削除

### Step 3: ttテストの動作確認

6.  `./scripts/process/build.sh` を実行してビルドが通ることを確認
7.  `./scripts/process/integration_test.sh --categories "tt"` を実行してttテストが全パスすることを確認

### Step 4: release-note統合テストの作成

8.  `tests/release-note/go.mod` を作成
9.  `tests/release-note/helpers_test.go` を作成
10. `tests/release-note/config_load_test.go` を作成
11. `tests/release-note/scanner_test.go` を作成
12. `tests/release-note/writer_test.go` を作成
13. `go mod tidy` を実行して依存を解決

### Step 5: 全テスト検証

14. `./scripts/process/build.sh` を実行
15. `./scripts/process/integration_test.sh --categories "release-note"` を実行してrelease-noteテストが全パスすることを確認
16. `./scripts/process/integration_test.sh` を実行して全カテゴリが正常実行されることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   release-noteの単体テストが引き続きパスすること

2.  **tt統合テスト検証 (R1, R5)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt"
    ```
    *   **Log Verification**: `Category 'tt' (Go) — all tests passed.` が出力されること
    *   全既存テストがパスすること

3.  **release-note統合テスト検証 (R3, R4)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "release-note"
    ```
    *   **Log Verification**: `Category 'release-note' (Go) — all tests passed.` が出力されること
    *   `TestConfigLoad_RealProjectFiles`, `TestScanner_RealPhaseStructure`, `TestWriter_GenerateReleaseNote` 等が全パスすること

4.  **全カテゴリ統合テスト (R2含む)**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: `tt` と `release-note` の2カテゴリが検出・実行されること
    *   `integration-test` カテゴリが表示されないこと（R2: 旧ディレクトリ削除済み）
