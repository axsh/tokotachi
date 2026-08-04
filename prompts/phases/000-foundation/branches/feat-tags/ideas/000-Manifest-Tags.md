# Manifest タグによるコンパイル/デプロイ対象の取捨選択

## 背景

本プロジェクトの prompt システムでは、`prompts/manifest/code_content/` 配下の Manifest（capability / policy / procedure / refs）をコンパイル・デプロイして、各 AI ツール向けの rules / skills 等へ展開している。現状、どの Manifest を作業対象にするかの制御は次に限られる。

- `project.yaml` の glob による取り込み
- `kind: skip` による emit 除外
- `target.includes` による kind 単位の on/off

一方で、「試験用の capability だけを一時的にデプロイしたい」「本番用の default セットと実験用セットを同居させたい」といった、**エンティティ単位のラベルによる取捨選択**はできない。

また、policy / skip スキーマには既に `tags` フィールド（Classification tags）が定義されているが、実行時パイプラインには未接続である。本仕様ではこのフィールドを **コンパイル/デプロイ対象の選択タグ** として意味付けを揃え、code_content 全体へ横断的に導入する。

一般利用者向けのカタログテンプレート（`catalog/originals/axsh/go-standard-project/`）も同期して改修し、新規スキャフォールドから同じタグ機構を使えるようにする。

## 要件

### 必須要件

1. **タグフィールドの導入**
   - `prompts/manifest/code_content/` 配下の Manifest（capability / policy / procedure / skip）の Frontmatter に `tags` を定義できること。
   - 対象スキーマ（プロジェクト側およびカタログ側の両方）:
     - `capability.schema.json`
     - `policy.schema.json`（既存の `tags` を本仕様のセマンティクスに合わせて更新）
     - `procedure.schema.json`
     - `skip.schema.json`（既存の `tags` を同様に更新）
   - `safety/`（guard / worker / bundle）および `targets/` はタグ対象外とする。
   - `prompts/memory/` 配下（memory_docs / branch skills）はタグの影響を受けず、**常に処理対象**とする。

2. **暗黙の `default` タグ**
   - Frontmatter に `tags` が無い（フィールド省略）場合、その Manifest は暗黙的に `default` タグを持つものとして扱う。
   - Frontmatter に `tags` が明示されている場合、その配列がタグ集合の唯一の定義となる。
     - 例: `tags: [test]` は `test` のみを持ち、`default` は持たない。
     - `default` も付けたい場合は `tags: [default, test]` のように明示する。

3. **CLI オプション `--tags`**
   - `prompt compile` および `prompt deploy` に `--tags` オプションを追加する。
   - 書式: `--tags <tag>[,<tag>...]`（カンマ区切り、複数指定可）。
   - **OR 条件**: 指定タグのいずれか 1 つ以上を持つ Manifest を対象とする（AND ではない）。
   - `--tags` を省略した場合は `--tags default` と同等に扱う。
   - ラッパースクリプト `scripts/code/prompt/compile.sh` / `deploy.sh` 経由でも同じフラグが透過されること（既存どおり `"$@"` 転送）。

4. **フィルタ適用範囲**
   - タグで取捨選択されるのは **code_content 由来の Manifest エンティティのみ**。
   - memory、safety、targets は `--tags` の値に関わらず従来どおり処理する。

5. **後方互換**
   - 既存の `tags` 無し Manifest は、暗黙 `default` により、従来どおり `--tags` 省略時のコンパイル/デプロイ対象であり続けること。
   - 既存のコンパイル/デプロイ呼び出し（`--tags` 無し）の結果が、タグ導入前と実質同等であること（code_content がすべて暗黙 `default` の場合）。

6. **カタログの明示タグ**
   - `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/` 配下の全 Manifest に `tags: [default]`（または同等の明示形）を記載する。
   - 同カタログの `schemas/` もプロジェクト側と同様に `tags` を定義する。

7. **タグ値の正規化**
   - Frontmatter の `tags: test`（スカラー）と `tags: [test]`（配列）の両方を受け付ける。
   - スカラーは内部的に 1 要素配列へ正規化する。
   - 空配列 `tags: []` は「どのタグも持たない」と解釈し、いかなる `--tags` 指定でも code_content フィルタ対象外とする（暗黙 `default` は付与しない。フィールドが存在する＝明示扱い）。

### 任意要件

1. `prompt update` への `--tags` 追加（compile/deploy と一貫させる場合）。仕様確定時に必須へ昇格してよい。
2. 環境変数 `TT_TAGS`（他の `TT_*` フラグとの一貫性）。省略時は CLI 省略と同じく `default`。
3. プロジェクト作業ツリー側（`prompts/manifest/code_content/`）への `tags: [default]` 明示付与。後方互換上は不要だが、カタログとの見た目統一には有用。

## 実現方針

### タグ解決ルール（Effective Tags）

各 code_content エンティティについて、実効タグ集合 `EffectiveTags` を次で定める。

```
if frontmatter に tags キーが存在しない:
    EffectiveTags = {"default"}
else:
    EffectiveTags = set(正規化後の tags 配列)
    # tags: [] → 空集合
    # tags: test → {"test"}
    # tags: [default, test] → {"default", "test"}
```

### 選択ルール（Selection）

```
RequestedTags = parse(--tags)  # 省略時は ["default"]
エンティティを選択 ⇔ EffectiveTags ∩ RequestedTags ≠ ∅
```

memory / safety / targets はこの判定を行わず常に選択。

### フィルタ適用タイミング

推奨: **Resolve 後〜 Emit 前**（または Resolve 内）で code_content エンティティを間引く。

- スキーマ検証・ID 一意性検証は、フィルタ前の全エンティティに対して実施する（不正な Manifest をタグで隠して通すことを防ぐ）。
- 参照整合性（`uses_capabilities` / `bundle.includes` 等）は次のいずれかで扱う（「検討事項」参照）:
  - **案 A（推奨案）**: フィルタ後の集合に対して参照検証する。選択外エンティティへの参照はエラー。
  - **案 B**: フィルタ前に参照検証し、emit 時のみ間引く。選択外への参照は警告または無視。

本仕様の初期推奨は **案 A**（選択された集合が自己完結であることを強制）。ただし利用者体験への影響が大きいため、実装前に確認する。

### スキーマ変更

`tags` プロパティを次の形で統一する（capability / procedure は新規追加、policy / skip は description を更新）:

```json
"tags": {
  "description": "Selection tags for compile/deploy filtering. If omitted, the entity implicitly has the 'default' tag. Explicit tags replace the implicit default; include 'default' explicitly when needed.",
  "oneOf": [
    { "type": "string", "minLength": 1 },
    {
      "type": "array",
      "items": {
        "type": "string",
        "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
        "minLength": 1
      }
    }
  ]
}
```

タグ名は kebab-case（既存 `id` と同様）を推奨パターンとする。

### CLI / 実装配置

| 層 | 変更概要 |
|---|---|
| `features/tt/cmd/prompt.go` | `compile` / `deploy`（任意で `update`）に `--tags` フラグ追加 |
| `features/tt/internal/prompt/compiler` | `CompileOptions` / `Deploy` に `Tags []string` を渡し、フィルタへ接続 |
| `features/tt/internal/prompt/manifest` | Effective Tags 算出、エンティティ選択、スキーマ更新に合わせたパース正規化 |
| `prompts/manifest/schemas/*.schema.json` | `tags` 追加・更新 |
| `catalog/.../schemas/*.schema.json` | 同上 |
| `catalog/.../code_content/**/*.md` | 全ファイルに `tags: [default]` を明示 |

### Digest との関係

`deploy` のソース digest は、**タグで除外された code_content も含め、従来どおり全 sources glob の内容**から算出する（推奨）。理由:

- タグ切替だけでは digest miss になり、意図しないスキップを防ぐ。
- 「どのタグセットをデプロイしたか」は digest とは別概念とし、必要なら将来 `TT_TAGS` を digest メタに含める拡張余地を残す。

（代替案は検討事項に記載。）

### 用語の整理

| 名称 | 意味 | 本仕様のタグとの関係 |
|---|---|---|
| タグ `default` | 暗黙／明示の選択ラベル | 本仕様の中心概念 |
| bundle `id: default` | safety bundle 名 | **無関係**（名前衝突に注意） |
| `kind: skip` | emit しないエンティティ種別 | 併存。skip もタグ付け可能だが emit されない点は従来どおり |
| 旧 policy.`tags`（Classification） | 未使用メタデータ | 本仕様の選択タグへ意味を統一 |

## 検証シナリオ

1. **暗黙 default（後方互換）**
   1. `tags` 無しの capability を用意する。
   2. `--tags` 無しで `prompt compile --dry-run` を実行する。
   3. 当該 capability が resolved manifest / emit 対象に含まれることを確認する。

2. **明示 test のみ**
   1. `pre-push-knowledge-check.md` 相当の Frontmatter に `tags: test`（または `tags: [test]`）を付ける。
   2. `--tags` 無し（＝暗黙 `default`）で compile すると、当該ファイルは対象外になる。
   3. `--tags test` で compile すると対象になる。
   4. memory_docs および safety / targets はどちらの実行でも従来どおり処理される。

3. **複数タグ OR**
   1. エンティティ A: `tags: [default]`、B: `tags: [test]`、C: `tags: [default, test]`、D: `tags` 省略（暗黙 default）。
   2. `--tags default,test` で A,B,C,D すべてが対象になる。
   3. `--tags test` で B,C のみが対象になる（A,D は対象外）。

4. **カタログ明示 default**
   1. カタログ `code_content` 配下の全 md に `tags` が明示されていることを確認する。
   2. スキャフォールド由来の新規プロジェクトでも `--tags` 省略時にそれらが対象になる。

5. **スカラー／配列の両対応**
   1. `tags: test` と `tags: [test]` が同一の Effective Tags になることを確認する。

## テスト項目

### 単体テスト（TDD）

`features/tt/internal/prompt/manifest/` および `compiler/` にテーブル駆動テストを追加する。

| ケース | 期待 |
|---|---|
| tags 省略 | EffectiveTags = {default}、`--tags` 省略で選択される |
| `tags: [test]` | {test}、`--tags default` で非選択、`--tags test` で選択 |
| `tags: [default, test]` | 両方の指定で選択 |
| `tags: test`（スカラー） | `[test]` と同等 |
| `tags: []` | 空集合、いずれの RequestedTags でも非選択 |
| memory / safety | `--tags test` でも常に処理される |
| OR 条件 | `--tags default,test` でいずれか一致すれば選択 |

CLI フラグ解析（`features/tt/cmd/prompt_test.go`）:

- `--tags` 省略 → 内部で `default`
- `--tags default,test` → `["default","test"]`

### ビルド・全体検証

1. ビルド＋単体テスト:
   `scripts/process/build.sh`

2. prompt 経路の統合テスト（パス／compile のリグレッション）:
   `scripts/process/integration_test.sh --categories "tt" --specify "PromptPaths|Prompt"`

3. （実装後に追加する）タグ選択の統合テストを同カテゴリへ追加し、上記 `--specify` に含める。

## 検討事項・仕様上の問題点

実装計画に進む前に、以下を確定したい。

### 1. 既存 `policy.tags` / `skip.tags` の意味変更

現状スキーマ上は「Classification tags」であり実行時未使用。本仕様で選択タグへ転用する。

- **問題**: 将来「分類用タグ」と「選択用タグ」を分けたくなった場合、フィールド名が足りない。
- **選択肢**: (a) 本仕様どおり `tags` を選択専用にする（推奨・単純） / (b) `select_tags` 等の別名を新設し旧 `tags` は残す。

### 2. 参照整合性（案 A vs 案 B）

`uses_capabilities` がタグ除外された capability を指す場合。

- 案 A（フィルタ後検証）: エラー → セットの自己完結を強制。実験タグだけの capability を default 手順から参照していると、通常デプロイが壊れる。
- 案 B（フィルタ前検証・emit のみ間引き）: 通常デプロイは通るが、実行時に欠落スキル参照が残る可能性。

**推奨は案 A**だが、既存データで default 手順が test 専用 capability を参照していないか要確認。

### 3. `prompt update` への適用有無

`--tags` を update にも付けるか未決。付けないと「compile/deploy と update で対象集合が食い違う」リスクがある。

### 4. Digest にタグを含めるか

推奨は「全ソース digest・タグは含めない」。ただし同一ソースで `--tags default` と `--tags test` を連続デプロイすると、digest 一致で 2 回目がスキップされうる。

- **対策案**: digest キーに RequestedTags を含める / またはタグ付き deploy では常に `--force` 相当の注意を文書化。

### 5. 空配列 `tags: []` の扱い

本仕様では「明示空＝どの集合にも入らない」とした。誤りやすいため、バリデーションで禁止（少なくとも 1 要素必須）する方が安全かもしれない。

### 6. 名前衝突: bundle `default` とタグ `default`

利用者ドキュメントで混同しないよう、用語を明示する必要がある。タグ名 `default` 自体の予約は維持してよいが、別名（例: `baseline`）への変更も議論余地あり。

### 7. プロジェクト作業ツリーへの明示 `tags: [default]`

カタログのみ明示、プロジェクト側は暗黙に任せる方針で後方互換は足りる。ただし「カタログと実リポジトリの見た目差」が残る。プロジェクト側も一括明示するか。

### 8. タグ名の制約強度

kebab-case を schema pattern で強制するか、自由文字列＋トリムのみか。強制する方が CLI やドキュメントが安定する。

### 9. Resolved Manifest の出力内容

タグ除外エンティティを `manifest.resolved.yaml` から落とすか、全件載せつつ `selected: false` のような印を付けるか。デバッグ容易性では後者が有利だが、フォーマット変更が大きい。

### 10. `--tags` の空白・重複

`--tags "default, test"` や `--tags default,default` をどう正規化するか（トリム必須、重複除去推奨）。

---

上記 1〜10 のうち、特に **1（フィールド転用）・2（参照整合）・4（digest）・5（空配列）・7（プロジェクト明示）** は挙動に直結するため、レビューでの方針決定を依頼する。
