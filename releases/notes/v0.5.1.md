# Release Notes

## What's New

[New]

- エディタ起動設定をカスタマイズできる外部ファイル editor.yaml の管理方式を導入
    - ユーザーホームに自動生成され、詳細なコメント付きマニュアルを付与
    - system（自動生成/アプリ更新時自動上書き）と user（手動編集/優先）の二系統セクションで運用
    - 新規エディタや引数の追加・編集、OS別設定、コマンド候補リスト(cmd/cmds)や起動タイプ(type)の柔軟な定義が可能
    - args・cmd・パスなどで {home}, {localappdata}, {path}, {container}, {uri}等の動的プレースホルダー展開に対応
    - 新たなカスタムエディタ追加で--editorに任意キー指定が可能
    - 既存環境変数(TT_CMD_CODE等)の上書きも引き続きサポート
    - 初回未存在時は分かりやすい構造のデフォルトファイル自動生成
    - PATHや指定パスの探索・動的解決による自動起動

[Changed]

- エディタ起動ロジックが、個別Go実装からeditor.yaml & 1ファイル(CustomLauncher)のランチャー方式に統一
- 既存全エディタの起動コマンド/引数をyamlで一元管理（例: Windows上のagコマンドはantigravity-ide.cmd、他OSはantigravityに自動統一）
- コマンド候補の解決手順が一般化（PATH,ファイル有無,プレースホルダー等リスト順で評価）
- --editorで選択できるエディタ名がyaml記載の全キーになり、固定リスト制限が撤廃
- editor.yamlのsystemセクションはアプリ本体更新時に自動同期。user未カスタマイズかつ古い場合は自動アップデート

[Removed]

- ag.go, vscode.go, cursor.go など従来の個別エディタ用Goファイルの実装を全廃
- ハードコーディングされていた特定コマンド名やパス依存の個別起動ロジック（例: antigravity-ide.cmd直指定等）を削除
- --editor判定の定数リストによる制限・factory.goの起動分岐ロジックを廃止
