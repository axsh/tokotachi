# .devrc.yaml の完全削除

## 背景

`.devrc.yaml` はプロジェクトルートに配置するグローバル設定ファイルとして設計されたが、実際にはプロジェクト内にファイルが存在しておらず、使用されていない。コード内には `LoadGlobalConfig` 関数が残っており、ファイルが見つからない場合はデフォルト値を返す実装となっているため、事実上ハードコードされたデフォルト値でのみ動作している。

不要なコードを残しておくとメンテナンスコストが増加し、新規メンバーに混乱を招くため、関連コードをすべて削除する。

## 要件

### 必須

1. `.devrc.yaml` を参照するすべてのGoコードを削除する
2. `GlobalConfig` 構造体の `LoadGlobalConfig` 関数を削除する
3. `GlobalConfig` から取得していた設定値（`DefaultEditor`, `DefaultContainerMode`, `ProjectName`, `WorkDir`）はハードコードのデフォルト値に置き換える
4. `tt doctor` の `.devrc.yaml` チェック機能を削除する
5. READMEの `.devrc.yaml` に関するConfigurationセクションを削除する
6. 関連するテストコードを削除・修正する
7. 既存のビルドとテストが通ること

### デフォルト値の置き換え

| 設定項目 | デフォルト値 | 使用箇所 |
|----------|-------------|----------|
| `DefaultEditor` | `"cursor"` | `features/tt/cmd/common.go` の `ResolveEditor` 呼び出し |
| `DefaultContainerMode` | `"docker-local"` | `features/tt/cmd/common.go`, `tokotachi.go` |
| `ProjectName` | `"tt"` | `tokotachi.go` の `resolveProjectName` |
| `WorkDir` | `"work"` | 使用箇所を要確認（他の箇所で既にハードコード済みの可能性あり） |

## 実現方針

### 削除対象ファイル・関数の一覧

#### `pkg/resolve/config.go`
- `GlobalConfig` 構造体を削除
- `LoadGlobalConfig` 関数を削除
- `FeatureConfig` と `LoadFeatureConfig` は残す（`feature.yaml` は引き続き使用）

#### `pkg/resolve/config_test.go`
- `LoadGlobalConfig` のテストを削除
- `LoadFeatureConfig` のテストは残す

#### `features/tt/cmd/common.go`
- `ResolveEnvironment` メソッド内の `LoadGlobalConfig` 呼び出しを削除
- `globalCfg.DefaultEditor` → リテラル `"cursor"` に置き換え
- `globalCfg.DefaultContainerMode` → リテラル `"docker-local"` に置き換え

#### `tokotachi.go`
- `resolveProjectName` メソッド → 常に `"tt"` を返すように簡略化
- `Up` / `Open` メソッド内の `LoadGlobalConfig` 呼び出しを削除
- `containerMode` → デフォルト値 `"docker-local"` に置き換え

#### `pkg/doctor/checks.go`
- `checkGlobalConfig` 関数を削除
- `fixGlobalConfig` 関数を削除
- `categoryConfig` 定数を削除

#### `pkg/doctor/doctor.go`
- `.devrc.yaml` の fix 判定ロジックを削除

#### `pkg/doctor/checks_test.go`, `pkg/doctor/doctor_test.go`, `pkg/doctor/result_test.go`
- `.devrc.yaml` 関連テストケースを削除・修正

#### `pkg/scaffold/catalog_test.go`
- `.devrc.yaml` を参照するテストデータを修正

#### `pkg/detect/editor.go`
- コメントのみ修正

#### `README.md`
- Configurationセクションの `.devrc.yaml` 関連部分を削除

## 検証シナリオ

1. `pkg/resolve/config.go` から `GlobalConfig` と `LoadGlobalConfig` が削除されていること
2. `grep -r "devrc" .` で `.devrc` への参照が0件であること
3. `features/tt/cmd/common.go` の `ResolveEnvironment` がデフォルト値で正しく動作すること
4. `tokotachi.go` の `resolveProjectName` が常に `"tt"` を返すこと
5. `tt doctor` が `.devrc.yaml` をチェックしないこと

## テスト項目

| 要件 | 検証方法 |
|------|----------|
| コード削除後にビルドが通る | `scripts/process/build.sh` |
| 既存テストがパスする | `scripts/process/build.sh` (含: 単体テスト) |
| `devrc` への参照が残っていないこと | `grep -r "devrc" --include="*.go" .` |
