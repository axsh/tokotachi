# 古いプロンプトフォルダから新しいプロンプトフォルダ構成へのマイグレーション仕様

## 背景

プロンプト管理のディレクトリ構成が変更されました。

- 古い構成: `prompts_old/phases/000-foundation/(ideas|plans)/[ブランチ名]/`
- 新しい構成: `prompts/phases/000-foundation/branches/[ブランチ名]/(ideas|plans)/`

古いフォルダ（`prompts_old/phases/000-foundation`）の下にあるすべてのブランチフォルダ内のファイルを、新しいフォルダ（`prompts/phases/000-foundation`）の新しい構成へと移行するためのマイグレーションスクリプトを作成し、実行します。

また、この変更に伴い、現在のステータスを表示するスクリプト `scripts/utils/show_current_status.sh` が新しいプロンプト管理のディレクトリ構成に対応していないため、常に `next_idea_id` や `next_plan_id` が `000` になるバグが発生していました。これについてはすでに先行して修正・動作確認を完了しています。

## 要件

1. マイグレーションスクリプトを `scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh` として新規作成します。
2. マイグレーションスクリプトはデフォルトで **dry-run** モードとして動作し、実際のファイル移動やディレクトリ削除は行わず、移動対象ファイルや削除対象ディレクトリのシミュレーション結果を出力します。
3. 実際のファイル移動・ディレクトリ削除を実行するには、明示的に `--adapt` オプションを指定する必要があります。
4. スクリプトは、`prompts_old/phases/000-foundation/ideas/` および `prompts_old/phases/000-foundation/plans/` 配下にあるブランチ名フォルダを検出します。
5. 検出したブランチ名フォルダごとに、対応する移行先ディレクトリ（`prompts/phases/000-foundation/branches/[ブランチ名]/ideas/` または `prompts/phases/000-foundation/branches/[ブランチ名]/plans/`）を作成します。
6. 古いフォルダ内のファイルを新しいフォルダへ移動（またはコピー）します。
7. コピー・移動が成功したあと、古いブランチ名フォルダを削除（または空に）します。
8. すでに移行先に同じブランチ名のフォルダが存在する場合でも、エラーにならずマージ・上書きができるようにします。
9. 進捗状況をログ（標準出力）に分かりやすく表示します。

## 実現方針

### マイグレーションスクリプトの設計

- **対象ディレクトリ**:
  - 元: `prompts_old/phases/000-foundation`
  - 先: `prompts/phases/000-foundation`
- **スクリプトのオプション**:
  - オプションなし: `dry-run` モードでシミュレーションを実行。
  - `--adapt`: 実際にマイグレーション（フォルダ作成、ファイル移動、古いフォルダ削除）を実行。
- **スクリプトの処理手順**:
  1. 引数に `--adapt` が指定されているかどうかを判定し、フラグ変数 `ADAPT_MODE` (true/false) を設定します。それ以外の未定義なオプションが指定された場合はエラーを出力して終了します。
  2. `prompts_old/phases/000-foundation/ideas/` の配下にあるサブディレクトリ（`.` や `..`、`.gitkeep` を除く）をループで処理します。
  3. 各サブディレクトリ名をブランチ名 `branch_name` とします。
  4. `ideas` フォルダの移行:
     - 移行元: `prompts_old/phases/000-foundation/ideas/${branch_name}`
     - 移行先: `prompts/phases/000-foundation/branches/${branch_name}/ideas`
     - `ADAPT_MODE` が `true` の場合、移行先ディレクトリを作成し、ファイルをコピー/移動します。`false` (dry-run) の場合は、コピー・作成予定の内容を画面に出力します。
  5. `plans` フォルダの移行:
     - 移行元: `prompts_old/phases/000-foundation/plans/${branch_name}`
     - 移行先: `prompts/phases/000-foundation/branches/${branch_name}/plans`
     - `ADAPT_MODE` が `true` の場合、移行先ディレクトリを作成し、ファイルをコピー/移動します。`false` (dry-run) の場合は、コピー・作成予定の内容を画面に出力します。
  6. 古いフォルダの削除:
     - 移行元ディレクトリにファイルが残っていないか、または移行完了後に、`ADAPT_MODE` が `true` なら古いブランチディレクトリ（`prompts_old/phases/000-foundation/ideas/${branch_name}` および `prompts_old/phases/000-foundation/plans/${branch_name}`）を削除します。`false` (dry-run) の場合は、削除予定として画面に出力します。

## 検証シナリオ

### 手動検証

1. スクリプトを実行する前に、古いディレクトリ配下のファイル数やブランチフォルダ一覧を記録しておきます。
2. まずオプションなしでマイグレーションスクリプトを実行し、**dry-run** モードとして動作することを確認します。
   - 画面に「[Dry-run] 移動予定: ...」や「[Dry-run] 削除予定: ...」と出力されることを確認します。
   - 実際にファイルが移動されていないこと、古いフォルダが削除されていないことを確認します。
3. 次に `--adapt` オプションを指定してスクリプトを実行します。
   `scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh --adapt`
4. 実行後、新しいディレクトリ `prompts/phases/000-foundation/branches/` 配下に各ブランチのディレクトリが作成され、その中に `ideas/` や `plans/` が正しく配置されていることを確認します。
5. 古いディレクトリ `prompts_old/phases/000-foundation/ideas/` および `plans/` 配下からブランチフォルダが削除されていることを確認します（`.gitkeep` は残します）。
6. `scripts/utils/show_current_status.sh` を実行し、`feat-arch-memory` ブランチにおける `next_idea_id` が `002` になることを確認します。

## テスト項目

### ビルド・全体検証

本スクリプトは、ソースコードのビルドや製品の統合テストには影響しませんが、プロジェクトの整合性を保つため、ビルドパイプラインに影響がないことを確認します。

1. マイグレーションスクリプトおよびステータス表示スクリプトの構文チェック:
   `bash -n scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh`
   `bash -n scripts/utils/show_current_status.sh`
2. プロジェクトのビルドが成功することの確認:
   `scripts/process/build.sh`
