# Knowledge Frontmatter 不一致の修正

## 背景 (Background)

`tt agent knowledge add` コマンドが生成する knowledge ドキュメントの YAML frontmatter と、`tt prompt compile` が期待する frontmatter の間にフィールド名・構造の不一致がある。

この不一致により、`tt agent knowledge add` で正常に作成された knowledge ファイルが `tt prompt compile` のバリデーションで失敗し、プロンプトに反映できない。

### 具体的な不一致

| 観点 | `agent/knowledge` (書き込み側) | `prompt/memory` (読み取り側) |
|------|-------------------------------|------------------------------|
| ID フィールド名 | `knowledge_id` | `id` |
| status フィールド | なし | 必須 (`current`, `target`, `transitional`, `question`, `deprecated`) |
| kind フィールド | なし | 任意 |
| topics フィールド | なし | 任意 |
| triggers フィールド | なし | 任意 |
| depends_on フィールド | なし | 任意 |
| evidence フィールド | なし | 任意 |
| last_reviewed フィールド | なし | 任意 |
| category_path フィールド | あり | なし |
| source_event_ids フィールド | あり | なし |
| created_at フィールド | あり | なし |
| last_updated フィールド | あり | なし |

### 関連するコード

- **書き込み側**: `features/tt/internal/agent/knowledge/types.go` - `KnowledgeFileMeta` 構造体
- **書き込み側**: `features/tt/internal/agent/knowledge/store.go` - frontmatter 生成ロジック
- **読み取り側**: `features/tt/internal/prompt/manifest/types.go` - `MemoryDoc` 構造体
- **読み取り側**: `features/tt/internal/prompt/memory/frontmatter.go` - `ParseFrontmatter` 関数

### 実際のエラー

```
branchpackageinfo-and-slugify.md: ERROR: required field 'id' is missing in frontmatter
```

## 要件 (Requirements)

### 必須要件

1. `tt agent knowledge add` で生成された knowledge ファイルが、変更なしで `tt prompt compile` のバリデーションを通過すること
2. 既存の `prompt/memory` 側の MemoryDoc frontmatter 仕様を正とし、`agent/knowledge` 側を合わせる方向で統一すること
3. `KnowledgeFileMeta` の frontmatter に `id` フィールドを追加し、`knowledge_id` と同じ値を設定すること
4. `KnowledgeFileMeta` の frontmatter に `status` フィールドを追加し、新規作成時はデフォルト値 `current` を設定すること
5. 既存のテストが全て通過すること

### 任意要件

1. `MemoryDoc` 側で `knowledge_id`, `category_path`, `source_event_ids`, `created_at`, `last_updated` など knowledge 固有フィールドも読み取れるようにする（将来の拡張のため）
2. `knowledge_id` フィールドは後方互換性のために残す（`id` と同じ値を持つ）

## 実現方針 (Implementation Approach)

### 方針: `agent/knowledge` 側の出力を `prompt/memory` の期待に合わせる

`MemoryDoc` は既にコンパイルパイプラインで利用されているため、こちらの仕様を正とする。`agent/knowledge` 側のフロンティマター生成を修正する。

### 変更対象

#### 1. `features/tt/internal/agent/knowledge/types.go`

`KnowledgeFileMeta` に以下のフィールドを追加:
- `ID string yaml:"id"` -- `knowledge_id` と同じ値
- `Status string yaml:"status"` -- デフォルト `"current"`

#### 2. `features/tt/internal/agent/knowledge/store.go`

`Add` および `Append` 関数で frontmatter を生成する際に:
- `ID` フィールドに `knowledgeID` の値を設定
- `Status` フィールドにデフォルト値 `"current"` を設定

#### 3. テストファイルの更新

- `features/tt/internal/agent/knowledge/store_test.go`
- `features/tt/internal/agent/knowledge/frontmatter_test.go`
- `features/tt/internal/agent/e2e/far_knowledge_e2e_test.go`

上記テストで新フィールドの存在を検証するアサーションを追加。

#### 4. 既存 knowledge ファイルの修正

- `prompts/memory/knowledge/agent/record/branch-package/branchpackageinfo-and-slugify.md` の frontmatter に `id` と `status` を追加

## 検証シナリオ (Verification Scenarios)

1. `tt agent knowledge add` で新しい knowledge を作成する
2. 作成されたファイルの frontmatter に `id` と `status: current` が含まれていることを確認する
3. `tt prompt compile --dry-run` を実行し、バリデーションエラーが発生しないことを確認する
4. `tt prompt update` が正常に完了することを確認する

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```

2. knowledge 関連の統合テスト:
   ```
   scripts/process/integration_test.sh --categories "common" --specify "Knowledge|FarKnowledge"
   ```

### 個別検証

1. knowledge パッケージの単体テスト:
   ```
   cd features/tt && go test ./internal/agent/knowledge/... -v -run "TestAdd|TestAppend|TestFrontmatter"
   ```

2. prompt compile バリデーション:
   ```
   ./bin/tt.exe prompt compile --dry-run
   ```
