# 仕様書: tt scaffold をカレントディレクトリ基準で動作させる

## 1. 背景 (Background)

現在、`tt scaffold` コマンドは `--root` オプションが指定されていない場合、自動的に親ディレクトリを遡って `git rev-parse --show-toplevel` により Git のルートディレクトリを検出し、そこを作理基準（RepoRoot）とします。

しかし、この挙動には以下の課題があります。
1. ユーザーがホームディレクトリ（`C:\Users\username` 等）に誤って `.git` ディレクトリを作っていた場合、ホームディレクトリがプロジェクトルートと判定され、そこに大量のファイルが展開されてしまう。
2. 展開後のパーミッション適用時に `filepath.WalkDir` がホームディレクトリ配下の全ファイルを走査するため、アクセス権限のないシステムフォルダ（`AppData/Local/ElevatedDiagnostics` など）に進入した時点で `Access is denied` エラーが発生し、処理が中断する。
3. 他の `tt` コマンド（`create`, `up`, `open` など）は `os.Getwd()` （カレントディレクトリ）をそのままルートとする仕様になっており、`scaffold` だけが自動で Git ルートを探索する挙動になっていて一貫性がない。

これらを解消するため、`tt scaffold` のデフォルト動作ルートを他のコマンド同様「カレントディレクトリ固定」に変更します。

---

## 2. 要件 (Requirements)

*   `tt scaffold` 実行時に `--root` オプションが指定されていない場合、Git ルートの自動探索（`git rev-parse --show-toplevel`）を行わず、デフォルトでカレントディレクトリ（CWD）をルートディレクトリ（`opts.RepoRoot`）として使用すること。
*   明示的に `--root` オプションが指定された場合は、指定されたパスを従来通りルートディレクトリとして使用すること。
*   この変更による、他の機能（テンプレートのフェッチやパラメータ収集、展開処理など）への影響がないこと。

---

## 3. 実実現方針 (Implementation Approach)

*   **変更対象ファイル**: `features/tt/cmd/scaffold.go` ([scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-download-scaffold/features/tt/cmd/scaffold.go))
*   **変更箇所**: `resolveRepoRoot` 関数
    *   `git rev-parse --show-toplevel` を実行している部分を削除し、直接 `os.Getwd()` の結果（エラー時は `.`）を返すように簡略化します。
    
```go
// 修正後イメージ
func resolveRepoRoot(rootPath string) string {
	if rootPath != "" {
		return rootPath
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
```

---

## 4. 検証シナリオ (Verification Scenarios)

1.  **デフォルト実行の検証 (CWDが基準となること)**:
    *   Git リポジトリ配下のサブディレクトリ `features` などに移動し、`tt scaffold` コマンドを実行する。
    *   テンプレートファイルが Git のルートではなく、現在実行したサブディレクトリ（CWD）配下に正しく展開されることを確認する。
2.  **`--root` オプションの検証 (指定ルートが優先されること)**:
    *   別の任意のディレクトリを `--root` オプションで指定して `tt scaffold` を実行する。
    *   指定したディレクトリ配下にテンプレートが展開されることを確認する。

---

## 5. テスト項目 (Testing for the Requirements)

要件が正しく満たされたことを自動化テストおよびビルド実行によって確認します。

### ビルド・全体検証

1.  **ビルド＋単体テストの実行**:
    ```bash
    scripts/process/build.sh
    ```
2.  **統合テスト（templateカテゴリ）の実行**:
    `tt scaffold` やテンプレートの適用処理に関する統合テストが正常に通過することを確認します。
    ```bash
    scripts/process/integration_test.sh --categories "template"
    ```
