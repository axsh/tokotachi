# DevContainer JSON ネイティブ解釈

## 背景 (Background)

devctl は現在、開発コンテナの起動に `docker build` と `docker run` を直接実行しているが、`devcontainer.json` の内容を活用していない。`devcontainer.json` はコンテナ構成を宣言的に記述する標準フォーマットであり、VSCode/Cursor の Dev Container 拡張機能や `devcontainer` CLI で利用される。

しかし、`devcontainer` CLI は Node.js に依存しており、Node.js のインストールは遅く、環境を複雑にする。devctl が `devcontainer.json` を自前で解釈し、必要な情報を `docker build` / `docker run` の引数に変換することで、Node.js 依存なしで DevContainer 互換の動作を実現する。

### 現状の問題

1. `action.buildImage()` は `devcontainer.json` を読まず、ハードコーディングされたパスで Dockerfile を探している
2. `docker run` のマウント設定、環境変数、workspaceFolder は `devcontainer.json` に記述されているが無視されている
3. 既存の `resolve.DevcontainerConfig` 構造体と `LoadDevcontainerConfig` 関数は実装済みだが、`mounts` や `remoteUser` のフィールドが不足しており、`up` フローで呼び出されていない

---

## 要件 (Requirements)

### 必須要件

#### R1: devcontainer.json の検索と読み込み

devctl は以下の優先順位で `devcontainer.json` を検索する:

1. `features/<feature>/.devcontainer/devcontainer.json`（feature ディレクトリ内）
2. `work/<feature>/<branch>/.devcontainer/devcontainer.json`（worktree 内）

見つかった場合は JSON を解析し、以下のフィールドを処理する。

#### R2: サポートする devcontainer.json フィールド

| フィールド | 型 | docker コマンドへの変換 |
|---|---|---|
| `build.dockerfile` | string | `docker build -f <dockerfile>` |
| `build.context` | string | ビルドコンテキストのパス（デフォルト: devcontainer.json の親ディレクトリ） |
| `image` | string | `docker run <image>`（build より優先度低） |
| `workspaceFolder` | string | `docker run -w <workspaceFolder>` |
| `mounts` | string[] | `docker run --mount <mount>` または `-v` に変換 |
| `containerEnv` | map | `docker run -e KEY=VALUE` |
| `remoteUser` | string | `docker run --user <user>` |
| `name` | string | コンテナ名のサフィックスとして参照（情報のみ） |

#### R3: docker build への反映

- `build.dockerfile`: Dockerfile のパスを指定。`devcontainer.json` からの相対パスとして解決する
- `build.context`: ビルドコンテキスト。未指定時は `devcontainer.json` の親ディレクトリ
- パス解決例:
  - `devcontainer.json` が `features/devctl/.devcontainer/devcontainer.json` にある場合
  - `"dockerfile": "./Dockerfile"` → `features/devctl/.devcontainer/Dockerfile`
  - `"context": ".."` → `features/devctl/`（親ディレクトリ）

#### R4: docker run への反映

- `workspaceFolder`: `-w` フラグ（デフォルト: `/workspace`）
- `mounts`: 各エントリを `--mount` フラグに変換
  - 文字列形式: `"source=...,target=...,type=bind"` → そのまま `--mount` に渡す
- `containerEnv`: 各エントリを `-e KEY=VALUE` に変換
- `remoteUser`: `--user` フラグに変換
- worktree パスのバインドマウント（`-v worktree:/workspace`）は常に自動追加する

#### R5: devcontainer.json がない場合のフォールバック

- `devcontainer.json` が見つからない場合は、従来通り Dockerfile の自動検出で動作する
- 検索順: `.devcontainer/Dockerfile` → ルートの `Dockerfile`
- Dockerfile も見つからない場合は `--no-build` 相当として扱い、イメージ名でコンテナを起動する

#### R6: 既存構造体の拡張

`resolve.DevcontainerConfig` に以下のフィールドを追加する:

```go
type DevcontainerConfig struct {
    Name            string            `json:"name"`
    Image           string            `json:"image"`
    Build           DevcontainerBuild `json:"build"`
    WorkspaceFolder string            `json:"workspaceFolder"`
    ContainerEnv    map[string]string `json:"containerEnv"`
    Mounts          []string          `json:"mounts"`
    RemoteUser      string            `json:"remoteUser"`
}
```

#### R7: LoadDevcontainerConfig の検索パス更新

`features/<feature>/` からも検索できるよう、関数シグネチャを更新する:

```go
// LoadDevcontainerConfig loads devcontainer configuration.
// Search priority:
//  1. features/<feature>/.devcontainer/devcontainer.json
//  2. work/<feature>/<branch>/.devcontainer/devcontainer.json  
//  3. .devcontainer/Dockerfile (fallback)
func LoadDevcontainerConfig(repoRoot, feature, branch string) (DevcontainerConfig, error)
```

### 任意要件

#### R8: customizations の読み取り（将来）

- `customizations.vscode.extensions` は VSCode/Cursor の Dev Container attach 時に利用される
- devctl からは直接利用しないが、レポートや status コマンドで情報表示に使える
- 本仕様では対象外

---

## 実現方針 (Implementation Approach)

### 変更対象ファイル

1. **`internal/resolve/devcontainer.go`**: `DevcontainerConfig` にフィールド追加、`LoadDevcontainerConfig` の検索パス拡張
2. **`internal/action/up.go`**: `buildImage()` と `Up()` を `DevcontainerConfig` から動的に引数生成するよう修正
3. **`cmd/up.go`**: `LoadDevcontainerConfig` を呼び出し、結果を `UpOptions` に渡す

### 処理フロー

```
cmd/up.go
  ├─ resolve.LoadDevcontainerConfig(repoRoot, feature, branch)
  │   ├─ features/<feature>/.devcontainer/devcontainer.json を検索
  │   └─ デフォルト値をフォールバック
  ├─ DevcontainerConfig から UpOptions を構築
  │   ├─ Dockerfile パス解決
  │   ├─ ビルドコンテキスト解決  
  │   ├─ mounts → docker run 引数
  │   ├─ containerEnv → docker run 引数
  │   └─ remoteUser → docker run 引数
  └─ action.Runner.Up(opts)
```

---

## 検証シナリオ (Verification Scenarios)

1. 現在の `features/devctl/.devcontainer/devcontainer.json` を使って `devctl up devctl test-001 --dry-run --verbose` を実行し、以下が表示されること:
   - `[DRY-RUN] docker build -f .devcontainer/Dockerfile ...`（Dockerfile パスが正しく解決）
   - `[DRY-RUN] docker run ... --mount source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind ...`（mounts が反映）
   - `[DRY-RUN] docker run ... -w /workspace ...`（workspaceFolder が反映）
   - `[DRY-RUN] docker run ... --user root ...`（remoteUser が反映）
2. `devcontainer.json` がない feature に対して `devctl up` を実行した場合、従来の Dockerfile 自動検出にフォールバックすること
3. `--report report.md` 付きで実行し、レポートに DevcontainerConfig の情報が含まれること

---

## テスト項目 (Testing for the Requirements)

| 要件 | 検証方法 |
|---|---|
| R1-R7 | `./scripts/process/build.sh` でビルド・単体テスト全 PASS |
| R2 (フィールド解析) | `internal/resolve/devcontainer_test.go` に JSON パースのテストケース追加 |
| R3 (docker build) | `internal/action/up_test.go` でビルド引数の生成テスト |
| R4 (docker run) | `internal/action/up_test.go` で run 引数の生成テスト |
| R5 (フォールバック) | `internal/resolve/devcontainer_test.go` で devcontainer.json なしのテスト |
| R6 (構造体) | 既存テストの更新 |
| シナリオ 1-3 | `devctl up --dry-run --verbose` の手動実行確認 |
