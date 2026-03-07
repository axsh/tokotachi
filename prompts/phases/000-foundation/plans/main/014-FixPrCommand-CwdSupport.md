# 014-FixPrCommand-CwdSupport

> **Source Specification**: [012-FixPrCommand-CwdSupport.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/012-FixPrCommand-CwdSupport.md)

## Goal Description

`devctl pr` コマンドが `gh pr create` 実行時に存在しない `--repo-dir` フラグを使用してエラーになる問題を修正する。`cmdexec.RunOption` に `Dir` フィールドを追加し、外部コマンドの作業ディレクトリ（cwd）を指定可能にする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: RunOption に Dir フィールドを追加 | Proposed Changes > cmdexec パッケージ > `cmdexec.go` |
| R2: pr.go の修正 (`--repo-dir` 削除、Dir 使用) | Proposed Changes > action パッケージ > `pr.go` |
| R3: DryRun 時のログに Dir 情報を表示 | Proposed Changes > cmdexec パッケージ > `cmdexec.go` |

## Proposed Changes

### cmdexec パッケージ

#### [MODIFY] [cmdexec_test.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/internal/cmdexec/cmdexec_test.go)

*   **Description**: `RunOption.Dir` の動作を検証するユニットテストを追加する
*   **Technical Design**:
    *   以下のテストケースを追加:
    ```go
    func TestRunWithOpts_Dir(t *testing.T)
    func TestRunInteractiveWithOpts_Dir(t *testing.T)
    func TestRun_DryRunWithDir(t *testing.T)
    ```
*   **Logic**:
    *   `TestRunWithOpts_Dir`: `Dir` に一時ディレクトリを指定し、`pwd` (Windows では `cd`) を実行して出力が指定ディレクトリと一致することを検証
    *   `TestRunInteractiveWithOpts_Dir`: `Dir` を設定して `RunInteractiveWithOpts` 呼び出し。標準的なコマンド実行でエラーが出ないことを検証（インタラクティブ操作はテスト困難なため、コマンドが正常終了することを確認）
    *   `TestRun_DryRunWithDir`: DryRun モードで `Dir` を設定した場合、ログ出力に `(in <dir>)` が含まれることを検証

---

#### [MODIFY] [cmdexec.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/internal/cmdexec/cmdexec.go)

*   **Description**: `RunOption` に `Dir` フィールドを追加し、`RunWithOpts` / `RunInteractiveWithOpts` で `exec.Cmd.Dir` を設定する
*   **Technical Design**:
    *   `RunOption` 構造体の変更:
    ```go
    type RunOption struct {
        FailLevel    log.Level
        FailLevelSet bool
        FailLabel    string
        QuietCmd     bool
        Dir          string // Working directory for command execution (empty = inherit process cwd)
    }
    ```
    *   `RunWithOpts` 内の `exec.Command` 呼び出し後:
    ```go
    cmd := exec.Command(name, args...)
    if opts.Dir != "" {
        cmd.Dir = opts.Dir
    }
    ```
    *   `RunInteractiveWithOpts` 内でも同様に `cmd.Dir` を設定
    *   DryRun ログの変更:
    ```go
    if r.DryRun {
        if opts.Dir != "" {
            r.Logger.Info("[DRY-RUN] (in %s) %s", opts.Dir, cmdLine)
        } else {
            r.Logger.Info("[DRY-RUN] %s", cmdLine)
        }
        // ...
    }
    ```
*   **Logic**:
    *   `Dir` が空文字列の場合、`cmd.Dir` は設定しない（Go の既定動作でプロセスの cwd を継承）
    *   `Dir` が設定されている場合、`cmd.Dir` にその値を代入する
    *   `DryRun` 時は `Dir` の有無に応じてログフォーマットを切り替える

---

### action パッケージ

#### [MODIFY] [pr.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/internal/action/pr.go)

*   **Description**: `--repo-dir` フラグを削除し、`RunOption.Dir` で worktree パスを cwd として指定する
*   **Technical Design**:
    ```go
    func (r *Runner) PR(worktreePath string) error {
        ghCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GH", "gh")
        r.Logger.Info("Creating PR from %s...", worktreePath)

        opts := cmdexec.RunOption{Dir: worktreePath}
        if err := r.CmdRunner.RunInteractiveWithOpts(opts, ghCmd, "pr", "create"); err != nil {
            return fmt.Errorf("gh pr create failed: %w", err)
        }
        return nil
    }
    ```
*   **Logic**:
    *   `--repo-dir` フラグの除去: `"pr", "create", "--repo-dir", worktreePath` → `"pr", "create"`
    *   `RunInteractive` → `RunInteractiveWithOpts` に変更し、`Dir: worktreePath` を指定
    *   `gh pr create` は worktree ディレクトリを cwd として実行される

## Step-by-Step Implementation Guide

- [x] 1. **テスト作成 (TDD - Red)**:
    - `cmdexec_test.go` に `TestRunWithOpts_Dir`, `TestRunInteractiveWithOpts_Dir`, `TestRun_DryRunWithDir` を追加
    - この時点ではテストはコンパイルエラー (`Dir` フィールドが存在しない) になることを確認

- [x] 2. **RunOption に Dir フィールド追加**:
    - `cmdexec.go` の `RunOption` 構造体に `Dir string` フィールドを追加
    - テストがコンパイル可能になるが、`Dir` が反映されないためテストが失敗することを確認

- [x] 3. **RunWithOpts に Dir サポートを実装**:
    - `RunWithOpts` 内で `opts.Dir` が空でない場合に `cmd.Dir` を設定するロジックを追加
    - `TestRunWithOpts_Dir` が成功することを確認

- [x] 4. **RunInteractiveWithOpts に Dir サポートを実装**:
    - `RunInteractiveWithOpts` 内で同様に `cmd.Dir` を設定するロジックを追加
    - `TestRunInteractiveWithOpts_Dir` が成功することを確認

- [x] 5. **DryRun ログに Dir 情報を追加**:
    - `RunWithOpts` と `RunInteractiveWithOpts` の DryRun 分岐で、`Dir` が設定されている場合にログフォーマットを変更
    - `TestRun_DryRunWithDir` が成功することを確認

- [x] 6. **pr.go の修正**:
    - `pr.go` の `RunInteractive` 呼び出しを `RunInteractiveWithOpts` に変更
    - `--repo-dir` フラグを削除し、`RunOption{Dir: worktreePath}` を指定

- [x] 7. **ビルド・全テスト実行**:
    - `./scripts/process/build.sh` でビルドと全ユニットテストが成功することを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ビルドスクリプトを実行して、全体のビルドとユニットテストが成功することを確認する。
    ```bash
    ./scripts/process/build.sh
    ```
    *   **検証項目**:
        *   `TestRunWithOpts_Dir`: `Dir` 指定時に正しい cwd でコマンドが実行される
        *   `TestRunInteractiveWithOpts_Dir`: `Dir` 指定時にエラーなく実行される
        *   `TestRun_DryRunWithDir`: DryRun ログに `(in <dir>)` が含まれる
        *   既存テスト全てがパスすること（リグレッションなし）

2.  **DryRun 動作確認**:
    ```bash
    ./bin/devctl pr devctl fix-git --dry-run
    ```
    *   **ログ検証**: `[DRY-RUN]` ログに `gh pr create` が表示され、`--repo-dir` が含まれていないこと

## Documentation

本修正は内部バグ修正であり、外部仕様の変更はないため、ドキュメント更新は不要。
