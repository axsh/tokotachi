# devctl doctor フィーチャーチェック項目の修正

## 背景 (Background)

`devctl doctor` サブコマンドの初期実装では、各フィーチャーディレクトリに対して以下のチェックを行っていた:

1. `feature.yaml` の存在・パース → **FAIL** (存在しない場合)
2. `.devcontainer/devcontainer.json` の存在・パース → WARN
3. `go.mod` の存在 → WARN

しかし、実際の使用状況を踏まえると、これらのチェック項目には問題がある:

- **`feature.yaml`**: パッケージング（ビルド・リリース）のために使用されるファイルであり、`devctl` の日常的な開発環境操作（`up`, `down`, `open` 等）には必要ない。存在しないことが FAIL であるべきではない。
- **`go.mod`**: フィーチャーは多言語対応を前提としており、Go言語に限定したチェックは不適切。Go以外の言語（Python, TypeScript等）のフィーチャーでは `go.mod` は存在しない。

## 要件 (Requirements)

### 必須要件

1. **`feature.yaml` チェックを WARN に変更**
   - `feature.yaml` が存在しない場合のステータスを FAIL → WARN に変更する。
   - メッセージ: パッケージング時に必要だが、devctl動作には影響しない旨を明記する。

2. **`go.mod` チェックを削除**
   - Go言語固有のチェックであるため、フィーチャーチェックから完全に削除する。
   - フィーチャーの言語はプロジェクトによって異なり、特定言語のチェックは `doctor` の責務外。

### 任意要件

3. **統合テストの修正**
   - 上記変更に伴い、統合テスト `TestDevctlDoctorBasic` を引数なし（`--feature` 指定なし）で実行できるように戻す。`feature.yaml` が存在しないフィーチャーがあっても FAIL にならないため、終了コード 0 が期待される。

## 実現方針 (Implementation Approach)

### 変更対象

- `features/devctl/internal/doctor/checks.go`: `checkFeature` 関数の修正
- `features/devctl/internal/doctor/checks_test.go`: テストケースの更新
- `tests/integration-test/devctl_doctor_test.go`: 統合テストの修正

### 変更内容

1. `checkFeature` 内の `feature.yaml` チェック:
   - 存在しない場合: `StatusFail` → `StatusWarn` に変更
   - メッセージを更新: devctl動作には影響しない旨を明記

2. `checkFeature` 内の `go.mod` チェック:
   - ブロック全体を削除

3. 統合テスト `TestDevctlDoctorBasic`:
   - `--feature devctl` を外し、引数なし `devctl doctor` で実行
   - 終了コード 0 を期待

## 検証シナリオ (Verification Scenarios)

1. `features/integration-test/` には `feature.yaml` が存在しないが、`devctl doctor` を引数なしで実行しても FAIL にならない（WARN のみ）。終了コードは 0。
2. `devctl doctor --feature devctl` を実行すると、出力に `go.mod` のチェック行が含まれない。
3. `devctl doctor` のテキスト出力で、`feature.yaml` 不存在が ⚠️ WARN で表示される。

## テスト項目 (Testing for the Requirements)

| 要件 | テスト | 検証コマンド |
|---|---|---|
| feature.yaml WARN化 | `TestCheckFeature/feature.yaml_missing_is_warn` (単体テスト) | `scripts/process/build.sh` |
| go.mod チェック削除 | `TestCheckFeature` に go.mod 関連テストが存在しないこと | `scripts/process/build.sh` |
| 引数なし doctor 実行 | `TestDevctlDoctorBasic` (統合テスト) | `scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"` |
