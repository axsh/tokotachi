# 001-ReservedBranch-Protection

> **Source Specification**: [001-ReservedBranch-Protection.md](file://prompts/phases/000-foundation/ideas/fix-nested-worktree-deletion/001-ReservedBranch-Protection.md)

## Goal Description

`main` および `master` を予約ブランチ名として定義し、ブランチ名を引数に取る全サブコマンド（`up`, `close`, `down`, `open`, `shell`, `exec`, `status`, `pr`）でこれらの指定を一律にエラーとする。`InitContext()` にバリデーションを集約し、各コマンド側の修正を不要にする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `main`, `master` を予約ブランチ名として定義 | Proposed Changes > `common.go` (`reservedBranchNames` 変数) |
| R2: 全コマンドで一律拒否 | Proposed Changes > `common.go` (`InitContext` 内呼び出し) |
| R3: わかりやすいエラーメッセージ | Proposed Changes > `common.go` (`validateBranchName` 関数) |
| R4: 大文字・小文字の完全一致 | Proposed Changes > `common.go` (`validateBranchName` — `==` 比較) |
| R5: 拡張容易な設計 | Proposed Changes > `common.go` (スライスで管理) |

## Proposed Changes

### cmd パッケージ

#### [MODIFY] [common_test.go](file://features/devctl/cmd/common_test.go)

*   **Description**: 予約ブランチ名バリデーションのユニットテストを追加
*   **Technical Design**:
    ```go
    // 追加するテスト関数
    func TestValidateBranchName(t *testing.T)
    func TestInitContext_ReservedBranch(t *testing.T)
    ```
*   **Logic**:
    *   `TestValidateBranchName`: テーブル駆動テストで以下ケースを検証
        | ケース名 | 入力 | 期待結果 |
        |---|---|---|
        | `main is reserved` | `"main"` | エラー（メッセージに `"main"` と `reserved` を含む） |
        | `master is reserved` | `"master"` | エラー（メッセージに `"master"` と `reserved` を含む） |
        | `normal branch` | `"my-feature"` | エラーなし |
        | `case sensitive Main` | `"Main"` | エラーなし |
        | `case sensitive MASTER` | `"MASTER"` | エラーなし |
        | `empty string` | `""` | エラーなし（空文字はこの関数では許容、`InitContext` 側で別途チェック） |
        | `main prefix` | `"main-feature"` | エラーなし（部分一致ではない） |
    *   `TestInitContext_ReservedBranch`: `InitContext([]string{"main"})` がエラーを返し、メッセージに `reserved` を含むことを検証

---

#### [MODIFY] [common.go](file://features/devctl/cmd/common.go)

*   **Description**: 予約ブランチ名リストとバリデーション関数を追加、`InitContext` にチェックを組み込む
*   **Technical Design**:
    ```go
    // パッケージレベル変数: 予約ブランチ名リスト
    var reservedBranchNames = []string{"main", "master"}

    // バリデーション関数
    func validateBranchName(branch string) error
    // branch が reservedBranchNames に含まれる場合エラーを返す
    // 比較は == (完全一致、大文字小文字区別あり)
    ```
*   **Logic**:
    1. `reservedBranchNames` をパッケージレベルの `var` で定義（将来の追加が容易）
    2. `validateBranchName(branch string) error`:
        *   `reservedBranchNames` をループし、`branch == name` で比較
        *   一致した場合: `fmt.Errorf("%q is a reserved branch name and cannot be used with devctl commands", branch)` を返す
        *   一致しない場合: `nil` を返す
    3. `InitContext` 関数内で `ParseBranchFeature(args)` の直後、`log.New()` の前に `validateBranchName(branch)` を呼び出す:
        ```go
        branch, feature := ParseBranchFeature(args)

        // ★ 追加: 予約ブランチ名チェック
        if err := validateBranchName(branch); err != nil {
            return nil, err
        }

        logger := log.New(os.Stderr, flagVerbose)
        ```

## Step-by-Step Implementation Guide

1.  [x] **テスト作成 (TDD: Red)**:
    *   `features/devctl/cmd/common_test.go` に `TestValidateBranchName` と `TestInitContext_ReservedBranch` を追加
    *   この時点では `validateBranchName` が存在しないためコンパイルエラーになる

2.  [x] **バリデーション関数の実装 (TDD: Green)**:
    *   `features/devctl/cmd/common.go` に `reservedBranchNames` 変数を追加
    *   `validateBranchName` 関数を実装
    *   `InitContext` 内に `validateBranchName` 呼び出しを追加

3.  [x] **ビルド & テスト**:
    *   `./scripts/process/build.sh` でビルドと全単体テストを実行
    *   新規テスト・既存テストの両方がパスすることを確認


## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認事項**:
        *   `TestValidateBranchName` の全サブテストがパス
        *   `TestInitContext_ReservedBranch` がパス
        *   既存の `TestInitContext_BranchOnly`, `TestInitContext_BranchAndFeature`, `TestInitContext_NoArgs` が引き続きパス（リグレッションなし）

## Documentation

本変更に影響を受ける仕様書・ドキュメントはありません。
