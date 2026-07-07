---
apiVersion: agent.meta/v1
id: Fix-Workflow-Frontmatter-In-PromptUpdate
kind: specification
---

# `tt prompt update --target ag` で出力される workflows に frontmatter (description) を追加する

## 背景 (Background)

`tt prompt update --target ag` を実行した際、`procedures` エンティティは `.agent/workflows/` (または設定されたパス) に Markdown ファイルとして展開されます。
しかし、現在の実装ではこれらのファイルに YAML frontmatter が付与されておらず、特に `description` フィールドが欠落しています。

Antigravity (IDE) は、これらの Markdown ファイルが `name` および `description` を含む frontmatter を持っていることを期待しており、これらが存在しない場合、ワークフローとして認識されない、あるいは説明が表示されないという問題が発生します。

既存の `capability` (Skill) の出力処理では既に frontmatter の生成が行われていますが、`procedure` (Workflow) の出力処理では行われていません。

## 要件 (Requirements)

1.  **Antigravity エミッターの修正**: `antigravity.go` の `procedures` 出力処理において、`capabilities` と同様に `name` および `description` を含む YAML frontmatter を生成し、Markdown ボディの先頭に付与するようにします。
2.  **カタログ側の不備修正**: 多くの `procedure` 定義において `description` フィールドが欠落している、あるいは不適切な位置（2つ目の frontmatter 内など）にあります。これらを整理し、`tt` のマニフェストパーサーが正しく `proc.Raw["description"]` として認識できる位置（最初の frontmatter 内）に配置します。
3.  **スキーマの更新**: `procedure.schema.json` に `description` プロパティを追加し、バリデーションを通るようにします。

## 実現方針 (Implementation Approach)

### 1. `AntigravityEmitter.Emit` の修正

[antigravity.go](file:///c:/Users/yamya/myprog/tokotachi/features/tt/internal/prompt/emitter/antigravity.go) の `Emit Procedures` セクションを修正します。

*   `proc.Raw["description"]` から説明文を取得します。
*   `proc.ID` を `name` とし、取得した説明文を `description` とする `SkillFrontmatter` 構造体（または同様の形式）をマーシャルして frontmatter を作成します。
*   作成した frontmatter を `body` の前に挿入します。

### 2. `procedure.schema.json` の更新

[procedure.schema.json](file:///c:/Users/yamya/myprog/tokotachi/prompts/manifest/schemas/procedure.schema.json) の `properties` に `description` を追加します。

```json
"description": {
  "type": "string",
  "description": "Human-readable description of the procedure."
}
```

### 3. カタログの `procedure` マニフェストの修正

カタログ内の各プロシージャ Markdown ファイルを確認し、`description` を最初の frontmatter に移動または追加します。

対象ディレクトリ: `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/`

## 検証シナリオ (Verification Scenarios)

1.  **再現確認**:
    *   `tt prompt update --target ag` を実行する。
    *   `.agent/workflows/` 配下の Markdown ファイルを確認し、frontmatter が存在しないことを確認する。
2.  **修正後確認**:
    *   コード修正およびカタログ修正を適用する。
    *   `tt prompt update --target ag` を再実行する。
    *   `.agent/workflows/` 配下の Markdown ファイルを確認し、以下の形式の frontmatter が存在することを確認する。
        ```yaml
        ---
        name: create-specification
        description: Create a structured specification document...
        ---
        ```
3.  **Antigravity での認識確認**:
    *   Antigravity IDE (または開発環境) で、これらのワークフローが正しく認識され、説明が表示されることを確認する。

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1.  ビルド＋単体テスト:
    `scripts/process/build.sh`

2.  コンパイラ統合テスト (リグレッション確認):
    `scripts/process/integration_test.sh --categories "common"`
    ※ emitter の挙動変更により既存のテスト期待値が変更になる可能性があるため、必要に応じてテストコードを更新します。
