# Manifest タグによるコンパイル/デプロイ対象の取捨選択

## 背景

本プロジェクトの prompt システムでは、`prompts/manifest/code_content/` 配下の Manifest（capability / policy / procedure / refs）をコンパイル・デプロイして、各 AI ツール向けの rules / skills 等へ展開している。現状、どの Manifest を作業対象にするかの制御は次に限られる。

- `project.yaml` の glob による取り込み
- `kind: skip` による emit 除外
- `target.includes` による kind 単位の on/off

一方で、「試験用の capability だけを一時的にデプロイしたい」「本番用の baseline セットと実験用セットを同居させたい」といった、**エンティティ単位のラベルによる取捨選択**はできない。

また、policy / skip スキーマには既に `tags` フィールド（Classification tags）が定義されているが、**実行時パイプラインには未接続であり、実 Manifest でも未使用**である。本仕様では旧 Classification 意味を廃止し、`tags` を **コンパイル/デプロイ対象の選択タグ** として再定義し、code_content 全体へ横断的に導入する。

一般利用者向けのカタログテンプレート（`catalog/originals/axsh/go-standard-project/`）も同期して改修し、新規スキャフォールドから同じタグ機構を使えるようにする。

## 要件

### 必須要件

1. **タグフィールドの導入**
   - `prompts/manifest/code_content/` 配下の Manifest（capability / policy / procedure / skip）の Frontmatter に `tags` を定義できること。
   - 対象スキーマ（プロジェクト側およびカタログ側の両方）:
     - `capability.schema.json`
     - `policy.schema.json`（既存の Classification 用 `tags` を廃止し、本仕様の選択タグへ置き換え）
     - `procedure.schema.json`
     - `skip.schema.json`（同上）
   - `safety/`（guard / worker / bundle）および `targets/` はタグ対象外とする。
   - `prompts/memory/` 配下（memory_docs / branch skills）はタグの影響を受けず、**常に処理対象**とする。

2. **暗黙の `baseline` タグ**
   - Frontmatter に `tags` が無い（フィールド省略）場合、その Manifest は暗黙的に `baseline` タグを持つものとして扱う。
   - Frontmatter に `tags` が明示されている場合、その配列がタグ集合の唯一の定義となる。
     - 例: `tags: [test]` は `test` のみを持ち、`baseline` は持たない。
     - `baseline` も付けたい場合は `tags: [baseline, test]` のように明示する。
   - 予約タグ名は `baseline` とする（bundle `id: default` との混同を避けるため、`default` は使わない）。

3. **CLI オプション `--tags`（compile / deploy / update で一貫）**
   - `prompt compile` / `prompt deploy` / `prompt update` のすべてに `--tags` オプションを追加する。
   - 書式: `--tags <tag>[,<tag>...]`（カンマ区切り、複数指定可）。
   - **OR 条件**: 指定タグのいずれか 1 つ以上を持つ Manifest を対象とする（AND ではない）。
   - `--tags` を省略した場合は `--tags baseline` と同等に扱う。
   - ラッパースクリプト `scripts/code/prompt/compile.sh` / `deploy.sh` / `update.sh` 経由でも同じフラグが透過されること（既存どおり `"$@"` 転送）。

4. **環境変数 `TT_TAGS`**
   - `--tags` 未指定時のフォールバックとして `TT_TAGS` を参照する。
   - 優先順位: CLI `--tags` > `TT_TAGS` > 暗黙 `baseline`。
   - 値の書式・正規化ルールは `--tags` と同一とする。

5. **フィルタ適用範囲**
   - タグで取捨選択されるのは **code_content 由来の Manifest エンティティのみ**。
   - memory、safety、targets は `--tags` の値に関わらず従来どおり処理する。

6. **後方互換**
   - 既存の `tags` 無し Manifest は、暗黙 `baseline` により、従来どおり `--tags` 省略時のコンパイル/デプロイ対象であり続けること。
   - 既存のコンパイル/デプロイ呼び出し（`--tags` 無し）の結果が、タグ導入前と実質同等であること（code_content がすべて暗黙または明示 `baseline` の場合）。

7. **カタログおよびプロジェクト作業ツリーへの明示タグ**
   - 次の両方の `code_content/` 配下全 Manifest に `tags: [baseline]` を明示する。
     - `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/`
     - `prompts/manifest/code_content/`（本プロジェクト作業ツリー）
   - カタログおよびプロジェクトの `schemas/` も同様に `tags` を定義する。

8. **タグ値の正規化とバリデーション**
   - Frontmatter の `tags: test`（スカラー）と `tags: [test]`（配列）の両方を受け付ける。
   - スカラーは内部的に 1 要素配列へ正規化する。
   - 空配列 `tags: []` は **バリデーションエラー**（少なくとも 1 要素必須）。
   - タグ名は kebab-case を **強制**する（pattern: `^[a-z0-9]+(-[a-z0-9]+)*$`）。
   - CLI / `TT_TAGS` の正規化:
     - カンマ区切り各要素をトリムする（例: `--tags "baseline, test"` → `baseline`, `test`）。
     - 重複はエラーにせず自動排除し、Warning を表示する（例: `--tags baseline,baseline`）。

9. **参照整合性モード（`--tag-refs`）**
   - `uses_capabilities` 等の参照は、タグ選択と独立して扱えるようにする。
   - CLI オプション `--tag-refs <mode>`（compile / deploy / update 共通）:
     - `include`（**デフォルト**）: タグ条件で選択されたエンティティが参照する先は、参照先自身のタグに関わらず選択集合へ追加する（必要なら参照閉包を辿る）。
     - `strict`: 参照先が RequestedTags と交差しない場合はエラーとする。
   - 環境変数 `TT_TAG_REFS` を同様にサポートする（優先: CLI > env > `include`）。

10. **Resolved Manifest の出力**
    - `manifest.resolved.yaml` にはフィルタ前の全エンティティを掲載する。
    - 各エンティティに選択結果を示す印（例: `selected: true|false`）を付与する。
    - emit / デプロイ対象は `selected: true` のものに限定する。
    - memory / safety / targets は常に `selected: true` 相当として扱う。

11. **Digest への RequestedTags 取り込み**
    - `deploy` の digest 計算に RequestedTags（正規化・重複排除・ソート後）を含める。
    - ソース内容が同一でも、異なるタグ集合でのデプロイは別 digest として扱い、意図しないスキップを防ぐ。
    - ソース側は従来どおり全 sources glob（タグ除外分も含む）を対象とする。

## 実現方針

### タグ解決ルール（Effective Tags）

各 code_content エンティティについて、実効タグ集合 `EffectiveTags` を次で定める。

```
if frontmatter に tags キーが存在しない:
    EffectiveTags = {"baseline"}
else:
    # tags: [] はスキーマ／バリデーションで拒否
    EffectiveTags = set(正規化後の tags 配列)
    # tags: test → {"test"}
    # tags: [baseline, test] → {"baseline", "test"}
```

### 選択ルール（Selection）

```
RequestedTags = parse(--tags | TT_TAGS)  # どちらも無い場合は ["baseline"]
TagMatched = EffectiveTags ∩ RequestedTags ≠ ∅

--tag-refs=include (default):
  selected = TagMatched
             OR (TagMatched なエンティティから参照閉包で到達可能)

--tag-refs=strict:
  selected = TagMatched
  かつ、selected エンティティの参照先がすべて TagMatched でなければエラー
```

memory / safety / targets はこの判定を行わず常に selected。

### フィルタ適用タイミング

1. 全エンティティをパース・スキーマ検証・ID 一意性検証する（タグで不正を隠さない）。
2. Effective Tags と RequestedTags から初期選択集合を算出する。
3. `--tag-refs` に従い参照閉包の追加、または strict 検証を行う。
4. Resolved Manifest に全件を書き出し、各エンティティへ `selected` を付与する。
5. Emit は `selected: true` のみを対象とする。

### スキーマ変更

`tags` プロパティを次の形で統一する（capability / procedure は新規追加、policy / skip は Classification 意味を廃止して置き換え）:

```json
"tags": {
  "description": "Selection tags for compile/deploy/update filtering. If omitted, the entity implicitly has the 'baseline' tag. Explicit tags replace the implicit baseline; include 'baseline' explicitly when needed. Empty arrays are forbidden.",
  "oneOf": [
    {
      "type": "string",
      "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
      "minLength": 1
    },
    {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "string",
        "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
        "minLength": 1
      }
    }
  ]
}
```

タグ名の kebab-case は schema pattern で強制する。

### CLI / 実装配置

| 層 | 変更概要 |
|---|---|
| `features/tt/cmd/prompt.go` | `compile` / `deploy` / `update` に `--tags` と `--tag-refs` を追加。`TT_TAGS` / `TT_TAG_REFS` を解決 |
| `features/tt/internal/prompt/compiler` | Options に Tags / TagRefs を渡し、選択・digest・emit に接続 |
| `features/tt/internal/prompt/manifest` | Effective Tags、選択、参照モード、`selected` 付与、パース正規化 |
| `prompts/manifest/schemas/*.schema.json` | `tags` 追加・更新（旧 Classification 記述を廃止） |
| `catalog/.../schemas/*.schema.json` | 同上 |
| `catalog/.../code_content/**/*.md` | 全ファイルに `tags: [baseline]` を明示 |
| `prompts/manifest/code_content/**/*.md` | 同上（プロジェクト作業ツリー） |

### Digest との関係

```
digest_input = hash(
  全 sources glob の内容,
  正規化・重複排除・ソート済み RequestedTags,
  TagRefs mode  # include/strict の切替でも出力集合が変わるため含める
)
```

同一ソースでも `--tags baseline` と `--tags test` は別 digest となり、連続デプロイで誤スキップしない。

### 用語の整理

| 名称 | 意味 | 本仕様のタグとの関係 |
|---|---|---|
| タグ `baseline` | 暗黙／明示の標準選択ラベル | 本仕様の中心概念（旧称案 `default` から変更） |
| bundle `id: default` | safety bundle 名 | **無関係**（名前衝突回避のためタグ側を `baseline` にした） |
| `kind: skip` | emit しないエンティティ種別 | 併存。skip もタグ付け可能だが emit されない点は従来どおり |
| 旧 policy.`tags`（Classification） | スキーマのみ・実行時未使用・実ファイル未使用 | **廃止**し、選択タグへ置き換え |

## 検証シナリオ

1. **暗黙 baseline（後方互換）**
   1. `tags` 無しの capability を用意する。
   2. `--tags` 無しで `prompt compile --dry-run` を実行する。
   3. 当該 capability が resolved manifest で `selected: true` となり、emit 対象になることを確認する。

2. **明示 test のみ**
   1. capability の Frontmatter に `tags: test`（または `tags: [test]`）を付ける。
   2. `--tags` 無し（＝暗黙 `baseline`）で compile すると、当該ファイルは `selected: false`（参照で引き込まれない限り）になる。
   3. `--tags test` で compile すると `selected: true` になる。
   4. memory_docs および safety / targets はどちらの実行でも従来どおり処理される。

3. **複数タグ OR**
   1. エンティティ A: `tags: [baseline]`、B: `tags: [test]`、C: `tags: [baseline, test]`、D: `tags` 省略（暗黙 baseline）。
   2. `--tags baseline,test` で A,B,C,D すべてがタグ条件で選択される。
   3. `--tags test` で B,C のみがタグ条件で選択される（A,D はタグ非一致）。

4. **参照モード include（デフォルト）**
   1. procedure P（`tags: [baseline]`）が capability X（`tags: [test]`）を `uses_capabilities` で参照する。
   2. `--tags baseline`（`--tag-refs` 省略＝include）で compile すると、P と X の両方が `selected: true` になる。

5. **参照モード strict**
   1. 同上の P → X 参照がある状態で `--tags baseline --tag-refs strict` を実行する。
   2. X が RequestedTags と交差しないためエラーで停止する。

6. **カタログ／プロジェクト明示 baseline**
   1. カタログおよび `prompts/manifest/code_content/` 配下の全 md に `tags: [baseline]` が明示されていることを確認する。
   2. `--tags` 省略時にそれらが対象になる。

7. **スカラー／配列・CLI 正規化**
   1. `tags: test` と `tags: [test]` が同一の Effective Tags になる。
   2. `--tags "baseline, test"` は `baseline` と `test` にトリムされる。
   3. `--tags baseline,baseline` は重複排除され Warning が出る。
   4. `tags: []` はバリデーションエラーになる。

8. **Digest にタグが効く**
   1. 同一ソースで `--tags baseline` の deploy 後、`--tags test` の deploy を行う。
   2. digest 不一致により 2 回目がスキップされないことを確認する。

9. **update 一貫性**
   1. `prompt update --tags test` が compile/deploy と同じ選択規則で動作することを確認する。

## テスト項目

### 単体テスト（TDD）

`features/tt/internal/prompt/manifest/` および `compiler/` にテーブル駆動テストを追加する。

| ケース | 期待 |
|---|---|
| tags 省略 | EffectiveTags = {baseline}、`--tags` 省略で選択される |
| `tags: [test]` | {test}、`--tags baseline` で非選択（参照無し時）、`--tags test` で選択 |
| `tags: [baseline, test]` | 両方の指定で選択 |
| `tags: test`（スカラー） | `[test]` と同等 |
| `tags: []` | バリデーションエラー |
| 不正タグ名 `Foo` / `a_b` | バリデーションエラー |
| memory / safety | `--tags test` でも常に selected |
| OR 条件 | `--tags baseline,test` でいずれか一致すれば選択 |
| `--tag-refs include` | タグ非一致でも参照先は selected |
| `--tag-refs strict` | タグ非一致の参照先があればエラー |
| CLI トリム・重複 | `"baseline, test"` → 2 要素、重複は Warning 付きで 1 要素 |
| Digest | RequestedTags / TagRefs の違いで digest が変わる |

CLI フラグ解析（`features/tt/cmd/prompt_test.go`）:

- `--tags` 省略かつ `TT_TAGS` 無し → 内部で `baseline`
- `TT_TAGS=test` かつ `--tags` 無し → `["test"]`
- `--tags baseline,test` → `["baseline","test"]`
- `--tag-refs` 省略 → `include`

### ビルド・全体検証

1. ビルド＋単体テスト:
   `scripts/process/build.sh`

2. prompt 経路の統合テスト（パス／compile のリグレッション）:
   `scripts/process/integration_test.sh --categories "tt" --specify "PromptPaths|Prompt"`

3. （実装後に追加する）タグ選択・参照モード・digest の統合テストを同カテゴリへ追加し、上記 `--specify` に含める。

## レビュー決定事項

本節はレビューで確定した方針の記録である（旧「検討事項」の解決結果）。

| # | 論点 | 決定 |
|---|---|---|
| 1 | 既存 `policy.tags` / `skip.tags` | 実行時・実ファイルとも未使用を確認。旧 Classification 意味は廃止し、選択タグへ置き換え（案 a） |
| 2 | 参照整合性 | `--tag-refs include\|strict` を導入。デフォルトは `include`（参照先をタグに関係なく取り込む）。`strict` は参照先不一致をエラー |
| 3 | `prompt update` | `--tags` / `--tag-refs` を compile/deploy と一貫して必須対応 |
| 4 | Digest | 正規化済み RequestedTags（および TagRefs mode）を digest に含める |
| 5 | `tags: []` | バリデーションで禁止（minItems: 1） |
| 6 | 暗黙タグ名 | `default` ではなく `baseline` を採用（bundle `default` との衝突回避） |
| 7 | プロジェクト明示タグ | `prompts/manifest/code_content/` にも `tags: [baseline]` を明示 |
| 8 | タグ名制約 | kebab-case を schema で強制 |
| 9 | Resolved 出力 | 全件掲載＋ `selected: true\|false` |
| 10 | CLI 空白・重複 | トリム必須。重複は Warning 付きで自動排除 |
| — | `TT_TAGS` / プロジェクト明示 / update | いずれも必須要件へ昇格 |
