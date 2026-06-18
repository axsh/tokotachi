# 000-FixDeployDriftDetection

> **Source Specification**: prompts/phases/000-foundation/branches/fix-deploy-bug/ideas/000-FixDeployDriftDetection.md

## Goal Description
デプロイ判定時に、入力ソースハッシュ値の一致だけでなく、デプロイ先の実ファイル群の状態（ドリフト）も考慮することで、成果物の欠落や不整合が発生した際に確実に再デプロイされるように改善します。

## User Review Required
None.

## Requirement Traceability

> **Traceability Check**:
> 仕様書(Specification)の要件・決定事項をリストアップし、この計画書のどこで対応するかをマッピングしてください。

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| デプロイ先検証によるスキップ判定の強化 | Proposed Changes > deploy.go, update.go |
| ドリフト（不整合）検出時の強制デプロイ | Proposed Changes > deploy.go, update.go |
| 無変更かつ整合時のスキップ | Proposed Changes > deploy.go |

## Proposed Changes

### Compiler (features/tt/internal/prompt/compiler)

#### [MODIFY] [deploy.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-deploy-bug/features/tt/internal/prompt/compiler/deploy.go)
*   **Description**: CheckDrift 共通関数の実装および Deploy での判定呼び出しの追加
*   **Technical Design**:
    *   以下の関数を追加します：
        ```go
        func CheckDrift(rootDir, projectPath, target string) bool
        ```
    *   Deploy 関数のハッシュチェック条件判定（7番）を、!opts.Force && prevInfo.Digest == currentDigest && currentDigest != "" に加えて !CheckDrift(rootDir, opts.ProjectPath, target) の時にスキップするように書き換えます。
*   **Logic**:
    *   CheckDrift は、cfg.Outputs.ResolvedManifest を読み込み、yaml.Unmarshal を用いてパースします。
    *   指定の target 用のエミッターを初期化し、emitter.Check() メソッドを実行します。
    *   ドリフトが検出された場合（Checkが false を返した場合）やエラー時には true（ドリフトあり）を返し、整合している場合には false（ドリフトなし）を返します。

#### [MODIFY] [update.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-deploy-bug/features/tt/internal/prompt/compiler/update.go)
*   **Description**: Update 処理でのドリフト検出の統合
*   **Technical Design**:
    *   Update 内で、needsUpdate が false になった場合、さらに CheckDrift(rootDir, opts.ProjectPath, t) を実行します。
    *   CheckDrift が true を返した（ドリフトが検出された）場合、needsUpdate を true に切り替えます。
*   **Logic**:
    *   これにより、ソースファイルが変更されていない場合でも、デプロイ先ファイルが手動で書き換えられたり削除されたりした際に Update 処理がスキップされず、再デプロイが行われるようになります。

#### [MODIFY] [deploy_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-deploy-bug/features/tt/internal/prompt/compiler/deploy_test.go)
*   **Description**: ドリフト検出用の単体テストの追加
*   **Technical Design**:
    *   TestDeploy_Drift 関数を新規追加します。
    *   1回目のデプロイ後、出力ファイルの一部を削除または書き換えて、2回目のデプロイを呼び出し、result.Skipped が false となることをアサートします。

#### [MODIFY] [update_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-deploy-bug/features/tt/internal/prompt/compiler/update_test.go)
*   **Description**: アップデート時のドリフト検出用単体テストの追加
*   **Technical Design**:
    *   TestUpdate_Drift を追加し、アップデート実行後に出力ファイルにドリフトを発生させて再度アップデートした際に、スキップされずデプロイされることをテストします。

## Step-by-Step Implementation Guide

1.  **Helper Implementation**:
    *   deploy.go に CheckDrift 関数を追加します。必要なインポート（os, gopkg.in/yaml.v3, github.com/axsh/tokotachi/features/tt/internal/prompt/manifest）を追加します。
2.  **Deploy Decision Update**:
    *   deploy.go 内の Deploy 関数に CheckDrift 呼び出しを組み込みます。
3.  **Update Decision Update**:
    *   update.go 内の Update 関数に CheckDrift 呼び出しを組み込み、ドリフト検出時に needsUpdate を true に設定します。
4.  **Test Implementation**:
    *   deploy_test.go に TestDeploy_Drift を実装します。
    *   update_test.go に TestUpdate_Drift を実装します。
5.  **Verify**:
    *   Verification Plan に記述されている自動テストおよび手動確認を実行します。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ビルドおよび単体テストを実行します。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    リグレッションテストとして common カテゴリのテストを実行します。
    ```bash
    ./scripts/process/integration_test.sh --categories "common"
    ```

## Documentation
None.
