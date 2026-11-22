---
apiVersion: agent.meta/v1
kind: procedure
id: systematize-knowledge-database
uid: 37ed65ab-529c-4b4f-9810-0c8e4f07077f
title: Systematize Knowledge Database
trigger:
  command: systematize-knowledge-database
description: >-
  Distill a knowledge-DB cache, dump to tmp, simple-merge into data/, pack to
  corpus/ under the configured kdb-root (default shared/kdb) so the repository
  holds the shared source of truth (symmetric to systematize-far-knowledge).
tags:
  - baseline
---

# Systematize Knowledge Database

遠方知識の systematize に相当する、知識DB向けの退避手順である。蓄積は作業中の各知識 CLI で行う。この手順は、矛盾の整理（蒸留）と、Git 正本（corpus）への退避を行う。

操作入口は **`tt kdb-track` / `tt kdb <id>`** である（既定の kdb-root は `shared/kdb`。上書きは `--registry` / `KDB_REGISTRY` / `--kdb-root` / `KDB_ROOT`）。Python や外部 `sqlite3` CLI は不要。未移植の知識は registry の `*_cmd` へ委譲する。操作の詳細は `tt help kdb-track` / `tt help kdb` を読む。

## Step 1: 未登録の確認

```bash
tt kdb-track unregistered
```

`ids` が空でなければ `WARN: unregistered corpus: ...` として報告する（新規知識DBの登録漏れの気づき）。空なら続行する。

## Step 2: 状態確認

```bash
tt kdb-track status-all
```

登録済みの status が集約される（backlog は builtin、他は `status_cmd`）。すべて `dirty` が false で、当該 `corpus_path` にも git 差分が無ければ、退避は不要である。その旨を報告して終了してよい。

## Step 3: 退避

```bash
tt kdb-track systematize
```

特定の知識だけなら:

```bash
tt kdb-track systematize --id backlog
```

`tt` は registry に従い、要退避のエントリだけ distill → dump → merge → pack を実行する。重複の畳み込みが要る知識は distill 側で行う。共通の単純マージでは畳まない。成功時、先頭行の JSON に `add_candidates` があり、続けて `ADD_CANDIDATE: <path>` 行も出ることがある。

## Step 4: コミット対象にする

`add_candidates` / `ADD_CANDIDATE:` に出た `corpus_path`（および必要な `{kdb-root}/<kdb-id>/corpus/`）を `git add` し、知識の共有コミットに含める。`data/` 配下は gitignore のためコミットしない。エージェント向けスキルの配り（`tt prompt update`）は、スキル本文を変えたときだけ行う。

## Step 5: 任意の確認

別環境や空キャッシュで、当該知識の unpack → restore → status を呼び、`dirty` が false で件数が一致することを確かめてよい（backlog なら `tt kdb backlog …`、他は `tt help kdb`）。

## Interaction Rules

- PR 作成や pull の前に、`pre-sync-knowledge-compile` からこの手順を要求されたら、確認を待たずに実行する。
- 遠方知識の pending 体系化（`systematize-far-knowledge`）とは別である。両方必要なら、どちらも完了してから sync する。
