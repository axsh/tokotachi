# CODE ステータスの修正 — 初期化・ファイルロック・バックグラウンド更新

## 背景 (Background)

`tt list` の出力には **CODE** カラムがあり、各ブランチのコードホスティング状態を表示する設計になっている。

しかし現状、以下の問題が発生している：

1. **CODE カラムが常に `(unknown)` と表示される** — state ファイル作成時（`tt up` / `tt open`）に `CodeStatus` が初期化されず `nil` のまま。
2. **バックグラウンド更新プロセスが機能していない** — `tt list` 実行時に起動されるはずのバックグラウンドプロセスが正常に動作しない。
3. **`tt pr` 実行時にステータスが更新されない** — PR作成後も `CodeStatus` が変更されない。
4. **現在のロック機構がプロセス間で安全でない** — PIDファイルベースのロックはレースコンディションに脆弱。

---

## 現状のステータス一覧 (Current Status Types)

`state.CodeStatusType` (`features/tt/internal/state/state.go`) に4種類が定義されている：

| ステータス | 値 | `tt list` 表示 | 意味 |
|---|---|---|---|
| `CodeStatusLocal` | `"local"` | `(local)` | リモートにブランチが存在しない（ローカルのみ） |
| `CodeStatusHosted` | `"hosted"` | `hosted` | リモートにブランチが存在する（PRなし） |
| `CodeStatusPR` | `"pr"` | `PR(Xd ago)` | リモートにブランチがあり、オープンなPRが存在する |
| `CodeStatusDeleted` | `"deleted"` | `deleted` | 以前はリモートにあったが、現在は削除されている |
| *(nil)* | — | `(unknown)` | `CodeStatus` が未設定（**現在のバグ**） |

### ステータス遷移ロジック (`Checker.Resolve()`)

```
┌────────────────────────────────────────────────┐
│ git ls-remote --heads origin <branch>          │
│                                                │
│ ブランチがリモートに存在しない場合:                │
│   前回 hosted/pr → deleted                      │
│   それ以外       → local                        │
│                                                │
│ ブランチがリモートに存在する場合:                  │
│   gh pr list --head <branch> で PR を確認        │
│     PR あり → pr (PRCreatedAt を記録)            │
│     PR なし → hosted                             │
│     gh 失敗 → hosted (フォールバック)             │
└────────────────────────────────────────────────┘
```

### ステータスの更新タイミング

CodeStatus が更新される（されるべき）タイミングは以下の **3つ**：

| # | トリガー | 更新方式 | 説明 |
|---|---|---|---|
| 1 | `tt list` | バックグラウンド (自動) | `NeedsUpdate()` が `true` のとき、`tt _update-code-status` を子プロセスで起動。5分ごとにポーリング。 |
| 2 | `tt list --update` | フォアグラウンド (手動) | 全 non-bare ブランチに対して即座に `Checker.UpdateAll()` を実行。 |
| 3 | `tt pr` | 即時 (イベント駆動) | PR作成成功後、当該ブランチの `CodeStatus` を `pr` に更新。**（現在未実装）** |

> **注意**: `hosted`, `pr`, `deleted` のステータスはリモート状態に依存するため、基本的にバックグラウンドポーリングでしか捕捉できない。唯一の例外が `tt pr` で、PR作成の成功を直後に反映できる。

---

## 要件 (Requirements)

### 必須要件

1. **`tt up` / `tt open` で state ファイル作成時に `CodeStatus` を初期化する**
   - 初期値は `CodeStatusLocal` (ステータス: `"local"`) とする
   - `LastCheckedAt` は `nil` のままにして、次回の `tt list` でバックグラウンド更新をトリガーさせる

2. **プロセス間で安全なファイルロック機構を作成する**
   - `shared/libs/go/filelock` として共通ライブラリを作成する（ただし Go モジュール構成に依存する場合は `features/tt/internal/filelock` に配置）
   - プラットフォームごとに最適な実装を選択する：
     - **最もシンプルな方法**: `/tmp/` 配下に一時ディレクトリを `os.Mkdir` で作成し、その成否でロック判定（MkdirはOSレベルでアトミック）
     - **プラットフォーム固有の方法**: もしより適切な手段があれば採用（例: Unix の `flock`, Windows の `LockFileEx` など）
   - ファイルロック単体のテストを先に実施し、各プラットフォームで正常に動作することを検証する
   - 既存の PID ファイルベースのロック (`bgrunner.go` の `AcquireLock`/`ReleaseLock`/`IsRunning`) をこの新ロック機構に置き換える

3. **バックグラウンドプロセスのプリミティブな動作を検証する**
   - `bgrunner.go` の基本機能（プロセス起動、デタッチ、ロック取得・解放）をテストする
   - バックグラウンドプロセスが期待通りに動作することを確認するテストを作成する
   - テスト時にはログを出力して動作を確認可能にする

4. **`tt pr` 実行時に `CodeStatus` を `pr` に更新する**
   - `pr.go` で PR 作成成功後、state ファイルの `CodeStatus` を `state.CodeStatusPR` に更新する

5. **エラーログはテスト専用とする**
   - バックグラウンドプロセスのログ出力は、テスト実行時にのみ有効にする
   - 本番環境では `cmd.Stdout` / `cmd.Stderr` は `nil`（サイレント）のまま維持する

---

## 実現方針 (Implementation Approach)

### 1. ファイルロック共通ライブラリ

#### 配置場所

Go モジュールが `features/tt` に閉じているため、`features/tt/internal/filelock` に配置する。
将来的に他のフィーチャーでも使う場合は `shared/libs/go/filelock` に移動する。

#### 実装方針

最もシンプルかつクロスプラットフォームな方法として、**ディレクトリ作成によるロック**を採用する：

```go
// Lock は os.Mkdir の原子性を利用したファイルロック。
type Lock struct {
    path string
}

func New(path string) *Lock { ... }

// TryLock はロックの取得を試みる。取得できなければ false を返す。
func (l *Lock) TryLock() (bool, error) {
    err := os.Mkdir(l.path, 0o755)
    if err != nil {
        if os.IsExist(err) {
            return false, nil // 既にロック中
        }
        return false, err
    }
    // ロック取得成功
    return true, nil
}

// Unlock はロックを解放する。
func (l *Lock) Unlock() error {
    return os.Remove(l.path)
}
```

この方式のメリット：
- `os.Mkdir` はどのOSでもアトミック
- 外部ライブラリ不要
- stale ロックの検出・タイムアウトも実装可能

staleロック対策として、ロックディレクトリ内にメタデータファイル（PID、タイムスタンプ）を置き、タイムアウト検出に使用する。

この方式と、プラットフォーム固有のロック（Unix: `flock`, Windows: `LockFileEx`）を比較検証し、最適なものを選択する。

### 2. CodeStatus の初期化

`tt up` (`up.go`) と `tt open` (`open.go`) で state ファイルを保存する直前に：

```go
if sf.CodeStatus == nil {
    sf.CodeStatus = &state.CodeStatus{
        Status: state.CodeStatusLocal,
    }
}
```

### 3. `tt pr` でのステータス更新

`pr.go` の `runPR()` で PR 作成成功後に：

```go
// PR 作成成功後、CodeStatus を更新
statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
sf, err := state.Load(statePath)
if err == nil {
    now := time.Now()
    sf.CodeStatus = &state.CodeStatus{
        Status:        state.CodeStatusPR,
        PRCreatedAt:   &now,
        LastCheckedAt: &now,
    }
    _ = state.Save(statePath, sf)
}
```

### 4. バックグラウンドプロセスの修正

- 既存のPIDロックを新しいファイルロック機構に置き換える
- テスト時のみログをキャプチャできるようにする（テスト用ログファイルの注入）
- Windows の `detachSysProcAttr` に `CREATE_NO_WINDOW` フラグを追加

### 5. 影響範囲

| ファイル | 変更内容 |
|---|---|
| `features/tt/internal/filelock/` [NEW] | ファイルロック共通ライブラリ |
| `features/tt/internal/filelock/filelock_test.go` [NEW] | ファイルロックのテスト |
| `features/tt/cmd/up.go` | CodeStatus の初期化を追加 |
| `features/tt/cmd/open.go` | CodeStatus の初期化を追加 |
| `features/tt/cmd/pr.go` | PR作成後に CodeStatus を `pr` に更新 |
| `features/tt/internal/codestatus/bgrunner.go` | ロック機構の置き換え、テスト用ログ注入 |
| `features/tt/internal/codestatus/bgrunner_test.go` | プリミティブな動作検証テスト追加 |
| `features/tt/internal/codestatus/process_windows.go` | `CREATE_NO_WINDOW` フラグの追加 |

---

## 検証シナリオ (Verification Scenarios)

### シナリオ1: ファイルロックの基本動作確認

1. ファイルロックを取得する。
2. 同じパスで再度ロック取得を試みる → 失敗すること。
3. ロックを解放する。
4. 再度ロック取得を試みる → 成功すること。
5. stale ロック（古いプロセスが残したロック）が正しく検出・解除されること。

### シナリオ2: バックグラウンドプロセスの基本動作確認

1. バックグラウンドプロセスの起動テスト — プロセスが正常に起動されること。
2. ロックの取得と解放がプロセス間で正しく動作すること。
3. プロセス終了後にロックが解放されていること。
4. テスト時にログが出力され、動作を確認できること。

### シナリオ3: CODE ステータスの初期化確認

1. `tt up <branch> <feature>` を実行する。
2. state YAML ファイルを確認し、`code_status.status` が `"local"` であること。
3. `tt list` を実行し、CODE カラムが `(local)` と表示されること。

### シナリオ4: `tt pr` でのステータス更新

1. `tt pr <branch>` を実行して PR を作成する。
2. state YAML ファイルの `code_status.status` が `"pr"` に更新されていること。
3. `tt list` を実行し、CODE カラムが `PR(0m ago)` のように表示されること。

### シナリオ5: フォアグラウンド強制更新

1. `tt list --update` を実行する。
2. CODE カラムにリモートの状態が正しく反映されること。

---

## テスト項目 (Testing for the Requirements)

### 単体テスト (`scripts/process/build.sh`)

| テスト対象 | テスト内容 | ファイル |
|---|---|---|
| `filelock` | ロック取得・解放・重複取得の拒否・stale検出 | `features/tt/internal/filelock/filelock_test.go` [NEW] |
| `bgrunner.go` | プリミティブな動作テスト（起動・ロック・解放） | `features/tt/internal/codestatus/bgrunner_test.go` |
| `bgrunner.go` | `NeedsUpdate()` が初期化済みstateで正しく動作すること | 既存テスト |
| `listing.go` | `FormatCodeColumn()` が `CodeStatusLocal` で `(local)` を返すこと | 既存テスト |
| `state.go` | `CodeStatus` のラウンドトリップ | 既存テスト |

### 統合テスト (`scripts/process/integration_test.sh`)

| テスト対象 | テスト内容 | ファイル |
|---|---|---|
| `tt list` の CODE カラム | state ファイルに `code_status` があるときの表示確認 | `tests/integration-test/tt_list_code_test.go` |
| `tt list --update` | フォアグラウンド更新の実行確認 | 既存テスト |
