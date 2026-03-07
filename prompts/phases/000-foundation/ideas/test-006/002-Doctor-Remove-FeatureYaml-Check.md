# devctl doctor: feature.yaml チェックの完全削除

## 背景 (Background)

前回の修正（001-Doctor-FeatureCheck-Fix）で `feature.yaml` チェックを FAIL→WARN に変更し、`go.mod` チェックを削除した。しかしユーザーの意図は「チェック自体をしない・表示もしない」であった。

`feature.yaml` はパッケージング専用のファイルであり、`devctl` の診断対象として不適切。doctor の出力に含めるべきではない。

## 要件 (Requirements)

### 必須要件

1. `checkFeature` 関数から `feature.yaml` チェックブロックを完全に削除する
2. `devctl doctor` の出力に `feature.yaml` に関する行が一切表示されないこと

## 実現方針 (Implementation Approach)

- `features/devctl/internal/doctor/checks.go`: `checkFeature` 内の feature.yaml チェックブロック全体を削除
- `features/devctl/internal/doctor/checks_test.go`: feature.yaml 関連テストケースを削除
- `features/devctl/internal/doctor/doctor_test.go`: feature.yaml 作成行があれば削除
- `tests/integration-test/devctl_doctor_test.go`: 必要に応じて更新

## 検証シナリオ (Verification Scenarios)

1. `devctl doctor` を実行し、出力に `feature.yaml` が含まれないこと
2. `devctl doctor --json` を実行し、JSON 出力に `feature.yaml` が含まれないこと

## テスト項目 (Testing for the Requirements)

| 要件 | 検証コマンド |
|---|---|
| feature.yaml チェック削除 | `scripts/process/build.sh` |
| 統合テスト通過 | `scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"` |
