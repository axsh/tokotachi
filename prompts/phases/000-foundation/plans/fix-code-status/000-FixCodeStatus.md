# 000-FixCodeStatus

> **Source Specification**: [000-FixCodeStatus.md](file://prompts/phases/000-foundation/ideas/fix-code-status/000-FixCodeStatus.md)

## Goal Description

`tt list` の CODE カラムが常に `(unknown)` と表示される問題を修正する。ファイルロック共通ライブラリの作成、CodeStatus の初期化、`tt pr` でのステータス更新、バックグラウンドプロセスの動作検証を行う。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `tt up`/`tt open` で CodeStatus を初期化 | Proposed Changes > cmd/up.go, cmd/open.go |
| R2: プロセス間安全なファイルロック機構 | Proposed Changes > internal/filelock/ |
| R3: バックグラウンドプロセスの動作検証 | Proposed Changes > bgrunner_test.go, bgrunner.go |
| R4: `tt pr` で CodeStatus を `pr` に更新 | Proposed Changes > cmd/pr.go |
| R5: エラーログはテスト専用 | Proposed Changes > bgrunner.go (テスト用ログ注入) |

## Proposed Changes

### filelock パッケージ (新規)

#### [NEW] [filelock_test.go](file://features/tt/internal/filelock/filelock_test.go)
*   **Description**: ファイルロックの単体テスト（TDD: テストを先に作成）
*   **Technical Design**:
    ```go
    func TestLock_TryLock_Success(t *testing.T)
    func TestLock_TryLock_AlreadyLocked(t *testing.T)
    func TestLock_Unlock(t *testing.T)
    func TestLock_Unlock_ThenRelock(t *testing.T)
    func TestLock_StaleLock_Timeout(t *testing.T)
    func TestLock_StaleLock_InvalidMeta(t *testing.T)
    func TestLock_ConcurrentAccess(t *testing.T)
    ```
*   **Logic**:
    *   `TryLock_Success`: 新しいパスでロック取得 → `true, nil` を返す。ロックディレクトリが存在すること。
    *   `TryLock_AlreadyLocked`: 同じパスで2回ロック → 2回目は `false, nil`。
    *   `Unlock`: ロック取得後に `Unlock()` → ディレクトリが削除されていること。
    *   `Unlock_ThenRelock`: `Unlock` 後に再度 `TryLock` → `true, nil`。
    *   `StaleLock_Timeout`: 古いタイムスタンプのメタデータで作成されたロックが `ForceUnlockIfStale()` でクリーンアップされること。
    *   `StaleLock_InvalidMeta`: メタデータファイルが壊れている/存在しない場合でも安全に動作すること。
    *   `ConcurrentAccess`: goroutine を使った並行アクセステスト。複数のgoroutineから同時にロック取得を試み、1つだけが成功すること。

#### [NEW] [filelock.go](file://features/tt/internal/filelock/filelock.go)
*   **Description**: `os.Mkdir` ベースのクロスプラットフォームファイルロック
*   **Technical Design**:
    ```go
    package filelock

    // Lock はディレクトリ作成ベースのファイルロック。
    type Lock struct {
        dir string // ロック用ディレクトリのパス
    }

    // Meta はロックのメタデータ。ロックディレクトリ内のJSONファイルに保存。
    type Meta struct {
        PID       int       `json:"pid"`
        CreatedAt time.Time `json:"created_at"`
    }

    // New は新しい Lock を生成する。dir はロックディレクトリの絶対パス。
    func New(dir string) *Lock

    // TryLock はロックの取得を試みる。
    // 取得成功: true, nil — ロックディレクトリ + メタデータファイルを作成。
    // 既にロック中: false, nil
    // エラー: false, err
    func (l *Lock) TryLock() (bool, error)

    // Unlock はロックを解放する（ディレクトリとメタデータを削除）。
    func (l *Lock) Unlock() error

    // ForceUnlockIfStale は古いロックを強制解除する。
    // メタデータの CreatedAt + timeout を超過しているか、
    // PID のプロセスが存在しない場合にロックを解除する。
    func (l *Lock) ForceUnlockIfStale(timeout time.Duration) (bool, error)

    // IsLocked はロックが保持されているかを返す。
    func (l *Lock) IsLocked() bool
    ```
*   **Logic**:
    *   `TryLock()`: `os.Mkdir(l.dir, 0o755)` を呼ぶ。`os.IsExist(err)` なら `false, nil`。成功したら `Meta{PID: os.Getpid(), CreatedAt: time.Now()}` を `l.dir/meta.json` に書き込む。
    *   `Unlock()`: `os.RemoveAll(l.dir)` でディレクトリごと削除。
    *   `ForceUnlockIfStale()`:
        1. `meta.json` を読む。読めなければ強制 `Unlock()` して `true` を返す。
        2. `Meta.CreatedAt + timeout < now` か、`Meta.PID` のプロセスが存在しなければ `Unlock()` して `true`。
        3. それ以外は `false` (まだ有効)。
    *   PID のプロセス存在チェックは既存の `isProcessAlive()` と同じロジック（Unix: signal 0, Windows: tasklist）を使用。ただしここでは `filelock` パッケージ内にプラットフォーム固有コードを配置する。

#### [NEW] [process_unix.go](file://features/tt/internal/filelock/process_unix.go)
*   **Description**: Unix 向け `isProcessAlive` 実装
*   **Technical Design**:
    ```go
    //go:build !windows
    package filelock
    // isProcessAlive は signal 0 でプロセスの存在を確認する。
    func isProcessAlive(pid int) bool
    ```

#### [NEW] [process_windows.go](file://features/tt/internal/filelock/process_windows.go)
*   **Description**: Windows 向け `isProcessAlive` 実装
*   **Technical Design**:
    ```go
    //go:build windows
    package filelock
    // isProcessAlive は tasklist で PID の存在を確認する。
    func isProcessAlive(pid int) bool
    ```

---

### codestatus パッケージ (変更)

#### [MODIFY] [bgrunner_test.go](file://features/tt/internal/codestatus/bgrunner_test.go)
*   **Description**: バックグラウンドプロセスのプリミティブ動作検証テストを追加
*   **Technical Design**:
    ```go
    // 新規追加テスト
    func TestStartBackground_ProcessSpawned(t *testing.T)
    func TestStartBackground_AlreadyRunning(t *testing.T)
    func TestStartBackground_LockAcquiredAndReleased(t *testing.T)
    ```
*   **Logic**:
    *   `TestStartBackground_ProcessSpawned`: テスト用のダミーバイナリ（簡単なGoプログラム）を `go build` で一時的に作成。`StartBackground()` を呼び、ロックファイルが存在すること、プロセスが起動していることを確認。テスト終了時にクリーンアップ。
    *   `TestStartBackground_AlreadyRunning`: ロックを先に取得してから `StartBackground()` → `nil` (既に実行中のため何もしない) を確認。
    *   `TestStartBackground_LockAcquiredAndReleased`: `_update-code-status` に相当するテストバイナリを起動し、終了後にロックが解放されていることを確認。テスト用にログをキャプチャする仕組みを使用。

#### [MODIFY] [bgrunner.go](file://features/tt/internal/codestatus/bgrunner.go)
*   **Description**: ロック機構を `filelock` パッケージに置き換え。テスト用ログ注入のサポート追加。
*   **Technical Design**:
    *   既存の `AcquireLock()`, `ReleaseLock()`, `IsRunning()` を `filelock.Lock` ベースに変更。
    *   `StartBackground()` のシグネチャにテスト用オプションを追加：
    ```go
    // StartBackgroundOptions はバックグラウンドプロセスの起動オプション。
    type StartBackgroundOptions struct {
        LogFile string // テスト用: 空文字なら /dev/null (本番)
    }

    func StartBackground(repoRoot, ttBinary string, opts *StartBackgroundOptions) error
    ```
    *   `opts` が `nil` または `LogFile` が空の場合、`cmd.Stdout`/`cmd.Stderr` は `nil` (サイレント)。
    *   `opts.LogFile` が指定された場合、ファイルハンドルを開いて `cmd.Stdout`/`cmd.Stderr` に設定。
*   **Logic**:
    *   `IsRunning()`: `filelock.New(lockPath(repoRoot)).IsLocked()` を呼ぶ。stale ロックの場合 `ForceUnlockIfStale(ProcessTimeout)` で自動解除。
    *   `AcquireLock()`: `filelock.New(lockPath(repoRoot)).TryLock()` に委譲。
    *   `ReleaseLock()`: `filelock.New(lockPath(repoRoot)).Unlock()` に委譲。
    *   ロックのパスはディレクトリに変更: `work/.codestatus.lock` → `work/.codestatus.lock/`（ディレクトリ）。

#### [MODIFY] [process_windows.go](file://features/tt/internal/codestatus/process_windows.go)
*   **Description**: `CREATE_NO_WINDOW` フラグの追加
*   **Technical Design**:
    ```go
    func detachSysProcAttr() *syscall.SysProcAttr {
        return &syscall.SysProcAttr{
            CreationFlags: 0x00000200 | 0x08000000, // CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW
        }
    }
    ```

---

### cmd パッケージ (変更)

#### [MODIFY] [up.go](file://features/tt/cmd/up.go)
*   **Description**: state ファイル保存時に `CodeStatus` を初期化する
*   **Technical Design**: L186 (`state.Save`) の直前に CodeStatus 初期化を追加:
    ```go
    // CodeStatus の初期化 (L185 付近に挿入)
    if sf.CodeStatus == nil {
        sf.CodeStatus = &state.CodeStatus{
            Status: state.CodeStatusLocal,
        }
    }
    ```

#### [MODIFY] [open.go](file://features/tt/cmd/open.go)
*   **Description**: state ファイル保存時に `CodeStatus` を初期化する
*   **Technical Design**: L154 (`state.Save`) の直前に CodeStatus 初期化を追加:
    ```go
    // CodeStatus の初期化 (L153 付近に挿入)
    if sf.CodeStatus == nil {
        sf.CodeStatus = &state.CodeStatus{
            Status: state.CodeStatusLocal,
        }
    }
    ```

#### [MODIFY] [pr.go](file://features/tt/cmd/pr.go)
*   **Description**: PR 作成成功後に `CodeStatus` を `pr` に更新する
*   **Technical Design**: `ctx.ActionRunner.PR()` 成功後（L36 の前）に追加:
    ```go
    // PR 作成成功後、CodeStatus を更新
    statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
    sf, loadErr := state.Load(statePath)
    if loadErr == nil {
        now := time.Now()
        sf.CodeStatus = &state.CodeStatus{
            Status:        state.CodeStatusPR,
            PRCreatedAt:   &now,
            LastCheckedAt: &now,
        }
        if saveErr := state.Save(statePath, sf); saveErr != nil {
            // ログのみ、PR作成自体は成功しているため致命的エラーにしない
            fmt.Fprintf(os.Stderr, "Warning: failed to update code status: %v\n", saveErr)
        }
    }
    ```
    *   `import` に `"time"`, `"os"`, `state` パッケージを追加。

#### [MODIFY] [list.go](file://features/tt/cmd/list.go)
*   **Description**: `StartBackground()` の呼び出しを新シグネチャに更新
*   **Technical Design**: L110 付近:
    ```go
    // opts は nil → ログなし(本番モード)
    if bgErr := codestatus.StartBackground(repoRoot, exe, nil); bgErr != nil {
    ```

---

## Step-by-Step Implementation Guide

### Part 1: ファイルロック (TDD)

- [x] **Step 1**: `features/tt/internal/filelock/filelock_test.go` を作成。全テストケースを記述（この時点では全テスト FAIL）。
- [x] **Step 2**: `features/tt/internal/filelock/filelock.go` を作成。`Lock` 構造体、`New()`, `TryLock()`, `Unlock()`, `IsLocked()`, `ForceUnlockIfStale()` を実装。
- [x] **Step 3**: `features/tt/internal/filelock/process_unix.go` と `process_windows.go` を作成。`isProcessAlive()` を実装。
- [x] **Step 4**: `./scripts/process/build.sh` を実行。filelock の全テストが PASS することを確認。

### Part 2: bgrunner のロック置き換え と動作検証 (TDD)

- [x] **Step 5**: `bgrunner_test.go` にプリミティブ動作検証テスト (`TestStartBackground_*`) を追加。
- [x] **Step 6**: `bgrunner.go` を修正。
  - ロック機構を `filelock` パッケージに移行。
  - `StartBackground()` のシグネチャに `StartBackgroundOptions` を追加。
  - テスト用ログ注入をサポート。
- [x] **Step 7**: `process_windows.go` に `CREATE_NO_WINDOW` フラグを追加。
- [x] **Step 8**: `./scripts/process/build.sh` を実行。codestatus パッケージの全テストが PASS することを確認。

### Part 3: CodeStatus 初期化と PR 更新

- [x] **Step 9**: `up.go` に CodeStatus 初期化ロジックを追加。
- [x] **Step 10**: `open.go` に CodeStatus 初期化ロジックを追加。
- [x] **Step 11**: `pr.go` に PR 作成後の CodeStatus 更新ロジックを追加。
- [x] **Step 12**: `list.go` の `StartBackground()` 呼び出しを新シグネチャに更新。
- [x] **Step 13**: `./scripts/process/build.sh` を実行。全テストが PASS することを確認。

### Part 4: 統合テストと最終検証

- [x] **Step 14**: `./scripts/process/integration_test.sh --categories "integration-test"` を実行。既存の統合テストが全 PASS することを確認。

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認内容**:
        *   `filelock` パッケージの全テストが PASS
        *   `codestatus` パッケージの全テスト（既存 + 新規）が PASS
        *   `listing` パッケージの既存テストが PASS（リグレッションなし）
        *   `state` パッケージの既存テストが PASS（リグレッションなし）

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```
    *   **確認内容**:
        *   `TestTtListCode_ColumnHeaders` — CODE ヘッダーが表示されること
        *   `TestTtListCode_UpdateFlagAccepted` — `--update` フラグが受け入れられること
        *   その他既存テストがリグレッションなく PASS

## Documentation

本計画で影響を受ける既存ドキュメントはなし。仕様書 `000-FixCodeStatus.md` が本変更を記録する唯一のドキュメント。
