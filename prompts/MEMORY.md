# メモリシステム (Far-Knowledge Memory System)

このディレクトリは、Coding Agent が作業中に獲得した**遠方知識 (Far-Knowledge)** を
構造的に蓄積・体系化し、プロジェクト全体で再利用可能にするためのシステムである。

## 遠方知識とは

遠方知識とは、**近傍コード (同一パッケージ、import先、呼び出し元) を検索しても発見できない知識** を指す。
具体的には以下のような情報が該当する:

| 種別 | 例 |
|------|-----|
| アーキテクチャ決定 | パッケージ間の依存方向、データオーナーシップ |
| 横断的パターン | 複数モジュールで共有すべき設計パターン |
| 慣例・規約 | ログ形式、API設計規約、テスト命名規則 |
| 教訓 | 過去の失敗やレビュー指摘から得た知見 |
| 設定・嗜好 | エンジニアの品質基準や優先度 |

逆に、同一パッケージの関数シグネチャや、importすれば分かる型定義は近傍知識であり、記録対象外である。

## ディレクトリ構成

```
prompts/memory/
  README.md          ... 本ファイル (システム全体の説明)
  index.md           ... メモリ文書のルートマップ (自動生成)
  schemas/           ... JSON Schema 定義
  var/               ... ランタイムデータ (intake events, logs)
    intake/
      pending/       ... 未処理の intake event (日付別)
      processed/     ... 処理済み event
      index.db       ... SQLite インデックス
  knowledge/         ... 体系化された知識ストア (カテゴリ階層)
  branches/          ... ブランチ固有データ
    <branch-id>/
      skills/        ... ブランチで生成された far-knowledge スキル
      knowledge/     ... ブランチ固有の知識 (将来拡張)
```

## データフロー

メモリシステムは以下の3段階で動作する:

```
[Record]           [Systematize]          [Emit]
Coding Agent       Coding Agent           tt prompt update
    |                   |                      |
    v                   v                      v
+--------+     +-------------+     +--------------------+
| record |---->| pending     |---->| knowledge/         |
|  .sh   |     | event (.json)|    |   category/        |
+--------+     +-------------+     |     item.md        |
                    |               +--------------------+
                    |                      |
                    v                      v
              +----------+     +---------------------+
              |processed |     | branches/*/skills/  |
              |event     |     |   __far-knowledge-* |
              +----------+     |     SKILL.md         |
                               +---------------------+
                                       |
                                       v
                               +-------------------+
                               | .agents/skills/   |
                               | .claude/skills/   |
                               | .cursor/skills/   |
                               +-------------------+
```

### Stage 1: Record (記録)

遠方知識の検出は、主に以下のタイミングで行われます：

1. **実装中の自発的検出**: 実装やデバッグ中に、他でも再利用可能な設計パターンや教訓をエージェントが発見した時。
2. **Gitプッシュ時の自動検証 (`pre-push-knowledge-check`)**: プッシュ前に変更差分を検証し、蓄積すべき遠方知識が含まれていると判定された時。

上記により遠方知識が検出された場合、`./scripts/code/agent/record.sh` を通じて intake event を記録します。

```bash
./scripts/code/agent/record.sh \
  --agent "antigravity" \
  --summary "API handlers must use apierror types" \
  --changed-paths-from-git \
  --design-pattern \
  --note "All API error responses use pkg/apierror for consistent client-facing messages"
```

**内部処理:**
1. `record.sh` が `tt agent record` コマンドを呼び出す
2. ペイロードが `agent-record-payload.schema.json` に基づいてバリデーションされる
3. 自動補完フィールド (event_id, timestamps, git情報, provenance) が付与される
4. `intake-event.schema.json` に準拠した JSON が `var/intake/pending/<date>/` に保存される

### Stage 2: Systematize (体系化)

Coding Agent が `systematize-far-knowledge` ワークフローを実行し、
pending events を分析・カテゴリ化して知識ストアに登録する。

**操作の流れ:**
1. `intake.sh list --status pending` で未処理イベントを一覧
2. `intake.sh show <event-id>` で各イベントの内容を確認
3. 距離判定: 近傍知識か遠方知識かを判別
4. `knowledge.sh add` / `knowledge.sh append` で知識ストアに登録
5. 必要に応じて `knowledge.sh split` / `merge` / `rename` / `move` で再整理
6. `intake.sh processed <event-id>` で処理済みに移動

**知識ストアの構造:**

```
knowledge/
  error-handling/
    _category.yaml        ... カテゴリメタデータ (title, description, timestamps)
    api-error-responses.md ... 知識ファイル (frontmatter + 本文)
    validation-errors.md
  testing/
    _category.yaml
    test-naming-conventions.md
  logging/
    _category.yaml
    structured-log-format.md
```

各知識ファイルには YAML frontmatter が付与される:

```yaml
---
knowledge_id: api-error-responses
title: API Error Responses
category_path: error-handling
created_at: 2026-06-15T10:00:00Z
last_updated: 2026-06-15T10:00:00Z
source_event_ids:
  - E-01TESTPROCESSED
---
```

### Stage 3: Emit (配信)

`tt prompt update` 実行時に、各 Coding Agent 向けのエミッターが
`branches/*/skills/` 以下の far-knowledge スキルを自動収集し、
各エージェントのスキルディレクトリに配信する。

- `.agents/skills/__far-knowledge-*/SKILL.md` (Antigravity / Codex)
- `.claude/skills/__far-knowledge-*/SKILL.md` (Claude Code)
- `.cursor/skills/__far-knowledge-*/SKILL.md` (Cursor)

## CLI コマンド体系

| コマンド | 目的 | Wrapper |
|---------|------|---------|
| `tt agent record` | intake event を記録 | `scripts/code/agent/record.sh` |
| `tt agent intake list` | event の一覧 | `scripts/code/agent/intake.sh list` |
| `tt agent intake show` | event の詳細表示 | `scripts/code/agent/intake.sh show` |
| `tt agent intake processed` | pending -> processed 移動 | `scripts/code/agent/intake.sh processed` |
| `tt agent knowledge add` | 新規カテゴリ + 知識追加 | `scripts/code/agent/knowledge.sh add` |
| `tt agent knowledge append` | 既存カテゴリに知識追記 | `scripts/code/agent/knowledge.sh append` |
| `tt agent knowledge list` | カテゴリツリー表示 | `scripts/code/agent/knowledge.sh list` |
| `tt agent knowledge split` | カテゴリ分割 | `scripts/code/agent/knowledge.sh split` |
| `tt agent knowledge merge` | カテゴリ統合 | `scripts/code/agent/knowledge.sh merge` |
| `tt agent knowledge rename` | カテゴリ改名 | `scripts/code/agent/knowledge.sh rename` |
| `tt agent knowledge move` | 知識ファイル移動 | `scripts/code/agent/knowledge.sh move` |

Wrapper スクリプトが存在する理由は、ワークフローからの呼び出しインターフェースとなり、
内部で使用する `tt` コマンドへの依存を分離するためである。

## カテゴリフラグ

`record.sh` で使用するフラグは2種類に分類される:

**既存フラグ (構造的変更)**

| フラグ | 用途 |
|--------|------|
| `--architecture-impact` | パッケージ追加/削除、モジュール境界変更 |
| `--memory-related` | メモリシステム自体の変更 |
| `--prompt-related` | プロンプトテンプレートの変更 |
| `--agent-behavior-related` | エージェントルール/ワークフロー変更 |
| `--requires-immediate-action` | 即時対応を要するイベント |

**遠方知識フラグ**

| フラグ | 用途 |
|--------|------|
| `--design-pattern` | モジュール横断の設計パターン |
| `--convention` | 規約・スタイルルール |
| `--lesson-learned` | 過去の失敗・レビュー指摘からの教訓 |
| `--preference` | エンジニアの品質基準・嗜好 |

## スキーマ定義

| ファイル | 用途 |
|---------|------|
| `schemas/agent-record-payload.schema.json` | `tt agent record` 入力ペイロード |
| `schemas/agent-record-result.schema.json` | `tt agent record` 出力結果 |
| `schemas/intake-event.schema.json` | 保存された intake event (自動補完後) |

## 再整理の設計思想

知識は階層化され、最下層に個別の知識ファイルが配置される。
カテゴリ構造は固定ではなく、知識が蓄積されるにつれて動的に再編成される。

再整理が必要になるシグナルは以下の通り:

- **split**: 1カテゴリの知識ファイルが多すぎる (5件以上)、
  または内容が2つ以上の明確に異なるサブトピックを含む
- **merge**: 2つのカテゴリの知識が頻繁に相互参照され、単独では不完全
- **rename**: カテゴリ名がその内容を正確に表現しなくなった
- **move**: 特定の知識項目が現在のカテゴリより別のカテゴリに強く関連する

再整理の操作は「何をすべきか (判断)」と「どうやるか (実行)」に分離されている。
判断は Coding Agent が行い、実行は `knowledge.sh` の各サブコマンドが担う。

## 関連するワークフロー・ポリシー

| ファイル | 役割 |
|---------|------|
| `prompts/manifest/code_content/capabilities/record-far-knowledge.md` | 記録スキル定義 |
| `prompts/manifest/code_content/capabilities/pre-push-knowledge-check.md` | push前チェック |
| `prompts/manifest/code_content/policies/far-knowledge-memory.md` | 記録ポリシー |
| `prompts/manifest/code_content/procedures/execute-implementation-plan.md` | 実装ワークフロー (Section 3.3) |
| `prompts/manifest/code_content/procedures/systematize-far-knowledge.md` | 体系化ワークフロー |

## 注意事項

- `var/` 以下はランタイムデータであり、`.gitignore` で除外される
- `knowledge/` 以下は git 管理対象であり、知識の永続化を担う
- `index.md` は `tt prompt compile` により自動生成される。手動編集しないこと
- Coding Agent は直接 `prompts/memory/` 以下のファイルを編集してはならない。
  必ず `record.sh` / `knowledge.sh` 経由で操作すること
