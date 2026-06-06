# 003-GitHub-Token-AutoResolve

> **Source Specification**: [003-GitHub-Token-AutoResolve.md](file://prompts/phases/000-foundation/ideas/feat-devctl-scafford/003-GitHub-Token-AutoResolve.md)

## Goal Description

GitHub Token の自動取得と、GitHub 操作の共通パッケージ化を行う。`internal/github` パッケージを新設し、現在 `scaffold/downloader.go` に実装されている HTTP API アクセスと `action/pr.go` の `gh` CLI 操作を集約する。呼び出し元はインターフェイスを通じて操作し、内部実装を意識しない設計とする。

## User Review Required

> [!IMPORTANT]
> **`DownloadedFile` の移動方針**: `scaffold.DownloadedFile` は `scaffold` パッケージ内で広く使用されている（`applier.go`, `locale.go`, テスト約20箇所）。循環依存を避けるため `github` パッケージに移動し、`scaffold` パッケージでは型エイリアス (`type DownloadedFile = github.DownloadedFile`) で完全互換を維持します。既存コードの変更は不要です。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. `gh auth token` によるフォールバック取得 | Proposed Changes > `github.go` - `resolveToken()` |
| 2. `GITHUB_TOKEN` 環境変数の優先 | Proposed Changes > `github.go` - `resolveToken()` |
| 3. エラーハンドリング (`gh` なし/未認証時) | Proposed Changes > `github.go` - `resolveToken()` |
| 4. 既存テストへの影響なし | Verification Plan |
| 5-a. HTTP アクセスの集約 | Proposed Changes > `github.go` - `Client.FetchFile`, `FetchDirectory` |
| 5-b. `gh` CLI 操作の集約 | Proposed Changes > `github.go` - `Client.CreatePR` |
| 5-c. インターフェイスを変えずに内部差し替え可能 | `scaffold.Downloader` インターフェイスは維持、`github.Client` が暗黙実装 |
| 6. verbose ログ (任意) | 対象外: スコープ外 |

## Proposed Changes

### github パッケージ (新規)

#### [NEW] [github_test.go](file://features/devctl/internal/github/github_test.go)

*   **Description**: `github.Client` の単体テスト。既存 `scaffold/downloader_test.go` のテストを移行し、トークン解決テストを追加
*   **Technical Design**:

    ```go
    package github

    import (
        "encoding/base64"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "os"
        "testing"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
    )
    ```

*   **テストケース**:

    **`TestResolveToken`** (テーブル駆動):

    | ケース | `GITHUB_TOKEN` 設定値 | 期待動作 |
    |---|---|---|
    | `"env var set"` | `"ghp_test123"` | `"ghp_test123"` を返す |
    | `"env var empty"` | `""` | パニックしないことを確認 (`assert.NotPanics`) |

    *   Logic:
        *   `wantEnv: true` の場合: `t.Setenv("GITHUB_TOKEN", envToken)` → `assert.Equal(t, envToken, resolveToken())`
        *   `wantEnv: false` の場合: `t.Setenv("GITHUB_TOKEN", "")` → `assert.NotPanics(t, func() { resolveToken() })`

    **`TestNewClient`** (テーブル駆動、既存 `TestNewGitHubDownloader` を移行):

    | ケース | URL | owner | repo | wantErr |
    |---|---|---|---|---|
    | `"valid HTTPS URL"` | `"https://github.com/axsh/tokotachi-scaffolds"` | `"axsh"` | `"tokotachi-scaffolds"` | `false` |
    | `"URL with trailing slash"` | `"https://github.com/axsh/tokotachi-scaffolds/"` | `"axsh"` | `"tokotachi-scaffolds"` | `false` |
    | `"invalid URL"` | `"https://github.com/axsh"` | | | `true` |
    | `"empty URL for PR-only"` | `""` | `""` | `""` | `false` |

    **`TestClient_FetchFile`** (既存 `TestGitHubDownloader_FetchFile` を移行):
    *   `httptest.NewServer` でモックサーバーを起動
    *   `NewClient` + `WithBaseURL` + `WithHTTPClient` でテスト用 Client を作成
    *   `client.FetchFile("catalog.yaml")` の結果を検証

    **`TestClient_FetchFile_NotFound`** (既存テスト移行):
    *   モックサーバーが 404 を返す場合のエラー検証

    **`TestClient_FetchDirectory`** (既存テスト移行):
    *   モックサーバーがディレクトリリスティングとファイルコンテンツを返す場合の検証

---

#### [NEW] [github.go](file://features/devctl/internal/github/github.go)

*   **Description**: GitHub 操作を集約するパッケージ。HTTP API アクセスと `gh` CLI 操作の両方を内包する
*   **Technical Design**:

    ```go
    package github

    import (
        "encoding/base64"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "net/url"
        "os"
        "os/exec"
        "path"
        "strings"
        "time"

        "github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
    )

    // DownloadedFile represents a file downloaded from a GitHub repository.
    type DownloadedFile struct {
        RelativePath string
        Content      []byte
    }

    // Client は GitHub 操作を集約する。
    // HTTP API によるリポジトリコンテンツ取得と、gh CLI による PR 操作を提供する。
    type Client struct {
        Owner     string
        Repo      string
        Branch    string
        Client    *http.Client
        BaseURL   string         // テスト用 Override (デフォルト: https://api.github.com)
        token     string
        cmdRunner *cmdexec.Runner // PR 操作用 (optional)
    }

    // githubContentResponse represents the GitHub API response for a single file.
    type githubContentResponse struct {
        Type     string `json:"type"`
        Encoding string `json:"encoding"`
        Content  string `json:"content"`
        Name     string `json:"name"`
        Path     string `json:"path"`
    }

    const defaultGitHubAPI = "https://api.github.com"

    // ClientOption は Client のオプション設定関数。
    type ClientOption func(*Client)

    // WithCmdRunner は gh CLI 操作用の cmdexec.Runner を設定する。
    func WithCmdRunner(r *cmdexec.Runner) ClientOption

    // WithBaseURL はテスト用に API Base URL を上書きする。
    func WithBaseURL(url string) ClientOption

    // WithHTTPClient はテスト用に HTTP Client を上書きする。
    func WithHTTPClient(c *http.Client) ClientOption

    // NewClient はリポジトリ URL から Client を作成する。
    // repoURL が空の場合は PR 操作専用の Client を返す（owner/repo は未設定）。
    // トークンは自動解決される（GITHUB_TOKEN → gh auth token → 空）。
    func NewClient(repoURL string, opts ...ClientOption) (*Client, error)

    // FetchFile はリポジトリから単一ファイルを取得する。
    func (c *Client) FetchFile(filePath string) ([]byte, error)

    // FetchDirectory はディレクトリ配下の全ファイルを再帰取得する。
    func (c *Client) FetchDirectory(dirPath string) ([]DownloadedFile, error)

    // CreatePR は Pull Request をインタラクティブに作成する。
    // cmdRunner が未設定の場合はエラーを返す。
    func (c *Client) CreatePR(workDir string) error

    // resolveToken は GitHub API トークンを解決する（非公開）。
    func resolveToken() string
    ```

*   **Logic (`resolveToken`)**:
    1. `token := os.Getenv("GITHUB_TOKEN")` を呼び出す
    2. `token != ""` であればその値を返す
    3. `cmd := exec.Command("gh", "auth", "token")` を実行
    4. `output, err := cmd.Output()` で stdout を取得
    5. `err == nil` の場合、`strings.TrimSpace(string(output))` を返す
    6. `err != nil` の場合（`gh` が PATH にない、未認証等）、空文字列 `""` を返す

*   **Logic (`NewClient`)**:
    1. `repoURL` が空の場合: `owner`, `repo` を空のまま、`Branch: "main"` を設定
    2. `repoURL` が非空の場合: `url.Parse` → `strings.Split(parsed.Path, "/")` で owner/repo を抽出（既存 `NewGitHubDownloader` と同一ロジック）
    3. `token: resolveToken()` でトークンを自動解決
    4. `Client: &http.Client{Timeout: 30 * time.Second}` をデフォルト設定
    5. `BaseURL: defaultGitHubAPI` をデフォルト設定
    6. `opts` を順に適用
    7. `*Client` を返す

*   **Logic (`FetchFile`)**: 既存 `GitHubDownloader.FetchFile` から完全移動
    1. `apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.BaseURL, c.Owner, c.Repo, filePath, c.Branch)` で API URL を構築
    2. `c.newRequest(apiURL)` で認証付き HTTP リクエストを作成
    3. `c.Client.Do(req)` で実行
    4. `resp.StatusCode != 200` の場合 `fmt.Errorf("failed to fetch %s: HTTP %d", filePath, resp.StatusCode)` を返す
    5. レスポンス JSON をパース → `base64.StdEncoding.DecodeString(content)` でデコード
    6. デコード済みバイト列を返す

*   **Logic (`FetchDirectory`)**: 既存 `GitHubDownloader.FetchDirectory` + `fetchDirectoryRecursive` から完全移動
    1. `c.fetchDirectoryRecursive(dirPath, "")` を呼び出す
    2. 再帰処理: ディレクトリエントリを取得、type="file" は `FetchFile` で取得、type="dir" は再帰呼び出し
    3. `[]DownloadedFile` を返す

*   **Logic (`CreatePR`)**:
    1. `c.cmdRunner == nil` の場合、`fmt.Errorf("CmdRunner is required for PR operations")` を返す
    2. `ghCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GH", "gh")` でコマンドパスを解決
    3. `opts := cmdexec.RunOption{Dir: workDir}` でオプションを設定
    4. `c.cmdRunner.RunInteractiveWithOpts(opts, ghCmd, "pr", "create")` を実行
    5. エラーがあれば返す

*   **Logic (`newRequest` - 非公開メソッド)**: 既存から移動
    1. `http.NewRequest(http.MethodGet, url, nil)` でリクエスト作成
    2. `c.token != ""` の場合、`req.Header.Set("Authorization", "Bearer "+c.token)` を追加
    3. リクエストを返す

---

### scaffold パッケージ (修正)

#### [MODIFY] [downloader.go](file://features/devctl/internal/scaffold/downloader.go)

*   **Description**: `GitHubDownloader` 実装と `DownloadedFile` 定義を削除し、`Downloader` インターフェイスと `DownloadedFile` 型エイリアスのみを残す
*   **Technical Design**:

    既存ファイルの全内容を以下に**置き換え**:

    ```go
    package scaffold

    import "github.com/axsh/tokotachi/features/devctl/internal/github"

    // DownloadedFile is a type alias for github.DownloadedFile.
    // This maintains backward compatibility with existing scaffold code.
    type DownloadedFile = github.DownloadedFile

    // Downloader is the interface for fetching files from a template repository.
    // github.Client implements this interface implicitly.
    type Downloader interface {
        FetchFile(path string) ([]byte, error)
        FetchDirectory(path string) ([]DownloadedFile, error)
    }
    ```

*   **Logic**:
    *   `DownloadedFile = github.DownloadedFile` は Go の型エイリアス。既存コード（`applier.go`, `locale.go`, テスト等）で `scaffold.DownloadedFile` を使っている箇所は、すべてそのまま動作する
    *   `GitHubDownloader` struct、`NewGitHubDownloader` 関数、`FetchFile`/`FetchDirectory`/`fetchDirectoryRecursive`/`newRequest` メソッド、`githubContentResponse` struct、`defaultGitHubAPI` 定数 — すべて `github.go` に移動済みのため削除

---

#### [DELETE] [downloader_test.go](file://features/devctl/internal/scaffold/downloader_test.go)

*   **Description**: テストは `github/github_test.go` に移行するため削除

---

#### [MODIFY] [scaffold.go](file://features/devctl/internal/scaffold/scaffold.go)

*   **Description**: `NewGitHubDownloader` の呼び出しを `github.NewClient` に変更する
*   **Technical Design**:
    *   import: `"github.com/axsh/tokotachi/features/devctl/internal/github"` を追加
*   **Logic**:
    *   L46 (`Run` 内): `downloader, err := NewGitHubDownloader(opts.RepoURL)` → `downloader, err := github.NewClient(opts.RepoURL)`
    *   L174 (`Apply` 内): `downloader, err := NewGitHubDownloader(opts.RepoURL)` → `downloader, err := github.NewClient(opts.RepoURL)`
    *   L292 (`List` 内): `downloader, err := NewGitHubDownloader(repoURL)` → `downloader, err := github.NewClient(repoURL)`
    *   戻り値 `*github.Client` は `Downloader` インターフェイスの `FetchFile`/`FetchDirectory` を暗黙的に満たすため、`downloader` 変数をそのまま使える
    *   import `"os"` は `os.Stdout`, `os.Stdin`, `os.Stderr` で引き続き使用されるため残す

---

### action パッケージ (修正)

#### [MODIFY] [pr.go](file://features/devctl/internal/action/pr.go)

*   **Description**: PR 作成を `github.Client.CreatePR` に委譲する
*   **Technical Design**:

    全内容を以下に**置き換え**:

    ```go
    package action

    import (
        "fmt"

        "github.com/axsh/tokotachi/features/devctl/internal/github"
    )

    // PR creates a GitHub Pull Request using github.Client.
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

*   **Logic**:
    *   `github.NewClient("")` : repoURL 空 → PR 操作専用 Client（owner/repo 不要）
    *   `github.WithCmdRunner(r.CmdRunner)` : `action.Runner.CmdRunner` (`*cmdexec.Runner`) を渡す
    *   `client.CreatePR(worktreePath)` : 内部で `gh pr create` を実行
    *   `cmdexec` の import は不要になる（`github` パッケージが内部で import）

## Step-by-Step Implementation Guide

- [x] **Step 1: github パッケージのテスト作成 (TDD - Red)**
    - `features/devctl/internal/github/github_test.go` を新規作成
    - `TestResolveToken`, `TestNewClient`, `TestClient_FetchFile`, `TestClient_FetchFile_NotFound`, `TestClient_FetchDirectory` を実装
    - この時点ではコンパイルエラー（`github` パッケージが未定義）

- [x] **Step 2: github パッケージの実装 (TDD - Green)**
    - `features/devctl/internal/github/github.go` を新規作成
    - `DownloadedFile`, `githubContentResponse`, `Client`, `ClientOption`, `NewClient`, `resolveToken`, `newRequest`, `FetchFile`, `FetchDirectory`, `fetchDirectoryRecursive`, `CreatePR`, `WithCmdRunner`, `WithBaseURL`, `WithHTTPClient` を実装
    - 既存 `scaffold/downloader.go` のコードを移動

- [x] **Step 3: scaffold/downloader.go の書き換え**
    - `scaffold/downloader.go` を `Downloader` インターフェイス + `DownloadedFile` 型エイリアスのみに置き換え

- [x] **Step 4: scaffold/downloader_test.go の削除**
    - テストは Step 1 で `github_test.go` に移行済み

- [x] **Step 5: scaffold/scaffold.go の修正**
    - 3箇所の `NewGitHubDownloader` → `github.NewClient` に変更, import 追加

- [x] **Step 6: action/pr.go の修正**
    - `github.Client.CreatePR` に委譲する実装に変更

- [x] **Step 7: ビルド・テスト**
    - `scripts/process/build.sh` を実行し、全体ビルドと単体テストが通ることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**:
        *   `github` パッケージの全テスト (`TestResolveToken`, `TestNewClient`, `TestClient_FetchFile`, `TestClient_FetchFile_NotFound`, `TestClient_FetchDirectory`) が PASS
        *   `scaffold` パッケージの既存テスト (`applier_test.go`, `locale_test.go`, `catalog_test.go`, `plan_test.go` 等) が引き続き PASS
        *   `action` パッケージのテストが PASS（既存テストがある場合）

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "devctl" --specify "scaffold"
    ```
    *   **Log Verification**: `scaffold` 関連の統合テストが引き続き PASS すること。`GITHUB_TOKEN` 環境変数または `gh auth token` でトークンが自動取得され、プライベートリポジトリへのアクセスが成功すること

## Documentation

#### [MODIFY] [000-Scaffold-Command.md](file://prompts/phases/000-foundation/ideas/feat-devctl-scafford/000-Scaffold-Command.md)

*   **更新内容**: GitHub API 認証セクションにて、`github` パッケージの存在とトークン取得の優先順位（`GITHUB_TOKEN` > `gh auth token` > 非認証）を追記
