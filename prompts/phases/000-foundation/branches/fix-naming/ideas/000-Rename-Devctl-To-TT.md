# CLIツール名の変更: `devctl` → `tt`

## 背景 (Background)

現在、tokotachi プロジェクトのCLIツールは `devctl` という名前で実装されている。この名前は汎用的すぎて、プロジェクト固有のアイデンティティを反映していない。

`tt` は **Tokotachi** の **Toko** と **Tachi** の頭文字を取った略称であり、以下のメリットがある:

- **短くて入力が速い**: `devctl up` → `tt up` (4文字削減/コマンド)
- **プロジェクト固有**: tokotachi プロジェクトとの結びつきが明確
- **ブランディング**: ツールの説明文を `Tokotachi - Development environment orchestrator` とすることで、プロジェクト名を前面に出す

> [!IMPORTANT]
> `prompts/` 配下のドキュメントは変更対象外です。

---

## 要件 (Requirements)

### 必須要件

1. **プログラム内の文字列リテラル**: コマンド名 `"devctl"` → `"tt"`、説明文の更新
2. **構造体名・関数名**: `Devctl` を含む Go の識別子を `TT` / `Tt` に変更
3. **テスト名**: `TestDevctl*` → `TestTt*`、`devctlBinary` → `ttBinary` 等
4. **環境変数プレフィックス**: `DEVCTL_*` → `TT_*`
5. **ファイル名**: `devctl_*.go` → `tt_*.go`、`devctl.yaml` → `tt.yaml` 等
6. **ディレクトリ名**: `features/devctl/` → `features/tt/`
7. **ドキュメント** (`prompts/` 配下は除外): README.md, changelogs, scripts/dist/README.md
8. **設定ファイル**: YAML マニフェスト、GoReleaser 設定、インストーラーテンプレート

### 任意要件

- Go モジュールパスの変更: `github.com/axsh/tokotachi/features/devctl` → `github.com/axsh/tokotachi/features/tt`
  - これは実質的にディレクトリ名変更に連動する

---

## 変更仕様 — 全件一覧

以下に、変更対象をカテゴリ別に網羅的に列挙する。

---

### カテゴリ1: Goソースコード — 文字列リテラル

#### 1-1. コマンド名・説明文 (`cmd/root.go`)

| 行 | 変更前 | 変更後 |
|---|---|---|
| L18 | `Use: "devctl"` | `Use: "tt"` |
| L19 | `Short: "Development environment orchestrator"` | `Short: "Tokotachi - Development environment orchestrator"` |
| L20 | `Long: "devctl manages feature-level..."` | `Long: "tt (Tokotachi) manages feature-level..."` |

#### 1-2. 環境変数名 (`DEVCTL_*` → `TT_*`)

全10種の環境変数を変更:

| 変更前 | 変更後 | 使用ファイル |
|---|---|---|
| `DEVCTL_EDITOR` | `TT_EDITOR` | `detect/editor.go`, `cmd/common.go`, `report/report_test.go` |
| `DEVCTL_CMD_CODE` | `TT_CMD_CODE` | `editor/vscode.go`, `cmd/common.go` |
| `DEVCTL_CMD_CURSOR` | `TT_CMD_CURSOR` | `editor/cursor.go`, `cmd/common.go`, `report/report_test.go` |
| `DEVCTL_CMD_AG` | `TT_CMD_AG` | `editor/ag.go`, `cmd/common.go` |
| `DEVCTL_CMD_CLAUDE` | `TT_CMD_CLAUDE` | `editor/claude.go`, `cmd/common.go` |
| `DEVCTL_CMD_GIT` | `TT_CMD_GIT` | `worktree/worktree.go`, `cmd/update_code_status.go`, `cmd/list.go`, `cmd/common.go` |
| `DEVCTL_CMD_GH` | `TT_CMD_GH` | `github/github.go`, `cmd/update_code_status.go`, `cmd/common.go` |
| `DEVCTL_LIST_WIDTH` | `TT_LIST_WIDTH` | `cmd/list.go`, `listing/listing.go`, `cmd/common.go` |
| `DEVCTL_LIST_PADDING` | `TT_LIST_PADDING` | `cmd/list.go`, `listing/listing.go`, `cmd/common.go` |
| (テスト用) `DEVCTL_TEST_*` | `TT_TEST_*` | `editor/editor_test.go`, `cmdexec/cmdexec_test.go` |

#### 1-3. コンテナ名プレフィックス

| 使用箇所 | 変更前 | 変更後 |
|---|---|---|
| `cmd/common.go` (ContainerName 生成) | `"devctl-" + feature` | `"tt-" + feature` |
| `state/state_test.go` (テストデータ) | `"devctl-devctl"`, `"proj-devctl"` | `"tt-tt"`, `"proj-tt"` |
| 統合テスト (`devctl_up_git_test.go`) | `"devctl-" + featureName` | `"tt-" + featureName` |

#### 1-4. Docker イメージ名

| 使用箇所 | 変更前 | 変更後 |
|---|---|---|
| `helpers_test.go` L150 | `"devctl-verify"` | `"tt-verify"` |
| `docker_build_test.go` L35 | `"devctl-verify"` | `"tt-verify"` |

#### 1-5. Scaffold チェックポイント

| 使用箇所 | 変更前 | 変更後 |
|---|---|---|
| `scaffold/scaffold.go` L160 | `"devctl-scaffold-checkpoint"` | `"tt-scaffold-checkpoint"` |
| `scaffold/checkpoint.go` L37 | `".devctl-scaffold-checkpoint"` | `".tt-scaffold-checkpoint"` |

#### 1-6. ビルド出力パス

| 使用箇所 | 変更前 | 変更後 |
|---|---|---|
| `helpers_test.go` L42 | `"bin", "devctl.exe"` | `"bin", "tt.exe"` |
| `helpers_test.go` L43 | `"bin", "devctl"` | `"bin", "tt"` |

#### 1-7. コメント・エラーメッセージ

`"devctl"` を含むコメントやエラーメッセージをすべて `"tt"` に変更する。対象:

- `cmd/common.go` — `knownEnvVars` コメント: `"all DEVCTL_*"` → `"all TT_*"`
- `helpers_test.go` — `"devctl binary not found"` → `"tt binary not found"`
- 統合テスト全般 — `"devctl up failed"` → `"tt up failed"` 等

---

### カテゴリ2: Goソースコード — 識別子 (構造体名・関数名・テスト名)

#### 2-1. 統合テスト関数名 (`tests/integration-test/`)

| ファイル | 変更前 | 変更後 |
|---|---|---|
| `helpers_test.go` | `devctlBinary()` | `ttBinary()` |
| `helpers_test.go` | `runDevctl()` | `runTT()` |
| `helpers_test.go` | `cleanupDevctlDown()` | `cleanupTTDown()` |
| `devctl_up_test.go` | `TestDevctlUpStartsContainer` | `TestTtUpStartsContainer` |
| `devctl_up_test.go` | `TestDevctlUpIdempotent` | `TestTtUpIdempotent` |
| `devctl_up_git_test.go` | `TestDevctlUpGitWorktree` | `TestTtUpGitWorktree` |
| `devctl_status_test.go` | `TestDevctlStatusWhenRunning` | `TestTtStatusWhenRunning` |
| `docker_build_test.go` | `TestDevctlDockerfileBuild` | `TestTtDockerfileBuild` |
| `devctl_down_test.go` | 同様のパターン | 同様に変更 |
| `devctl_doctor_test.go` | 同様のパターン | 同様に変更 |
| `devctl_scaffold_test.go` | 同様のパターン | 同様に変更 |
| `devctl_list_code_test.go` | 同様のパターン | 同様に変更 |
| `devctl_env_option_test.go` | 同様のパターン | 同様に変更 |

---

### カテゴリ3: ファイル名・ディレクトリ名の変更

#### 3-1. ディレクトリ

| 変更前 | 変更後 |
|---|---|
| `features/devctl/` | `features/tt/` |
| `features/devctl/work/devctl/` | `features/tt/work/tt/` |

#### 3-2. テストファイル名 (`tests/integration-test/`)

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
| `docker_build_test.go` | (変更なし — `devctl` を含まないため) |

#### 3-3. 設定・マニフェストファイル名

| 変更前 | 変更後 |
|---|---|
| `tools/manifests/devctl.yaml` | `tools/manifests/tt.yaml` |
| `packaging/goreleaser/devctl.yaml` | `packaging/goreleaser/tt.yaml` |
| `tools/installers/homebrew/Formula/devctl.rb.tmpl` | `tools/installers/homebrew/Formula/tt.rb.tmpl` |
| `tools/installers/scoop/devctl.json.tmpl` | `tools/installers/scoop/tt.json.tmpl` |
| `releases/changelogs/devctl.md` | `releases/changelogs/tt.md` |

---

### カテゴリ4: 設定ファイル — 内容の変更

#### 4-1. `go.mod`

| 変更前 | 変更後 |
|---|---|
| `module github.com/axsh/tokotachi/features/devctl` | `module github.com/axsh/tokotachi/features/tt` |

> [!IMPORTANT]
> `go.mod` の変更に伴い、すべてのGoソースファイル内の `import` パスも連動して変更が必要。

#### 4-2. `feature.yaml` (`features/tt/feature.yaml`)

| 行 | 変更前 | 変更後 |
|---|---|---|
| L1 | `name: devctl` | `name: tt` |
| L8 | `expose_as: devctl` | `expose_as: tt` |

#### 4-3. `tools/manifests/tools.yaml`

| 行 | 変更前 | 変更後 |
|---|---|---|
| L2 | `id: devctl` | `id: tt` |
| L3 | `feature_path: features/devctl` | `feature_path: features/tt` |
| L5 | `binary_name: devctl` | `binary_name: tt` |

#### 4-4. `tools/manifests/tt.yaml` (旧: `devctl.yaml`)

| 行 | 変更前 | 変更後 |
|---|---|---|
| L1 | `id: devctl` | `id: tt` |
| L2 | `feature_path: features/devctl` | `feature_path: features/tt` |
| L4 | `binary_name: devctl` | `binary_name: tt` |

#### 4-5. `packaging/goreleaser/tt.yaml` (旧: `devctl.yaml`)

| 行 | 変更前 | 変更後 |
|---|---|---|
| L1 | `# ...configuration for devctl` | `# ...configuration for tt` |
| L3 | `project_name: devctl` | `project_name: tt` |
| L7 | `id: devctl` | `id: tt` |
| L8 | `dir: features/devctl` | `dir: features/tt` |
| L9 | `main: ./cmd/devctl` | `main: ./cmd/tt`  ※ 該当ディレクトリ構造に依存 |
| L10 | `binary: devctl` | `binary: tt` |
| L24 | `dist: dist/devctl` | `dist: dist/tt` |

#### 4-6. インストーラーテンプレート

**Homebrew** (`tools/installers/homebrew/Formula/tt.rb.tmpl`):
- `class Devctl < Formula` → `class Tt < Formula`
- `devctl_darwin_arm64.tar.gz` → `tt_darwin_arm64.tar.gz`
- `devctl_darwin_amd64.tar.gz` → `tt_darwin_amd64.tar.gz`
- `devctl_linux_arm64.tar.gz` → `tt_linux_arm64.tar.gz`
- `devctl_linux_amd64.tar.gz` → `tt_linux_amd64.tar.gz`
- `bin.install "devctl"` → `bin.install "tt"`
- `system "#{bin}/devctl", "version"` → `system "#{bin}/tt", "version"`

**Scoop** (`tools/installers/scoop/tt.json.tmpl`):
- `devctl_windows_amd64.zip` → `tt_windows_amd64.zip`
- `"bin": "devctl.exe"` → `"bin": "tt.exe"`

#### 4-7. `features/tt/work/tt/test-001.state.yaml`

| 変更前 | 変更後 |
|---|---|
| `feature: devctl` | `feature: tt` |

---

### カテゴリ5: シェルスクリプト

#### 5-1. `scripts/process/build.sh`

| 行 | 変更前 | 変更後 |
|---|---|---|
| L183 | `# devctl Build & Unit Test` | `# tt Build & Unit Test` |
| L185 | `build_devctl()` | `build_tt()` |
| L186 | `step "devctl (Go): Build & Unit Test"` | `step "tt (Go): Build & Unit Test"` |
| L188 | `local devctl_dir="$PROJECT_ROOT/features/devctl"` | `local tt_dir="$PROJECT_ROOT/features/tt"` |
| L190-191 | `devctl_dir`, `devctl build` の参照 | `tt_dir`, `tt build` に変更 |
| L195 | `cd "$devctl_dir"` | `cd "$tt_dir"` |
| L198-213 | メッセージ内の `devctl` | `tt` に変更 |
| L199 | `go build -o "$PROJECT_ROOT/bin/devctl"` | `go build -o "$PROJECT_ROOT/bin/tt"` |
| L237-238 | `build_devctl` 呼び出し | `build_tt` に変更 |

#### 5-2. `scripts/dist/*.sh`

各スクリプト内の `devctl` 引数例とメッセージを `tt` に変更:
- `build.sh` — コメント、使用例
- `dev.sh` — 変数名 `DEVCTL` → `TT_BIN`、パスの `devctl` → `tt`、メッセージ
- `github-upload.sh` — 使用例、コメント
- `install-tools.sh` — 使用例
- `publish.sh` — 使用例、Homebrew/Scoop パス案内
- `release.sh` — 使用例
- `_lib.sh` — コメント内の例

---

### カテゴリ6: ドキュメント (`prompts/` 配下は除外)

#### 6-1. `README.md` (プロジェクトルート)

- L22: `**devctl CLI**` → `**tt CLI**`
- L32: ディレクトリツリー内 `devctl/` → `tt/`
- L55: セクション見出し `### devctl —` → `### tt —`
- L57: 説明文内 `devctl is a CLI tool` → `tt (Tokotachi) is a CLI tool`
- L60-73: コマンド使用例 `devctl up` → `tt up` 等 (全14行)
- L90: リンク `features/devctl/README.md` → `features/tt/README.md`
- L102-128: インストール手順内の `devctl` → `tt` (ファイル名、バイナリ名)
- L144-148: ビルド手順内の `devctl` → `tt`

#### 6-2. `releases/changelogs/tt.md` (旧: `devctl.md`)

- L1: `# devctl Changelog` → `# tt Changelog`

#### 6-3. `scripts/dist/README.md`

- 全体的に使用例の `devctl` → `tt`
  - L39, L42, L45, L55, L58, L66, L69, L76

---

### カテゴリ7: Go `import` パスの全面変更

`go.mod` のモジュール名変更に伴い、全Goファイルの内部パッケージ `import` を更新:

```
github.com/axsh/tokotachi/features/devctl/...
  → github.com/axsh/tokotachi/features/tt/...
```

対象パッケージ (確認済み):
- `features/devctl/cmd`
- `features/devctl/internal/action`
- `features/devctl/internal/cmdexec`
- `features/devctl/internal/codestatus`
- `features/devctl/internal/detect`
- `features/devctl/internal/doctor`
- `features/devctl/internal/editor`
- `features/devctl/internal/github`
- `features/devctl/internal/listing`
- `features/devctl/internal/log`
- `features/devctl/internal/matrix`
- `features/devctl/internal/plan`
- `features/devctl/internal/report`
- `features/devctl/internal/resolve`
- `features/devctl/internal/scaffold`
- `features/devctl/internal/state`
- `features/devctl/internal/worktree`

---

## 変更しないもの (除外リスト)

| 対象 | 理由 |
|---|---|
| `prompts/` 配下のすべてのファイル | ユーザー指示による除外 |
| `.devrc.yaml` | `devctl` を含まない |
| `go.sum` | `go mod tidy` で自動再生成 |
| `.git/` 配下 | Git内部メタデータ |
| テストデータ内のフィーチャー名 `"devctl"` | テストデータのフィーチャー名として使用されている場合はコンテキストに応じて `"tt"` に変更する（カテゴリ1-3で対応済み） |

---

## 検証シナリオ (Verification Scenarios)

1. ディレクトリ `features/devctl/` を `features/tt/` にリネーム
2. `go.mod` のモジュール名を変更し、全 `import` パスを更新
3. すべての文字列リテラル・識別子・環境変数を置換
4. `go build` が成功し、バイナリが `bin/tt` として出力される
5. `bin/tt --help` の出力に `Tokotachi - Development environment orchestrator` が含まれる
6. `bin/tt --help` の出力に `devctl` が含まれない
7. 全単体テストが `go test ./...` でパスする
8. grep -r で `devctl` がソースコード上に残存しないことを確認 (`prompts/` 配下、`.git/` 配下は除外)

---

## テスト項目 (Testing for the Requirements)

### ビルドと単体テスト

```bash
./scripts/process/build.sh
```

- ビルドが成功し、`bin/tt` (Windows: `bin/tt.exe`) が生成されること
- 全単体テストがパスすること

### 統合テスト

```bash
./scripts/process/integration_test.sh
```

- 既存の統合テストがすべてパスすること（テスト名は変更されているが機能は同一）

### 残存チェック

```bash
# prompts/ と .git/ を除外して devctl が残っていないことを確認
grep -rI --include='*.go' --include='*.sh' --include='*.yaml' --include='*.yml' \
  --include='*.md' --include='*.tmpl' --include='*.rb' \
  "devctl" . --exclude-dir=prompts --exclude-dir=.git
```

期待結果: 出力なし（0件）
