---
name: committed-doc-links
description: リポジトリにコミットする Markdown から、ワークスペース相対パスへの Markdown リンクを書くとき、リンク先は git 追跡済みでなければならない。 gitignore 対象・未追跡・欠落へのリンクを禁止する（_runs/・tmp/ など）。
paths:
    - prompts/phases/**/*.md
    - '**/*.md'
disable-model-invocation: false
---


# コミット文書のリンク境界

GitHub 上や他環境で開かれる **コミット対象の Markdown** は、クリック可能なリンクの先が
リポジトリに含まれるファイルでなければならない。`.gitignore` 対象や未追跡の成果物へ
リンクすると、レビュー画面で 404 になり、文書グラフが壊れる。

ポータブルなパス表記そのものは .agent/skills/portable-file-references/SKILL.md を使う。
本 capability は「相対でも、追跡されていない先へはリンクしない」を定める。

## 原則

> リポジトリにコミットする Markdown から、ワークスペース相対パスへの Markdown リンクを書くとき、
> リンク先が **`.gitignore` 対象であってはならない**。
> ローカル専用（`_runs/`・`tmp/` など）は平文（バッククォートや「報告 054」）で引用する。
> まだ作っていないパスへのリンク（計画での提案）は、ignore 対象でなければ許容する。

## 文脈の切り分け

| 文脈 | ignore / 未追跡パスへの Markdown リンク |
| --- | --- |
| リポジトリ内のコミット対象 MD | **禁止** |
| エージェントのチャット応答 | 可（人がローカルで開く案内）。「リポジトリには含まれない」と添える |
| ローカル専用ツリーどうし（例: `_runs/` 内） | 可（そもそもコミットしない） |

## 代表例: `_runs/`

ブランチ文書ツリー直下の `_runs/`（`procedures` / `e2e-tests` / `screenshots`）は
ローカル検証ログであり gitignore される。

- コミット対象（`ideas/` / `plans/` / `investigations/`）から `_runs/` へリンクしない。バックログは知識 DB の項目（`backlog:<uuid>`）で、コミット対象のマークダウンのフォルダではない。
- 引用例: `確かめた報告: 054-Clock-Gate-...（ローカル _runs/procedures。リポジトリには含めない）`
- `_runs/procedures` から `_runs/e2e-tests`・`_runs/screenshots` への相対リンクは可

マーカー規約: `prompts/phases/*/branches/*/` 直下で `_` で始まるディレクトリはローカル作業ログであり、コミットしない。

## 機械的な見張り

repo 試験 `TestTrackedMarkdownLinksAvoidIgnoredTargets` が、追跡中 Markdown のリポジトリ相対リンク先を
`git check-ignore` で確かめ、ignore 対象なら落とす。パス名のブラックリストではなく、本原則と同じ判定である。

## チャットでの案内

ローカル専用パスを人に開かせるときは相対リンクでよいが、「リポジトリには含まれない」と添える。
コミットする文書にはそのリンクを書かない。
