# tt prompt コマンドのパス指定オプション拡張

## 背景 (Background)

現状、`tt prompt compile` / `deploy` / `update` は、リポジトリ直下の `prompts/` をソースとし、`project.yaml` の設定に従って `tmp/dist/` に中間成果物を出力し、`deploy` ではエディタ向け設定（`.cursor/`、`.agent/` 等）を**同一リポジトリルート**へ展開する前提で動作している。

この前提は tokotachi 本体リポジトリでは十分だが、以下のような用途拡張には対応できない。

- プロンプト定義だけ別ディレクトリ／別リポジトリに置き、アプリケーション側へ deploy したい
- 中間成果物（resolved manifest や build 出力）を `tmp/dist/` 以外へ出したい
- deploy 先（エディタ設定の配置ルート）を compile 元と別のワークスペースにしたい
- monorepo や scaffold 生成物のように、`prompts/` 以外のパス構成を使いたい

現状の CLI・設定でどこまで指定可能かを整理し、不足するオプションを追加する。

## 現状調査: 既存のパス関連オプション

### CLI フラグ（`tt prompt` サブコマンド）

| コマンド | フラグ | デフォルト | 役割 |
|---|---|---|---|
| compile / deploy / update | `--project <path>` | `prompts/manifest/project.yaml` | `project.yaml` へのパス。CWD からの相対パス |
| compile / deploy / update | `--target <name>` | `TT_TARGET` 環境変数、未設定時 `all` | エミッタターゲット（`antigravity`, `cursor`, `claude-code`, `codex`, `all`） |
| compile | `--dry-run` | false | ファイルを書き出さず stdout に resolved manifest を出力 |
| compile | `--apply` | false | 生成物をターゲットディレクトリへ直接配置（deploy 相当の apply モード） |
| deploy | `--force` | false | ダイジェスト一致時も強制実行 |
| deploy | `--dry-run` | false | シミュレーション実行 |
| deploy | `--mode <mode>` | `overwrite` | `overwrite` / `immune` / `skip` |
| update | `--force` | false | git 変更検知・メタデータチェックをバイパス |
| update | `--dry-run` | false | シミュレーション実行 |

**グローバルフラグ**（全サブコマンド）: `--verbose`, `--dry-run`, `--report <file>`

**参考**: `scaffold` コマンドには `--root <path>` があり、Git ルートの代わりに展開先ルートを明示指定できる。`prompt` コマンドには同等のフラグはない。

### 環境変数

| 変数 | 用途 | デフォルト |
|---|---|---|
| `TT_TARGET` | `prompt` 系のデフォルト `--target` | `all` |
| `TT_TOOL` | ラッパースクリプトが呼び出す `tt` バイナリのパス | PATH / `bin/tt` |

パス（workspace / prompts / dist / deploy 先）を上書きする環境変数は**現状存在しない**。

### `project.yaml` による設定（ワークスペースルートからの相対パス）

`prompts/manifest/project.yaml` の例:

```yaml
sources:
  policies: prompts/manifest/code_content/policies/**/*.md
  memory_docs: prompts/memory/**/*.md
outputs:
  resolved_manifest: tmp/dist/manifest.resolved.yaml
defaults:
  build_dir: tmp/dist/
```

| 設定項目 | 役割 |
|---|---|
| `sources.*` | 各エンティティ種別のソース glob（`prompts/` 配下を前提としたパターン） |
| `outputs.resolved_manifest` | resolved manifest の出力先 |
| `defaults.build_dir` | エミッタの build 出力ルート（compile 時の staging 領域） |

### 暗黙のパス解決（CLI では指定不可）

以下はコード上ハードコードまたは `--project` から**暗黙推論**される。

| 項目 | 現状の解決方法 | 実装箇所 |
|---|---|---|
| **ワークスペースルート** | `--project` の 3 階層上を root とみなす（`{root}/prompts/manifest/project.yaml` 前提） | `ResolveProjectRoot()` |
| **スキーマディレクトリ** | `{root}/prompts/manifest/schemas` 固定 | `compiler.Compile()` |
| **deploy 先ルート** | ワークスペースルートと同一（`apply=true` 時） | 各 Emitter の `RootDir` |
| **build 出力先** | `{root}/{defaults.build_dir}` | `compiler.Compile()`, `Deploy()` |
| **ターゲット配下パス** | target エンティティの `paths`（例: `.cursor/rules/`） | `prompts/manifest/targets/*.yaml` |
| **update の git 監視パス** | `prompts/manifest/`, `prompts/memory/` 固定 | `CheckForChanges()` |
| **update メタデータ配置** | `{root}/.cursor/.meta/` 等、ターゲット別固定 | `pkg/resolve.MetaDir()` |

### ラッパースクリプト（`scripts/code/prompt/*.sh`）

`compile.sh` / `deploy.sh` / `update.sh` は `--force`, `--verbose`, `--dry-run` のみ `tt` に転送する。

`--project`, `--target`, `--apply`, `--mode` 等は**転送されない**（直接 `tt prompt ...` を呼べば利用可能）。

### 現状で可能なこと・不可能なこと

**可能（間接的）**:

- 別の `project.yaml` を `--project path/to/project.yaml` で指定（ただし root は `project.yaml` 位置から 3 階層上と推論される）
- `project.yaml` 内で `sources` / `build_dir` / `resolved_manifest` をカスタムパスに変更
- target エンティティの `paths` で deploy 先サブディレクトリ（`.cursor/rules/` 等）を変更

**不可能（CLI / 環境変数レベル）**:

- ワークスペースルートを `project.yaml` 位置と独立して指定
- deploy 先ルートを compile 元ワークスペースと分離
- `prompts/` ディレクトリ名・位置の共通上書き（スキーマパス・update 監視パス含む）
- `build_dir` / `resolved_manifest` の CLI 上書き
- ラッパースクリプト経由でのパス系オプション指定

## 要件 (Requirements)

### 必須要件

#### R1. ワークスペースルートの明示指定

- 新フラグ `--workspace <path>` を `prompt compile` / `deploy` / `update` に追加する
- 対応環境変数 `TT_WORKSPACE` を追加する（フラグが優先）
- デフォルト: 従来通り `--project` から `ResolveProjectRoot()` で推論
- `--workspace` 指定時:
  - `--project` が相対パスの場合、`{workspace}/{project}` として解決
  - deploy メタデータ（`.cursor/.meta/` 等）、Emitter の `RootDir`（apply 時）、digest 計算の root 基準に使用

#### R2. プロンプトソースルート（`prompts/` 相当）の指定

- 新フラグ `--prompts-dir <path>` を追加する
- 対応環境変数 `TT_PROMPTS_DIR` を追加する（デフォルト: `prompts`）
- 効果:
  - デフォルト `--project` を `{workspace}/{prompts-dir}/manifest/project.yaml` に解決（`--project` 明示時はそちらを優先）
  - スキーマディレクトリを `{workspace}/{prompts-dir}/manifest/schemas` に変更
  - `prompt update` の git 変更監視を `{prompts-dir}/manifest/`, `{prompts-dir}/memory/` に変更
- `project.yaml` の `sources` glob は従来通り workspace 相対パスとして解釈する（既存 project.yaml との互換維持）。`--prompts-dir` を使う場合は、project.yaml 側のパスも整合させる必要がある旨をドキュメント化する

#### R3. ビルド出力ディレクトリ（`tmp/dist/` 相当）の CLI 上書き

- 新フラグ `--build-dir <path>` を追加する
- 対応環境変数 `TT_BUILD_DIR` を追加する
- 指定時は `project.yaml` の `defaults.build_dir` より CLI/環境変数を優先
- resolved manifest 出力先は、従来どおり `outputs.resolved_manifest` を使うが、`--build-dir` 指定時に `--resolved-manifest` 未指定なら `{build-dir}/manifest.resolved.yaml` をデフォルトとする（後述 R3b）

#### R3b. resolved manifest 出力先の CLI 上書き（任意だが推奨）

- 新フラグ `--resolved-manifest <path>` を追加する
- 対応環境変数 `TT_RESOLVED_MANIFEST` を追加する
- `project.yaml` の `outputs.resolved_manifest` より優先

#### R4. deploy 先ルートの指定

- 新フラグ `--deploy-root <path>` を追加する
- 対応環境変数 `TT_DEPLOY_ROOT` を追加する
- デフォルト: `--workspace`（または推論された workspace）
- `deploy` / `compile --apply` / `update` で Emitter が `apply=true` のとき、target エンティティ `paths`（`.cursor/rules/` 等）の基準ルートとして使用
- compile の staging（`apply=false`）は従来通り `--build-dir` 配下

#### R5. 優先順位の統一

各パス設定の優先順位を以下に統一する。

```
CLI フラグ > 環境変数 > project.yaml > 組み込みデフォルト
```

#### R6. ラッパースクリプトの更新

`scripts/code/prompt/compile.sh`, `deploy.sh`, `update.sh` が新フラグを透過的に `tt` へ転送できるようにする。

- 転送対象: `--workspace`, `--prompts-dir`, `--build-dir`, `--resolved-manifest`, `--deploy-root`, `--project`, `--target`, `--apply`, `--mode` および既存フラグ
- 未知引数はエラーにせず `tt` に渡す方式（`compile.sh` 等の case 文を拡張、または `"$@"` 透過）を採用

#### R7. 後方互換性

- 新フラグ・環境変数未指定時は現行挙動と同一結果になること
- 既存の `project.yaml` および `ResolveProjectRoot()` の推論ロジックは維持

### 任意要件

- `tt-user-manual.md` に新フラグ・環境変数の説明を追記
- `ResolveProjectRoot()` を `--workspace` + `--project` ベースの明示解決に段階的移行（将来 `--project` のみでの root 推論を非推奨化）

## 実現方針 (Implementation Approach)

### パス解決モデル

4 つの論理パスを分離する。

```mermaid
flowchart LR
  subgraph sources [Source]
    PD["prompts-dir\n(manifest + memory)"]
  end
  subgraph build [Build]
    BD["build-dir\n(tmp/dist)"]
    RM["resolved-manifest"]
  end
  subgraph deploy [Deploy]
    DR["deploy-root"]
    TP["target paths\n(.cursor/, .agent/)"]
  end
  PD -->|compile| BD
  BD -->|deploy apply| DR
  DR --> TP
```

### 新パッケージ: パス解決ヘルパー

`features/tt/internal/prompt/compiler/paths.go`（仮）に `PathConfig` を導入する。

```go
type PathConfig struct {
    Workspace         string // 推論または --workspace
    ProjectYAML       string // 解決済み project.yaml 絶対パス
    PromptsDir        string // 相対: workspace 基準
    BuildDir          string // 相対: workspace 基準
    ResolvedManifest  string // 相対: workspace 基準
    DeployRoot        string // deploy apply 時のルート
}
```

`ResolvePaths(opts PathOptions) (PathConfig, error)` で CLI / 環境変数 / project.yaml をマージする。

### 変更対象

| ファイル | 変更内容 |
|---|---|
| `features/tt/cmd/prompt.go` | 新フラグ定義、`PathConfig` 解決呼び出し、各 `CompileOptions` / `DeployOptions` / `UpdateOptions` へ伝播 |
| `features/tt/internal/prompt/compiler/compiler.go` | `CompileOptions` に `PathConfig` 追加。スキーマ・buildDir・resolved manifest 書き込みを `PathConfig` 参照に変更 |
| `features/tt/internal/prompt/compiler/deploy.go` | digest パス、Emitter `RootDir` を `DeployRoot` / `BuildDir` に分離 |
| `features/tt/internal/prompt/compiler/update.go` | `CheckForChanges()` の監視パスを `PromptsDir` から生成 |
| `features/tt/internal/prompt/compiler/config.go` | `ResolveProjectRoot` を `PathConfig` 解決に統合（後方互換維持） |
| `features/tt/internal/prompt/emitter/*.go` | Emitter 構造体に `DeployRoot` を渡す（現 `RootDir` の意味を deploy apply 専用に明確化） |
| `scripts/code/prompt/*.sh` | 新フラグ透過 |
| `docs/manual/tt-user-manual.md` | ドキュメント追記 |

### CLI フラグ一覧（追加後）

```
tt prompt compile|deploy|update \
  [--workspace <path>] \
  [--prompts-dir <path>] \
  [--project <path>] \
  [--build-dir <path>] \
  [--resolved-manifest <path>] \
  [--deploy-root <path>] \
  [--target <name>] \
  ...既存フラグ...
```

### 使用例

**例 1: 同一リポジトリ（現行互換、変更なし）**

```bash
tt prompt deploy --target cursor
```

**例 2: プロンプト定義を `catalog/prompts/` に置く**

```bash
tt prompt deploy \
  --workspace . \
  --prompts-dir catalog/prompts \
  --project catalog/prompts/manifest/project.yaml
```

**例 3: 中間成果物だけ別ディレクトリへ**

```bash
tt prompt compile \
  --build-dir .cache/tt-dist \
  --target cursor
```

**例 4: 別ワークスペースへ deploy**

```bash
tt prompt deploy \
  --workspace ./prompt-repo \
  --deploy-root ../app-checkout \
  --target cursor
```

## 検証シナリオ (Verification Scenarios)

1. **後方互換**: 新フラグなしで `tt prompt deploy --target cursor` を実行し、従来と同一の `.cursor/` 出力および `tmp/dist/` 生成が行われること
2. **build-dir 上書き**: `--build-dir .cache/tt-dist` 指定時、resolved manifest と digest が `.cache/tt-dist/` 配下に生成され、`tmp/dist/` は変更されないこと
3. **deploy-root 分離**: `--deploy-root /path/to/other-ws` 指定時、`.cursor/rules/` 等が other-ws 側に出力され、prompt ソース側 workspace の `prompts/` は読み取り専用で触られないこと
4. **prompts-dir 変更**: `--prompts-dir custom/prompts` 指定時、スキーマ読み込み・update の git 監視パスが `custom/prompts/manifest/` / `custom/prompts/memory/` になること
5. **環境変数優先**: `TT_BUILD_DIR=env-dist TT_WORKSPACE=. tt prompt compile --build-dir cli-dist` で `cli-dist` が使われること（CLI > env）
6. **ラッパー透過**: `scripts/code/prompt/deploy.sh --deploy-root /tmp/ws --target cursor` が `tt` に正しく渡ること

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド＋単体テスト:
   ```
   scripts/process/build.sh
   ```

2. コンパイラ・パス解決の単体テスト（新規）:
   - `features/tt/internal/prompt/compiler/paths_test.go` で `ResolvePaths` の優先順位・デフォルト推論を検証
   - 既存 `compiler_test.go`, `deploy_test.go`, `update_test.go` に `--build-dir` / `--deploy-root` ケースを追加

3. 統合テスト（prompt コンパイラ関連）:
   ```
   scripts/process/integration_test.sh --categories "common" --specify "PromptCompile|PromptDeploy|ResolvePaths"
   ```

   影響範囲が `common` カテゴリに該当テストが無い場合は、上記 `--specify` で追加した Go 単体テストを `scripts/process/build.sh` の実行で代替確認する。

### 要件とテストの対応

| 要件 | 検証方法 |
|---|---|
| R1 workspace | `paths_test.go`: workspace 推論・CLI 上書き |
| R2 prompts-dir | `update_test.go`: git 監視パス生成、`compiler_test.go`: schemas パス |
| R3 build-dir | `deploy_test.go`: digest ファイル位置 |
| R3b resolved-manifest | `compiler_test.go`: 出力ファイルパス |
| R4 deploy-root | `deploy_test.go` + emitter テスト: apply 先 RootDir |
| R5 優先順位 | `paths_test.go`: flag > env > yaml |
| R6 ラッパー | shell スクリプトの引数透過テスト（可能なら bats、最低限手動シナリオ 6） |
| R7 後方互換 | 既存 `compiler/testdata/valid` による regression |
