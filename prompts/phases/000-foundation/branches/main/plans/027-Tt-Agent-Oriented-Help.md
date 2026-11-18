# 027-Tt-Agent-Oriented-Help

> **Source Specification**: prompts/phases/000-foundation/branches/main/ideas/026-Tt-Kdb-Std-And-Agent-Help.md  
> **Series**: Part 1 of 3（Part 2: `028-Tt-Kdb-Track.md`、Part 3: `029-Tt-Kdb-Backlog.md`）

## Goal Description

`tt help …` / `--help` / 位置引数 `help` が、エージェント手順書として使える **英語 Markdown（7 見出し固定）** を返すようにする。既存の全トップレベルコマンドと、エージェントが日常使う主要子孫（`prompt` / `agent` / ワークツリー系）を対象とする。KDB コマンド本体は Part 2/3 で追加し、本 Part の help 基盤を再利用する。

## User Review Required

1. R12 の「全トップレベル」に `scaffold` / `pr` / `doctor` / `update-code-status` / `merge` も含める（仕様どおり全部）。文言品質は実装時に現行 Long を拡張する一次品質でよい（後追い polish 可）。
2. ルート `tt help`（引数なし）はコマンド一覧 + `tt help <cmd>` 案内の Markdown。cobra 既定の usage スタイルは使わない。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point |
| :--- | :--- |
| R10 help 共通契約（英語・7見出し・副作用なし・Markdown・exit 0） | `internal/clihelp` + 各 cmd の `AgentHelp` 登録 |
| R11 `tt help <path>` と `--help`/`-h`/`help` の同等到達 | `cmd/help_agent.go`（SetHelpFunc / help コマンド置換） |
| R12 既存コマンドへの適用 | 各 `features/tt/cmd/*.go` の help 本文登録 |
| R13（KDB 分） | Part 2/3 で同一基盤に本文を載せる（本 Part で基盤のみ） |
| R14-5, R14-7 help 受け入れ | `tests/tt/tt_agent_help_test.go` |
| O2 自動生成 | 本 Part では **先送り**（人手定数）。Exit codes は本文に明示 |

## Proposed Changes

### clihelp（共通）

#### [NEW] [features/tt/internal/clihelp/doc.go](file://features/tt/internal/clihelp/doc.go)

*   **Description**: パッケージ説明。
*   **Logic**: `Package clihelp builds agent-oriented Markdown help pages.`

#### [NEW] [features/tt/internal/clihelp/doc_test.go](file://features/tt/internal/clihelp/doc_test.go)

*   **Description**: TDD — 見出し順・必須セクション検証を先に書く。
*   **Logic**:
    *   `TestRender_SectionOrder`: `Render(Page{...})` の出力に次が **この順** で現れること:
      `## Purpose`, `## When to use`, `## When not to use`, `## Preconditions`, `## Input and output`, `## Exit codes`, `## Read next`
    *   `TestRender_TitleHeading`: 先頭が `# <Title>`（例: `# tt up`）
    *   `TestHasRequiredHeadings`: ヘルパーが 7 見出し欠落を検出

#### [NEW] [features/tt/internal/clihelp/page.go](file://features/tt/internal/clihelp/page.go)

*   **Description**: ページ構造とレンダラ。
*   **Technical Design**:
    ```go
    package clihelp

    // Page is one agent-oriented help topic (English Markdown).
    type Page struct {
        Title          string // e.g. "tt up", "tt prompt update"
        Purpose        string
        WhenToUse      string
        WhenNotToUse   string
        Preconditions  string
        InputAndOutput string
        ExitCodes      string
        ReadNext       string // lines of real topics only, e.g. "`tt help down`"
    }

    // RequiredHeadings in order.
    var RequiredHeadings = []string{
        "Purpose", "When to use", "When not to use", "Preconditions",
        "Input and output", "Exit codes", "Read next",
    }

    func Render(p Page) string { /* markdown with ## headings; trim spaces; ensure trailing newline */ }

    // ContainsRequiredHeadings reports whether s includes all ## headings in order.
    func ContainsRequiredHeadings(s string) bool { ... }
    ```
*   **Logic**:
    *   本文フィールドは生 Markdown 断片（箇条書き可）。空欄は `_(none)_` を出さず、最低1文を登録側で必ず埋める（空 Page はテストで拒否）。
    *   `Validate(p Page) error`: Title と 7 フィールド非空。

#### [NEW] [features/tt/internal/clihelp/registry.go](file://features/tt/internal/clihelp/registry.go)

*   **Description**: コマンドパス → Page の登録表。
*   **Technical Design**:
    ```go
    // path key: space-separated cobra command path without "tt", e.g. "up", "prompt update", "agent record"
    var pages = map[string]Page{}

    func Register(path string, p Page) { pages[path] = p }
    func Lookup(path string) (Page, bool)
    func MustLookup(path string) Page // panic in init only if missing — prefer Lookup in runtime
    func AllPaths() []string
    ```
*   **Logic**: `init` は各 cmd ファイル側の `func init() { clihelp.Register(...) }` で埋める。

### cmd（help 配線）

#### [NEW] [features/tt/cmd/help_agent_test.go](file://features/tt/cmd/help_agent_test.go)

*   **Description**: TDD — `tt help` / `tt help up` が Markdown 7 見出し・exit 0。
*   **Logic**:
    *   root の `Execute` をテスト用に `SetArgs` + `SetOut`/`SetErr` で実行（既存 prompt テスト流儀に合わせる）。
    *   `TestAgentHelp_RootListsCommands`: 出力に `tt help up` 案内と主要コマンド名。
    *   `TestAgentHelp_UpHasSevenHeadings`
    *   `TestAgentHelp_PromptUpdateHasSevenHeadings`
    *   `TestAgentHelp_NoSideEffects`: 一時ディレクトリを workspace にしてもファイルが増えない（help のみ）。

#### [NEW] [features/tt/cmd/help_agent.go](file://features/tt/cmd/help_agent.go)

*   **Description**: cobra help をエージェント向けに置換。
*   **Technical Design**:
    ```go
    func init() {
        rootCmd.SetHelpCommand(&cobra.Command{
            Use:   "help [command path...]",
            Short: "Show agent-oriented Markdown help",
            RunE:  runAgentHelp,
        })
        rootCmd.SetHelpFunc(agentHelpFunc) // --help / -h
    }

    func commandPathOf(cmd *cobra.Command) string {
        // join cmd.CommandPath() without root name "tt" → "prompt update"
    }

    func agentHelpFunc(cmd *cobra.Command, args []string) {
        path := commandPathOf(cmd)
        if path == "" {
            writeRootHelp(cmd.OutOrStdout())
            return
        }
        if p, ok := clihelp.Lookup(path); ok {
            fmt.Fprint(cmd.OutOrStdout(), clihelp.Render(p))
            return
        }
        // fallback: still emit a minimal Page synthesized from Short/Long + stub sections
        // so R12 rollout never leaves a command on raw cobra usage
        fmt.Fprint(cmd.OutOrStdout(), clihelp.Render(synthesizeFromCobra(cmd)))
    }
    ```
*   **Logic**:
    *   `runAgentHelp`: args が空 → root 一覧。非空 → `rootCmd.Find(args)` でコマンド特定し `agentHelpFunc`。
    *   Find 失敗時は stderr JSON ではなく、help 文脈では Markdown エラー文 + exit 1（既存 cobra に近い）。KDB 操作失敗 JSON とは別（help 専用）。
    *   `synthesizeFromCobra`: Purpose=Long/Short、他セクションに暫定英文を入れ **必ず 7 見出し**を満たす（未登録コマンドの安全網）。本番対象コマンドは Register 必須。

#### [MODIFY] [features/tt/cmd/root.go](file://features/tt/cmd/root.go)

*   **Description**: root 用 Page 登録。Long を短く保ち、詳細は help へ。
*   **Logic**:
    *   `clihelp.Register("", Page{Title: "tt", ...})` または root 専用 `writeRootHelp` が一覧を動的生成（登録済み paths + 子コマンド Short）。
    *   Short に「Agent manuals: `tt help <command>`」を含める。

#### [MODIFY] 既存コマンドファイル（help 本文登録）

対象（トップレベル + 主要子孫）。各ファイルの `init` 末尾で `clihelp.Register`。

| path key | ファイル |
| :--- | :--- |
| `up` | `up.go` |
| `down` | `down.go` |
| `open` | `open.go` |
| `close` | `close.go` |
| `create` | `create.go` |
| `delete` | `delete.go` |
| `editor` | `editor.go` |
| `status` | `status.go` |
| `shell` | `shell.go` |
| `exec` | `exec_cmd.go` |
| `list` | `list.go` |
| `merge` | `merge.go` |
| `pr` | `pr.go` |
| `doctor` | `doctor.go` |
| `scaffold` | `scaffold.go` |
| `update-code-status` | `update_code_status.go` |
| `prompt` | `prompt.go` |
| `prompt compile` / `prompt deploy` / `prompt update` / `prompt instruction` | `prompt.go` |
| `agent` | `agent.go` |
| `agent record` / `agent status` / `agent intake`（+ list/show/processed） / `agent knowledge`（+ 主要 sub） | 各 `agent_*.go` |

*   **Logic（各 Page の書き方）**:
    *   Purpose: 現行 Long を1段落に整理（英語）。
    *   When to use / When not to use: 典型契機と禁止事項を各2–5行。
    *   Preconditions: 例: worktree が存在する、Docker が使える、など。
    *   Input and output: 主要フラグと成功時の見え方。
    *   Exit codes: `0` success、`1` failure（現行がそれ以上あれば列挙）。
    *   Read next: 関連 `tt help …` のみ。
    *   `prompt instruction` は既存マニュアル出力を維持しつつ、`--help` は 7 見出し Page（マニュアル本体は `instruction` 実行時）。矛盾しないよう Purpose から `tt prompt instruction` へ誘導。

### 統合テスト

#### [NEW] [tests/tt/tt_agent_help_test.go](file://tests/tt/tt_agent_help_test.go)

*   **Description**: ビルド済み `tt` バイナリで help 契約を検証（R14-7、検証シナリオ 2/11）。
*   **Logic**:
    *   `ttBinary(t)` を使用。
    *   テーブル: `{"help"}, {"help","up"}, {"help","prompt","update"}, {"help","agent","record"}, {"up","--help"}, {"prompt","update","--help"}`
    *   各ケース: exit 0、stdout に 7 見出し、stderr 空（または help 以外のノイズなし）。
    *   `TestAgentHelp_DoesNotCreateSqlite`: 一時 DIR を cwd にして `tt help agent record` 後、DIR 内に新規ファイルが無いこと。

## Step-by-Step Implementation Guide

1. **TDD clihelp**: `page_test.go` / `Render` / `Validate` を追加し、失敗を確認してから `page.go` を実装。
2. **Registry**: `registry.go` + 単体テスト（Register/Lookup）。
3. **help_agent 配線**: `help_agent_test.go` を先に書き、`help_agent.go` + `root.go` を実装。
4. **既存コマンド Page 登録**: トップレベルから順に Register。各追加後に `ContainsRequiredHeadings` を通すテーブルテストを `help_agent_test.go` に `AllPaths` 走査で追加（未登録トップレベル子が無いこと）。
5. **統合テスト**: `tt_agent_help_test.go` を追加。
6. **Verification Plan を実行**（本 Part 範囲）。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Integration Tests（help）**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "AgentHelp|Help"
    ```
    *   **Log Verification**: 全ケース exit 0、見出し欠落失敗が無いこと。

3. **E2E / 統合テストコード**:
    #### [NEW] [tt_agent_help_test.go](file://tests/tt/tt_agent_help_test.go)
    *   GUI E2E は不要（CLI のみ）。理由: 画面変更なし。CLI 契約は `tests/tt` 統合テストで足りる。

### テスト項目セルフレビュー（§8.4）

| # | 問い | 結果 |
| :--- | :--- | :--- |
| 1 網羅性 | help の契約（7見出し・exit0・副作用なし・主要 path）を単体+統合で押さえたか | はい。KDB path は Part 2/3 で同テストへ行追加 |
| 2 証拠 | 見出し文字列と exit code を断言しているか | はい |
| 3 迂回 | synthesize フォールバックだけで「登録済み」とみなしていないか | `AllPaths` に必須コマンドが Register されていることを単体で固定 |
| 4 依存 | clihelp 単体 → cmd 単体 → tests/tt | 順序どおり |

### 総合判定プロセス（§9）の計画

本 Part 完了後、実装者は次を計画書末尾またはウォークスルーに記録する:

*   スキップ/WARN の有無、help が DB を触っていないこと、必須 path の Register 漏れが無いこと。
*   判定フォーマットは testing-rules §9.3 に従う。

## Documentation

`prompts/specifications/` はリポジトリに存在しない。本 Part では:

#### [MODIFY] [prompts/manifest/code_content/capabilities/knowledge-database.md](file://prompts/manifest/code_content/capabilities/knowledge-database.md)

*   **更新内容**: Part 2/3 完了時に bash スクリプト参照を `tt kdb-track` / `tt kdb …` / `tt help …` へ更新する（**本 Part では触らない**。継続計画の Documentation に委譲）。

## 継続計画について

*   **Part 2** (`028-Tt-Kdb-Track.md`): registry、`tt kdb-track`、委譲、`tt kdb` ディスパッチ骨格、help 登録。
*   **Part 3** (`029-Tt-Kdb-Backlog.md`): backlog builtin 全 op、`bl_json` 相当、受け入れ round-trip。
