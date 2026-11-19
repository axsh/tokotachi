# 029-Tt-Kdb-Backlog

> **Source Specification**: prompts/phases/000-foundation/branches/main/ideas/026-Tt-Kdb-Std-And-Agent-Help.md  
> **Series**: Part 3 of 3（Part 1: `027-Tt-Agent-Oriented-Help.md`、Part 2: `028-Tt-Kdb-Track.md`）  
> **Depends on**: Part 1 `clihelp`、Part 2 `paths` / `jsonout` / `builtin.RegisterBacklog` / `kdb` dispatch

## Goal Description

`tt kdb backlog <op>` として、現行 `scripts/kdb-backlog` + `bl_json.py` の振る舞いを Go（埋め込み SQLite）で提供する。Python / 外部 `sqlite3` CLI を不要にし、Tokotachi フィクスチャ試験で受け入れ条件 R14 を満たす。

## User Review Required

1. merge 衝突時 exit code **3**（現行 `bl_json`）を維持。stdout には `ok:true` 相当を出さず、merge.sh 互換で「衝突時は restored=false の成功 JSON を exit 0 で返す」現行 bash ラッパ挙動があるが、**正本は bl_json の exit 3 + track/pipeline から見た失敗**とする。CLI `tt kdb backlog merge` は: 衝突時 exit 3、stderr または stdout に衝突サマリ JSON（実装は stdout に `ok:false` ではなく、現行 merge.sh に合わせ `ok:true` + `restored:false` + `conflict_count` かつ exit 0、という分岐がある）。**採用**: `merge.sh` のユーザー契約を優先 — 衝突時 exit 0、`data.restored=false`、`conflict_count>0`。`bl_json` 単体の exit 3 は内部関数が返し、CLI が吸収する。
2. `migrate-uuids` は `tt kdb backlog migrate-uuids` として提供するが、親 help の「主要 op」一覧では Optional 扱い。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point |
| :--- | :--- |
| R6 全 op | `internal/kdb/backlog/*` + `cmd` dispatch |
| R7 スキーマ・UUID・state/kind・パイプライン | `schema.sql` embed, `item.go`, `store.go`, `pipeline_*.go` |
| R8 パス | Part 2 `paths.ForEntry` + BL_* |
| R9 JSON / 終了コード / help hint | `jsonout` + 各 op |
| R10–R13 help | 各 op の `clihelp.Register("kdb backlog …")` |
| R14-1..3, R14-6 受け入れ | `tests/tt/tt_kdb_backlog_test.go` |
| R15 Tokotachi が正 | 同上（依頼元スモーク先送り） |
| R16 依頼元ラッパ | 実装しない |
| O4 catalog mirror | **先送り**（既存 `knowledge-database.md` 更新のみ） |

## Proposed Changes

### ドメインモデルとスキーマ

#### [NEW] [features/tt/internal/kdb/backlog/schema.sql](file://features/tt/internal/kdb/backlog/schema.sql)

*   **Description**: 現行 schema をそのまま埋め込み正本にする。
*   **Logic**（全文を計画に含める）:
    ```sql
    CREATE TABLE IF NOT EXISTS items (
      phase TEXT NOT NULL,
      branch TEXT NOT NULL,
      local_id TEXT NOT NULL,
      slug TEXT NOT NULL DEFAULT '',
      title TEXT NOT NULL DEFAULT '',
      state TEXT NOT NULL DEFAULT '未確認',
      state_date TEXT,
      kind TEXT NOT NULL DEFAULT '',
      ask_to TEXT NOT NULL DEFAULT '',
      due TEXT NOT NULL DEFAULT '',
      origin TEXT NOT NULL DEFAULT '',
      decision TEXT NOT NULL DEFAULT '',
      problem TEXT NOT NULL DEFAULT '',
      current_handling TEXT NOT NULL DEFAULT '',
      options TEXT NOT NULL DEFAULT '',
      questions TEXT NOT NULL DEFAULT '',
      after_decision TEXT NOT NULL DEFAULT '',
      history TEXT NOT NULL DEFAULT '',
      created_at TEXT NOT NULL DEFAULT '',
      updated_at TEXT NOT NULL DEFAULT '',
      uuid TEXT NOT NULL,
      PRIMARY KEY (phase, branch, local_id)
    );
    CREATE UNIQUE INDEX IF NOT EXISTS items_uuid ON items(uuid);
    ```

#### [NEW] [features/tt/internal/kdb/backlog/item.go](file://features/tt/internal/kdb/backlog/item.go)

*   **Technical Design**:
    ```go
    var Columns = []string{
        "phase","branch","local_id","slug","title","state","state_date","kind",
        "ask_to","due","origin","decision","problem","current_handling","options",
        "questions","after_decision","history","created_at","updated_at","uuid",
    }
    var States = []string{"未確認","確認中","決着","取り下げ"}
    var Kinds = []string{"仮の決定","確認事項","先送り"}
    const ChunkSize = 100
    const FormatVersion = 1

    type Item struct {
        Phase            string  `json:"phase"`
        Branch           string  `json:"branch"`
        LocalID          string  `json:"local_id"`
        Slug             string  `json:"slug"`
        Title            string  `json:"title"`
        State            string  `json:"state"`
        StateDate        *string `json:"state_date"` // null vs ""
        Kind             string  `json:"kind"`
        AskTo            string  `json:"ask_to"`
        Due              string  `json:"due"`
        Origin           string  `json:"origin"`
        Decision         string  `json:"decision"`
        Problem          string  `json:"problem"`
        CurrentHandling  string  `json:"current_handling"`
        Options          string  `json:"options"`
        Questions        string  `json:"questions"`
        AfterDecision    string  `json:"after_decision"`
        History          string  `json:"history"`
        CreatedAt        string  `json:"created_at"`
        UpdatedAt        string  `json:"updated_at"`
        UUID             string  `json:"uuid"`
    }

    func NormItem(m map[string]any) (Item, error) // bl_json.norm_item 相当
    func KeyOf(it Item) (phase, branch, localID string)
    func ValidState/ValidKind/CheckUUID/CheckLocalID/CheckName/CheckDate ...
    ```

#### [NEW] [features/tt/internal/kdb/backlog/item_test.go](file://features/tt/internal/kdb/backlog/item_test.go)

*   **Logic**: Norm、UUID 正規表現、state/kind 検証。

### JSON 配列パイプライン（bl_json 置換）

#### [NEW] [features/tt/internal/kdb/backlog/itemsio.go](file://features/tt/internal/kdb/backlog/itemsio.go)

*   **Logic**:
    *   `ReadItems(path) ([]Item, error)`: 欠落/空 → `[]`。UTF-8 BOM 除去。配列以外は error。
    *   `WriteItems(path, items)`: 親 dir 作成。canonical compact JSON array + newline。
    *   `CanonicalArrayBytes(items []Item) []byte`: `json.Marshal` で `ensure_ascii=False` 相当（UTF-8）、compact separators。
    *   並び: `SortItems` by phase, branch, local_id。

#### [NEW] [features/tt/internal/kdb/backlog/collapse.go](file://features/tt/internal/kdb/backlog/collapse.go)

*   **Logic**（`bl_json.collapse` / `find_problems` を再記述）:
    *   同一論理キーで **完全一致** duplicate → collapsed++、1つ残す。
    *   同一キーで内容違い → conflicts にキーを残し、両方 out に残す。
    *   `FindProblems`: empty key 成分、invalid state/kind、empty title、empty/invalid uuid、duplicate uuid。

#### [NEW] [features/tt/internal/kdb/backlog/corpus.go](file://features/tt/internal/kdb/backlog/corpus.go)

*   **Logic**（`build_chunks` / pack / unpack / content_hash）:
    *   `CHUNK_SIZE=100`。chunk path `items/chunk-%04d.json`。
    *   chunk sha: SHA-256 of bytes with CRLF→LF（`sha256_lf`）。
    *   `content_hash`: SHA-256 hex of **concatenation of chunk sha hex strings** as ASCII（`bl_json.content_hash`）。
    *   `Pack(dataJSON, corpusDir)`: 重複キーがあれば error。旧 chunk 削除。manifest:
        ```json
        {
          "kdb": "backlog",
          "format_version": 1,
          "content_hash": "...",
          "generated_at": "RFC3339",
          "chunk_size": 100,
          "chunks": [{"path":"items/chunk-0000.json","kind":"items","count":N,"sha256":"..."}]
        }
        ```
      `content_hash` 不変なら `generated_at` を旧 manifest から維持。
    *   `Unpack(corpusDir, outJSON)`: manifest `kdb=="backlog"`、各 chunk sha 検証、content_hash 一致、items を out に書く。

#### [NEW] [features/tt/internal/kdb/backlog/merge.go](file://features/tt/internal/kdb/backlog/merge.go)

*   **Logic**（`cmd_merge`）:
    *   `merged = data + tmp`（配列連結）。
    *   `--prefer tmp|data`: 両側にあるキーについて prefer 側を残す。
    *   `collapse` 後、conflicts 無ければ sort して write。conflicts あれば write したうえで内部 `ErrMergeConflict`。
    *   戻り: summary `{data_before,tmp_count,count,collapsed,conflict_count,conflicts,prefer}`。

#### [NEW] [features/tt/internal/kdb/backlog/listfilter.go](file://features/tt/internal/kdb/backlog/listfilter.go)

*   **Logic**（`list-filter`）: match regex（title/problem/slug、case-insensitive）、sort keys、order、limit、offset。`total` は limit 前件数。

#### [NEW] [features/tt/internal/kdb/backlog/pipeline_test.go](file://features/tt/internal/kdb/backlog/pipeline_test.go)

*   **Logic**: collapse / pack hash 安定 / unpack round-trip / merge prefer / merge conflict。**TDD で先に書く。**

### SQLite ストア

#### [NEW] [features/tt/internal/kdb/backlog/store.go](file://features/tt/internal/kdb/backlog/store.go)

*   **Technical Design**:
    ```go
    type Store struct { db *sql.DB; path string }

    func Open(dbPath string) (*Store, error) // modernc.org/sqlite; embed schema.sql; mkdir
    func (s *Store) Close() error
    func (s *Store) Insert(it Item) error
    func (s *Store) Update(it Item) error
    func (s *Store) GetByKey(phase,branch,id string) (Item, bool, error)
    func (s *Store) GetByUUID(uuid string) (Item, bool, error)
    func (s *Store) ListAll() ([]Item, error) // order phase,branch,local_id
    func (s *Store) NextLocalID(phase,branch string) (string, error) // %03d
    func (s *Store) ReplaceAll(items []Item) error // restore: delete all + insert
    func (s *Store) DumpJSON(path string) error
    ```
*   **Logic**: 外部 sqlite3 CLI は使わない。`github.com/google/uuid` で uuid v4（既に go.mod indirect → direct 化してよい）。

#### [NEW] [features/tt/internal/kdb/backlog/store_test.go](file://features/tt/internal/kdb/backlog/store_test.go)

*   **Logic**: temp db で insert/get/uuid/next id/replaceAll。

### CRUD / ops

#### [NEW] [features/tt/internal/kdb/backlog/ops_crud.go](file://features/tt/internal/kdb/backlog/ops_crud.go)

*   **Logic**:
    *   **Add**: 必須 phase/branch/title/kind。state 既定 `未確認`、state_date/created/updated=`Today`。uuid 新規。id 省略時 NextLocalID、衝突時1回リトライ。成功 `{"ok":true,"item":{...}}`。
    *   **Get**: `--uuid` または phase+branch+id。両方時は不一致なら exit 1。
    *   **Update**: 決着/取り下げへの state 変更は exit 2。`--history` と `--append-history` 同時は exit 2。title 空禁止。updated_at 更新。
    *   **Resolve**: state を `決着` または `取り下げ` に。decision 等は現行 resolve.sh フラグに合わせる（実装時に `resolve.sh` を読み、同名フラグをコピー）。
    *   **List**: DB → items → listfilter。既定で `決着`/`取り下げ` 除外（`--state` 明示時はそのまま、`--include-closed 1` で含む）。

#### [NEW] [features/tt/internal/kdb/backlog/ops_pipeline.go](file://features/tt/internal/kdb/backlog/ops_pipeline.go)

*   **Logic**:
    *   **Distill**: DB dump を一時 JSON 化して problems 検査 + data/items.json があれば rewrite 畳み込み。problems あれば exit 1 + reports。
    *   **Dump**: `Store.DumpJSON(tmp/items.json)`。
    *   **Merge**: merge tmp+data → 衝突無ければ Restore。CLI 契約は User Review #1。
    *   **Pack / Unpack / Restore / Status**: bl_json + 現行 sh と同じ JSON 形。
    *   **ImportMD / MigrateUUIDs**: `bl_json.py` の `cmd_import_md` / `cmd_migrate_uuids` を移植（パーサ互換を単体テスト1本以上）。

#### [NEW] [features/tt/internal/kdb/backlog/ops_test.go](file://features/tt/internal/kdb/backlog/ops_test.go)

*   **Logic**: CRUD 状態遷移、append-history、resolve 以外の決着拒否、round-trip distill→dump→merge→pack→unpack→restore。

#### [NEW] [features/tt/internal/kdb/backlog/register.go](file://features/tt/internal/kdb/backlog/register.go)

*   **Logic**: `builtin.RegisterBacklog(statusAdapter, pipelineAdapter)` を `init` または cmd から呼ぶ。

### CLI 接続

#### [MODIFY] [features/tt/cmd/kdb.go](file://features/tt/cmd/kdb.go)

*   **Logic**: `id=="backlog"` のとき op を `backlog.Ops` にディスパッチ。フラグは `pflag` で op ごとに定義（現行 sh の flag 名を踏襲: `--phase`, `--branch`, `--id`, `--uuid`, `--append-history`, `--prefer`, `--match`, …）。
*   失敗 JSON の `help`: `tt kdb backlog <op> help`。

#### Help

*   各 op を `clihelp.Register("kdb backlog add", Page{...})` 等。現行 `*.sh` help から Python/sqlite3 前提を削除し、埋め込み SQLite と書く。

### 統合テスト（受け入れの正）

#### [NEW] [tests/tt/tt_kdb_backlog_test.go](file://tests/tt/tt_kdb_backlog_test.go)

*   **テストケース**（依頼元 `kdb_backlog_cli_test.go` 相当を `tt` 呼び出しに置換）:
    1. `TestKdbBacklogHelpWithoutDB` — help exit 0、DB ファイル未作成。
    2. `TestKdbBacklogCRUD` — add/list/get/update/resolve。
    3. `TestKdbBacklogListFilters` — exclude closed、match、paginate、sort。
    4. `TestKdbBacklogGetByUUID` — uuid / mismatch。
    5. `TestKdbBacklogRoundTrip` — distill→dump→merge→pack、content_hash 再実行安定、unpack→restore で uuid 保持。
    6. `TestKdbBacklogMergeConflict` — restored=false / conflict_count。
    7. `TestKdbBacklogNoPythonNoSqliteCLI` — PATH から除外しても成功。
    8. `TestKdbTrackSystematizeBacklog` — dirty backlog を `tt kdb-track systematize --id backlog` で退避、`add_candidates` + `ADD_CANDIDATE:`。
    9. `TestKdbPathsScriptsLayout` — registry が `scripts/kdb/backlog/corpus` でも動く。

*   **フィクスチャ手順**:
    1. temp dir を workspace に。
    2. `features/kdb/registry.yaml` に backlog（+ 必要なら designdb スタブ）。
    3. env: `BL_TODAY=2026-11-18` 等。
    4. `ttBinary` + `--workspace`。

#### [MODIFY] [tests/tt/tt_kdb_track_test.go](file://tests/tt/tt_kdb_track_test.go)

*   **Logic**: Part 2 の fake に加え、実 backlog dirty systematize ケースを1つ（または本ファイルに集約）。

## Step-by-Step Implementation Guide

1. **TDD pipeline**: `collapse` / `corpus` / `merge` テストを先に書き、実装してグリーン。
2. **schema + store**: embed SQL、CRUD ストア単体。
3. **ops_crud + ops_pipeline**: 単体で R14-1..3 を仮再現。
4. **CLI dispatch + help 登録**。
5. **builtin.RegisterBacklog** で Part 2 track と接続。
6. **tests/tt** 受け入れテストを追加し、失敗→実装でグリーン。
7. **knowledge-database.md** の CLI 参照を `tt …` に更新。
8. **Verification Plan + §9 総合判定**（3 Part まとめて）。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Integration Tests（KDB 全体 + help）**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "Kdb|AgentHelp|Help"
    ```
    *   **Log Verification**: RoundTrip の content_hash 同一性、Help の 7 見出し、PATH 縮小ケース成功、systematize の add_candidates。

3. **E2E**: GUI 不要。理由: CLI/SQLite/FS のみ。統合テストが受け入れの正。

#### [NEW] [tt_kdb_backlog_test.go](file://tests/tt/tt_kdb_backlog_test.go)

*   上記テストケース一覧を実装。

### 先に書く試験（TDD レッド記録用）

| 試験名 | 狙い | Part |
| :--- | :--- | :--- |
| `TestCollapse_IdenticalAndConflict` | collapse 契約 | 3 unit |
| `TestPack_ContentHashStable` | content_hash / generated_at | 3 unit |
| `TestStore_UUIDAndNextID` | SQLite embed | 3 unit |
| `TestKdbBacklogRoundTrip` | 受け入れ R14-1..2 | 3 integration |
| `TestKdbBacklogCRUD` | R14-3 | 3 integration |
| `TestKdbTrackSystematizeBacklog` | R14-4 | 3 integration |
| `TestAgentHelp_UpHasSevenHeadings` | R14-7 | 1 |

### テスト項目セルフレビュー（§8.4）

| # | 問い | 結果 |
| :--- | :--- | :--- |
| 1 網羅性 | R14 の1–7を試験名に落としたか | はい（上表 + Part1/2 help/track） |
| 2 証拠 | hash・uuid・JSON フィールド・exit を断言するか | はい |
| 3 迂回 | Python/sqlite3 無し PATH で通すか / builtin が委譲に落ちないか | はい |
| 4 依存 | pipeline unit → store → ops → CLI → tests/tt | はい |

### 総合判定プロセス（§9）の計画

3 Part すべてグリーン後、実装者は次を記録する:

```markdown
### 総合判定結果
**判定**: （実装完了時に記入）
#### テスト結果サマリ
- ...
#### チェック項目の結果
| # | チェック項目 | 結果 | 備考 |
| 1 | スキップ | | |
| 2 | 部分的エラー | | |
| 3 | 迂回（python/sqlite3/委譲） | | |
| 4 | アダプタ誤適用（backlog builtin） | | |
| 5 | 順序依存 | | |
| 6 | 新機能カバレッジ | | |
| 7 | 外部状態（git/bash on Windows） | | |
```

## Documentation

#### [MODIFY] [prompts/manifest/code_content/capabilities/knowledge-database.md](file://prompts/manifest/code_content/capabilities/knowledge-database.md)

*   **更新内容**:
    *   操作入口を `tt kdb-track` / `tt kdb <id>` に変更。
    *   help は `tt help …`。
    *   置き場例に registry 正と `scripts/kdb/backlog` 推奨を追記。
    *   bash スクリプト前提の文を削除。

#### [MODIFY] [prompts/phases/000-foundation/branches/main/ideas/026-Tt-Kdb-Std-And-Agent-Help.md](file://prompts/phases/000-foundation/branches/main/ideas/026-Tt-Kdb-Std-And-Agent-Help.md)

*   **更新内容**: frontmatter `status: approved`（ユーザーが計画承認した時点で。実装開始時に実施してよい）。

## 継続計画について

本 Part で仕様 026 の必須要件実装は完了する。任意 O1–O4 は別計画。依頼元リポジトリのスクリプト削除・スキル差し替えは R16 どおり依頼元作業。

---

## 実装完了記録 (Execution Complete)

### 総合判定結果

**判定**: ✅ 動作確認完了

#### テスト結果サマリ
- Build `--backend-only`: PASS
- Integration `--categories tt --specify Kdb|AgentHelp|Help`: PASS (0 failures)
- 追加修正: failure JSON を stderr へ（仕様 R9）

#### チェック項目の結果
| # | チェック項目 | 結果 | 備考 |
|---|------------|------|------|
| 1 | スキップされたテスト | ✅ | 一部 git 未導入時 Skip のみ |
| 2 | 部分的なエラー | ✅ | ログに異常なし |
| 3 | 迂回（python/sqlite3） | ✅ | PATH 縮小試験 PASS |
| 4 | backlog builtin | ✅ | designdb 委譲と切り分け試験あり |
| 5 | 順序依存 | ✅ | 単独実行可能な統合試験 |
| 6 | 新機能カバレッジ | ✅ | help / track / backlog round-trip |
| 7 | 外部状態 | ✅ | modernc sqlite / bash 委譲 |

#### 判定理由
仕様 026 の必須受け入れ（R14）を Tokotachi フィクスチャ試験で確認した。失敗 JSON の stderr 契約も修正済み。依頼元スクリプト移行（R16）と O1–O4 は範囲外のまま。
