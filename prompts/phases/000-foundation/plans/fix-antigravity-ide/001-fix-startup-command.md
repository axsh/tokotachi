# 001-fix-startup-command

> **Source Specification**: [001-fix-startup-command.md](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/prompts/phases/000-foundation/ideas/fix-antigravity-ide/001-fix-startup-command.md)

## Goal Description
Windows版の Antigravity (Antigravity IDE 2.0) の起動コマンドを本来の `antigravity-ide.cmd` に戻した上で、環境変数 `%PATH%` に登録されていない場合でも自動的にインストールディレクトリを探索して絶対パスで起動するフォールバックロジックを導入します。また、設定ファイルの自動アップデートとメモリ上の強制上書き機能も合わせて維持します。

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. 起動コマンド名の修正 (`antigravity-ide.cmd` への変更) | Proposed Changes > [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go) |
| 2. 自動探索（フォールバック）の実装 | Proposed Changes > [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go) |
| 3. システム設定のメモリ上での強制上書き | Proposed Changes > [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go) |
| 4. 未カスタマイズ設定ファイルの自動アップデート | Proposed Changes > [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go) |

## Proposed Changes

### editor (pkg/editor)

#### [MODIFY] [config_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config_test.go)
*   **Description**:
    - `ag` の Windows用コマンドの期待値が `"antigravity-ide.cmd"` になっていることを確認するようにアサーションを更新します。
    - `TestLoadConfig_AutoUpdateOutdated` において、マージ・上書きされた後の期待値を `"antigravity-ide.cmd"` に設定します。

#### [MODIFY] [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go)
*   **Description**:
    - `Launch()` 関数の冒頭で `cmd` が解決された後、Windows環境かつ `cmd == "antigravity-ide.cmd"` で、かつ `exec.LookPath` がエラーを返す場合に、インストーラー標準のパス `C:\Users\<Username>\AppData\Local\Programs\Antigravity IDE\bin\antigravity-ide.cmd` を自動探索して `cmd` を絶対パスに書き換える処理を実装します。
    - `os/exec` と `path/filepath` をインポートに追加します。
*   **Technical Design**:
    ```go
    // LookPath fallback for Windows Antigravity IDE
    if runtime.GOOS == "windows" && cmd == "antigravity-ide.cmd" {
        if _, err := exec.LookPath(cmd); err != nil {
            if home, err := os.UserHomeDir(); err == nil {
                fallbackPath := filepath.Join(home, "AppData", "Local", "Programs", "Antigravity IDE", "bin", "antigravity-ide.cmd")
                if _, err := os.Stat(fallbackPath); err == nil {
                    cmd = fallbackPath
                }
            }
        }
    }
    ```

#### [MODIFY] [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go)
*   **Description**:
    - `defaultYAMLTemplate` 内の `ag` の起動コマンド定義において、`cmd` (共通) を `"antigravity"` に、`windows.cmd` を `"antigravity-ide.cmd"` に、その他OSの `cmd` も `"antigravity"` に戻します。
    - `defaultConfig()` 内の `ag` の `EditorConfig` 定義において、`Cmd` を `"antigravity"`、`Windows.Cmd` を `"antigravity-ide.cmd"` に、その他OSも `"antigravity"` に戻します。
    - `LoadConfig()` の処理におけるシステム設定の上書きおよび自動アップデートロジックは維持します。

### Integration Tests (tests/)

#### [MODIFY] [tt_editor_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/tests/tt/tt_editor_test.go)
*   **Description**:
    - 特に追加の変更はありません。

## Step-by-Step Implementation Guide

1.  **設定ファイルの修正**:
    - [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go) の `defaultYAMLTemplate` と `defaultConfig()` を修正し、`agy` を `"antigravity-ide.cmd"` / `"antigravity"` に戻します。
2.  **テストコードの修正**:
    - [config_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config_test.go) の `TestLoadConfig_AutoUpdateOutdated` における Windowsコマンドの期待値を `"antigravity-ide.cmd"` に更新します。
3.  **フォールバック処理の実装**:
    - [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go) に `os/exec` と `path/filepath` をインポートし、Windows用自動探索ロジックを追加します。
4.  **検証とテスト**:
    - 単体テストとドライランを実行し、PATHが通っていなくても自動で絶対パスに解決されて起動することを確認します。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```

## Documentation
None.
