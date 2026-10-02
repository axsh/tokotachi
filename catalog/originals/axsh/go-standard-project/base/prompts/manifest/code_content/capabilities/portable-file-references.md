---
apiVersion: agent.meta/v1
kind: capability
id: portable-file-references
title: Markdown 内ファイル参照のポータビリティ規約
description: >-
  Markdown ドキュメントおよびチャット提示で、ファイル参照を
  ワークスペースルート（プロジェクトホーム）からの相対パスで記述し、
  file:// の単純付与や環境依存の絶対パスを禁止する規約。
paths:
  - "**/*.md"
manual_only: false
body: inline
---

# Markdown 内ファイル参照のポータビリティ規約

このプロジェクトの Markdown ドキュメントは、Git リポジトリを通じて複数の開発者・複数の環境
(Windows, macOS, Linux, CI)で共有される。ドキュメント内のファイル参照が特定の環境に
依存していると、他の開発者がドキュメントを読む際にパスが無効になり、情報の追跡が困難になる。

この capability は、リポジトリ内 Markdown とエージェントのチャット提示の両方で、
ポータブルなファイル参照形式を定義する。

## 規約

### 禁止: 絶対パスおよび file:// スキーム

以下は使用してはならない（リポジトリ内 Markdown・チャット応答の両方）。

```markdown
<!-- 禁止: 絶対パス + file:// -->
[仕様書](file:///c:/Users/yamya/myprog/vv5/work/feat-minimum-vv/prompts/phases/refs/drafts/014-spec.md)

<!-- 禁止: 相対パスの先頭に file:// を付けただけ（ホスト名誤解釈） -->
[レポート](file://tmp/autoeval/card/analysis/analysis-report.md)
[調査](file://prompts/phases/000-foundation/branches/x/investigations/012-Foo.md)

<!-- 禁止: OS 絶対パス -->
`c:\Users\yamya\myprog\vv5\work\feat-minimum-vv\features\chord\internal\store\db.go`
```

絶対パスや絶対 `file://` URL は作成者のローカル環境でしか通用しない。
`file://相対パス` は URL 上「ホスト=先頭セグメント」となり、ユーザホーム配下などを誤参照する。

### 推奨: ワークスペース相対パス（プロジェクトホーム基準）

ワークスペースルート（リポジトリルート＝プロジェクトホーム）からの相対パスを使う。

**開かせる・クリックさせる提示は Markdown 相対リンク一択**（`file://` なし）。
バッククォートの相対パスはインラインコードであり、通常はリンクにならない。

```markdown
<!-- 推奨（クリック可能）: Markdown 相対リンク -->
[analysis-report.md](tmp/autoeval/<cardId>/analysis/analysis-report.md)
[012-Foo.md](prompts/phases/000-foundation/branches/x/investigations/012-Foo.md)

<!-- 文書本文でパスを識別子として書くだけならバッククォート可（リンクにはならない） -->
実装は `features/example/src/main/java/com/example/repository/UserRepository.java` を参照。
```

### 表や箇条書き内での記述

表の中や箇条書きでも、同じ形式を使用する。

```markdown
| 先行仕様書 | 関係 |
| :--- | :--- |
| `prompts/phases/000-foundation/branches/feat-minimum-vv/ideas/032-subgraph-actor-generalization.md` | 子 Actor モデルは本仕様で継続利用 |

- 先行仕様: `prompts/phases/000-foundation/branches/feat-minimum-vv/ideas/032-subgraph-actor-generalization.md`
- スキーマ定義: `prompts/manifest/schemas/capability.schema.json`
```

### 行番号付きの参照

特定の行範囲を示したい場合は、パスの後に行番号情報をテキストで付記する。

```markdown
`features/auth/src/main/java/com/example/auth/controller/AuthController.java` (L10-L45) の認証処理を参照。
```

### パス区切り文字

パス区切り文字にはスラッシュ (`/`) を使用する。
バックスラッシュ (`\`) は使用しないこと。
スラッシュであれば Windows / macOS / Linux のいずれでも正しく解釈できる。

### Markdown リンク記法の使い分け

| 対象 | 形式 | 例 |
| :--- | :--- | :--- |
| リポジトリ内のファイルを**開かせる**（チャット・提示） | Markdown **相対**リンク（`file://` なし）一択 | `[014-spec.md](prompts/phases/refs/drafts/014-spec.md)` |
| リポジトリ内のパスを**識別子として書くだけ**（表・箇条書き） | バッククォート可（リンクにはならない） | `` `prompts/phases/refs/drafts/014-spec.md` `` |
| 外部 URL (HTTP/HTTPS) | Markdown リンク記法 | `[Spring Boot 公式ドキュメント](https://spring.io/projects/spring-boot)` |

リポジトリ内パスへの `[text](url)` では、`url` に必ずワークスペース相対パスを書く。`file://` は付けない。
チャットでファイルを示すときはバッククォートだけにせず、必ず相対リンクにする。

## 適用範囲

この規約は次に適用する。

- Git リポジトリに記録される全てのマークダウンファイル
- エージェントがチャット上でファイルを提示するとき

以下はこの規約の適用外とする:

- CI/CD スクリプト内のパス指定
- コード内のファイルパス定数
