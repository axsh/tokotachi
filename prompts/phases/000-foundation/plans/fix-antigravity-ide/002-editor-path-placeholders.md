# 002-editor-path-placeholders

> **Source Specification**: [002-editor-path-placeholders.md](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/prompts/phases/000-foundation/ideas/fix-antigravity-ide/002-editor-path-placeholders.md)

## Goal Description
`editor.yaml` 内のエディタ起動設定において、コマンドパス内の動的プレースホルダー（`{home}`、`{localappdata}`）の解決と、複数の候補から利用可能なものを優先順に選択するフォールバックリスト（`cmds`）機能を導入します。これにより、プログラム側に直接ハードコーディングされていた特定の起動パスの特例解決処理を完全に排除し、設定ファイルから宣言的かつ汎用的に定義可能にします。

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. 複数コマンド候補（フォールバックリスト）の指定 | Proposed Changes > [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go) |
| 2. コマンドパス内のプレースホルダー動的解決 | Proposed Changes > [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go) |
| 3. 優先順位付きコマンド解決ロジック | Proposed Changes > [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go) |
| 4. ハードコードされた個別ロジックの排除 | Proposed Changes > [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go) |
| 5. デフォルトYAMLテンプレートのコメント更新 | Proposed Changes > [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go) |

## Proposed Changes

### editor (pkg/editor)

#### [MODIFY] [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go)
*   **Description**:
    - `EditorConfig` および `PlatformConfig` 構造体に `Cmds []string` フィールド（YAMLタグ: `cmds`）を追加します。
    - `defaultYAMLTemplate` 内に `cmds` の説明コメント（プレースホルダーやフォールバック解決について）を追記し、`ag` の Windows用設定に `cmds` のリストを定義します。
    - `defaultConfig()` 内の `ag` 設定において、`Windows.Cmds` をスライス `[]string{"antigravity-ide.cmd", "{home}/AppData/Local/Programs/Antigravity IDE/bin/antigravity-ide.cmd"}` に変更します。
*   **Technical Design**:
    ```go
    type EditorConfig struct {
        Cmd   string          `yaml:"cmd"`
        Cmds  []string        `yaml:"cmds,omitempty"` // 追加
        Type  string          `yaml:"type"`
        // ...
    }

    type PlatformConfig struct {
        Cmd  *string     `yaml:"cmd,omitempty"`
        Cmds []string    `yaml:"cmds,omitempty"` // 追加
        Type *string     `yaml:"type,omitempty"`
        Args *ArgsConfig `yaml:"args,omitempty"`
    }
    ```

#### [MODIFY] [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go)
*   **Description**:
    - 特例的な `antigravity-ide.cmd` の解決コードを削除し、プレースホルダー置換と複数候補のフォールバック解決を行う汎用的な `resolveCommand` ヘルパー関数を実装します。
    - `Launch` メソッド内で、決定されたコマンド設定から有効なコマンドを解決して使用します。
*   **Technical Design**:
    ```go
    func (l *CustomLauncher) resolveCommand(cmd string, cmds []string) string {
        var candidates []string
        if len(cmds) > 0 {
            candidates = cmds
        } else if cmd != "" {
            candidates = []string{cmd}
        }

        if len(candidates) == 0 {
            return cmd
        }

        for _, c := range candidates {
            resolved := l.bindPlaceholders(c)
            // PATH上にあるか、直接ファイルが存在するか確認
            if _, err := exec.LookPath(resolved); err == nil {
                return resolved
            }
            if _, err := os.Stat(resolved); err == nil {
                return resolved
            }
        }

        // 何も見つからなければ、プレースホルダー置換した最初の候補を返す
        return l.bindPlaceholders(candidates[0])
    }

    func (l *CustomLauncher) bindPlaceholders(c string) string {
        if home, err := os.UserHomeDir(); err == nil {
            c = strings.ReplaceAll(c, "{home}", home)
        }
        if localappdata := os.Getenv("LOCALAPPDATA"); localappdata != "" {
            c = strings.ReplaceAll(c, "{localappdata}", localappdata)
        }
        return c
    }
    ```

#### [MODIFY] [config_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config_test.go)
*   **Description**:
    - YAMLパースおよびデフォルトロードで `cmds` フィールドが正しくロードされることをテストします。
    - `TestLoadConfig_AutoUpdateOutdated` 等で Windows コマンドリストの解決が正しく行われることを確認します。

#### [NEW] [launcher_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher_test.go)
*   **Description**:
    - プレースホルダー `{home}` や `{localappdata}` の置換テスト、および複数候補（`cmds`）からの優先度付きフォールバック解決が意図した通り動くことを検証する単体テストを追加します。

### Integration Tests (tests/)

#### [MODIFY] [tt_editor_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/tests/tt/tt_editor_test.go)
*   **Description**:
    - 統合テストカテゴリ `tt` における `TestEditor_CustomEditorDynamicLaunch` などのテストで、`cmds` 定義を含むエディタ起動が正しく動作することを検証します。

## Step-by-Step Implementation Guide

1.  **データ構造とYAMLデフォルトの修正**:
    - [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go) に `Cmds` フィールドを追加し、`defaultYAMLTemplate` 内にコメントと `cmds` 配列の定義を追加します。`defaultConfig()` における `ag` 設定を `cmds` ベースに置き換えます。
2.  **設定テストの更新**:
    - [config_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config_test.go) で `cmds` フィールドが正しくロードされるアサーションを追加します。
3.  **解決ロジックの実装**:
    - [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go) から以前のハードコードされたフォールバック処理を削除し、`resolveCommand` および `bindPlaceholders` 関数を追加します。`Launch` メソッド内のコマンド決定部でこれらを実行するように変更します。
4.  **ランチャーテストの作成**:
    - [launcher_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher_test.go) を新規作成し、プレースホルダー置換とフォールバック解決の挙動を検証する単体テストを実装します。
5.  **全体のビルドと動作確認**:
    - 単体テストおよび統合テストを実行して検証します。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt"
    ```

## Documentation
None.
