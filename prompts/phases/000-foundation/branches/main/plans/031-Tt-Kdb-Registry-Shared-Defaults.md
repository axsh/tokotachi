# 031-Tt-Kdb-Registry-Shared-Defaults

> **Source Specification**: prompts/phases/000-foundation/branches/main/ideas/028-Tt-Kdb-Registry-Shared-Defaults.md

## Goal Description

registry.yaml 欠落時も `tt kdb-track` / `tt kdb backlog` が成功し、組み込み `backlog` を常に登録集合へ含める。既定ルートを `shared/kdb/` に移し、`--kdb-root` / `KDB_ROOT` で変更可能にする。`list` に `source`（builtin|registry）を付け、`tt kdb-track skeleton` で雛形を出す。

## User Review Required

None. 任意要件 O1（`--stdout`）は実装しない。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1 欠落時 exit 0・`registry_missing` | `registry.LoadOptional` + `cmd.loadKdbRegistryTo` + list JSON |
| R2 常時 backlog・yaml 優先 | `track.EffectiveEntries` / `ApplyEffective` |
| R3 `shared/kdb/` 既定 | `paths.KdbRoot` / `RegistryPath` / `BuiltinBacklogEntry` |
| R4 `--kdb-root` / `KDB_ROOT` / `--registry` | `paths` + `cmd` flags + `parseKdbArgs` |
| R5 `skeleton` / `--write` | `cmd/kdb_track.go` + `track/skeleton.go` |
| R6 `*_cmd` 引数付き | help/skeleton 例。delegate は現状 `bash -lc` のまま（回帰試験） |
| R7 unregistered は kdb-root のみ | `track.Unregistered(workspace, kdbRoot, f)` |
| R8 `source` | `track.ListedEntry` + list JSON |
| R9 テスト | unit + `tests/tt/tt_kdb_track_test.go` / backlog 試験更新 |
| O1 `--stdout` | **先送り**（実装しない） |

## Proposed Changes

### paths

#### [MODIFY] [features/tt/internal/kdb/paths/paths_test.go](features/tt/internal/kdb/paths/paths_test.go)

*   **Description**: TDD — KdbRoot / RegistryPath の新既定と優先順位。
*   **Logic**:
    *   `TestKdbRoot_Priority`: flag > `KDB_ROOT` > `shared/kdb`（slash 正規化、trailing slash 除去）
    *   `TestRegistryPath_DefaultShared`: env/flag 空で `{ws}/shared/kdb/registry.yaml`
    *   `TestRegistryPath_UsesKdbRoot`: kdb-root=`other/kdb` → `{ws}/other/kdb/registry.yaml`
    *   `TestBuiltinBacklogEntry`: module=`shared/kdb/backlog`、corpus=`shared/kdb/backlog/corpus`、全 `*_cmd` 非空（`tt kdb backlog <op>` 形式）
    *   既存 `TestRegistryPath_Priority` の既定期待値を `shared/kdb` に更新

#### [MODIFY] [features/tt/internal/kdb/paths/paths.go](features/tt/internal/kdb/paths/paths.go)

*   **Technical Design**:
    ```go
    const DefaultKdbRoot = "shared/kdb"

    // ResolveInput に KdbRoot string を追加（flag / KDB_ROOT）。
    func ResolveKdbRoot(flagKdbRoot string) string
    // Priority: flag > KDB_ROOT > DefaultKdbRoot. Clean slash, trim trailing separators.
    func RegistryPath(workspace, flagRegistry, kdbRoot string) string
    // Priority: flagRegistry > KDB_REGISTRY > Join(workspace, ResolveKdbRoot(kdbRoot), "registry.yaml")
    // When flagRegistry/KDB_REGISTRY set, return as-is (abs or relative string; callers Abs as needed).
    func BuiltinBacklogEntry(kdbRoot string) registry.Entry
    // ID=backlog, ModulePath=kdbRoot+"/backlog", CorpusPath=kdbRoot+"/backlog/corpus",
    // StatusCmd="tt kdb backlog status", DistillCmd=... distill, Dump/Merge/Pack 同様。
    ```
*   **Logic**:
    *   `ForEntry` は `RegistryFile: RegistryPath(ws, in.Registry, in.KdbRoot)` に更新。
    *   `InputFromEnv` に `KdbRoot: os.Getenv("KDB_ROOT")` を載せる（flag は呼び出し側）。

### registry

#### [MODIFY] [features/tt/internal/kdb/registry/registry_test.go](features/tt/internal/kdb/registry/registry_test.go)

*   **Logic**:
    *   `TestLoadOptional_Missing`: 存在しないパス → `f != nil`, `missing==true`, `err==nil`, `Version==1`, `len(KDBs)==0`
    *   `TestLoadOptional_Present`: 既存 Load 相当
    *   `TestLoadOptional_Corrupt`: 存在する壊れた YAML → `err != nil`

#### [MODIFY] [features/tt/internal/kdb/registry/registry.go](features/tt/internal/kdb/registry/registry.go)

*   **Technical Design**:
    ```go
    func LoadOptional(path string) (f *File, missing bool, err error)
    // os.IsNotExist → (&File{Version:1}, true, nil)
    // other read/parse errors → (nil, false, err)
    // success → (f, false, nil)  // Validate は呼ばない
    ```

### track

#### [MODIFY] [features/tt/internal/kdb/track/track_test.go](features/tt/internal/kdb/track/track_test.go)

*   **Logic**（新規／更新）:
    *   `TestEffectiveEntries_NoYAML`: 空 File → 1 件 backlog、`source=="builtin"`、paths が kdbRoot 由来
    *   `TestEffectiveEntries_YAMLOtherOnly`: designdb のみ → 2 件、designdb=`registry`、backlog=`builtin`
    *   `TestEffectiveEntries_YAMLOverridesBacklog`: yaml backlog の module_path が勝つ、`source=="registry"`、重複なし
    *   `TestUnregistered_KdbRootOnly`: `{kdbRoot}/orphan/corpus` を検出。`features/orphan/corpus` は検出しない
    *   `TestSkeleton_ContainsVersionAndArgExample`: 文字列に `version: 1`、`status_cmd:` と `--` 付き例、必須として `id: backlog` を強制しない
    *   `TestWriteSkeleton_CreateAndRefuseOverwrite`

#### [MODIFY] [features/tt/internal/kdb/track/track.go](features/tt/internal/kdb/track/track.go)

*   **Technical Design**:
    ```go
    const SourceBuiltin = "builtin"
    const SourceRegistry = "registry"

    type ListedEntry struct {
        registry.Entry
        Source string `json:"source"`
    }

    type ListResult struct {
        OK              bool          `json:"ok"`
        Registry        string        `json:"registry"`
        RegistryMissing bool          `json:"registry_missing,omitempty"`
        Message         string        `json:"message,omitempty"`
        Entries         []ListedEntry `json:"entries"`
    }

    func EffectiveEntries(f *registry.File, kdbRoot string) []ListedEntry
    // yaml の各 entry を SourceRegistry で列挙。
    // yaml に id=backlog が無ければ BuiltinBacklogEntry(kdbRoot) を先頭に SourceBuiltin で挿入。

    func ApplyEffective(f *registry.File, kdbRoot string) []ListedEntry
    // EffectiveEntries の結果で f.KDBs を Entry だけに書き戻し（ByID/StatusAll 用）、ListedEntry も返す。

    func List(f *registry.File, registryPath string, missing bool, kdbRoot string) ListResult
    // ApplyEffective → ListResult。missing 時 Message="registry file not found: "+registryPath

    func Unregistered(workspace, kdbRoot string, f *registry.File) (UnregisteredResult, error)
    // 走査: filepath.Join(workspace, kdbRoot) 配下の */corpus。rel は slash で kdbRoot+"/"+id+"/corpus"
    // features/*/corpus は走査しない。

    func StatusCmds(f *registry.File) ... // 呼び出し前に ApplyEffective 済みの f を前提（cmd 側で適用）
    ```
*   **Logic**:
    *   `Systematize` の unregistered 警告文言を `kdbRoot/id/corpus` 形式に更新。
    *   `Unregistered` 署名変更に合わせテスト・呼び出しを更新。

#### [NEW] [features/tt/internal/kdb/track/skeleton.go](features/tt/internal/kdb/track/skeleton.go)

*   **Technical Design**:
    ```go
    func SkeletonYAML() string // 固定雛形（英語コメント可）
    func WriteSkeleton(path string) error
    // 既存あれば error（os.ErrExist 相当のメッセージ）。親 MkdirAll。O_CREATE|O_EXCL。
    ```
*   **Logic — SkeletonYAML 必須内容**:
    ```yaml
    version: 1
    kdbs:
      # Example external KDB (commands may include arguments/flags):
      # - id: designdb
      #   module_path: shared/kdb/designdb
      #   corpus_path: shared/kdb/designdb/corpus
      #   status_cmd: "./scripts/design/status.sh --verbose"
      #   distill_cmd: "./scripts/design/distill.sh"
      #   dump_cmd: "./scripts/design/dump.sh"
      #   merge_cmd: "./scripts/design/merge.sh"
      #   pack_cmd: "./scripts/design/pack.sh"
      #
      # Optional override for the builtin backlog (uncomment to replace defaults):
      # - id: backlog
      #   module_path: shared/kdb/backlog
      #   corpus_path: shared/kdb/backlog/corpus
      #   status_cmd: "tt kdb backlog status"
      #   ...
    ```
    *   既定では `kdbs:` の下に有効なエントリは 0 件（コメントのみ）でよい。`version: 1` は必須。

### cmd

#### [MODIFY] [features/tt/cmd/kdb_track.go](features/tt/cmd/kdb_track.go)

*   **Technical Design**:
    *   `flagKdbRoot string` 追加。PersistentFlags `--kdb-root`。
    *   `--registry` の help 文言を `{kdb-root}/registry.yaml` に更新。
    *   `loadKdbRegistryTo(w, hint, workspaceFlag, registryFlag, kdbRootFlag)`:
        1. ResolveWorkspace
        2. kdbRoot := paths.ResolveKdbRoot(kdbRootFlag)
        3. regPath := paths.RegistryPath(ws, registryFlag, kdbRoot)
        4. `LoadOptional(regPath)`
        5. missing → Validate スキップ（空 File Version 1）
        6. !missing → Validate；失敗は従来どおり Fail
        7. `listed := track.ApplyEffective(f, kdbRoot)`
        8. 戻り値に `(workspace, f, regPath, missing, kdbRoot, err)` — または小さな struct `kdbLoad`
    *   `runKdbTrackList`: `track.List(...)` の全フィールドを JSON へ（`registry_missing` は true のときのみ omitempty）
    *   `runKdbTrackUnregistered`: `track.Unregistered(ws, kdbRoot, f)`
    *   `skeleton` サブコマンド: registry ロード不要。`--write` フラグ。
        *   無し: stdout に `SkeletonYAML()`（YAML 本文。JSON ではない）
        *   有り: `WriteSkeleton(regPath)`；成功時は JSON `{"ok":true,"path":"..."}` または YAML と同様にメッセージ JSON。仕様は stdout YAML が既定。`--write` 成功は JSON OK で path を返す（エージェント向け）。
    *   unregistered の Short を kdb-root 走査に合わせて更新。

#### [MODIFY] [features/tt/cmd/kdb.go](features/tt/cmd/kdb.go)

*   **Logic**: `parseKdbArgs` に `--kdb-root` / `--kdb-root=` を追加。`loadKdbRegistryTo` に渡す。非 builtin の ByID は ApplyEffective 後の File を使う。

#### [MODIFY] [features/tt/cmd/kdb_backlog.go](features/tt/cmd/kdb_backlog.go)

*   **Logic**: `loadKdbRegistryTo` に kdbRoot を渡し、欠落時も `ByID("backlog")` 成功。`paths.InputFromEnv` に KdbRoot を載せる。

#### [MODIFY] [features/tt/cmd/kdb_help.go](features/tt/cmd/kdb_help.go)

*   **Logic**: 既定パスを `shared/kdb/registry.yaml` に。`*_cmd` は引数を含めてよいと明記。`list` の成功例に `source` / `registry_missing`。`unregistered` は `{kdb-root}/*/corpus`。`skeleton` ページ追加。Preconditions から「必須の registry ファイル」を外し、欠落時の挙動を書く。

### capability / docs

#### [MODIFY] [prompts/manifest/code_content/capabilities/knowledge-database.md](prompts/manifest/code_content/capabilities/knowledge-database.md)

*   **更新内容**: 既定置き場・registry を `shared/kdb/` に。欠落時・builtin backlog・`tt kdb-track skeleton`・`--kdb-root` / `KDB_ROOT` を短く追記。  
    ※ `tt prompt update` はユーザー明示時のみ（本計画の Verification では raw ソース更新まで。エージェント配布は任意）。

#### [MODIFY] [.cursor/skills/knowledge-database/SKILL.md](.cursor/skills/knowledge-database/SKILL.md)（manifest 由来の場合は compile 後）

*   本計画では **manifest のみ必須**。スキル配布の更新はユーザーが `tt prompt update` したとき。

## Step-by-Step Implementation Guide

1. **Red: paths tests** — `paths_test.go` を新既定に合わせて失敗させる → **Green: paths.go**
2. **Red: registry LoadOptional tests** → **Green: LoadOptional**
3. **Red: track EffectiveEntries / Unregistered / skeleton tests** → **Green: track.go + skeleton.go**
4. **Red: tests/tt** — `newKdbWorkspace` を `shared/kdb/registry.yaml` へ。追加ケース:
   - `TestKdbTrack_List_MissingRegistry`
   - `TestKdbTrack_List_SourceFields`
   - `TestKdbTrack_Skeleton_StdoutAndWrite`
   - `TestKdbTrack_KdbRootOverride`
   - `TestKdbTrack_Unregistered_SharedRoot`
   - `TestKdbBacklog_WorksWithoutRegistry`
   - `TestKdbTrack_StatusCmds_KeepsArgs`（`echo hello --flag`）
5. **Green: cmd** — loadKdbRegistry / flags / skeleton / help / parseKdbArgs / backlog
6. **Docs**: capability manifest 更新
7. **Verification Plan** 実行 → 計画末尾に総合判定を追記
8. **record-far-knowledge** → **git push**

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```

2. **Integration Tests** (tt KDB 回帰):
   ```bash
   ./scripts/process/integration_test.sh --categories "tt" --specify "Kdb"
   ```
   *   **Log Verification**: Missing registry exit 0 + `registry_missing` + backlog `source=builtin`；yaml 上書きで `source=registry`；skeleton stdout/`--write`；kdb-root 上書き；unregistered が shared 配下；backlog CRUD/status が registry 無しで成功；引数付き status_cmd が status-cmds に残る。

3. **E2E Tests**:
   *   GUI E2E 不要（CLI のみ）。受け入れは `tests/tt/tt_kdb_*.go` が担う。

### 先に書く試験（TDD レッド記録用）

| 試験名 | 狙い |
| :--- | :--- |
| `TestKdbRoot_Priority` / `TestRegistryPath_DefaultShared` | R3/R4 パス |
| `TestLoadOptional_Missing` | R1 |
| `TestEffectiveEntries_*` | R2/R8 |
| `TestUnregistered_KdbRootOnly` | R7 |
| `TestSkeleton_*` / `TestWriteSkeleton_*` | R5 |
| `TestKdbTrack_List_MissingRegistry` | R1 CLI |
| `TestKdbTrack_List_SourceFields` | R8 CLI |
| `TestKdbTrack_Skeleton_StdoutAndWrite` | R5 CLI |
| `TestKdbBacklog_WorksWithoutRegistry` | R2 CLI |
| `TestKdbTrack_StatusCmds_KeepsArgs` | R6 |

### テスト項目設計（§11）

| 順序 | 対象 | 観点 |
| :--- | :--- | :--- |
| 1 | paths / LoadOptional | 正常・欠落・優先順位（#1,#2,#6） |
| 2 | EffectiveEntries / Unregistered / skeleton | 結合・FS（#1,#3,#4,#5） |
| 3 | `tt kdb-track` / `tt kdb backlog` CLI | 欠落成功・source・skeleton・kdb-root（#1,#3,#6,#7） |

**セルフレビュー (§11.4)**:
1. **網羅性**: R1–R8 を unit→CLI で覆えば「欠落しても backlog が見え、shared 既定で動く」と言い切れる。
2. **証拠**: JSON の `ok`/`registry_missing`/`source`/path、ファイル存在、exit code。
3. **迂回排除**: builtin 経路は registry 無し workspace；委譲は引数付き status_cmd の文字列一致。
4. **依存**: paths → registry → track → cmd → tests/tt。

### 総合判定プロセス (§12)

全テスト成功後、本計画末尾に総合判定結果を追記する（実装完了時）。

### 総合判定結果

（実装完了時に記入）

## Documentation

#### [MODIFY] [prompts/manifest/code_content/capabilities/knowledge-database.md](prompts/manifest/code_content/capabilities/knowledge-database.md)

*   **更新内容**: 上記 Proposed Changes のとおり。

#### [MODIFY] [prompts/phases/000-foundation/branches/main/ideas/028-Tt-Kdb-Registry-Shared-Defaults.md](prompts/phases/000-foundation/branches/main/ideas/028-Tt-Kdb-Registry-Shared-Defaults.md)

*   **更新内容**: `status: approved`（計画着手時）。
