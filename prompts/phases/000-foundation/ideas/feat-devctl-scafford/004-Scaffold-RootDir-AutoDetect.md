# 004: Scaffold ルートディレクトリ自動検出

## 背景 (Background)

現在、`devctl scaffold` はテンプレートの展開先ルートディレクトリとして、コマンドを実行した**カレントワーキングディレクトリ (CWD)** をそのまま使用している（`os.Getwd()`）。

```go
// cmd/scaffold.go:40
repoRoot, err := os.Getwd()
```

この挙動は以下の問題を引き起こす：

- ユーザーがリポジトリ内のサブディレクトリ（例: `features/devctl/`）にいる状態で `devctl scaffold` を実行すると、テンプレートがサブディレクトリに展開されてしまい、意図しない結果になる
- `devctl` の他のコマンド（`up`, `open` 等）はGitワーキングツリー前提で動作しているため、scaffold だけがCWDベースなのは一貫性に欠ける

同様の問題は `cmd/doctor.go` と `cmd/common.go`（`InitContext`）にも存在するが、本仕様のスコープは `scaffold` コマンドに限定する。

## 要件 (Requirements)

### 必須要件

1. **Gitルート自動検出**: `devctl scaffold` は、デフォルトで `git rev-parse --show-toplevel` を使用してGitリポジトリのルートディレクトリを検出し、それをテンプレート展開先とする
2. **ハイブリッドフォールバック**: Gitリポジトリ外で実行された場合（`git rev-parse` が失敗した場合）は、従来通りCWDをルートディレクトリとして使用する
3. **`--cwd` フラグ**: `--cwd` フラグを指定した場合、Gitルート検出を無視し、CWDをルートディレクトリとして使用する
   - 用途: ユーザーの試運転、開発時の統合テストなどで、任意のディレクトリにテンプレートを展開したい場合

### 任意要件

- なし

## 実現方針 (Implementation Approach)

### ルート検出ロジック

`cmd/scaffold.go` の `runScaffold` 関数内で、以下のハイブリッド方式でルートディレクトリを決定する：

```
--cwd フラグが指定されている?
  → Yes: CWD を使用
  → No: git rev-parse --show-toplevel を実行
          → 成功: Git ルートを使用
          → 失敗: CWD にフォールバック
```

### 変更対象

#### `cmd/scaffold.go`
- `scaffoldFlagCwd` (bool型) フラグを追加
- `runScaffold` 内の `repoRoot` 決定ロジックを変更:
  - `--cwd` が指定されていない場合: `git rev-parse --show-toplevel` で検出を試み、失敗時にCWDにフォールバック
  - `--cwd` が指定されている場合: 従来通り `os.Getwd()` を使用

#### `internal/scaffold/scaffold.go`
- `resolveRepoRoot()` のようなヘルパー関数を追加してもよいが、ロジックが単純なため `cmd/scaffold.go` 側で完結させる方が適切

### `--cwd` フラグの設計

```
devctl scaffold --cwd [category] [name]
```

- `--cwd`: Gitルート自動検出を無効化し、カレントディレクトリを展開先として強制使用する
- フラグ名の根拠: 「current working directory を使え」という意味で直感的

## 検証シナリオ (Verification Scenarios)

### シナリオ1: Gitリポジトリのサブディレクトリから実行

1. Gitリポジトリ内のサブディレクトリ（例: `features/devctl/`）に移動
2. `devctl scaffold --yes` を実行
3. テンプレートがGitリポジトリのルートに展開されることを確認

### シナリオ2: `--cwd` フラグ付きで実行

1. 一時ディレクトリ（Gitリポジトリ外）を作成
2. そのディレクトリに移動
3. `devctl scaffold --cwd --yes` を実行
4. テンプレートがCWD（一時ディレクトリ）に展開されることを確認

### シナリオ3: Gitリポジトリ外から実行（フォールバック）

1. `/tmp/scaffold-test/` のようなGitリポジトリ外のディレクトリを作成
2. そのディレクトリに移動
3. `devctl scaffold --yes` を実行（`--cwd` なし）
4. `git rev-parse` が失敗し、CWDにフォールバックしてテンプレートが展開されることを確認

### シナリオ4: 既存統合テストが壊れないこと

1. 既存の統合テスト（`TestScaffoldDefault` 等）を実行
2. テストが引き続きパスすることを確認（テストはtmpDirをCWDにして実行するため、`--cwd` 相当の挙動になる）

## テスト項目 (Testing for the Requirements)

### 単体テスト

| 要件 | テスト内容 | ファイル |
|------|-----------|---------|
| Gitルート自動検出 | `resolveRepoRoot` 相当のロジックテスト | `cmd/scaffold.go` のロジック（もしヘルパー関数を切り出す場合） |

### 統合テスト

| 要件 | テスト内容 | 検証コマンド |
|------|-----------|-------------|
| 既存テスト維持 | `TestScaffoldDefault` がパスすること | `scripts/process/integration_test.sh` |
| `--cwd` フラグ | tmpDir で `--cwd` 指定時にCWDに展開される | `scripts/process/integration_test.sh` |

### ビルド検証

```bash
scripts/process/build.sh
```

### 統合テスト検証

```bash
scripts/process/integration_test.sh --specify TestScaffoldDefault
```

> [!NOTE]
> 既存の統合テスト `TestScaffoldDefault` は `runDevctlInDir(t, tmpDir, ...)` でCWD を tmpDir に設定して実行している。
> Gitルート自動検出の導入後、tmpDir が Git リポジトリとして初期化されているため（テスト内で `git init` → `git commit` を実施）、`git rev-parse --show-toplevel` は tmpDir を返す。
> したがって、既存テストは **変更なし** で引き続きパスする見込み。
