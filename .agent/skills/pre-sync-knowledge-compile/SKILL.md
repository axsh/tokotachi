---
name: pre-sync-knowledge-compile
description: Before creating a Pull Request or pulling from a remote/local branch, ensure all pending intake events have been systematized and compiled into agent-readable skills via the systematize-far-knowledge workflow, and that tracked knowledge-DB caches are not left dirty.
paths:
    - prompts/memory/**
    - .agents/skills/**
disable-model-invocation: false
---


# Pre-Sync Knowledge Compile

You are an AI coding agent working in this repository.

Before performing any of the following **sync operations**, you **MUST** check
for pending intake events and systematize them into compiled knowledge:

## Trigger Conditions

This capability is triggered when the agent is about to perform any of:

1. **Pull Request (PR) 作成**: `gh pr create`, GitHub PR の作成操作
2. **Pull**: `git pull`, `git fetch && git merge`, `git rebase` (リモートからの取得)
3. **Branch Switch with Pull**: `git checkout <remote-branch>`, `git switch` (リモート追跡ブランチへの切替)

## Step 1: Pending Events の確認

```bash
./scripts/code/agent/intake.sh list --status pending
```

pending events が **0 件** の場合、以下を報告する:

```text
Knowledge compile: no pending events.
```

pending events が **1 件以上** ある場合、sync 操作の **前に** 以下を実行する:

1. `systematize-far-knowledge` ワークフローを実行する
   - 全 pending events のカテゴリ化・登録
   - processed への移行
   - スキル化 (capability ファイルの生成)
   - `prompt compile --apply` によるデプロイ
2. 生成されたファイルをコミットする

## Step 1b: 知識DB（追跡一覧）の未退避確認

遠方知識に加え、registry（既定 `shared/kdb/registry.yaml`）に載っている知識DBキャッシュに未退避（dirty）が無いかを確認する:

```bash
tt kdb-track status-all
```

- いずれかの Entry で `dirty` が true の場合、sync の **前に** `systematize-knowledge-database`（実体は `tt kdb-track systematize`）を実行し、当該 `corpus_path` を更新してコミットする。
- ある知識DBに接続できない場合は、その旨を報告する。作業ツリーに当該 corpus の未コミット差分があればコミットを促し、接続不能を理由に dirty チェックを黙殺して PR しない。
- 未登録の corpus があれば WARN を報告する（`tt kdb-track unregistered`。新規KDBの登録漏れの気づき）。
- 全 Entry で `dirty` が false で corpus 差分も無ければ、次へ進む。

```text
Knowledge DB: tracked kdbs not dirty (or systematized).
```

## Step 2: Sync 操作の続行

遠方知識の pending と知識DBの未退避が片付いたら、元の sync 操作を続行する:

- **PR 作成の場合**: コミット・push 後に PR を作成
- **Pull の場合**: pull/fetch を実行

## 例外

以下の場合はこのチェックをスキップしてよい:

- ユーザーが明示的に「スキップしてよい」と指示した場合
- emergency fix で時間的制約がある場合 (ただしその旨を報告すること)
- pending events が全て自分の作業とは無関係なブランチのものである場合

## Interaction Rules

- pending events がある場合、ユーザーに確認せず自動的に systematize-far-knowledge を実行する
- systematize-far-knowledge の結果はユーザーに報告する
- sync 操作を忘れないこと (本来の操作を完遂する)
