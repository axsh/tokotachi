# 032-Catalog-Code-Content-Kdb-Prompt-Suite

> **Source Specification**: [prompts/phases/000-foundation/branches/main/ideas/030-Catalog-Code-Content-Kdb-Prompt-Suite.md](../ideas/030-Catalog-Code-Content-Kdb-Prompt-Suite.md)

## Goal Description

`go-standard-project` カタログ（BASE）に KDB / `_runs` / 文書規約の共通プロンプト一式を同梱し、tokotachi 本体（REPO）を同じ正本へ収束する。続けて `tt prompt update` と content release（ツールリリースなし）を行う。

## User Review Required

None.（仕様レビュー済み。ユーザー指示により計画→実装→content release を連続実行する）

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1 命名・`shared/kdb`・製品パス禁止 | 全新規/改訂本文 + ガードテスト禁止パターン |
| R2 KDB 最小 4 ファイル追加 | Proposed Changes > NEW capabilities/procedures |
| R3 接続改訂 | portable / pre-sync / record-fk / create-spec / create-reproducible-report |
| R4 REPO 収束 + `tt prompt update` | Step-by-Step の REPO 同期・deploy |
| R5 verify-and-specify-next + project-instructions | NEW procedure + MODIFY project-instructions |
| R6 非ゴール（coding/testing 製品差分を持ち込まない） | 変更対象外。テストで BASE coding-rules に legalchk 無し |
| R7 機械的ガード | NEW `tests/tt/tt_catalog_code_content_kdb_suite_test.go` |
| R8 スキャフォールド整合 | scaffold.yaml `gitignore_entries` + 文言検査 |
| R9 `_runs/` gitignore・レポート非 commit | scaffold.yaml + create-reproducible-report + tokotachi `.gitignore` |
| O1 terminology-rules | 先送り（実装しない） |
| O2 REPO 欠落 procedure 復元 | create-reproducible-report / create-pull-request / execute-all-after-spec / workflow-blueprint を REPO へ |
| O3 purge / record-e2e | 先送り（実装しない） |

## Proposed Changes

パス略称:

- `BASE` = `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content`
- `REPO` = `prompts/manifest/code_content`

### Guard test (TDD first)

#### [NEW] [tests/tt/tt_catalog_code_content_kdb_suite_test.go](tests/tt/tt_catalog_code_content_kdb_suite_test.go)

*   **Description**: R7 の存在・禁止/期待パターン・BASE=REPO（共有 id）・gitignore を断言する統合テスト。
*   **Technical Design**:
    *   パッケージは既存どおり `integration_test`。`projectRoot()` を利用。
    *   正規化: `strings.ReplaceAll(text, "\r\n", "\n")` 後に比較・検索。
*   **Logic**:
    *   **必須ファイル（BASE と REPO）**:
        *   `capabilities/knowledge-database.md`
        *   `capabilities/committed-doc-links.md`
        *   `procedures/create-backlog-item.md`
        *   `procedures/systematize-knowledge-database.md`
        *   `procedures/verify-and-specify-next.md`
        *   `procedures/create-reproducible-report.md`
    *   **バイト一致（EOL 正規化後）する共有 id**:
        *   上記 6 ファイル + `capabilities/portable-file-references.md` + `capabilities/pre-sync-knowledge-compile.md`
    *   **禁止サブストリング**（共有 id 本文）:
        *   `./scripts/kdb/`
        *   `features/kdb/registry.yaml` を必須とする文脈（簡易: 本文に `features/kdb/registry.yaml` が現れないこと。一例コメントも避ける）
        *   `legalchk`（共有 id のみ。coding-rules は別検査）
    *   **期待サブストリング**:
        *   `knowledge-database` / `systematize-knowledge-database` / `create-backlog-item` / `pre-sync`: `tt kdb-track` と `shared/kdb`
        *   `create-reproducible-report`: `_runs/procedures` を含み、かつ `git commit -m 'docs: add reproducible report` のような完了 commit 指示を含まない（`git commit` が本文に無い、または「git commit しない」を含む）
        *   `create-specification`（BASE）: `create-backlog-item` と `backlog:`
        *   `committed-doc-links`: `_runs/`
    *   **gitignore**:
        *   `catalog/originals/axsh/go-standard-project/scaffold.yaml` の本文に `prompts/phases/*/branches/*/_runs/`
        *   リポジトリルート `.gitignore` にも同パターン
    *   **非ゴール軽検査**: `BASE/policies/coding-rules.md` に `legalchk` が無い

### Catalog scaffold

#### [MODIFY] [catalog/originals/axsh/go-standard-project/scaffold.yaml](catalog/originals/axsh/go-standard-project/scaffold.yaml)

*   **Description**: R9 の `_runs/` を `post_actions.gitignore_entries` に追加。
*   **Logic**:
    *   既存 `post_actions.file_permissions` を維持しつつ、`gitignore_entries` を追加:

```yaml
  post_actions:
    gitignore_entries:
      - "prompts/phases/*/branches/*/_runs/"
    file_permissions:
      - pattern: "scripts/**/*.sh"
        executable: true
```

#### [MODIFY] [.gitignore](.gitignore)

*   **Description**: tokotachi 本体ワークスペースでも `_runs/` を無視する（R9 / REPO）。
*   **Logic**: ルートに `prompts/phases/*/branches/*/_runs/` を追記。

### Shared prompt content (BASE 正本 → REPO 同一)

実装時は BASE に正本を書き、同じバイト列を REPO へコピーする（共有 id）。

#### [NEW] `BASE|REPO/capabilities/knowledge-database.md`

*   **起点**: 現行 REPO の同名を正本とし、製品固有（`legalchk` 固定など）が残っていれば削除。`shared/kdb`・`tt kdb-track`・`systematize-knowledge-database`・UUID 節を維持。

#### [NEW] `BASE|REPO/capabilities/committed-doc-links.md`

*   **起点**: 現行 REPO / PLC 共通本文をそのまま採用。

#### [NEW] `BASE|REPO/procedures/create-backlog-item.md`

*   **起点**: vv6 の同名。
*   **Logic 変更点**:
    *   退避参照を `{{capability:kdb-systematize}}` → `{{procedure:systematize-knowledge-database}}`
    *   `--registry features/...` は書かない（既定 `shared/kdb`）
    *   `tt help kdb backlog` へ詳細を委譲する方針を維持

#### [NEW] `BASE|REPO/procedures/systematize-knowledge-database.md`

*   **起点**: PLC の同名。
*   **Logic 変更点**:
    *   全コマンド例から `--registry features/kdb/registry.yaml --kdb-root features` を除去（既定で動作）
    *   `features/<kdb>/corpus` 固定表現を `{kdb-root}/<id>/corpus` または `shared/kdb/...` の一般表現へ
    *   `tt kdb-track unregistered|status-all|systematize` を入口とする

#### [NEW] `BASE|REPO/procedures/verify-and-specify-next.md`

*   **Description**: 骨格版（R5）。
*   **Logic（必須要素）**:
    *   元仕様 + 報告を熟読 → 自分で確かめる → 直し仕様 or 次仕様を `{{procedure:create-specification}}` で書く
    *   報告の置き場は `_runs/procedures`（gitignore）。`ideas/` から `_runs/` へ Markdown リンクしない（`{{capability:committed-doc-links}}`）
    *   backlog は `tt kdb backlog` / `backlog:<uuid>`（既定フラグ）
    *   画面 blueprint・製品 E2E カテゴリ・RLS 固定・`features/kdb` 必須は書かない
    *   実装・計画作成はしない旨を明記

#### [MODIFY] `BASE|REPO/capabilities/portable-file-references.md`

*   **起点**: 現行 REPO（committed-doc-links 連携済み）を BASE にも適用し、共有 id として一致させる。

#### [MODIFY] `BASE|REPO/capabilities/pre-sync-knowledge-compile.md`

*   **Logic**:
    *   references に `systematize-knowledge-database`
    *   Step 1b: `tt kdb-track status-all` / dirty なら `tt kdb-track systematize`（または procedure 実行）
    *   `./scripts/kdb/*`・`features/kdb/registry.yaml`・`scripts/design/*` 必須依存を削除
    *   scripts frontmatter から design スクリプト参照を外す（残すなら intake のみ）

#### [MODIFY] `BASE|REPO/capabilities/record-far-knowledge.md`

*   **Logic**: REPO にある「KDB 立て付け全文を遠方知識に書かない」短文を BASE にも入れ、一致させる。

#### [MODIFY] `BASE/procedures/create-specification.md` および `REPO/procedures/create-specification.md`

*   **Logic**（両ファイルに同じ差分を入れる。他節の差異が残る場合はパターン検査のみで一致強制しない）:
    *   仮決定・確認事項 → `{{procedure:create-backlog-item}}` + `backlog:<uuid>`
    *   完了時の提示: ワークスペース相対 Markdown リンク（`file://` 禁止、`portable-file-references`）

#### [MODIFY] `BASE|REPO/procedures/create-reproducible-report.md`

*   **起点**: PLC 版を汎用化。
*   **Logic**:
    *   出力先 `_runs/procedures/`、**git commit しない**
    *   `{{procedure:record-e2e-results}}` / `CHKAI_E2E_RECORD` / 製品固有パスを削除
    *   E2E を含む場合は「プロジェクトが持つ E2E 記録手順があればそれに従う。無ければ結果を手順文に書く」程度の一般文に留める
    *   `committed-doc-links` 参照を維持
    *   BASE と REPO でバイト一致

### project-instructions（一致強制しない）

#### [MODIFY] [BASE/policies/project-instructions.md](catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/policies/project-instructions.md)

*   **Logic**:
    *   再現レポートの出力先表現を `_runs/procedures`（gitignore）に更新（旧 `procedures/` コミット対象表現があれば置換）
    *   `{{procedure:verify-and-specify-next}}` と `{{procedure:create-reproducible-report}}` をフローに接続（REPO 既存節を参考）

#### [MODIFY] [REPO/policies/project-instructions.md](prompts/manifest/code_content/policies/project-instructions.md)

*   **Logic**: 再現レポート参照が `_runs` になっていることを確認・修正。参照切れが無いこと。

### O2 REPO 復元（共有 id 以外）

#### [NEW] REPO へ BASE からコピー

*   `procedures/create-pull-request.md`
*   `procedures/execute-all-after-spec.md`
*   `capabilities/workflow-blueprint-design-philosophy.md`

（`create-reproducible-report` は共有 id として上記で扱う）

### prompt-manifest-update

#### [MODIFY] BASE / REPO `capabilities/prompt-manifest-update.md`

*   **Logic**: 仕様 025 と矛盾しないことを確認。大規模改訂なし。必要なら BASE の `\{{kind:id}}` 明示を維持し、REPO をそれに合わせる（任意で一致）。

### Deploy

*   `tt prompt update --target all`（失敗時は `--target cursor` で最低限）
*   生成物（`.cursor/skills/**` 等）をコミット対象に含める
*   `.cursor/skills/knowledge-database/SKILL.md` に `scripts/kdb` / 必須 `features/kdb` が残っていないこと

## Step-by-Step Implementation Guide

1. **[ ] 計画コミット**: 本計画を `git commit`。
2. **[ ] TDD レッド**: `tt_catalog_code_content_kdb_suite_test.go` を追加し、`./scripts/process/integration_test.sh --categories tt --specify CatalogCodeContentKdbSuite` が失敗することを確認。
3. **[ ] scaffold / root gitignore**: `scaffold.yaml` とルート `.gitignore` に `_runs/` 規則を追加。
4. **[ ] 共有 id 正本作成**: knowledge-database / committed-doc-links / create-backlog-item / systematize-knowledge-database / verify-and-specify-next / create-reproducible-report / portable-file-references / pre-sync / record-far-knowledge を BASE に書き、同一内容を REPO へコピー。
5. **[ ] create-specification / project-instructions**: BASE・REPO を改訂（R3/R5/R9）。
6. **[ ] O2 復元**: create-pull-request / execute-all-after-spec / workflow-blueprint を REPO へ。
7. **[ ] ガードテスト緑**: 同 `--specify` が通るまで修正。
8. **[ ] deploy**: `tt prompt update --target all`。生成物を確認。
9. **[ ] ビルド＋tt カテゴリ検証**: Verification Plan を実行。
10. **[ ] コミット**: 論理単位でコミット（テスト→コンテンツ→deploy）。
11. **[ ] 遠方知識**: `record-far-knowledge` スキルに従う。
12. **[ ] Content release**: `./scripts/dist/content/release.sh`（ツールリリースは行わない）。

最後に Verification Plan を実行すること。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:

```bash
./scripts/process/build.sh
```

2. **Integration Tests（本変更のガード + tt リグレッション）**:

```bash
./scripts/process/integration_test.sh --categories "tt" --specify "CatalogCodeContentKdbSuite|TtKdb|Prompt"
```

*   **Log Verification**: `CatalogCodeContentKdbSuite` が PASS。KDB / prompt 既存テストが回帰していない。

3. **E2E Tests**:

不要。理由: GUI / ブラウザ操作の変更がなく、対象はカタログ Markdown・gitignore・プロンプト配布のみ。動作確認は `tests/tt` のファイル内容断言と `tt prompt update` で足りる。

### 総合判定（testing-rules §12）

*   build.sh 成功
*   上記 integration 成功
*   `tt prompt update` 後の knowledge-database スキルに旧パス無し
*   content release が catalog scaffolds を更新して push 完了

### テスト項目設計のセルフレビュー（§11.4 相当）

| 観点 | 結果 |
| :--- | :--- |
| 網羅性 | R1–R9 のファイル存在・禁止/期待・一致・gitignore をテスト表に落とした |
| 証拠の十分性 | リポジトリ実ファイルを読む統合テストで恒久化 |
| 迂回排除 | 製品 coding-rules 全面置換を変更対象外にし、legalchk 非混入を断言 |
| 依存関係 | ガードテスト → コンテンツ → deploy → build → tt カテゴリ → content release |

## Documentation

#### [MODIFY] 仕様 status（任意）

*   `ideas/030-...md` の `status: draft` → `approved` または `implemented`（実装完了時）。

`prompts/specifications` 配下に本変更の影響ドキュメントは無い想定。無ければ Documentation 更新なし。
