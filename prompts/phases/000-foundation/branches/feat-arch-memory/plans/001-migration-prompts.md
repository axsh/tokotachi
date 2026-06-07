# 001-migration-prompts

> **Source Specification**: [001-migration-prompts.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/prompts/phases/000-foundation/branches/feat-arch-memory/ideas/001-migration-prompts.md)

## Goal Description

古いプロンプトフォルダ（`prompts_old/phases/000-foundation`）の下にあるすべてのブランチフォルダ内のファイルを、新しいプロンプトフォルダ（`prompts/phases/000-foundation`）の新しい構成へと移行するためのマイグレーションスクリプト `scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh` を作成・実行します。

また、`show_current_status.sh` が新しいプロンプト構成に対応できず、常に `000` を返していたバグの修正についても検証プランに含めます（修正自体は先行して完了しています）。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| マイグレーションスクリプトを `scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh` として新規作成する。 | [Proposed Changes > scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh](#new-restruct_prompts_folder_to_v20260604shfilecusersyamyamyprogtokotachiworkfeat-arch-memoryscriptsutilsmigrationrestruct_prompts_folder_to_v20260604sh) |
| デフォルトで `dry-run` モードとして動作し、シミュレーション結果を出力する。 | [Proposed Changes > scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh](#new-restruct_prompts_folder_to_v20260604shfilecusersyamyamyprogtokotachiworkfeat-arch-memoryscriptsutilsmigrationrestruct_prompts_folder_to_v20260604sh) (ADAPT_MODE) |
| `--adapt` オプションが指定されたときのみ、実際の移動や削除を実行する。 | [Proposed Changes > scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh](#new-restruct_prompts_folder_to_v20260604shfilecusersyamyamyprogtokotachiworkfeat-arch-memoryscriptsutilsmigrationrestruct_prompts_folder_to_v20260604sh) (ADAPT_MODE) |
| ブランチ名フォルダを検出し、新しい移行先ディレクトリを作成・移動する。 | [Proposed Changes > scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh](#new-restruct_prompts_folder_to_v20260604shfilecusersyamyamyprogtokotachiworkfeat-arch-memoryscriptsutilsmigrationrestruct_prompts_folder_to_v20260604sh) (Ideas/Plans loop) |
| コピー・移動が成功したあと、古いブランチ名フォルダを削除する。 | [Proposed Changes > scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh](#new-restruct_prompts_folder_to_v20260604shfilecusersyamyamyprogtokotachiworkfeat-arch-memoryscriptsutilsmigrationrestruct_prompts_folder_to_v20260604sh) (Cleanup) |
| すでに移行先に同じブランチ名フォルダが存在する場合でも、マージ・上書きを許容する。 | [Proposed Changes > scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh](#new-restruct_prompts_folder_to_v20260604shfilecusersyamyamyprogtokotachiworkfeat-arch-memoryscriptsutilsmigrationrestruct_prompts_folder_to_v20260604sh) (mkdir -p, cp/mv) |
| `scripts/utils/show_current_status.sh` を修正し、新しいプロンプトディレクトリ構成から正しく次の ID を計算できるようにする。 | 修正完了済み。動作検証は [Verification Plan](#verification-plan) にて実施。 |

## Proposed Changes

### scripts/utils/migration

#### [NEW] [restruct_prompts_folder_to_v20260604.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh)
*   **Description**: 古いプロンプト構成から新しい構成へブランチ単位で移行するためのスクリプト。
*   **Technical Design**:
    *   `ADAPT_MODE=false` (デフォルト)
    *   コマンドライン引数の解析:
        *   `--adapt`: `ADAPT_MODE=true`
        *   `-h` / `--help`: ヘルプを表示して終了。
        *   その他の引数: エラーメッセージを表示して `exit 1`。
*   **Logic**:
    1.  古い移行元ディレクトリ `prompts_old/phases/000-foundation` を基準とし、`ideas/` 配下のサブディレクトリ（ブランチ名）を一覧取得する。
    2.  各ブランチ `branch` について以下の処理を行う:
        - `ideas/` フォルダの移行:
            - 古いパス: `prompts_old/phases/000-foundation/ideas/${branch}`
            - 新しいパス: `prompts/phases/000-foundation/branches/${branch}/ideas`
            - `ADAPT_MODE=true` の場合、移行先を `mkdir -p` し、古いパス内のすべてのファイル（`.gitkeep` 等があればそれも含む）を新しいパスにコピー/移動する。
            - `ADAPT_MODE=false` の場合、移動予定の内容を `echo` 出力する。
        - `plans/` フォルダの移行:
            - 古いパス: `prompts_old/phases/000-foundation/plans/${branch}`
            - 新しいパス: `prompts/phases/000-foundation/branches/${branch}/plans`
            - `plans` 側も古いパスが存在すれば、同様に移行先を `mkdir -p` してコピー/移動する。
            - `ADAPT_MODE=false` の場合、移動予定の内容を `echo` 出力する。
        - 古いブランチフォルダの削除:
            - `ADAPT_MODE=true` の場合、移行に成功した古いブランチフォルダ（`ideas/${branch}` および `plans/${branch}`）を `rm -rf` で削除する。
            - `ADAPT_MODE=false` の場合、削除予定の内容を `echo` 出力する。

## Step-by-Step Implementation Guide

1.  **Create Migration Script File**:
    *   `scripts/utils/migration/` ディレクトリを作成（なければ）。
    *   `scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh` を新規作成する。
    *   実行権限を付与する。
2.  **Implement Script Logic**:
    *   引数解析（`--adapt`, `--help` など）を実装。
    *   `dry-run` モードと `adapt` モードの分岐処理を実装。
    *   ループによるディレクトリ走査とコピー/移動・削除処理を実装。
3.  **Syntactic Verification**:
    *   `bash -n scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh` で構文エラーがないことを確認する。
4.  **Dry-run Testing**:
    *   オプションなしで実行し、期待されるファイル移動・削除ログが出力されること、および実際のファイル移動が発生していないことを確認する。
5.  **Actual Migration Execution**:
    *   `--adapt` オプションを付与して実行し、実際にマイグレーションを完了させる。
6.  **Post-Migration Status Check**:
    *   `scripts/utils/show_current_status.sh` を実行し、`next_idea_id: 002`, `next_plan_id: 001` と出力されることを確認する。

## Verification Plan

### Automated Verification

#### 1. 構文チェックと全体ビルド確認
マイグレーションスクリプトが作成された状態で、プロジェクトのビルドおよび構文チェックが通ることを確認します。

```bash
# 構文チェック
bash -n scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh
bash -n scripts/utils/show_current_status.sh

# 全体ビルド (Goコード等に影響がないことの確認)
./scripts/process/build.sh
```

#### 2. Dry-run 動作の検証
スクリプトをオプションなしで起動し、ログ出力のみが行われ、実ファイルが変化しないことを検証します。

```bash
# dry-run の実行
./scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh

# ファイルが移動していないことを確認 (例として特定の古いブランチフォルダ内のファイルが存在し続けていること)
ls prompts_old/phases/000-foundation/ideas/feat-devctl-list-up/
```

#### 3. 実際のマイグレーション実行と整合性検証
実際に `--adapt` オプションを付けて実行し、マイグレーションが完了した後の状態を検証します。

```bash
# マイグレーション実行
./scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh --adapt

# 古いブランチフォルダが削除されていることの確認
# (ideas/ と plans/ にブランチ名フォルダがなく、.gitkeep のみが残っている状態)
ls prompts_old/phases/000-foundation/ideas/

# 新しいブランチフォルダ構成にファイルが移動していることの確認
ls prompts/phases/000-foundation/branches/feat-devctl-list-up/ideas/

# ステータス表示スクリプトが正しく動作することの確認
./scripts/utils/show_current_status.sh
```

## Documentation

None.
