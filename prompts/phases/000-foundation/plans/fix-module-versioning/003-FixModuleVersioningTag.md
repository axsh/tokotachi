# 003-FixModuleVersioningTag

> **Source Specification**: [003-FixModuleVersioningTag.md](file://prompts/phases/000-foundation/ideas/fix-module-versioning/003-FixModuleVersioningTag.md)

## Goal Description

GitHub Release のタグ形式を `tt-vX.Y.Z` から `vX.Y.Z` に変更するスクリプト修正を再実施し、既存タグ `v0.4.4` を追加する。また、リリースノート生成の LLM プロンプトを英語化し、英語のリリースノートが出力されるようにする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. `publish.sh` のタグ形式を `vX.Y.Z` に変更 | Proposed Changes > `publish.sh` |
| 2. `github-upload.sh` の `get_current_version()` 修正 | Proposed Changes > `github-upload.sh` |
| 3. `tt-v0.4.4` と同コミットに `v0.4.4` タグ作成・push | Step-by-Step > Step 3（手動作業） |
| 4. 外部プロジェクトでの動作確認 | Verification Plan > Manual Verification |
| 5. リリースノートの英語化 | Proposed Changes > `summarizer.go` + `summarizer_test.go` |

## Proposed Changes

### リリースノート生成 (features/release-note)

> [!IMPORTANT]
> planning-rules に従い、テストファイルを先に記述する。

#### [MODIFY] [summarizer_test.go](file://features/release-note/internal/summarizer/summarizer_test.go)

*   **Description**: プロンプト英語化に伴い、日本語カテゴリ名を検証しているアサーションを英語カテゴリ名に変更する
*   **Technical Design**:
    *   `TestSummarizeBranch`:
        *   mock レスポンス: `"【新規】Feature A was added.\n【変更】Feature B was changed."` → `"[New] Feature A was added.\n[Changed] Feature B was changed."`
        *   結果検証: 上記に合わせて変更
        *   system prompt 検証: `"新規"`, `"変更"`, `"削除"` → `"New"`, `"Changed"`, `"Removed"`
    *   `TestConsolidate`:
        *   mock レスポンス内の日本語カテゴリ名を英語に変更: `"【新規】..."` → `"[New] ..."`、`"【変更】..."` → `"[Changed] ..."`
        *   system prompt 検証: `"統合"` / `"最終"` → `"consolidat"` / `"final"` (case-insensitive に `strings.Contains` で検証)

---

#### [MODIFY] [summarizer.go](file://features/release-note/internal/summarizer/summarizer.go)

*   **Description**: LLM への system prompt を日本語から英語に変更する
*   **Technical Design**:
    *   `branchSummarySystemPrompt` 定数（L12-20）を以下の内容に差し替え:

        ```
        You are a release note author. Read the specification files below and
        classify the changes into the following three categories, focusing on
        the impact to the end user (the person using the program):

        (1) [New]: New features, new settings, etc.
        (2) [Changed]: How existing features/settings have changed (Before → After)
        (3) [Removed]: Deprecated features, settings, etc.

        List items as bullet points under each category. Omit any category that
        has no items. Describe the "diff" from the user's perspective concisely.
        ```

    *   `consolidateSystemPrompt` 定数（L22-34）を以下の内容に差し替え:

        ```
        Below are summaries of multiple changes. Consolidate them into a final
        release note.

        Consolidation rules:
        - Remove intermediate states and describe only the final state
          (e.g. "A became B" + "B became C" → "A became C")
        - For duplicate changes to the same item, describe only the final state
        - If a removal and an addition share the same name, consolidate into
          "behavior changed" or similar
        - Group related items together
        - Focus on "what is the final outcome"
        - Provide a clear final diff for the user, not a verbose change history

        Output format:
        Classify items under (1) [New], (2) [Changed], (3) [Removed] as bullet
        points. Omit any category that has no items.
        ```

*   **Logic**: プロンプトの文章構造・指示内容は日本語版と同一。カテゴリ名のみ `【新規】`→`[New]`、`【変更】`→`[Changed]`、`【削除】`→`[Removed]` に変更。

---

### リリーススクリプト (scripts/dist)

#### [MODIFY] [publish.sh](file://scripts/dist/publish.sh)

*   **Description**: タグ生成からツールIDプレフィックスを削除
*   **Technical Design**:
    *   L41: `TAG` 変数の生成式を変更
*   **Logic**:
    *   変更前: `TAG="${TOOL_ID}-${VERSION}"` → 例: `tt-v0.4.3`
    *   変更後: `TAG="${VERSION}"` → 例: `v0.4.3`
    *   `TITLE="${TOOL_ID} ${VERSION}"` は変更なし（GitHub Release の表示名のみ）

---

#### [MODIFY] [github-upload.sh](file://scripts/dist/github-upload.sh)

*   **Description**: `get_current_version()` のタグ検索ロジックを新形式に対応させる
*   **Technical Design**:
    *   `get_current_version()` 関数（L35-54）を変更
*   **Logic**:
    *   **jq クエリの変更** (L45-46):
        *   変更前: `startswith("${tool_id}-v")` でツールIDプレフィックス付きタグを検索
        *   変更後: `test("^v[0-9]")` で `v` + 数字で始まるタグのみ検索
    *   **バージョン抽出の変更** (L48-53):
        *   変更前: `echo "${tag#${tool_id}-}"` でプレフィックスを除去
        *   変更後: `echo "$tag"` でタグをそのまま返す
    *   コメント `# Strip tool-id prefix: "tt-v1.0.0" → "v1.0.0"` を削除

## Step-by-Step Implementation Guide

1.  [x] **`summarizer_test.go` のアサーションを英語カテゴリに変更**:
    *   `features/release-note/internal/summarizer/summarizer_test.go` を編集
    *   `TestSummarizeBranch`: mock レスポンスとアサーション文字列を英語カテゴリ名に変更
    *   `TestSummarizeBranch`: system prompt 検証を `"New"`, `"Changed"`, `"Removed"` に変更
    *   `TestConsolidate`: mock レスポンスを英語カテゴリ名に変更
    *   `TestConsolidate`: system prompt 検証を `"consolidat"` / `"final"` に変更
2.  [x] **`summarizer.go` のプロンプトを英語化**:
    *   `features/release-note/internal/summarizer/summarizer.go` を編集
    *   `branchSummarySystemPrompt` を英語に差し替え
    *   `consolidateSystemPrompt` を英語に差し替え
3.  [x] **ビルド検証 (release-note モジュール)**:
    *   `./scripts/process/build.sh` を実行し、ビルドと単体テストが通ることを確認
4.  [x] **`publish.sh` のタグ生成を修正**:
    *   `scripts/dist/publish.sh` の L41 を `TAG="${VERSION}"` に変更
5.  [x] **`github-upload.sh` の `get_current_version()` を修正**:
    *   jq クエリを `test("^v[0-9]")` に変更
    *   バージョン抽出のプレフィックス除去コードを削除し `echo "$tag"` に変更
6.  [x] **ビルド検証 (全体)**:
    *   `./scripts/process/build.sh` を実行し、全体のビルドと単体テストが通ることを確認
7.  [x] **既存タグ `v0.4.4` の作成と push（手動作業）**:
    *   `git rev-list -n 1 tt-v0.4.4` でコミットハッシュを取得
    *   `git tag v0.4.4 <COMMIT>` でタグ作成
    *   `git push origin v0.4.4` で push
8.  [x] **外部プロジェクトでの動作確認（手動検証）**:
    *   外部プロジェクトで `go get github.com/axsh/tokotachi@v0.4.4` を実行
    *   `go.mod` に `github.com/axsh/tokotachi v0.4.4` が記録されることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    全体ビルドと単体テスト（`summarizer_test.go` 含む）を実行する。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **変更箇所の正確性チェック**:
    ```bash
    # publish.sh: TAG が ${VERSION} のみであること
    grep -n 'TAG=' scripts/dist/publish.sh

    # github-upload.sh: startswith が存在せず test("^v[0-9]") が存在すること
    grep -n 'test\|startswith' scripts/dist/github-upload.sh

    # summarizer.go: 英語カテゴリ名が存在すること
    grep -n 'New\|Changed\|Removed' features/release-note/internal/summarizer/summarizer.go
    ```

### Manual Verification

1.  **`v0.4.4` タグの確認**:
    *   `git rev-list -n 1 v0.4.4` と `git rev-list -n 1 tt-v0.4.4` の出力が一致すること
2.  **外部プロジェクトでのバージョン指定**:
    *   外部プロジェクトで `go get github.com/axsh/tokotachi@v0.4.4` を実行
    *   `go.mod` に pseudo-version ではなく `v0.4.4` が記録されること

## Documentation

変更対象となる既存の仕様書およびドキュメントはありません。
