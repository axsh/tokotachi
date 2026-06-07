# 仕様書: Antigravity IDE 起動コマンドの修正および自動アップデート

## 背景 (Background)
Windows版の Antigravity (Antigravity IDE 2.0) は、`C:\Users\<UserName>\AppData\Local\Programs\Antigravity IDE\bin` ディレクトリ配下に `antigravity-ide.cmd` という名前でインストールされます。
しかし、インストール時にこの `bin` ディレクトリが環境変数 `%PATH%` に自動的に追加されないケースがあり、その場合に `tt open --editor ag` を実行すると `exec: "antigravity-ide.cmd": executable file not found in %PATH%` というエラーが発生して起動に失敗していました。

本変更では、Windowsにおける Antigravity の起動コマンド定義を本来の `antigravity-ide.cmd` に統一しつつ、PATHが通っていない環境でも動作するように、インストーラー標準のパスを自動探索して絶対パスで実行するフォールバックロジックを導入します。

## 要件 (Requirements)
1. **起動コマンド名の修正（本来の定義へ戻す）**
   - `editor.yaml` のシステム定義（`defaultConfig` および `defaultYAMLTemplate`）における `ag` (Antigravity) の Windows用起動コマンドを本来の `"antigravity-ide.cmd"` に戻し、他OS用のコマンドも `"antigravity"` に戻します。
2. **自動探索（フォールバック）の実装**
   - Windows環境において、実行コマンドが `"antigravity-ide.cmd"` であり、かつPATHに見つからない（`exec.LookPath` がエラーを返す）場合、標準のインストールパス `C:\Users\<Username>\AppData\Local\Programs\Antigravity IDE\bin\antigravity-ide.cmd` の存在を確認し、存在すればその絶対パスを用いて起動します。
3. **システム設定のメモリ上での強制上書き**
   - 設定ファイルからロードした `cfg.System` セクションを、プログラムのビルトインデフォルトである `defaultConfig().System` で常に上書きし、プログラムのアップデート時に最新のシステム起動設定が即座に反映されるようにします。
4. **未カスタマイズ設定ファイルの自動アップデート**
   - ユーザーが `user` セクションでカスタマイズを行っていない（`user.editors` が空）かつファイルの内容が最新のテンプレートと異なる場合、設定ファイルを最新の `defaultYAMLTemplate` で自動的に上書き更新します。

## 実現方針 (Implementation Approach)

### `pkg/editor/config.go` の変更
- `defaultYAMLTemplate` 内の `ag` 設定における `cmd` の値を `"antigravity"` に、`windows.cmd` を `"antigravity-ide.cmd"` に戻します。
- `defaultConfig()` 内の `ag` の `EditorConfig` 定義において、`Cmd` を `"antigravity"`、`Windows.Cmd` を `"antigravity-ide.cmd"` に戻します。
- `LoadConfig()` の処理において、自動アップデートとメモリ上の `cfg.System` の上書き処理を維持します。

### `pkg/editor/launcher.go` の変更
- `Launch()` 関数のコマンド実行前（`opts.DryRun` や実際の実行の前）に、Windows用フォールバックロジックを実装します。
  - OSがWindowsかつコマンドが `"antigravity-ide.cmd"` の時、`exec.LookPath` でパスが通っているか確認。
  - パスが通っていない場合、`C:\Users\<Username>\AppData\Local\Programs\Antigravity IDE\bin\antigravity-ide.cmd` の存在を確認。
  - 存在すれば `cmd` の値をその絶対パスに書き換える。

## 検証シナリオ (Verification Scenarios)

### 手動検証
1. **ドライランでの確認**
   - PATHに `antigravity-ide.cmd` が通っていない状態のWindows環境で `tt open {branch} --editor ag --dry-run` を実行します。
   - 自動探索が働き、`[DRY-RUN] C:\Users\<Username>\AppData\Local\Programs\Antigravity IDE\bin\antigravity-ide.cmd --new-window ...` のように絶対パスでコマンドが表示されることを確認します。
2. **実際の起動**
   - Windows環境で `tt open {branch} --editor ag` を実行し、エラーなく Antigravity IDE が起動することを確認します。

## テスト項目 (Testing for the Requirements)

### 自動化テスト
- **単体テスト (`pkg/editor/config_test.go`)**:
  - `LoadConfig()` が `ag` の Windows用コマンドとして `antigravity-ide.cmd` を正しくロードすることを確認します。
- **全体ビルドとテスト (`scripts/process/build.sh`)**:
  - すべての単体テストがパスすることを確認します。
