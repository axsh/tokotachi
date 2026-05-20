# 仕様書: editor.yaml によるエディタ起動方法のカスタマイズ機能

## 背景 (Background)
Antigravity のバージョンが 2.0 に上がり、VSCode ベースのエディタである **Antigravity IDE** に変更されました。これに伴い、Windows でのエディタ起動コマンドが `antigravity` から `antigravity-ide.cmd` へ変更されるなどの影響が出ています。

従来の実装では、`code`, `cursor`, `ag`, `claude` などのエディタ名と起動コマンドの関係が `tt` コマンド内にハードコーディングされており、コマンド名の変更や新規エディタ追加のたびに `tt` コマンドを再リリースする必要がありました。

本変更では、エディタの起動方法を外部設定ファイル `editor.yaml` でカスタマイズ可能にし、リリース不要で柔軟に対応できる仕組みを導入します。また、既存のハードコーディングされた個別エディタ実装（`ag.go`, `vscode.go`, `cursor.go` 等）を完全に廃止し、動的解決された単一のランチャー実装に一本化してコードを大幅に整理します。

## 要件 (Requirements)
1. **設定ファイルの配置と自動生成**
   - 設定ファイルは `{HOME}/.kotoshiro/tokotachi/editor.yaml` に配置します。
   - ファイルが存在しない場合は、ビルトインのデフォルト設定（`system` セクション）を含んだ設定ファイルを自動生成します。
   - **マニュアルの補記**: 生成されるファイルには、各設定値の意味、プレースホルダーの種類、優先度、およびタイプ別のデフォルト引数仕様などを説明するコメント（マニュアル）を先頭に補記し、ユーザーが編集しやすいようにします。

2. **YAML 構造の定義**
   - 設定ファイルは、`system` セクションと `user` セクションに大別されます。
     - **`system`**: `tt` コマンドが管理・自動生成するセクションです。リリース更新等で上書きされる可能性があります。
     - **`user`**: ユーザーが任意の設定を記述するセクションです。`tt` が勝手に書き換えることはありません。
   - 同名のエディタ設定が存在する場合、`user` セクションの設定が優先されます。

3. **エディタ設定項目**
   各エディタ設定（`system.editors.<name>` および `user.editors.<name>`）はトップレベルに共通設定を持ち、さらにOS別の固有設定（子要素）を定義可能です。
   
   **共通設定項目 (トップレベル)**:
   - `cmd` (string, 必須): 標準の起動コマンド名、または実行ファイルへの絶対パス。
   - `type` (string, 必須): 起動タイプ。以下のいずれかを指定します。
     - `vscode`: VSCode 系の起動方法。Dev Container アタッチ（`--folder-uri`）、新規ウィンドウ（`--new-window`）等の引数構成をサポートします。
     - `local`: ローカルオープン。ワークツリーのディレクトリパスを引数として渡して起動します。
     - `cli`: Claude Code などの CLI ツール。ワークツリーのディレクトリパスを引数にし、インタラクティブに起動します。
   - `args` (map, 必須): 起動状況ごとの引数のテンプレートリスト。以下のサブキーを持ちます。
     - `default` (slice of string, 必須): 通常のローカル起動時に渡す引数のテンプレートリスト。プレースホルダーを使用できます。
     - `new_window` (slice of string, 任意): 新規ウィンドウ起動（`opts.NewWindow == true`）の際に `default` の代わりに使用する引数のテンプレートリスト。
     - `devcontainer` (slice of string, 任意): Dev Container アタッチを試みる際（`opts.TryDevcontainer == true`）に使用する引数のテンプレートリスト。
   - `windows` (map, 任意): Windows 環境用の固有オーバーライド設定。
   - `darwin` (map, 任意): macOS 環境用の固有オーバーライド設定。
   - `linux` (map, 任意): Linux 環境用の固有オーバーライド設定。

   **OS別固有設定項目 (`windows` / `darwin` / `linux` の子要素)**:
   以下の項目を子要素として個別に記述でき、定義された項目のみがトップレベルの共通設定を上書き（オーバーライド）します。
   - `cmd` (string, 任意): プラットフォーム固有の起動コマンド名、または実行ファイルへの絶対パス。
   - `type` (string, 任意): プラットフォーム固有の起動タイプ。
   - `args` (map, 任意): プラットフォーム固有の引数設定（`default`, `new_window`, `devcontainer` を上書き可能）。

   **引数プレースホルダー**:
   引数テンプレート内の以下のプレースホルダーは、起動時に動的に置換されます。
   - `{path}`: ワークツリーのローカル絶対パス（例: `C:\Users\...\worktree`）
   - `{container}`: アタッチ対象のコンテナ名
   - `{uri}`: VSCode Dev Container 拡張で要求される attached-container 用の URI（例: `vscode-remote://attached-container+...`）

4. **既存個別エディタ実装の廃止**
   - これまでプログラム内にハードコーディングされていた `ag`, `code`, `cursor`, `claude` 用の個別起動ロジックはすべて廃止します。
   - 全てのエディタ起動を、ロードされた `editor.yaml` の設定に基づき動的にパラメータ置換を行う単一のランチャー（`CustomLauncher`）に一本化します。
   - `--editor` オプションで指定可能なエディタ名の判定も、ハードコーディングを廃止し、設定（`editor.yaml` からロードされた定義）に存在する任意のキーを許容するように変更します。

5. **堅牢性とエラーハンドリング**
   - `editor.yaml` の読み込みエラー（構文エラーやアクセス権エラーなど）が発生した場合でも、`tt` コマンドの動作を中断させてはいけません。
   - 読み込みエラー時は、プログラム内に定義されたデフォルト値（フォールバック）で動作を継続します。
   - ただし、読み込みエラーを検知した場合は、都度設定ファイルの再確認を促す警告（`WARNING: Failed to load editor.yaml...`）を標準エラー出力などに表示します。

6. **環境変数による上書きの優先度（下位互換性）**
   - 既存の環境変数 `TT_CMD_CODE`, `TT_CMD_CURSOR`, `TT_CMD_AG`, `TT_CMD_CLAUDE` は、`editor.yaml` の設定よりも優先して動作し続けます。

---

## 実装方針 (Implementation Approach)

### 1. 既存ファイルの廃止（削除予定）
- `pkg/editor/ag.go`
- `pkg/editor/vscode.go`
- `pkg/editor/cursor.go`
- `pkg/editor/factory.go`

### 2. 設定管理モジュールと動的ランチャーの追加
- `pkg/editor/config.go` を新規追加し、設定の定義とロードロジックを実装します。
- `pkg/editor/launcher.go` を新規追加（または既存ファイルを書き換え）し、YAML設定から生成される単一の `CustomLauncher` を実装します。
- `gopkg.in/yaml.v3` を使用して YAML のシリアライズ/デシリアライズを行います。
- ホームディレクトリの取得には `os.UserHomeDir()` を利用し、`~/.kotoshiro/tokotachi/editor.yaml` のパスを解決します。
- 設定ロード時にエラーが発生した場合、エラー情報を保持して後続処理に引き継ぎ、警告表示ができるようにします。

### 3. ビルトインデフォルト値の定義
自動生成される設定ファイルには、構造化された `args` ブロックを定義し、暗黙の動作を無くしてYAML単体で動作が把握できるようにします。

```yaml
# tokotachi editor settings
#
# この設定ファイルはエディタの起動コマンドや引数をカスタマイズするために使用します。
# 
# セクション説明:
# - system: tt コマンドが自動管理するセクションです。アップデートにより上書きされる可能性があります。
# - user:   ユーザーが任意の設定を記述するセクションです。tt が勝手に書き換えることはありません。
#           system セクションと同名のエディタ設定が存在する場合、user セクションが優先されます。
# 
# 各エディタの設定項目:
# - cmd:               基本の起動コマンド名、または絶対パス
# - type:              起動タイプ。以下から選択します:
#                      "vscode" -> VSCode系。Dev Containerアタッチや --new-window などの引数をサポート。
#                      "local"  -> ローカルフォルダオープン。
#                      "cli"    -> Claude Codeなどの対話型CLI。
# - args:              起動状況ごとの引数の階層設定。以下のサブキーを持ちます:
#                      default:      通常のローカル起動時に渡す引数リスト
#                      new_window:   新規ウィンドウ起動時に使用する引数リスト（任意）
#                      devcontainer: Dev Containerアタッチ時に使用する引数リスト（任意）
# 
# OS別の固有オーバーライド設定 (windows / darwin / linux):
#   上記の設定項目 (cmd, type, args) は、各 OS ブロックの子要素として個別に定義することでオーバーライドが可能です。
# 
# 引数プレースホルダー (args 内で利用可能):
# - {path}:      ローカルのワークツリー絶対パス
# - {container}: 対象のコンテナ名
# - {uri}:       VSCode/Cursor 等の Dev Container リモートURI
#

system:
  editors:
    code:
      cmd: "code"
      type: "vscode"
      args:
        default: ["{path}"]
        new_window: ["--new-window", "{path}"]
        devcontainer: ["--folder-uri", "{uri}"]
      windows:
        cmd: "code"
      darwin:
        cmd: "code"
      linux:
        cmd: "code"
    cursor:
      cmd: "cursor"
      type: "vscode"
      args:
        default: ["{path}"]
        new_window: ["--new-window", "{path}"]
        devcontainer: ["--folder-uri", "{uri}"]
      windows:
        cmd: "cursor"
      darwin:
        cmd: "cursor"
      linux:
        cmd: "cursor"
    ag:
      cmd: "antigravity"
      type: "vscode"
      args:
        default: ["{path}"]
        new_window: ["--new-window", "{path}"]
        devcontainer: ["--folder-uri", "{uri}"]
      windows:
        cmd: "antigravity-ide.cmd"
      darwin:
        cmd: "antigravity"
      linux:
        cmd: "antigravity"
    claude:
      cmd: "claude"
      type: "local"
      args:
        default: ["{path}"]
      windows:
        cmd: "claude"
      darwin:
        cmd: "claude"
      linux:
        cmd: "claude"
user:
  editors: {}
```

### 4. エディタ解決処理の動的化と引数置換
- `pkg/detect/editor.go` の `ParseEditor` は、定義されたエディタ一覧（`editor.yaml` からロードされたキーを含む）に基づいて動的に判定できるように修正します。
- `pkg/editor/factory.go` の `NewLauncher` の呼び出し部分（または代替される新しい初期化処理）を修正し、`editor.yaml` から取得した `EditorConfig` を持つ動的な `CustomLauncher` インスタンスを作成して返すようにリファクタリングします。
- `CustomLauncher.Launch` における、実行時設定の解決フロー：
  1. 実行環境の OS（`runtime.GOOS`）に対応するプラットフォーム設定ブロックが存在するか確認します。
  2. 存在する場合、OS 固有ブロック内に定義された非 nil の値（`cmd`, `type`）や固有の `args`（`default`, `new_window`, `devcontainer` の各項目）があればそれを採用し、定義されていない値はトップレベル共通設定から継承します。
  3. 起動状況（通常、新規ウィンドウ、Dev Container）に応じて、解決された `args` 設定から適切な引数テンプレート（`default`, `new_window`, `devcontainer`）を選択します。
  4. 選択された引数テンプレートに対して、プレースホルダー `{path}`、`{container}`、`{uri}` の置換を適用し、最終的な起動引数を組み立てます。

### 5. 警告表示のタイミング
- `tt` コマンド実行時、設定ファイルのロードエラーがある場合は、起動時（`tokotachi.go` の実行初期段階、または `ctx.Logger` が準備できた段階）に `WARNING` を出力します。

---

## 検証シナリオ (Verification Scenarios)

1. **初期状態での自動生成確認**
   - 事前に `~/.kotoshiro/tokotachi/editor.yaml` が存在しない状態にします。
   - `tt open` などのコマンドを実行します。
   - ファイルが自動生成され、先頭に説明用のコメント（構造化された引数設定についてのマニュアル）が含まれていること、および構造化された `args` を含むデフォルト設定がすべて含まれていることを確認します。

2. **`user` セクションによる上書き動作の確認**
   - `editor.yaml` の `user` セクションで以下のように `ag` を上書きします。
     ```yaml
     user:
       editors:
         ag:
           cmd: "custom-ag-cmd"
           type: "vscode"
     ```
   - `tt open {branch} --editor ag --dry-run` を実行します。
   - 実行されるコマンドが `custom-ag-cmd` になっていることを確認します。

3. **新規カスタムエディタの追加と起動確認**
   - `editor.yaml` の `user` セクションに新エディタを追加します。
     ```yaml
     user:
       editors:
         myeditor:
           cmd: "my-editor-cli"
           type: "local"
           args:
             default: ["{path}"]
     ```
   - `tt open {branch} --editor myeditor --dry-run` を実行します。
   - `my-editor-cli` がローカルフォルダを引数にして起動（シミュレーション）されることを確認します。

4. **プラットフォーム固有設定（構造化引数のオーバーライド）の確認**
   - `editor.yaml` の `user` セクションに、Windows 環境用で固有の引数を持つエディタを追加します。
     ```yaml
     user:
       editors:
         custom_code:
           cmd: "code"
           type: "vscode"
           windows:
             args:
               default: ["--windows-only-flag", "{path}"]
     ```
   - Windows 環境で `tt open {branch} --editor custom_code --dry-run` を実行します。
   - 起動引数として `--windows-only-flag` が正しく渡されることを確認します。

5. **設定ファイルの破損時フォールバックと警告表示確認**
   - `editor.yaml` を不正な YAML 構文（インデントミスなど）に変更します。
   - `tt open {branch} --dry-run` を実行します。
   - 警告メッセージ `WARNING: Failed to load editor.yaml...` が出力されることを確認します。
   - コマンドは中断せず、デフォルトの動作（フォールバック）で処理が続行することを確認します。

---

## テスト項目 (Testing for the Requirements)

### 自動化テスト
- **単体テスト (`pkg/editor/config_test.go` [NEW])**:
  - `editor.yaml` のパース、マージ（`user` が `system` をオーバーライドすること）、OSごとの設定（コマンド・構造化された引数・起動タイプ）の解決・上書きロジックのテスト。
  - 設定ファイルが存在しない場合の自動生成ロジックのテスト。
- **統合テスト (`tests/tt/tt_editor_test.go` [MODIFY])**:
  - 読み込みエラー時の警告出力と、デフォルト設定へのフォールバック動作のテスト。
  - テスト実行中のホームディレクトリ環境変数を一時的なディレクトリにモックして実行します。
