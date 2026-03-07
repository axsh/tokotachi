# 009 — 配布スクリプト実装

## 背景 (Background)

仕様書 007 で定義したアーティファクトフロー（features → manifests → build → dist → release → publish）に対し、`scripts/dist/` 配下のスクリプトは現在スタブ（ヘルプ表示のみ）である。CLIツールを実際にビルド・リリース・公開するには、これらのスクリプトに実際のロジックを実装する必要がある。

### 現状

```
scripts/dist/
  build            # スタブ (exit 1)
  release          # スタブ (exit 1)
  publish          # スタブ (exit 1)
  dev              # スタブ (exit 1)
  install-tools    # スタブ (exit 1)
  bootstrap-tools  # スタブ (exit 1)
  README.md        # 完成済み
```

### 利用可能なメタデータ

- `tools/manifests/tools.yaml` — ツール一覧
- `tools/manifests/<tool-id>.yaml` — 個別マニフェスト (platforms, archive形式等)
- `features/<name>/feature.yaml` — Feature メタデータ (entrypoint等)
- `packaging/checksums/policy.yaml` — チェックサムポリシー (sha256)

## 要件 (Requirements)

### 必須要件

1. **`scripts/dist/_lib.sh` (共通ライブラリ) の作成**
   - YAML読み取り: `python` を使って YAML をパースし、値を取得する関数
   - 色付き出力: `[INFO]`, `[PASS]`, `[FAIL]`, `[WARN]` メッセージ関数
   - プラットフォーム検出: 現在の OS/Arch を検出する関数
   - マニフェスト読み取り: `tools/manifests/<tool-id>.yaml` からフィールドを取得する関数
   - ツール一覧取得: `tools/manifests/tools.yaml` から全ツールIDを取得する関数

2. **`scripts/dist/build` の実装**
   - `tools/manifests/<tool-id>.yaml` を読み取り、定義されたプラットフォームごとに `go build` を実行する
   - 出力先: `dist/<tool-id>/` (例: `dist/devctl/devctl_linux_amd64`)
   - 環境変数 `GOOS`, `GOARCH`, `CGO_ENABLED=0` を設定してクロスコンパイル
   - `feature.yaml` の `entrypoint` (例: `./cmd/devctl`) をビルド対象とする
   - 引数: `<tool-id>` (必須)
   - 成功/失敗を色付きメッセージで表示

3. **`scripts/dist/release` の実装**
   - ビルド済みバイナリからリリース成果物（アーカイブ + チェックサム）を作成する
   - アーカイブ形式: tar.gz (Linux/macOS), zip (Windows)
   - アーカイブファイル名: `<tool-id>_<os>_<arch>.<ext>` (例: `devctl_linux_amd64.tar.gz`)
   - チェックサム: sha256、`checksums.txt` として出力
   - 出力先: `dist/<tool-id>/<version>/` (例: `dist/devctl/v1.0.0/`)
   - 引数: `<tool-id>` (必須), `<version>` (必須)

4. **`scripts/dist/publish` の実装**
   - リリース成果物を GitHub Releases にアップロードする
   - `gh` CLI を使用して GitHub Release を作成し、アーカイブとチェックサムをアタッチする
   - リリースノートは `releases/notes/latest.md` から読み取る（存在しない場合は自動生成）
   - Homebrew / Scoop への公開は、インストーラーテンプレートからマニフェストを生成して表示する（フォーミュラの実際のpushはユーザーが手動で行う）
   - 引数: `<tool-id>` (必須), `<version>` (必須)
   - `gh` CLI が未インストールの場合はエラーを表示して終了

5. **`scripts/dist/install-tools` の実装**
   - ビルド済みバイナリをローカルの `bin/` にコピーしてインストールする
   - `--all` で全ツール、個別指定で特定ツールのみ
   - ネイティブプラットフォーム (現在のOS/Arch) のバイナリのみをインストール
   - バイナリが未ビルドの場合は `build` を先に自動実行する

6. **`scripts/dist/dev` の実装**
   - 指定された Feature の開発環境を起動する
   - 内部的に `devctl up <feature-name>` を呼び出すラッパー
   - `bin/devctl` が存在しない場合は `install-tools devctl` を先に実行する
   - 引数: `<feature-name>` (必須)

7. **`scripts/dist/bootstrap-tools` の実装**
   - 新規開発者向けの初期セットアップスクリプト
   - 実行内容:
     1. Go がインストールされているか確認（未インストールならエラー）
     2. `tools/manifests/tools.yaml` から全ツールを取得
     3. 全ツールをビルド (`build`)
     4. 全ツールをインストール (`install-tools --all`)
   - 結果サマリーを表示

## 実現方針 (Implementation Approach)

### 共通設計 (`_lib.sh`)

```bash
# 色付き出力関数
info()  { echo -e "\033[0;34m[INFO]\033[0m $*"; }
pass()  { echo -e "\033[0;32m[PASS]\033[0m $*"; }
fail()  { echo -e "\033[1;31m[FAIL]\033[0m $*"; }
warn()  { echo -e "\033[1;33m[WARN]\033[0m $*"; }

# YAML値取得 (python)
yaml_get() { python -c "import yaml,sys; d=yaml.safe_load(open('$1')); print(d$2)" 2>/dev/null; }

# プラットフォーム情報の検出
detect_os()   { uname -s | tr '[:upper:]' '[:lower:]' → linux/darwin/windows }
detect_arch() { uname -m → amd64/arm64 に変換 }

# マニフェスト読み取り
manifest_path() { echo "tools/manifests/${1}.yaml"; }
get_platforms() { yaml から os/arch ペアのリストを返す }
```

### `build` スクリプトのフロー

```
1. tool-id 引数を受け取る
2. tools/manifests/<tool-id>.yaml を読み取る
   → feature_path, binary_name, main_package, platforms[] を取得
3. platforms[] をループ:
   a. GOOS, GOARCH を設定
   b. 出力パス: dist/<tool-id>/<binary_name>_<os>_<arch>[.exe]
   c. go build -o <output> <feature_path>/<main_package>
   d. 成功/失敗をログ
4. 全プラットフォームの結果サマリーを表示
```

### `release` スクリプトのフロー

```
1. tool-id, version 引数を受け取る
2. dist/<tool-id>/ からビルド済みバイナリを取得
3. 出力先: dist/<tool-id>/<version>/ を作成
4. プラットフォームごとにアーカイブ作成:
   a. Linux/macOS: tar czf <name>.tar.gz <binary>
   b. Windows: zip <name>.zip <binary>
5. sha256sum でチェックサム生成 → checksums.txt
6. 結果サマリーを表示
```

### `publish` スクリプトのフロー

```
1. tool-id, version 引数を受け取る
2. gh CLI の存在を確認 (なければエラー)
3. dist/<tool-id>/<version>/ のファイルを確認 (なければエラー)
4. gh release create <tag> --title "<tool> <version>" --notes-file <notes>
5. gh release upload <tag> dist/<tool-id>/<version>/*
6. Homebrew/Scoop テンプレートから生成したマニフェストを表示
   (実際のリポジトリへのpushはユーザーが手動で行う)
```

### `install-tools` スクリプトのフロー

```
1. --all または tool-id 引数を受け取る
2. --all の場合: tools/manifests/tools.yaml から全ツールを取得
3. 各ツールについて:
   a. ネイティブプラットフォームのバイナリを特定
   b. dist/<tool-id>/<binary> が存在しなければ build を実行
   c. bin/<binary-name> にコピー
   d. chmod +x で実行権限を付与
```

### `dev` スクリプトのフロー

```
1. feature-name 引数を受け取る
2. bin/devctl が存在するか確認
   → なければ install-tools devctl を実行
3. bin/devctl up <feature-name> を実行
```

### `bootstrap-tools` スクリプトのフロー

```
1. Go の存在を確認 (go version)
2. tools/manifests/tools.yaml から全ツールIDを取得
3. 各ツールを build
4. install-tools --all を実行
5. 結果サマリーを表示
```

## 検証シナリオ (Verification Scenarios)

1. `./scripts/dist/build devctl` を実行 → `dist/devctl/` に5つのバイナリが生成される:
   - `devctl_linux_amd64`
   - `devctl_linux_arm64`
   - `devctl_darwin_amd64`
   - `devctl_darwin_arm64`
   - `devctl_windows_amd64.exe`
2. `./scripts/dist/release devctl v0.1.0` を実行 → `dist/devctl/v0.1.0/` に以下が生成される:
   - `devctl_linux_amd64.tar.gz`
   - `devctl_linux_arm64.tar.gz`
   - `devctl_darwin_amd64.tar.gz`
   - `devctl_darwin_arm64.tar.gz`
   - `devctl_windows_amd64.zip`
   - `checksums.txt`
3. `./scripts/dist/install-tools devctl` を実行 → `bin/devctl` (ネイティブ) が作成される
4. `./scripts/dist/dev devctl` を実行 → `devctl up devctl` が呼び出される
5. `./scripts/dist/bootstrap-tools` を実行 → 全ツールがビルド＆インストールされる
6. `./scripts/dist/publish devctl v0.1.0` を実行 → `gh` CLI がなければエラー表示
7. 引数なしで各スクリプトを実行 → usage メッセージが表示される (exit 1)
8. 存在しない tool-id を指定 → エラーメッセージが表示される (exit 1)

## テスト項目 (Testing for the Requirements)

| 要件 | 検証コマンド |
|---|---|
| ビルド成功 (既存) | `./scripts/process/build.sh` |
| クロスビルド成功 | `./scripts/dist/build devctl` 後に `ls dist/devctl/` で5バイナリ確認 |
| リリース成果物作成 | `./scripts/dist/release devctl v0.1.0` 後に `ls dist/devctl/v0.1.0/` で6ファイル確認 |
| チェックサム検証 | `cd dist/devctl/v0.1.0 && sha256sum -c checksums.txt` |
| ローカルインストール | `./scripts/dist/install-tools devctl` 後に `./bin/devctl version` |
| 開発環境起動 | `./scripts/dist/dev devctl` が `devctl up` を呼び出すこと |
| 初期セットアップ | `./scripts/dist/bootstrap-tools` が全ツールをビルド＆インストールすること |
| publish (gh未導入時) | `./scripts/dist/publish devctl v0.1.0` がエラーを返すこと |
| エラーハンドリング | `./scripts/dist/build nonexistent` が exit 1 を返すこと |
