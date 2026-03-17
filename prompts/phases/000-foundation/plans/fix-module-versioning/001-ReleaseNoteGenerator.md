# 001-ReleaseNoteGenerator

> **Source Specification**: [001-ReleaseNoteGenerator.md](file://prompts/phases/000-foundation/ideas/fix-module-versioning/001-ReleaseNoteGenerator.md)

## Goal Description

リリースノート自動生成プログラムを `features/release-note/` にGoで実装する。Git履歴からブランチ名を抽出し、`prompts/phases/` 以下の仕様書フォルダを探索してLLMで要約、最終統合要約を経てリリースノートを生成し `releases/notes/latest.md` に出力する。`github-upload.sh` にリリースノート生成ステップを統合する。

## User Review Required

> [!IMPORTANT]
> - `build.sh` に `build_release_note()` 関数を追加し、`features/release-note/` のビルドと単体テストを実行するようにする。既存の `build_tt()` と同等のパターン。
> - LLMプロバイダのOpenAI実装ではChat Completions API (`/v1/chat/completions`) を使用する。外部SDKは使わず `net/http` で直接呼び出す方針とする（依存を最小化するため）。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: リリースノート生成CLI | Proposed Changes > main.go |
| R2: 設定管理 | Proposed Changes > internal/config/config.go |
| R3: Git履歴からブランチ名の特定 | Proposed Changes > internal/git/history.go |
| R4: 仕様書フォルダの探索 | Proposed Changes > internal/scanner/scanner.go |
| R5: LLM個別ブランチ要約 | Proposed Changes > internal/summarizer/summarizer.go |
| R6: LLM最終統合要約 | Proposed Changes > internal/summarizer/summarizer.go |
| R7: リリースノートの保存とGitHub連携 | Proposed Changes > internal/writer/writer.go |
| R8: github-upload.sh への統合 | Proposed Changes > scripts/dist/github-upload.sh |
| R9: LLMプロバイダのファクトリーパターン | Proposed Changes > internal/llm/ |
| R10: バージョン履歴の保存 | Proposed Changes > internal/writer/writer.go |

## Proposed Changes

### テスト: internal/config

#### [NEW] [config_test.go](file://features/release-note/internal/config/config_test.go)
*   **Description**: 設定読み込みの単体テスト
*   **Technical Design**:
    ```go
    package config

    func TestLoadConfig(t *testing.T)
    // Table-driven test cases:
    // - valid config with all fields
    // - missing credentials_path → error
    // - missing llm section → error
    // - invalid YAML syntax → error

    func TestLoadCredentials(t *testing.T)
    // Table-driven test cases:
    // - valid credentials with openai key
    // - specified provider not in credentials → error
    // - empty api_key → error
    // - file not found → error
    ```
*   **Logic**:
    *   テスト用の一時 YAML ファイルを `t.TempDir()` に作成してパースを検証
    *   `Config` 構造体の各フィールドが正しくマッピングされることを確認

---

### テスト: internal/git

#### [NEW] [history_test.go](file://features/release-note/internal/git/history_test.go)
*   **Description**: Git履歴からのブランチ名抽出の単体テスト
*   **Technical Design**:
    ```go
    package git

    func TestExtractBranchNames(t *testing.T)
    // Table-driven test cases:
    // - merge commit message "Merge branch 'feat-xxx' into main" → "feat-xxx"
    // - merge commit message "Merge pull request #123 from user/feat-yyy" → "feat-yyy"
    // - duplicate branches → deduplicated unique list
    // - no merge commits → empty list
    // - mixed formats → all branches extracted

    func TestParseTagVersion(t *testing.T)
    // tool-id "tt", tag "tt-v1.0.0" → "tt-v1.0.0"
    // tag not found → empty string (initial release)
    ```
*   **Logic**:
    *   テスト用のgit logメッセージ文字列を直接渡してパースを検証
    *   実際のgitコマンドは統合テストで検証

---

### テスト: internal/scanner

#### [NEW] [scanner_test.go](file://features/release-note/internal/scanner/scanner_test.go)
*   **Description**: 仕様書フォルダ探索の単体テスト
*   **Technical Design**:
    ```go
    package scanner

    func TestFindSpecFolders(t *testing.T)
    // Table-driven test cases:
    // - branches = ["feat-xxx"], phase "001" has "feat-xxx" folder → found
    // - branches = ["feat-xxx"], phase "001" has no match, phase "000" has match → found in fallback
    // - branches = ["nonexistent"], no match in any phase → empty result (give up)
    // - phase number reaches lower bound → stop searching

    func TestParsePhases(t *testing.T)
    // "000-foundation" → PhaseInfo{Number: 0, Name: "foundation", DirName: "000-foundation"}
    // "001-webservices" → PhaseInfo{Number: 1, Name: "webservices", DirName: "001-webservices"}
    // sorted descending by number
    ```
*   **Logic**:
    *   `t.TempDir()` にダミーのフェーズフォルダ構造を作成してテスト
    *   フォールバック探索のロジック: 最大フェーズから開始し、`max(0, maxPhase - 5)` まで下降

---

### テスト: internal/llm

#### [NEW] [factory_test.go](file://features/release-note/internal/llm/factory_test.go)
*   **Description**: LLMプロバイダファクトリーの単体テスト
*   **Technical Design**:
    ```go
    package llm

    func TestNewProvider(t *testing.T)
    // Table-driven test cases:
    // - provider = "openai" → returns OpenAI client (no error)
    // - provider = "google" → returns ErrNotImplemented
    // - provider = "anthropic" → returns ErrNotImplemented
    // - provider = "unknown" → returns ErrUnknownProvider
    ```

---

### テスト: internal/summarizer

#### [NEW] [summarizer_test.go](file://features/release-note/internal/summarizer/summarizer_test.go)
*   **Description**: 要約エンジンの単体テスト（LLMをモックして検証）
*   **Technical Design**:
    ```go
    package summarizer

    func TestSummarizeBranch(t *testing.T)
    // Mock LLM provider を使用し、ファイル内容を渡して要約が呼ばれることを確認
    // System prompt に 【新規】【変更】【削除】のカテゴリが含まれることを確認

    func TestConsolidateSummaries(t *testing.T)
    // Mock LLM provider を使用し、複数ブランチの要約を統合する呼び出しを確認
    // System prompt に統合ルール（中間状態除去等）が含まれることを確認
    ```

---

### テスト: internal/writer

#### [NEW] [writer_test.go](file://features/release-note/internal/writer/writer_test.go)
*   **Description**: リリースノートファイル出力の単体テスト
*   **Technical Design**:
    ```go
    package writer

    func TestWriteReleaseNote(t *testing.T)
    // Table-driven test cases:
    // - valid content → latest.md written with correct format
    // - version specified → {version}.md archive also created (R10)
    // - output directory does not exist → created automatically
    ```
*   **Logic**:
    *   `t.TempDir()` に出力して内容を検証
    *   出力ファイルが `# Release Notes` ヘッダと `## What's New` セクションを含むことを確認

---

### 実装: internal/llm (Interface → Struct → Logic)

#### [NEW] [provider.go](file://features/release-note/internal/llm/provider.go)
*   **Description**: LLMプロバイダの共通インターフェース定義
*   **Technical Design**:
    ```go
    package llm

    import "context"

    // Provider is the common interface for LLM access.
    // Each provider (OpenAI, Google, Anthropic) implements this interface.
    type Provider interface {
        // Summarize sends systemPrompt and userContent to the LLM
        // and returns the generated summary text.
        Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error)
    }

    // ErrNotImplemented is returned when a provider is not yet implemented.
    var ErrNotImplemented = errors.New("provider not implemented")

    // ErrUnknownProvider is returned when an unknown provider name is specified.
    var ErrUnknownProvider = errors.New("unknown provider")
    ```

#### [NEW] [factory.go](file://features/release-note/internal/llm/factory.go)
*   **Description**: プロバイダファクトリー
*   **Technical Design**:
    ```go
    package llm

    // NewProvider creates a Provider instance for the given provider name.
    // Supported: "openai". TODO: "google", "anthropic".
    func NewProvider(providerName string, apiKey string, model string) (Provider, error)
    // - "openai" → return openai.New(apiKey, model), nil
    // - "google" → return nil, fmt.Errorf("google (Gemini): %w", ErrNotImplemented)
    // - "anthropic" → return nil, fmt.Errorf("anthropic: %w", ErrNotImplemented)
    // - other → return nil, fmt.Errorf("%s: %w", providerName, ErrUnknownProvider)
    ```

#### [NEW] [openai/client.go](file://features/release-note/internal/llm/openai/client.go)
*   **Description**: OpenAI Chat Completions API の実装
*   **Technical Design**:
    ```go
    package openai

    // Client implements llm.Provider for OpenAI.
    type Client struct {
        apiKey     string
        model      string
        httpClient *http.Client
        baseURL    string  // default: "https://api.openai.com"
    }

    func New(apiKey string, model string) *Client

    // Summarize sends a chat completion request.
    // Request body:
    //   {"model": model, "messages": [
    //     {"role": "system", "content": systemPrompt},
    //     {"role": "user", "content": userContent}
    //   ]}
    // Endpoint: POST /v1/chat/completions
    // Parses response: choices[0].message.content
    func (c *Client) Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error)
    ```
*   **Logic**:
    *   `net/http` で直接 HTTP リクエストを構築（外部SDKは使用しない）
    *   レスポンスをJSONデコードし、`choices[0].message.content` を返す
    *   HTTPステータスコードがエラーの場合は適切なエラーメッセージを返す

#### [NEW] [google/client.go](file://features/release-note/internal/llm/google/client.go)
*   **Description**: Google Gemini の TODO スタブ
*   **Technical Design**:
    ```go
    package google

    // Client is a placeholder for Google Gemini provider.
    // TODO: Implement Google Gemini API integration.
    type Client struct{}

    func New(apiKey string, model string) *Client
    func (c *Client) Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error)
    // → returns llm.ErrNotImplemented
    ```

#### [NEW] [anthropic/client.go](file://features/release-note/internal/llm/anthropic/client.go)
*   **Description**: Anthropic の TODO スタブ
*   **Technical Design**:
    ```go
    package anthropic

    // Client is a placeholder for Anthropic provider.
    // TODO: Implement Anthropic API integration.
    type Client struct{}

    func New(apiKey string, model string) *Client
    func (c *Client) Summarize(ctx context.Context, systemPrompt string, userContent string) (string, error)
    // → returns llm.ErrNotImplemented
    ```

---

### 実装: internal/config

#### [NEW] [config.go](file://features/release-note/internal/config/config.go)
*   **Description**: 設定ファイルの読み込みと解析
*   **Technical Design**:
    ```go
    package config

    // Config represents the main configuration from config.yaml.
    type Config struct {
        CredentialsPath string    `yaml:"credentials_path"`
        LLM             LLMConfig `yaml:"llm"`
    }

    type LLMConfig struct {
        Provider string `yaml:"provider"`
        Model    string `yaml:"model"`
    }

    // Credentials represents the secrets from credential.yaml.
    type Credentials struct {
        LLM LLMCredentials `yaml:"llm"`
    }

    type LLMCredentials struct {
        Providers map[string]ProviderCredential `yaml:"providers"`
    }

    type ProviderCredential struct {
        APIKey string `yaml:"api_key"`
    }

    // Load reads config.yaml from the given path and also loads
    // the referenced credential.yaml.
    // Returns Config and the resolved API key for the configured provider.
    func Load(configPath string) (*Config, string, error)
    ```
*   **Logic**:
    1. `configPath` を読み込み `Config` にアンマーシャル
    2. `Config.CredentialsPath` を `configPath` の親ディレクトリからの相対パスとして解決
    3. `credential.yaml` を `Credentials` にアンマーシャル
    4. `Config.LLM.Provider` に対応する `api_key` を取得して返す
    5. プロバイダが見つからない場合はエラー

---

### 実装: internal/git

#### [NEW] [history.go](file://features/release-note/internal/git/history.go)
*   **Description**: Git履歴からのブランチ名抽出
*   **Technical Design**:
    ```go
    package git

    // Collector gathers Git history information.
    type Collector struct {
        repoRoot string
    }

    func NewCollector(repoRoot string) *Collector

    // GetLatestReleaseTag returns the latest release tag for a tool-id,
    // or empty string if no release exists.
    // Executes: gh release list --limit 100 --json tagName ...
    func (c *Collector) GetLatestReleaseTag(toolID string) (string, error)

    // GetCommitSHA returns the commit SHA for a given tag.
    // Executes: git rev-list -n 1 <tag>
    func (c *Collector) GetCommitSHA(tag string) (string, error)

    // GetBranchNames extracts branch names from merge commits since a given commit.
    // If sinceCommit is empty, gets all merge commits.
    // Executes: git log --merges --oneline [sinceCommit..HEAD]
    // Parses branch names from merge commit messages.
    func (c *Collector) GetBranchNames(sinceCommit string) ([]string, error)

    // ExtractBranchFromMessage parses a merge commit message and returns
    // the branch name, or empty string if not a merge commit.
    // Patterns:
    //   "Merge branch 'xxx' into yyy" → "xxx"
    //   "Merge pull request #N from user/xxx" → "xxx"
    func ExtractBranchFromMessage(message string) string
    ```
*   **Logic**:
    *   `os/exec` で `git` / `gh` コマンドを実行
    *   正規表現でブランチ名を抽出:
        *   `Merge branch '([^']+)'` → group 1
        *   `Merge pull request #\d+ from [^/]+/(.+)` → group 1
    *   結果を `map[string]struct{}` で重複排除し、ソートされたスライスとして返す

---

### 実装: internal/scanner

#### [NEW] [scanner.go](file://features/release-note/internal/scanner/scanner.go)
*   **Description**: 仕様書フォルダの探索
*   **Technical Design**:
    ```go
    package scanner

    // PhaseInfo represents a phase directory.
    type PhaseInfo struct {
        Number  int    // e.g. 0, 1, 2
        Name    string // e.g. "foundation"
        DirName string // e.g. "000-foundation"
    }

    // BranchSpec represents a found specification folder for a branch.
    type BranchSpec struct {
        BranchName string   // e.g. "feat-xxx"
        PhaseName  string   // e.g. "000-foundation"
        FolderPath string   // absolute path to the ideas/{branch} folder
        Files      []string // list of .md files in the folder
    }

    // Scanner searches for specification folders.
    type Scanner struct {
        phasesRoot string // path to prompts/phases/
    }

    func NewScanner(phasesRoot string) *Scanner

    // ListPhases reads prompts/phases/ and returns phases sorted descending.
    func (s *Scanner) ListPhases() ([]PhaseInfo, error)

    // FindSpecFolders finds spec folders for the given branch names.
    // Algorithm:
    //   1. Start from the highest phase number
    //   2. Check ideas/{branch} folder existence for each branch
    //   3. If not found, decrement phase number and retry
    //   4. Stop at phase 000 or (maxPhase - 5), whichever is larger
    //   5. Branches not found are silently skipped
    func (s *Scanner) FindSpecFolders(branches []string) ([]BranchSpec, error)
    ```
*   **Logic**:
    *   `os.ReadDir()` でフェーズディレクトリ一覧を取得
    *   `{nnn}-{name}` 形式の正規表現でパース: `^(\d{3})-(.+)$`
    *   各ブランチについて:
        *   フェーズ番号を最大値から下降しながら `ideas/{branch}` の存在を `os.Stat()` で確認
        *   見つかったら `os.ReadDir()` で `.md` ファイル一覧を取得
        *   下限 = `max(0, maxPhaseNumber - 5)` に到達したら探索終了

---

### 実装: internal/summarizer

#### [NEW] [summarizer.go](file://features/release-note/internal/summarizer/summarizer.go)
*   **Description**: LLMを用いた要約エンジン
*   **Technical Design**:
    ```go
    package summarizer

    // Summarizer orchestrates LLM-based summarization.
    type Summarizer struct {
        provider llm.Provider
    }

    func New(provider llm.Provider) *Summarizer

    // SummarizeBranch reads all files in a BranchSpec and asks the LLM
    // to produce a categorized summary (新規/変更/削除).
    // Returns the summary text or error.
    func (s *Summarizer) SummarizeBranch(ctx context.Context, branch scanner.BranchSpec) (string, error)

    // Consolidate takes all per-branch summaries and produces a final
    // integrated summary following the consolidation rules:
    //   - Remove intermediate states
    //   - Merge duplicates
    //   - Group related items
    //   - Focus on "what is the final state"
    func (s *Summarizer) Consolidate(ctx context.Context, branchSummaries []string) (string, error)
    ```
*   **Logic**:
    *   `SummarizeBranch` の system prompt:
        ```
        あなたはリリースノートの作成者です。以下の仕様書ファイルの内容を読み、
        ユーザー（プログラムの利用者）が受ける影響に着目して、変更を以下の3カテゴリに分類してください:
        (1)【新規】: 新しい機能、新しい設定などの登場
        (2)【変更】: 既存の機能・設定がどう変わるのか（Before → After）
        (3)【削除】: 廃止される機能、設定など
        ```
    *   `Consolidate` の system prompt:
        ```
        以下は複数の変更の要約です。これらを統合し、最終的なリリースノートを作成してください。
        ルール:
        - 中間状態を除去し、最終状態のみ記述する
        - 同じ項目への重複した変更は1つにまとめる
        - 削除と追加が同名の場合は「新しい挙動になった」と統合する
        - 関連項目はグルーピングする
        - 「結局最終的にどうなったのか」に着目すること
        ```

---

### 実装: internal/writer

#### [NEW] [writer.go](file://features/release-note/internal/writer/writer.go)
*   **Description**: リリースノートファイルの出力
*   **Technical Design**:
    ```go
    package writer

    // Writer handles release note file output.
    type Writer struct {
        notesDir string // path to releases/notes/
    }

    func New(notesDir string) *Writer

    // Write saves the release note content.
    // 1. Writes to {notesDir}/latest.md
    // 2. Writes to {notesDir}/{version}.md as archive (R10)
    func (w *Writer) Write(content string, version string) error
    ```
*   **Logic**:
    *   出力フォーマット:
        ```markdown
        # Release Notes

        ## What's New

        {consolidated summary content}
        ```
    *   `latest.md` を上書き
    *   `{version}.md` を新規作成（既存の場合は上書き）
    *   必要に応じてディレクトリを `os.MkdirAll()` で作成

---

### 実装: エントリポイント

#### [MODIFY] [main.go](file://features/release-note/main.go)
*   **Description**: CLI エントリポイントの実装
*   **Technical Design**:
    ```go
    package main

    // CLI flags:
    //   --tool-id    string   Target tool ID (e.g. "tt")
    //   --version    string   Release version (e.g. "v1.0.0")
    //   --repo-root  string   Repository root path
    //   --config     string   Config file path (default: "./settings/config.yaml")
    func main()
    ```
*   **Logic**:
    1. `flag` パッケージでコマンドライン引数をパース
    2. `config.Load()` で設定とAPIキーを読み込み
    3. `llm.NewProvider()` でLLMプロバイダを生成
    4. `git.NewCollector()` でGit履歴を収集、ブランチ名を抽出
    5. `scanner.NewScanner()` で仕様書フォルダを探索
    6. `summarizer.New()` で各ブランチの要約を生成
    7. `summarizer.Consolidate()` で統合要約を生成
    8. `writer.New()` でリリースノートファイルを出力
    9. 各ステップの進捗をログ出力

#### [MODIFY] [go.mod](file://features/release-note/go.mod)
*   **Description**: 依存パッケージの追加
*   **Technical Design**:
    ```
    module github.com/axsh/tokotachi/features/release-note

    go 1.24.0

    require gopkg.in/yaml.v3 v3.0.1
    ```
*   **Logic**: YAML パース用に `gopkg.in/yaml.v3` を追加。OpenAI API アクセスは `net/http` 標準ライブラリを使用するため追加依存は不要。

---

### 実装: 設定ファイル

#### [MODIFY] [config.yaml](file://features/release-note/settings/config.yaml)
*   **Description**: LLMプロバイダ/モデル設定の追加
*   **Technical Design**:
    ```yaml
    # Global Application Configuration
    credentials_path: "./secrets/credential.yaml"

    llm:
      provider: "openai"
      model: "gpt-4.1"
    ```

---

### 実装: スクリプト統合

#### [MODIFY] [github-upload.sh](file://scripts/dist/github-upload.sh)
*   **Description**: リリースノート生成ステップの追加
*   **Technical Design**:
    *   Step 1/3 → Step 1/4: Build (変更なし)
    *   **Step 2/4: Generate Release Notes (新規追加)**
    *   Step 2/3 → Step 3/4: Release
    *   Step 3/3 → Step 4/4: Publish
*   **Logic**:
    *   Build完了後に以下を挿入:
    ```bash
    # ─── Step 2: Generate Release Notes ─────────────────────────────
    info "=== Step 2/4: Generate Release Notes ==="
    RELEASE_NOTE_DIR="${REPO_ROOT}/features/release-note"
    if (cd "$RELEASE_NOTE_DIR" && go run . \
          --tool-id "$TOOL_ID" \
          --version "$NEW_VERSION" \
          --repo-root "$REPO_ROOT"); then
      pass "Release notes generated."
    else
      warn "Release note generation failed. Continuing with auto-generated notes."
    fi
    ```
    *   リリースノート生成失敗時はパイプライン全体を止めず、`publish.sh` 側で `--generate-notes` にフォールバックする既存の仕組みに委ねる

#### [MODIFY] [build.sh](file://scripts/process/build.sh)
*   **Description**: `features/release-note/` のビルドと単体テストを追加
*   **Technical Design**:
    *   `build_tt()` の直後に `build_release_note()` 関数を追加
    *   `build_tt()` と同等のパターン:
    ```bash
    build_release_note() {
        step "release-note (Go): Build & Unit Test"
        local rn_dir="$PROJECT_ROOT/features/release-note"
        if [[ ! -f "$rn_dir/go.mod" ]]; then
            warn "features/release-note/go.mod not found — skipping."
            return 0
        fi
        cd "$rn_dir"
        info "Building release-note..."
        if go build .; then
            success "release-note build succeeded."
        else
            fail "release-note build failed."
            FAILED=true
            cd "$PROJECT_ROOT"
            return 1
        fi
        info "Running release-note unit tests..."
        if go test -v -count=1 ./...; then
            success "All release-note unit tests passed."
        else
            fail "release-note unit tests failed."
            FAILED=true
            cd "$PROJECT_ROOT"
            return 1
        fi
        cd "$PROJECT_ROOT"
    }
    ```
    *   `main()` 内の `build_tt` の後に `build_release_note` を呼び出す

## Step-by-Step Implementation Guide

### Phase 1: 基盤パッケージ (Interface + Config)

1.  **LLMインターフェース定義**:
    *   `internal/llm/provider.go` を作成し、`Provider` インターフェース、`ErrNotImplemented`、`ErrUnknownProvider` を定義
2.  **LLMファクトリーテスト作成**:
    *   `internal/llm/factory_test.go` を作成
3.  **LLMファクトリー実装**:
    *   `internal/llm/factory.go` を作成
4.  **OpenAIクライアントテスト準備**:
    *   OpenAI は HTTP 通信が必要のため単体テストでは `httptest.NewServer()` でモックサーバーを用意
5.  **OpenAIクライアント実装**:
    *   `internal/llm/openai/client.go` を作成
6.  **Google/Anthropicスタブ作成**:
    *   `internal/llm/google/client.go` と `internal/llm/anthropic/client.go` をTODOスタブとして作成
7.  **Config テスト作成**:
    *   `internal/config/config_test.go` を作成
8.  **Config 実装**:
    *   `internal/config/config.go` を作成
9.  **go.mod 更新**:
    *   `go mod tidy` を実行して `gopkg.in/yaml.v3` を追加

### Phase 2: 情報収集 (Git + Scanner)

10. **Git履歴テスト作成**:
    *   `internal/git/history_test.go` を作成 (`ExtractBranchFromMessage` のテスト)
11. **Git履歴実装**:
    *   `internal/git/history.go` を作成
12. **Scannerテスト作成**:
    *   `internal/scanner/scanner_test.go` を作成
13. **Scanner実装**:
    *   `internal/scanner/scanner.go` を作成

### Phase 3: 要約と出力

14. **Summarizerテスト作成**:
    *   `internal/summarizer/summarizer_test.go` を作成 (Mock Provider を使用)
15. **Summarizer実装**:
    *   `internal/summarizer/summarizer.go` を作成
16. **Writerテスト作成**:
    *   `internal/writer/writer_test.go` を作成
17. **Writer実装**:
    *   `internal/writer/writer.go` を作成

### Phase 4: CLI + スクリプト統合

18. **設定ファイル更新**:
    *   `features/release-note/settings/config.yaml` を更新
19. **main.go 実装**:
    *   CLI引数パース、全パッケージの統合を実装
20. **github-upload.sh 修正**:
    *   リリースノート生成ステップを Step 2/4 として追加
21. **build.sh 修正**:
    *   `build_release_note()` 関数を追加

### Phase 5: ビルド検証

22. **ビルドと単体テスト実行**:
    *   `./scripts/process/build.sh` を実行し、全テストがパスすることを確認
23. **統合テスト実行**:
    *   `./scripts/process/integration_test.sh` を実行し、既存テストに影響がないことを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `features/release-note/` 以下の全パッケージがビルド成功すること
    *   全単体テストがパスすること
    *   既存の `features/tt/` のビルドとテストに影響がないこと

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   既存の統合テストに影響がないことを確認

### Manual Verification

*   リリースノート生成プログラムの手動エンドツーエンドテストは、LLM APIキーが必要なため手動で実行する:
    1. `cd features/release-note && go run . --tool-id tt --version v0.1.0 --repo-root <project_root>`
    2. `releases/notes/latest.md` が正しいマークダウン形式で出力されることを確認
    3. `releases/notes/v0.1.0.md` がアーカイブとして作成されることを確認

## Documentation

#### [MODIFY] [config.yaml](file://features/release-note/settings/config.yaml)
*   **更新内容**: LLMプロバイダとモデル設定の追加。`credentials_path` のタイポ修正 (`credentials.yaml` → `credential.yaml`)
