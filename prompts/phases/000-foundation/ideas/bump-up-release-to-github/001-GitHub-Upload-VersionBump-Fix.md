# GitHub Upload スクリプト: バージョンバンプ修正

## 背景 (Background)

`scripts/dist/github-upload.sh` は、ツールのビルド・リリース・パブリッシュを一括で行うスクリプトである。
現在の引数仕様は以下の通り:

```
Usage: ./scripts/dist/github-upload.sh <tool-id> [version|+increment]
```

**問題点**: ツールが1つしか登録されていない状況（`tools.yaml` に `devctl` のみ）で、以下のような省略記法が使えない:

```bash
# 現状: これはエラーになる
./scripts/dist/github-upload.sh +v0.1.0

# 期待: tool-id を省略して、+v0.1.0 だけで動いてほしい
```

### 発生するエラー

```
[INFO] Fetching current version for +v0.1.0...
[INFO] Current version: v0.0.0
...
[INFO] Tool:    +v0.1.0
[INFO] Current: v0.0.0
[INFO] New:     v0.0.1
...
[FAIL] Manifest not found: .../tools/manifests/+v0.1.0.yaml
```

### 原因分析

1. **引数の誤解釈**: `$1` が常に `TOOL_ID` として扱われるため、`+v0.1.0` が `TOOL_ID` にセットされる
2. **Current バージョンの取得失敗**: `get_current_version "+v0.1.0"` で GitHub Release を検索するが、`+v0.1.0-v` プレフィックスのタグは存在しないため `v0.0.0` にフォールバック
3. **New バージョンの計算ミス**: `VERSION_ARG` が `${2:-+v0.0.1}`（デフォルト値）になるため、`v0.0.0 + v0.0.1 = v0.0.1` と表示される（ユーザーが期待する `v0.1.0` ではない）

## 要件 (Requirements)

### 必須要件

1. **tool-id 省略時の自動判定**: `tools.yaml` にツールが1つだけ登録されている場合、第1引数をバージョン指定として解釈し、tool-id を自動的に解決する
2. **引数のスマート解析**: 第1引数がバージョンパターン（`vN.N.N` または `+vN.N.N`）にマッチする場合、それをバージョン指定として扱い、tool-id は自動検出する
3. **曖昧さの排除**: `tools.yaml` に複数ツールが登録されている場合に、tool-id を省略するとエラーメッセージで tool-id の指定を求める
4. **後方互換性**: 従来の `github-upload.sh devctl +v0.1.0` のような呼び出し方は引き続き動作すること
5. **`--dry-run` オプション**: 実際のビルド・リリース・パブリッシュを実行せず、引数解析結果とバージョン計算結果のみを表示して終了するモードを追加する
   - ビルド・リリース・パブリッシュの Step 1〜3 は実行しない
   - 引数解析、バージョン取得・計算、バージョン比較までを実行し、結果を表示する
   - 終了コード 0 で終了する（バージョン比較エラーの場合は非ゼロ）
   - `--dry-run` はどの位置でも指定可能（例: `github-upload.sh --dry-run +v0.1.0`, `github-upload.sh devctl --dry-run +v0.1.0`）

### 任意要件

6. **他のスクリプトへの波及**: `build.sh`, `release.sh`, `publish.sh` は tool-id が必須のままで良い（`github-upload.sh` のみ対応）

## 実現方針 (Implementation Approach)

### `github-upload.sh` の引数解析ロジック改修

現在の固定順序の引数解析:

```bash
TOOL_ID="$1"
VERSION_ARG="${2:-+v0.0.1}"
```

を、以下のスマート解析に変更する:

#### ステップ1: `--dry-run` フラグの抽出

引数リストから `--dry-run` を取り除き、フラグ変数にセットする:

```bash
DRY_RUN=false
ARGS=()
for arg in "$@"; do
  if [[ "$arg" == "--dry-run" ]]; then
    DRY_RUN=true
  else
    ARGS+=("$arg")
  fi
done
```

#### ステップ2: 残りの引数を位置引数として解析

```
ARGS が 0個の場合:
  → tool-id を自動解決、バージョンは +v0.0.1 (patch bump)

ARGS が 1個の場合:
  → 第1引数がバージョンパターンにマッチするか判定
    マッチする場合: tool-id を自動解決、第1引数をバージョンとして使用
    マッチしない場合: 第1引数を tool-id とし、バージョンは +v0.0.1

ARGS が 2個の場合:
  → 従来通り: $1 = tool-id, $2 = バージョン
```

### tool-id 自動検出の実装

`_lib.sh` の `get_all_tool_ids` 関数を利用して、登録されているツール数を確認する:

```bash
resolve_single_tool_id() {
  local ids
  ids=$(get_all_tool_ids)
  local count
  count=$(echo "$ids" | wc -l)
  
  if [[ $count -eq 1 ]]; then
    echo "$ids"
  else
    fail "Multiple tools registered. Please specify tool-id explicitly."
    echo "  Available tools:"
    echo "$ids" | sed 's/^/    - /'
    exit 1
  fi
}
```

### バージョンパターン判定

```bash
is_version_arg() {
  [[ "$1" =~ ^\+?v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}
```

### `--dry-run` モードの実装

バージョン計算・比較の後、`DRY_RUN=true` の場合は要約情報を表示して正常終了する:

```bash
if [[ "$DRY_RUN" == true ]]; then
  echo ""
  info "=== Dry Run ==="
  info "Tool:    ${TOOL_ID}"
  info "Current: ${CURRENT_VERSION}"
  info "New:     ${NEW_VERSION}"
  info "Mode:    ${MODE}"
  info "No changes were made."
  exit 0
fi
```

## 検証シナリオ (Verification Scenarios)

### シナリオ1: tool-id 省略 + インクリメントバージョン（`--dry-run`）

```bash
# 入力
./scripts/dist/github-upload.sh --dry-run +v0.1.0

# 期待される出力
# === Dry Run ===
# Tool:    devctl
# Current: (GitHub Releases から devctl の最新を取得)
# New:     (Current の minor を +1 した値)
# Mode:    increment
# No changes were made.
```

### シナリオ2: tool-id 省略 + 絶対バージョン（`--dry-run`）

```bash
# 入力
./scripts/dist/github-upload.sh --dry-run v1.0.0

# 期待される出力
# === Dry Run ===
# Tool:    devctl
# Current: (GitHub Releases から devctl の最新を取得)
# New:     v1.0.0
# Mode:    absolute
# No changes were made.
```

### シナリオ3: tool-id 省略 + 引数なし（デフォルト patch bump, `--dry-run`）

```bash
# 入力
./scripts/dist/github-upload.sh --dry-run

# 期待される出力
# === Dry Run ===
# Tool:    devctl
# Current: (GitHub Releases から devctl の最新を取得)
# New:     (Current の patch を +1 した値)
# Mode:    increment
# No changes were made.
```

### シナリオ4: 従来の明示的指定 + `--dry-run`（後方互換）

```bash
# 入力
./scripts/dist/github-upload.sh --dry-run devctl +v0.1.0

# 期待される出力
# === Dry Run ===
# Tool:    devctl
# Current: (GitHub Releases から devctl の最新を取得)
# New:     (Current の minor を +1 した値)
# Mode:    increment
# No changes were made.
```

### シナリオ5: `--dry-run` の位置自由度

```bash
# 以下はすべて同じ結果になること
./scripts/dist/github-upload.sh --dry-run +v0.1.0
./scripts/dist/github-upload.sh +v0.1.0 --dry-run
./scripts/dist/github-upload.sh devctl --dry-run +v0.1.0
```

### シナリオ6: 複数ツール登録時の省略エラー

`tools.yaml` に複数ツールが登録されている場合:

```bash
# 入力
./scripts/dist/github-upload.sh --dry-run +v0.1.0

# 期待される出力
# [FAIL] Multiple tools registered. Please specify tool-id explicitly.
#   Available tools:
#     - devctl
#     - other-tool
```

## テスト項目 (Testing for the Requirements)

### 自動テスト

`--dry-run` オプションを使った引数解析ロジックの検証。`--dry-run` は GitHub API（`gh release list`）へのアクセスは行うが、ビルド・リリース・パブリッシュは実行しない。

#### テスト1: tool-id 省略 + インクリメント

```bash
./scripts/dist/github-upload.sh --dry-run +v0.1.0
# 確認: Tool が devctl、Mode が increment、New が Current の minor +1
```

#### テスト2: tool-id 省略 + 絶対バージョン

```bash
./scripts/dist/github-upload.sh --dry-run v1.0.0
# 確認: Tool が devctl、Mode が absolute、New が v1.0.0
```

#### テスト3: 明示的 tool-id + インクリメント（後方互換）

```bash
./scripts/dist/github-upload.sh --dry-run devctl +v0.1.0
# 確認: テスト1 と同じ結果
```

#### テスト4: 引数なし（デフォルト patch bump）

```bash
./scripts/dist/github-upload.sh --dry-run
# 確認: Tool が devctl、Mode が increment、New が Current の patch +1
```

#### テスト5: ビルドパイプラインでの検証

修正後、既存のスクリプトが壊れていないことを確認:

```bash
scripts/process/build.sh
```
