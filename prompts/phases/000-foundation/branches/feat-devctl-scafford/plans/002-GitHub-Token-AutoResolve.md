# 002-GitHub-Token-AutoResolve

> **Source Specification**: [003-GitHub-Token-AutoResolve.md](file://prompts/phases/000-foundation/ideas/feat-devctl-scafford/003-GitHub-Token-AutoResolve.md)

## Goal Description

GitHub Token の自動取得と、GitHub 操作の共通パッケージ化を行う。`internal/github` パッケージを新設し、現在 `scaffold/downloader.go` に分散している HTTP アクセスと `action/pr.go` の `gh` CLI 操作を集約する。高次の操作インターフェイスを公開し、内部実装を意識させない設計とする。

## User Review Required

> [!IMPORTANT]
> `DownloadedFile` を `github` パッケージに移動し、`scaffold` パッケージでは型エイリアス (`type DownloadedFile = github.DownloadedFile`) で互換性を維持します。Go の型エイリアスは完全な互換性があるため、既存コードの変更は不要ですが、設計上の判断としてレビューをお願いします。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. `gh auth token` によるフォールバック取得 | Proposed Changes > `github.go` - `resolveToken()` |
| 2. `GITHUB_TOKEN` 環境変数の優先 | Proposed Changes > `github.go` - `resolveToken()` |
| 3. エラーハンドリング (`gh` なし/未認証時) | Proposed Changes > `github.go` - `resolveToken()` |
| 4. 既存テストへの影響なし | Verification Plan |
| 5. GitHub 操作の共通パッケージ化 | Proposed Changes > `github.go` (新規), `downloader.go` (削除), `pr.go` (修正) |
| 6. verbose ログ (任意) | 対象外: スコープ外 |

## Proposed Changes

### github パッケージ (新規)

#### [NEW] [github_test.go](file://features/devctl/internal/github/github_test.go)

*   **Description**: `github.Client` のテーブル駆動テストを作成。既存 `downloader_test.go` のテストを移行し、トークン解決のテストを追加する
*   **Technical Design**:

    ```go
    package github

    func TestResolveToken(t *testing.T) {
        tests := []struct {
            name     string
            envToken string
            wantEnv  bool
        }{...}
    }

    func TestNewClient(t *testing.T) {
        tests := []struct {
            name    string
            url     string
            owner   string
            repo    string
            wantErr bool
        }{...}
    }

    func TestClient_FetchFile(t *testing.T) { ... }
    func TestClient_FetchFile_NotFound(t *testing.T) { ... }
    func TestClient_FetchDirectory(t *testing.T) { ... }
    ```

*   **テストケース設計**:
    *   `TestResolveToken`:
        | ケース | `GITHUB_TOKEN` | 期待 |
        |---|---|---|
        | 環境変数設定済み | `"ghp_test123"` | `"ghp_test123"` を返す |
        | 環境変数未設定 | `""` | パニックしない (`assert.NotPanics`) |
    *   `TestNewClient`: 既存 `TestNewGitHubDownloader` と同等のケース
    *   `TestClient_FetchFile` / `FetchFile_NotFound` / `FetchDirectory`: 既存テストを `httptest.NewServer` パターンのまま移行

---

#### [NEW] [github.go](file://features/devctl/internal/github/github.go)

*   **Description**: GitHub 操作を集約するパッケージ。HTTP API アクセスと `gh` CLI 操作の両方を内包する
*   **Technical Design**:

    ```go
    package github

    // DownloadedFile represents a file downloaded from a GitHub repository.
    type DownloadedFile struct {
        RelativePath string
        Content      []byte
    }

    // Client は GitHub 操作を集約する。
    type Client struct {
        Owner      string
        Repo       string
        Branch     string
        Client     *http.Client
        BaseURL    string
        token      string
        cmdRunner  CmdRunner // PR操作用 (optional)
    }

    // CmdRunner は gh CLI のコマンド実行を抽象化するインターフェイス。
    // cmdexec.Runner が実装する。
    type CmdRunner interface {
        RunInteractiveWithOpts(opts RunOption, name string, args ...string) error
    }

    // RunOption は CmdRunner に渡すオプション（cmdexec.RunOption と同じ構造）。
    type RunOption struct {
        Dir string
    }

    // NewClient はリポジトリ URL から Client を作成する。
    // トークンは自動解決される（GITHUB_TOKEN → gh auth token → 空）
    func NewClient(repoURL string, opts ...ClientOption) (*Client, error)

    // ClientOption は Client のオプション設定関数。
    type ClientOption func(*Client)

    // WithCmdRunner は gh CLI 操作用の CmdRunner を設定する。
    func WithCmdRunner(r CmdRunner) ClientOption

    // WithBaseURL はテスト用に API Base URL を上書きする。
    func WithBaseURL(url string) ClientOption

    // WithHTTPClient はテスト用に HTTP Client を上書きする。
    func WithHTTPClient(c *http.Client) ClientOption
    ```

*   **Logic (`resolveToken` - 非公開関数)**:
    1. `os.Getenv("GITHUB_TOKEN")` を呼び出す
    2. 非空であればその値を返す
    3. 空の場合、`exec.Command("gh", "auth", "token")` を実行し `cmd.Output()` で取得
    4. エラーなしの場合、`strings.TrimSpace(string(output))` を返す
    5. エラーの場合、空文字列を返す

*   **Logic (`FetchFile` - `scaffold/downloader.go` から移動)**:
    1. 既存の `GitHubDownloader.FetchFile` と同一ロジック
    2. GitHub Contents API (`/repos/{owner}/{repo}/contents/{path}?ref={branch}`) へ GET
    3. レスポンスの base64 エンコードされた `content` をデコードして返す

*   **Logic (`FetchDirectory` - `scaffold/downloader.go` から移動)**:
    1. 既存の `GitHubDownloader.FetchDirectory` / `fetchDirectoryRecursive` と同一ロジック
    2. ディレクトリ一覧を取得し、ファイルは `FetchFile` で、サブディレクトリは再帰で処理

*   **Logic (`CreatePR`)**:
    1. `cmdexec.ResolveCommand("DEVCTL_CMD_GH", "gh")` でコマンドパスを解決
    2. `c.cmdRunner.RunInteractiveWithOpts(RunOption{Dir: workDir}, ghCmd, "pr", "create")` を実行
    3. `cmdRunner` が未設定なら `fmt.Errorf("CmdRunner is required for PR operations")` を返す

---

### scaffold パッケージ (修正)

#### [DELETE] [downloader.go](file://features/devctl/internal/scaffold/downloader.go)

*   **Description**: GitHub HTTP アクセスのコードは `github.go` に移動するため削除する

---

#### [DELETE] [downloader_test.go](file://features/devctl/internal/scaffold/downloader_test.go)

*   **Description**: テストは `github/github_test.go` に移行するため削除する

---

#### [NEW] [downloader.go](file://features/devctl/internal/scaffold/downloader.go)

*   **Description**: `Downloader` インターフェイスと `DownloadedFile` 型エイリアスのみを残す薄いファイル
*   **Technical Design**:

    ```go
    package scaffold

    import "github.com/axsh/tokotachi/features/devctl/internal/github"

    // DownloadedFile is a type alias for github.DownloadedFile.
    // This maintains backward compatibility with existing scaffold code.
    type DownloadedFile = github.DownloadedFile

    // Downloader is the interface for fetching files from a template repository.
    // github.Client implements this interface.
    type Downloader interface {
        FetchFile(path string) ([]byte, error)
        FetchDirectory(path string) ([]DownloadedFile, error)
    }
    ```

---

#### [MODIFY] [scaffold.go](file://features/devctl/internal/scaffold/scaffold.go)

*   **Description**: `NewGitHubDownloader` の呼び出しを `github.NewClient` に変更する
*   **Technical Design**:
    *   import に `"github.com/axsh/tokotachi/features/devctl/internal/github"` を追加
*   **Logic**:
    *   `scaffold.Run` (L46): `NewGitHubDownloader(opts.RepoURL)` → `github.NewClient(opts.RepoURL)`
    *   `scaffold.Apply` (L174): `NewGitHubDownloader(opts.RepoURL)` → `github.NewClient(opts.RepoURL)`
    *   `scaffold.List` (L292): `NewGitHubDownloader(repoURL)` → `github.NewClient(repoURL)`
    *   戻り値の型は `*github.Client` だが、`Downloader` インターフェイスを満たすため既存コードに影響なし

---

### action パッケージ (修正)

#### [MODIFY] [pr.go](file://features/devctl/internal/action/pr.go)

*   **Description**: PR 作成を `github.Client.CreatePR` に委譲する
*   **Technical Design**:
    *   import を `"github.com/axsh/tokotachi/features/devctl/internal/github"` に変更（`cmdexec` は不要になる）
*   **Logic**:
    *   `Runner.PR` メソッド内で `github.NewClient` を呼び、`client.CreatePR(worktreePath)` に差し替え
    *   `NewClient` に `github.WithCmdRunner(r.CmdRunner)` を渡す
    *   注: `CmdRunner` は `cmdexec.Runner` だが、`github.CmdRunner` インターフェイスを満たす必要がある。`cmdexec.Runner` が同じシグネチャの `RunInteractiveWithOpts` を持つため、アダプタは不要（Go のダックタイピング）

    ```go
    func (r *Runner) PR(worktreePath string) error {
        r.Logger.Info("Creating PR from %s...", worktreePath)
        client, err := github.NewClient("", github.WithCmdRunner(r.CmdRunner))
        if err != nil {
            return fmt.Errorf("github client creation failed: %w", err)
        }
        if err := client.CreatePR(worktreePath); err != nil {
            return fmt.Errorf("gh pr create failed: %w", err)
        }
        return nil
    }
    ```

    *   注: `NewClient("")` は PR 操作のみの場合、リポジトリ URL が不要。URL 空の場合は owner/repo のパースをスキップする設計とする

## Step-by-Step Implementation Guide

- [ ] **Step 1: テスト作成 (TDD - Red)**
    - `features/devctl/internal/github/github_test.go` を新規作成
    - `TestResolveToken`, `TestNewClient`, `TestClient_FetchFile`, `TestClient_FetchFile_NotFound`, `TestClient_FetchDirectory` を実装
    - 既存 `downloader_test.go` のテストパターンを移行

- [ ] **Step 2: github パッケージ実装 (TDD - Green)**
    - `features/devctl/internal/github/github.go` を新規作成
    - `DownloadedFile`, `Client`, `NewClient`, `resolveToken`, `FetchFile`, `FetchDirectory`, `CreatePR` を実装
    - 既存 `downloader.go` のロジックを移動

- [ ] **Step 3: scaffold パッケージの書き換え**
    - `scaffold/downloader.go` を削除し、`Downloader` インターフェイス + 型エイリアスのみのファイルに置き換え
    - `scaffold/downloader_test.go` を削除
    - `scaffold.go` の3箇所で `NewGitHubDownloader` → `github.NewClient` に変更

- [ ] **Step 4: action パッケージの書き換え**
    - `action/pr.go` で `github.Client.CreatePR` を利用するように変更

- [ ] **Step 5: ビルド・テスト**
    - `scripts/process/build.sh` を実行し、全体ビルドと単体テストが通ることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `github` パッケージの全テストが PASS すること
    *   `scaffold` パッケージの既存テスト（`applier_test.go`, `locale_test.go` 等）が引き続き PASS すること
    *   `action` パッケージのテストが PASS すること

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "devctl" --specify "scaffold"
    ```
    *   既存の統合テストが引き続き PASS すること

## Documentation

#### [MODIFY] [000-Scaffold-Command.md](file://prompts/phases/000-foundation/ideas/feat-devctl-scafford/000-Scaffold-Command.md)

*   **更新内容**: GitHub API 認証セクションにて、`github` パッケージの存在とトークン取得の優先順位を追記
