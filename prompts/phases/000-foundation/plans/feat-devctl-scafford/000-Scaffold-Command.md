# 000-Scaffold-Command

> **Source Specification**: [000-Scaffold-Command.md](file://prompts/phases/000-foundation/ideas/feat-devctl-scafford/000-Scaffold-Command.md)

## Goal Description

`devctl scaffold` サブコマンドを新規実装する。外部リポジトリ (`tokotachi-scaffolds`) からテンプレートをダウンロードし、3セグメント設計（カタログ / テンプレート実体 / 配置定義）に基づいてファイルを展開する。dry-run・ユーザー確認・チェックポイント/ロールバック・多言語対応・スピナー表示を含むフル機能実装。

## User Review Required

> [!IMPORTANT]
> - 外部 HTTP クライアントを使用するため、`go.mod` に `net/http` 標準ライブラリ以外の追加依存は不要だが、スピナー表示用に外部ライブラリ（例: `github.com/briandowns/spinner`）の導入を検討。
> - `tokotachi-scaffolds` リポジトリは本計画のスコープ外（リポジトリ作成は手動 or 別タスク）。テスト時はモック/テストフィクスチャで代替。
> - 統合テスト用の `tests/` ディレクトリは現在存在しないため、単体テストのみを計画。統合テストは `tokotachi-scaffolds` リポジトリ整備後に別計画で対応。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: 基本コマンド | `cmd/scaffold.go` + `internal/scaffold/scaffold.go` |
| R2: デフォルトのプロジェクト構成 | `internal/scaffold/catalog.go` (default_scaffold 解決) |
| R3: メタデータ駆動の後処理 | `internal/scaffold/applier.go` (PostActions) |
| R4: パターン指定によるテンプレート展開 | `internal/scaffold/catalog.go` (ResolvePattern) |
| R5: 外部リポジトリからのダウンロード | `internal/scaffold/downloader.go` |
| R6: 前提条件チェック | `internal/scaffold/catalog.go` (CheckRequirements) |
| R7: コンフリクト解決ポリシー | `internal/scaffold/applier.go` (ConflictPolicy) |
| R8: 実行計画の表示とユーザー確認 | `internal/scaffold/plan.go` + `cmd/scaffold.go` |
| R9: チェックポイントとロールバック | `internal/scaffold/checkpoint.go` |
| R10: テンプレートオプション | `internal/scaffold/catalog.go` (Options) + `cmd/scaffold.go` (インタラクティブ入力) |
| R11: 利用可能テンプレート一覧 | `cmd/scaffold.go` (--list) + `internal/scaffold/catalog.go` (ListScaffolds) |
| R12: リポジトリ変更オプション | `cmd/scaffold.go` (--repo フラグ) |
| R13: Loading アニメーション | `internal/scaffold/spinner.go` |
| R14: テンプレートの多言語対応 | `internal/scaffold/locale.go` + `internal/scaffold/downloader.go` |

## Proposed Changes

### コマンド層 (cmd)

#### [NEW] [scaffold.go](file://features/devctl/cmd/scaffold.go)

*   **Description**: `devctl scaffold` サブコマンドの定義。cobra コマンド + フラグ登録 + `runScaffold` 関数。
*   **Technical Design**:
    ```go
    package cmd

    // scaffoldCmd: cobra.Command
    //   Use:   "scaffold [category] [name]"
    //   Args:  cobra.MaximumNArgs(2)
    //   RunE:  runScaffold

    // フラグ:
    //   --dry-run (rootCmd の PersistentFlags を流用)
    //   --yes      bool
    //   --rollback bool
    //   --list     bool
    //   --repo     string
    //   --lang     string

    // init():
    //   scaffoldCmd に上記フラグを登録
    //   rootCmd.AddCommand(scaffoldCmd) は root.go の init() に追加

    // runScaffold(cmd *cobra.Command, args []string) error:
    //   1. --rollback → scaffold.Rollback(repoRoot) を呼んで終了
    //   2. --list → scaffold.List(repoURL) を呼んで一覧表示して終了
    //   3. スピナー開始
    //   4. scaffold.Run(scaffold.RunOptions{...}) を呼ぶ
    //      RunOptions にはパターン引数・フラグ値・repoRoot を渡す
    //   5. 実行計画を stdout に表示
    //   6. --dry-run なら終了
    //   7. --yes でなければ [y/N] プロンプト表示、N なら終了
    //   8. scaffold.Apply(plan) を呼ぶ
    ```
*   **Logic**:
    - `--dry-run` は rootCmd に既存の `flagDryRun` を利用。ただし scaffold 固有の `--yes` は scaffoldCmd にローカル定義。
    - テンプレートオプションの動的フラグ: Phase 1 では `--Name`, `--GoModule` 等はカタログ取得後にインタラクティブ入力で対応。動的 cobra フラグの追加は Phase 2 以降。
    - インタラクティブプロンプトは `bufio.Scanner` + `os.Stdin` で実装。

#### [MODIFY] [root.go](file://features/devctl/cmd/root.go)

*   **Description**: `scaffoldCmd` を rootCmd に登録。
*   **Technical Design**:
    ```diff
     rootCmd.AddCommand(doctorCmd)
    +rootCmd.AddCommand(scaffoldCmd)
    ```

---

### scaffold パッケージ (internal/scaffold)

#### [NEW] [scaffold_test.go](file://features/devctl/internal/scaffold/scaffold_test.go)

*   **Description**: `scaffold.go` の統合的なテスト。`Run` 関数の全体フローをテスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テーブル駆動テスト:
    // - TestRun_DefaultPattern: パターン未指定 → default_scaffold 解決 → 実行計画生成
    // - TestRun_NamedPattern: 名前指定 → カタログ検索 → 解決
    // - TestRun_CategoryOnly: カテゴリのみ → カテゴリ一覧返却
    // - TestRun_RequirementsFail: 前提条件未達 → エラー
    //
    // モック: Downloader インターフェースをモック化し、
    //         実際の GitHub API を呼ばずにテスト用データを返す
    ```

#### [NEW] [scaffold.go](file://features/devctl/internal/scaffold/scaffold.go)

*   **Description**: scaffold 処理全体のオーケストレーション。
*   **Technical Design**:
    ```go
    package scaffold

    // RunOptions は scaffold 実行の入力パラメータ
    type RunOptions struct {
        Pattern  []string   // コマンド引数 [category, name]
        RepoURL  string     // テンプレートリポジトリ URL
        RepoRoot string     // 適用先リポジトリのルートパス
        DryRun   bool
        Yes      bool
        Lang     string     // 明示的なロケール指定（空なら自動検出）
        Logger   *log.Logger
    }

    // Plan は実行計画を表す
    type Plan struct {
        ScaffoldName string
        FilesToCreate []FileAction
        FilesToModify []FileAction
        PostActions   PostActions
        Warnings      []string       // コンフリクト判定の警告
    }

    // FileAction はファイル操作を表す
    type FileAction struct {
        Path           string
        Action         string  // "create", "skip", "overwrite", "append"
        ConflictPolicy string  // 適用されたポリシー
        Exists         bool    // 既存ファイルの有無
    }

    // Run はカタログ取得 → パターン解決 → 前提条件チェック →
    //       テンプレートダウンロード → 実行計画生成 を行う
    func Run(opts RunOptions) (*Plan, error)
    // 処理フロー:
    //   1. Downloader でカタログ取得
    //   2. カタログからパターン解決 (catalog.ResolvePattern)
    //   3. 前提条件チェック (catalog.CheckRequirements)
    //   4. 配置定義ダウンロード + パース
    //   5. テンプレート実体ダウンロード（ロケール解決込み）
    //   6. 実行計画生成（コンフリクト判定込み）

    // Apply は実行計画に基づいてファイル操作を実行する
    func Apply(plan *Plan, opts RunOptions) error
    // 処理フロー:
    //   1. チェックポイント作成
    //   2. ファイル配置（conflict_policy に従う）
    //   3. 後処理（gitignore_entries 等）
    //   4. チェックポイント情報削除

    // Rollback は直前の scaffold 操作を取り消す
    func Rollback(repoRoot string) error

    // List はカタログから利用可能テンプレート一覧を返す
    func List(repoURL string) ([]ScaffoldEntry, error)

    // PrintPlan は実行計画を人間可読な形式で stdout に出力する
    func PrintPlan(plan *Plan, w io.Writer)
    ```

---

#### [NEW] [catalog_test.go](file://features/devctl/internal/scaffold/catalog_test.go)

*   **Description**: カタログのパース・パターン解決・前提条件チェックのテスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テーブル駆動テスト:
    // - TestParseCatalog: 正常な YAML → Catalog 構造体
    // - TestParseCatalog_InvalidYAML: 不正 YAML → エラー
    // - TestResolvePattern_Default: 引数なし → default_scaffold 解決
    // - TestResolvePattern_ByName: name 指定 → 一致するエントリ
    // - TestResolvePattern_ByCategory: category のみ → 該当エントリ一覧
    // - TestResolvePattern_ByCategoryAndName: category + name → 一致
    // - TestResolvePattern_NotFound: 存在しないパターン → エラー
    // - TestCheckRequirements_Satisfied: 全ディレクトリ存在 → nil
    // - TestCheckRequirements_Missing: ディレクトリ不足 → エラー + ヒント
    ```

#### [NEW] [catalog.go](file://features/devctl/internal/scaffold/catalog.go)

*   **Description**: Segment 1 - カタログの取得・パース・パターン解決・前提条件チェック。
*   **Technical Design**:
    ```go
    package scaffold

    // Catalog はカタログ全体を表す
    type Catalog struct {
        Version         string          `yaml:"version"`
        DefaultScaffold string          `yaml:"default_scaffold"`
        Scaffolds       []ScaffoldEntry `yaml:"scaffolds"`
    }

    // ScaffoldEntry はカタログの各テンプレートエントリ
    type ScaffoldEntry struct {
        Name         string       `yaml:"name"`
        Category     string       `yaml:"category"`
        Description  string       `yaml:"description"`
        TemplateRef  string       `yaml:"template_ref"`
        PlacementRef string       `yaml:"placement_ref"`
        Requirements Requirements `yaml:"requirements"`
        Options      []Option     `yaml:"options"`
    }

    // Requirements は前提条件
    type Requirements struct {
        Directories []string `yaml:"directories"`
        Files       []string `yaml:"files"`
    }

    // Option はテンプレートオプション変数
    type Option struct {
        Name        string `yaml:"name"`
        Description string `yaml:"description"`
        Required    bool   `yaml:"required"`
        Default     string `yaml:"default"`
    }

    // ParseCatalog は YAML バイト列を Catalog にパースする
    func ParseCatalog(data []byte) (*Catalog, error)

    // ResolvePattern はコマンド引数からテンプレートを特定する
    // pattern が空の場合は default_scaffold を使用
    // pattern が 1 要素でカテゴリのみの場合は該当エントリ一覧を返す
    func (c *Catalog) ResolvePattern(pattern []string) ([]ScaffoldEntry, error)

    // CheckRequirements は前提条件をチェックする
    // 不足があればエラーを返す（ヒントメッセージ付き）
    func CheckRequirements(reqs Requirements, repoRoot string) error

    // ListScaffolds はカタログからテンプレート一覧を返す
    func (c *Catalog) ListScaffolds() []ScaffoldEntry
    ```

---

#### [NEW] [downloader_test.go](file://features/devctl/internal/scaffold/downloader_test.go)

*   **Description**: Downloader のテスト（HTTP モック使用）。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestGitHubDownloader_FetchFile: httptest.Server でモック → ファイル取得成功
    // - TestGitHubDownloader_FetchFile_NotFound: 404 → エラー
    // - TestGitHubDownloader_FetchDirectory: ディレクトリ一覧 + 各ファイル取得
    // - TestGitHubDownloader_FetchDirectory_WithLocale: locale.ja 存在 → マージ
    ```

#### [NEW] [downloader.go](file://features/devctl/internal/scaffold/downloader.go)

*   **Description**: 外部リポジトリからのファイルダウンロード。GitHub Contents API を使用。
*   **Technical Design**:
    ```go
    package scaffold

    // Downloader はテンプレートリポジトリからファイルを取得するインターフェース
    type Downloader interface {
        // FetchFile は指定パスのファイル内容を取得する
        FetchFile(path string) ([]byte, error)
        // FetchDirectory は指定パスのディレクトリ内ファイル一覧を再帰取得し、
        // 各ファイルの相対パスと内容を返す
        FetchDirectory(path string) ([]DownloadedFile, error)
    }

    // DownloadedFile はダウンロードしたファイルを表す
    type DownloadedFile struct {
        RelativePath string
        Content      []byte
    }

    // GitHubDownloader は GitHub Contents API を使ったダウンローダー
    type GitHubDownloader struct {
        Owner  string  // e.g. "axsh"
        Repo   string  // e.g. "tokotachi-scaffolds"
        Branch string  // e.g. "main"
        Client *http.Client
    }

    // NewGitHubDownloader はリポジトリ URL をパースして GitHubDownloader を生成する
    func NewGitHubDownloader(repoURL string) (*GitHubDownloader, error)
    // URL パース: "https://github.com/owner/repo" → Owner, Repo
    // Client: http.Client{Timeout: 30 * time.Second}

    func (d *GitHubDownloader) FetchFile(path string) ([]byte, error)
    // GET https://api.github.com/repos/{owner}/{repo}/contents/{path}?ref={branch}
    // レスポンス JSON の "content" フィールドを base64 デコード

    func (d *GitHubDownloader) FetchDirectory(path string) ([]DownloadedFile, error)
    // GET https://api.github.com/repos/{owner}/{repo}/contents/{path}?ref={branch}
    // type=="file" のエントリを再帰取得
    // type=="dir" のエントリは再帰呼び出し
    ```

---

#### [NEW] [placement_test.go](file://features/devctl/internal/scaffold/placement_test.go)

*   **Description**: 配置定義のパース・検証テスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestParsePlacement: 正常 YAML → Placement 構造体
    // - TestParsePlacement_DefaultPolicy: policy 未指定 → "skip" デフォルト
    // - TestParsePlacement_InvalidPolicy: 不正ポリシー → エラー
    // - TestParsePlacement_WithPostActions: gitignore_entries 含む
    ```

#### [NEW] [placement.go](file://features/devctl/internal/scaffold/placement.go)

*   **Description**: Segment 3 - 配置定義のパース・検証。
*   **Technical Design**:
    ```go
    package scaffold

    // Placement は配置定義を表す
    type Placement struct {
        Version        string         `yaml:"version"`
        BaseDir        string         `yaml:"base_dir"`
        ConflictPolicy string         `yaml:"conflict_policy"`
        TemplateConfig TemplateConfig `yaml:"template_config"`
        FileMappings   []FileMapping  `yaml:"file_mappings"`
        PostActions    PostActions    `yaml:"post_actions"`
    }

    // TemplateConfig はテンプレート処理の設定
    type TemplateConfig struct {
        TemplateExtension string `yaml:"template_extension"`
        StripExtension    bool   `yaml:"strip_extension"`
    }

    // FileMapping はファイル名マッピング
    type FileMapping struct {
        Source string `yaml:"source"`
        Target string `yaml:"target"`
    }

    // PostActions は後処理アクションの定義
    type PostActions struct {
        GitignoreEntries []string `yaml:"gitignore_entries"`
    }

    // ValidConflictPolicies は許可されるポリシー値
    var ValidConflictPolicies = []string{"skip", "overwrite", "append", "error"}

    // ParsePlacement は YAML バイト列を Placement にパースし、検証する
    func ParsePlacement(data []byte) (*Placement, error)
    // conflict_policy が空の場合はデフォルト "skip" を設定
    // 不正なポリシー値の場合はエラー
    ```

---

#### [NEW] [applier_test.go](file://features/devctl/internal/scaffold/applier_test.go)

*   **Description**: ファイル配置・コンフリクト解決・後処理のテスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestApplyFiles_CreateNew: 新規ファイル作成
    // - TestApplyFiles_SkipExisting: skip ポリシー → 既存ファイルスキップ
    // - TestApplyFiles_OverwriteExisting: overwrite → 上書き
    // - TestApplyFiles_AppendExisting: append → 追記
    // - TestApplyFiles_ErrorOnExisting: error ポリシー → エラー返却
    // - TestApplyGitignore_AddEntries: .gitignore に新規エントリ追加
    // - TestApplyGitignore_NoDuplicate: 既存エントリは重複追加しない
    // - TestApplyGitignore_CreateFile: .gitignore 未存在 → 新規作成
    // - TestBuildPlan_ConflictDetection: 既存ファイル有 → Plan に skip/overwrite 表示
    //
    // テスト環境: t.TempDir() でテスト用ディレクトリを使用
    ```

#### [NEW] [applier.go](file://features/devctl/internal/scaffold/applier.go)

*   **Description**: テンプレートファイルの配置・コンフリクト解決・後処理の実行。
*   **Technical Design**:
    ```go
    package scaffold

    // ApplyFiles はダウンロードしたテンプレートファイルを配置する
    func ApplyFiles(files []DownloadedFile, placement *Placement,
        repoRoot string, optionValues map[string]string) error
    // 処理:
    //   1. base_dir をオプション値で展開（Go template）
    //   2. 各ファイルについて:
    //      a. file_mappings でファイル名マッピング
    //      b. .tmpl ファイルはテンプレート処理 + 拡張子除去
    //      c. 出力先パス = repoRoot / base_dir / relative_path
    //      d. 既存ファイル有無を確認
    //      e. conflict_policy に従って処理:
    //         - skip: 既存あり → スキップ
    //         - overwrite: 上書き
    //         - append: 末尾追記
    //         - error: エラー返却

    // ApplyPostActions は後処理アクションを実行する
    func ApplyPostActions(actions PostActions, repoRoot string) error
    // gitignore_entries:
    //   1. repoRoot/.gitignore を読み込み（なければ空文字列）
    //   2. 各エントリについて:
    //      a. 既存行に完全一致するものがあればスキップ
    //      b. なければ末尾に追加
    //   3. ファイルを書き戻す

    // BuildPlan は実行計画を生成する（実際のファイル操作はしない）
    func BuildPlan(files []DownloadedFile, placement *Placement,
        repoRoot string, scaffoldName string,
        optionValues map[string]string) (*Plan, error)
    // 各ファイルについてコンフリクト判定を行い、Plan を構築
    ```

---

#### [NEW] [checkpoint_test.go](file://features/devctl/internal/scaffold/checkpoint_test.go)

*   **Description**: チェックポイント作成・ロールバックのテスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestCreateCheckpoint_CleanWorktree: 未コミット変更なし → HEAD 記録
    // - TestCreateCheckpoint_DirtyWorktree: 未コミット変更あり → stash 作成
    // - TestRollback_CleanFiles: 追加ファイル削除
    // - TestRollback_WithStash: stash pop 実行
    // - TestCheckpointFile_SaveLoad: 保存・読み込みの往復テスト
    //
    // モック: git コマンド実行は cmdexec.Runner のモックで代替
    ```

#### [NEW] [checkpoint.go](file://features/devctl/internal/scaffold/checkpoint.go)

*   **Description**: git を利用したチェックポイント作成・ロールバック。
*   **Technical Design**:
    ```go
    package scaffold

    // CheckpointInfo はチェックポイントの情報
    type CheckpointInfo struct {
        CreatedAt          string            `yaml:"created_at"`
        ScaffoldName       string            `yaml:"scaffold_name"`
        HeadCommit         string            `yaml:"head_commit"`
        StashRef           string            `yaml:"stash_ref,omitempty"`
        FilesCreated       []string          `yaml:"files_created"`
        FilesModified      []ModifiedFile    `yaml:"files_modified"`
    }

    // ModifiedFile は変更されたファイルの情報
    type ModifiedFile struct {
        Path                string `yaml:"path"`
        Action              string `yaml:"action"`
        OriginalContentHash string `yaml:"original_content_hash"`
    }

    const CheckpointFileName = ".devctl-scaffold-checkpoint"

    // CreateCheckpoint はチェックポイントを作成する
    func CreateCheckpoint(repoRoot string, plan *Plan) (*CheckpointInfo, error)
    // 処理:
    //   1. `git status --porcelain` で未コミット変更を確認
    //   2. 変更あり: `git stash push -m "devctl-scaffold-checkpoint"`
    //   3. `git rev-parse HEAD` で HEAD コミットハッシュ取得
    //   4. Plan から files_created, files_modified を取得
    //   5. CheckpointInfo を YAML で保存

    // Rollback はチェックポイントから復元する
    func Rollback(repoRoot string) error
    // 処理:
    //   1. .devctl-scaffold-checkpoint を読み込み
    //   2. files_created のファイルを削除
    //   3. files_modified のファイルを stash or git checkout で復元
    //   4. stash_ref があれば `git stash pop`
    //   5. チェックポイントファイルを削除

    // SaveCheckpoint は CheckpointInfo をファイルに保存する
    func SaveCheckpoint(repoRoot string, info *CheckpointInfo) error

    // LoadCheckpoint は CheckpointInfo をファイルから読み込む
    func LoadCheckpoint(repoRoot string) (*CheckpointInfo, error)

    // RemoveCheckpoint はチェックポイントファイルを削除する
    func RemoveCheckpoint(repoRoot string) error
    ```

---

#### [NEW] [locale_test.go](file://features/devctl/internal/scaffold/locale_test.go)

*   **Description**: ロケール検出・オーバーレイ解決のテスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestDetectLocale_FromLANG: LANG=ja_JP.UTF-8 → "ja"
    // - TestDetectLocale_FromLCAll: LC_ALL=en_US.UTF-8 → "en"
    // - TestDetectLocale_Empty: 環境変数なし → ""
    // - TestDetectLocale_ExplicitFlag: --lang ja → "ja"
    // - TestMergeLocaleFiles_WithOverlay: base + locale.ja → マージ結果
    // - TestMergeLocaleFiles_NoOverlay: base のみ → base そのまま
    // - TestMergeLocaleFiles_PartialOverlay: 一部ファイルのみオーバーレイ
    ```

#### [NEW] [locale.go](file://features/devctl/internal/scaffold/locale.go)

*   **Description**: ロケール検出・テンプレートの多言語オーバーレイ解決。
*   **Technical Design**:
    ```go
    package scaffold

    // DetectLocale は実行環境のロケールを検出する
    // 優先順位: explicitLang > LC_ALL > LANG > Windows システムロケール
    // 返却値: "ja", "en" 等の言語コード（2文字）。検出不可なら ""
    func DetectLocale(explicitLang string) string

    // MergeLocaleFiles は base ファイル群に locale オーバーレイを適用する
    // localeDir = "{template_ref}/locale.{lang}"
    // locale ディレクトリが存在しない場合は base をそのまま返す
    // 存在する場合は同パスのファイルを locale 版で上書き
    func MergeLocaleFiles(baseFiles []DownloadedFile,
        localeFiles []DownloadedFile) []DownloadedFile
    // ロジック:
    //   1. base ファイルを map[RelativePath]DownloadedFile に変換
    //   2. locale ファイルを走査し、同じ RelativePath があれば上書き
    //   3. locale のみに存在するファイルは追加
    //   4. map から slice に戻して返却
    ```

---

#### [NEW] [spinner_test.go](file://features/devctl/internal/scaffold/spinner_test.go)

*   **Description**: スピナーのテスト（出力確認）。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestSpinner_StartStop: スピナー開始・停止がパニックしない
    // - TestSpinner_UpdateMessage: メッセージ更新
    ```

#### [NEW] [spinner.go](file://features/devctl/internal/scaffold/spinner.go)

*   **Description**: CLI 上のスピナー（Loading アニメーション）表示。
*   **Technical Design**:
    ```go
    package scaffold

    // Spinner はターミナルにスピナーアニメーションを表示する
    type Spinner struct {
        writer  io.Writer
        message string
        stop    chan struct{}
        done    chan struct{}
    }

    // 自前で実装（外部依存を避ける）:
    //   frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
    //   interval: 100ms

    // NewSpinner はスピナーを生成する
    func NewSpinner(w io.Writer) *Spinner

    // Start はスピナーの表示を開始する
    func (s *Spinner) Start(message string)

    // UpdateMessage はスピナーのメッセージを更新する
    func (s *Spinner) UpdateMessage(message string)

    // Stop はスピナーの表示を停止する
    func (s *Spinner) Stop()
    ```

---

#### [NEW] [template_test.go](file://features/devctl/internal/scaffold/template_test.go)

*   **Description**: Go template 処理のテスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestProcessTemplate_Simple: "Hello {{.Name}}" → "Hello world"
    // - TestProcessTemplate_NoVars: 変数なしテンプレート → そのまま
    // - TestProcessTemplate_MissingVar: 必須変数不足 → エラー
    // - TestProcessTemplatePath: パス内の {{.Name}} → 展開
    ```

#### [NEW] [template.go](file://features/devctl/internal/scaffold/template.go)

*   **Description**: Go template によるオプション値の展開処理。
*   **Technical Design**:
    ```go
    package scaffold

    // ProcessTemplate はテンプレート文字列をオプション値で展開する
    func ProcessTemplate(tmplContent string, values map[string]string) (string, error)
    // text/template を使用。Missing key はエラーにする（option.missingkey=error）

    // ProcessTemplatePath はパス文字列内のテンプレート変数を展開する
    func ProcessTemplatePath(path string, values map[string]string) (string, error)
    // base_dir 等のパスに含まれる {{.Name}} を展開

    // CollectOptionValues は Options 定義に基づいてオプション値を収集する
    // コマンドライン引数に未指定の必須オプションはインタラクティブに入力を促す
    func CollectOptionValues(options []Option, provided map[string]string,
        reader io.Reader, writer io.Writer) (map[string]string, error)
    // ロジック:
    //   1. provided に値があればそれを使用
    //   2. required かつ未提供 → reader からインタラクティブ入力
    //      プロンプト: "? {Name} ({Description}): "
    //   3. default があり未提供 → デフォルト値を使用
    //   4. required かつ入力なし → エラー
    ```

---

#### [NEW] [plan_test.go](file://features/devctl/internal/scaffold/plan_test.go)

*   **Description**: 実行計画の表示フォーマットのテスト。
*   **Technical Design**:
    ```go
    package scaffold

    // テスト:
    // - TestPrintPlan_CreateOnly: 新規作成のみ → 出力確認
    // - TestPrintPlan_WithConflicts: skip/overwrite/error → 各ポリシー表示
    // - TestPrintPlan_WithPostActions: gitignore エントリ → 表示
    ```

#### [NEW] [plan.go](file://features/devctl/internal/scaffold/plan.go)

*   **Description**: 実行計画の表示フォーマッター。
*   **Technical Design**:
    ```go
    package scaffold

    // PrintPlan は実行計画を人間可読な形式で出力する
    func PrintPlan(plan *Plan, w io.Writer)
    // 出力フォーマット例:
    //   Scaffold: "default"
    //
    //   Files to create:
    //     [CREATE] features/README.md
    //     [CREATE] prompts/phases/README.md
    //     [SKIP]   shared/README.md (already exists)
    //
    //   Post-actions:
    //     [GITIGNORE] Add "work/*" to .gitignore
    //
    //   Summary: 10 files to create, 2 files to skip, 1 post-action
    ```

---

### go.mod 更新

#### [MODIFY] [go.mod](file://features/devctl/go.mod)

*   **Description**: 新しい依存関係は不要。標準ライブラリ (`net/http`, `encoding/json`, `encoding/base64`, `text/template`, `crypto/sha256`) のみ使用。

## Step-by-Step Implementation Guide

> [!NOTE]
> TDD 方式: 各ステップで `_test.go` を先に作成し、失敗を確認してから実装する。

### Step 1: データモデル定義 + カタログパース (R1, R4, R5, R6)

1. `internal/scaffold/catalog_test.go` を作成
   - `TestParseCatalog`, `TestResolvePattern_*`, `TestCheckRequirements_*` のテストケースを定義
2. `internal/scaffold/catalog.go` を作成
   - `Catalog`, `ScaffoldEntry`, `Requirements`, `Option` 構造体を定義
   - `ParseCatalog`, `ResolvePattern`, `CheckRequirements`, `ListScaffolds` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 2: 配置定義パース (R3, R7)

1. `internal/scaffold/placement_test.go` を作成
   - `TestParsePlacement_*` のテストケースを定義
2. `internal/scaffold/placement.go` を作成
   - `Placement`, `TemplateConfig`, `FileMapping`, `PostActions` 構造体を定義
   - `ParsePlacement` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 3: ダウンローダー (R5)

1. `internal/scaffold/downloader_test.go` を作成
   - `httptest.Server` でモック API サーバーを構築
   - `TestGitHubDownloader_FetchFile_*`, `TestGitHubDownloader_FetchDirectory_*` を定義
2. `internal/scaffold/downloader.go` を作成
   - `Downloader` インターフェース、`GitHubDownloader` 構造体
   - `FetchFile`, `FetchDirectory` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 4: ロケール解決 (R14)

1. `internal/scaffold/locale_test.go` を作成
   - `TestDetectLocale_*`, `TestMergeLocaleFiles_*` を定義
2. `internal/scaffold/locale.go` を作成
   - `DetectLocale`, `MergeLocaleFiles` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 5: テンプレートエンジン (R10)

1. `internal/scaffold/template_test.go` を作成
   - `TestProcessTemplate_*`, `TestCollectOptionValues_*` を定義
2. `internal/scaffold/template.go` を作成
   - `ProcessTemplate`, `ProcessTemplatePath`, `CollectOptionValues` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 6: ファイル配置 + 後処理 (R2, R3, R7)

1. `internal/scaffold/applier_test.go` を作成
   - `TestApplyFiles_*`, `TestApplyGitignore_*`, `TestBuildPlan_*` を定義
2. `internal/scaffold/applier.go` を作成
   - `ApplyFiles`, `ApplyPostActions`, `BuildPlan` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 7: 実行計画表示 (R8)

1. `internal/scaffold/plan_test.go` を作成
   - `TestPrintPlan_*` を定義
2. `internal/scaffold/plan.go` を作成
   - `PrintPlan` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 8: チェックポイント / ロールバック (R9)

1. `internal/scaffold/checkpoint_test.go` を作成
   - `TestCreateCheckpoint_*`, `TestRollback_*` を定義
2. `internal/scaffold/checkpoint.go` を作成
   - `CreateCheckpoint`, `Rollback`, `SaveCheckpoint`, `LoadCheckpoint`, `RemoveCheckpoint` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 9: スピナー (R13)

1. `internal/scaffold/spinner_test.go` を作成
   - `TestSpinner_StartStop`, `TestSpinner_UpdateMessage` を定義
2. `internal/scaffold/spinner.go` を作成
   - `Spinner` 構造体、`NewSpinner`, `Start`, `UpdateMessage`, `Stop` を実装
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 10: オーケストレーション + コマンド登録 (R1, R8, R11, R12)

1. `internal/scaffold/scaffold_test.go` を作成
   - `TestRun_*` のテストケースを定義（Downloader モック使用）
2. `internal/scaffold/scaffold.go` を作成
   - `Run`, `Apply`, `Rollback`, `List`, `PrintPlan` を実装
3. `cmd/scaffold.go` を作成
   - コマンド定義、フラグ登録、`runScaffold` を実装
4. `cmd/root.go` の `init()` に `rootCmd.AddCommand(scaffoldCmd)` を追加
5. ビルド & テスト: `./scripts/process/build.sh`

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    全単体テストが PASS することを確認。特に以下のテストファイル:
    - `internal/scaffold/catalog_test.go`
    - `internal/scaffold/placement_test.go`
    - `internal/scaffold/downloader_test.go`
    - `internal/scaffold/locale_test.go`
    - `internal/scaffold/template_test.go`
    - `internal/scaffold/applier_test.go`
    - `internal/scaffold/plan_test.go`
    - `internal/scaffold/checkpoint_test.go`
    - `internal/scaffold/spinner_test.go`
    - `internal/scaffold/scaffold_test.go`

2. **ヘルプ出力確認**:
    ビルド後に以下を実行し、コマンドが正しく登録されていることを確認:
    ```bash
    ./features/devctl/devctl scaffold --help
    ```

## Documentation

#### [MODIFY] [README.md](file://features/devctl/README.md)
*   **更新内容**: `## CLI Commands` セクションに `scaffold` コマンドの説明を追加。使用例とフラグ一覧を記載。
