# 仕様書: Scaffoldテンプレートダウンロードにおけるブランチ名の指定機能追加

## 1. 背景
現在、テンプレート取得元リポジトリのブランチ名は、共有ライブラリの `internal/github` パッケージ内で `"main"` に固定されています。
特定の開発ブランチやベータブランチ等からテンプレートをダウンロードしてテスト・実行できるようにするため、リポジトリURLと同様にブランチ名も動的に指定可能にする必要があります。
これを、コマンドラインフラグ、環境変数、およびハードコードされたデフォルト（`main`）の3層の優先順位で制御できる仕組みを導入します。

## 2. 要件

### 2.1 デフォルトブランチ名の定義
- デフォルトのブランチ名は `"main"` とする。

### 2.2 環境変数 `TT_CONTENT_BRANCH` によるオーバーライド
- 起動時に環境変数 `TT_CONTENT_BRANCH` を読み取り、値が設定されている場合はデフォルトのブランチ名としてそれを使用する。
- 環境変数が設定されていない、または空値の場合は、デフォルトの `"main"` ブランチにフォールバックする。

### 2.3 コマンド引数（フラグ） `--branch` による指定
- `tt scaffold` コマンドに `--branch` フラグを追加し、コマンドラインから直接ブランチ名を指定できるようにする。

### 2.4 優先順位の制御
ブランチ名の解決は以下の優先順位で制御される。
1. `tt scaffold --branch <BRANCH>` コマンドラインフラグによる指定（最優先）
2. 環境変数 `TT_CONTENT_BRANCH` の値
3. ハードコードされたデフォルト値 `"main"`

---

## 3. 実現方針

### 3.1 影響範囲
- [features/tt/cmd/scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/features/tt/cmd/scaffold.go)
  - Cobraコマンドのフラグ `--branch` の追加。
  - `scaffold.RunOptions` へのブランチ情報の追加、および `scaffold.List` 呼び出し時の引数追加。
- [pkg/scaffold/scaffold.go](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/pkg/scaffold/scaffold.go)
  - `RunOptions` 構造体への `RepoBranch string` フィールドの追加。
  - ブランチ解決用の定数 `defaultRepoBranch = "main"` とヘルパー関数 `resolveRepoBranch(specifiedBranch string) string` の追加。
  - `Run`、`Apply`、`List` 各関数内でのブランチ名の解決、および `github.Client.Branch` への設定。
  - `List` 関数の引数シグネチャの変更（`repoBranch` 引数の追加）。
- 関連するテストコード。

### 3.2 詳細設計

#### 3.2.1 ブランチ解決ヘルパー関数の追加 (pkg/scaffold)
`pkg/scaffold/scaffold.go` に以下を追加します。

```go
const defaultRepoBranch = "main"

func resolveRepoBranch(specifiedBranch string) string {
    if specifiedBranch != "" {
        return specifiedBranch
    }
    if envBranch := os.Getenv("TT_CONTENT_BRANCH"); envBranch != "" {
        return envBranch
    }
    return defaultRepoBranch
}
```

#### 3.2.2 `scaffold.go` 内の適用
`scaffold.go` 内で `github.NewClient` により生成したクライアント（`downloader` など）の `Branch` フィールドに対して、解決したブランチ名をセットします。

- **`Run` 関数**:
  ```go
  opts.RepoBranch = resolveRepoBranch(opts.RepoBranch)
  // ...
  // fetchAndResolveEntry 内で downloader.Branch = opts.RepoBranch を設定
  ```
- **`List` 関数**:
  ```go
  func List(repoURL string, repoBranch string, repoRoot string, filterCategory string) ([]ScaffoldEntry, error) {
      repoURL = resolveRepoURL(repoURL)
      repoBranch = resolveRepoBranch(repoBranch)
      
      downloader, err := github.NewClient(repoURL)
      if err != nil {
          return nil, err
      }
      downloader.Branch = repoBranch
      // ...
  }
  ```

---

## 4. 検証シナリオ

### シナリオ1: フラグも環境変数も指定しない場合
- **条件**:
  - `TT_CONTENT_BRANCH` 環境変数を設定しない。
  - `--branch` フラグを指定しない。
- **期待される挙動**:
  - デフォルトブランチ名 `"main"` が使用される。

### シナリオ2: 環境変数のみ指定する場合
- **条件**:
  - `TT_CONTENT_BRANCH` 環境変数に `"develop"` を設定する。
  - `--branch` フラグを指定しない。
- **期待される挙動**:
  - 環境変数に指定した `"develop"` ブランチが使用される。

### シナリオ3: 環境変数とフラグの両方を指定する場合
- **条件**:
  - `TT_CONTENT_BRANCH` 環境変数に `"develop"` を設定する。
  - `--branch` フラグに `"feature/test"` を指定する。
- **期待される挙動**:
  - コマンドラインフラグが優先され、`"feature/test"` ブランチが使用される。

---

## 5. テスト項目

### 5.1 自動化された検証手順

#### 単体テスト (Unit Test)
- `pkg/scaffold/scaffold_test.go` において、`resolveRepoBranch` 関数の挙動を検証するテーブル駆動テストを追加する（環境変数なし、環境変数あり、フラグ競合の各組み合わせ）。
- テスト実行コマンド:
  ```bash
  scripts/process/build.sh --backend-only
  ```

#### 統合テスト (Integration Test)
- `TT_CONTENT_BRANCH` に存在しない無効なブランチ名（例: `nonexistent-branch-for-testing`）を環境変数に指定して `tt scaffold --list` を実行し、APIがそのブランチを参照しようとしてエラーになる（エラーメッセージ内にブランチ名が含まれる等）ことを検証する。
- テスト実行コマンド:
  ```bash
  scripts/process/build.sh && scripts/process/integration_test.sh --categories common
  ```
