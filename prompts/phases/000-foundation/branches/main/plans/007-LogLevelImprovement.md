# 007-LogLevelImprovement

> **Source Specification**: [006-LogLevelImprovement.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/006-LogLevelImprovement.md)

## Goal Description

`cmdexec.Runner` のコマンド失敗時ログレベルを呼び出し元が制御可能にし、条件チェック（`[DEBUG]`）・許容エラー（`[WARN]`）・致命的エラー（`[ERROR]`）の3段階を区別することで、ユーザーに矛盾のないログ出力を提供する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 失敗時ログレベルの柔軟化 | Proposed Changes > cmdexec.go (`RunOption`, `RunWithOpts`, `RunInteractiveWithOpts`) |
| 条件チェック: docker inspect → `[DEBUG]` | Proposed Changes > action/runner.go (`DockerRunOutputCheck`) + action/status.go |
| 条件チェック: docker image inspect → `[DEBUG]` | Proposed Changes > action/runner.go (`DockerRunOutputCheck`) + action/up.go |
| 許容エラー: docker stop/rm → `[WARN]` | Proposed Changes > action/runner.go (`DockerRunTolerated`) + action/down.go |
| 許容エラー: git branch -d → `[WARN]` | Proposed Changes > worktree.Manager (呼び出し元がWARNを出力) — 既に `[WARN]` 出力済みのため、cmdexecのERRORログ抑制で対応 |
| `[ERROR] [FAIL]` の冗長二重タグ簡潔化 | Proposed Changes > cmdexec.go (`FailLabel` でタグ制御) |
| verboseなしで条件チェックのCMDログ非表示 | Proposed Changes > cmdexec.go (条件チェック用の `[CMD]` を `[DEBUG]` レベルに) |
| 後方互換: 未指定時は従来通り[ERROR] | Proposed Changes > cmdexec.go (デフォルト値) |

## Proposed Changes

### cmdexec パッケージ

#### [MODIFY] [cmdexec_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/cmdexec/cmdexec_test.go)
*   **Description**: 新APIのユニットテストを追加
*   **Technical Design**:
    ```go
    func TestRunWithOpts_FailLevelDebug(t *testing.T)
    // `false` コマンドをRunWithOptsで実行、FailLevel=LevelDebug
    // → ログに[ERROR]が含まれないことをassert
    // → ログに[DEBUG]が含まれることをassert
    // → err != nil であることをassert

    func TestRunWithOpts_DefaultIsError(t *testing.T)
    // RunOption{} (ゼロ値) で実行
    // → ログに[ERROR]が含まれることをassert（後方互換）

    func TestRunWithOpts_CustomLabel(t *testing.T)
    // FailLabel="SKIP" で実行
    // → ログに[SKIP]が含まれることをassert

    func TestRunWithOpts_QuietCmd(t *testing.T)
    // QuietCmd=true で成功コマンド実行
    // verbose=false の場合 [CMD] が表示されないことをassert
    // verbose=true の場合 [DEBUG] [CMD] として表示されることをassert
    ```

#### [MODIFY] [cmdexec.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/cmdexec/cmdexec.go)
*   **Description**: ログレベル指定可能な新APIを追加
*   **Technical Design**:
    ```go
    // RunOption controls logging behavior for command execution.
    type RunOption struct {
        FailLevel log.Level  // Log level on failure (default: LevelError)
        FailLabel string     // Label tag on failure (default: "FAIL")
        QuietCmd  bool       // If true, [CMD] log uses LevelDebug instead of LevelInfo
    }

    // applyDefaults fills zero values with sensible defaults.
    func (o RunOption) applyDefaults() RunOption {
        if o.FailLabel == "" {
            o.FailLabel = "FAIL"
        }
        // FailLevel: LevelDebug=0 なので、ゼロ値判定不可
        // → 新フィールド FailLevelSet bool を使う、
        //   もしくは LevelError をデフォルトとする別のアプローチ
        // 簡易アプローチ: FailLevel=0(LevelDebug)の場合もそのまま使用
        // デフォルトをLevelErrorにしたい → FailLevelPtr *log.Level を使う
        return o
    }
    ```

    **ゼロ値問題の解決策**: `log.Level` の `LevelDebug=0` がGoのゼロ値と一致するため、
    未指定かDebug指定かを区別できない。以下のように解決する:

    ```go
    // RunOption controls logging behavior for command execution.
    type RunOption struct {
        FailLevel    log.Level  // Log level on failure
        FailLevelSet bool       // If true, use FailLevel; if false, default to LevelError
        FailLabel    string     // Label tag on failure (default: "FAIL")
        QuietCmd     bool       // If true, [CMD] log uses LevelDebug
    }
    ```

    これにより `RunOption{}` はデフォルトで `LevelError` となり、後方互換。
    `RunOption{FailLevel: log.LevelDebug, FailLevelSet: true}` で明示的にDebugを指定。

    **プリセットファクトリ関数**:
    ```go
    // CheckOpt returns a RunOption for condition-check commands.
    // Failures are logged at DEBUG level with [SKIP] label.
    func CheckOpt() RunOption {
        return RunOption{FailLevel: log.LevelDebug, FailLevelSet: true, FailLabel: "SKIP", QuietCmd: true}
    }

    // ToleratedOpt returns a RunOption for tolerated-failure commands.
    // Failures are logged at WARN level.
    func ToleratedOpt() RunOption {
        return RunOption{FailLevel: log.LevelWarn, FailLevelSet: true, FailLabel: "FAIL", QuietCmd: false}
    }
    ```

    **新メソッド**:
    ```go
    func (r *Runner) RunWithOpts(opts RunOption, name string, args ...string) (string, error)
    func (r *Runner) RunInteractiveWithOpts(opts RunOption, name string, args ...string) error
    ```

*   **Logic**:
    *   `RunWithOpts` は既存 `Run` のロジックをベースに、以下を変更:
        *   `opts.QuietCmd` が true の場合、`[CMD]` ログを `r.Logger.Debug()` で出力
        *   失敗時のログ出力: `opts.FailLevelSet` が false なら `LevelError`、true なら `opts.FailLevel` を使用
        *   失敗時のタグ: `opts.FailLabel` を使用（デフォルト `"FAIL"`）
    *   既存の `Run()` は `RunWithOpts(RunOption{}, name, args...)` に委譲
    *   既存の `RunInteractive()` は `RunInteractiveWithOpts(RunOption{}, name, args...)` に委譲

---

### action パッケージ

#### [MODIFY] [runner.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/runner.go)
*   **Description**: 条件チェック・許容エラー用のラッパーメソッドを追加
*   **Technical Design**:
    ```go
    // DockerRunCheck executes a Docker command for condition checking.
    // Failures are logged at DEBUG level (not ERROR).
    func (r *Runner) DockerRunCheck(args ...string) error {
        return r.CmdRunner.RunInteractiveWithOpts(cmdexec.CheckOpt(), "docker", args...)
    }

    // DockerRunOutputCheck is like DockerRunOutput but for condition checks.
    func (r *Runner) DockerRunOutputCheck(args ...string) (string, error) {
        return r.CmdRunner.RunWithOpts(cmdexec.CheckOpt(), "docker", args...)
    }

    // DockerRunTolerated executes a Docker command where failure is acceptable.
    // Failures are logged at WARN level.
    func (r *Runner) DockerRunTolerated(args ...string) error {
        return r.CmdRunner.RunInteractiveWithOpts(cmdexec.ToleratedOpt(), "docker", args...)
    }
    ```

#### [MODIFY] [status.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/status.go)
*   **Description**: `docker inspect` を `DockerRunOutputCheck` に変更
*   **Logic**:
    *   L20: `r.DockerRunOutput(...)` → `r.DockerRunOutputCheck(...)`
    *   効果: コンテナ不存在時に `[ERROR] [FAIL]` ではなく `[DEBUG] [SKIP]` を出力

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/up.go)
*   **Description**: `imageExists` のdocker image inspectを `DockerRunOutputCheck` に変更
*   **Logic**:
    *   L106: `r.DockerRunOutput(...)` → `r.DockerRunOutputCheck(...)`
    *   効果: イメージ不存在時に `[DEBUG] [SKIP]` を出力

#### [MODIFY] [down.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/down.go)
*   **Description**: `docker stop/rm` を `DockerRunTolerated` に変更
*   **Logic**:
    *   L6: `r.DockerRun("stop", ...)` → `r.DockerRunTolerated("stop", ...)`
    *   L11: `r.DockerRun("rm", ...)` → `r.DockerRunTolerated("rm", ...)`
    *   効果: 既に停止/削除済みの場合に `[WARN]` を出力（`[ERROR]` ではなく）

## Step-by-Step Implementation Guide

1.  **ユニットテスト追加（TDD）**:
    *   `cmdexec_test.go` に `TestRunWithOpts_FailLevelDebug`, `TestRunWithOpts_DefaultIsError`, `TestRunWithOpts_CustomLabel`, `TestRunWithOpts_QuietCmd` を追加
    *   テストが「コンパイルエラーまたはFAIL」であることを確認

2.  **`cmdexec.go` に新API追加**:
    *   `RunOption` 構造体を定義
    *   `CheckOpt()`, `ToleratedOpt()` ファクトリ関数を定義
    *   `RunWithOpts()`, `RunInteractiveWithOpts()` を実装
    *   既存 `Run()` / `RunInteractive()` を新メソッドに委譲

3.  **`action/runner.go` にラッパー追加**:
    *   `DockerRunCheck()`, `DockerRunOutputCheck()`, `DockerRunTolerated()` を追加

4.  **`action/status.go` の改修**:
    *   `DockerRunOutput` → `DockerRunOutputCheck` に変更

5.  **`action/up.go` の改修**:
    *   `imageExists()` 内の `DockerRunOutput` → `DockerRunOutputCheck` に変更

6.  **`action/down.go` の改修**:
    *   `DockerRun("stop"/"rm")` → `DockerRunTolerated(...)` に変更

7.  **ビルドとテスト実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: 新ユニットテスト4つがPASS、既存テスト6つもPASS

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```
    *   **Log Verification**: 全8テストPASS、ログに `[ERROR] [FAIL]` が正常フロー中に出力されないことを確認

## Documentation

なし（内部リファクタリングのため）
