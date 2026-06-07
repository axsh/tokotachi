# github-upload パイプラインの修正

## 背景 (Background)

`github-upload.sh` の使用時に2つの問題が確認された。

### 問題1: 引数の誤解釈

`github-upload.sh` は `<tool-id> [version|+increment]` の順で引数を受け取る。
しかし、ユーザーが tool-id を指定し忘れてバージョンだけ渡すと、**version が tool-id として誤解釈**され、分かりにくいエラーが発生する:

```bash
$ ./scripts/dist/github-upload.sh v0.2.0

# 内部では TOOL_ID="v0.2.0" と解釈される
# → "Manifest not found: .../tools/manifests/v0.2.0.yaml"
# → バージョンも v0.0.1 と表示される（意図は v0.2.0）
```

根本原因は、第1引数がバージョン形式であっても tool-id として無条件に受け入れてしまうこと。
ユーザーが tool-id を明示的に指定するのは正しい設計であり、**省略ではなく、誤入力時に明確なエラーで案内する**ことが必要。

### 問題2: クロスコンパイルビルドエラー

正しい引数 `./scripts/dist/github-upload.sh tt v0.2.0` で実行しても、**Windows以外のプラットフォーム向けビルドが全て失敗**する:

```
# linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 → 全て失敗
internal\codestatus\bgrunner.go:77:6: undefined: isProcessAlive
internal\codestatus\bgrunner.go:121:3: unknown field CreationFlags in struct literal of type "syscall".SysProcAttr

# windows/amd64 → 成功
```

原因は `features/tt/internal/codestatus/` パッケージにある:

1. **`isProcessAlive` 関数**: `process_windows.go` にのみ定義されており、Linux/macOS用の対応ファイルが存在しない
2. **`bgrunner.go` の `StartBackground` 関数**: `syscall.SysProcAttr{CreationFlags: ...}` を使用しているが、`CreationFlags` はWindows専用フィールド。このコードがビルドタグなしの共通ファイルに存在するため、非Windows環境でコンパイルエラーとなる

## 要件 (Requirements)

### 必須要件

#### 引数バリデーション強化

1. **tool-id のバリデーション**: 第1引数がバージョン形式（`vN.N.N` や `+vN.N.N`）の場合、tool-id ではないと判断し、分かりやすいエラーメッセージを表示して終了する
   - エラーメッセージには正しい使い方と、利用可能な tool-id の一覧を含める
2. **`tools.yaml` に存在する tool-id かチェック**: 第1引数が `tools.yaml` に登録されていない tool-id の場合もエラーとする
3. **対象スクリプト**: `github-upload.sh` だけでなく、tool-id を受け取る他のdistスクリプト（`build.sh`, `release.sh`, `publish.sh`）にも同様のバリデーションを適用し、一貫性を保つ

#### クロスコンパイル対応

4. **プラットフォーム別ファイル分離**: `bgrunner.go` 内のWindows専用コードをプラットフォーム別ファイルに分離し、全プラットフォームでビルドが成功するようにする
5. **Linux/macOS 用の `isProcessAlive` 実装**: `process_unix.go`（`//go:build !windows`）を新規作成し、Unix系OSでのプロセス存在確認を実装する
6. **プラットフォーム別プロセス分離処理**: `StartBackground` 内の `SysProcAttr` 設定をプラットフォーム別のヘルパー関数に切り出す

## 実現方針 (Implementation Approach)

### 引数バリデーション: `_lib.sh` に `validate_tool_id` 関数を追加

```bash
validate_tool_id() {
  local arg="$1"
  local available
  available=$(get_all_tool_ids)

  # バージョン形式チェック
  if [[ "$arg" =~ ^(\+)?v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    fail "First argument looks like a version, not a tool-id: '${arg}'"
    echo "  Usage: $0 <tool-id> [version|+increment]"
    echo "  Available tools: $(echo "$available" | tr '\n' ', ' | sed 's/,$//')"
    exit 1
  fi

  # tools.yaml 存在チェック
  if ! echo "$available" | grep -qx "$arg"; then
    fail "Unknown tool-id: '${arg}'"
    echo "  Available tools: $(echo "$available" | tr '\n' ', ' | sed 's/,$//')"
    exit 1
  fi
}
```

各スクリプトで `TOOL_ID="$1"` の直後に `validate_tool_id "$TOOL_ID"` を追加する。

### クロスコンパイル対応: プラットフォーム別ファイル分離

共通の `bgrunner.go` からプラットフォーム依存コードを分離する:

#### `process_windows.go`（既存、修正）

- `isProcessAlive` 関数を維持（`tasklist` コマンド使用）
- `detachSysProcAttr` 関数を追加: `&syscall.SysProcAttr{CreationFlags: 0x00000200}` を返す

#### `process_unix.go`（新規作成、`//go:build !windows`）

- `isProcessAlive` 関数: `os.FindProcess` + `Signal(0)` でプロセス存在を確認
- `detachSysProcAttr` 関数: `&syscall.SysProcAttr{Setsid: true}` を返す（Unix系でのプロセス分離）

#### `bgrunner.go`（修正）

- `syscall` パッケージのインポートを削除
- `StartBackground` 内の `cmd.SysProcAttr = &syscall.SysProcAttr{...}` を `cmd.SysProcAttr = detachSysProcAttr()` に置き換え

### 変更対象ファイル

| ファイル | 変更内容 |
|---------|---------|
| `scripts/dist/_lib.sh` | `validate_tool_id` 関数の追加 |
| `scripts/dist/github-upload.sh` | `validate_tool_id` 呼び出し追加 |
| `scripts/dist/build.sh` | 同上 |
| `scripts/dist/release.sh` | 同上 |
| `scripts/dist/publish.sh` | 同上 |
| `features/tt/internal/codestatus/bgrunner.go` | Windows専用コードの削除、`detachSysProcAttr()` への置き換え |
| `features/tt/internal/codestatus/process_windows.go` | `detachSysProcAttr` 関数の追加 |
| `features/tt/internal/codestatus/process_unix.go` | **[NEW]** `isProcessAlive` + `detachSysProcAttr` のUnix実装 |

## 検証シナリオ (Verification Scenarios)

### 引数バリデーション

1. `./scripts/dist/github-upload.sh v0.2.0` を実行 → バージョン形式のエラーメッセージが表示され、利用可能なtool一覧が案内される
2. `./scripts/dist/github-upload.sh unknown-tool` を実行 → 未知のtool-idエラーが表示される
3. `./scripts/dist/github-upload.sh tt v0.2.0` を実行 → 従来通り正常に動作する
4. `./scripts/dist/build.sh v0.2.0` を実行 → 同様のエラーメッセージが表示される

### クロスコンパイル

5. `./scripts/dist/build.sh tt` を実行 → 5/5 プラットフォーム全てのビルドが成功する
6. `go build ./...` を `features/tt/` で実行（ローカルOS向け） → 成功する

## テスト項目 (Testing for the Requirements)

### 自動化された検証

#### ビルド・単体テスト

```bash
./scripts/process/build.sh
```

#### クロスコンパイル確認

```bash
./scripts/dist/build.sh tt
# 期待: All 5 builds succeeded.
```

#### バリデーション動作確認

```bash
# バージョン形式を tool-id として渡す → エラー
./scripts/dist/github-upload.sh v0.2.0
# 期待: [FAIL] First argument looks like a version, not a tool-id: 'v0.2.0'

# 未知の tool-id → エラー
./scripts/dist/github-upload.sh unknown-tool
# 期待: [FAIL] Unknown tool-id: 'unknown-tool'
```

> [!NOTE]
> `github-upload.sh` は実際にGitHub APIを呼び出すため、バリデーション以降の動作は手動で確認する。
