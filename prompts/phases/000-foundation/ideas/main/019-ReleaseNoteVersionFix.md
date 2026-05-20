# 仕様書: リリースノート生成の差分限定化およびバージョン増分フォーマットの厳密化

## 1. 背景 (Background)
1. **マージコミット検出の不具合**
   現在、リリース公開スクリプト（Bash）とリリースノート自動生成ツール（Go）におけるGitタグの命名規則の解釈が一致していません。
   - [publish.sh](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/publish.sh) は `v0.4.5` のようなプレフィックスなしのタグを作成します。
   - [history.go](file:///c:/Users/yamya/myprog/tokotachi/features/release-note/internal/git/history.go) は `tt-v0.4.5` のようなプレフィックス付きのタグを検索します。
   この不一致により、最新のタグが取得できず、Gitのヒストリ全体からマージコミット（全27ブランチ）を収集してしまっています。直近のタグからの差分のみを対象にするよう、タグの検索ロジックを修正する必要があります。

2. **バージョンカウントアップにおける問題**
   - **セマンティックバージョニング（SemVer）への非準拠**: 現行の `github-upload.sh` では、インクリメント指定時に単に各桁を加算しているため、上位バージョンが上がった際に下位バージョンが `0` にリセットされません（例: `v0.4.5` + `+v0.1.0` = `v0.5.5` になり、本来の `v0.5.0` にならない）。
   - **冗長なインクリメント指定フォーマット**: `+v0.1.0` のような末尾の `.0` は意味を持ちません。マイナーアップなら `+v0.1`、メジャーアップなら `+v1` のように、簡潔な表記を強制すべきです。末尾に `.0` を含む指定は、バリデーションでエラーとして検出・制限する必要があります。

---

## 2. 要件 (Requirements)
1. **タグ検出の一致と差分収集**
   - Goツールのタグ検出ロジックにおいて、`v[0-9]` で始まる標準的なセマンティックバージョニング形式のタグを検出できるようにします。
   - 最新タグが正しく検出された場合、前回のタグから `HEAD` までの範囲（`Tag..HEAD`）のマージコミットのみを抽出するようにし、差分対象のブランチのみをリリースノート生成の対象とします。

2. **バージョンインクリメントにおけるSemVer準拠の繰り上がり**
   - メジャーのインクリメント（例: `+v1`）が指定された場合、マイナーとパッチを `0` にリセットします。
   - マイナーのインクリメント（例: `+v0.1`）が指定された場合、パッチを `0` にリセットします。
   - パッチのインクリメント（例: `+v0.0.1`）が指定された場合は、パッチ番号のみを加算します。

3. **インクリメント指定フォーマットの制限（エラーチェック）**
   - バージョンインクリメント指定（`+*`）において、以下のフォーマットのみを許可します：
     - メジャーアップ： `+v[1-9][0-9]*` （例: `+v1`）
     - マイナーアップ： `+v0\.[1-9][0-9]*` （例: `+v0.1`）
     - パッチアップ： `+v0.0\.[1-9][0-9]*` （例: `+v0.0.1`）
   - 末尾に不要な `.0` が含まれる指定（例: `+v0.1.0`, `+v1.0.0`, `+v1.0` など）は無効とし、バリデーションエラーとして処理を中断します。

---

## 3. 実現方針 (Implementation Approach)

### 3.1. Goツール（`release-note`）の修正
- [history.go](file:///c:/Users/yamya/myprog/tokotachi/features/release-note/internal/git/history.go) の `GetLatestReleaseTag` において、`jqExpr` のフィルタを修正します。
  - 現在: `select(.tagName | startswith("%s-v"))`（例: `tt-v` を検索）
  - 変更後: `select(.tagName | test("^v[0-9]+"))`（`github-upload.sh` と同様に `v[0-9]` で始まる最新タグを取得）

### 3.2. Bashスクリプト（`github-upload.sh`）の修正
- **インクリメント指定フォーマットのバリデーション**
  [github-upload.sh](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/github-upload.sh) 内で、`+` から始まるバージョン引数（`VERSION_ARG`）を解析する際、以下の正規表現チェックを追加します。
  ```bash
  if [[ "$VERSION_ARG" == +* ]]; then
    # 許可するフォーマット: +v{major}, +v0.{minor}, +v0.0.{patch} (ただし末尾に.0は禁止)
    if [[ ! "$VERSION_ARG" =~ ^\+v([1-9][0-9]*|0\.[1-9][0-9]*|0\.0\.[1-9][0-9]*)$ ]]; then
      fail "Invalid increment format: '${VERSION_ARG}'."
      echo "  Allowed formats:"
      echo "    Major bump: +v1, +v2..."
      echo "    Minor bump: +v0.1, +v0.2..."
      echo "    Patch bump: +v0.0.1, +v0.0.2..."
      echo "  (Trailing .0 is not allowed. e.g. +v0.1.0 or +v1.0.0 is invalid)"
      exit 1
    fi
  fi
  ```
- **SemVer準拠のインクリメント計算**
  `compute_incremented_version` 関数を以下のように書き換えて、下位バージョンのリセット処理を行います。
  ```bash
  compute_incremented_version() {
    local cur="$1" inc="$2"

    parse_semver "$cur"
    local cur_major=$_major cur_minor=$_minor cur_patch=$_patch

    # 増分指定値から各桁を取得 (+を除いた上でパース)
    local raw_inc="${inc#+}"
    
    # 桁数が足りない場合に備えてプレースホルダを補ってパース
    # 例: v0.1 -> v0.1.0, v1 -> v1.0.0
    if [[ "$raw_inc" =~ ^v[0-9]+$ ]]; then
      raw_inc="${raw_inc}.0.0"
    elif [[ "$raw_inc" =~ ^v[0-9]+\.[0-9]+$ ]]; then
      raw_inc="${raw_inc}.0"
    fi

    parse_semver "$raw_inc"
    local inc_major=$_major inc_minor=$_minor inc_patch=$_patch

    local new_major=$cur_major
    local new_minor=$cur_minor
    local new_patch=$cur_patch

    if [[ $inc_major -gt 0 ]]; then
      new_major=$((cur_major + inc_major))
      new_minor=0
      new_patch=0
    elif [[ $inc_minor -gt 0 ]]; then
      new_minor=$((cur_minor + inc_minor))
      new_patch=0
    elif [[ $inc_patch -gt 0 ]]; then
      new_patch=$((cur_patch + inc_patch))
    fi

    echo "v${new_major}.${new_minor}.${new_patch}"
  }
  ```

---

## 4. 検証シナリオ (Verification Scenarios)

### シナリオ1: バリデーション機能の確認
1. `./scripts/dist/github-upload.sh tt +v0.1.0` を実行する。
   - **期待結果**: `Invalid increment format: '+v0.1.0'.` のようなエラーメッセージを出力し、異常終了する。
2. `./scripts/dist/github-upload.sh tt +v1.0.0` を実行する。
   - **期待結果**: 同様にエラーを出力し、異常終了する。
3. `./scripts/dist/github-upload.sh tt +v0.1` を実行する。
   - **期待結果**: バリデーションを通過する（ビルド処理等へ遷移する）。

### シナリオ2: バージョンインクリメント計算の確認
（現在の最新バージョンが `v0.4.5` の場合）
1. `+v0.1` を指定して実行する。
   - **期待結果**: `New: v0.5.0` と算出されること。
2. `+v1` を指定して実行する。
   - **期待結果**: `New: v1.0.0` と算出されること。
3. `+v0.0.1` を指定して実行する。
   - **期待結果**: `New: v0.4.6` と算出されること。

### シナリオ3: リリースノート差分収集の動作確認
1. 適当なブランチを新しくマージし、`./scripts/dist/github-upload.sh tt +v0.1` を実行する。
2. リリースノート生成処理（`Generate Release Notes`）のログを確認する。
   - **期待結果**:
     - `Latest release tag: v0.4.5` (または実際の最新タグ名) が検出されること。
     - 全てのブランチではなく、検出されたタグ以降に新しくマージされたブランチのみが検出されて要約処理が走ること。

---

## 5. テスト項目 (Testing for the Requirements)

### 5.1. 単体テスト
1. **Goツール (`release-note`)**:
   - [history_test.go](file:///c:/Users/yamya/myprog/tokotachi/features/release-note/internal/git/history_test.go) にテストを追加し、タグ取得ロジックが正しくプレフィックスなしの `vN.N.N` フォーマットのタグを最新順にソートして取得できることを検証します。
2. **自動ビルド検証**:
   - `scripts/process/build.sh` を実行し、既存および新規追加のテストを含むすべてのビルド・単体テストが通ることを検証します。
