---
apiVersion: agent.meta/v1
id: verify-and-specify-next
uid: 80714fc7-114c-401f-a053-1b7ff839a34d
kind: procedure
title: Verify and Specify Next
trigger:
    command: verify-and-specify-next
description: >-
  Read a target specification and a given document (implementation report,
  investigation, or similar), verify independently, then write a fix
  specification or the next thing to build via create-specification.
tags:
  - baseline
---

# 確かめて次の仕様を書くワークフロー (Verify and Specify Next)

このワークフローは、実現しようとしていた仕様と、与えられたドキュメント（実装の再現手順の報告 `_runs/procedures/NNN`、調査の報告 `investigations/NNN` など）を熟読し、実装を自分で確かめて、次の仕様を 1 つ書きます。報告が `_runs/` 配下にあるときはローカル検証ログ（gitignore）として読む。次仕様（`ideas/`）の参照欄にはそれらへの Markdown リンクを書かず、番号とファイル名を平文で書く（{{capability:committed-doc-links}}）。

- 問題があれば、**直しの仕様**を書きます。
- 問題が無ければ、**次に実装すべきもの**を提案し、その仕様を書きます。

どちらの場合も、仕様は `{{procedure:create-specification}}` の手順と形で書きます。

> [!IMPORTANT]
> このワークフローの成果は、仕様（と、そのバックログ）だけです。実装計画を作らず、実装もしません。
> 確かめのあいだは、製品のコードと、ほかの人のファイルを変えません。破壊的な試しが必要なときは `tmp/` 配下の git worktree の中だけに限ります。
> ドキュメントの「一致」「通った」を、そのまま信じません。自分で流し、読み、見て確かめます。

## 1. 準備

1. **ステータス**: `scripts/utils/show_current_status.sh` を実行し、`phase`・`branch`・`next_idea_id` を控えます。`next_backlogs_id` が出ても採番には使わない（backlog は `tt kdb backlog add`）。
2. **入力を決める**:
   - 与えられたドキュメント（ユーザーが示したもの）。
   - 実現しようとしていた仕様。ドキュメントの「元になる仕様・計画」の欄、同じ番号の `ideas/`、調査の範囲から決めます。分からなければ、ユーザーに確かめます。
   - 関わる計画（`plans/`）、ローカル報告（`_runs/`）、調査の報告、バックログ（`tt kdb backlog list` で読み、文書では `backlog:<uuid>` と参照する）。
3. **作業ツリーの様子**: `git status --porcelain` で、ほかの人の作業中の変更を控えます。それらには触れません。
4. **従う決まり**: `{{policy:testing-rules}}`、`{{policy:coding-rules}}`。

## 2. 熟読する

次を読み、仕様の要件ごとに「何を根拠に満たしたと言っているか」の表を作ります。

| 読むもの | 見るところ |
| --- | --- |
| 仕様 | 必須と任意、この仕様で決めること、作らないもの、検証シナリオ、テスト項目、ビルド・全体検証 |
| 計画 | 先に書く試験、レッド記録、総合判定 |
| 報告 | 確認すること、試験の件数、計画・仕様との突き合わせ、備考 |
| 調査の報告 | 直接の原因、根拠、推奨 |

ドキュメントが自分で挙げている差や不足は、判定のとき 1 つずつ扱います。

## 3. 自分で確かめる

1. 仕様の「ビルド・全体検証」に書かれたコマンドを、影響範囲に応じて実行します（標準は `scripts/process/build.sh` と、必要なカテゴリに絞った `scripts/process/integration_test.sh`）。
2. 件数や合否を、ドキュメントの記載と比べます。
3. 要件ごとに要のコードを読みます。必要なら読み取り専用で状態を確認します（書き込みを残さない）。
4. 試験の効き目が疑わしいときは、`tmp/` 配下の worktree で壊す試しをしてよいが、元の作業ツリーは汚しません。

## 4. 判定する

要件ごとに、満たしている / 問題がある / 判断できない、を決めます。問題は直しの仕様の材料にし、満たしていれば次に作るものの候補にします。

## 5. 仕様を書く

1. `{{procedure:create-specification}}` に従い、直しの仕様または次の仕様を 1 つ書きます。
2. 仮の決定や確認事項は `{{procedure:create-backlog-item}}` で残し、`backlog:<uuid>` で参照します。
3. `_runs/` への Markdown リンクは書きません（{{capability:committed-doc-links}}）。

## 6. 完了

1. 仕様書をコミットする（create-specification の手順に従う）。
2. 実装計画や実装には進まない。ユーザーの指示を待つ。
