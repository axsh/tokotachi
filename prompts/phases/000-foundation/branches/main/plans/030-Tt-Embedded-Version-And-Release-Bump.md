# 030-Tt-Embedded-Version-And-Release-Bump

> **Source Specification**: prompts/phases/000-foundation/branches/main/ideas/027-Tt-Embedded-Version-And-Release-Bump.md

## Goal Description

`tt` に製品バージョン定数（初期値 `v0.9.1`）をソース正本として埋め込み、`tt version` / `tt --version` / `tt -V` で表示する。`scripts/dist/tool/release.sh` は Build 前にその定数を `NEW_VERSION` へ書き換え・検証し、Publish 前にコミットしてタグとソースとバイナリの版を一致させる。

## User Review Required

None. O1（ldflags メタ）と O2（`--json`）は本計画では実装しない（仕様の任意要件・非ゴール寄り）。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1 ソース定数 `v0.9.1`（1 ファイル・1 シンボル） | `features/tt/internal/version/version.go` |
| R2 `tt version` stdout・exit 0 | `features/tt/cmd/version.go` |
| R2 `tt --version` / `tt -V` | `features/tt/cmd/root.go`（`rootCmd.Version` + shorthand `V`） |
| R2 `tt help version` 7 見出し英語 | `features/tt/cmd/help_pages.go` `registerMiscHelp` |
| R2 ルートから `version` へ辿れる | ルート help が全コマンド一覧を動的出力；`version` Short 明記 |
| R3 Build 前に定数書き換え＋検証 | `scripts/dist/tool/internal/set_embedded_version.sh` + `release.sh` |
| R3 対象は正本ファイルのみ | ヘルパが `const Version = "..."` 行のみ置換 |
| R3 ローカル `build.sh` は定数をそのまま埋め込み | `internal/build.sh` 変更なし（ldflags なし） |
| R4 タグ・ソース定数・バイナリ一致 | rewrite → verify → build → package → **commit version.go** → publish |
| R5 / O1 / O2 非ゴール・任意 | 本計画で実装しない |
| 検証シナリオ 1–3 CLI | `tests/tt/tt_version_test.go` |
| 検証シナリオ 4–6 書き換え | `tests/tt/tt_set_embedded_version_test.go`（フィクスチャ＋スクリプト） |

## Proposed Changes

### version package

#### [NEW] [features/tt/internal/version/version_test.go](file://features/tt/internal/version/version_test.go)

*   **Description**: TDD — 定数の初期値と形式を先に固定する。
*   **Logic**:
    *   `TestVersion_InitialValue`: `Version == "v0.9.1"`
    *   `TestVersion_SemVerWithLeadingV`: `regexp.MustCompile(^v[0-9]+\.[0-9]+\.[0-9]+$)` にマッチ
    *   `TestVersion_NonEmpty`: `Version != ""`

#### [NEW] [features/tt/internal/version/version.go](file://features/tt/internal/version/version.go)

*   **Description**: 製品バージョンの単一正本。
*   **Technical Design**:
    ```go
    package version

    // Version is the tt product version (semver with leading "v").
    // Updated by the tool release procedure before rebuild.
    const Version = "v0.9.1"
    ```
*   **Logic**: 他パッケージはこの定数のみを参照する。プレリリース接尾辞は置かない。

### cmd（CLI）

#### [NEW] [features/tt/cmd/version_test.go](file://features/tt/cmd/version_test.go)

*   **Description**: TDD — サブコマンド実行の単体（`SetArgs` + buffer）。
*   **Logic**:
    *   `TestVersionCmd_PrintsEmbedded`: `rootCmd.SetArgs([]string{"version"})` → stdout に `version.Version` を含む・err nil
    *   出力形式は `tt <version>\n`（例: `tt v0.9.1\n`）

#### [NEW] [features/tt/cmd/version.go](file://features/tt/cmd/version.go)

*   **Description**: `tt version` サブコマンド。
*   **Technical Design**:
    ```go
    var versionCmd = &cobra.Command{
        Use:   "version",
        Short: "Print the tt product version",
        Long:  "Print the embedded product version string (no network, no workspace writes).",
        Args:  cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Fprintf(cmd.OutOrStdout(), "tt %s\n", version.Version)
            return nil
        },
    }
    ```
*   **Logic**: ネットワーク・ワークスペース書き込みなし。成功時 exit 0。

#### [MODIFY] [features/tt/cmd/root.go](file://features/tt/cmd/root.go)

*   **Description**: ルートに Version と `version` コマンドを配線。
*   **Technical Design**:
    ```go
    import "github.com/axsh/tokotachi/features/tt/internal/version"

    // in var rootCmd:
    // Version field cannot be set in composite literal easily with import cycle — set in init:

    func init() {
        rootCmd.Version = version.Version
        rootCmd.SetVersionTemplate("tt {{.Version}}\n")
        rootCmd.InitDefaultVersionFlag()
        if f := rootCmd.Flags().Lookup("version"); f != nil {
            f.Shorthand = "V" // spec: tt -V (avoid conflict with --verbose)
        }
        // ... existing flags ...
        rootCmd.AddCommand(versionCmd)
        // ... existing AddCommand ...
    }
    ```
*   **Logic**:
    *   `tt --version` と `tt -V` は cobra 標準 Version 機構で `tt v0.9.1\n` を出す
    *   `versionCmd` も同じ定数を参照

#### [MODIFY] [features/tt/cmd/help_pages.go](file://features/tt/cmd/help_pages.go)

*   **Description**: `version` のエージェント向け help を登録。ルート Short から辿れるよう Read next に version を含める余地あり。
*   **Logic** (`registerMiscHelp` に追加):
    ```go
    clihelp.Register("version", page(
        "tt version",
        "Print the embedded tt product version string.",
        "When confirming which release binary or local build is running.",
        "Do not use to bump the release; use the tool release procedure.",
        "None (no network, no workspace writes).",
        "`tt version`\nAlso: `tt --version` or `tt -V`\nSuccess prints `tt vN.N.N` on stdout.",
        "0 success",
        "`tt help`\n`tt help doctor`",
    ))
    ```
*   **Logic (root help Read next)**: `writeRootHelp` の Read next に `` `tt help version` `` を 1 行追加し、ルートから辿れることを明示。

### release scripts

#### [NEW] [scripts/dist/tool/internal/set_embedded_version.sh](file://scripts/dist/tool/internal/set_embedded_version.sh)

*   **Description**: R1 正本ファイルの `const Version = "..."` だけを置換し、置換後の値が一致することを検証する。
*   **Technical Design**:
    ```bash
    # Usage: set_embedded_version.sh <version-go-path> <new-version>
    # Example: set_embedded_version.sh features/tt/internal/version/version.go v0.9.2
    ```
*   **Logic**:
    1. 引数チェック；`validate_version_format` 相当（`^v[0-9]+\.[0-9]+\.[0-9]+$`）
    2. ファイル存在確認
    3. `const Version = "OLD"` 行がちょうど 1 行あることを確認（無い／複数なら fail）
    4. 置換: `const Version = "<NEW_VERSION>"`（perl または sed で当該行のみ）
    5. 置換後に `grep -E 'const Version = "'"$NEW"'"'` で一致検証；失敗なら exit 1
    6. 無関係な `version:` YAML 等は触らない（パス固定＋行パターン固定）

#### [MODIFY] [scripts/dist/tool/release.sh](file://scripts/dist/tool/release.sh)

*   **Description**: Build 前に埋め込み定数を書き換え、Publish 前にコミットする。
*   **Logic**（`compare_versions` 成功・確認表示の後）:
    1. **Step 0（新）: Embed version**
       *   `VERSION_GO="${REPO_ROOT}/features/tt/internal/version/version.go"`
       *   `"${SCRIPT_DIR}/internal/set_embedded_version.sh" "$VERSION_GO" "$NEW_VERSION"`
       *   成功ログ: `Embedded version ${NEW_VERSION} into ${VERSION_GO}`
    2. **Step 1: Build**（既存 `internal/build.sh`）— 書き換え済みソースでクロスコンパイル
    3. Step 2–3: notes / package（既存）
    4. **Step 3.5（新）: Commit embedded version（Publish 前）**
       *   `git -C "$REPO_ROOT" add -- features/tt/internal/version/version.go`
       *   差分が無い場合（既に同版）は commit スキップしてよいが、ファイル内容は `NEW_VERSION` であること
       *   差分がある場合: `git commit -m "chore(tt): bump embedded version to ${NEW_VERSION}"`
       *   理由: `publish.sh` の `gh release create` は **現在の HEAD** にタグを打つため、コミット前に Publish するとタグ内ソースが旧版のままになる
    5. **Step 4: Publish**（既存）

*   **制約**: `TOOL_ID` が `tt` のときのみ埋め込み書き換えを行う。他ツールは現状どおり（R5: 他ツール一括展開は非ゴール）。`tt` 以外で呼ばれた場合は Step 0 をスキップし warn する。

### integration / E2E tests（tt カテゴリ）

#### [NEW] [tests/tt/tt_version_test.go](file://tests/tt/tt_version_test.go)

*   **Description**: ビルド済み `bin/tt` で version 表示と help を検証。
*   **テストケース**:
    *   `TestVersion_Subcommand`: `tt version` → exit 0、stdout に `v0.9.1`（または現行 `version.Version` と同型の `vN.N.N`）。実装後はソース定数と一致させるため、テストはバイナリ出力が `tt v` で始まり SemVer 形式であること、かつ `features/tt/internal/version/version.go` から読んだ定数と一致することを確認
    *   `TestVersion_FlagLong`: `tt --version` 同様
    *   `TestVersion_FlagShortV`: `tt -V` 同様
    *   `TestVersion_Help`: `tt help version` が 7 見出しを含む
*   **検証ポイント**: ネットワーク不要；一時 dir で実行してもファイルを書かない（任意で確認可）

#### [NEW] [tests/tt/tt_set_embedded_version_test.go](file://tests/tt/tt_set_embedded_version_test.go)

*   **Description**: 書き換えヘルパの自動検証（Publish なし）。
*   **テストケース**:
    *   `TestSetEmbeddedVersion_RewritesOnlyConst`: 一時ディレクトリに `version.go` フィクスチャ（`const Version = "v0.9.1"` とダミーコメント `version: 2`）を置き、`set_embedded_version.sh` を `v0.9.2` で実行 → `Version` が `v0.9.2`、コメント `version: 2` は不変
    *   `TestSetEmbeddedVersion_RejectsBadFormat`: `not-a-version` で非ゼロ終了
    *   `TestSetEmbeddedVersion_RejectsMissingConst`: const 行の無いファイルで非ゼロ終了

## Step-by-Step Implementation Guide

1. **Red: version package tests**:
    *   Create `features/tt/internal/version/version_test.go`（初期値・形式）。
    *   Create stub `version.go` 無し、または空で失敗を確認（実装と同時コミット可だが順序はテスト先行）。
2. **Green: version.go**:
    *   Add `const Version = "v0.9.1"`.
3. **Red: cmd version tests + CLI integration tests**:
    *   Add `cmd/version_test.go` と `tests/tt/tt_version_test.go`（まだ失敗）。
4. **Green: CLI wiring**:
    *   Add `cmd/version.go`；modify `root.go`（Version / -V / AddCommand）；register help in `help_pages.go`；update root Read next。
5. **Red: set_embedded_version tests**:
    *   Add `tests/tt/tt_set_embedded_version_test.go`（スクリプト未実装で失敗）。
6. **Green: release helper + release.sh**:
    *   Add `set_embedded_version.sh`；wire into `release.sh`（tt のみ、Build 前 rewrite、Publish 前 commit）。
7. **Docs**:
    *   Update `scripts/dist/README.md` と必要ならルート `README.md` の Tool Release 節に「埋め込み定数を書き換えてからビルド／Publish 前に commit」を追記。
8. **Verification Plan を実行**し、計画書末尾に総合判定結果を追記する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests** (tt version + help リグレッション):
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "Version|AgentHelp|SetEmbeddedVersion"
    ```
    *   **Log Verification**: `TestVersion_*` が exit 0 と埋め込み定数一致；`TestSetEmbeddedVersion_*` が書き換え成功／不正入力拒否；`TestAgentHelp_*` が引き続き 7 見出し。

3.  **E2E Tests**:
    *   GUI E2E は不要（CLI のみ）。CLI の受け入れは `tests/tt/tt_version_test.go` / `tt_set_embedded_version_test.go` が担う（本リポジトリの tt 統合テストが E2E 相当）。

### テスト項目設計（§11）

| 順序 | 対象 | 観点 |
| :--- | :--- | :--- |
| 1 | `version.Version` 定数 | 正常・形式（§11.3 #1,#2） |
| 2 | `tt version` / flags（cmd 単体＋bin） | 正常・副作用なし（#1,#7） |
| 3 | `tt help version` | 契約 7 見出し（#1,#6） |
| 4 | `set_embedded_version.sh` | 置換・非対象保全・異常系（#1,#2,#4,#7） |

**セルフレビュー (§11.4)**:
1. **網羅性**: 定数 → CLI 表示 → リリース書き換えの三段が揃えば「埋め込み版が表示され、リリースで更新できる」と言い切れる。
2. **証拠**: 出力文字列がソース定数と一致；フィクスチャ差分で意図した行のみ変化。
3. **迂回排除**: バイナリ試験は `bin/tt`；ヘルパ試験はスクリプト実体を実行（モックなし）。
4. **依存**: 定数試験が先、CLI が次、スクリプトが最後。

### 総合判定プロセス (§12)

全テスト成功後、計画書に以下を追記する:

```markdown
### 総合判定結果
**判定**: ...
#### テスト結果サマリ / チェック項目の結果 / 判定理由
```

## Documentation

#### [MODIFY] [scripts/dist/README.md](file://scripts/dist/README.md)

*   **更新内容**: Quick Release の流れに「`NEW_VERSION` を `features/tt/internal/version/version.go` に書き込んでから build」「Publish 前に version.go を commit」を追記。`set_embedded_version.sh` を Directory Structure に追加。

#### [MODIFY] [README.md](file://README.md)

*   **更新内容**: Tool Release 節に、埋め込みバージョンの正本パスと `tt version` での確認、リリース時にソースが書き換わる旨を短く追記。
