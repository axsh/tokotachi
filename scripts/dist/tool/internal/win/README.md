# Tokotachi (tt) Windows リリースパッケージ

本パッケージには、Tokotachi コマンドラインツール（`tt`）および Windows 環境用の簡易インストーラーが含まれています。

## パッケージ内容
- `tt.exe` : Tokotachi コマンドラインツールの実行バイナリ
- `install.ps1` : インストール用 PowerShell スクリプト
- `uninstall.ps1` : アンインストール用 PowerShell スクリプト
- `README.md` : 本ファイル（説明書）

---

## インストール手順

### 推奨：インストーラースクリプトを使用した自動インストール
PowerShell スクリプトを実行することで、自動的に `tt.exe` を適切なフォルダへ配置し、`PATH`（環境変数）への登録を行います。管理者権限は不要です。

1. 本 ZIP ファイルを展開（解凍）します。
2. 展開したフォルダ内で PowerShell（またはターミナル）を開きます。
3. 以下のコマンドを実行します：
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\install.ps1
   ```
4. インストール完了メッセージが表示されたら、**新しいターミナルウィンドウを開き直し**、以下を実行して正しくインストールされたか確認します：
   ```cmd
   tt --help
   ```

### 手動インストール
1. `tt.exe` を任意のフォルダ（例: `C:\bin` など）に移動します。
2. そのフォルダのパスをシステムのユーザー環境変数 `PATH` に追加します。

---

## アンインストール手順
自動インストールで追加されたファイルを削除し、`PATH` から登録を解除します。

1. `uninstall.ps1` があるフォルダ内で PowerShell を開きます。
2. 以下のコマンドを実行します：
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\uninstall.ps1
   ```
