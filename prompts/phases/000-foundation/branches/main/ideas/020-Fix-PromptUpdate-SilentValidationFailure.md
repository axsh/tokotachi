# 020: tt prompt update のサイレントバリデーション失敗とスキーマ不整合の修正

## 背景 (Background)

`tt scaffold feature axsh-go-standard` でクリーンなプロジェクトを生成した後、`tt prompt update --target ag` を実行すると以下の問題が発生する:

1. `.agents/` ディレクトリが生成される (意図しない `.agent/` ではないディレクトリ)
2. `.agent/rules/` や `.agent/skills/` にコンテンツが配置されない
3. コマンドは "Update succeeded" と誤って報告する

### 根本原因

カタログの `target.schema.json` と scaffold で配置される target YAML ファイルの間にフィールド名の不整合がある:

- **スキーマ**: `capabilities` (required) + プロパティ `rules`, `skills`, `workflows`, `subagents`
- **YAML**: `includes` + プロパティ `policy`, `capability`, `procedure`, `subagent`

スキーマに `additionalProperties: false` が設定されているため、`includes` は不正プロパティとして拒否され、全ターゲットがバリデーション失敗する。しかし `tt prompt update` はこのエラーを握りつぶし、メタデータだけを書き込んで "Update succeeded" と報告する。

### 関連ファイル

- `catalog/originals/axsh/go-standard-project/base/prompts/manifest/schemas/target.schema.json`
- `catalog/originals/axsh/go-standard-project/base/prompts/manifest/targets/antigravity.yaml`
- `catalog/originals/axsh/go-standard-project/base/prompts/manifest/targets/codex.yaml`
- `catalog/originals/axsh/go-standard-project/base/prompts/manifest/targets/claude-code.yaml`
- `catalog/originals/axsh/go-standard-project/base/prompts/manifest/targets/cursor.yaml`
- `pkg/resolve/target.go` (targetMetaDirs)
- `features/tt/internal/prompt/compiler/update.go` (Update 関数)
- `features/tt/internal/prompt/compiler/deploy.go` (Deploy 関数)
- `features/tt/cmd/prompt.go` (runPromptUpdate 関数)
- `features/tt/internal/prompt/emitter/includes.go` (ExtractIncludes 関数)

## 要件 (Requirements)

### 必須要件

#### R1: target.schema.json を実際の YAML 構造に合わせる

`target.schema.json` を更新し、コード側 (`includes.go` の `ExtractIncludes`) が既にサポートしている `includes` フィールドおよび `policy`, `capability`, `procedure`, `subagent` プロパティを正式に受け入れるようにする。

具体的には:
- `required` から `capabilities` を除外し、`includes` も `capabilities` も optional とする (後方互換性)
- `includes` フィールドを追加: プロパティは `policy`, `capability`, `procedure`, `subagent` (全て boolean)
- `capabilities` フィールドは既存のまま残す (レガシー互換)
- `additionalProperties: false` はスキーマ全体で維持する (不正フィールドの検出は有用)

> 設計メモ: `ExtractIncludes` は既に `includes` -> `capabilities` フォールバックを実装しており、
> プロパティも新旧両方をサポートしている。スキーマだけがこの互換性に追随していない。

#### R2: tt prompt update でバリデーションエラーを報告する

`Update` 関数が `Deploy` の結果に含まれるバリデーションエラーを検出し、以下の動作を行う:

- バリデーションエラーがある場合、エラーメッセージを stderr に出力する
- メタデータ (`WriteMetadata`) を書き込まない
- CLI は "Update succeeded" ではなくエラーを報告する

影響箇所:
- `update.go`: `Deploy` 結果の `CompileResult.Errors` をチェックする
- `prompt.go`: `runPromptUpdate` でエラーがある場合に適切なメッセージを表示する

#### R3: targetMetaDirs の antigravity パスを修正する

`pkg/resolve/target.go` の `targetMetaDirs` で、antigravity のメタデータパスを `.agents/.meta/antigravity/` から `.agent/.meta/antigravity/` に変更する。

これにより、メタデータの配置先がエミッターの出力先パスのデフォルト (`.agent/`) と一致する。

> 注意: codex の `.agents/.meta/codex/` は `.codex/.meta/` に変更すべきだが、
> これは antigravity の問題とは独立しており、本仕様のスコープとする。

### 任意要件

#### R4 (任意): codex の targetMetaDirs パスも修正する

codex のメタデータパスを `.agents/.meta/codex/` から `.codex/.meta/` に変更する。他のターゲット (cursor: `.cursor/.meta/`, claude-code: `.claude/.meta/`) と同様に、自分のターゲットディレクトリ配下にメタデータを配置する設計に統一する。

## 実現方針 (Implementation Approach)

### 方針 1: スキーマ修正 (R1)

`target.schema.json` を以下のように修正する:

```json
{
  "required": ["apiVersion", "kind", "id", "paths"],
  "properties": {
    ...
    "includes": {
      "type": "object",
      "properties": {
        "policy": { "type": "boolean" },
        "capability": { "type": "boolean" },
        "procedure": { "type": "boolean" },
        "subagent": { "type": "boolean" }
      },
      "additionalProperties": false
    },
    "capabilities": {
      "type": "object",
      "properties": {
        "rules": { "type": "boolean" },
        "skills": { "type": "boolean" },
        "workflows": { "type": "boolean" },
        "subagents": { "type": "boolean" }
      },
      "additionalProperties": false
    },
    ...
  }
}
```

`oneOf` や `anyOf` で `includes` か `capabilities` のどちらか一方を必須にすることも可能だが、`ExtractIncludes` がどちらもなくてもデフォルト値を返す設計なので、両方 optional で問題ない。

カタログの `target.schema.json` と、scaffold で既に配置済みのプロジェクト内スキーマの両方を更新する必要がある。

### 方針 2: Update のエラーハンドリング修正 (R2)

`update.go` の `Update` 関数内で、`Deploy` 結果を受け取った後に `CompileResult.Errors` をチェックする:

```go
// Deploy 後にバリデーションエラーをチェック
if deployResult.CompileResult != nil && len(deployResult.CompileResult.Errors) > 0 {
    for _, e := range deployResult.CompileResult.Errors {
        fmt.Fprintln(os.Stderr, e.Error())
    }
    return nil, fmt.Errorf("deploy failed for target %s: %d validation error(s)", t, len(deployResult.CompileResult.Errors))
}
```

### 方針 3: targetMetaDirs の修正 (R3, R4)

`pkg/resolve/target.go` を修正:

```go
var targetMetaDirs = map[string]string{
    "antigravity": ".agent/.meta/antigravity/",   // .agents/ -> .agent/
    "cursor":      ".cursor/.meta/",
    "claude-code": ".claude/.meta/",
    "codex":       ".codex/.meta/",                // .agents/.meta/codex/ -> .codex/.meta/
}
```

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: クリーンフォルダでの scaffold + prompt update

1. 空のディレクトリで `tt scaffold feature axsh-go-standard` を実行
2. `tt prompt update --target ag` を実行
3. `.agent/rules/` 配下にポリシーファイルが配置されていることを確認
4. `.agent/skills/` 配下にスキルファイルが配置されていることを確認
5. `.agents/` ディレクトリが生成されていないことを確認
6. `tt prompt compile --target ag --dry-run` がバリデーションエラーなしで resolved manifest を出力することを確認

### シナリオ 2: バリデーションエラー時のエラー報告

1. target YAML を意図的に壊す (例: required フィールドを削除)
2. `tt prompt update --target ag` を実行
3. "Update succeeded" ではなくエラーメッセージが表示されることを確認
4. `.agent/.meta/antigravity/last_update.yaml` が書き込まれていないことを確認

### シナリオ 3: 既存の includes/capabilities 互換性

1. `includes` フィールドを使った target YAML で `tt prompt compile` が成功することを確認
2. `capabilities` フィールド (レガシー) を使った target YAML でも `tt prompt compile` が成功することを確認

## テスト項目 (Testing for the Requirements)

### R1: スキーマ修正の検証

- 既存の `target_test.go` を拡張し、`includes` フィールドを使った target YAML がバリデーションに通ることを確認
- `capabilities` フィールドを使った target YAML もバリデーションに通ることを確認 (後方互換)

### R2: Update エラーハンドリングの検証

- `update_test.go` にテストケースを追加: `CompileResult.Errors` が非空の場合、`Update` がエラーを返すことを確認
- `Update` がエラー時に `WriteMetadata` を呼ばないことを確認

### R3/R4: targetMetaDirs の検証

- `target_test.go` の `MetaDir` テストを更新: antigravity が `.agent/.meta/antigravity/` を返すことを確認
- codex が `.codex/.meta/` を返すことを確認

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```
