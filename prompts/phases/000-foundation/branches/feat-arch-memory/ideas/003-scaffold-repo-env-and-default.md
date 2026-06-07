# 仕様書: ScaffoldテンプレートリポジトリURLの変更および環境変数による指定機能の追加

## 1. 背景
現在、`tt scaffold` コマンドが使用するデフォルトのテンプレートリポジトリURLは、`https://github.com/axsh/tokotachi-scaffolds` に固定されています。
リポジトリの統合および整理に伴い、このデフォルトURLを `https://github.com/axsh/tokotachi` に変更する必要があります。
また、開発やデバッグの利便性向上のため、環境変数からこのURLを動的に指定できるようにし、環境変数が指定されていない場合にはデフォルトのハードコードされたURLにフォールバックする仕組みを導入します。

## 2. 要件

### 2.1 デフォルトURLの変更
- テンプレート取得用のデフォルトリポジトリURLを `https://github.com/axsh/tokotachi-scaffolds` から `https://github.com/axsh/tokotachi` に変更する。

### 2.2 環境変数 `TT_CONTENT_REPO` によるオーバーライド
- アプリケーション起動時に環境変数 `TT_CONTENT_REPO` を読み取る。
- `TT_CONTENT_REPO` が設定されている（かつ空値ではない）場合、テンプレート取得元のデフォルトURLとしてその値を使用する。
- 環境変数が設定されていない、または空値である場合は、ハードコードされたデフォルトURL `https://github.com/axsh/tokotachi` にフォールバックする。

### 2.3 優先順位の定義
テンプレート取得時に使用するリポジトリURLの優先順位は以下のように制御される。
1. `tt scaffold --repo <URL>` コマンドラインフラグによる指定（最優先）
2. 環境変数 `TT_CONTENT_REPO` の値
3. ハードコードされたデフォルトURL `https://github.com/axsh/tokotachi`

---

## 3. 実現方針

### 3.1 影響範囲
- [pkg/scaffold/scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go)
  - `defaultRepoURL` 定数定義の変更。
  - URL解決ロジック（`opts.RepoURL` または `repoURL` が空である場合の処理）の修正。
- 関連するテストコード（`scaffold_test.go` など）。

### 3.2 詳細設計

#### 3.2.1 定数定義の変更
`defaultRepoURL` 定数の値を変更します。
```go
const defaultRepoURL = "https://github.com/axsh/tokotachi"
```

#### 3.2.2 URL解決ロジックの修正
`scaffold.go` 内の `Run` 関数、`List` 関数、および `Apply` 関数等において、URLが指定されていない場合の処理を修正します。
URL解決の共有ヘルパー関数（例: `resolveRepoURL(specifiedURL string) string`）を導入し、以下のロジックで解決します。

```go
func resolveRepoURL(specifiedURL string) string {
    if specifiedURL != "" {
        return specifiedURL
    }
    if envURL := os.Getenv("TT_CONTENT_REPO"); envURL != "" {
        return envURL
    }
    return defaultRepoURL
}
```

このヘルパー関数を、`scaffold.go` 内の以下の箇所に適用します。
- `Run` 関数の冒頭: `opts.RepoURL = resolveRepoURL(opts.RepoURL)`
- `List` 関数の冒頭: `repoURL = resolveRepoURL(repoURL)`
- `Apply` 関数の冒件（依存関係解決など）: `opts.RepoURL = resolveRepoURL(opts.RepoURL)`

---

## 4. 検証シナリオ

### シナリオ1: フラグも環境変数も指定しない場合
- **条件**:
  - `TT_CONTENT_REPO` 環境変数を設定しない。
  - `--repo` フラグを指定しない。
- **期待される挙動**:
  - ハードコードされたデフォルトURL `https://github.com/axsh/tokotachi` が使用される。

### シナリオ2: 環境変数のみ指定する場合
- **条件**:
  - `TT_CONTENT_REPO` 環境変数に `https://github.com/some-owner/some-repo` を設定する。
  - `--repo` フラグを指定しない。
- **期待される挙動**:
  - 環境変数に指定した `https://github.com/some-owner/some-repo` が使用される。

### シナリオ3: 環境変数とフラグの両方を指定する場合
- **条件**:
  - `TT_CONTENT_REPO` 環境変数に `https://github.com/some-owner/some-repo` を設定する。
  - `--repo` フラグに `https://github.com/flag-owner/flag-repo` を指定する。
- **期待される挙動**:
  - コマンドラインフラグが優先され、`https://github.com/flag-owner/flag-repo` が使用される。

---

## 5. テスト項目

### 5.1 自動化された検証手順

#### 単体テスト (Unit Test)
- `pkg/scaffold/scaffold_test.go`（もしくは新規テストファイル）において、`resolveRepoURL` ロジック（または環境変数に応じたURL解決の挙動）を検証するテーブル駆動テストを追加する。
- テスト内で `t.Setenv("TT_CONTENT_REPO", ...)` を使用し、各シナリオ（設定なし、設定あり、フラグ競合）で正しいURLが返るか検証する。
- テスト実行コマンド:
  ```bash
  scripts/process/build.sh --skip-frontend --skip-etc
  ```

#### 統合テスト (Integration Test)
- コマンド実行時のURL解決が意図通り行われているかを検証するため、`tests/` 配下に統合テストを追加または修正する。
- `TT_CONTENT_REPO` 環境変数にダミーの不正なURL（例: `https://github.com/invalid-owner/invalid-repo`）を設定し、`tt scaffold --list` を実行して、エラーメッセージ内に設定した不正なリポジトリ名が含まれているかをアサーションする（これにより、実際にその環境変数で指定されたリポジトリにアクセスしようとしたことを実証する）。
- テスト実行コマンド:
  ```bash
  # 事前にビルドを成功させておくこと
  scripts/process/build.sh --skip-frontend --skip-etc && scripts/process/integration_test.sh --categories common
  ```
