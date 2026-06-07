# 017: 統合テスト TestScaffoldRootFlag の失敗修正（build.sh バイナリ出力先の不整合）

## 背景 (Background)

統合テスト `TestScaffoldRootFlag` が `unknown flag: --root` エラーで失敗している。

### 原因分析

根本原因は **`build.sh` のビルド出力ファイル名と統合テストのバイナリ検索ロジックの不整合** である。

1. **`build.sh`** ([build.sh:199](file://scripts/process/build.sh#L199)) は `bin/tt`（拡張子なし）にビルドする:
   ```bash
   go build -o "$PROJECT_ROOT/bin/tt" .
   ```

2. **統合テストの `ttBinary()`** ([helpers_test.go:42-48](file://tests/integration-test/helpers_test.go#L42-L48)) は `bin/tt.exe` を **優先的に** 検索する:
   ```go
   exePath := filepath.Join(projectRoot(), "bin", "tt.exe")
   if _, err := os.Stat(exePath); err == nil {
       return exePath  // .exe が存在すればそちらを使う
   }
   ```

3. `bin/tt.exe` は古いビルド（2026-03-13、`--cwd` フラグ版）のまま残っている
4. `bin/tt` は最新ビルド（2026-03-17、`--root` フラグ版）

結果として、統合テストは**古い `bin/tt.exe`** を使い、`--root` フラグが存在しないためエラーとなる。

> [!IMPORTANT]
> この問題のもう1つの側面として、`--root` フラグ（string型）と `--cwd` フラグ（bool型）のどちらが正しい設計なのかという点がある。現在の `features/tt/cmd/scaffold.go`（mainブランチ）は `--root` を定義しているが、`work/integration-test/` のコピーは `--cwd` を定義している。テスト `TestScaffoldRootFlag` は `--root` フラグの動作を検証しているため、`--root` が正しい仕様として扱う。

## 要件 (Requirements)

### 必須要件

1. **`build.sh` のWindows対応**: Windows環境では `bin/tt.exe` にビルドすること。これにより `ttBinary()` が常に最新バイナリを使用する
2. **古い `bin/tt.exe` の削除**: ビルドスクリプト内で古いバイナリが残らないようにする
3. **テスト `TestScaffoldRootFlag` の通過**: 修正後、`--root` フラグが正しく動作し、統合テストがパスすること

### 任意要件

- `ttBinary()` のロジック改善（`.exe` 優先ではなく、最新日時のバイナリを使う等）

## 実現方針 (Implementation Approach)

### 修正方針

`build.sh` に OS 判定を追加し、Windows 環境では `.exe` 拡張子を付けてビルドする。

```bash
# OS判定でバイナリ名を決定
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" || "$OSTYPE" == "win32" ]]; then
    go build -o "$PROJECT_ROOT/bin/tt.exe" .
else
    go build -o "$PROJECT_ROOT/bin/tt" .
fi
```

### 修正対象ファイル

| ファイル | 修正内容 |
|---|---|
| `scripts/process/build.sh` | `build_tt()` 関数内でOS判定を追加し、Windows時は `bin/tt.exe` にビルド |

## 検証シナリオ (Verification Scenarios)

1. `./scripts/process/build.sh` を実行する
2. `bin/tt.exe` が最新ビルドであることを確認する
3. `bin/tt.exe scaffold --help` の出力に `--root` フラグが含まれることを確認する
4. `./scripts/process/integration_test.sh --specify "TestScaffoldRootFlag"` がパスすることを確認する
5. 全統合テストでリグレッションがないことを確認する

## テスト項目 (Testing for the Requirements)

| 要件 | テスト方法 | 検証コマンド |
|---|---|---|
| ビルド出力先の修正 | ビルド後に `bin/tt.exe` が存在し最新であること | `./scripts/process/build.sh` |
| `--root` フラグの動作 | `TestScaffoldRootFlag` がパスすること | `./scripts/process/integration_test.sh --specify "TestScaffoldRootFlag"` |
| リグレッション確認 | 全統合テストのパス | `./scripts/process/integration_test.sh` |
