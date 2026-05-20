# 仕様書: エディタ起動コマンドのプレースホルダー解決とフォールバック指定の汎用化

## 背景 (Background)
以前の対応において、Windows環境で Antigravity IDE 2.0 が起動できない問題を解決するため、`launcher.go` 内に「コマンドが `antigravity-ide.cmd` かつPATHに通っていない場合に、`C:\Users\<Username>\AppData\Local\Programs\Antigravity IDE\bin\...` を自動探索する」という**特例的なハードコーディング（ロジック）**を実装しました。

しかし、このように特定のエディタ名やファイルパスに依存した処理をプログラム（Goコード）に直接記述することは、コードの保守性を損ない、将来的に別のエディタが追加された際の拡張性を狭める原因となります。

本仕様では、`editor.yaml` の設定スキーマを拡張し、コマンドパス内のプレースホルダー（`{home}` など）の動的置換と、複数の起動コマンド候補（フォールバックリスト）の指定をサポートすることで、特例的なハードコードを排除し、すべて設定ファイル上で解決可能にすることを目的とします。

## 要件 (Requirements)

### 1. 複数コマンド候補（フォールバックリスト）の指定
- `editor.yaml` の各エディタ設定（およびデフォルトの `EditorConfig`）に、複数の起動コマンド候補を優先順位順に指定できる `cmds` フィールド（文字列配列）を追加します。
- 従来の `cmd` フィールド（単一文字列）も後方互換性のために残し、`cmds` が未指定の場合は `cmd` のみを候補として扱います。
- 例：
  ```yaml
  ag:
    cmd: "antigravity"
    windows:
      cmds:
        - "antigravity-ide.cmd" # PATHが通っていればこれを使う
        - "{home}/AppData/Local/Programs/Antigravity IDE/bin/antigravity-ide.cmd" # 無ければこちらを使う
  ```

### 2. コマンドパス内のプレースホルダー動的解決
- コマンド定義（`cmd` または `cmds` 内の要素）において、以下のプレースホルダーをサポートし、実行時に動的に置換します。
  - `{home}` : 実行ユーザーのホームディレクトリ（Go の `os.UserHomeDir()` の戻り値）に置換します。
  - `{localappdata}` : Windows用、環境変数 `%LOCALAPPDATA%`（または `os.Getenv("LOCALAPPDATA")`）に置換します。

### 3. 優先順位付きコマンド解決ロジック
- `launcher.go` におけるエディタ起動処理（`Launch`）の冒頭で、利用可能なコマンドを以下の手順で動的に解決します。
  1. `cmds`（または `cmd`）の候補リストを順番に走査します。
  2. 各候補文字列のプレースホルダー（`{home}`、`{localappdata}`等）を実値に置換します。
  3. 置換後のパスに対して、以下のいずれかを満たすか確認します。
     - 環境変数 `PATH` 内にコマンド名として存在する（`exec.LookPath` が成功する）。
     - 絶対パスまたは相対パスとしてファイルが存在する（`os.Stat` が成功する）。
  4. 最初に条件を満たしたコマンドを採用し、それを実行します。
  5. いずれの候補も解決できなかった場合は、リストの最初の候補を採用し（または適切なエラーをスローし）、起動を試みます。

### 4. ハードコードされた個別ロジックの排除
- 新しい汎用的ロジックが実装された後、[launcher.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/launcher.go) に追加した `antigravity-ide.cmd` に対する個別のハードコーディングを完全に削除します。

### 5. デフォルトYAMLテンプレートのコメント更新
- `defaultYAMLTemplate`（`editor.yaml` 自動生成用の組み込みテキスト）内のエディタ設定例やスキーマの説明コメントを更新します。
- `cmds` フィールドおよび動的プレースホルダー（`{home}`、`{localappdata}`）の意味とフォールバックの動作について説明するコメントを記述し、ユーザーが手動で設定を追加する際のガイドとします。

## 実現方針 (Implementation Approach)

### データモデルの拡張 (`pkg/editor/config.go`)
- `EditorConfig` および `PlatformConfig` 構造体に `Cmds []string` フィールド（YAMLタグ: `cmds`）を追加します。
- `defaultYAMLTemplate` 内の `ag` の設定定義、および `defaultConfig()` 内の構造体生成処理において、今回実装した Antigravity 用のパスを `cmds` 配列として定義します。

### 動的バインドと解決 (`pkg/editor/launcher.go`)
- コマンド解決を行う `resolveCommand` ヘルパー関数を `launcher.go` に導入します。
- リストを上から順番にプレースホルダー置換し、`exec.LookPath` や `os.Stat` を用いて有効なコマンドを選択します。

## 検証シナリオ (Verification Scenarios)

### シナリオ1：PATHが通っていない環境でのフォールバック確認
1. Windows環境において、PATHに `antigravity-ide.cmd` が含まれていない状態にします。
2. `editor.yaml` の `cmds` 定義に `{home}/AppData/Local/Programs/Antigravity IDE/bin/antigravity-ide.cmd` が含まれていることを確認します。
3. `tt open {branch} --editor ag --dry-run` を実行します。
4. `[DRY-RUN] C:\Users\<Username>\AppData\Local\Programs\Antigravity IDE\bin\antigravity-ide.cmd ...` のように絶対パスへ解決されることを確認します。

### シナリオ2：PATHが通っている環境での優先解決確認
1. Windows環境において、PATHにダミーの `antigravity-ide.cmd` または本物を通します。
2. `tt open {branch} --editor ag --dry-run` を実行します。
3. リストで優先度の高い `antigravity-ide.cmd`（絶対パスに展開されない生のコマンド名）で解決されることを確認します。

## テスト項目 (Testing for the Requirements)

### 自動化テスト
- **単体テスト (`pkg/editor/launcher_test.go` または `config_test.go`)**:
  - `resolveCommand` ロジックのテストを記述し、以下の項目を検証します。
    - `{home}` プレースホルダーが正しく置換されること。
    - 複数候補がある場合に、PATH上にあるものが優先して解決されること。
    - PATH上に存在しないが直接ファイルが存在するパス（`os.Stat` 経由）が解決されること。
- **検証スクリプト**:
  - `./scripts/process/build.sh`
