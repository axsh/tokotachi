# github-upload.sh スクリプト仕様書

## 背景 (Background)

`scripts/dist/` ディレクトリには、CLIツールのビルド・リリース・公開を行うスクリプト群が存在する：

- `build` — クロスプラットフォームビルド
- `release` — リリース成果物（アーカイブ＋チェックサム）作成
- `publish` — GitHub Releases への公開

現在、これらを順番に手動で実行する必要がある。また、既存スクリプトにはファイル拡張子 `.sh` が付いていない。

### 課題

1. **手動実行の手間**: `build` → `release` → `publish` を毎回3コマンド実行する必要がある
2. **バージョン管理の手間**: バージョン番号を毎回手打ちする必要がある
3. **ファイル命名の一貫性**: `_lib.sh` のみ `.sh` 拡張子がついており、他のスクリプトには拡張子がない

## 要件 (Requirements)

### 必須要件

#### R1: `github-upload.sh` スクリプトの作成

`scripts/dist/github-upload.sh` を新規作成し、以下を連続して実行する：

1. `build` — ツールのビルド
2. `release` — リリース成果物の作成
3. `publish` — GitHub Releases への公開

#### R2: バージョン指定オプション

以下の3パターンでバージョンを指定可能にする：

| パターン | 例 | 説明 |
|----------|------|------|
| 絶対指定 | `v1.0.0` | 指定したバージョンをそのまま使用 |
| 増分指定 | `+v0.0.1` | 現在のバージョンに加算 |
| 省略 | (なし) | `+v0.0.1` と同等（パッチバージョンを1つ上げる） |

**使用例:**

```bash
# 絶対指定
./scripts/dist/github-upload.sh devctl v1.2.0

# 増分指定
./scripts/dist/github-upload.sh devctl +v0.1.0

# 省略（パッチバージョン+1）
./scripts/dist/github-upload.sh devctl
```

#### R3: バージョン形式のバリデーション

- バージョンは `v{n1}.{n2}.{n3}` 形式のみ許可（`n1`, `n2`, `n3` は非負整数）
- これにマッチしない形式はエラーとする
- 増分指定の場合は `+v{n1}.{n2}.{n3}` 形式

#### R4: バージョンダウングレード防止

- 絶対指定の場合、現在のバージョンより低いバージョンはエラーとする
- 現在のバージョンは `gh release list` や git tag から取得する

#### R5: 既存スクリプトのリネーム

`scripts/dist/` 配下の拡張子なしスクリプトに `.sh` 拡張子を付ける：

| 変更前 | 変更後 |
|--------|--------|
| `build` | `build.sh` |
| `release` | `release.sh` |
| `publish` | `publish.sh` |
| `dev` | `dev.sh` |
| `install-tools` | `install-tools.sh` |
| `bootstrap-tools` | `bootstrap-tools.sh` |

> [!IMPORTANT]
> リネーム後、以下の参照箇所も更新する必要がある：
> - `bootstrap-tools` 内での `build`、`install-tools` の呼び出し
> - `install-tools` 内での `build` の呼び出し
> - `dev` 内での `install-tools` の呼び出し
> - `_lib.sh` は既に `.sh` 拡張子が付いているため変更不要

#### R6: README.md の更新

スクリプトのリネームと `github-upload.sh` の新規追加に伴い、以下の README.md を更新する：

##### トップフォルダ `README.md`

- 「Build from Source」セクション（L140-146）のスクリプト参照を `.sh` 拡張子付きに更新
  - `./scripts/dist/bootstrap-tools` → `./scripts/dist/bootstrap-tools.sh`
  - `./scripts/dist/build devctl` → `./scripts/dist/build.sh devctl`
  - `./scripts/dist/install-tools devctl` → `./scripts/dist/install-tools.sh devctl`

##### `scripts/dist/README.md`

- Scripts テーブル内の全スクリプト名を `.sh` 拡張子付きに更新
- `github-upload.sh` の行をテーブルに追加
- Release Workflow セクションのコマンド例を `.sh` 拡張子付きに更新
- Artifact Flow セクションの図を更新
- `github-upload.sh` のワンコマンドリリース手順を追記

## 実現方針 (Implementation Approach)

### 現在のバージョン取得方法

既存の `publish` スクリプトでは、タグ名を `{tool-id}-{version}` 形式（例：`devctl-v1.0.0`）で管理している。

現在のバージョンは以下の方法で取得する：

```bash
# GitHub Releases から最新のタグを取得
gh release list --limit 1 --json tagName --jq '.[0].tagName'
# 例: "devctl-v1.0.0" → "v1.0.0" を抽出
```

もし一度もリリースされていない場合は `v0.0.0` をデフォルトとする。

### バージョン比較ロジック

セマンティックバージョニングの各コンポーネント（major, minor, patch）を数値として比較する：

```
v1.2.3 → major=1, minor=2, patch=3
```

比較は `major` → `minor` → `patch` の順で行い、新しいバージョンが現在のバージョン以下の場合エラーとする。

### スクリプト構成

```bash
#!/usr/bin/env bash
# All-in-one: build → release → publish to GitHub
# Usage: ./scripts/dist/github-upload.sh <tool-id> [version|+increment]

# 1. 引数パース・バリデーション
# 2. 現在のバージョン取得
# 3. バージョン算出（絶対 or 増分）
# 4. バージョンバリデーション（ダウングレードチェック）
# 5. build.sh 呼び出し
# 6. release.sh 呼び出し
# 7. publish.sh 呼び出し
```

### バージョン増分計算

```
現在: v1.2.3
+v0.0.1 → v1.2.4  (パッチ増分)
+v0.1.0 → v1.3.0  (マイナー増分、パッチはリセットしない＝単純加算)
+v1.0.0 → v2.2.3  (メジャー増分、同上)
```

> [!NOTE]
> 増分は各コンポーネントの単純加算とする。セマンティックバージョニングの慣例（メジャー更新時にマイナー・パッチをリセット）は適用しない。

## 検証シナリオ (Verification Scenarios)

### シナリオ1: 増分指定でのアップロード（正常系）

1. `./scripts/dist/github-upload.sh devctl +v0.0.1` を実行
2. 現在のGitHubリリースから最新バージョンが取得される
3. パッチバージョンが+1されたバージョンが算出される
4. `build.sh devctl` → `release.sh devctl v{new}` → `publish.sh devctl v{new}` が順に実行される

### シナリオ2: 絶対指定でのアップロード（正常系）

1. `./scripts/dist/github-upload.sh devctl v2.0.0` を実行
2. 現在のバージョンが `v2.0.0` 未満であることが確認される
3. `build.sh devctl` → `release.sh devctl v2.0.0` → `publish.sh devctl v2.0.0` が順に実行される

### シナリオ3: 省略時のデフォルト増分（正常系）

1. `./scripts/dist/github-upload.sh devctl` を実行
2. オプション省略なので `+v0.0.1` として扱われる
3. シナリオ1と同じ動作をする

### シナリオ4: 不正なバージョン形式（異常系）

1. `./scripts/dist/github-upload.sh devctl v1.0` を実行
2. バージョン形式エラーが表示される
3. 何も実行されずに終了コード1で終了する

### シナリオ5: バージョンダウングレード（異常系）

1. 現在のバージョンが `v1.2.0` の場合
2. `./scripts/dist/github-upload.sh devctl v1.1.0` を実行
3. ダウングレードエラーが表示される
4. 何も実行されずに終了コード1で終了する

### シナリオ6: 同一バージョン指定（異常系）

1. 現在のバージョンが `v1.2.0` の場合
2. `./scripts/dist/github-upload.sh devctl v1.2.0` を実行
3. 同一バージョンのエラーが表示される
4. 何も実行されずに終了コード1で終了する

### シナリオ7: 既存スクリプトのリネーム確認

1. `scripts/dist/` 配下の全スクリプトが `.sh` 拡張子付きになっている
2. スクリプト内の相互参照が更新されている
3. `scripts/dist/README.md` が更新されている（テーブル、コマンド例、図）
4. トップフォルダ `README.md` が更新されている（Build from Source セクション）

## テスト項目 (Testing for the Requirements)

### 自動検証

本タスクはシェルスクリプトの作成であり、実際のGitHub APIを呼び出すため、完全な自動テストは困難である。以下のレベルで検証を行う：

#### 1. 構文チェック

```bash
# bash の構文チェック
bash -n ./scripts/dist/github-upload.sh
```

#### 2. ビルドパイプライン

```bash
# プロジェクト全体のビルドとテスト
./scripts/process/build.sh
```

#### 3. スクリプト間参照の整合性確認

```bash
# リネーム後の参照が正しいか確認
grep -r "scripts/dist/" ./scripts/dist/ | grep -v ".sh"
```

### 手動検証

> [!IMPORTANT]
> 以下のテストは実際のGitHub Releases への公開を伴うため、ユーザーの判断で実施する。

1. **ドライラン**: `github-upload.sh` のバリデーション部分のみを手動で確認
2. **実際のアップロード**: 実際のバージョンを指定してGitHub Releasesへの公開を確認
