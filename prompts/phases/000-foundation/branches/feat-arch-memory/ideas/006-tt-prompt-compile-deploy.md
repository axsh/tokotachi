# ttツールへのプロンプトコンパイル・デプロイ機能の追加

## 背景

本リポジトリ(tokotachi)には、サブリポジトリ vv5 由来のプロンプト管理基盤（`prompts/manifest/` および `prompts/memory/`）が既に存在している。vv5 では `agentctl` というコンパイラツールが以下の機能を提供している:

1. **compile**: manifest (YAML/Markdown) と memory ドキュメントを解析・バリデーション・解決し、`index.md` や `resolved manifest` を生成する
2. **deploy**: compile の成果物をターゲット別（antigravity, cursor, claude-code, codex）にエミットし、`.agent/`, `.cursor/` 等のルートフォルダに配備する
3. **validate**: manifest の構文・スキーマ検証
4. **check**: 配備済みファイルとコンパイル結果の差分(ドリフト)検知
5. **init**: ワークスペースの初期化（ttの`scaffold`に相当するため不要）

現在、本リポジトリでは `scripts/prompt/compile.sh` と `scripts/prompt/deploy.sh` が `agentctl` のラッパーとして存在するが、これを `tt` コマンドのサブコマンドとして組み込むことで、エージェント設定ファイル（`.agent/rules/`, `.agent/skills/`, `.agent/workflows/` 等）の自動生成・配備を `tt` から直接実行可能にする。

### 現状の課題

- `agentctl` は独立したバイナリで、PATHまたは`bin/`に配置が必要。ttとは別のモジュールである
- `_resolve_tool.sh` でagentctlを探索するが、見つからない場合はスキップされてしまう
- ttユーザにとって、agentctlのインストール・管理が追加負担になる
- manifest/memoryの変更後、AIエージェントが呼び出すスクリプトからttを利用できない

## 要件

### 必須要件

1. **`tt prompt compile` サブコマンドの実装**
   - `agentctl compile` と同等の機能を提供する
   - `prompts/manifest/project.yaml` を起点にmanifestとmemoryドキュメントを解析・検証する
   - 解析結果からターゲット別（antigravity, cursor, claude-code, codex）のファイルを生成する
   - `--project` オプション: project.yaml のパス指定（デフォルト: `prompts/manifest/project.yaml`）
   - `--dry-run` オプション: ファイル出力せず標準出力に結果を表示
   - `--target` オプション: エミットターゲットの指定（antigravity, cursor, claude-code, codex）
   - `--apply` オプション: 生成ファイルをターゲットディレクトリに直接配置

2. **`tt prompt deploy` サブコマンドの実装**
   - `agentctl deploy` と同等の機能を提供する
   - ソースのダイジェスト（SHA-256）を計算し、前回のダイジェストと比較して変更検知を行う
   - 変更がなければスキップ（`--force` で強制再コンパイル）
   - compile を内部で呼び出し、成功後にターゲットディレクトリにファイルを配備する
   - `--project` オプション: project.yaml のパス指定
   - `--target` オプション: エミットターゲットの指定（デフォルト: `antigravity`）
   - `--force` オプション: ダイジェストに関わらず強制再コンパイル
   - `--dry-run` オプション: シミュレーション実行
   - `--mode` オプション: ファイル配備モード（overwrite, immune, skip）

3. **`scripts/prompt/compile.sh` と `scripts/prompt/deploy.sh` の書き換え**
   - 既存の `agentctl` 呼び出しを `tt prompt compile` / `tt prompt deploy` に置き換える
   - `_resolve_tool.sh` を `tt` ツールを探索する形に変更する

4. **agentctl の内部パッケージの移植**
   - agentctl の `internal/compiler`, `internal/manifest`, `internal/memory`, `internal/emitter` パッケージのコードを ttモジュール内に移植する
   - 独立した Go モジュール（`github.com/axsh/tokotachi/features/agentctl`）から、ttの依存範囲内で利用可能な形にする

### 除外要件

- `agentctl init` コマンドの移植は不要（ttの`scaffold`が上位互換機能を持つ）
- `agentctl validate` / `agentctl check` の移植はこのフェーズでは対象外とする（将来的に追加可能だが、compile/deployが優先）

## 実現方針

### アーキテクチャ

agentctl の内部パッケージを `features/tt` のモジュール内に取り込む。agentctl 自体は `github.com/axsh/tokotachi/features/agentctl` として独立モジュールだが、主要ロジックは Go パッケージとして移植可能である。

```
features/tt/
  cmd/
    prompt.go          # [NEW] tt prompt サブコマンドの定義（compile/deploy）
  internal/
    prompt/             # [NEW] agentctlのロジック移植先
      compiler/         # compiler.go, config.go, deploy.go, digest.go
      manifest/         # types.go, parser.go, resolver.go, validator.go
      memory/           # frontmatter.go, indexer.go
      emitter/          # emitter.go, antigravity.go, cursor.go, claude_code.go, codex.go
                        # emit_mode.go, template.go, marker.go, limits.go
```

### コンパイルパイプライン（agentctlからの移植）

agentctlのコンパイルパイプラインは以下のステップで構成される:

1. **設定読み込み**: `prompts/manifest/project.yaml` をパース → `ProjectConfig`
2. **プロジェクトルート算出**: project.yaml のパスから2階層上をルートとする
3. **エンティティパース**: sources セクションの glob パターンに従い、YAML/Markdown ファイルを `Entity` に変換
4. **メモリドキュメントパース**: `memory_docs` パターンにマッチする Markdown の frontmatter を `MemoryDoc` に変換（GENERATED FILE バナー付きファイルはスキップ）
5. **スキーマバリデーション**: `prompts/manifest/schemas/` 配下の JSON Schema で検証
6. **ID一意性検証**: Entity と MemoryDoc 全体でIDの重複をチェック
7. **参照整合性検証**: depends_on 等の参照先が実際に存在するか検証
8. **解決 (Resolve)**: エンティティを kind 別にグループ化し `ResolvedManifest` を構築
9. **index.md 生成**: MemoryDoc からルーティングテーブルを含む index.md を生成
10. **Resolved Manifest 出力**: YAML として解決済みマニフェストを書き出し
11. **エミッター呼び出し**: ターゲット指定時、対応するエミッターで各種設定ファイルを生成

### デプロイパイプライン

1. **設定読み込み** → プロジェクトルート算出
2. **ダイジェスト計算**: ソースファイル全体の SHA-256 ハッシュを計算
3. **前回ダイジェストとの比較**: `tmp/dist/.compile-digest` (または `tmp/dist/.compile-digest-{target}`) に保存された値と比較
4. **変更なし → スキップ** (force フラグで上書き可能)
5. **compile 実行** (ターゲット指定 + apply 有効)
6. **ダイジェスト保存**: コンパイル後に再計算したダイジェストを保存

### エミッターの動作

各ターゲットのエミッターは `ResolvedManifest` を受け取り、ターゲット固有のファイル群を生成する:

| ターゲット | 出力先 | 内容 |
|:---|:---|:---|
| antigravity | `.agent/rules/`, `.agent/skills/`, `.agent/workflows/` | Policy -> rules, Capability -> skills, Procedure -> workflows |
| cursor | `.cursor/rules/`, `.cursor/skills/` | Policy -> rules, Capability -> skills (workflowsなし) |
| claude-code | `.agents/rules/`, `.agents/skills/` | Policy -> rules, Capability -> skills |
| codex | `.agents/rules/`, `.agents/skills/` | Policy -> rules, Capability -> skills |

エミットモード:
- **overwrite**: 常に上書き（デフォルト）
- **immune**: 上書き + 対象ディレクトリ内の管理外ファイル（orphan）を削除
- **skip**: ファイルが存在しない場合のみ書き込み

### cobra コマンド構造

```
tt prompt
  tt prompt compile [--project PATH] [--target TARGET] [--dry-run] [--apply]
  tt prompt deploy  [--project PATH] [--target TARGET] [--force] [--dry-run] [--mode MODE]
```

`tt prompt` をサブコマンドグループとし、`compile` と `deploy` を子コマンドとして配置する。

### スクリプト書き換え

`scripts/prompt/_resolve_tool.sh` を以下のように変更する:
- `agentctl` ではなく `tt` コマンドを探索する
- `TT_TOOL` 環境変数が設定されていればそれを使用
- `command -v tt` で PATH 上の tt を探索
- プロジェクトローカルの `bin/tt` をフォールバック

`scripts/prompt/compile.sh`:
```bash
exec "$TOOL" prompt compile "$@"
```

`scripts/prompt/deploy.sh`:
```bash
exec "$TOOL" prompt deploy "$@"
```

### 依存関係

agentctl が使用している依存パッケージ:
- `gopkg.in/yaml.v3`: ttで既に使用中
- `github.com/santhosh-tekuri/jsonschema/v6`: スキーマバリデーション用（新規追加）
- `github.com/yuin/goldmark` + `github.com/yuin/goldmark-meta`: frontmatter パース用（新規追加）

ttの `go.mod` にこれらを追加する必要がある。

## 検証シナリオ

### シナリオ1: compile の基本動作

1. `tt prompt compile --project prompts/manifest/project.yaml --dry-run` を実行する
2. 標準出力に `index.md` の内容と resolved manifest が表示されることを確認する
3. ファイルが書き込まれないことを確認する

### シナリオ2: compile + apply

1. `tt prompt compile --project prompts/manifest/project.yaml --target antigravity --apply` を実行する
2. `.agent/rules/`, `.agent/skills/`, `.agent/workflows/` 以下にファイルが生成されることを確認する
3. 生成ファイルの内容が `prompts/manifest/` のソースファイルと対応していることを確認する

### シナリオ3: deploy の基本動作

1. `tt prompt deploy --project prompts/manifest/project.yaml --target antigravity` を実行する
2. `.agent/` 以下にファイルが配備されることを確認する
3. 再度同じコマンドを実行し、「No changes detected. Skipping deploy.」と表示されることを確認する
4. `--force` を付けて実行し、再コンパイル・配備されることを確認する

### シナリオ4: deploy --dry-run

1. `tt prompt deploy --dry-run` を実行する
2. ファイルが実際に書き込まれないことを確認する
3. 「Deploy dry-run completed.」と表示されることを確認する

### シナリオ5: スクリプト経由の実行

1. `scripts/prompt/compile.sh --dry-run` を実行する
2. 内部で `tt prompt compile --dry-run` が呼び出されることを確認する
3. `scripts/prompt/deploy.sh --target antigravity` を実行する
4. 内部で `tt prompt deploy --target antigravity` が呼び出されることを確認する

### シナリオ6: 複数ターゲットへのデプロイ

1. `tt prompt deploy --target cursor` を実行する
2. `.cursor/rules/`, `.cursor/skills/` 以下にファイルが配備されることを確認する
3. `tt prompt deploy --target antigravity` を実行する
4. `.agent/rules/`, `.agent/skills/`, `.agent/workflows/` 以下にファイルが配備されることを確認する
5. 各ターゲットの配備結果が独立していることを確認する

### シナリオ7: バリデーションエラー時の動作

1. 不正な manifest ファイル（IDなし等）を一時的に作成する
2. `tt prompt compile --project prompts/manifest/project.yaml` を実行する
3. バリデーションエラーが表示され、ファイル生成が行われないことを確認する
4. 終了コードが非ゼロであることを確認する

## テスト項目

### 単体テスト (Unit Test)

テスト対象は `features/tt/internal/prompt/` 配下の各パッケージ。agentctl の既存テストコードを移植・適応する。

| パッケージ | テスト対象 | テスト内容 |
|:---|:---|:---|
| manifest | ParseEntity | YAML/Markdown からの Entity パース、必須フィールド検証 |
| manifest | ExpandGlob | `**` パターンを含む glob 展開 |
| manifest | Validator | JSON Schema によるバリデーション |
| manifest | ValidateIDUniqueness | ID重複検知 |
| manifest | ValidateReferences | 参照整合性検証 |
| memory | ParseFrontmatter | frontmatter パース、必須フィールド検証 |
| memory | GenerateIndex | index.md 生成、ルーティングテーブルの正確性 |
| memory | ParseAllMemoryDocs | GENERATED FILE スキップ動作 |
| compiler | LoadConfig | project.yaml の読み込みと検証 |
| compiler | Compile | パイプライン全体の正常系・異常系 |
| compiler | Deploy | ダイジェスト比較、スキップ判定、強制再コンパイル |
| compiler | ComputeSourceDigest | ダイジェスト計算の再現性、ファイル変更時の変化 |
| emitter | AntigravityEmitter | .agent/ 以下へのファイル生成 |
| emitter | CursorEmitter | .cursor/ 以下へのファイル生成 |
| emitter | writeFileWithMode | overwrite/immune/skip モードの動作 |
| emitter | CleanOrphanFiles | orphanファイルの検出と削除 |
| emitter | ResolveTemplateVars | テンプレート変数の解決 |

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh --skip-frontend --skip-etc
   ```

2. バックエンド統合テスト（共通機能のリグレッション確認）:
   ```
   scripts/process/integration_test.sh --categories "common"
   ```

   > 注: この機能はバックエンドの新規コマンド追加であり、GUI は影響しない。統合テストでは compile/deploy の E2E 動作を `common` カテゴリで検証する。
