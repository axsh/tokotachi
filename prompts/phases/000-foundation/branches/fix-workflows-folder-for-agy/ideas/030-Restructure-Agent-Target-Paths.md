# 030-Agentターゲットパスの再構成とAGワークフローの復元

## 背景 (Background)

現在、`tt prompt update` によるターゲット出力において、以下の2つの課題が発生しています。

1.  **Antigravity (AG) における `workflows` フォルダの消失**:
    実装計画 `007-Unify-Skills-Output.md` により、AGターゲットの `workflows` 出力が廃止され、`procedures` が `skills` に統合されました。しかし、AGは `Workflows` を決定性の高いプロンプトとして重視しており、これらを `workflows` フォルダに独立して展開することが望ましい状況です。
2.  **ターゲットディレクトリ名の競合と曖昧さ**:
    AGとCodexの両方が `.agents/` という共通のフォルダを使用していますが、これらを明確に区別し、AG用は `.agent/`、Codex用は `.codex/` とすることで、各エージェント専用の環境を構築する必要があります。

## 要件 (Requirements)

1.  **Antigravity ターゲットの変更**:
    *   デフォルトのベースディレクトリを `.agents/` から `.agent/` に変更する。
    *   `workflows` フォルダ出力を復元し、`procedures` ( `prompts/templates/procedures` 以下) をこのフォルダに展開する。
    *   `workflows` フォルダ内のファイルは `{id}.md` 形式とする（`skills` のようにサブフォルダ＋`SKILL.md` にしない）。
2.  **Codex ターゲットの変更**:
    *   デフォルトのベースディレクトリを `.agents/` から `.codex/` に変更する。
3.  **テンプレート変数の解決**:
    *   `{{procedure:id}}` テンプレート変数が、ターゲット設定に応じた正しいパス（AGなら `.agent/workflows/id.md`、その他なら `.agents/skills/id/SKILL.md` 等）に解決されるようにする。
4.  **互換性と設定**:
    *   ターゲット定義ファイル (`antigravity.yaml`, `codex.yaml`) の `paths` 設定を新しいデフォルト値に更新する。
    *   既存のコード内で `.agents/` がハードコードされている箇所を、各ターゲットのデフォルト値に合わせる。

## 実現方針 (Implementation Approach)

### 1. Emitter 構造体と共通型の更新 (`features/tt/internal/prompt/emitter/`)

*   **`template.go`**:
    *   `TargetPaths` 構造体に `Workflows` フィールドを追加。
    *   `resolveRef` 関数の `procedure` ケースを更新し、`ctx.Paths.Workflows` が存在する場合はそこを、存在しない場合は `ctx.Paths.Skills` を参照するように変更。
*   **`antigravity.go`**:
    *   `resolvePaths` において、デフォルト値を `.agent/rules/`, `.agent/skills/` に変更し、`.agent/workflows/` を追加。
    *   `Emit` 関数において、`inc.Procedure` が真の場合、`procedures` を `workflowsDir` に `{id}.md` として出力するロジックを実装。
    *   `resolveTargetPaths` において `Workflows` パスを解決するように修正。
*   **`codex.go`**:
    *   `resolvePaths` において、デフォルト値を `.codex/rules/`, `.codex/skills/` に変更。
    *   `generateMarkerContent` 内の要約テキストや変数定義に含まれる `.agents/` を `.codex/` に修正。

### 2. ターゲットマニフェストの更新 (`prompts/manifest/targets/`)

*   **`antigravity.yaml`**:
    *   `paths.rules` を `.agent/rules/` に。
    *   `paths.skills` を `.agent/skills/` に。
    *   `paths.workflows` を `.agent/workflows/` として新規追加。
*   **`codex.yaml`**:
    *   `paths.rules` を `.codex/rules/` に。
    *   `paths.skills` を `.codex/skills/` に。
    *   `index_file` を `CODEX.md` に変更（検討事項）。

## 検証シナリオ (Verification Scenarios)

1.  **Antigravity ターゲットの確認**:
    *   (1) `tt prompt update --target ag` を実行。
    *   (2) `.agent/workflows/` ディレクトリが作成され、中に `procedures` が `.md` ファイルとして存在することを確認。
    *   (3) `.agent/rules/` および `.agent/skills/` が作成されていることを確認。
    *   (4) 生成されたファイル内のテンプレート変数 `{{procedure:xxx}}` が `.agent/workflows/xxx.md` に解決されていることを確認。
2.  **Codex ターゲットの確認**:
    *   (1) `tt prompt update --target codex` を実行。
    *   (2) `.codex/rules/` および `.codex/skills/` が作成されていることを確認（`.agents/` が作成されないこと）。
    *   (3) インデックスファイル（`AGENTS.md` または `CODEX.md`）内のリンクが `.codex/` を指していることを確認。
3.  **既存ターゲットへの影響確認**:
    *   (1) `cursor` や `claude-code` ターゲットで `tt prompt update` を実行。
    *   (2) 従来のパス（`.cursor/`, `.claude/`）に正しく出力され、後退がないことを確認。

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1.  ビルド＋単体テスト:
    `& "C:\Program Files\Git\bin\bash.exe" -c "scripts/process/build.sh"`

2.  プロンプトコンパイラ統合テスト（パス解決とエミットの検証）:
    `& "C:\Program Files\Git\bin\bash.exe" -c "scripts/process/integration_test.sh --categories \"template\""`

3.  ドリフトチェックの検証:
    `& "C:\Program Files\Git\bin\bash.exe" -c "tt prompt update --target ag --check"`
    `& "C:\Program Files\Git\bin\bash.exe" -c "tt prompt update --target codex --check"`
