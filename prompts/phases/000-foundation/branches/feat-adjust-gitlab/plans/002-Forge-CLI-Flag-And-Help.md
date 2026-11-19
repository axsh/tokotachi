# 002-Forge-CLI-Flag-And-Help

> **Source Specification**: [003-Forge-CLI-Flag-And-Help.md](file://prompts/phases/000-foundation/branches/feat-adjust-gitlab/ideas/003-Forge-CLI-Flag-And-Help.md)

## Goal Description

`tt pr` / `tt list` / `tt doctor` に CLI フラグ `--forge <github|gitlab>` を追加し、ホスト解決の最優先手段とする。あわせて各コマンドの Help（`Long` / Examples）にホスト解決契約（`--forge` > `TT_FORGE` > origin URL）と関連 env を記載し、人間および LLM が `--help` だけで GitLab 向け実行を構成できるようにする。既存の `TT_FORGE`・origin URL 検出・`TT_CMD_GH` / `TT_CMD_GLAB` は維持する。

## User Review Required

None.

## Requirement Traceability

> **Traceability Check**:
> 仕様書の要件・決定事項を本計画の実装箇所へマッピングする。先送りは無し。

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `--forge` を `pr` / `list` / `doctor` に追加（値 `github`\|`gitlab`、不正は Fail Fast、永続設定しない） | Proposed Changes > cmd `pr.go` / `list.go` / `doctor.go`；forge `ResolveHostPriority` |
| R1: doctor のフラグ不正はコマンド失敗（env 不正の unknown 緩和は env のみ） | Proposed Changes > doctor `doctor.go` `Options.ForgeOverride` |
| R2: 優先順位 `--forge` > `TT_FORGE` > origin URL > unknown | Proposed Changes > forge `host.go` `ResolveHostPriority` / `DetectOpts` |
| R2: 両方指定時は `--forge` が勝つ | Unit: `TestResolveHostPriority`；Integration: `TestTtDoctor_ForgeFlag_BeatsTTForge` |
| R2: hostname に gitlab 無しでも `--forge gitlab` で GitLab 経路 | Integration: `TestTtDoctor_ForgeFlag_OverridesLocalhostURL` |
| R3: Help にコマンド説明・優先順位・値・関連 env・実行例 | Proposed Changes > cmd Help 定数 / `pr.go` / `list.go` / `doctor.go` |
| R4: フラグ未指定時は既存 `TT_FORGE`/URL 挙動を維持 | Unit: 既存 `ResolveHost` / `DetectHostFromRepo` 回帰；`CLIOverride=""` 経路 |
| R4: `TT_CMD_GH` / `TT_CMD_GLAB` の意味は変更しない | 変更対象外（触らない） |
| R5: README / tt-user-manual に `--forge` と更新後優先順位を追記 | Documentation |
| R5: 001 の「`TT_FORGE` 最優先」記述を更新 | Documentation > ideas `001-GitLab-MR-Support.md` |
| 任意: open/merge 等への `--forge` | 対象外（実装しない） |
| 任意: シェル補完 | 対象外（必須ではない） |
| 対応不要: forge クライアント実装やり直し / Docker GitLab 必須化 / scaffold | 実装しない |

## Proposed Changes

### forge パッケージ（解決優先順位）

#### [MODIFY] [host_test.go](file://internal/forge/host_test.go)

*   **Description**: CLI 明示値を最優先とする解決ロジックの単体テスト（TDD 先書き）
*   **Technical Design**:
    ```go
    func TestResolveHostPriority(t *testing.T) {
        // table-driven:
        // cli="gitlab", env="github", url=github.com     -> HostGitLab  (--forge wins)
        // cli="github", env="gitlab", url=gitlab.com     -> HostGitHub
        // cli="",       env="gitlab", url=git.example.com -> HostGitLab  (env wins when no flag)
        // cli="",       env="",       url=gitlab.com      -> HostGitLab  (URL)
        // cli="",       env="",       url=git.example.com -> HostUnknown
        // cli="bogus",  env="gitlab", url=...             -> error (Fail Fast; env ignored)
        // cli="",       env="bogus",  url=...             -> error (既存と同様)
        // cli="  GitLab  ", env="", url=github.com        -> HostGitLab (trim + case-insensitive)
        // cli="", env="", url=""                          -> HostUnknown
    }

    func TestDetectHostFromRepoOpts_CLIOverrideBeatsEnvAndURL(t *testing.T) {
        // temp repo origin=https://github.com/a/b.git
        // t.Setenv(EnvForge, "github")
        // DetectHostFromRepoOpts{CLIOverride:"gitlab"} -> HostGitLab
    }

    func TestDetectHostFromRepoOpts_EmptyCLI_UsesEnv(t *testing.T) {
        // origin=https://git.example.com/g/p.git, TT_FORGE=gitlab, CLIOverride=""
        // -> HostGitLab
    }

    func TestDetectHostFromRepoOpts_InvalidCLI_ReturnsError(t *testing.T) {
        // CLIOverride="bogus" -> error containing "--forge"
    }
    ```
*   **Logic**:
    *   `ResolveHost` 既存テストは維持（単一 override 互換 API）。
    *   新規優先順位は `ResolveHostPriority` / `DetectHostFromRepoOpts` で検証する。
    *   CLI 不正エラー文言は `invalid --forge` を含むこと（env 不正の `invalid TT_FORGE` と区別）。

#### [MODIFY] [host.go](file://internal/forge/host.go)

*   **Description**: CLI 明示オーバーライドを最優先で受け取る解決 API を追加し、既存 API は薄い互換ラッパとして残す
*   **Technical Design**:
    ```go
    // DetectOpts configures host detection from a local git repo.
    type DetectOpts struct {
        RepoRoot    string
        GitCmd      string
        CLIOverride string // --forge value; empty = unset (do not override)
    }

    // ResolveHostPriority applies:
    //   1. cliOverride (--forge) if non-empty after trim
    //   2. envOverride (TT_FORGE) if non-empty after trim
    //   3. DetectHost(remoteURL)
    // Invalid cliOverride => error with "invalid --forge ...".
    // Invalid envOverride => error with existing ParseForgeOverride message ("invalid TT_FORGE ...").
    func ResolveHostPriority(cliOverride, envOverride, remoteURL string) (Host, error)

    // DetectHostFromRepoOpts runs git remote get-url origin then ResolveHostPriority.
    // When remote lookup fails but a valid override (CLI or env) is set, override wins and remoteErr is ignored.
    func DetectHostFromRepoOpts(opts DetectOpts) (Host, string, error)

    // DetectHostFromRepo keeps backward compatibility:
    //   return DetectHostFromRepoOpts(DetectOpts{RepoRoot: repoRoot, GitCmd: gitCmd, CLIOverride: ""})
    func DetectHostFromRepo(repoRoot, gitCmd string) (Host, string, error)

    // ResolveHost keeps backward compatibility (single override = env-style):
    //   return ResolveHostPriority("", override, remoteURL)
    func ResolveHost(override, remoteURL string) (Host, error)
    ```
*   **Logic**:
    *   `ResolveHostPriority`:
        1. `cli := strings.TrimSpace(cliOverride)` が空でない → `ParseForgeOverride(cli)`。成功ならその Host を返す（URL / env 無視）。失敗なら `fmt.Errorf("invalid --forge %q: want github or gitlab", cliOverride)` を返す（元の `ParseForgeOverride` エラーをそのまま返すのではなく、CLI 向け文言に差し替える）。
        2. 否则 `ResolveHost(envOverride, remoteURL)`（既存: env パース → URL 検出）。
    *   `DetectHostFromRepoOpts`:
        1. `envOverride := os.Getenv(EnvForge)`
        2. `remoteURL, remoteErr := getOriginURL(opts.RepoRoot, opts.GitCmd)`
        3. `host, err := ResolveHostPriority(opts.CLIOverride, envOverride, remoteURL)`
        4. `err != nil` なら `(HostUnknown, remoteURL, err)`
        5. `remoteErr != nil` かつ host が GitHub/GitLab なら `(host, "", nil)`（既存と同趣旨）
        6. `remoteErr != nil` かつ host が unknown なら `(HostUnknown, "", remoteErr)`
        7. 成功時 `(host, remoteURL, nil)`
    *   `ParseForgeOverride` 本体の仕様（空 → Unknown/nil、github/gitlab、それ以外エラー）は変更しない。
    *   永続設定ファイルや git config への書き込みは行わない。

#### [MODIFY] [resolve_test.go](file://internal/forge/resolve_test.go)

*   **Description**: `NewClientForRepoOpts` の CLI オーバーライド単体テスト（TDD 先書き）
*   **Technical Design**:
    ```go
    func TestNewClientForRepoOpts_CLIOverrideBeatsTTForgeAndURL(t *testing.T) {
        // origin=https://github.com/a/b.git, TT_FORGE=github, CLIOverride=gitlab
        // -> host HostGitLab, client.Host() HostGitLab
    }

    func TestNewClientForRepoOpts_EmptyCLI_UsesExistingBehavior(t *testing.T) {
        // TT_FORGE=gitlab + non-gitlab URL -> HostGitLab（既存 TestNewClientForRepo_TTForgeGitlab と同等）
    }

    func TestNewClientForRepoOpts_InvalidCLI_ReturnsError(t *testing.T) {
        // CLIOverride=bogus -> error
    }
    ```
*   **Logic**: 既存 `NewClientForRepo_*` テストは `CLIOverride=""` 互換経路の回帰として残す。

#### [MODIFY] [resolve.go](file://internal/forge/resolve.go)

*   **Description**: クライアント生成に CLI オーバーライドを渡せるようにする
*   **Technical Design**:
    ```go
    type ClientForRepoOpts struct {
        RepoRoot    string
        Runner      *cmdexec.Runner
        CLIOverride string // --forge; empty = unset
    }

    func NewClientForRepoOpts(opts ClientForRepoOpts) (Client, Host, error) {
        gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
        host, _, err := DetectHostFromRepoOpts(DetectOpts{
            RepoRoot:    opts.RepoRoot,
            GitCmd:      gitCmd,
            CLIOverride: opts.CLIOverride,
        })
        if err != nil {
            return nil, HostUnknown, err
        }
        c, err := NewClient(Options{Host: host, CmdRunner: opts.Runner})
        return c, host, err
    }

    // NewClientForRepo keeps compatibility:
    //   return NewClientForRepoOpts(ClientForRepoOpts{RepoRoot: repoRoot, Runner: runner, CLIOverride: ""})
    func NewClientForRepo(repoRoot string, runner *cmdexec.Runner) (Client, Host, error)
    ```
*   **Logic**: ホスト解決以外（`NewClient` の unknown→GitHub フォールバック等）は変更しない。

### action / doctor（呼び出し側へ CLI 値を伝播）

#### [MODIFY] [pr.go](file://pkg/action/pr.go) および [runner.go](file://pkg/action/runner.go)

*   **Description**: `tt pr` から渡された `--forge` を `NewClientForRepoOpts` へ接続する
*   **Technical Design**:
    ```go
    // Runner
    type Runner struct {
        Logger        pkglog.Logger
        DryRun        bool
        CmdRunner     *cmdexec.Runner
        ForgeOverride string // from --forge; empty = unset
    }

    func (r *Runner) PR(repoRoot, worktreePath string) error {
        client, host, err := forge.NewClientForRepoOpts(forge.ClientForRepoOpts{
            RepoRoot:    repoRoot,
            Runner:      r.CmdRunner,
            CLIOverride: r.ForgeOverride,
        })
        // ... existing logging + CreateChangeRequest
    }
    ```
*   **Logic**:
    *   `ForgeOverride` が空なら既存どおり env/URL。
    *   不正値は `NewClientForRepoOpts` のエラーをそのまま返し、コマンド失敗とする。
    *   Info ログ `Detected forge host: %s` は維持（統合テストで経路観測に利用可）。

#### [MODIFY] [doctor.go](file://pkg/doctor/doctor.go)（必要なら [checks_test.go](file://pkg/doctor/checks_test.go) に Run 経路テスト追加）

*   **Description**: doctor に `ForgeOverride` を追加し、CLI 不正は Fail Fast、env 不正のみ unknown 緩和
*   **Technical Design**:
    ```go
    type Options struct {
        RepoRoot      string
        FeatureFilter string
        ToolChecker   ToolChecker
        Fix           bool
        ForgeOverride string // from --forge; empty = unset
    }

    func Run(opts Options) (*Report, error) {
        gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
        host, _, hostErr := forge.DetectHostFromRepoOpts(forge.DetectOpts{
            RepoRoot:    opts.RepoRoot,
            GitCmd:      gitCmd,
            CLIOverride: opts.ForgeOverride,
        })
        if hostErr != nil {
            if strings.TrimSpace(opts.ForgeOverride) != "" {
                return nil, hostErr // R1: CLI invalid => Fail Fast
            }
            host = forge.HostUnknown // existing: invalid/missing env softens to unknown
        }
        // ... rest unchanged
    }
    ```
*   **Logic**:
    *   `ForgeOverride` 未設定時の挙動は現行どおり（`TT_FORGE` 不正 → unknown で診断継続）。
    *   `ForgeOverride` 設定時の不正値はレポートを返さずエラー終了。
    *   単体で `Run` の Fail Fast を検証するテストを追加する（例: `TestRun_InvalidForgeOverride_ReturnsError`）。モック `ToolChecker` は既存パターンを流用。

### tt CLI（フラグ・Help）

#### [MODIFY] [common.go](file://features/tt/cmd/common.go)（共有 Help 文言）

*   **Description**: `pr` / `list` / `doctor` で共有するホスト解決 Help テキストを定義する
*   **Technical Design**:
    ```go
    // forgeHostResolveHelp is appended to Long help of pr / list / doctor (R3).
    const forgeHostResolveHelp = `
    Host resolution (highest priority first):
      1. --forge <github|gitlab>   (this process only; does not persist)
      2. TT_FORGE                  (environment variable)
      3. git remote get-url origin (hostname heuristics)

    --forge values: github | gitlab (case-insensitive).
    Invalid --forge values fail the command (not ignored).

    Related environment variables:
      TT_FORGE      Explicit forge host when --forge is omitted (github|gitlab)
      TT_CMD_GH     Path/name of GitHub CLI (default: gh)
      TT_CMD_GLAB   Path/name of GitLab CLI (default: glab)
    `

    const forgeHostExamplesHelp = `
    Examples:
      tt pr --forge gitlab <branch> [feature]
      TT_FORGE=gitlab tt pr <branch> [feature]
    `
    ```
*   **Logic**:
    *   文言は R3 を充足する内容をそのまま埋め込む（「仕様どおり」と書かず、優先順位・値・env を列挙）。
    *   `list` / `doctor` の Examples はコマンド名を合わせる（`tt list --forge gitlab --update`、`tt doctor --forge gitlab`）。共有定数に Examples を含めず、各コマンドで個別に足してもよい。

#### [MODIFY] [pr.go](file://features/tt/cmd/pr.go)

*   **Description**: `--forge` フラグ追加、Help 更新、`ActionRunner.ForgeOverride` へ渡す
*   **Technical Design**:
    ```go
    var flagPRForge string

    var prCmd = &cobra.Command{
        Use:   "pr <branch> [feature]",
        Short: "Create a Pull/Merge Request on the remote forge",
        Long: `Create a Pull Request (GitHub: gh pr create) or Merge Request (GitLab: glab mr create)
    for the worktree of the given branch.` + forgeHostResolveHelp + `
    Examples:
      tt pr --forge gitlab <branch> [feature]
      TT_FORGE=gitlab tt pr <branch> [feature]
    `,
        Args: cobra.RangeArgs(1, 2),
        RunE: runPR,
    }

    func init() {
        prCmd.Flags().StringVar(&flagPRForge, "forge", "",
            "Forge host override: github|gitlab (takes precedence over TT_FORGE; process-local only)")
    }

    func runPR(cmd *cobra.Command, args []string) error {
        ctx, err := InitContext(args)
        // ...
        ctx.ActionRunner.ForgeOverride = flagPRForge
        if err := ctx.ActionRunner.PR(ctx.RepoRoot, worktreePath); err != nil {
            // ...
        }
        // ...
    }
    ```
*   **Logic**:
    *   フラグ未指定（空文字）はオーバーライドなし。
    *   値の正規化・検証は forge 層に委譲（cmd 層で独自パースしない）。
    *   設定ファイルや env への書き戻しはしない。

#### [MODIFY] [list.go](file://features/tt/cmd/list.go)

*   **Description**: `--forge` フラグ追加、Help 更新、`--update` 時のクライアント生成に渡す
*   **Technical Design**:
    ```go
    var flagListForge string

    // listCmd.Long を更新し forgeHostResolveHelp を含める。
    // Examples に:
    //   tt list --forge gitlab --update
    //   TT_FORGE=gitlab tt list --update

    func init() {
        // existing flags...
        listCmd.Flags().StringVar(&flagListForge, "forge", "",
            "Forge host override: github|gitlab (takes precedence over TT_FORGE; used with --update; process-local only)")
    }

    // in runListBranches, when flagListUpdate:
    client, host, err := forge.NewClientForRepoOpts(forge.ClientForRepoOpts{
        RepoRoot:    repoRoot,
        Runner:      runner,
        CLIOverride: flagListForge,
    })
    ```
*   **Logic**:
    *   `--update` 無しでも `--forge bogus` を指定した場合は、解決を先に走らせて Fail Fast する（黙殺しない）。実装方針: `flagListForge != ""` のとき、`--update` の有無にかかわらず `ResolveHostPriority(flagListForge, os.Getenv("TT_FORGE"), "")` 相当で検証し、エラーなら return。`--update` 時は `NewClientForRepoOpts` に渡す。
    *   バックグラウンド `_update-code-status` 経路は CLI フラグを持たないため変更不要（env/URL のみ）。

#### [MODIFY] [doctor.go](file://features/tt/cmd/doctor.go)

*   **Description**: `--forge` フラグ追加、Help 更新、`doctor.Options.ForgeOverride` へ渡す
*   **Technical Design**:
    ```go
    var doctorFlagForge string

    // doctorCmd.Long を更新:
    // Diagnose repository structure, config, and external tools (gh or glab by forge host).
    // + forgeHostResolveHelp
    // Examples:
    //   tt doctor --forge gitlab
    //   TT_FORGE=gitlab tt doctor --json

    func init() {
        // existing flags...
        doctorCmd.Flags().StringVar(&doctorFlagForge, "forge", "",
            "Forge host override: github|gitlab (takes precedence over TT_FORGE; process-local only)")
    }

    func runDoctor(...) error {
        opts := doctor.Options{
            RepoRoot:      repoRoot,
            FeatureFilter: doctorFlagFeature,
            ToolChecker:   &doctor.DefaultToolChecker{},
            Fix:           doctorFlagFix,
            ForgeOverride: doctorFlagForge,
        }
        // ...
    }
    ```
*   **Logic**: `doctor.Run` が CLI 不正で error を返したら、そのままコマンド失敗（exit non-zero）。

### 統合テスト（tt CLI）

#### [MODIFY] [tt_forge_host_test.go](file://tests/tt/tt_forge_host_test.go)

*   **Description**: `--forge` の発見性（Help）と解決優先順位・Fail Fast の統合テスト（TDD 先書き）
*   **Technical Design**:
    ```go
    func TestTtPr_Help_MentionsForgeFlagAndPriority(t *testing.T)
    func TestTtList_Help_MentionsForgeFlagAndPriority(t *testing.T)
    func TestTtDoctor_Help_MentionsForgeFlagAndPriority(t *testing.T)
    // 共通アサーション: 出力に "--forge", "TT_FORGE", "github", "gitlab",
    // 優先順位を示す文言（"--forge" が TT_FORGE / origin より先、または "1." / "highest" 等）が含まれる。
    // TT_CMD_GH / TT_CMD_GLAB のいずれか（Help に記載した関連 env）が含まれる。

    func TestTtDoctor_ForgeFlag_OverridesGitHubURL(t *testing.T) {
        // origin=https://github.com/axsh/tokotachi.git
        // tt doctor --forge gitlab --json -> External Tools に glab、gh は含まない
    }

    func TestTtDoctor_ForgeFlag_BeatsTTForge(t *testing.T) {
        // origin=https://gitlab.com/g/p.git, env TT_FORGE=gitlab
        // tt doctor --forge github --json -> External Tools に gh、glab は含まない
    }

    func TestTtDoctor_NoForgeFlag_TTForgeStillWorks(t *testing.T) {
        // 既存 TestTtDoctor_TTForgeGitlab_OverridesNonGitlabURL の回帰を明示維持
        // （本ファイル既存ケースで充足。新規は不要でもよいが、フラグ未指定+TT_FORGE の確認を残す）
    }

    func TestTtDoctor_ForgeFlag_Invalid_FailsFast(t *testing.T) {
        // tt doctor --forge bogus --json -> exitCode != 0、stderr/stdout に forge/invalid 系
    }

    func TestTtDoctor_ForgeFlag_OverridesLocalhostURL(t *testing.T) {
        // origin=http://localhost:8929/group/proj.git （hostname に gitlab 無し）
        // tt doctor --forge gitlab --json -> glab を含む
    }

    func TestTtPr_ForgeFlag_DryRun_SelectsGitLabHost(t *testing.T) {
        // 一時 repo + worktree 相当の最小構成、または既存 helpers で branch/worktree を用意
        // TT_FORGE=github かつ origin=github URL
        // tt pr --forge gitlab --dry-run <branch>
        // stderr/stdout に "Detected forge host: gitlab" または glab 経路の痕跡をアサート
        // （worktree が無い環境では doctor ケースを主証拠とし、本ケースは worktree 準備可能な範囲で実装）
    }
    ```
*   **Logic**:
    *   Help 検証は `runTT(t, "pr", "--help")` 等（cwd 不問）。
    *   経路検証は既存 `setupForgeDoctorRepo` / `runTTInDirEnv` / `doctorToolNames` を再利用。
    *   ライブ GitLab CE（002）は必須化しない。`localhost:8929` は URL 文字列のみの一時 remote で足りる（実サーバ不要）。

### ドキュメント

（Documentation セクション参照）

## Step-by-Step Implementation Guide

1.  [x] **Unit tests first (forge resolve priority)**:
    *   Edit `internal/forge/host_test.go` to add `TestResolveHostPriority` and `TestDetectHostFromRepoOpts_*` as specified above.
    *   Edit `internal/forge/resolve_test.go` to add `TestNewClientForRepoOpts_*`.
    *   Run `./scripts/process/build.sh --skip-frontend --skip-etc` and confirm the new tests **fail** (API 未実装）。

2.  [x] **Implement forge resolve APIs**:
    *   Edit `internal/forge/host.go` to add `DetectOpts`, `ResolveHostPriority`, `DetectHostFromRepoOpts`; wrap existing `ResolveHost` / `DetectHostFromRepo`.
    *   Edit `internal/forge/resolve.go` to add `ClientForRepoOpts` / `NewClientForRepoOpts`; wrap `NewClientForRepo`.
    *   Re-run unit build; forge unit tests pass.

3.  [x] **Unit tests first (doctor Fail Fast)**:
    *   Add `TestRun_InvalidForgeOverride_ReturnsError` (and optionally `TestRun_ForgeOverride_SelectsGitLabTools`) in `pkg/doctor/`.
    *   Confirm fail-then-pass with step 4.

4.  [x] **Propagate override through doctor / action**:
    *   Edit `pkg/doctor/doctor.go` `Options` + `Run` as specified（CLI 不正 Fail Fast / env 不正 soft）。
    *   Edit `pkg/action/runner.go` to add `ForgeOverride string`.
    *   Edit `pkg/action/pr.go` to use `NewClientForRepoOpts` with `r.ForgeOverride`.

5.  [x] **CLI flags + Help (TDD via integration tests)**:
    *   Edit `tests/tt/tt_forge_host_test.go` to add Help / `--forge` 統合テスト（先に失敗させる）。
    *   Edit `features/tt/cmd/common.go` to add `forgeHostResolveHelp`（および必要なら Examples 定数）。
    *   Edit `features/tt/cmd/pr.go`, `list.go`, `doctor.go` to add `--forge`, Long help, and wire overrides.
    *   Ensure `list` validates invalid `--forge` even without `--update`.

6.  [x] **Documentation**:
    *   Update `README.md`, `docs/manual/tt-user-manual.md`, and `prompts/phases/000-foundation/branches/feat-adjust-gitlab/ideas/001-GitLab-MR-Support.md` priority lists as in Documentation.

7.  [/] **Verification Plan**:
    *   Execute the Verification Plan below (build → selective integration → comprehensive verdict).

## Verification Plan

### Test Item Design (Bottom-Up, testing-rules §11)

依存関係: Help/CLI フラグ → doctor/action 伝播 → `NewClientForRepoOpts` → `DetectHostFromRepoOpts` → `ResolveHostPriority` → `ParseForgeOverride` / `DetectHost`

| Order | Layer | Test items | Evidence |
| :---: | :--- | :--- | :--- |
| 1 | C: `ResolveHostPriority` / `ParseForgeOverride` | CLI>env>URL テーブル、不正 CLI/env、case/trim | 単体 assert Host / error 文言 |
| 2 | C: `DetectHostFromRepoOpts` | CLI が env+URL に勝つ、空 CLI で env、remote 無し+CLI | 一時 git repo + `t.Setenv` |
| 3 | B: `NewClientForRepoOpts` | CLI で GitLab client 選択、不正で error | client.Host() |
| 4 | B: `doctor.Run` | CLI 不正 Fail Fast、CLI gitlab で glab ツール列 | error / CheckResult names |
| 5 | A: `tt doctor/pr/list --help` | `--forge`・優先順位・値・関連 env | 統合 stdout |
| 6 | A: `tt doctor --forge ...` | URL 上書き、TT_FORGE より優先、bogus 失敗、localhost | doctor JSON External Tools |
| 7 | A: `tt pr --forge`（可能な範囲） | dry-run で host=gitlab | ログ / 終了コード |

#### §11.3 観点チェックリスト

| # | 観点 | 本計画での充足 |
|---|------|----------------|
| 1 | 正常系 | CLI / env / URL 各経路で期待 Host |
| 2 | 異常系・境界 | `--forge bogus` Fail Fast；空フラグ；大小文字 |
| 3 | 外部連携 | 実 `tt` バイナリ + 一時 git remote（統合） |
| 4 | データ一貫性 | N/A（永続設定なし；プロセスローカルのみ） |
| 5 | 状態遷移 | N/A（状態ファイル変更なし） |
| 6 | 設定反映 | `--forge` が `TT_FORGE` より優先されること |
| 7 | 副作用 | 永続設定・env を書き換えない（フラグは変数のみ） |

#### §11.4 セルフレビュー結果

1. **網羅性**: Help 発見性（R3）と解決優先（R1/R2）と互換（R4）を単体＋統合でカバー。言える。
2. **証拠の十分性**: doctor JSON のツール名で実際の経路を観測。Help は文字列アサート。不正は exit code。
3. **迂回排除**: `--forge github` + `TT_FORGE=gitlab` で gh のみを要求し、env 経路の偽成功を排除。
4. **依存関係**: forge 単体 → doctor/action → tt 統合の順。呼び出し元成功だけで末端未検証にしない。

### Automated Verification

1.  **Build & Unit Tests**:
    run the build script.
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    Run integration tests.
    ```bash
    ./scripts/process/integration_test.sh --categories tt --specify "Forge|Doctor_Help|Pr_Help|List_Help|TTForge"
    ```
    *   **Log Verification**:
        *   Help ケースが `--forge` / `TT_FORGE` / `github` / `gitlab` を含むこと。
        *   `TestTtDoctor_ForgeFlag_BeatsTTForge` が gh 経路であること。
        *   `TestTtDoctor_ForgeFlag_Invalid_FailsFast` が非ゼロ終了であること。
        *   既存 `TestTtDoctor_TTForgeGitlab_OverridesNonGitlabURL` が引き続き成功すること。

3.  **E2E Tests (CLI integration as end-to-end)**:
    本変更は CLI ホスト解決と Help のみで、agentservice / VSCode GUI 経路は無い。したがって `tests/agentservice_e2e_test.go` 系の E2E は不要。エンドツーエンド相当は `tests/tt/tt_forge_host_test.go` の実バイナリ実行で担保する。

    #### [MODIFY] [tt_forge_host_test.go](file://tests/tt/tt_forge_host_test.go)
    *   **テストケース**: 上記 Proposed Changes の Help / ForgeFlag / Invalid / Localhost 関数群
    *   **検証ポイント**: Help から実行契約が読めること；`--forge` が解決と診断ツール選択に反映されること；不正値で失敗すること

4.  **Optional live GitLab**（必須ではない・002 必須化しない）:
    ローカル CE がある場合のみ、任意で `TT_LIVE_GITLAB=1` の既存 smoke に `--forge gitlab` を添えた手動確認を行ってよい。本計画の合否判定には含めない。

5.  **総合判定プロセス (testing-rules §12)**:
    全自動テスト成功後、実装エージェントは次を確認しウォークスルーに総合判定を記載する。

    | # | チェック項目 | 確認内容 |
    |---|------------|----------|
    | 1 | スキップ | `t.Skip` / 条件回避が無いこと |
    | 2 | 部分エラー | 成功ログに ERROR/panic が混入していないこと |
    | 3 | 迂回 | `--forge` 指定時に `TT_FORGE` や URL 検出へ落ちていないこと（BeatsTTForge / Overrides*） |
    | 4 | アダプタ誤適用 | doctor JSON のツール名が期待ホストと一致 |
    | 5 | 順序依存 | `--specify` で単独実行しても同結果 |
    | 6 | カバレッジ | Help 3 コマンド + 優先順位 + 不正 + localhost + 既存 TT_FORGE 回帰 |
    | 7 | 外部状態 | 一時 git repo のみ；ライブ CE 非依存 |

    判定フォーマットは testing-rules §12.3 に従う。

## Documentation

`prompts/specifications` 配下に forge/CLI 専用仕様は現状無い。ユーザー向け文書と、矛盾を残す 001 アイデア記述を更新する。

#### [MODIFY] [README.md](file://README.md)
*   **更新内容**:
    *   `tt pr` 行に `--forge gitlab` 例を追加（例: `tt pr --forge gitlab <branch> [feature]`）。
    *   `tt doctor` / `tt list` に `--forge` が使える旨を短く追記。
    *   ホスト解決優先順位を `--forge` > `TT_FORGE` > origin と明記（`TT_FORGE` のみ最優先と読める文言を残さない）。

#### [MODIFY] [tt-user-manual.md](file://docs/manual/tt-user-manual.md)
*   **更新内容**:
    *   § `pr`: 「`TT_FORGE` が最優先」を「`--forge` > `TT_FORGE` > origin URL」に置換。
    *   `--forge github|gitlab` の説明と実行例（`tt pr --forge gitlab ...` / `TT_FORGE=gitlab tt pr ...`）を追加。
    *   `list` / `doctor` 節に `--forge`（list は `--update` と併用）を追記。
    *   環境変数表の `TT_FORGE` 説明を「`--forge` 未指定時の明示指定」に更新。

#### [MODIFY] [001-GitLab-MR-Support.md](file://prompts/phases/000-foundation/branches/feat-adjust-gitlab/ideas/001-GitLab-MR-Support.md)
*   **更新内容** (R5: 矛盾解消):
    *   R4 の優先順位リスト先頭に CLI `--forge` を追加し、本文・mermaid・検証シナリオの「`TT_FORGE` 最優先」表現を本仕様の順序に合わせて更新。
    *   注記: 詳細要件の正本は `ideas/003-Forge-CLI-Flag-And-Help.md` とする旨を一文追加してよい。
