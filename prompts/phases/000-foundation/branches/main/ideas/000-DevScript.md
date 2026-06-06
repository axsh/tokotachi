# devctl 実装仕様書

## 背景 (Background)

### 課題

feature 単位の開発において、以下の運用上の課題が存在する。

- **環境構築の手順がバラバラ**: 開発者ごとに Docker コンテナの起動方法・エディタの設定方法が異なり、再現性が低い
- **エディタ間の差異**: VSCode / Cursor / Antigravity / Claude Code でコンテナ接続の方法が異なる
- **OS 間の差異**: Linux / macOS / Windows でツールの挙動・制約が異なる
- **ライフサイクル管理の欠如**: エディタを閉じるとコンテナも停止する、再接続の手順が煩雑などの問題がある
- **SSH 利用の非統一**: SSH 経由でのコンテナ接続が必要な場合の手順が標準化されていない

### 目指す姿

`devctl` コマンドを**チーム共通の開発環境操作の入口**とし、OS・IDE・コンテナモードの差異を**マトリクス駆動の分岐制御**により吸収する。

本ツールは Go 言語で実装する**開発環境オーケストレータ**であり、`features/devctl/` 配下に配置する。

---

## 要件 (Requirements)

### 必須要件

#### R1: feature 単位の開発環境管理

- feature 論理名を指定して、対応する worktree を開発対象として操作できること
- worktree は `work/<feature>` に存在することを前提とする

#### R2: コンテナライフサイクル管理

- `--up` でコンテナを起動できること
- `--down` でコンテナを停止・削除できること
- `--status` で対象 feature の状態を確認できること
- エディタの起動/終了とコンテナのライフサイクルは独立していること

#### R3: エディタ / エージェント起動

- `--open` でエディタ / エージェントを起動できること
- `--editor code|cursor|ag|claude` で対象を選択できること
- 省略時は以下の優先順位でデフォルト値を解決すること：
  1. CLI フラグ `--editor`
  2. 環境変数 `DEVCTL_EDITOR`
  3. feature 別設定（`feature.yaml` の `editor_default`）
  4. グローバル設定（`.devrc.yaml` の `default_editor`）
  5. ハードコードデフォルト `cursor`

#### R4: エディタ別の接続方式

- **VSCode / Cursor**: Dev Container 接続を試行し、失敗時はローカル worktree 起動にフォールバック
- **Antigravity**: Dev Container 接続は試行せず、ローカル worktree を直接開く
- **Claude Code**: CLI/agent として扱い、対象 worktree を current directory にして起動する

#### R5: コンテナ内操作

- `--shell` でコンテナに対話的シェル接続できること
- `--exec <command...>` でコンテナ内コマンドを実行できること

#### R6: SSH モード

- `--ssh` でコンテナを SSH 接続可能な構成で起動する要求を出せること
- スクリプトは SSH モード要求をコンテナ側へ伝える責務のみ持つ

#### R7: マトリクス駆動の分岐制御

- OS・editor・container mode の組み合わせに応じて、利用可能な機能とフォールバック挙動を決定すること
- 各組み合わせに対して互換レベル（L1〜L4）を定義し、明示的な decision log を出力できること

### 任意要件

- `--new-window` でエディタを新規ウィンドウで起動
- `--no-build` で build をスキップした起動
- `--rebuild` で既存イメージの再 build
- `--detach` でバックグラウンド起動（デフォルト動作）
- `--verbose` で詳細ログ出力
- `--dry-run` で実行予定処理のみ表示
- `--force` で確認なし停止・削除

---

## 実現方針 (Implementation Approach)

### コマンド体系

```bash
devctl <feature> [options]
```

第1引数に feature 論理名を取り、オプションで操作を指定する。

呼び出しは以下のいずれかで行う：

```bash
# go run 経由（開発時）
go run ./features/devctl feature-a --up

# ビルド済みバイナリ経由
./bin/devctl feature-a --up
```

### 想定ディレクトリ構成

```text
repo/
  features/
    devctl/              # ← devctl Go 実装本体
      main.go
      cmd/
      internal/
      go.mod
      go.sum
      .devcontainer/     # devctl 開発用コンテナ定義
        devcontainer.json
      Dockerfile         # devctl ビルド・開発用 Dockerfile
    feature-a/
      .devcontainer/
      Dockerfile
  work/
    feature-a/
    feature-b/
  scripts/
    process/
      build.sh           # ビルド・ユニットテスト
      integration_test.sh
  environments/
  shared/
  catalog/
  bin/                   # ビルド済みバイナリ出力先（.gitignore 対象）
    devctl
```

> [!IMPORTANT]
> `bin/` ディレクトリは `.gitignore` に記載し、ビルド成果物をバージョン管理対象外とする。

### 設計方針

#### ライフサイクル分離

- エディタを閉じてもコンテナは残る
- コンテナが起動したまま、エディタだけ再接続できる
- `--down` はコンテナのみ停止する（エディタ終了は行わない）

#### environment-first

- Docker コンテナ環境が正本
- エディタはその環境に接続または並走するクライアントとして扱う

#### マトリクス駆動設計

`dev` ツールは、if 文の寄せ集めではなく、**OS × Editor × Container mode × Action の組み合わせマトリクスに基づいて挙動を決定する構造**を持つ。

---

### 実行環境マトリクス

#### 制御軸

| 軸 | 値 |
|---|---|
| OS | `linux`, `macos`, `windows` |
| Editor | `code`, `cursor`, `ag`, `claude` |
| Container | `none`, `devcontainer`, `docker-local`, `docker-ssh` |
| Action | `up`, `open`, `up_open`, `down`, `shell`, `exec`, `status` |

#### OS × Editor の基本互換マトリクス

| OS | Editor | ローカルフォルダ起動 | Dev Container attach 試行 | SSH 接続補助 | 備考 |
|---|---|---:|---:|---:|---|
| Linux | VSCode | supported | best_effort | supported | 標準対象 |
| Linux | Cursor | supported | best_effort | supported | 標準対象 |
| Linux | Antigravity | supported | unsupported | best_effort | local only 基本 |
| Linux | Claude Code | supported | unsupported | supported | CLI として扱う |
| macOS | VSCode | supported | best_effort | supported | 標準対象 |
| macOS | Cursor | supported | best_effort | supported | 標準対象 |
| macOS | Antigravity | supported | unsupported | best_effort | local only 基本 |
| macOS | Claude Code | supported | unsupported | supported | CLI として扱う |
| Windows | VSCode | supported | best_effort | best_effort | Docker/WSL 依存あり |
| Windows | Cursor | supported | best_effort | best_effort | Docker/WSL 依存あり |
| Windows | Antigravity | supported | unsupported | best_effort | local only 基本 |
| Windows | Claude Code | supported | supported | best_effort | shell 環境差異あり |

> [!IMPORTANT]
> この表は参考表ではなく、**実装判断の基準表**として扱う。

#### コンテナモード別の意味

| モード | 説明 |
|---|---|
| `none` | コンテナを使わない。editor/agent はローカル worktree を開く |
| `docker-local` | Docker コンテナを起動し worktree を bind mount。editor はローカル worktree を開く。コンテナ実行は `shell` / `exec` / SSH で行う |
| `docker-ssh` | `docker-local` に加え SSH 接続可能な状態を要求。実際の SSH 成立はコンテナ実装に依存 |
| `devcontainer` | `.devcontainer` を解釈し Dev Container 統合を試みる。VSCode / Cursor でのみ meaningful。失敗時は `docker-local` またはローカル open にフォールバック |

#### Editor ごとの役割定義

| Editor | 基本モード | devcontainer | SSH | 備考 |
|---|---|---|---|---|
| VSCode / Cursor | devcontainer 優先利用 | best effort attach | 補助 | 失敗時はローカル open にフォールバック |
| Antigravity | docker-local + local open | 非対応 | 手動利用前提 | `--open` は常にローカル worktree 起動 |
| Claude Code | CLI/agent として起動 | 非対応 | `--exec` / `--shell` 併用 | 対象 worktree を cwd にして起動 |

---

### 互換レベル仕様

各組み合わせは以下の互換レベルで表現する。

| レベル | 名称 | 動作 |
|---|---|---|
| `L1` | 正式サポート | 通常実行 |
| `L2` | best effort | 試行し、失敗時は fallback |
| `L3` | フォールバックのみ | 直接 fallback |
| `L4` | 非対応 | エラーまたは警告付き no-op |

#### 互換レベルの例

| 組み合わせ | レベル |
|---|---|
| macOS + Cursor + devcontainer + open | `L2` |
| macOS + AG + devcontainer + open | `L4` |
| macOS + AG + docker-local + open | `L1` |
| Linux + Claude Code + docker-local + open | `L1` |

---

### 実装アーキテクチャ

処理は以下の段階で分離する。分岐ロジックを各所に散らしてはならない。

```text
1. detect        → 実行環境の検出（OS, editor, container mode）
2. resolve       → マトリクスに照らして capability を決定
3. plan          → 実行計画を構築
4. execute       → 実行
5. fallback      → 必要なら代替動作
```

#### Capability オブジェクト（内部表現）

最低限以下の capability を持つことを推奨する。

- `can_open_local`
- `can_try_devcontainer_attach`
- `can_use_ssh_mode`
- `can_launch_new_window`
- `can_run_claude_locally`
- `can_run_claude_in_container`
- `requires_best_effort`

---

### コンテナ識別

| 項目 | 形式 | 例 |
|---|---|---|
| コンテナ名 | `<project-name>-<feature>` | `myproj-feature-a` |
| イメージ名 | `<project-name>-dev-<feature>` | `myproj-dev-feature-a` |

### build/run 設定の解決優先順位

1. `work/<feature>/.devcontainer/devcontainer.json`
2. `work/<feature>/.devcontainer/Dockerfile`
3. `work/<feature>/Dockerfile`
4. リポジトリ標準の共通設定

#### 初期版で解釈する devcontainer.json フィールド

- `build` or `image`
- `workspaceFolder`
- `containerEnv`

> [!NOTE]
> 初期版では `.devcontainer` の完全互換は求めず、実運用に必要な最小サブセットのみ扱う。

### 内部状態モデル

| 状態 | 説明 |
|---|---|
| `NOT_FOUND` | worktree が存在しない |
| `WORKTREE_ONLY` | worktree は存在するがコンテナ未起動 |
| `CONTAINER_RUNNING` | コンテナが稼働中 |
| `CONTAINER_STOPPED` | コンテナが停止中 |
| `EDITOR_OPEN_UNKNOWN` | 参考情報（厳密管理しない） |

### SSH モード

- `--ssh` は「SSH 接続可能な開発環境として起動したい」という要求
- 実装はコンテナ定義に依存（例: `ENABLE_SSH=1` 環境変数、ポート公開、authorized_keys マウントなど）
- スクリプトは要求伝達・ポート/鍵オプション組み立て・接続先情報表示のみ保証

### エラー処理

| 分類 | 説明 | 例 |
|---|---|---|
| Fatal | 処理続行不能 | repo root 取得不可、worktree 不在、Docker 不在、build 失敗 |
| Warning | 一部機能のみ失敗 | container attach 失敗、editor オプション非対応、SSH 設定不足 |

#### 終了コード

| コード | 意味 |
|---|---|
| `0` | 成功（初期版では warning 含む） |
| `1` | fatal error |
| `2` | usage error |

### ログ仕様

- レベル: `INFO`, `WARN`, `ERROR`, `DEBUG`
- `--verbose` 時のみ `DEBUG` を表示

### 設定ファイル

#### グローバル設定 (`repo/.devrc.yaml`)

```yaml
default_editor: cursor
project_name: myproj
work_dir: work
default_container_mode: docker-local

compatibility:
  linux:
    code:
      devcontainer_open: best_effort
      local_open: supported
    cursor:
      devcontainer_open: best_effort
      local_open: supported
    ag:
      local_open: supported
      devcontainer_open: unsupported
    claude:
      local_open: supported
      container_exec: supported
  macos:
    code:
      devcontainer_open: best_effort
      local_open: supported
    cursor:
      devcontainer_open: best_effort
      local_open: supported
    ag:
      local_open: supported
      devcontainer_open: unsupported
    claude:
      local_open: supported
      container_exec: supported
  windows:
    code:
      devcontainer_open: best_effort
      local_open: supported
    cursor:
      devcontainer_open: best_effort
      local_open: supported
    ag:
      local_open: supported
      devcontainer_open: unsupported
    claude:
      local_open: supported
      container_exec: best_effort
```

マトリクスを設定ファイルとして外出しできる構造を推奨する。

#### feature 別設定 (`work/<feature>/feature.yaml` or `features/<feature>/feature.yaml`)

```yaml
dev:
  editor_default: ag
  ssh_supported: true
  shell: bash
```

### 実装方式

**Go 言語**で実装し、`features/devctl/` 配下に配置する。

#### Go パッケージ構成

```text
features/devctl/
  main.go                    # エントリポイント
  go.mod
  go.sum
  .devcontainer/             # devctl 開発用コンテナ定義
    devcontainer.json
  Dockerfile                 # devctl ビルド・開発用
  cmd/
    root.go                  # CLI ルートコマンド定義
  internal/
    detect/
      os.go                  # OS 検出
      editor.go              # editor 検出・解釈
    matrix/
      capability.go          # Capability オブジェクト
      compatibility.go       # 互換レベル判定
      matrix.go              # マトリクス定義・ルックアップ
    resolve/
      worktree.go            # worktree 解決
      container.go           # コンテナ名・イメージ名解決
      devcontainer.go        # devcontainer.json 解釈
      config.go              # 設定ファイル読み込み
    plan/
      planner.go             # 実行計画構築
    action/
      up.go                  # コンテナ起動
      down.go                # コンテナ停止
      open.go                # エディタ起動
      shell.go               # シェル接続
      exec.go                # コンテナ内コマンド実行
      status.go              # ステータス表示
    editor/
      vscode.go              # VSCode 起動ロジック
      cursor.go              # Cursor 起動ロジック
      ag.go                  # Antigravity 起動ロジック
      claude.go              # Claude Code 起動ロジック
    log/
      logger.go              # ログ出力
```

#### 開発用コンテナ（`.devcontainer/` + `Dockerfile`）

`features/devctl/` には `.devcontainer/` と `Dockerfile` を配置し、devctl 自体の開発環境差異をなくす。

- **Dockerfile**: Go ツールチェイン、Docker CLI、必要な開発ツール（linter、テストツール等）を含む
- **devcontainer.json**: VSCode / Cursor で開いた際に自動的に開発コンテナ内で作業できるようにする
- devctl 自身が管理するコンテナとは別のものであり、devctl の開発者向けの環境である

#### ビルド・テスト

ビルドとテストは `scripts/process/` 配下のスクリプトで実行する。

```bash
# ビルド・ユニットテスト
scripts/process/build.sh

# 統合テスト
scripts/process/integration_test.sh
```

`build.sh` は devctl のビルドを含み、成果物を `bin/devctl` に出力する。

> [!IMPORTANT]
> `bin/` ディレクトリは `.gitignore` に記載し、バージョン管理対象外とする。

#### Go を選択する理由

- OS 別分岐（`runtime.GOOS`）が明確
- editor 別分岐を型安全に扱える
- YAML/JSON の読み込みが標準的
- マトリクス・Capability の構造化が自然
- 実行計画生成・fallback 制御のテストが容易
- クロスプラットフォームビルドが容易

---

## 検証シナリオ (Verification Scenarios)

### シナリオ1: コンテナ起動のみ

```bash
devctl feature-a --up
```

1. worktree `work/feature-a` を解決する
2. コンテナ設定を解決する
3. 必要に応じて Docker イメージを build する
4. コンテナ `myproj-feature-a` を起動する
5. 起動結果を `[INFO]` レベルで表示する

### シナリオ2: エディタのみ起動（Cursor）

```bash
devctl feature-a --open --editor cursor
```

1. worktree `work/feature-a` を解決する
2. editor を `cursor` と判定する
3. マトリクスから互換レベルを判定（例: macOS + cursor + devcontainer → L2）
4. Dev Container attach を試行する
5. 失敗時はローカルフォルダ `work/feature-a` を開く
6. ログに `[WARN] Container-aware open failed` を出力し、正常終了する

### シナリオ3: コンテナ起動 + VSCode 起動

```bash
devctl feature-a --up --open --editor code
```

1. コンテナを起動する
2. VSCode を起動する
3. Dev Container attach を試行する
4. attach 失敗時はローカル起動へフォールバックする

### シナリオ4: Antigravity でローカル起動

```bash
devctl feature-a --open --editor ag
```

1. worktree `work/feature-a` を解決する
2. マトリクスから判定（AG + devcontainer → L4: 非対応）
3. AG を起動し、対象 worktree を開く
4. Dev Container attach は**試行しない**

### シナリオ5: Claude Code で起動

```bash
devctl feature-a --open --editor claude
```

1. worktree `work/feature-a` を解決する
2. 対象 worktree を current directory にして Claude Code を起動する
3. Dev Container attach は行わない

### シナリオ6: コンテナ停止

```bash
devctl feature-a --down
```

1. 対象コンテナ `myproj-feature-a` を `docker stop` する
2. 対象コンテナを `docker rm` する
3. 結果を表示する
4. エディタプロセスの終了は**行わない**

### シナリオ7: シェル接続

```bash
devctl feature-a --shell
```

1. 対象コンテナを特定する
2. `docker exec -it <container> bash`（または `sh`）を実行する

### シナリオ8: コンテナ内コマンド実行

```bash
devctl feature-a --exec go test ./...
```

1. 対象コンテナを特定する
2. `docker exec` でコマンドを実行する
3. 結果を返却する

### シナリオ9: SSH モード起動

```bash
devctl feature-a --up --ssh
```

1. SSH モード要求をコンテナ側へ伝達する
2. SSH 用ポートや鍵ファイル指定のオプションを組み立てる
3. コンテナを起動する
4. 接続先情報を表示する

### シナリオ10: エディタ再接続

```bash
devctl feature-a --open --editor cursor
```

1. コンテナが既に起動している状態で実行する
2. Cursor を起動する
3. Dev Container attach を試行する（コンテナ起動中のため成功する可能性が高い）
4. 失敗時はローカル起動へフォールバックする

### シナリオ11: マトリクス decision log 確認

```bash
devctl feature-a --open --editor ag --verbose
```

1. `[DEBUG]` レベルで OS / editor / container mode / 互換レベルの判定結果を表示する
2. 例: `[DEBUG] OS=macos, Editor=ag, ContainerMode=docker-local, Level=L1`

---

## テスト項目 (Testing for the Requirements)

### 自動テスト方針

#### ユニットテスト（関数単位）

| テスト対象 | 検証内容 | 対応要件 |
|---|---|---|
| `resolve_worktree` | 正しいパスを返すこと、存在しない場合にエラーとなること | R1 |
| `resolve_container_name` | `<project>-<feature>` 形式のコンテナ名を返すこと | R2 |
| `resolve_image_name` | `<project>-dev-<feature>` 形式のイメージ名を返すこと | R2 |
| `resolve_devcontainer_config` | 設定ソースの優先順位が正しいこと | R2 |
| `detect_os` | 実行環境の OS を正しく検出すること | R7 |
| `detect_editor` | 指定された editor を正しく解釈すること | R3 |
| `resolve_capabilities` | マトリクスに基づき正しい capability を返すこと | R7 |
| `resolve_container_mode` | コンテナモードが正しく解決されること | R7 |

#### 統合テスト（Docker を使用）

| テスト対象 | 検証内容 | 対応要件 |
|---|---|---|
| `--up` | コンテナが起動すること、`docker ps` で確認 | R2 |
| `--down` | コンテナが停止・削除されること | R2 |
| `--status` | 正しいステータスが表示されること | R2 |
| `--shell` | コンテナ内でシェルコマンドが実行できること | R5 |
| `--exec` | 指定コマンドの結果が返ること | R5 |
| `--ssh` によるコンテナ起動 | SSH モード要求が伝達されること | R6 |
| `--open --editor ag` | AG がローカル worktree を開くこと | R3, R4 |
| `--open --editor claude` | Claude Code が対象 worktree で起動すること | R3, R4 |
| `--open --editor code` フォールバック | attach 失敗時にローカル起動すること | R3, R4 |
| `--verbose` での decision log | マトリクス判定結果が出力されること | R7 |

#### 検証コマンド

```bash
# ビルド・ユニットテスト
scripts/process/build.sh

# 統合テスト（Docker 環境必須）
scripts/process/integration_test.sh
```

> [!IMPORTANT]
> 統合テストの実行には Docker が利用可能な環境が必要です。

---

## スコープ外 (Out of Scope)

- feature テンプレート生成
- worktree 自動生成
- Git ブランチ作成
- コンテナイメージ内部の設計
- SSHD の具体的な実装方式
- エディタ内部拡張の開発
- Dev Container attach 機能の完全自動保証

---

## 将来拡張

- `--create-worktree`
- `--logs`
- `--restart`
- `--attach-only`
- `--ssh-open`（SSH 接続を直接開く）
- `--ssh-port <port>`
- `--ssh-key <path>`
- feature ごとの editor 推奨値
- 複数コンテナ feature 対応
- Docker Compose 対応
- status の JSON 出力

---

## 成功条件

1. `devctl feature-a --up` で feature 用コンテナが起動できる
2. `devctl feature-a --open --editor ag` で AG が対象 worktree を開ける
3. `devctl feature-a --open --editor code|cursor` で少なくともローカル起動できる
4. code/cursor で attach 試行して失敗してもフォールバックできる
5. `devctl feature-a --open --editor claude` で対象 worktree を cwd にして Claude Code を起動できる
6. `devctl feature-a --down` で対象コンテナを確実に停止・削除できる
7. `--shell` / `--exec` でコンテナ内実行ができる
8. OS / editor / mode の組み合わせに応じて、明示的な decision log を出せる
