---
apiVersion: agent.meta/v1
id: investigate
kind: procedure
title: Investigate
trigger:
    command: investigate
description: Investigate a codebase, bug, or technical question and produce a structured investigation report under investigations/.
---

# 調査ワークフロー (Investigation)

このワークフローは、コードベースの調査・分析を行い、その結果をレポートとしてまとめます。
**コードの修正や変更は一切行いません。**

調査レポートの正本出力先は、仕様書の `ideas/` と同様に
`prompts/phases/.../branches/.../investigations/` とする

## 目的

- バグの原因特定、設計の理解、依存関係の把握などの調査を体系的に実施する
- 調査結果を構造化されたレポートとしてまとめ、ユーザーに報告する
- コードベースに影響を与えず、読み取り専用で作業する（レポート文書の配置・コミットは例外）

## 制約事項

> [!CAUTION]
> - ソースコードの変更（追加・編集・削除）は**絶対に行わない**こと
> - `git push` などのリモート変更操作は**禁止**（調査レポートのローカル `git commit` は §5 で実施）
> - ビルドスクリプトや生成系コマンドの実行は**禁止**（ただし調査目的の読み取り系コマンドは可）

### 許可される操作

- ファイルの閲覧・検索（grep, find, cat, head, tail 等）
- `git log`, `git blame`, `git diff`, `git show` 等の読み取り系Gitコマンド
- 既存のバイナリやコマンドの実行（現状把握のため。例: `--version`, `--help`, ステータス確認系）
- ログファイルの閲覧・分析
- `javadoc`, `mvn -q dependency:tree` 等の読み取り系 Java/Maven ツール
- 中間ファイルの出力（`tmp/` ディレクトリ以下のみ）
- **調査レポート**の作成・保存（`prompts/phases/.../investigations/` 以下のみ）と、そのファイルの `git commit`

### 禁止される操作

- ソースコード（`.java`, `.go`, `.xml` 等）の作成・編集・削除
- 設定ファイル（`.json`, `.yaml`, `.toml` 等）の作成・編集・削除（本ワークフローで明示した調査レポート以外）
- 仕様書・実装計画書の作成・編集（`ideas/` / `plans/` 等。別ワークフロー専用）
- `mvn package`, `./gradlew build` 等のビルドコマンド（直接実行禁止。スクリプト経由のみ）
- テストの実行（`mvn test`, `scripts/process/build.sh`, `scripts/process/integration_test.sh` 等）

## 1. 準備: ステータスとコンテキストの確認

1.  **ステータスの取得**:
    *   `scripts/utils/show_current_status.sh` を実行します。
    *   JSON出力から `phase`, `branch` を取得します。
    *   調査レポート用の連番は `next_investigations_id` を使います
        （`show_current_status.sh` はブランチ配下の各サブディレクトリ名に対し
        `next_<dirname>_id` を動的に出力する。`investigations/` が未作成だと
        キーが無いので、§2 でディレクトリを作ったあと再実行してよい）。
    *   以下、`[Phase]`, `[Branch]`, `[NextID]` とします（`[NextID]` =
        `next_investigations_id`。無い場合は `000`）。

2.  **調査対象の確認**:
    *   ユーザーの依頼内容を正確に把握する。
    *   調査の目的（バグの原因特定、設計理解、パフォーマンス分析など）を明確にする。
    *   不明点がある場合は、調査前にユーザーに確認する。

3.  **調査スコープの決定**:
    *   調査対象のファイル、モジュール、機能を特定する。
    *   必要に応じて、関連する仕様書（`prompts/phases/` 配下の `ideas/` 等）や設計ドキュメントを参照する。

## 2. 出力先の決定

1.  **ディレクトリの確定**:
    *   基本パス: `prompts/phases/[Phase]/branches/[Branch]/investigations/`
    *   例: `prompts/phases/000-foundation/branches/feat-all-agents/investigations/`
    *   このディレクトリが存在しない場合は作成します。
    *   作成直後は `scripts/utils/show_current_status.sh` を再実行し、
        `next_investigations_id` を確定します。
2.  **ファイル名の決定**:
    *   形式: `[NextID]-[名前].md`（`[NextID]` は3桁ゼロ埋め。例: `000`, `001`）
    *   `[名前]` 部分は、調査内容を適切に表現する簡潔な名称を使用します
        （例: `AgentVm-Launch-Latency`, `KanbanGui-Comment-LiveDelivery`）。
    *   `create-specification` が `ideas/` に `[NextID]-[Name].md` を置くのと同じ規約である。

## 3. 調査の実施

以下の手法を適宜組み合わせて調査を進める:

1. **コード解析**:
    * ソースコードの閲覧と理解
    * 関数・クラスの依存関係のトレース
    * grep/ripgrep によるパターン検索
    * `git blame` / `git log` による変更履歴の追跡

2. **ログ・出力の分析**:
    * アプリケーションログの確認
    * エラーメッセージやスタックトレースの分析
    * 設定値の確認

3. **動的な現状把握**（必要に応じて）:
    * 既存のコマンドやバイナリの実行によるバージョン・ステータス確認
    * 環境変数や設定の確認
    * プロセスやサービスの状態確認

4. **ドキュメント・仕様の参照**:
    * 既存の仕様書（`prompts/phases/` 配下）の確認
    * コード内のコメントや Javadoc の確認
    * README や設計ドキュメントの確認

作業中の下書き・抽出ログは `tmp/` に置いてよいが、**最終レポートの正本は §2 の `investigations/` パス**とする（`tmp/` のみで終えない）。

## 4. レポートの作成

調査結果を §2 で決めたパスにマークダウンで保存する。

### レポートの構成

レポートには以下の項目を含める:

1. **調査概要 (Investigation Summary)**:
    * 調査の目的と背景
    * 調査対象のスコープ

2. **調査手法 (Methodology)**:
    * 実施した調査の手法を簡潔に記載
    * 使用したコマンドや検索パターンなど

3. **調査結果 (Findings)**:
    * 発見した事実を構造化して記載
    * コードスニペット、ログ抜粋、図表などを活用して分かりやすくまとめる
    * ファイルパスへのリンクを含める

4. **分析・考察 (Analysis)**:
    * 調査結果に基づく分析と考察
    * 原因の推定や問題の構造の説明
    * 関連する既知の問題や注意点

5. **推奨事項 (Recommendations)**（該当する場合）:
    * 調査結果を踏まえた対応の提案
    * 優先度の提示
    * **注意**: ここでは方向性や方針の提案のみを行う。実際のコード変更は行わない。

### レポートのフォーマット

- マークダウン形式で記述する
- コードスニペットはコードブロックで記載する
- 必要に応じて Mermaid 図やテーブルを使用する

## 5. 完了確認とコミット

1. **内容のレビュー**:
    * 作成したレポートが、概要・手法・結果・分析をカバーしているか確認します。

2. **ドキュメントの Git コミット**:
    * 作成・更新した調査レポートのみを `git add` → `git commit` してください。
    * コミットメッセージ例: `docs: add investigation XXX-Name`
    * 修正が複数回あった場合は、最終版をまとめてコミットしても構いません。
    ```bash
    git add prompts/phases/[Phase]/branches/[Branch]/investigations/[ファイル名]
    git commit -m 'docs: add investigation XXX-Name'
    ```
    * `git push` はユーザーが明示したときのみ行う。

3. **レポートの提示**:
    * 作成したファイルへのリンクをユーザーに提示します。
      * ワークスペース相対の Markdown リンクを用いること（例: `[XXX-Name.md](prompts/phases/.../investigations/XXX-Name.md)`）。`file://` は付けないこと（`{{capability:portable-file-references}}`）
    * 重要な発見事項がある場合は、レポートとは別にチャットでもハイライトする。

4. **フォローアップ**:
    * ユーザーからの追加質問に対応する。
    * 必要に応じて追加調査を実施し、同一レポートを更新するか、
      新しい `[NextID]` で続報レポートを作成する（更新方針は内容の連続性で判断）。
    * 調査結果を受けて修正が必要な場合は、ユーザーに別ワークフロー
      （`/create-specification` → `/create-implementation-plan` → `/execute-implementation-plan`）
      の利用を提案する。勝手に仕様書・実装計画を作らない。

## 禁止事項（再掲）

- **ソースコードへの一切の変更**
- **仕様書・実装計画書（`ideas/` / `plans/`）の作成・変更**
- **ビルドの実行**
- 調査の範囲を超えた作業の実施
- 調査レポート以外の `prompts/` 改変
