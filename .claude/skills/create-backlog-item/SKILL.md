---
name: create-backlog-item
description: Create Backlog Item
disable-model-invocation: false
---


# バックログ項目作成ワークフロー (Backlog Item)

このワークフローは、あとで確かめる事項や決める事項を、builtin backlog（知識DB）に 1 件として残す。
正本は Markdown ファイルではない。操作は `tt kdb backlog` のみとする。

## 目的

- 仕様や実装の途中で仮に決めたことを、あとで見直せるようにする。
- 運営者（発注者）など、外の人に確かめる事項を、答えが出るまで失わないようにする。
- 先送りにした決めごとを、いつまでに決めるかと一緒に残す。

## 1. 対象にするもの

| 種類 | 使うとき |
| --- | --- |
| 仮の決定 | 資料や関係者が決めていないことを、仮に決めて先へ進んだ。答えによっては直す |
| 確認事項 | 運営者・社内・外のサービスに確かめないと決められない |
| 先送り | 今は決めず、後の仕様や作業で決める |

次のものは対象にしない。

- その場で決まり、見直す予定の無いこと。仕様書に書けば足りる。
- 作業の TODO。実装計画に書く。

## 2. 制約事項

> [!CAUTION]
> - Markdown の backlog ファイルを作らない。SQLite や corpus を手編集しない。
> - 詳細なフラグ・終了コードはスキルに写さない。必ず `tt help kdb backlog <op>` を読む。
> - 出どころの文書への `backlog:<uuid>` リンクは、出どころを作っているワークフロー側で足す。このワークフローだけを実行したときは、リンクを足すかをユーザーに確かめる。
> - `git push` はユーザーが明示したときだけ行う。
> - corpus を他者と共有するために commit するときは `.claude/skills/systematize-knowledge-database/SKILL.md` を使う。ローカルだけの追加なら任意。

## 3. ステータスと重なりの確認

1.  **ステータスの取得**:
    *   `scripts/utils/show_current_status.sh` を実行し、JSON 出力から `phase`、`branch` を取得する。
    *   採番に `next_backlogs_id` は使わない（`tt kdb backlog add` が次の空き id を決める）。
    *   以下、`[Phase]`、`[Branch]` とする。
2.  **重なりの確認**:
    *   `tt help kdb backlog list` を読む。
    *   `tt kdb backlog list --phase [Phase] --branch [Branch] --match <要約の正規表現>` で同じ事項が無いかを確かめる。
    *   同じ事項があれば新規は作らず、`tt kdb backlog update`（経緯の追記は `--append-history`）で直す。詳細は help。

## 4. 新規作成

1.  `tt help kdb backlog add` を読む。
2.  次を実行する（`--id` は原則省略）:

```bash
tt kdb backlog add \
  --phase [Phase] \
  --branch [Branch] \
  --title "<事項を短く表す見出し>" \
  --kind "<仮の決定|確認事項|先送り>" \
  --ask-to "<確かめる相手>" \
  --due "<いつまでに>" \
  --origin "<出どころ>" \
  --problem "<何が問題か>" \
  --current-handling "<今の扱い>" \
  --options "<選択肢と影響>" \
  --questions "<確かめるときの質問>" \
  --after-decision "<決まったらすること>" \
  --history "<経緯の初行>"
```

3.  任意フラグの文章の書き方（中身の指針）:

| フラグ | 書くこと |
| --- | --- |
| `--problem` | 背景と、何が決まっていないか。資料を開かなくても判断できるように書く |
| `--current-handling` | 仮に決めた内容と理由。先送りなら「決めていない」と、それまでの扱い |
| `--options` | 考えられる答えと、答えごとに変わること |
| `--questions` | 相手に送る質問をそのまま使える文で。不要なら省略可 |
| `--after-decision` | 答えが出たときに直す場所 |
| `--history` | 日付付きの経緯（作成時点） |

4.  成功 JSON の `item.uuid`（または `uuid`）を取り、文書では `backlog:<uuid>` と書く。
5.  `prompts/phases/.../backlogs/` に新しい `.md` が増えていないことを確かめる。

## 5. 答えが出たとき

1.  `tt help kdb backlog resolve` を読む。
2.  `tt kdb backlog resolve --phase [Phase] --branch [Branch] --id <NNN> --state 決着|取り下げ`（必要なら `--decision`）。
3.  部分更新や経緯の追記だけなら `tt kdb backlog update`（状態を `決着` / `取り下げ` にするのは `resolve`）。

## 6. 完了

1.  corpus を共有するなら `.claude/skills/systematize-knowledge-database/SKILL.md` のあと、呼び出し元の成果物と一緒に commit してよい。
2.  ユーザーには `backlog:<uuid>` をインラインコードで示す（`.claude/skills/portable-file-references/SKILL.md` の知識DB参照）。
