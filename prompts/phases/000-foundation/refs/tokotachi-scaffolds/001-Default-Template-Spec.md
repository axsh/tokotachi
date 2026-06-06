# default テンプレート仕様書

## 概要

`devctl scaffold` をパターン指定なしで実行した際に適用される **tokotachi 標準プロジェクト構成** テンプレート。新規リポジトリの初期セットアップに使用する。

## テンプレート情報

| 項目 | 値 |
|---|---|
| テンプレート名 | `default` |
| カテゴリ | `root` |
| 前提条件 | なし（新規リポジトリ向け） |
| オプション | なし |
| コンフリクトポリシー | `skip`（既存ファイルはスキップ） |
| 後処理 | `.gitignore` への `work/*` 追記 |

---

## リポジトリ内での配置

### catalog.yaml への登録

```yaml
default_scaffold: "default"

scaffolds:
  - name: "default"
    category: "root"
    description: "tokotachi 標準プロジェクト構成"
    template_ref: "templates/project-default"
    placement_ref: "placements/default.yaml"
    requirements:
      directories: []
      files: []
```

### 配置定義: `placements/default.yaml`

```yaml
version: "1.0.0"
base_dir: "."
conflict_policy: "skip"

template_config:
  template_extension: ".tmpl"
  strip_extension: true

file_mappings: []

post_actions:
  gitignore_entries:
    - "work/*"
```

---

## フォルダ・ファイル構成

### `templates/project-default/base/`

```
templates/project-default/base/
├── features/
│   └── README.md
├── prompts/
│   ├── phases/
│   │   ├── README.md
│   │   └── 000-foundation/
│   │       ├── ideas/
│   │       │   └── .gitkeep
│   │       └── plans/
│   │           └── .gitkeep
│   └── rules/
│       └── .gitkeep
├── scripts/
│   └── .gitkeep
├── shared/
│   ├── README.md
│   └── libs/
│       └── README.md
└── work/
    └── README.md
```

### ファイル内容

#### `features/README.md`

```markdown
# Features

This directory contains individual features of the project.

Each feature is an independent module with its own codebase, 
tests, and configuration.

## Adding a Feature

Use `devctl scaffold features <template>` to generate a new feature structure.

## Directory Convention

```
features/
  <feature-name>/
    cmd/           # CLI entry points
    internal/      # Internal packages
    go.mod         # Go module definition
```
```

#### `prompts/phases/README.md`

```markdown
# Phases

This directory organizes specifications and plans by project phase.

## Structure

```
phases/
  000-foundation/     # Phase 0: Foundation
    ideas/            # Specification documents
    plans/            # Implementation plans
  001-<next-phase>/   # Phase 1: ...
    ideas/
    plans/
```

## Workflow

1. Create a specification in `ideas/<branch-name>/`
2. Review and approve the specification
3. Create an implementation plan in `plans/<branch-name>/`
4. Review and approve the plan
5. Execute the plan
```

#### `shared/README.md`

```markdown
# Shared

This directory contains code and resources shared across multiple features.

## Structure

```
shared/
  libs/       # Shared libraries
```
```

#### `shared/libs/README.md`

```markdown
# Shared Libraries

This directory contains reusable libraries shared across features.

Place language-specific library packages here:

```
libs/
  go/         # Shared Go packages
  python/     # Shared Python modules
```
```

#### `work/README.md`

```markdown
# work

This directory is reserved for temporary working directories and development worktrees.

It is primarily used during development when working with:

* Git worktrees
* Parallel feature development
* Automated agent sessions
* Temporary experimentation

## Typical Usage

```
devctl up <branch> <feature>
```

This creates a worktree under `work/<branch>/` for isolated development.

## Important Notes

* The contents of this directory are **temporary**.
* Worktrees can be safely removed when development tasks are completed.
* This directory is **excluded from version control** via `.gitignore`.
```

#### `.gitkeep` ファイル

すべて **空ファイル**（0 bytes）。空ディレクトリを Git に認識させるためのもの。

対象:
- `prompts/phases/000-foundation/ideas/.gitkeep`
- `prompts/phases/000-foundation/plans/.gitkeep`
- `prompts/rules/.gitkeep`
- `scripts/.gitkeep`

---

## ロケールオーバーレイ: `templates/project-default/locale.ja/`

`base/` のすべての README.md を日本語版で上書き。
`.gitkeep` やディレクトリ構造ファイルはオーバーレイ不要（言語に依存しないため）。

```
templates/project-default/locale.ja/
├── features/
│   └── README.md
├── prompts/
│   └── phases/
│       └── README.md
├── shared/
│   ├── README.md
│   └── libs/
│       └── README.md
└── work/
    └── README.md
```

### ファイル内容

#### `features/README.md`

```markdown
# Features

このディレクトリには、プロジェクトの各機能（feature）が格納されます。

各 feature は独立したモジュールで、コードベース・テスト・設定を持ちます。

## Feature の追加

`devctl scaffold features <template>` で新しい feature 構成を生成します。

## ディレクトリ規約

```
features/
  <feature-name>/
    cmd/           # CLI エントリポイント
    internal/      # 内部パッケージ
    go.mod         # Go モジュール定義
```
```

#### `prompts/phases/README.md`

```markdown
# Phases

このディレクトリは、プロジェクトのフェーズごとに仕様書と計画を管理します。

## 構成

```
phases/
  000-foundation/     # Phase 0: 基盤構築
    ideas/            # 仕様書
    plans/            # 実装計画
  001-<next-phase>/   # Phase 1: ...
    ideas/
    plans/
```

## ワークフロー

1. `ideas/<ブランチ名>/` に仕様書を作成
2. 仕様書をレビュー・承認
3. `plans/<ブランチ名>/` に実装計画を作成
4. 実装計画をレビュー・承認
5. 計画に基づいて実装を実行
```

#### `shared/README.md`

```markdown
# Shared

このディレクトリには、複数の feature にまたがって共有されるコードやリソースを格納します。

## 構成

```
shared/
  libs/       # 共有ライブラリ
```
```

#### `shared/libs/README.md`

```markdown
# 共有ライブラリ

feature 間で再利用可能なライブラリを格納します。

言語別にパッケージを配置してください:

```
libs/
  go/         # 共有 Go パッケージ
  python/     # 共有 Python モジュール
```
```

#### `work/README.md`

```markdown
# work

このディレクトリは、開発用の一時的な作業ディレクトリ・ワークツリーのために予約されています。

主な用途:

* Git ワークツリー
* 複数機能の並行開発
* AI エージェントセッション
* 一時的な実験

## 使い方

```
devctl up <branch> <feature>
```

`work/<branch>/` にワークツリーが作成され、隔離された開発環境が提供されます。

## 注意事項

* このディレクトリの内容は**一時的**です。
* 開発タスク完了後、ワークツリーは安全に削除できます。
* `.gitignore` によりバージョン管理から除外されます。
```

---

## 適用結果

`devctl scaffold` 実行後のプロジェクト構成:

```
(リポジトリルート)/
├── features/
│   └── README.md
├── prompts/
│   ├── phases/
│   │   ├── README.md
│   │   └── 000-foundation/
│   │       ├── ideas/.gitkeep
│   │       └── plans/.gitkeep
│   └── rules/.gitkeep
├── scripts/.gitkeep
├── shared/
│   ├── README.md
│   └── libs/
│       └── README.md
├── work/
│   └── README.md
└── .gitignore            # "work/*" が追記される
```
