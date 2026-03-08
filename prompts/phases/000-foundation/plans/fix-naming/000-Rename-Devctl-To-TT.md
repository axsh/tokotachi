# 000-Rename-Devctl-To-TT

> **Source Specification**: [000-Rename-Devctl-To-TT.md](file://prompts/phases/000-foundation/ideas/fix-naming/000-Rename-Devctl-To-TT.md)

## Goal Description

CLIツール名を `devctl` から `tt` (Tokotachi の略称) へ全面的にリネームする。対象はソースコード内の文字列リテラル、Go識別子（関数名・テスト名）、環境変数プレフィックス `DEVCTL_*` → `TT_*`、ファイル名・ディレクトリ名、設定ファイル、シェルスクリプト、ドキュメント（`prompts/` 配下は除外）。

## User Review Required

> [!IMPORTANT]
> **ディレクトリリネーム `features/devctl/` → `features/tt/`** は Git のファイル履歴に影響します。`git mv` を使用して履歴を維持します。

> [!WARNING]
> **Go モジュールパス変更**: `github.com/axsh/tokotachi/features/devctl` → `github.com/axsh/tokotachi/features/tt` は全 import パス（53ファイル以上）に波及します。一括 `sed` 置換で対応します。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| プログラム内の文字列リテラル変更 | Step 3: sed 一括置換 (全 .go ファイル) |
| 構造体名・関数名の変更 | Step 3: sed 一括置換 (Devctl→Tt, devctl→tt) |
| テスト名の変更 (単体テスト) | Step 3: sed 一括置換 |
| テスト名の変更 (統合テスト) | Step 4: 統合テストファイルリネーム + sed 置換 |
| 環境変数プレフィックス DEVCTL_* → TT_* | Step 3: sed 一括置換 |
| ファイル名/ディレクトリ名の変更 | Step 1-2: git mv による物理リネーム |
| go.mod / import パスの変更 | Step 3: sed 一括置換 |
| 設定ファイル (YAML/テンプレート) の変更 | Step 5: sed 一括置換 |
| シェルスクリプトの変更 | Step 6: sed 一括置換 |
| ドキュメントの変更 (prompts/ 除外) | Step 7: sed 一括置換 |
| ビルド出力パスの変更 bin/devctl → bin/tt | Step 3, 6: sed 置換 |

## Proposed Changes

### コンポーネント1: ディレクトリ構造 (物理リネーム)

#### [MODIFY] features/ ディレクトリ

*   **Description**: `features/devctl/` → `features/tt/` へディレクトリごとリネーム
*   **Technical Design**:
    ```bash
    git mv features/devctl features/tt
    ```
*   **Logic**: Git がファイル履歴を追跡できるよう `git mv` を使用。サブディレクトリ `work/devctl/` も自動的に移動される。移動後 `features/tt/work/devctl/` → `features/tt/work/tt/` も個別に `git mv` する。

---

### コンポーネント2: Go ソースコード (features/tt/ 内)

対象ファイル数: 53 ファイル以上。一括 `sed` 置換で対応。

#### [MODIFY] [go.mod](file://features/tt/go.mod)
*   **Description**: モジュールパスの変更
*   **Logic**:
    - `module github.com/axsh/tokotachi/features/devctl` → `module github.com/axsh/tokotachi/features/tt`

#### [MODIFY] 全 .go ファイル — import パス
*   **Description**: 全 Go ファイルの import パスを一括置換
*   **Logic**:
    ```bash
    # import パスの置換
    sed -i 's|github.com/axsh/tokotachi/features/devctl/|github.com/axsh/tokotachi/features/tt/|g'
    ```

#### [MODIFY] [cmd/root.go](file://features/tt/cmd/root.go)
*   **Description**: コマンド名と説明文の変更
*   **Logic**:
    - `Use: "devctl"` → `Use: "tt"`
    - `Short: "Development environment orchestrator"` → `Short: "Tokotachi - Development environment orchestrator"`
    - `Long: "devctl manages feature-level..."` → `Long: "tt (Tokotachi) manages feature-level..."`

#### [MODIFY] [cmd/common.go](file://features/tt/cmd/common.go)
*   **Description**: 環境変数定義リスト、コメント、エラーメッセージの変更
*   **Logic**:
    - `knownEnvVars` 配列内の全 `DEVCTL_*` → `TT_*` (9件)
    - コメント `// knownEnvVars lists all DEVCTL_*` → `// knownEnvVars lists all TT_*`
    - コメント `// CollectEnvVars gathers all DEVCTL_*` → `// CollectEnvVars gathers all TT_*`
    - エラーメッセージ `"cannot be used with devctl commands"` → `"cannot be used with tt commands"`
    - コメント `"cannot be used with devctl."` → `"cannot be used with tt."`

#### [MODIFY] 環境変数定数 (各 editor/*.go, detect/editor.go)
*   **Description**: 環境変数のハードコード定数を変更
*   **Logic**:
    - `editor/ag.go`: `envKeyAG = "DEVCTL_CMD_AG"` → `"TT_CMD_AG"`
    - `editor/cursor.go`: `envKeyCursor = "DEVCTL_CMD_CURSOR"` → `"TT_CMD_CURSOR"`
    - `editor/vscode.go`: `envKeyCode = "DEVCTL_CMD_CODE"` → `"TT_CMD_CODE"`
    - `editor/claude.go`: `envKeyClaude = "DEVCTL_CMD_CLAUDE"` → `"TT_CMD_CLAUDE"`
    - `detect/editor.go`: `EnvKeyEditor = "DEVCTL_EDITOR"` → `"TT_EDITOR"`

#### [MODIFY] `DEVCTL_CMD_GIT` / `DEVCTL_CMD_GH` 参照 (worktree/*.go, cmd/*.go, github/*.go)
*   **Description**: `ResolveCommand("DEVCTL_CMD_GIT", ...)` 等の呼び出しを変更
*   **Logic**:
    - `"DEVCTL_CMD_GIT"` → `"TT_CMD_GIT"` (5箇所)
    - `"DEVCTL_CMD_GH"` → `"TT_CMD_GH"` (3箇所)

#### [MODIFY] `DEVCTL_LIST_*` 参照 (cmd/list.go, listing/listing.go)
*   **Description**: リスト表示関連の環境変数名を変更
*   **Logic**:
    - `"DEVCTL_LIST_WIDTH"` → `"TT_LIST_WIDTH"`
    - `"DEVCTL_LIST_PADDING"` → `"TT_LIST_PADDING"`

#### [MODIFY] scaffold チェックポイント名
*   **Description**: stash メッセージとファイル名の変更
*   **Logic**:
    - `scaffold/scaffold.go`: `"devctl-scaffold-checkpoint"` → `"tt-scaffold-checkpoint"`
    - `scaffold/checkpoint.go`: `".devctl-scaffold-checkpoint"` → `".tt-scaffold-checkpoint"`

#### [MODIFY] 単体テスト内のテストデータ文字列
*   **Description**: テストフィクスチャ内の `"devctl"` 文字列を `"tt"` に変更
*   **Logic** (主要箇所):
    - `state/state_test.go`: フィーチャー名 `"devctl"` → `"tt"`, コンテナ名 `"devctl-devctl"` → `"tt-tt"`, `"proj-devctl"` → `"proj-tt"` 等
    - `report/report_test.go`: 環境変数名 `"DEVCTL_EDITOR"` → `"TT_EDITOR"`, `"DEVCTL_CMD_CURSOR"` → `"TT_CMD_CURSOR"`
    - `editor/editor_test.go`: `"DEVCTL_TEST_*"` → `"TT_TEST_*"`
    - `cmdexec/cmdexec_test.go`: `"DEVCTL_TEST_*"` → `"TT_TEST_*"`
    - `listing/listing.go`: コメント内 `DEVCTL_LIST_WIDTH` → `TT_LIST_WIDTH`

---

### コンポーネント3: 統合テスト (tests/integration-test/)

#### [MODIFY] 統合テストファイルリネーム

| 変更前 | 変更後 |
|---|---|
| `devctl_up_test.go` | `tt_up_test.go` |
| `devctl_up_git_test.go` | `tt_up_git_test.go` |
| `devctl_status_test.go` | `tt_status_test.go` |
| `devctl_down_test.go` | `tt_down_test.go` |
| `devctl_doctor_test.go` | `tt_doctor_test.go` |
| `devctl_scaffold_test.go` | `tt_scaffold_test.go` |
| `devctl_list_code_test.go` | `tt_list_code_test.go` |
| `devctl_env_option_test.go` | `tt_env_option_test.go` |

#### [MODIFY] [helpers_test.go](file://tests/integration-test/helpers_test.go)
*   **Description**: ヘルパー関数名とバイナリパスの変更
*   **Logic**:
    - 関数名: `devctlBinary()` → `ttBinary()`, `runDevctl()` → `runTT()`, `cleanupDevctlDown()` → `cleanupTTDown()`
    - バイナリパス: `"bin", "devctl.exe"` → `"bin", "tt.exe"`, `"bin", "devctl"` → `"bin", "tt"`
    - Docker イメージ名: `"devctl-verify"` → `"tt-verify"`
    - コメント・エラーメッセージ内の `devctl` → `tt`

#### [MODIFY] [docker_build_test.go](file://tests/integration-test/docker_build_test.go)
*   **Description**: テスト関数名、ビルドコンテキストパス、イメージ名の変更
*   **Logic**:
    - `TestDevctlDockerfileBuild` → `TestTtDockerfileBuild`
    - `"features", "devctl"` → `"features", "tt"`
    - `"devctl-verify"` → `"tt-verify"`

#### [MODIFY] 全統合テストファイル — テスト関数名とメッセージ
*   **Description**: `TestDevctl*` → `TestTt*`, `runDevctl` → `runTT`, `cleanupDevctlDown` → `cleanupTTDown`, メッセージ内の `devctl` → `tt`
*   **Logic**: `sed` 一括置換:
    ```bash
    sed -i 's/TestDevctl/TestTt/g; s/devctlBinary/ttBinary/g; s/runDevctl/runTT/g; s/cleanupDevctlDown/cleanupTTDown/g'
    sed -i 's/"devctl /"tt /g; s/"devctl"/"tt"/g'
    ```

---

### コンポーネント4: 設定ファイル

#### [MODIFY] → [RENAME] [tools/manifests/devctl.yaml](file://tools/manifests/devctl.yaml) → `tools/manifests/tt.yaml`
*   **Logic**: `id: devctl` → `id: tt`, `feature_path: features/devctl` → `features/tt`, `binary_name: devctl` → `tt`

#### [MODIFY] [tools/manifests/tools.yaml](file://tools/manifests/tools.yaml)
*   **Logic**: `id: devctl` → `id: tt`, `feature_path: features/devctl` → `features/tt`, `binary_name: devctl` → `tt`

#### [MODIFY] → [RENAME] [packaging/goreleaser/devctl.yaml](file://packaging/goreleaser/devctl.yaml) → `packaging/goreleaser/tt.yaml`
*   **Logic**: `project_name: devctl` → `tt`, `id: devctl` → `id: tt`, `dir: features/devctl` → `features/tt`, `binary: devctl` → `binary: tt`, `dist: dist/devctl` → `dist/tt`

#### [MODIFY] [feature.yaml](file://features/tt/feature.yaml)
*   **Logic**: `name: devctl` → `name: tt`, `expose_as: devctl` → `expose_as: tt`

#### [MODIFY] → [RENAME] インストーラーテンプレート
*   `tools/installers/homebrew/Formula/devctl.rb.tmpl` → `tt.rb.tmpl`
    - `class Devctl < Formula` → `class Tt < Formula`
    - すべての `devctl_` → `tt_` (アーカイブ名)
    - `bin.install "devctl"` → `bin.install "tt"`
*   `tools/installers/scoop/devctl.json.tmpl` → `tt.json.tmpl`
    - `devctl_windows_amd64.zip` → `tt_windows_amd64.zip`
    - `"bin": "devctl.exe"` → `"bin": "tt.exe"`

#### [MODIFY] [features/tt/work/tt/test-001.state.yaml](file://features/tt/work/tt/test-001.state.yaml) ※ リネーム後
*   **Logic**: `feature: devctl` → `feature: tt`

---

### コンポーネント5: シェルスクリプト

#### [MODIFY] [scripts/process/build.sh](file://scripts/process/build.sh)
*   **Description**: 関数名、変数名、パス、メッセージの変更
*   **Logic**:
    - 関数名: `build_devctl()` → `build_tt()`
    - 変数名: `devctl_dir` → `tt_dir`
    - パス: `features/devctl` → `features/tt`, `bin/devctl` → `bin/tt`
    - メッセージ: `"devctl (Go): Build"` → `"tt (Go): Build"` 等
    - 呼び出し: `build_devctl` → `build_tt`

#### [MODIFY] [scripts/dist/dev.sh](file://scripts/dist/dev.sh)
*   **Logic**:
    - 変数: `DEVCTL=` → `TT_BIN=`, `$DEVCTL` → `$TT_BIN`
    - パス: `bin/devctl` → `bin/tt`
    - メッセージ: `"devctl not found"` → `"tt not found"`
    - 使用例コメント: `devctl` → `tt`

#### [MODIFY] scripts/dist/ の他のスクリプト
*   `build.sh`, `github-upload.sh`, `install-tools.sh`, `publish.sh`, `release.sh`, `_lib.sh`
*   **Logic**: 各ファイルのコメント・使用例内の `devctl` → `tt`

---

### コンポーネント6: ドキュメント (prompts/ 除外)

#### [MODIFY] [README.md](file://README.md)
*   **Description**: CLIツール名、コマンド使用例、インストール手順、リンク先の全面変更
*   **Logic**: `devctl` → `tt` の `sed` 置換。ただし以下を手動調整:
    - `**devctl CLI**` → `**tt CLI**`
    - セクション見出し `### devctl —` → `### tt —`
    - 説明文本体: `devctl is a CLI tool` → `tt (Tokotachi) is a CLI tool`
    - インストール手順のファイル名: `devctl_linux_amd64.tar.gz` → `tt_linux_amd64.tar.gz` 等

#### [MODIFY] → [RENAME] [releases/changelogs/devctl.md](file://releases/changelogs/devctl.md) → `releases/changelogs/tt.md`
*   **Logic**: `# devctl Changelog` → `# tt Changelog`

#### [MODIFY] [scripts/dist/README.md](file://scripts/dist/README.md)
*   **Logic**: 使用例内の `devctl` → `tt`

---

## Step-by-Step Implementation Guide

> [!IMPORTANT]
> この作業は大量のファイルに対する一括置換が主体であるため、TDD (テストを先に書く) ではなく、「一括リネーム → ビルド確認」のアプローチが最適です。テスト自体もリネーム対象のため。

### Step 1: ディレクトリの物理リネーム

1. `git mv features/devctl features/tt`
2. `git mv features/tt/work/devctl features/tt/work/tt` (存在する場合)

### Step 2: 設定・テンプレートファイルのリネーム

1. `git mv tools/manifests/devctl.yaml tools/manifests/tt.yaml`
2. `git mv packaging/goreleaser/devctl.yaml packaging/goreleaser/tt.yaml`
3. `git mv tools/installers/homebrew/Formula/devctl.rb.tmpl tools/installers/homebrew/Formula/tt.rb.tmpl`
4. `git mv tools/installers/scoop/devctl.json.tmpl tools/installers/scoop/tt.json.tmpl`
5. `git mv releases/changelogs/devctl.md releases/changelogs/tt.md`

### Step 3: Go ソースコード一括置換 (features/tt/)

`find + sed` による一括置換を以下の順序で実行:

1. **go.mod**: モジュールパス変更
    ```bash
    sed -i 's|github.com/axsh/tokotachi/features/devctl|github.com/axsh/tokotachi/features/tt|g' features/tt/go.mod
    ```

2. **全 .go ファイル import パス**:
    ```bash
    find features/tt -name '*.go' -exec sed -i 's|github.com/axsh/tokotachi/features/devctl/|github.com/axsh/tokotachi/features/tt/|g' {} +
    ```

3. **環境変数プレフィックス DEVCTL_ → TT_**:
    ```bash
    find features/tt -name '*.go' -exec sed -i 's/DEVCTL_/TT_/g' {} +
    ```

4. **コマンド名 "devctl" → "tt"** (文字列リテラル):
    ```bash
    find features/tt -name '*.go' -exec sed -i 's/"devctl"/"tt"/g' {} +
    ```

5. **コンテナ名プレフィックス**:
    ```bash
    find features/tt -name '*.go' -exec sed -i "s/devctl-/tt-/g" {} +
    ```

6. **Long description の手動更新** (`cmd/root.go`):
    - `"devctl manages feature-level..."` → `"tt (Tokotachi) manages feature-level..."`
    - `Short: "Development environment orchestrator"` → `Short: "Tokotachi - Development environment orchestrator"`

7. **コメント内の devctl** (残り):
    ```bash
    find features/tt -name '*.go' -exec sed -i 's/devctl/tt/g' {} +
    ```
    ※ 上記の順序により、先に固有パターン (import, env, string) を処理してからコメントの残りを処理

8. **PascalCase 識別子**: `Devctl` → `Tt` (もし存在する場合)

### Step 4: 統合テストのリネームと置換 (tests/integration-test/)

1. ファイルリネーム:
    ```bash
    cd tests/integration-test
    git mv devctl_up_test.go tt_up_test.go
    git mv devctl_up_git_test.go tt_up_git_test.go
    git mv devctl_status_test.go tt_status_test.go
    git mv devctl_down_test.go tt_down_test.go
    git mv devctl_doctor_test.go tt_doctor_test.go
    git mv devctl_scaffold_test.go tt_scaffold_test.go
    git mv devctl_list_code_test.go tt_list_code_test.go
    git mv devctl_env_option_test.go tt_env_option_test.go
    ```

2. 全統合テストファイル内の置換:
    ```bash
    find tests/integration-test -name '*.go' -exec sed -i \
      's/DEVCTL_/TT_/g; s/TestDevctl/TestTt/g; s/devctlBinary/ttBinary/g; s/runDevctl/runTT/g; s/cleanupDevctlDown/cleanupTTDown/g' {} +
    find tests/integration-test -name '*.go' -exec sed -i \
      's|"devctl"|"tt"|g; s|devctl-|tt-|g; s|"devctl |"tt |g; s|features/devctl|features/tt|g; s|bin/devctl|bin/tt|g' {} +
    find tests/integration-test -name '*.go' -exec sed -i 's/devctl/tt/g' {} +
    ```

### Step 5: 設定ファイルの内容置換

```bash
# YAML マニフェスト
sed -i 's/devctl/tt/g' tools/manifests/tt.yaml tools/manifests/tools.yaml
sed -i 's/devctl/tt/g; s/Devctl/Tt/g' packaging/goreleaser/tt.yaml

# feature.yaml (移動後)
sed -i 's/devctl/tt/g' features/tt/feature.yaml

# テスト状態ファイル
sed -i 's/devctl/tt/g' features/tt/work/tt/test-001.state.yaml

# インストーラーテンプレート
sed -i 's/devctl/tt/g; s/Devctl/Tt/g' tools/installers/homebrew/Formula/tt.rb.tmpl
sed -i 's/devctl/tt/g' tools/installers/scoop/tt.json.tmpl
```

### Step 6: シェルスクリプトの置換

```bash
# build.sh — 関数名・変数名含む
sed -i 's/build_devctl/build_tt/g; s/devctl_dir/tt_dir/g; s|features/devctl|features/tt|g; s|bin/devctl|bin/tt|g; s/devctl/tt/g' scripts/process/build.sh

# dist/ 系スクリプト
find scripts/dist -name '*.sh' -exec sed -i 's/DEVCTL/TT_BIN/g; s|bin/devctl|bin/tt|g; s/devctl/tt/g' {} +
```

> [!WARNING]
> `scripts/dist/dev.sh` の変数名 `DEVCTL` は他の `devctl` 置換と衝突しないよう、先に `DEVCTL=` → `TT_BIN=` の個別置換を行ってから一般置換する。

### Step 7: ドキュメントの置換

```bash
# README.md
sed -i 's|features/devctl|features/tt|g; s/devctl/tt/g' README.md

# 個別手動調整 (README.md)
# - "tt CLI" → そのまま OK
# - "tt is a CLI tool" → "tt (Tokotachi) is a CLI tool" に手動調整
# - セクション見出しの体裁確認

# changelogs
sed -i 's/devctl/tt/g' releases/changelogs/tt.md

# scripts/dist/README.md
sed -i 's/devctl/tt/g' scripts/dist/README.md
```

### Step 8: go mod tidy

```bash
cd features/tt && go mod tidy
```

### Step 9: 残存チェックとビルド検証

```bash
# devctl が残っていないことを確認 (prompts/, .git/ 除外)
grep -rI "devctl" . --include='*.go' --include='*.sh' --include='*.yaml' --include='*.yml' \
  --include='*.md' --include='*.tmpl' --include='*.rb' \
  --exclude-dir=prompts --exclude-dir=.git --exclude-dir=.agent | head -50
```

残存があれば個別修正。

---

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    - ビルドが成功し、`bin/tt` (Windows: `bin/tt.exe`) が生成されること
    - 全単体テストがパスすること

2. **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```
    - 全統合テストがパスすること

3. **残存チェック**:
    ```bash
    grep -rI "devctl" . --include='*.go' --include='*.sh' --include='*.yaml' --include='*.yml' \
      --include='*.md' --include='*.tmpl' --include='*.rb' \
      --exclude-dir=prompts --exclude-dir=.git --exclude-dir=.agent --exclude-dir=work
    ```
    - 出力が 0 件であること

4. **CLI 出力確認**:
    ```bash
    ./bin/tt --help
    ```
    - 出力に `Tokotachi - Development environment orchestrator` が含まれること
    - 出力に `devctl` が含まれないこと

---

## Documentation

#### [MODIFY] [README.md](file://README.md)
*   **更新内容**: 全 `devctl` 参照を `tt` に変更。セクション見出し、使用例、インストール手順を更新。

#### [MODIFY] [releases/changelogs/tt.md](file://releases/changelogs/tt.md)
*   **更新内容**: タイトルを `# tt Changelog` に変更。

#### [MODIFY] [scripts/dist/README.md](file://scripts/dist/README.md)
*   **更新内容**: 使用例内の `devctl` → `tt` を変更。
