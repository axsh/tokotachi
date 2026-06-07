# 000-Fix-ToolID-AutoDetect

> **Source Specification**: `prompts/phases/000-foundation/ideas/fix-bump-new-version/000-Fix-ToolID-AutoDetect.md`

## Goal Description

github-upload パイプラインの2つの問題を修正する:
1. dist スクリプト群に tool-id バリデーション関数を追加し、バージョン文字列や未登録IDが tool-id として誤解釈されることを防ぐ
2. `bgrunner.go` 内の Windows 専用コードをプラットフォーム別ファイルに分離し、全 OS 向けクロスコンパイルを成功させる

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 要件1: tool-id のバージョン形式バリデーション | Proposed Changes > `_lib.sh` の `validate_tool_id` |
| 要件2: `tools.yaml` 存在チェック | Proposed Changes > `_lib.sh` の `validate_tool_id` |
| 要件3: 全 dist スクリプトへの適用 | Proposed Changes > `github-upload.sh`, `build.sh`, `release.sh`, `publish.sh` |
| 要件4: プラットフォーム別ファイル分離 | Proposed Changes > `bgrunner.go`, `process_windows.go`, `process_unix.go` |
| 要件5: Unix 用 `isProcessAlive` 実装 | Proposed Changes > `process_unix.go` |
| 要件6: `detachSysProcAttr` ヘルパー関数 | Proposed Changes > `process_windows.go`, `process_unix.go` |

## Proposed Changes

### dist スクリプト群

#### [MODIFY] [_lib.sh](file://scripts/dist/_lib.sh)
*   **Description**: `validate_tool_id` 関数を追加
*   **Technical Design**:
    ```bash
    # validate_tool_id <arg>
    # 第1引数を検証する。無効なら fail + exit 1。
    validate_tool_id() { ... }
    ```
*   **Logic**:
    1. バージョン形式チェック: `$arg` が `^(\+)?v[0-9]+\.[0-9]+\.[0-9]+$` にマッチする場合
       - `fail "First argument looks like a version, not a tool-id: '${arg}'"` を出力
       - 正しい Usage と `get_all_tool_ids` の結果を案内して `exit 1`
    2. `tools.yaml` 存在チェック: `get_all_tool_ids` の出力に `$arg` が含まれない場合
       - `fail "Unknown tool-id: '${arg}'"` を出力
       - 利用可能な tool-id 一覧を案内して `exit 1`
    3. どちらにも該当しなければ正常 return

#### [MODIFY] [github-upload.sh](file://scripts/dist/github-upload.sh)
*   **Description**: `TOOL_ID` 設定直後に `validate_tool_id` 呼び出しを追加
*   **Logic**:
    - 120行目 `TOOL_ID="$1"` の直後（121行目付近）に `validate_tool_id "$TOOL_ID"` を1行追加

#### [MODIFY] [build.sh](file://scripts/dist/build.sh)
*   **Description**: `TOOL_ID` 設定直後に `validate_tool_id` 呼び出しを追加
*   **Logic**:
    - 16行目 `TOOL_ID="$1"` の直後に `validate_tool_id "$TOOL_ID"` を1行追加

#### [MODIFY] [release.sh](file://scripts/dist/release.sh)
*   **Description**: `TOOL_ID` 設定直後に `validate_tool_id` 呼び出しを追加
*   **Logic**:
    - 16行目 `TOOL_ID="$1"` の直後に `validate_tool_id "$TOOL_ID"` を1行追加

#### [MODIFY] [publish.sh](file://scripts/dist/publish.sh)
*   **Description**: `TOOL_ID` 設定直後に `validate_tool_id` 呼び出しを追加
*   **Logic**:
    - 16行目 `TOOL_ID="$1"` の直後に `validate_tool_id "$TOOL_ID"` を1行追加

---

### codestatus パッケージ（クロスコンパイル対応）

#### [MODIFY] [bgrunner.go](file://features/tt/internal/codestatus/bgrunner.go)
*   **Description**: Windows 専用コードを除去し、プラットフォーム共通のヘルパー関数呼び出しに置き換え
*   **Technical Design**:
    ```go
    // StartBackground 内の変更箇所
    func StartBackground(repoRoot, ttBinary string) error {
        // ...
        cmd.SysProcAttr = detachSysProcAttr()  // ← 変更後
        // ...
    }
    ```
*   **Logic**:
    1. import から `"syscall"` を削除
    2. `StartBackground` 関数内の以下を置き換え:
       - 変更前（118-122行目）:
         ```go
         cmd.SysProcAttr = &syscall.SysProcAttr{
             CreationFlags: 0x00000200,
         }
         ```
       - 変更後:
         ```go
         cmd.SysProcAttr = detachSysProcAttr()
         ```
    3. コメント `// On Windows, CREATE_NEW_PROCESS_GROUP detaches the child.` を `// Detach child process (platform-specific)` に変更

#### [MODIFY] [process_windows.go](file://features/tt/internal/codestatus/process_windows.go)
*   **Description**: `detachSysProcAttr` 関数を追加、`syscall` import を追加
*   **Technical Design**:
    ```go
    //go:build windows

    package codestatus

    import (
        "fmt"
        "os/exec"
        "strconv"
        "strings"
        "syscall"
    )

    // isProcessAlive: 既存のまま維持（tasklist を使用）

    // detachSysProcAttr returns platform-specific SysProcAttr for detaching child process.
    func detachSysProcAttr() *syscall.SysProcAttr {
        return &syscall.SysProcAttr{
            CreationFlags: 0x00000200, // CREATE_NEW_PROCESS_GROUP
        }
    }
    ```
*   **Logic**:
    - ファイル先頭に `//go:build windows` ビルドタグを追加（明示化）
    - import に `"syscall"` を追加
    - `detachSysProcAttr` 関数を追加: `&syscall.SysProcAttr{CreationFlags: 0x00000200}` を返す

#### [NEW] [process_unix.go](file://features/tt/internal/codestatus/process_unix.go)
*   **Description**: Unix 系 OS 用の `isProcessAlive` と `detachSysProcAttr` を実装
*   **Technical Design**:
    ```go
    //go:build !windows

    package codestatus

    import (
        "os"
        "syscall"
    )

    // isProcessAlive checks whether a process with the given PID is still running.
    // On Unix, sends signal 0 to check process existence.
    func isProcessAlive(pid int) bool {
        p, err := os.FindProcess(pid)
        if err != nil {
            return false
        }
        // Signal 0 does not kill but checks if process exists.
        // Returns nil if process exists.
        err = p.Signal(syscall.Signal(0))
        return err == nil
    }

    // detachSysProcAttr returns platform-specific SysProcAttr for detaching child process.
    func detachSysProcAttr() *syscall.SysProcAttr {
        return &syscall.SysProcAttr{
            Setsid: true, // Create new session to detach from parent
        }
    }
    ```
*   **Logic**:
    - `//go:build !windows` でWindows以外のみコンパイル
    - `isProcessAlive`: `os.FindProcess(pid)` でプロセスハンドル取得 → `Signal(0)` で存在確認（`err == nil` なら alive）
    - `detachSysProcAttr`: `Setsid: true` で新しいセッションを作成し、親プロセスから分離

## Step-by-Step Implementation Guide

### Phase 1: クロスコンパイル修正 (TDD)

1. [x] **process_unix.go を新規作成**:
   - `features/tt/internal/codestatus/process_unix.go` を作成
   - `//go:build !windows` タグ、`isProcessAlive` 関数、`detachSysProcAttr` 関数を実装

2. [x] **process_windows.go を修正**:
   - ファイル先頭に `//go:build windows` ビルドタグを追加
   - import に `"syscall"` を追加
   - `detachSysProcAttr` 関数を追加

3. [x] **bgrunner.go を修正**:
   - import から `"syscall"` を削除
   - `StartBackground` 内の `cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: ...}` を `cmd.SysProcAttr = detachSysProcAttr()` に置き換え

4. [x] **ビルド検証（ローカル + クロスコンパイル）**:
   - `./scripts/process/build.sh` でローカルビルド・単体テスト
   - `./scripts/dist/build.sh tt` で5プラットフォームのクロスコンパイル確認

### Phase 2: 引数バリデーション

5. [x] **_lib.sh に validate_tool_id 関数を追加**:
   - `scripts/dist/_lib.sh` の末尾に `validate_tool_id` 関数を追加

6. [x] **各 dist スクリプトに呼び出しを追加**:
   - `github-upload.sh`: `TOOL_ID="$1"` 直後に `validate_tool_id "$TOOL_ID"`
   - `build.sh`: 同上
   - `release.sh`: 同上
   - `publish.sh`: 同上

7. [x] **バリデーション動作確認**:
   - `./scripts/dist/github-upload.sh v0.2.0` → エラーメッセージ確認
   - `./scripts/dist/github-upload.sh unknown-tool` → エラーメッセージ確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```

2. **Cross-Compile Verification**:
   ```bash
   ./scripts/dist/build.sh tt
   ```
   - **期待結果**: `All 5 builds succeeded.`
   - 5プラットフォーム（linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64）全てが `[PASS]` となること

3. **Validation Error Check**:
   ```bash
   ./scripts/dist/github-upload.sh v0.2.0
   ```
   - **期待結果**: `[FAIL] First argument looks like a version, not a tool-id: 'v0.2.0'` が表示され、`exit 1` で終了
   - `Available tools: tt` が表示されること

   ```bash
   ./scripts/dist/build.sh unknown-tool
   ```
   - **期待結果**: `[FAIL] Unknown tool-id: 'unknown-tool'` が表示され、`exit 1` で終了

## Documentation

本変更で影響を受ける既存ドキュメントはなし。`scripts/dist/README.md` の Usage セクションはバリデーション追加に伴い動作が改善されるが、Usage 自体は変更不要。
