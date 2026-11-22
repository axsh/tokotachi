---
apiVersion: agent.meta/v1
kind: capability
id: acs-external-agent-sample
uid: c2b45d98-a12e-46f9-a8b1-a26a6cc231a2
title: ACS Sample (external-agent)
description: >-
  Agent Config Set の見本。baseline ではなく external-agent タグのみを持つ。
  エディタ外エージェント向けにデプロイする例。
body: inline
tags:
  - external-agent

---

# ACS Sample (external-agent)

この capability は **ACS（Agent Config Set）** のサンプルである。

- 単一の `prompts/manifest/` ツリーに置き、フォルダを A/B/C に複製しない
- `tags: [external-agent]` のみ（暗黙の `baseline` は持たない）
- 既定の `tt prompt update`（`--tags` 省略 = baseline）では **emit されない**
- 取り込むとき: `tt prompt update --tags external-agent` または `tt prompt update --tags baseline,external-agent`

タグ一覧の確認: `tt prompt tags`
