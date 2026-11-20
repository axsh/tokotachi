# 000-GitLab-MR-Support

> **Source Specification**: [000-GitLab-Command-Inventory.md](file://prompts/phases/000-foundation/branches/feat-adjust-gitlab/ideas/000-GitLab-Command-Inventory.md)

## Goal Description

`tt pr` / `tt list(--update)` / `tt doctor` を、ホスト種別に応じて GitHub (`gh`) と GitLab (`glab`) の両方で動作させる。ホスト解決は `TT_FORGE` 明示指定を最優先し、未設定時のみ origin URL から自動検出する。Change Request（PR/MR）操作は新規パッケージ `internal/forge` に抽象化し、既存の GitHub 専用呼び出しを置き換える。`CodeStatusPR` / `PRCreatedAt` は後方互換のため維持し、GitLab MR も同一ステータスとして扱う。`tt scaffold` の取得先（GitHub Contents API）は変更しない。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `tt pr` を GitLab MR 作成に対応 | Proposed Changes > forge client / action `pr.go` / cmd `pr.go` |
| R2: `tt list` の PR 判定を GitLab MR 判定に対応 | Proposed Changes > forge client / codestatus / cmd `list.go` / `update_code_status.go` |
| R3: `tt doctor` が `glab` を診断 | Proposed Changes > doctor `checks.go` / `doctor.go` |
| R4: `TT_CMD_GLAB` 導入 | Proposed Changes > cmd `common.go` / forge Options |
| R4: origin URL 自動検出 | Proposed Changes > forge `host.go` `DetectHost` |
| R4: `TT_FORGE` 明示指定（URL に gitlab が無くても可） | Proposed Changes > forge `ParseForgeOverride` / `ResolveHost` |
| R4: 優先順位 `TT_FORGE` > URL > unknown→GitHub 経路 | Proposed Changes > forge `ResolveHost` / `NewClient` |
| 状態モデル互換 (`status: pr`) | state 変更なし。MR を `CodeStatusPR` にマップ |
| `tt scaffold` 取得先は変更しない | github Contents API / `resolveToken` を触らない |
| release / インストーラは対象外 | 本計画では実装しない |
| ドキュメント追随 | Proposed Changes > Documentation |

## Proposed Changes

### forge パッケージ（新規）

#### [NEW] [host_test.go](file://internal/forge/host_test.go)

*   **Description**: ホスト判定と `TT_FORGE` 優先の単体テスト（TDD 先書き）
*   **Technical Design**:
    ```go
    func TestDetectHost(t *testing.T) {
        // https://github.com/axsh/tokotachi.git -> HostGitHub
        // git@github.com:axsh/tokotachi.git -> HostGitHub
        // https://gitlab.com/group/proj.git -> HostGitLab
        // git@gitlab.com:group/proj.git -> HostGitLab
        // https://gitlab.example.com/g/p.git -> HostGitLab
        // https://git.example.com/g/p.git -> HostUnknown
        // "" -> HostUnknown
    }
    func TestParseForgeOverride(t *testing.T) {
        // "gitlab"/"GitLab"/"GITHUB" -> 対応 Host
        // "" -> HostUnknown, nil error（オーバーライドなし）
        // "bitbucket"/"foo" -> error
    }
    func TestResolveHost_OverrideBeatsURL(t *testing.T) {
        // TT_FORGE=gitlab + https://git.example.com/g/p.git -> HostGitLab
        // TT_FORGE=github + https://gitlab.com/g/p.git -> HostGitHub
        // unset + https://git.example.com/g/p.git -> HostUnknown
        // unset + https://gitlab.com/g/p.git -> HostGitLab
        // invalid TT_FORGE -> error
    }
    ```
*   **Logic**: テーブル駆動。`t.Setenv` は `DetectHostFromRepo` 統合時のみ使用し、`ResolveHost` 自体は引数注入でテストする。

#### [NEW] [host.go](file://internal/forge/host.go)

*   **Description**: ホスト種別・URL 検出・明示指定
*   **Technical Design**:
    ```go
    package forge

    type Host string

    const (
        HostGitHub  Host = "github"
        HostGitLab  Host = "gitlab"
        HostUnknown Host = "unknown"
    )

    const EnvForge = "TT_FORGE" // "github" | "gitlab" (case-insensitive)

    func DetectHost(remoteURL string) Host
    func ParseForgeOverride(value string) (Host, error)
    func ResolveHost(override, remoteURL string) (Host, error)
    func DetectHostFromRepo(repoRoot, gitCmd string) (Host, string, error)
    ```
*   **Logic**:
    *   `DetectHost`: SSH (`git@host:path`) / HTTPS をパース。hostname が `github.com` または `.github.com` で終わる → `HostGitHub`。hostname が `gitlab.com` または hostname に `gitlab` を含む → `HostGitLab`。それ以外 → `HostUnknown`
    *   `ParseForgeOverride`: trim + ToLower。`github`/`gitlab` なら対応 Host。空なら `(HostUnknown, nil)`。その他は `fmt.Errorf("invalid TT_FORGE %q: want github or gitlab", value)`
    *   `ResolveHost`: (1) override が HostGitHub/HostGitLab ならそれを返す（URL 無視）(2) override パース error なら error を返す (3) 否则 `DetectHost(remoteURL)`
    *   `DetectHostFromRepo`: `override := os.Getenv(EnvForge)`。`git remote get-url origin` を `Dir=repoRoot` で実行。remote 失敗時でも有効な override があれば `(host, "", nil)`。両方失敗なら `(HostUnknown, "", err)` または override 無効時は `ResolveHost(override, "")`

#### [NEW] [client_test.go](file://internal/forge/client_test.go)

*   **Description**: Client 選択と CLI 引数の単体テスト
*   **Technical Design**:
    ```go
    func TestNewClient_SelectsGitHub(t *testing.T)
    func TestNewClient_SelectsGitLab(t *testing.T)
    func TestNewClient_UnknownHost_DefaultsGitHub(t *testing.T)
    func TestGitHubAdapter_CreateChangeRequest_InvokesGhPrCreate(t *testing.T)
    func TestGitHubAdapter_FindOpen_InvokesGhPrList(t *testing.T)
    func TestGitLabAdapter_CreateChangeRequest_InvokesGlabMrCreate(t *testing.T)
    func TestGitLabAdapter_FindOpen_ParsesJSON(t *testing.T)
    ```
*   **Logic**:
    *   `cmdexec.Runner` + `Recorder`（DryRun）で引数を検証
    *   GitHub create: `gh pr create`、FindOpen: `gh pr list --head <branch> --json number,createdAt --limit 1`
    *   GitLab create: `glab mr create`、FindOpen: `glab mr list --source-branch <branch> --output json`
    *   GitLab JSON フィクスチャ例: `[{"iid":42,"created_at":"2026-01-02T03:04:05Z"}]` → `ChangeRequestInfo{Number:42, CreatedAt:...}`

#### [NEW] [client.go](file://internal/forge/client.go)

*   **Description**: PR/MR 共通インターフェースとホスト別実装
*   **Technical Design**:
    ```go
    package forge

    import (
        "time"
        "github.com/axsh/tokotachi/internal/cmdexec"
    )

    type ChangeRequestInfo struct {
        Number    int
        CreatedAt time.Time
    }

    type Client interface {
        Host() Host
        CreateChangeRequest(workDir string) error
        FindOpenChangeRequest(workDir, branch string) (*ChangeRequestInfo, error)
    }

    type Options struct {
        Host      Host
        CmdRunner *cmdexec.Runner
        GhCmd     string  // default: ResolveCommand("TT_CMD_GH", "gh")
        GlabCmd   string  // default: ResolveCommand("TT_CMD_GLAB", "glab")
    }

    func NewClient(opts Options) (Client, error)

    type githubClient struct {
        runner *cmdexec.Runner
        ghCmd  string
        host   Host
    }
    type gitlabClient struct {
        runner  *cmdexec.Runner
        glabCmd string
        host    Host
    }
    ```
*   **Logic**:
    *   `NewClient`: `HostGitLab` → `gitlabClient`。`HostGitHub` / `HostUnknown` → `githubClient`（後方互換）
    *   `GhCmd`/`GlabCmd` が空ならそれぞれ `ResolveCommand` で埋める
    *   `CmdRunner` 必須。nil なら error
    *   `githubClient.CreateChangeRequest`: `RunInteractiveWithOpts(RunOption{Dir: workDir}, ghCmd, "pr", "create")`
    *   `githubClient.FindOpenChangeRequest`: QuietCmd で `pr list --head branch --json number,createdAt --limit 1`。`[]`/空なら nil。パースして先頭を返す
    *   `gitlabClient.CreateChangeRequest`: `RunInteractiveWithOpts(..., glabCmd, "mr", "create")`
    *   `gitlabClient.FindOpenChangeRequest`: QuietCmd で `mr list --source-branch branch --output json`。配列先頭の `iid`→`Number`、`created_at`→`CreatedAt`。空配列なら nil

#### [NEW] [resolve_test.go](file://internal/forge/resolve_test.go)

*   **Description**: `NewClientForRepo` のオーバーライド優先テスト
*   **Technical Design**:
    ```go
    func TestNewClientForRepo_TTForgeGitlab_OverridesNonGitlabURL(t *testing.T)
    func TestNewClientForRepo_InvalidTTForge_ReturnsError(t *testing.T)
    ```

#### [NEW] [resolve.go](file://internal/forge/resolve.go)

*   **Description**: repoRoot からホスト解決して Client を生成
*   **Technical Design**:
    ```go
    func NewClientForRepo(repoRoot string, runner *cmdexec.Runner) (Client, Host, error) {
        gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
        host, _, err := DetectHostFromRepo(repoRoot, gitCmd)
        if err != nil {
            return nil, HostUnknown, err // invalid TT_FORGE 等は silent fallback しない
        }
        c, err := NewClient(Options{Host: host, CmdRunner: runner})
        return c, host, err
    }
    ```

---

### action パッケージ

#### [NEW] [pr_test.go](file://pkg/action/pr_test.go)

*   **Description**: `Runner.PR` がホスト別に CLI を切替えるテスト
*   **Technical Design**:
    ```go
    func TestPR_GitHub_UsesGhPrCreate(t *testing.T)
    func TestPR_GitLab_UsesGlabMrCreate(t *testing.T)
    ```
*   **Logic**: 一時 git repo の origin を差し替え、または forge Client を注入可能な設計にする。確定方針: `Runner.PR` は `forge.NewClientForRepo` を呼ぶため、テストでは origin URL と `TT_FORGE` を制御し、`cmdexec.Recorder` + DryRun で `gh pr create` / `glab mr create` と `Dir=worktreePath` を検証する。

#### [MODIFY] [pr.go](file://pkg/action/pr.go)

*   **Description**: `github.Client.CreatePR` を forge 経由に置換
*   **Technical Design**:
    ```go
    // PR creates a Pull Request (GitHub) or Merge Request (GitLab).
    func (r *Runner) PR(repoRoot, worktreePath string) error {
        r.Logger.Info("Creating change request from %s...", worktreePath)
        client, host, err := forge.NewClientForRepo(repoRoot, r.CmdRunner)
        if err != nil {
            return fmt.Errorf("forge client creation failed: %w", err)
        }
        r.Logger.Info("Detected forge host: %s", host)
        if err := client.CreateChangeRequest(worktreePath); err != nil {
            return fmt.Errorf("change request create failed: %w", err)
        }
        return nil
    }
    ```
*   **Logic**: シグネチャを `PR(repoRoot, worktreePath string)` に拡張。成功後の CodeStatus 更新は cmd 層のまま（`CodeStatusPR`）。

---

### cmd（tt）

#### [MODIFY] [pr.go](file://features/tt/cmd/pr.go)

*   **Description**: Short をホスト中立化し、新シグネチャで呼ぶ
*   **Technical Design**:
    *   `Short: "Create a Pull/Merge Request on the remote forge"`
    *   `ctx.ActionRunner.PR(ctx.RepoRoot, worktreePath)`
*   **Logic**: CodeStatus 更新ロジックは変更しない。

#### [MODIFY] [list.go](file://features/tt/cmd/list.go)

*   **Description**: `--update` 時の Checker を forge 対応にする
*   **Technical Design**:
    ```go
    client, host, err := forge.NewClientForRepo(repoRoot, runner)
    // err 時は logger.Warn して従来の gh-only Checker にフォールバックせず、
    // invalid TT_FORGE はエラーを呼び出し元へ返す（確定: --update は error 返却）。
    // NewClient 成功時:
    checker := &codestatus.Checker{
        GitCmd:   gitCmd,
        GhCmd:    cmdexec.ResolveCommand("TT_CMD_GH", "gh"),
        GlabCmd:  cmdexec.ResolveCommand("TT_CMD_GLAB", "glab"),
        Host:     host,
        Forge:    client,
        RepoRoot: repoRoot,
        Timeout:  30 * time.Second,
    }
    ```
*   **Logic**: `GhCmd` 単独での findPR 直呼びをやめ、`Forge` 経由にする。

#### [MODIFY] [update_code_status.go](file://features/tt/cmd/update_code_status.go)

*   **Description**: バックグラウンド更新でも同様に forge Client をセット
*   **Logic**: `list.go` と同じ Checker 初期化。`NewClientForRepo` 失敗時は Warn して return nil（バックグラウンドのためプロセスを落とさない）。ただし invalid `TT_FORGE` は Warn に理由を出す。

#### [MODIFY] [common.go](file://features/tt/cmd/common.go)

*   **Description**: 環境変数を knownEnvVars に追加
*   **Technical Design**:
    ```go
    {"TT_CMD_GH", "gh"},
    {"TT_CMD_GLAB", "glab"},
    {"TT_FORGE", ""}, // empty = auto-detect; "github" | "gitlab"
    ```

---

### codestatus パッケージ

#### [NEW] [codestatus_test.go](file://pkg/codestatus/codestatus_test.go)

*   **Description**: ホスト別 open CR 検出とフォールバックの単体テスト
*   **Technical Design**:
    ```go
    func TestResolve_GitHub_OpenPR(t *testing.T)
    func TestResolve_GitLab_OpenMR(t *testing.T)
    func TestResolve_ForgeCLIFailure_FallsBackToHosted(t *testing.T)
    func TestResolve_NoRemoteBranch_LocalOrDeleted(t *testing.T)
    ```
*   **Logic**: `Forge` にモック Client（または Recorder 付き実 Client）を注入。GitLab で MR あり → `CodeStatusPR` + `PRCreatedAt`。CLI error → `CodeStatusHosted`。remote なし + prev PR/hosted → `deleted`。

#### [MODIFY] [codestatus.go](file://pkg/codestatus/codestatus.go)

*   **Description**: `gh pr list` 直呼びを forge 経由に変更
*   **Technical Design**:
    ```go
    type Checker struct {
        GitCmd   string
        GhCmd    string // 後方互換で残す（Forge nil 時の legacy 経路）
        GlabCmd  string
        Host     forge.Host
        Forge    forge.Client // 非 nil なら FindOpenChangeRequest を使用
        RepoRoot string
        Timeout  time.Duration
    }

    func (c *Checker) findPR(ctx context.Context, branch string) (*prInfo, error) {
        if c.Forge != nil {
            info, err := c.Forge.FindOpenChangeRequest(c.RepoRoot, branch)
            if err != nil {
                return nil, err
            }
            if info == nil {
                return nil, nil
            }
            return &prInfo{Number: info.Number, CreatedAt: info.CreatedAt}, nil
        }
        // legacy: 既存の gh pr list 実装を維持
    }
    ```
*   **Logic**: `Resolve` Step1（`git ls-remote`）は不変。Step2 のみ forge 化。

---

### doctor パッケージ

#### [MODIFY] [checks_test.go](file://pkg/doctor/checks_test.go)

*   **Description**: ホスト別ツール診断テストを追加
*   **Technical Design**:
    ```go
    func TestCheckExternalTools_GitHub_ChecksGh(t *testing.T)
    func TestCheckExternalTools_GitLab_ChecksGlab(t *testing.T)
    func TestCheckExternalTools_Unknown_ChecksBothAsWarn(t *testing.T)
    func TestCheckExternalTools_TTForgeOverride_GitLabWithoutGitlabInURL(t *testing.T)
    ```

#### [MODIFY] [checks.go](file://pkg/doctor/checks.go)

*   **Description**: 固定 `gh` チェックをホスト依存に変更
*   **Technical Design**:
    ```go
    func checkExternalTools(checker ToolChecker, host forge.Host) []CheckResult
    // git, docker: required（従来どおり）
    // HostGitHub: gh warn-only
    // HostGitLab: glab warn-only
    // HostUnknown: gh と glab 両方 warn-only
    ```
*   **Logic**:
    *   チェック対象コマンド名は `cmdexec.ResolveCommand` のデフォルト名（`gh` / `glab`）を使用。環境変数でパスが変わっていても `CheckTool` はベース名でよい（既存と同様）
    *   fixHint: GitHub → `https://cli.github.com/ (only required for 'tt pr')`、GitLab → glab インストール案内 + `only required for 'tt pr'`

#### [MODIFY] [doctor.go](file://pkg/doctor/doctor.go)

*   **Description**: `Run` 内でホスト解決して `checkExternalTools` に渡す
*   **Technical Design**:
    ```go
    gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
    host, _, err := forge.DetectHostFromRepo(opts.RepoRoot, gitCmd)
    if err != nil {
        host = forge.HostUnknown // doctor は診断継続（invalid TT_FORGE は結果に Warn メッセージを足してもよい）
    }
    report.Results = append(report.Results, checkExternalTools(opts.ToolChecker, host)...)
    ```

---

### github パッケージ

#### [MODIFY] [github.go](file://internal/github/github.go)

*   **Description**: scaffold 用 Contents API / `resolveToken` は**変更しない**。`CreatePR` / `ListPRs` は互換のため残すが、`pkg/action` と `pkg/codestatus` からは呼ばない。
*   **Logic**: PR/MR 本体ロジックは forge の githubClient に置く。github 側の CreatePR/ListPRs は既存実装を維持（他参照が無いことを確認し、必要なら deprecate コメント）。

---

### 統合テスト

#### [NEW] [tt_forge_host_test.go](file://tests/tt/tt_forge_host_test.go)

*   **Description**: doctor 経由でホスト解決が CLI チェックに反映されることを検証（実 GitLab/`glab` ログインは不要）
*   **Technical Design**:
    ```go
    func TestTtDoctor_GitLabRemote_ChecksGlab(t *testing.T)
    func TestTtDoctor_GitHubRemote_ChecksGh(t *testing.T)
    func TestTtDoctor_TTForgeGitlab_OverridesNonGitlabURL(t *testing.T)
    ```
*   **Logic**:
    1. 一時ディレクトリで `git init` + `git remote add origin <url>`
    2. `runTTInDir` で `doctor --json`（必要なら最小 features 構成を用意。doctor が features 必須ならダミー `features/x/` を作る）
    3. JSON `results` からツール名 `glab` / `gh` の有無をアサート
    4. Override ケース: origin=`https://git.example.com/g/p.git`、環境変数 `TT_FORGE=gitlab` で `glab` が含まれること

#### [MODIFY] [tt_doctor_test.go](file://tests/tt/tt_doctor_test.go)

*   **Description**: 既存 doctor テストが引き続き PASS すること（回帰）。必要なら JSON 内に `gh` または `glab` のいずれかが含まれることを緩くアサート。

#### [MODIFY] [tt_list_code_test.go](file://tests/tt/tt_list_code_test.go)

*   **Description**: `TestTtListCode_UpdateFlagAccepted` の回帰を維持。実 PR/MR 作成 E2E は追加しない（既存 GitHub 側にも `tt pr` E2E が無いため。CLI 切替の証拠は単体 + doctor 統合で担保）。Verification Plan にその理由を明記する。

---

## Step-by-Step Implementation Guide

1. [x] **TDD: forge host**
    *   Add `internal/forge/host_test.go`（失敗確認）→ implement `host.go`（`DetectHost` / `ParseForgeOverride` / `ResolveHost` / `DetectHostFromRepo`）。
2. [x] **TDD: forge client**
    *   Add `client_test.go` → implement `client.go`（github/gitlab adapters）。
    *   Add `resolve_test.go` → implement `resolve.go`（`NewClientForRepo`、`TT_FORGE` 優先）。
3. [x] **codestatus**
    *   Add `pkg/codestatus/codestatus_test.go` → modify `codestatus.go`（`Forge` / `Host` / `GlabCmd`、`findPR`）。
4. [x] **action / cmd pr**
    *   Add `pkg/action/pr_test.go` → modify `pkg/action/pr.go` シグネチャと forge 呼び出し。
    *   Modify `features/tt/cmd/pr.go`。
5. [x] **list / update_code_status**
    *   Modify `list.go` / `update_code_status.go` の Checker 初期化。
    *   Modify `common.go` に `TT_CMD_GLAB` / `TT_FORGE`。
6. [x] **doctor**
    *   Extend `checks_test.go` → modify `checks.go` / `doctor.go`。
7. [x] **統合テスト**
    *   Add `tests/tt/tt_forge_host_test.go`、必要なら `tt_doctor_test.go` を軽く更新。
8. [x] **ドキュメント**
    *   Update `docs/manual/tt-user-manual.md` / `README.md`。
9. [x] **Verification Plan を実行**する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**（関連のみ）:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories tt --specify "Doctor|List|Forge|GitLab|Host"
    ```
    *   **Log Verification**:
        *   `tt doctor` 系が exit 0
        *   `TestTtDoctor_GitLabRemote_ChecksGlab` 等で JSON に `glab` が含まれる
        *   `TestTtDoctor_TTForgeGitlab_OverridesNonGitlabURL` が PASS
        *   既存 `tt list --update` が exit 0（回帰）
        *   scaffold テストは本指定に含めない（取得先未変更・長時間・API 依存）

3.  **E2E / Integration（新規）**:

    #### [NEW] [tests/tt/tt_forge_host_test.go](file://tests/tt/tt_forge_host_test.go)
    *   **テストケース**: `TestTtDoctor_GitLabRemote_ChecksGlab` / `TestTtDoctor_GitHubRemote_ChecksGh` / `TestTtDoctor_TTForgeGitlab_OverridesNonGitlabURL`
    *   **検証ポイント**: ホスト解決結果が doctor の外部ツール診断に反映されること

    **`tt pr` 実作成 E2E を追加しない理由**: 既存 GitHub 経路にも `tt pr` / `gh pr create` の実ホスト E2E が無く、対話的 CLI と認証が必要。切替の正しさは forge/action 単体（Recorder）と doctor 統合で証拠化する。

### テスト項目設計（§11）

ボトムアップ順序:

1. `DetectHost` / `ParseForgeOverride` / `ResolveHost`（末端）
2. forge Client create/list 引数・JSON パース
3. `codestatus.Resolve` / `action.PR` / `doctor.checkExternalTools`
4. `tt doctor` 統合（`tt_forge_host_test.go`）

観点チェックリスト:

| # | 観点 | 本計画での担保 |
|---|---|---|
| 1 | 正常系 | GitHub URL / GitLab URL / `TT_FORGE` 明示 |
| 2 | 異常系 | 不正 `TT_FORGE`、CLI 失敗→hosted フォールバック |
| 3 | 外部連携 | `git remote get-url`、Recorder 上の gh/glab 引数 |
| 4 | データ一貫性 | MR JSON → `ChangeRequestInfo` → `prInfo` / `PRCreatedAt` |
| 5 | 状態遷移 | open MR → `CodeStatusPR` |
| 6 | 設定反映 | `TT_FORGE` / `TT_CMD_GLAB` |
| 7 | 副作用 | scaffold / Contents API 未変更 |

セルフレビュー（§11.4）:

1. **網羅性**: R1–R4 を host → client → action/codestatus/doctor → 統合の順でカバー。実 `tt pr` E2E 欠落は既存 GitHub と同水準であり、Recorder で primary path を証明する。
2. **証拠の十分性**: CLI 名と引数、JSON フィールド、doctor JSON のツール名をアサートする。
3. **迂回排除**: `Host()` / Recorder で githubClient と gitlabClient のどちらが使われたかを確認する。
4. **依存関係**: 末端 forge テスト無しでは上位テストの成功を信頼しない順序で実装・実行する。

### 総合判定プロセス（§12）

全テスト成功後、以下を確認して判定を記録する:

1. スキップ/WARN の有無（特に doctor のツール欠落 WARN は環境依存で許容）
2. ログに意図しない ERROR / panic が無いか
3. GitLab ケースで `gh` 経路へフォールバックしていないか（Recorder / doctor 結果）
4. `TT_FORGE=gitlab` + non-gitlab URL が primary path か
5. `tt scaffold` / github Contents API が差分に含まれていないか
6. 単独 `--specify` 実行でも同結果か

## Documentation

#### [MODIFY] [docs/manual/tt-user-manual.md](file://docs/manual/tt-user-manual.md)

*   **更新内容**:
    *   `pr` 節: GitHub PR / GitLab MR の両対応、自動検出と `TT_FORGE` を記載
    *   環境変数表に `TT_CMD_GLAB`（default `glab`）、`TT_FORGE`（empty / `github` / `gitlab`）を追加
    *   scaffold 節は取得先変更なし（触らないか、必要なら「引き続き GitHub」と明記）

#### [MODIFY] [README.md](file://README.md)

*   **更新内容**: `tt pr` を Pull/Merge Request 対応と記述。GitLab 利用時は `glab`、必要なら `TT_FORGE=gitlab` を短く追記。
