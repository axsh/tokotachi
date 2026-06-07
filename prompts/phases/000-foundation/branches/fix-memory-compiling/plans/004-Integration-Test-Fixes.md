# 004-Integration-Test-Fixes

> **Source Specification**: [004-Integration-Test-Fixes.md](../ideas/004-Integration-Test-Fixes.md)

## Goal Description

統合テストで発生している5件の失敗を修正し、全統合テストが成功する状態にする。
加えて、Antigravity IDE がプロセス環境変数に注入するダミー `GITHUB_TOKEN` を `integration_test.sh` でクリアし、IDE 環境でも安定して統合テストを実行できるようにする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: テストコードのバグを修正し、全テストが成功すること | Proposed Changes: scanner_test.go, tt_scaffold_test.go, helpers_test.go |
| R2: テストの期待値が現在のディレクトリ構造・テンプレート出力と一致すること | Proposed Changes: scanner_test.go (パス修正), tt_scaffold_test.go (ファイルリスト修正) |
| R3: integration_test.sh で GITHUB_TOKEN のダミー値をクリアすること | Proposed Changes: integration_test.sh |
| R4: TestTtDownStopsContainer の残留コンテナ問題への対処 | Proposed Changes: helpers_test.go (TestMain に事前クリーンアップ追加) |
| R5 (任意): credential テストの扱い | 現状維持 (credential ファイルが存在しない環境では t.Fatalf で明示的に失敗する動作は正しい) |

## Proposed Changes

### tests/release-note (release-note 統合テスト)

#### [MODIFY] [scanner_test.go](file:///tests/release-note/scanner_test.go)
*   **Description**: ディレクトリ構造の変更に追従し、テストの検証パスを `branches/` 構造に更新する。
*   **Technical Design**:

    **TestScanner_RealPhaseStructure (L50-54)**: `000-foundation/ideas/` の存在確認を `000-foundation/branches/` に変更する。

    変更前:
    ```go
    // Verify "000-foundation/ideas/" subdirectory exists
    ideasDir := filepath.Join(phasesDir, "000-foundation", "ideas")
    if _, err := os.Stat(ideasDir); os.IsNotExist(err) {
        t.Fatalf("000-foundation/ideas/ directory not found at %s", ideasDir)
    }
    ```

    変更後:
    ```go
    // Verify "000-foundation/branches/" subdirectory exists
    branchesDir := filepath.Join(phasesDir, "000-foundation", "branches")
    if _, err := os.Stat(branchesDir); os.IsNotExist(err) {
        t.Fatalf("000-foundation/branches/ directory not found at %s", branchesDir)
    }
    ```

    **TestScanner_FindBranchFolder (L57-86)**: パスを `branches/{branch}/ideas/` 構造に変更し、ブランチ名のハードコードを避けて動的に取得する。

    変更前:
    ```go
    func TestScanner_FindBranchFolder(t *testing.T) {
        branchDir := filepath.Join(
            projectRoot(), "prompts", "phases",
            "000-foundation", "ideas", "fix-module-versioning",
        )
        // ...
        t.Logf("Found %d .md file(s) in fix-module-versioning branch folder", mdCount)
    }
    ```

    変更後:
    ```go
    func TestScanner_FindBranchFolder(t *testing.T) {
        branchesDir := filepath.Join(
            projectRoot(), "prompts", "phases",
            "000-foundation", "branches",
        )

        // Find the first branch directory under branches/
        entries, err := os.ReadDir(branchesDir)
        if err != nil {
            t.Fatalf("failed to read branches directory: %v", err)
        }

        var branchName string
        for _, entry := range entries {
            if entry.IsDir() {
                branchName = entry.Name()
                break
            }
        }
        if branchName == "" {
            t.Fatal("no branch directory found under branches/")
        }

        branchDir := filepath.Join(branchesDir, branchName, "ideas")

        // Verify branch ideas folder exists
        if _, err := os.Stat(branchDir); os.IsNotExist(err) {
            t.Fatalf("branch ideas folder not found at %s", branchDir)
        }

        // Verify at least one .md file exists
        ideaEntries, err := os.ReadDir(branchDir)
        if err != nil {
            t.Fatalf("failed to read branch ideas directory: %v", err)
        }

        mdCount := 0
        for _, entry := range ideaEntries {
            if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
                mdCount++
            }
        }

        if mdCount == 0 {
            t.Fatal("no .md files found in branch ideas folder")
        }

        t.Logf("Found %d .md file(s) in %s branch ideas folder", mdCount, branchName)
    }
    ```

*   **Logic**:
    - `TestScanner_RealPhaseStructure`: `ideas/` の代わりに `branches/` ディレクトリの存在を検証する。`branches/` は scaffold テンプレートが生成する新しいディレクトリ構造。
    - `TestScanner_FindBranchFolder`: `ideas/fix-module-versioning` のハードコードパスを削除し、`branches/` 配下の最初のブランチディレクトリを動的に探索する。ブランチの `ideas/` サブディレクトリ内に `.md` ファイルが1つ以上存在することを検証する。

---

### tests/tt (tt 統合テスト)

#### [MODIFY] [tt_scaffold_test.go](file:///tests/tt/tt_scaffold_test.go)
*   **Description**: `TestScaffoldDefault/CreatesExpectedStructure` の期待ファイルリストを、実際の scaffold テンプレート出力に合わせて更新する。
*   **Technical Design**:

    変更前 (L116-126):
    ```go
    expectedFiles := []string{
        "features/README.md",
        "prompts/phases/README.md",
        "prompts/phases/000-foundation/ideas/.gitkeep",
        "prompts/phases/000-foundation/plans/.gitkeep",
        "prompts/rules/.gitkeep",
        "scripts/.gitkeep",
        "shared/README.md",
        "shared/libs/README.md",
        "work/README.md",
    }
    ```

    変更後:
    ```go
    expectedFiles := []string{
        "AGENTS.md",
        "features/README.md",
        "prompts/phases/README.md",
        "prompts/phases/000-foundation/branches/.gitkeep",
        "prompts/phases/000-foundation/refs/.gitkeep",
        "scripts/.gitkeep",
        "shared/README.md",
        "shared/libs/README.md",
        "work/README.md",
    }
    ```

*   **Logic**:
    - scaffold テンプレートの出力が `ideas/.gitkeep` + `plans/.gitkeep` + `prompts/rules/.gitkeep` から `branches/.gitkeep` + `refs/.gitkeep` + `AGENTS.md` に変わった。
    - 実際の `tt scaffold --yes` コマンドの出力を元に、生成されるファイルリストを正確に反映する。

#### [MODIFY] [helpers_test.go](file:///tests/tt/helpers_test.go)
*   **Description**: `TestMain` に全テスト開始前のコンテナクリーンアップ処理を追加し、前回のテスト実行で残留したコンテナによる競合を防ぐ。
*   **Technical Design**:

    変更前 (L200-205):
    ```go
    // TestMain runs all tests and performs cleanup afterward.
    func TestMain(m *testing.M) {
        code := m.Run()
        cleanupContainers()
        os.Exit(code)
    }
    ```

    変更後:
    ```go
    // TestMain runs all tests with pre/post container cleanup.
    func TestMain(m *testing.M) {
        // Pre-cleanup: remove stale containers from previous test runs
        cleanupContainers()

        code := m.Run()

        // Post-cleanup: remove containers created during this test run
        cleanupContainers()
        os.Exit(code)
    }
    ```

*   **Logic**:
    - `TestTtDownStopsContainer` のセットアップで `tt up` が `docker run` を呼ぶ際、前回のテスト実行で残ったコンテナ `tt-integration-test` がまだ存在する場合に `Conflict. The container name is already in use` エラーが発生する。
    - `TestMain` の `m.Run()` 前に `cleanupContainers()` を呼ぶことで、前回残留したコンテナを事前に除去する。
    - 既存の `cleanupContainers()` 関数は `docker rm -f` を使い、存在しない場合もエラーにならない設計のため、冪等に動作する。

---

### scripts/process (テスト実行スクリプト)

#### [MODIFY] [integration_test.sh](file:///scripts/process/integration_test.sh)
*   **Description**: テスト実行前に `GITHUB_TOKEN` 環境変数をクリアする処理を追加する。Antigravity IDE がプロセス環境にダミートークン `github_pat_antigravitydummytoken` を注入するため、scaffold テスト等で GitHub API の認証エラー (HTTP 401) が発生する問題を防ぐ。
*   **Technical Design**:

    L132 (`# Functions` セクション) の直前に以下を追加:
    ```bash
    # ============================================================
    # Environment Cleanup
    # ============================================================

    # Clear GITHUB_TOKEN if it contains a dummy/invalid value.
    # Antigravity IDE injects a dummy token (github_pat_antigravitydummytoken)
    # into the process environment, which causes HTTP 401 errors when
    # accessing GitHub API in scaffold tests.
    if [[ "${GITHUB_TOKEN:-}" == *"antigravitydummytoken"* ]]; then
        warn "Detected Antigravity dummy GITHUB_TOKEN — clearing it."
        unset GITHUB_TOKEN
    fi
    ```

*   **Logic**:
    - `GITHUB_TOKEN` の値に `antigravitydummytoken` が含まれている場合のみ unset する。
    - ユーザーが有効な `GITHUB_TOKEN` を設定している場合はそのまま保持する。
    - 無条件に unset すると、認証が必要なプライベートリポジトリへのアクセスが失敗する可能性があるため、ダミー値の検出ロジックを使用する。

## Step-by-Step Implementation Guide

1.  **Step 1: tests/tt/helpers_test.go の修正 (TestMain 事前クリーンアップ)**
    *   Edit `tests/tt/helpers_test.go` の `TestMain` 関数。
    *   `m.Run()` の前に `cleanupContainers()` を呼ぶ行を追加する。
    *   コメントを `Pre-cleanup` / `Post-cleanup` に更新する。
    *   `git add tests/tt/helpers_test.go && git commit -m 'fix: add pre-cleanup in TestMain to remove stale containers'`

2.  **Step 2: tests/tt/tt_scaffold_test.go の修正 (期待ファイルリスト更新)**
    *   Edit `tests/tt/tt_scaffold_test.go` L116-126 の `expectedFiles` スライスを更新。
    *   `prompts/phases/000-foundation/ideas/.gitkeep` → `prompts/phases/000-foundation/branches/.gitkeep`
    *   `prompts/phases/000-foundation/plans/.gitkeep` → `prompts/phases/000-foundation/refs/.gitkeep`
    *   `prompts/rules/.gitkeep` を削除し、`AGENTS.md` を追加。
    *   `git add tests/tt/tt_scaffold_test.go && git commit -m 'fix: update expected scaffold files to match new directory structure'`

3.  **Step 3: tests/release-note/scanner_test.go の修正 (ディレクトリパス更新)**
    *   Edit `tests/release-note/scanner_test.go` の `TestScanner_RealPhaseStructure` (L50-54): `ideas` → `branches` に変更。
    *   Edit `tests/release-note/scanner_test.go` の `TestScanner_FindBranchFolder` (L57-86): パスを `branches/` 構造に変更し、ブランチ名を動的探索にする。
    *   `git add tests/release-note/scanner_test.go && git commit -m 'fix: update scanner tests to match branches/ directory structure'`

4.  **Step 4: scripts/process/integration_test.sh の修正 (GITHUB_TOKEN クリア)**
    *   Edit `scripts/process/integration_test.sh` L132 の前に Environment Cleanup セクションを追加。
    *   ダミー `GITHUB_TOKEN` を検出して unset するロジックを追加。
    *   `git add scripts/process/integration_test.sh && git commit -m 'fix: clear Antigravity dummy GITHUB_TOKEN before running tests'`

5.  **Step 5: Verification Plan を実行する。**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests (release-note カテゴリ -- 修正対象テスト)**:

    scanner テストの修正が正しくパスすることを確認する:
    ```bash
    ./scripts/process/integration_test.sh --categories "release-note" --specify "TestScanner"
    ```
    *   **Log Verification**: `TestScanner_RealPhaseStructure` が `000-foundation/branches/` の存在を検証して PASS すること。`TestScanner_FindBranchFolder` がブランチ名を動的に取得して `.md` ファイルの存在を検証して PASS すること。

3.  **Integration Tests (tt カテゴリ -- 修正対象テスト)**:

    scaffold テストの期待ファイルリスト修正と、Docker テストの事前クリーンアップが機能することを確認する:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "TestScaffoldDefault|TestTtDownStopsContainer"
    ```
    *   **Log Verification**: `TestScaffoldDefault/CreatesExpectedStructure` が新しいファイルリストで PASS すること。`TestTtDownStopsContainer` が残留コンテナがあっても事前クリーンアップで PASS すること。

4.  **Integration Tests (全カテゴリ -- リグレッション確認)**:

    `unset GITHUB_TOKEN` なしで全統合テストを実行し、`integration_test.sh` の GITHUB_TOKEN クリア処理が機能することを確認する:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: scaffold テストが `GITHUB_TOKEN` をクリアして PASS すること。全テスト (credential テスト除く) が PASS すること。
    *   **期待結果**: `TestConfigLoad_CredentialFileExists` のみが失敗し、それ以外は全て PASS。

### テスト項目設計のセルフレビュー

**1. 網羅性の検証**: 修正対象の5つの失敗テストすべてに対応する検証ステップがある。`TestConfigLoad_CredentialFileExists` は環境依存であり修正対象外のため、全カテゴリ実行時の期待失敗として扱う。全テストが PASS (1件の期待失敗を除く) すれば、修正が正しく適用されたと言える。

**2. 証拠の十分性**: 各テストの PASS/FAIL ステータスに加え、ログ出力で具体的な検証パス（`branches/` ディレクトリ、動的ブランチ名、新しいファイルリスト）が期待通りであることを確認する。

**3. 迂回・抜け道の排除**: Step 3 の `--specify` で修正対象テストのみを実行した後、Step 4 でフィルタなしの全テスト実行を行い、修正が他のテストに影響していないことを確認する。

**4. 依存関係の整合性**: helpers_test.go (TestMain) → tt_scaffold_test.go / scanner_test.go の順で修正するため、ボトムアップの順序を満たしている。

### 総合判定

全テスト完了後、以下の基準で総合判定を行う:
- `release-note` カテゴリ: 6テスト中5テストが PASS、`TestConfigLoad_CredentialFileExists` のみが FAIL (credential ファイル未配置のため期待通り)
- `tt` カテゴリ: 34テスト全て PASS
- 判定基準: 上記を満たせば PASS。満たさない場合は失敗テストの原因を調査して修正する。

## Documentation

本変更はテストコードの修正とテストインフラの改善のみであり、既存の仕様書やアーキテクチャドキュメントへの影響はない。
