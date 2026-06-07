# 004-Logger-Interface

> **Source Specification**: [004-Logger-Interface.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/004-Logger-Interface.md)

## Goal Description

`internal/log` パッケージに閉じた具象 Logger 型を、`pkg/log/` パッケージの公開インターフェースに抽象化する。
これにより、外部プロジェクトが `tokotachi` をライブラリとして利用する際に、任意の Logger 実装（`log/slog`、`zerolog`、`zap` 等）を注入可能にする。

## User Review Required

> [!IMPORTANT]
> **`log.Level` 型の公開について**: `internal/cmdexec/cmdexec.go` の `RunOption.FailLevel` が `log.Level` 型を使用しているため、`pkg/log/` には `Logger` インターフェースだけでなく `Level` 型とログレベル定数も配置します。これにより `cmdexec.RunOption` が `internal/log` から切り離されます。

> [!WARNING]
> **影響範囲**: `Logger` だけでなく `Level` 型も公開するため、変更対象ファイルが仕様書より多くなります。ただし各ファイルの変更は import パスの差し替えと型名の変更のみであり、ロジックの変更はありません。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. 外部プロジェクトが独自 Logger を渡せること | `pkg/log/logger.go` (インターフェース定義) + `tokotachi.go` (Client.Logger フィールド) |
| 2. Tokotachi のカスタムフォーマット維持 | `internal/log/logger.go` (変更なし、既存実装がインターフェースを満たす) |
| 3. 既存の内部利用に影響を与えないこと | 全ファイルの import 差し替え (ロジック変更なし) |
| 4. Logger インターフェースは internal/ 外に配置 | `pkg/log/logger.go` に配置 |
| 5. log/slog アダプタ提供 | `pkg/log/slog_adapter.go` |

## Proposed Changes

### pkg/log (新規パッケージ)

#### [NEW] [logger.go](file://pkg/log/logger.go)
*   **Description**: Logger インターフェースと Level 型を定義
*   **Technical Design**:
    ```go
    package log

    // Level はログの重要度を表す。
    type Level int

    const (
        LevelDebug Level = iota
        LevelInfo
        LevelWarn
        LevelError
    )

    // Logger はログ出力を抽象化するインターフェース。
    type Logger interface {
        Info(format string, args ...any)
        Warn(format string, args ...any)
        Error(format string, args ...any)
        Debug(format string, args ...any)
        // Log は指定されたレベルでメッセージを出力する。
        Log(level Level, format string, args ...any)
    }
    ```
*   **Logic**:
    - `Level` 型と4つのレベル定数は `internal/log/logger.go` と同一の定義
    - `Logger` インターフェースは `Info/Warn/Error/Debug/Log` の5メソッド
    - `Log` メソッドは `cmdexec` が `Logger.Log(failLevel, ...)` で呼び出すために必要

#### [NEW] [slog_adapter.go](file://pkg/log/slog_adapter.go)
*   **Description**: Go 標準 `log/slog` 用アダプタ
*   **Technical Design**:
    ```go
    package log

    import (
        "fmt"
        "log/slog"
    )

    // SlogAdapter は *slog.Logger を Logger インターフェースに適合させる。
    type SlogAdapter struct {
        SlogLogger *slog.Logger
    }

    // NewSlogAdapter は *slog.Logger から Logger を生成する。
    func NewSlogAdapter(l *slog.Logger) *SlogAdapter {
        return &SlogAdapter{SlogLogger: l}
    }

    func (a *SlogAdapter) Info(format string, args ...any)  { a.SlogLogger.Info(fmt.Sprintf(format, args...)) }
    func (a *SlogAdapter) Warn(format string, args ...any)  { a.SlogLogger.Warn(fmt.Sprintf(format, args...)) }
    func (a *SlogAdapter) Error(format string, args ...any) { a.SlogLogger.Error(fmt.Sprintf(format, args...)) }
    func (a *SlogAdapter) Debug(format string, args ...any) { a.SlogLogger.Debug(fmt.Sprintf(format, args...)) }
    func (a *SlogAdapter) Log(level Level, format string, args ...any) {
        // Level → slog.Level マッピング: Debug=slog.LevelDebug, Info=slog.LevelInfo, Warn=slog.LevelWarn, Error=slog.LevelError
    }
    ```

#### [NEW] [slog_adapter_test.go](file://pkg/log/slog_adapter_test.go)
*   **Description**: SlogAdapter の単体テスト
*   **Technical Design**:
    ```go
    // テーブル駆動テストケース:
    // - Info/Warn/Error/Debug 各メソッドが slog.Logger の対応メソッドに委譲されること
    // - Log メソッドの Level マッピングが正しいこと
    // - fmt.Sprintf のフォーマット引数が正しく展開されること
    ```

---

### internal/log (既存パッケージ修正)

#### [MODIFY] [logger.go](file://internal/log/logger.go)
*   **Description**: `Level` 型と定数を `pkg/log` パッケージから import するように変更
*   **Technical Design**:
    - `Level` 型定義と `LevelDebug/LevelInfo/LevelWarn/LevelError` 定数を **削除**
    - `import pkglog "github.com/axsh/tokotachi/pkg/log"` を追加
    - `type Level = pkglog.Level` のエイリアスを定義（既存コードとの互換性維持）
    - `LevelDebug = pkglog.LevelDebug` 等の定数エイリアスを定義
    - `Logger` struct と `New()`, `Info()`, `Warn()`, `Error()`, `Debug()`, `Log()` メソッドは **変更なし**
*   **Logic**:
    - `*log.Logger` は `pkg/log.Logger` インターフェースの5メソッド (`Info/Warn/Error/Debug/Log`) を全て持つため、追加実装なしでインターフェースを満たす

---

### internal/cmdexec (既存パッケージ修正)

#### [MODIFY] [cmdexec.go](file://internal/cmdexec/cmdexec.go)
*   **Description**: `Runner.Logger` を `pkg/log.Logger` インターフェース型に変更、`RunOption.FailLevel` を `pkg/log.Level` に変更
*   **Technical Design**:
    - import を `"github.com/axsh/tokotachi/internal/log"` → `pkglog "github.com/axsh/tokotachi/pkg/log"` に変更
    - `Runner.Logger *log.Logger` → `Runner.Logger pkglog.Logger`
    - `RunOption.FailLevel log.Level` → `RunOption.FailLevel pkglog.Level`
    - `effectiveFailLevel()` の戻り値を `pkglog.Level` に変更
    - `CheckOpt()`, `ToleratedOpt()` 関数内の `log.LevelDebug`, `log.LevelWarn` → `pkglog.LevelDebug`, `pkglog.LevelWarn`
    - `effectiveFailLevel()` のデフォルト値 `log.LevelError` → `pkglog.LevelError`
*   **Logic**: ロジック変更なし。型と import パスの差し替えのみ

#### [MODIFY] [cmdexec_test.go](file://internal/cmdexec/cmdexec_test.go)
*   **Description**: Logger 生成部分のインポート変更
*   **Technical Design**:
    - `newTestRunner` 関数は引き続き `internal/log.New()` で具象 Logger を生成する（`internal/` 内のテストなので問題なし）
    - `log.LevelWarn` の参照はエイリアス経由で動作する

---

### pkg/scaffold (公開パッケージ修正)

#### [MODIFY] [scaffold.go](file://pkg/scaffold/scaffold.go)
*   **Description**: `RunOptions.Logger` と関数引数の Logger 型を `pkg/log.Logger` インターフェースに変更
*   **Technical Design**:
    - import を `"github.com/axsh/tokotachi/internal/log"` → `pkglog "github.com/axsh/tokotachi/pkg/log"` に変更
    - `RunOptions.Logger *log.Logger` → `RunOptions.Logger pkglog.Logger`
    - `Rollback(repoRoot string, logger *log.Logger)` → `Rollback(repoRoot string, logger pkglog.Logger)`
    - `fetchTemplateAndPlacement` 等の内部関数の `logger *log.Logger` 引数を `logger pkglog.Logger` に変更
*   **Logic**: ロジック変更なし。全ての Logger 呼び出し (`Info`, `Warn`, `Error`, `Debug`) はインターフェースのメソッドと一致する

---

### pkg/action (公開パッケージ修正)

#### [MODIFY] [runner.go](file://pkg/action/runner.go)
*   **Description**: `Runner.Logger` を `pkg/log.Logger` インターフェース型に変更
*   **Technical Design**:
    - import を `"github.com/axsh/tokotachi/internal/log"` → `pkglog "github.com/axsh/tokotachi/pkg/log"` に変更
    - `Runner.Logger *log.Logger` → `Runner.Logger pkglog.Logger`
*   **Logic**: ロジック変更なし

#### [MODIFY] [pending_changes.go](file://pkg/action/pending_changes.go)
*   **Description**: `displayPendingChanges` の Logger 引数を `pkg/log.Logger` インターフェース型に変更、`log.LevelDebug` の参照を更新
*   **Technical Design**:
    - import を `"github.com/axsh/tokotachi/internal/log"` → `pkglog "github.com/axsh/tokotachi/pkg/log"` に変更
    - `displayPendingChanges(logger *log.Logger, ...)` → `displayPendingChanges(logger pkglog.Logger, ...)`
    - `collectPendingChanges` 内の `log.LevelDebug` → `pkglog.LevelDebug`
*   **Logic**: ロジック変更なし

#### [MODIFY] [close_test.go](file://pkg/action/close_test.go)
*   **Description**: テストの Logger 生成部分のインポート変更
*   **Technical Design**:
    - `newTestEnv` 関数内の `log.New(&buf, false)` は引き続き `internal/log.New()` を使用
    - `Runner.Logger` への代入は `internal/log.Logger` が `pkg/log.Logger` インターフェースを満たすため動作する

---

### pkg/editor (公開パッケージ修正)

#### [MODIFY] [editor.go](file://pkg/editor/editor.go)
*   **Description**: `LaunchOptions.Logger` を `pkg/log.Logger` インターフェース型に変更
*   **Technical Design**:
    - import を `"github.com/axsh/tokotachi/internal/log"` → `pkglog "github.com/axsh/tokotachi/pkg/log"` に変更
    - `LaunchOptions.Logger *log.Logger` → `LaunchOptions.Logger pkglog.Logger`
*   **Logic**: ロジック変更なし

#### [MODIFY] [editor_test.go](file://pkg/editor/editor_test.go)
*   **Description**: テストの Logger 生成部分のインポート変更
*   **Technical Design**:
    - `log.New(&buf, true)` は引き続き `internal/log.New()` を使用（テスト用デフォルト Logger）

---

### pkg/editor 配下の実装ファイル

`pkg/editor/` 配下に `vscode.go`, `cursor.go`, `ag.go`, `claude.go` 等の Launcher 実装があり、これらが `LaunchOptions.Logger` を使用している場合は同様に import と型を更新する。ただし `LaunchOptions.Logger` はインターフェース経由でアクセスするため、呼び出し側は変更不要。

---

### tokotachi.go (外部API)

#### [MODIFY] [tokotachi.go](file://tokotachi.go)
*   **Description**: `Client.Logger` フィールド追加、`newContext()` で Logger を伝播
*   **Technical Design**:
    - import に `pkglog "github.com/axsh/tokotachi/pkg/log"` を追加
    - `Client` struct に `Logger pkglog.Logger` フィールドを追加
    - `newContext()` メソッドを修正: `c.Logger != nil` の場合はそれを使用、`nil` の場合は `log.New(stderr, c.Verbose)` でデフォルト Logger を生成
    - `opContext.logger` の型を `pkglog.Logger` に変更
    - `Scaffold()` メソッド内の `logger := log.New(stderr, c.Verbose)` を `c.Logger` 優先に変更
*   **Logic**:
    ```go
    func (c *Client) newContext() (pkglog.Logger, *cmdexec.Runner, *action.Runner) {
        // c.Logger が設定済みならそれを使用、なければデフォルト生成
        var logger pkglog.Logger
        if c.Logger != nil {
            logger = c.Logger
        } else {
            logger = log.New(stderr, c.Verbose)
        }
        // ... runner, actionRunner 生成
    }
    ```

---

### features/tt/cmd (CLI コマンド)

#### [MODIFY] [scaffold.go](file://features/tt/cmd/scaffold.go)
*   **Description**: import の更新（`internal/log` は引き続き使用可、CLI は internal パッケージ内）
*   **Technical Design**:
    - `log.New(os.Stderr, flagVerbose)` でデフォルト Logger を生成する部分は変更なし
    - `scaffold.RunOptions.Logger` に代入する部分は `internal/log.Logger` が `pkg/log.Logger` インターフェースを満たすため動作する

#### [MODIFY] [common.go](file://features/tt/cmd/common.go)
*   **Description**: Logger 参照があれば同様に確認・更新
*   **Technical Design**: `internal/log` を引き続き使用（CLI は internal 参照可能）

## Step-by-Step Implementation Guide

### Phase 1: インターフェース定義 (テストファースト)

- [x] 1. **`pkg/log/logger.go` を作成**: `Logger` インターフェースと `Level` 型・定数を定義
- [x] 2. **`pkg/log/slog_adapter_test.go` を作成**: SlogAdapter のテストを先に記述（TDD）
- [x] 3. **`pkg/log/slog_adapter.go` を作成**: テストが通るようにアダプタを実装
- [x] 4. **ビルド確認**: `./scripts/process/build.sh` を実行して新パッケージのコンパイルを確認

### Phase 2: internal/log の Level エイリアス化

- [x] 5. **`internal/log/logger.go` を修正**: `Level` 型を `pkg/log.Level` のエイリアスに変更、定数もエイリアス化
- [x] 6. **ビルド確認**: `./scripts/process/build.sh` で既存テストがパスすることを確認

### Phase 3: internal/cmdexec の Logger インターフェース化

- [x] 7. **`internal/cmdexec/cmdexec.go` を修正**: `Runner.Logger` と `RunOption.FailLevel` の型を変更
- [x] 8. **`internal/cmdexec/cmdexec_test.go` を確認**: 型エイリアスにより変更不要なことを確認
- [x] 9. **ビルド確認**: `./scripts/process/build.sh`

### Phase 4: pkg/ パッケージの Logger インターフェース化

- [x] 10. **`pkg/scaffold/scaffold.go` を修正**: `RunOptions.Logger` と関数引数をインターフェース型に変更
- [x] 11. **`pkg/action/runner.go` を修正**: `Runner.Logger` をインターフェース型に変更
- [x] 12. **`pkg/action/pending_changes.go` を修正**: Logger 引数と Level 参照を更新
- [x] 13. **`pkg/editor/editor.go` を修正**: `LaunchOptions.Logger` をインターフェース型に変更
- [x] 14. **ビルド確認**: `./scripts/process/build.sh`

### Phase 5: tokotachi.go の Logger 注入対応

- [x] 15. **`tokotachi.go` を修正**: `Client.Logger` フィールド追加、`newContext()` と `Scaffold()` を修正
- [x] 16. **ビルド確認**: `./scripts/process/build.sh`

### Phase 6: 最終検証

- [x] 17. **全テスト実行**: `./scripts/process/build.sh`
- [x] 18. **統合テスト実行**: `./scripts/process/integration_test.sh`

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    各 Phase 完了時にビルドスクリプトを実行。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    全変更完了後に統合テストを実行。
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: 統合テストの Scaffold 実行ログに `[INFO]`, `[WARN]` 等のプレフィックスが正常に出力されていることを確認

3.  **SlogAdapter 単体テスト**:
    新規テストファイル `pkg/log/slog_adapter_test.go` で以下を検証:
    - `Info/Warn/Error/Debug` が `slog.Logger` の対応メソッドに委譲される
    - `Log(level, ...)` が Level に応じた slog メソッドを呼ぶ
    - フォーマット引数が正しく展開される

## Documentation

#### [MODIFY] [004-Logger-Interface.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/004-Logger-Interface.md)
*   **更新内容**: `Level` 型の公開と `cmdexec.RunOption` への影響を追記
