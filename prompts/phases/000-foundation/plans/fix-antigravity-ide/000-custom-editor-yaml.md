# 実装計画: editor.yaml によるエディタ起動方法のカスタマイズ機能

> **Source Specification**: [000-custom-editor-yaml.md](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/prompts/phases/000-foundation/ideas/fix-antigravity-ide/000-custom-editor-yaml.md)

## Goal Description
外部設定ファイル `editor.yaml` を導入し、エディタの起動方法（コマンド、引数、OS別上書きなど）をカスタマイズ可能にします。同時に、従来のハードコーディングされた個別エディタ実装（`ag.go`, `vscode.go`, `cursor.go`, `factory.go` 等）を完全に廃止し、設定駆動の単一の共通ランチャー実装に一本化・リファクタリングして、コード全体の保守性と柔軟性を飛躍的に高めます。

## User Review Required

> [!IMPORTANT]
> **エディタ別Goファイルの廃止**: 既存 of `ag.go`, `vscode.go`, `cursor.go`, `factory.go` などのファイルはすべて削除され、`CustomLauncher` に統合されます。
> **エディタ判定の動的化**: `--editor` の判定ロジックもハードコーディングを排除し、設定に定義されている任意のキーをエディタとして識別・許可するように変更します。
> **循環参照の回避**: `pkg/matrix` が `pkg/editor` の設定情報を動的に参照できるよう、フック関数（コールバック）を導入して動的解決を可能にします。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 設定ファイルの配置と自動生成 (`~/.kotoshiro/tokotachi/editor.yaml`) | `Proposed Changes > pkg/editor/config.go` |
| マニュアルコメントを含んだ初期設定ファイルの書き込み | `Proposed Changes > pkg/editor/config.go` |
| `system` セクションと `user` セクションの分離とマージ処理 | `Proposed Changes > pkg/editor/config.go` |
| `vscode`, `local`, `cli` の各起動タイプに対応した引数処理 | `Proposed Changes > pkg/editor/launcher.go` |
| 構造化された `args` ブロック（`default`, `new_window`, `devcontainer`） | `Proposed Changes > pkg/editor/config.go` |
| OS別固有設定のオーバーライド（`windows`, `darwin`, `linux`） | `Proposed Changes > pkg/editor/launcher.go` |
| 引数プレースホルダー（`{path}`, `{container}`, `{uri}`）の置換 | `Proposed Changes > pkg/editor/launcher.go` |
| 既存個別エディタ起動処理 (`ag.go`, `vscode.go`, `cursor.go`, `factory.go`) の廃止 | `Proposed Changes > pkg/editor/` 既存ファイルの削除 |
| `--editor` の判定動的化（任意のキーの許容） | `Proposed Changes > pkg/detect/editor.go` |
| ロードエラー時の警告ログ出力とデフォルトへのフォールバック動作 | `Proposed Changes > tokotachi.go`, `features/tt/cmd/editor.go` 等 |
| 環境変数（`TT_CMD_CODE` 等）による上書きの優先 | `Proposed Changes > pkg/editor/launcher.go` |

## Proposed Changes

### pkg/detect

#### [MODIFY] [editor.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/detect/editor.go)
*   **Description**: `ParseEditor` におけるエディタ名のハードコード制限を廃止し、エイリアス解決を行った後、任意のカスタムエディタ名をそのまま `Editor` 型として許容するように変更します。
*   **Technical Design**:
    ```go
    func ParseEditor(s string) (Editor, error) {
        switch s {
        case "code", "vscode":
            return "code", nil
        case "cursor":
            return "cursor", nil
        case "ag", "antigravity":
            return "ag", nil
        case "claude":
            return "claude", nil
        default:
            if s == "" {
                return "", fmt.Errorf("editor name cannot be empty")
            }
            // 未知のエディタであってもエラーとせずそのまま許容
            return Editor(s), nil
        }
    }
    ```

### pkg/matrix

#### [MODIFY] [matrix.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/matrix/matrix.go)
*   **Description**: `pkg/editor` への循環参照を回避しつつ、未知のカスタムエディタに対しても正しい `Capability` （Dev Container アタッチの可否など）を動的に決定できるよう、エディタタイプ解決のフック関数 `EditorTypeResolver` を追加します。
*   **Technical Design**:
    ```go
    // EditorTypeResolver allows external registration of a function to resolve 
    // an editor's type (e.g., "vscode", "local", "cli") dynamically.
    var EditorTypeResolver func(editor detect.Editor) string

    func ResolveCapability(os detect.OS, editor detect.Editor) Capability {
        key := matrixKey{os: os, editor: editor}
        if cap, ok := defaultMatrix[key]; ok {
            return cap
        }
        
        var editorType string
        if EditorTypeResolver != nil {
            editorType = EditorTypeResolver(editor)
        }

        // タイプ別の動的Capability構築
        if editorType == "vscode" {
            return Capability{
                CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
                CanLaunchNewWindow: true,
                LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
            }
        } else if editorType == "cli" {
            return Capability{
                CanOpenLocal: true, CanRunClaudeLocally: true, CanUseSSHMode: true,
                LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L1Supported,
            }
        }

        // デフォルトフォールバック: ローカルオープンのみ
        return Capability{
            CanOpenLocal:          true,
            LocalOpenLevel:        L1Supported,
            DevcontainerOpenLevel: L4Unsupported,
            SSHLevel:              L4Unsupported,
        }
    }
    ```

### pkg/editor

#### [DELETE] [ag.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/ag.go)
*   **Description**: 個別エディタ起動コード。共通化のため廃止・削除します。

#### [DELETE] [vscode.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/vscode.go)
*   **Description**: 個別エディタ起動コード。共通化のため廃止・削除します。

#### [DELETE] [cursor.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/cursor.go)
*   **Description**: 個別エディタ起動コード。共通化のため廃止・削除します。

#### [DELETE] [factory.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/factory.go)
*   **Description**: 個別エディタを生成するファクトリコード。共通化のため廃止・削除します。

#### [NEW] [config.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go)
*   **Description**: `editor.yaml` のデータ構造定義、ファイルロード、`user` セクションのマージ、および初期設定ファイル自動生成（コメントマニュアル付き）ロジックを実装します。
*   **Technical Design**:
    ```go
    package editor

    import (
        "io/ioutil"
        "os"
        "path/filepath"
        "gopkg.in/yaml.v3"
        "github.com/axsh/tokotachi/pkg/detect"
    )

    type ArgsConfig struct {
        Default      []string `yaml:"default"`
        NewWindow    []string `yaml:"new_window,omitempty"`
        Devcontainer []string `yaml:"devcontainer,omitempty"`
    }

    type PlatformConfig struct {
        Cmd  *string     `yaml:"cmd,omitempty"`
        Type *string     `yaml:"type,omitempty"`
        Args *ArgsConfig `yaml:"args,omitempty"`
    }

    type EditorConfig struct {
        Cmd     string          `yaml:"cmd"`
        Type    string          `yaml:"type"`
        Args    ArgsConfig      `yaml:"args"`
        Windows *PlatformConfig `yaml:"windows,omitempty"`
        Darwin  *PlatformConfig `yaml:"darwin,omitempty"`
        Linux   *PlatformConfig `yaml:"linux,omitempty"`
    }

    type Config struct {
        System struct {
            Editors map[string]EditorConfig `yaml:"editors"`
        } `yaml:"system"`
        User struct {
            Editors map[string]EditorConfig `yaml:"editors"`
        } `yaml:"user"`
    }

    // ResolveEditor finds the configuration for the given editor name, 
    // prioritizing the user section, then the system section.
    func (c *Config) ResolveEditor(name string) (EditorConfig, bool) {
        if ec, ok := c.User.Editors[name]; ok {
            return ec, true
        }
        if ec, ok := c.System.Editors[name]; ok {
            return ec, true
        }
        return EditorConfig{}, false
    }

    // LoadConfig loads the editor config. If file does not exist, it auto-generates one.
    // In case of load errors, it returns a fallback config and the error.
    func LoadConfig() (*Config, error) { ... }
    ```
*   **Logic**:
    *   `LoadConfig` は、まず `{HOME}/.kotoshiro/tokotachi/editor.yaml` の存在を確認します。
    *   存在しない場合、親ディレクトリを作成し、仕様書記載のテンプレート（マニュアルコメントと初期値を含んだ YAML 文字列）を新規書き込みします。
    *   ロード時には `yaml.Unmarshal` を用いてパースします。ロードエラーがあった場合は、警告ログ用にエラーを返しつつ、プログラム内にハードコードされた初期デフォルト構成を組み立てて返却します。

#### [NEW] [launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go)
*   **Description**: ロードされた `EditorConfig` に基づき、OS固有の上書き、引数の選択、プレースホルダー置換、環境変数優先処理を行い、動的かつ汎用的にエディタを起動する `CustomLauncher` を実装します。
*   **Technical Design**:
    ```go
    package editor

    import (
        "fmt"
        "os"
        "runtime"
        "strings"
        "github.com/axsh/tokotachi/pkg/detect"
    )

    type CustomLauncher struct {
        name string
        cfg  EditorConfig
    }

    func NewLauncher(ed detect.Editor, cfg *Config) (Launcher, error) {
        editorCfg, ok := cfg.ResolveEditor(string(ed))
        if !ok {
            return nil, fmt.Errorf("editor %q not found in config", ed)
        }
        return &CustomLauncher{
            name: string(ed),
            cfg:  editorCfg,
        }, nil
    }

    func (l *CustomLauncher) Name() string { return l.name }

    func (l *CustomLauncher) Launch(opts LaunchOptions) (LaunchResult, error) { ... }
    ```
*   **Logic**:
    *   **環境変数による上書き**:
        `TT_CMD_CODE`, `TT_CMD_CURSOR`, `TT_CMD_AG`, `TT_CMD_CLAUDE` の環境変数が定義されている場合、該当するエディタ名の起動コマンド（`cmd`）を環境変数の値で最優先して上書きします。
    *   **OS別上書き解決**:
        `runtime.GOOS` が `windows`, `darwin`, `linux` のどれかであるかを判定し、対応する OS ブロック（`Windows`, `Darwin`, `Linux`）が `EditorConfig` 内にあれば、その定義値（非 nil の `Cmd`, `Type`, `Args` 内のサブキー）で共通設定を上書き（オーバーライド）します。
    *   **状況別引数テンプレート選択**:
        *   `vscode` タイプ、かつアタッチ対象コンテナが有効な場合： `devcontainer` 引数テンプレートを選択。
        *   `vscode` タイプ、かつ `opts.NewWindow` の場合： `new_window` 引数テンプレートを選択。
        *   それ以外： `default` 引数テンプレートを選択。
    *   **プレースホルダー置換**:
        引数テンプレート内の `{path}`, `{container}`, `{uri}` を実際のパスやコンテナ情報、リモートURI（`DevcontainerURI` の戻り値）に動的置換します。
    *   **コマンド実行**:
        アタッチ起動（成功時は終了）と、失敗時のローカルオープンフォールバック処理を実装します。

### tokotachi

#### [MODIFY] [tokotachi.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/tokotachi.go)
*   **Description**: `doEditor` で `editor.LoadConfig` を実行し、ロードエラーがあれば警告を表示しつつ、設定を `editor.NewLauncher` に渡してランチャーを作成します。
*   **Technical Design**:
    ```go
    // doEditor opens the editor for the given branch and feature.
    func (c *Client) doEditor(ctx *opContext, branch, feature, editorFlag string) error {
        ...
        cfg, cfgErr := editor.LoadConfig()
        if cfgErr != nil {
            ctx.logger.Warn("WARNING: Failed to load editor.yaml (%v). Using built-in fallback configuration.", cfgErr)
        }

        launcher, err := editor.NewLauncher(editorName, cfg)
        if err != nil {
            return fmt.Errorf("editor launcher creation failed: %w", err)
        }
        ...
    }
    ```

### features/tt/cmd

#### [MODIFY] [editor.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/features/tt/cmd/editor.go)
*   **Description**: `tt editor` コマンド起動時に `editor.LoadConfig` を実行し、解決された `Config` から `editor.NewLauncher` を構築するように変更します。
*   **Technical Design**:
    ```go
    func runEditor(cmd *cobra.Command, args []string) error {
        ...
        cfg, cfgErr := editor.LoadConfig()
        if cfgErr != nil {
            ctx.Logger.Warn("WARNING: Failed to load editor.yaml (%v). Using built-in fallback configuration.", cfgErr)
        }

        launcher, err := editor.NewLauncher(ed, cfg)
        if err != nil {
            return fmt.Errorf("editor launcher creation failed: %w", err)
        }
        ...
    }
    ```

#### [MODIFY] [open.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/features/tt/cmd/open.go)
*   **Description**: `tt open` コマンド起動時に `editor.LoadConfig` を実行し、解決された `Config` から `editor.NewLauncher` を構築するように変更します。
*   **Technical Design**:
    ```go
    func runOpen(cmd *cobra.Command, args []string) error {
        ...
        cfg, cfgErr := editor.LoadConfig()
        if cfgErr != nil {
            ctx.Logger.Warn("WARNING: Failed to load editor.yaml (%v). Using built-in fallback configuration.", cfgErr)
        }

        launcher, err := editor.NewLauncher(ed, cfg)
        if err != nil {
            return fmt.Errorf("editor launcher creation failed: %w", err)
        }
        ...
    }
    ```

### Integration Tests (tests/)

#### [MODIFY] [tt_editor_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/tests/tt/tt_editor_test.go)
*   **Description**: `editor.yaml` が破損している場合に警告ログが出力されつつデフォルト値でフォールバックして起動できるか、また `editor.yaml` に追加された未知のカスタムエディタが正しく動的に解決・起動されるかどうかの統合テストを追加します。
*   **Category**: `tt`
*   **Test Cases**:
    *   `TestEditor_LoadWarningAndFallback`: 不正な YAML ファイルを配置して `tt open` を実行した際、警告メッセージ `WARNING: Failed to load editor.yaml` が出力され、かつデフォルトのフォールバック動作で起動が完了すること。
    *   `TestEditor_CustomEditorDynamicLaunch`: `editor.yaml` にカスタムエディタ `myeditor` を追加した状態で `tt open --editor myeditor` を実行し、カスタムコマンドが想定通りの引数で呼び出されること（ドライランによるコマンド名検証）。

## Step-by-Step Implementation Guide

1.  **初期化とフック設定 (`pkg/matrix`)**:
    *   [matrix.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/matrix/matrix.go) に `EditorTypeResolver` フックと動的 Capability 解決のコードを追加。
2.  **エディタ名判定の動的化 (`pkg/detect`)**:
    *   [editor.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/detect/editor.go) の `ParseEditor` を、未知のエディタ名を許可するように修正。
3.  **設定処理の実装 (`pkg/editor/config.go` [NEW])**:
    *   YAMLの構造体定義、`LoadConfig` 関数（自動生成機能とマニュアルコメント書き出し）、`ResolveEditor` によるセクションマージ処理を記述。
    *   `init` 関数の中で `matrix.EditorTypeResolver` コールバックを登録し、ロードした config からエディタの type を返せるように設定。
4.  **共通ランチャーの実装 (`pkg/editor/launcher.go` [NEW])**:
    *   `CustomLauncher` の `Launch` メソッドを実装。
    *   環境変数上書き、OS別上書き、状況別引数テンプレート（`default`, `new_window`, `devcontainer`）の決定、およびプレースホルダーの文字列置換ロジックを実装。
5.  **個別起動コードの廃止**:
    *   `pkg/editor/ag.go`, `pkg/editor/vscode.go`, `pkg/editor/cursor.go`, `pkg/editor/factory.go` を削除。
6.  **単体テストの更新**:
    *   `pkg/editor/config_test.go` [NEW] を作成し、マージ動作やOS別引数解決、自動生成ロジックのテストを実装。
    *   `pkg/editor/editor_test.go` を修正し、新しい `CustomLauncher` をターゲットとしたテストに置き換える。
7.  **呼び出し元の修正**:
    *   [tokotachi.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/tokotachi.go) の `doEditor` を修正し、`LoadConfig` の結果を `NewLauncher` に渡す。警告出力を追加。
    *   [editor.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/features/tt/cmd/editor.go) および [open.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/features/tt/cmd/open.go) を修正。
8.  **統合テストの作成・修正**:
    *   `tests/tt/tt_editor_test.go` にフォールバック警告テストおよびカスタムエディタ動的テストを追加。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ビルドおよび新旧の単体テストを実行します。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    統合テストを実行し、エディタの動的解決、エラー警告が正常に動作することを確認します。
    ```bash
    ./scripts/process/integration_test.sh --categories "tt"
    ```
    *   **Log Verification**: 統合テストログにて `WARNING: Failed to load editor.yaml` の警告出力と、カスタムエディタ `myeditor` の起動（ドライランログにカスタムコマンド名が正しく出力されていること）を確認します。
