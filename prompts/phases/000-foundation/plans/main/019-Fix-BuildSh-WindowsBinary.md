# 019-Fix-BuildSh-WindowsBinary

> **Source Specification**: [017-Fix-BuildSh-WindowsBinary.md](file://prompts/phases/000-foundation/ideas/main/017-Fix-BuildSh-WindowsBinary.md)

## Goal Description

`build.sh` が Windows 環境で `bin/tt`（拡張子なし）にビルドするため、統合テストの `ttBinary()` が古い `bin/tt.exe` を使って失敗する問題を修正する。OS判定を追加し、Windows では `bin/tt.exe` にビルドするようにする。あわせて古い `bin/tt.exe` を削除する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req.1: build.sh の Windows 対応 | Proposed Changes > `build.sh` |
| Req.2: 古い bin/tt.exe の削除 | Step-by-Step > Step 1 (手動削除) |
| Req.3: TestScaffoldRootFlag の通過 | Verification Plan > Integration Tests |

## Proposed Changes

### ビルドスクリプト

#### [MODIFY] [build.sh](file://scripts/process/build.sh)

*   **Description**: `build_tt()` 関数内のビルド出力先をOS判定で切り替える
*   **Technical Design**:

    現在 (199行目):
    ```bash
    if go build -o "$PROJECT_ROOT/bin/tt" .; then
    ```

    修正後:
    ```bash
    # Determine binary output name based on OS
    local binary_name="tt"
    if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" || "$OSTYPE" == "win32" ]]; then
        binary_name="tt.exe"
    fi

    info "Building tt..."
    if go build -o "$PROJECT_ROOT/bin/$binary_name" .; then
    ```

*   **Logic**:
    1. `$OSTYPE` 環境変数でOS判定（Git Bash は `msys`、Cygwin は `cygwin`）
    2. Windows 環境なら `binary_name` を `tt.exe` に設定
    3. `go build -o` の出力先に `$binary_name` を使用
    4. これにより `ttBinary()` が `.exe` を優先検索するロジックと整合する

## Step-by-Step Implementation Guide

- [x] **Step 1: 古い bin/tt.exe の手動削除**
    *   `bin/tt.exe`（2026-03-13 ビルド）を削除する
    *   コマンド: `rm -f bin/tt.exe`

- [x] **Step 2: build.sh の修正**
    *   `scripts/process/build.sh` の `build_tt()` 関数を修正
    *   199行目の `go build -o` の出力先にOS判定を追加

- [x] **Step 3: ビルドと単体テスト**
    *   `./scripts/process/build.sh` を実行
    *   Windows環境で `bin/tt.exe` が生成されることを確認
    *   全単体テストがパスすることを確認

- [x] **Step 4: 統合テスト (TestScaffoldRootFlag)**
    *   `./scripts/process/integration_test.sh --specify "TestScaffoldRootFlag"` を実行
    *   テストがパスすることを確認

- [x] **Step 5: 全統合テスト** *(手動実行を推奨)*
    *   `./scripts/process/integration_test.sh` を実行してリグレッションがないことを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: `bin/tt.exe` が新たにビルドされ、`tt build succeeded.` が表示されること

2.  **Integration Tests (対象テスト)**:
    ```bash
    ./scripts/process/integration_test.sh --specify "TestScaffoldRootFlag"
    ```
    *   **Log Verification**: `PASS: TestScaffoldRootFlag` が出力されること

3.  **Integration Tests (全体)**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: 全カテゴリが `Passed` であること

## Documentation

本修正は内部ビルドスクリプトの修正であり、外部向けドキュメントの更新は不要。
