# tt prompt update がコンパイル・デプロイをスキップする条件の分析

## 背景 (Background)

`tt prompt update` コマンドは、プロンプトソースファイルに変更があった場合にのみコンパイルとデプロイを実行する差分検出機能を持つ。しかし、ソースファイルを変更したにもかかわらず「変更無し」と判定されてスキップされるケースが報告されている。

本仕様書は、`tt prompt update` の変更検出ロジックを詳細に分析し、どのような条件でコンパイルがスキップされるかを明らかにする。

## 要件 (Requirements)

### 分析対象

`tt prompt update` のスキップ判定は、2つのレイヤーで行われる:

1. **Update レイヤー** ([update.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-compile-bug/features/tt/internal/prompt/compiler/update.go))
2. **Deploy レイヤー** ([deploy.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-compile-bug/features/tt/internal/prompt/compiler/deploy.go))

## 実現方針 (Implementation Approach) - 現状ロジックの分析

### Update レイヤーの判定フロー (update.go)

`Update()` 関数は各ターゲットに対して以下の3段階の変更検出を行い、いずれも「変更あり」と判定しなかった場合にスキップする:

```
shouldUpdate() -> CheckForChanges() (git log) -> CheckDrift() -> スキップ判定
```

#### ステップ 1: `shouldUpdate()` (L132-140)

```go
func shouldUpdate(meta *UpdateMetadata, force bool) bool {
    if force { return true }
    if meta == nil { return true }
    return false
}
```

- `--force` フラグが指定されていれば常に更新
- メタデータ (`last_update.yaml`) が存在しなければ更新
- **メタデータが存在する場合は `false` を返す** (ここでは更新不要と判定)

#### ステップ 2: `CheckForChanges()` (L77-85)

`shouldUpdate()` が `false` を返した場合にのみ実行される。

```go
if meta != nil {
    updatedAt, err := time.Parse(time.RFC3339, meta.UpdatedAt)
    if err == nil {
        gitChanged, _ := CheckForChanges(rootDir, updatedAt)
        needsUpdate = gitChanged
    }
}
```

`CheckForChanges()` (L144-157) は `git log --since=<UpdatedAt>` を実行して `prompts/manifest/` と `prompts/memory/` ディレクトリへのコミットがあるかを確認する。

**スキップされるケース:**
- git log の `--since` 時刻以降にコミットがない場合
- **ファイルを変更したがコミットしていない場合 (git add / git commit していない)**

> [!WARNING]
> **重大な問題**: `CheckForChanges()` は **コミットされた変更のみ** を検出する。`git log` を使用しているため、ワーキングツリーやステージングエリアの変更は検出されない。つまり、ファイルを編集しただけでは「変更あり」と判定されない。

#### ステップ 3: `CheckDrift()` (L88-92)

`CheckForChanges()` でも変更が検出されなかった場合に実行される。

`CheckDrift()` (deploy.go L124-162) は、コンパイル済みの出力ファイル(`.agents/rules/`, `.agents/skills/` など)が期待される内容と一致しているかを確認する。

- 出力ファイルが欠けている場合 → drift 検出 → 更新実行
- 出力ファイルの内容が期待と異なる場合 → drift 検出 → 更新実行
- 出力ファイルが全て最新の場合 → drift なし → **スキップ**

### Deploy レイヤーの判定フロー (deploy.go)

Update レイヤーで「更新が必要」と判定された場合、`Deploy()` が呼ばれるが、Deploy にも独自のダイジェストチェックがある:

#### ステップ 4: ソースダイジェスト比較 (deploy.go L72-78)

```go
if !opts.Force && prevInfo.Digest == currentDigest && currentDigest != "" {
    if !CheckDrift(rootDir, opts.ProjectPath, target) {
        result.Skipped = true
        return result, nil
    }
}
```

- `ComputeSourceDigest()` (digest.go) がソースファイルの SHA-256 ハッシュを計算
- 前回保存したダイジェスト (`tmp/dist/.compile-digest`) と比較
- **ダイジェストが一致し、かつ drift がなければスキップ**

> [!IMPORTANT]
> **二重ゲートの問題**: Update レイヤーと Deploy レイヤーの両方にスキップ判定がある。Update レイヤーで「更新が必要」と判定されても、Deploy レイヤーのダイジェスト比較で再びスキップされる可能性がある。ただし、通常は Update で更新必要と判定されれば Deploy でも同様になるはず。

### ソースダイジェストの計算対象 (digest.go)

`ComputeSourceDigest()` は `project.yaml` の `sources` セクションで定義されたグロブパターンにマッチするファイルを対象とする:

```yaml
sources:
  policies: prompts/manifest/code_content/policies/**/*.md
  procedures: prompts/manifest/code_content/procedures/**/*.md
  capabilities: prompts/manifest/code_content/capabilities/**/*.md
  refs: prompts/manifest/code_content/refs/**/*.md
  guards: prompts/manifest/safety/guards/**/*.yaml
  workers: prompts/manifest/safety/workers/**/*.yaml
  bundles: prompts/manifest/safety/bundles/**/*.yaml
  targets: prompts/manifest/targets/**/*.yaml
  memory_docs: prompts/memory/**/*.md
```

> [!CAUTION]
> **ダイジェスト対象外のファイル**: 以下のファイルはダイジェスト計算に含まれない。これらを変更しても「変更なし」と判定される:
> - `project.yaml` 自体
> - `prompts/manifest/schemas/` 配下のスキーマファイル
> - グロブパターンにマッチしないファイル（拡張子違い、配置場所違いなど）
> - `.agents/` 配下のデプロイ済みファイル（出力先の手動編集）

## スキップされる全ケースのまとめ

| No | 条件 | 根拠 | 影響度 |
|----|------|------|--------|
| 1 | ソースファイルを編集したがコミットしていない | `CheckForChanges()` が `git log` ベースのため未コミット変更を検出不可 | 高 |
| 2 | `project.yaml` を変更した | ダイジェスト計算対象に含まれない | 中 |
| 3 | スキーマファイルを変更した | ダイジェスト計算対象に含まれない | 低 |
| 4 | グロブパターンにマッチしない場所にファイルを追加した | `ExpandGlob` がマッチしない | 中 |
| 5 | `last_update.yaml` の `UpdatedAt` がソースコミットより新しい | `git log --since` で変更が見つからない | 中 |
| 6 | Deploy レイヤーのダイジェストが一致し、出力ファイルにドリフトがない | ダイジェスト比較で一致 | 低(正常動作) |
| 7 | `memory_config` に変更を加えた場合（ダイジェスト対象に `memory_config` パターンがない場合） | ダイジェスト計算対象外 | 低 |

## 検証シナリオ (Verification Scenarios)

### シナリオ A: 未コミット変更のスキップ確認

1. `tt prompt deploy --force` で初回デプロイを実行する
2. `prompts/manifest/code_content/policies/` 配下の `.md` ファイルを編集する（コミットしない）
3. `tt prompt update` を実行する
4. 期待結果: 「No changes detected」と表示されスキップされる（現在の動作）
5. `git add . && git commit -m 'test'` でコミットする
6. `tt prompt update` を再度実行する
7. 期待結果: コンパイル・デプロイが実行される

### シナリオ B: ダイジェスト対象外ファイルの変更

1. `tt prompt deploy --force` で初回デプロイを実行する
2. `prompts/manifest/project.yaml` の `defaults.language` を変更してコミットする
3. `tt prompt update` を実行する
4. 期待結果: `project.yaml` はダイジェスト対象外のためスキップされる

### シナリオ C: ドリフト検出による再デプロイ

1. `tt prompt deploy --force` で初回デプロイを実行する
2. `.agents/rules/` 配下のデプロイ済みファイルを手動で削除する
3. `tt prompt update` を実行する
4. 期待結果: ドリフト検出によりコンパイル・デプロイが実行される

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド+単体テスト:
   ```
   scripts/process/build.sh
   ```

2. コンパイラ関連の統合テスト（変更検出ロジックのリグレッション確認）:
   ```
   scripts/process/integration_test.sh --categories "common" --specify "Compile|Deploy|Update"
   ```

### 個別テスト

修正を行う場合、以下の既存テストが影響を受ける:

- `TestUpdate_Drift` (update_test.go): ドリフト検出の正常動作
- `TestDeploy_NoChanges` (deploy_test.go): 変更なし時のスキップ
- `TestDeploy_WithChanges` (deploy_test.go): 変更あり時のデプロイ
- `TestCheckForChanges_NoGitChanges` (update_test.go): git変更なし時の判定

修正時には、未コミット変更を検出する新しいテストケースを追加する必要がある。
