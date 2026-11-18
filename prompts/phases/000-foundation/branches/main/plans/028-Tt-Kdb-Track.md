# 028-Tt-Kdb-Track

> **Source Specification**: prompts/phases/000-foundation/branches/main/ideas/026-Tt-Kdb-Std-And-Agent-Help.md  
> **Series**: Part 2 of 3（Part 1: `027-Tt-Agent-Oriented-Help.md`、Part 3: `029-Tt-Kdb-Backlog.md`）  
> **Depends on**: Part 1 の `clihelp` 基盤（help 登録に使用）

## Goal Description

`tt kdb-track`（registry 追跡・巡回）と `tt kdb <id> …` のディスパッチ骨格を実装する。`backlog` は Part 3 の builtin に接続するフックを用意し、非 builtin（designdb 等）は registry の `*_cmd` を委譲する。横断 I/O は JSON + エージェント向け help。

## User Review Required

1. **status-all / systematize の部分失敗方針（仕様 R5 の具体化）**: 全エントリを最後まで処理する。各エントリ結果を JSON 配列に含める。1件でも失敗したらプロセス exit code は `1`。失敗を黙殺しない。
2. **corpus git dirty**: `git -C <repo> status --porcelain -- <corpus_path>` が非空なら dirty（現行 `systematize.sh` と同じ）。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point |
| :--- | :--- |
| R1 非ゴール（designdb builtin 化しない） | `delegate` のみ。builtin は `backlog` のみ |
| R2 コマンド木 `kdb-track` + `kdb <id>` | `cmd/kdb_track.go`, `cmd/kdb.go` |
| R3 registry 読取 | `internal/kdb/registry` |
| R4 list/unregistered/status-cmds/status-all/systematize | `internal/kdb/track` + cmd |
| R5 builtin vs 委譲 | `internal/kdb/builtin`, `internal/kdb/delegate` |
| R8 パス（registry 正、workspace） | `internal/kdb/paths` |
| R9 JSON / help フィールド形式 | `internal/kdb/jsonout` |
| R10–R13 help | `clihelp.Register` for kdb-track / kdb paths |
| R14-4, R14-5 track 受け入れ | `tests/tt/tt_kdb_track_test.go` |
| R16 移行境界 | ドキュメント注記のみ（実装しない） |
| O1 並列 / O3 git 探索 | **先送り** |

## Proposed Changes

### データ構造（共通）

#### [NEW] [features/tt/internal/kdb/registry/registry.go](file://features/tt/internal/kdb/registry/registry.go)

*   **Description**: registry.yaml モデルとロード。
*   **Technical Design**:
    ```go
    type File struct {
        Version int     `yaml:"version"`
        KDBs    []Entry `yaml:"kdbs"`
    }

    type Entry struct {
        ID         string `yaml:"id"`
        ModulePath string `yaml:"module_path"`
        CorpusPath string `yaml:"corpus_path"`
        StatusCmd  string `yaml:"status_cmd"`
        DistillCmd string `yaml:"distill_cmd"`
        DumpCmd    string `yaml:"dump_cmd"`
        MergeCmd   string `yaml:"merge_cmd"`
        PackCmd    string `yaml:"pack_cmd"`
    }

    func Load(path string) (*File, error)
    func (f *File) Validate() error // version>=1; each entry all fields non-empty; unique ids
    func (f *File) ByID(id string) (Entry, bool)
    ```
*   **Logic**:
    *   `gopkg.in/yaml.v3` でロード。未知キー無視。
    *   不完全エントリは `Validate` で error（現行 awk の incomplete entry 相当）。

#### [NEW] [features/tt/internal/kdb/registry/registry_test.go](file://features/tt/internal/kdb/registry/registry_test.go)

*   **Logic**: 正常2件、version 欠落、必須フィールド欠落、重複 id。

#### [NEW] [features/tt/internal/kdb/paths/paths.go](file://features/tt/internal/kdb/paths/paths.go)

*   **Description**: リポジトリルートと KDB パス解決。
*   **Technical Design**:
    ```go
    type ResolveInput struct {
        Workspace string // flag / TT_WORKSPACE / BL_ROOT / cwd
        Registry  string // flag / KDB_REGISTRY / {workspace}/features/kdb/registry.yaml
        // Backlog overrides (BL_*):
        DataDir, DBPath, CorpusDir, TmpDir string
        Today string // BL_TODAY YYYY-MM-DD; empty → UTC today
    }

    type Paths struct {
        Workspace    string
        RegistryFile string
        Entry        registry.Entry // optional zero if not bound
        ModuleDir    string // abs: workspace + module_path
        CorpusDir    string // abs
        DataDir      string // abs: prefer override; else Dir(CorpusDir)+"/data" OR ModuleDir+"/data"
        DBPath       string // abs: DataDir+"/backlog.sqlite" for backlog; generic "{id}.sqlite" は backlog のみ固定名 backlog.sqlite
        TmpDir       string // abs: override or workspace+"/tmp/kdb/"+id
        DataJSON     string // DataDir+"/items.json"
        TmpJSON      string // TmpDir+"/items.json"
        Today        string
    }

    func ResolveWorkspace(flagWorkspace string) (string, error)
    func RegistryPath(workspace, flagRegistry string) string
    func ForEntry(in ResolveInput, e registry.Entry) (Paths, error)
    ```
*   **Logic**:
    *   Workspace 優先順位: flag > env `TT_WORKSPACE` > env `BL_ROOT` > `os.Getwd()`（O3 は先送り）。
    *   Registry: flag > `KDB_REGISTRY` > `{workspace}/features/kdb/registry.yaml`。
    *   CorpusDir: override `BL_CORPUS_DIR`（backlog 時）else `filepath.Join(workspace, entry.CorpusPath)`。
    *   DataDir: `BL_DATA_DIR` else `filepath.Join(filepath.Dir(CorpusDir), "data")`（corpus の親が module ルートである前提。`features/backlog/corpus` → `features/backlog/data`、`scripts/kdb/backlog/corpus` → `scripts/kdb/backlog/data`）。
    *   TmpDir: `BL_TMP_DIR` else `{workspace}/tmp/kdb/{id}`。
    *   DBPath: `BL_DB_PATH` else `{DataDir}/backlog.sqlite`（id=backlog）。他 id の DB は委譲先に任せ、Paths では空でもよい。

#### [NEW] [features/tt/internal/kdb/paths/paths_test.go](file://features/tt/internal/kdb/paths/paths_test.go)

*   **Logic**: features 配置と scripts/kdb/backlog 配置の両方で DataDir が対になること。環境変数上書き。

#### [NEW] [features/tt/internal/kdb/jsonout/jsonout.go](file://features/tt/internal/kdb/jsonout/jsonout.go)

*   **Description**: 成功/失敗 JSON 出力と終了コード。
*   **Technical Design**:
    ```go
    type FailCode int // 1 business, 2 usage, 3 merge conflict

    func OK(w io.Writer, fields map[string]any) error
    // writes {"ok":true, ...fields} compact JSON + newline; ensure_ascii 相当は Go json の UTF-8

    func Fail(w io.Writer, helpHint string, code int, msg string, extra map[string]any) error
    // {"ok":false,"error":msg,"help":helpHint,...}; caller os.Exit or returns *ExitError

    type ExitError struct { Code int; Err error }
    ```
*   **Logic**: helpHint 例 `"tt kdb-track list help"`。

#### [NEW] [features/tt/internal/kdb/builtin/builtin.go](file://features/tt/internal/kdb/builtin/builtin.go)

*   **Technical Design**:
    ```go
    const BacklogID = "backlog"
    func IsBuiltin(id string) bool { return id == BacklogID }

    // Runner is implemented by Part 3 backlog package.
    type StatusRunner interface {
        Status(ctx context.Context, p paths.Paths) (json.RawMessage, error)
    }
    type PipelineRunner interface {
        Distill(ctx context.Context, p paths.Paths) error
        Dump(ctx context.Context, p paths.Paths) error
        Merge(ctx context.Context, p paths.Paths, prefer string) error
        Pack(ctx context.Context, p paths.Paths) error
    }

    var status StatusRunner // set from backlog via Register in Part 3
    var pipe PipelineRunner

    func RegisterBacklog(s StatusRunner, p PipelineRunner) { status, pipe = s, p }
    ```
*   **Logic**: Part 2 のテストでは fake runner を Register して systematize を検証可能にする。

#### [NEW] [features/tt/internal/kdb/delegate/delegate.go](file://features/tt/internal/kdb/delegate/delegate.go)

*   **Description**: registry `*_cmd` をシェル実行。
*   **Technical Design**:
    ```go
    func Run(ctx context.Context, workspace string, cmdLine string) (stdout string, err error)
    // exec.CommandContext("bash", "-lc", cmdLine); Dir=workspace; capture stdout/stderr
    ```
*   **Logic**: 現行 `bash -lc "$status_cmd"` 相当。Windows は Git Bash の `bash` を PATH から探す（見つからなければ明確な error）。exit non-zero を error に。

#### [NEW] [features/tt/internal/kdb/delegate/delegate_test.go](file://features/tt/internal/kdb/delegate/delegate_test.go)

*   **Logic**: `bash -lc 'echo hi'` 成功。`false` で error。

#### [NEW] [features/tt/internal/kdb/track/track.go](file://features/tt/internal/kdb/track/track.go)

*   **Description**: 横断操作のコア。
*   **Technical Design**:
    ```go
    type ListResult struct {
        OK      bool           `json:"ok"`
        Entries []registry.Entry `json:"entries"`
    }

    type UnregisteredResult struct {
        OK    bool     `json:"ok"`
        IDs   []string `json:"ids"`
    }

    func List(f *registry.File) ListResult
    func Unregistered(workspace string, f *registry.File) (UnregisteredResult, error)
    // scan workspace/features/*/corpus dirs; id=basename(dir); rel=features/{id}/corpus;
    // if rel not in registry corpus_path set → unregistered

    func StatusCmds(f *registry.File) map[string]any // {"ok":true,"commands":[{id,status_cmd},...]}

    type EntryStatus struct {
        ID      string          `json:"id"`
        OK      bool            `json:"ok"`
        Dirty   bool            `json:"dirty,omitempty"`
        Output  json.RawMessage `json:"output,omitempty"`
        Error   string          `json:"error,omitempty"`
    }

    func StatusAll(ctx context.Context, workspace string, f *registry.File) (statuses []EntryStatus, anyFail bool)

    type SystematizeResult struct {
        OK            bool     `json:"ok"`
        AddCandidates []string `json:"add_candidates"`
        Entries       []EntrySystematize `json:"entries"`
    }

    func Systematize(ctx context.Context, workspace string, f *registry.File, onlyID string) (SystematizeResult, int /*exitCode*/)
    ```
*   **Logic（StatusAll）**:
    1. 各 Entry について `IsBuiltin` なら `builtin.Status`、否则 `delegate.Run(status_cmd)`。
    2. stdout を JSON としてパースし `"dirty": true` を検出（現行 grep 相当: raw 文字列に `"dirty"` と `true` の組み合わせでも可だが、まずは `json.Unmarshal` で `dirty` フィールドを読む。パース不能なら dirty=false とし output を文字列ラップ）。
    3. 失敗エントリは `OK=false`, `Error=...`。`anyFail` 集約。
*   **Logic（Systematize）**:
    1. `onlyID` 非空ならその id のみ。
    2. 未登録 corpus があれば stderr 相当の警告をエントリに `warning` として含めてよい（現行 WARN）。
    3. 各対象: status（上記）+ `CorpusGitDirty(workspace, corpus_path)`。
    4. dirty でなければ `skip`。
    5. dirty なら distill→dump→merge→pack（builtin or delegate cmds）。段の途中失敗で当該エントリ失敗、後続エントリは継続。
    6. 成功退避した `corpus_path`（registry 相対のまま）を `add_candidates` へ。
    7. 全体 `ok` は anyFail の否定。exit 1 if anyFail else 0。

#### [NEW] [features/tt/internal/kdb/track/gitdirty.go](file://features/tt/internal/kdb/track/gitdirty.go)

*   **Logic**: `exec.Command("git", "-C", workspace, "status", "--porcelain", "--", corpusPath)`。出力 trim 非空なら dirty。

#### [NEW] [features/tt/internal/kdb/track/track_test.go](file://features/tt/internal/kdb/track/track_test.go)

*   **Logic**:
    *   temp workspace + registry fixture（backlog + designdb）。
    *   List / Unregistered（余分な `features/extra/corpus`）。
    *   fake builtin Status dirty + Pipeline 呼び出し回数。
    *   designdb は `status_cmd: echo '{"ok":true,"dirty":false}'` 等。
    *   Systematize が add_candidates を返す。

### CLI

#### [NEW] [features/tt/cmd/kdb_track.go](file://features/tt/cmd/kdb_track.go)

*   **Description**: `tt kdb-track` サブコマンド群。
*   **Technical Design**:
    ```go
    var kdbTrackCmd = &cobra.Command{Use: "kdb-track", Short: "Track and circulate registered knowledge DBs"}
    // children: list, unregistered, status-cmds, status-all, systematize
    // persistent flags: --workspace, --registry
    // systematize: --id
    ```
*   **Logic**:
    *   位置引数 `help` / `--help` → clihelp（Part 1）。
    *   成功時 `jsonout.OK`。失敗時 `jsonout.Fail` + `SilenceErrors`/`SilenceUsage`。
    *   systematize 成功時: JSON を stdout に書いた後、移行期互換で各 candidate について追加で `ADD_CANDIDATE: <path>\n` を stdout に追記（仕様許可）。テストは JSON を先にパースできるよう、**JSON 行を最初の1行（または最初の `{...}`）に固定**し、ADD_CANDIDATE はその後。

#### [NEW] [features/tt/cmd/kdb.go](file://features/tt/cmd/kdb.go)

*   **Description**: `tt kdb` ディスパッチ。
*   **Technical Design**:
    ```go
    var kdbCmd = &cobra.Command{
        Use:                "kdb",
        DisableFlagParsing: true,
        RunE:               runKdb,
    }

    func runKdb(cmd *cobra.Command, args []string) error {
        // manual parse: extract help/-h/--help
        // args[0] = id; reject reserved: help already handled; completion
        // if id == "backlog": SetArgs(rest) on backlogRootCmd.Execute()  // Part 3 wires backlogRootCmd
        // else: load registry; ByID; if missing → Fail code 2
        // op := rest[0]; map status|distill|dump|merge|pack → delegate or builtin hook
        // unknown op → Fail code 2
    }
    ```
*   **Logic**:
    *   Part 2 時点では `backlog` は「Part 3 未接続」なら明確なエラー、または no-op stub。**実装順**: Part 3 と同時マージ前提なら、本 Part 完了定義を「delegate 経路 + track」とし、`backlog` サブツリーは Part 3 で `kdbCmd` に `DisableFlagParsing` をやめ静的 AddCommand する案も可。
    *   **採用（具体）**: Part 2 では `kdbCmd` に静的には何も足さず `DisableFlagParsing` + dispatch。Part 3 で `backlog` ops を `internal` から呼び、dispatch の `id=="backlog"` 分岐を本実装に差し替える。テストは designdb 委譲を主に検証。

#### [MODIFY] [features/tt/cmd/root.go](file://features/tt/cmd/root.go)

*   **Logic**: `rootCmd.AddCommand(kdbTrackCmd); rootCmd.AddCommand(kdbCmd)`。

#### Help 登録

*   `clihelp.Register` for: `kdb-track`, `kdb-track list|unregistered|status-cmds|status-all|systematize`, `kdb`（概要）。
*   本文は現行 `scripts/kdb/*.sh --help` を英語7見出しに整形し、出力が JSON であること・Python 不要を明記。

### 統合テスト

#### [NEW] [tests/tt/tt_kdb_track_test.go](file://tests/tt/tt_kdb_track_test.go)

*   **Logic**:
    1. temp repo: registry（backlog + designdb）、`features/backlog/corpus` 空、designdb の cmd は `printf '%s' '{"ok":true,"dirty":false}\n'`。
    2. `tt kdb-track list --workspace TEMP` → entries に両 id。
    3. 余分な `features/orphan/corpus` → `unregistered` に `orphan`。
    4. `tt kdb designdb status --workspace TEMP` → 委譲成功（スタブが呼ばれた証拠として固定 JSON）。
    5. `tt help kdb-track systematize` → 7 見出し、exit 0。
    6. PATH から `python`/`python3`/`sqlite3` を外しても list/help が成功（環境変数で最小 PATH）。

## Step-by-Step Implementation Guide

1. registry / paths / jsonout の単体テストを先に書き実装。
2. delegate + track（fake builtin）の単体テスト → 実装。
3. `kdb_track` CLI と help 登録。
4. `kdb` dispatch（designdb 委譲）と統合テスト。
5. Verification Plan 実行。
6. Part 3 へ（backlog Register）。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "KdbTrack|AgentHelp"
    ```

3. **E2E**: GUI 不要。CLI は `tests/tt/tt_kdb_track_test.go`。

### テスト項目セルフレビュー（§8.4）

| # | 結果 |
| :--- | :--- |
| 網羅性 | list/unregistered/委譲/systematize(fake)/help/PATH 縮小をカバー。backlog 実パイプラインは Part 3 |
| 証拠 | JSON フィールドと exit code を断言 |
| 迂回 | designdb が builtin に落ちないこと（IsBuiltin 否定 + スタブ出力）を断言 |
| 依存 | registry → track → CLI → tests/tt |

### 総合判定プロセス（§9）

Part 2 完了時に §9.3 フォーマットで記録。委譲がスキップされていないこと、部分失敗が exit 1 になるケースを1つ以上証拠として残す。

## Documentation

#### [MODIFY] [prompts/manifest/code_content/capabilities/knowledge-database.md](file://prompts/manifest/code_content/capabilities/knowledge-database.md)

*   **更新内容（Part 2+3 の終わりに一度で可）**: bash `./scripts/kdb/…` 記述を `tt kdb-track` / `tt kdb <id>` / `tt help …` に置換。置き場の例に `scripts/kdb/backlog` を追記。

## 継続計画について

*   **Part 3** (`029-Tt-Kdb-Backlog.md`): backlog CRUD + distill/dump/merge/pack/unpack/restore/status/migrate-uuids + 受け入れ round-trip。
