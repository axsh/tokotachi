---
name: scaffold-manifest-update
description: tt scaffold の --update で上流カタログの prompts/manifest を取り込み、 instruction.md と doc-relations-NNN.md に従ってローカルへマージする手順。
disable-model-invocation: false
---


# Scaffold Manifest Update（上流マニフェストの取り込みとマージ）

上流スキャフォールドの `prompts/manifest/` を、ローカル編集を壊さずに取り込むときの手順。

ローカルテンプレートを編集して各エージェントへデプロイする手順は `{{capability:prompt-manifest-update}}` を参照。本スキルは **上流取り込み** の正本である。

## 1. いつ使うか

- カタログ側のプロンプトが更新され、プロジェクトへ差分を取り込みたいとき
- `tt scaffold` がダウンロード履歴でスキップし、単純な再実行では内容が変わらないとき
- `--force` での破壊的再適用をしたくないとき

## 2. 実行前の確認（必須）

`tt scaffold` の位置引数（`<category> <name>`）や追加フラグはプロジェクトごとに異なる。
**エージェントは、次をユーザに問い合わせてから実行すること。推測や固定例の流用でコマンドを組み立ててはならない。**

確認する項目の例:

- `<category> <name>`（例としてあり得る形: `project some-scaffold`。実際の値はユーザ指定）
- 作業ディレクトリ（`--update` の値。省略時はデフォルト）
- 必要なら `--branch` / `--repo` / `--yes` など追加フラグ

ユーザの回答をそのまま使い、次の形で実行する:

```bash
tt scaffold <category> <name> --update
# または作業ディレクトリを指定
tt scaffold <category> <name> --update <workdir>
# ユーザが指定した追加フラグがあればそれも付ける
```

- `--update` は履歴があっても最新を取得する（スキップしない）
- `--force` との併用は不可
- workdir 省略時は `tmp/scaffold-update/<category>-<name>/`

## 3. ワークスペースレイアウト

| パス | 意味 |
| :--- | :--- |
| `workdir/backup/` | 開始時点のローカル `prompts/manifest/` |
| `workdir/latest/` | 上流から展開した最新マニフェスト |
| `workdir/instruction.md` | AI 向けマージ指示（英語方針を含む） |
| `workdir/doc-relations-NNN.md` | 連番の対応表（1 件でも `001` から。連番なしの `doc-relations.md` は無い） |

`tt` の標準出力にも `instruction.md` と同じ指示と、保存先パスが出力される。

## 4. クリーン更新と AI マージ

- **クリーン**: backup と latest が全ファイル一致 → ローカル `prompts/manifest/` を latest に揃える（AI マージ不要）
- **それ以外**: ローカルは自動では壊さない。`instruction.md` と `doc-relations-NNN.md` に従ってエージェントが `prompts/manifest/` へマージする

## 5. マージ方針（instruction.md と同一）

1. Do not discard or rewrite project-specific information without cause.
2. Prefer incorporating improvements and convenience from `latest/`.
3. Do not break existing references (`uses_capabilities`, template refs, relative links, skill ids).
4. Keep current behavior working while absorbing upstream improvements.
5. Use `doc-relations-NNN.md` (zero-padded sequence starting at 001) as mechanical hints; do not invent renames that contradict uid matches.
6. After merging into the real `prompts/manifest/`, leave the workdir intact for review unless the user asks to delete it.

## 6. doc-relations の読み方

- `uid` 一致: ファイル名が違っても同一ドキュメント
- `uid` 無し／不一致: similar top 3 はヒントのみ（自動マージの唯一根拠にしない）
- `__far-knowledge-` 接頭の文書は主対象外

## 7. マージ後

- 必要なら `{{capability:prompt-manifest-update}}` に従い `tt prompt update --target <target>` で各エージェントへ反映する
- ACS で集合を切り替える場合は `tt prompt tags` でタグを確認し、`tt prompt update --tags <...>` を使う
