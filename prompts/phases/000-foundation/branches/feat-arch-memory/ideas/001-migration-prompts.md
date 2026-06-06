# 古いプロンプトフォルダから新しいプロンプトフォルダ構成へのマイグレーション仕様

## 背景

プロンプト管理のディレクトリ構成が変更されました。

- 古い構成: `prompts_old/phases/000-foundation/(ideas|plans)/[ブランチ名]/`
- 新しい構成: `prompts/phases/000-foundation/branches/[ブランチ名]/(ideas|plans)/`

古いフォルダ（`prompts_old/phases/000-foundation`）の下にあるすべてのブランチフォルダ内のファイルを、新しいフォルダ（`prompts/phases/000-foundation`）の新しい構成へと移行するためのマイグレーションスクリプトを作成し、実行します。

また、この変更に伴い、現在のステータスを表示するスクリプト `scripts/utils/show_current_status.sh` が新しいプロンプト管理のディレクトリ構成に対応していないため、常に `next_idea_id` や `next_plan_id` が `000` になるバグが発生しています。これも併せて修正します。

## 要件

1. マイグレーションスクリプトを `scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh` として新規作成します。
2. マイグレーションスクリプトは、`prompts_old/phases/000-foundation/ideas/` および `prompts_old/phases/000-foundation/plans/` 配下にあるブランチ名フォルダを検出します。
3. 検出したブランチ名フォルダごとに、対応する移行先ディレクトリ（`prompts/phases/000-foundation/branches/[ブランチ名]/ideas/` または `prompts/phases/000-foundation/branches/[ブランチ名]/plans/`）を作成します。
4. 古いフォルダ内のファイルを新しいフォルダへ移動（またはコピー）します。
5. コピー・移動が成功したあと、古いブランチ名フォルダを削除（または空に）します。
6. すでに移行先に同じブランチ名のフォルダが存在する場合でも、エラーにならずマージ・上書きができるようにします。
7. 進捗状況をログ（標準出力）に分かりやすく表示します。
8. `scripts/utils/show_current_status.sh` を修正し、新しいプロンプトディレクトリ構成（`prompts/phases/[Phase]/branches/[Branch]/(ideas|plans)`）から正しく次の ID を計算できるようにします。

## 実現方針

### マイグレーションスクリプトの設計

- **対象ディレクトリ**:
  - 元: `prompts_old/phases/000-foundation`
  - 先: `prompts/phases/000-foundation`
- **スクリプトの処理手順**:
  1. `prompts_old/phases/000-foundation/ideas/` の配下にあるサブディレクトリ（`.` や `..`、`.gitkeep` を除く）をループで処理します。
  2. 各サブディレクトリ名をブランチ名 `branch_name` とします。
  3. `ideas` フォルダの移行:
     - 移行元: `prompts_old/phases/000-foundation/ideas/${branch_name}`
     - 移行先: `prompts/phases/000-foundation/branches/${branch_name}/ideas`
     - 移行先ディレクトリを作成し、ファイルをコピー/移動します。
  4. `plans` フォルダの移行:
     - 移行元: `prompts_old/phases/000-foundation/plans/${branch_name}`
     - 移行先: `prompts/phases/000-foundation/branches/${branch_name}/plans`
     - 移行先ディレクトリを作成し、ファイルをコピー/移動します。
  5. 移行が成功したブランチについて、古いブランチディレクトリ（`prompts_old/phases/000-foundation/ideas/${branch_name}` および `prompts_old/phases/000-foundation/plans/${branch_name}`）を削除します。

### ステータス表示スクリプトの修正

`scripts/utils/show_current_status.sh` 内の `IDEAS_DIR` と `PLANS_DIR` の定義を以下のように修正します。

- 修正前:
  ```bash
  IDEAS_DIR="prompts/phases/${PHASE_NAME}/ideas/${BRANCH_NAME}"
  PLANS_DIR="prompts/phases/${PHASE_NAME}/plans/${BRANCH_NAME}"
  ```
- 修正後:
  ```bash
  IDEAS_DIR="prompts/phases/${PHASE_NAME}/branches/${BRANCH_NAME}/ideas"
  PLANS_DIR="prompts/phases/${PHASE_NAME}/branches/${BRANCH_NAME}/plans"
  ```

## 検証シナリオ

### 手動検証

1. スクリプトを実行する前に、古いディレクトリ配下のファイル数やブランチフォルダ一覧を記録しておきます。
2. スクリプト `scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh` を実行します。
3. 実行後、新しいディレクトリ `prompts/phases/000-foundation/branches/` 配下に各ブランチのディレクトリが作成され、その中に `ideas/` や `plans/` が正しく配置されていることを確認します。
4. 古いディレクトリ `prompts_old/phases/000-foundation/ideas/` および `plans/` 配下からブランチフォルダが削除されていることを確認します（`.gitkeep` は残します）。
5. 修正した `scripts/utils/show_current_status.sh` を実行し、`feat-arch-memory` ブランチにおける `next_idea_id` が `002`（`000-reorganize-release-scripts.md` と `001-migration-prompts.md` が存在するため）になることを確認します。

## テスト項目

### ビルド・全体検証

本スクリプトは、ソースコードのビルドや製品の統合テストには影響しませんが、プロジェクトの整合性を保つため、ビルドパイプラインに影響がないことを確認します。

1. マイグレーションスクリプトおよび修正したステータス表示スクリプトの構文チェック:
   `bash -n scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh`
   `bash -n scripts/utils/show_current_status.sh`
2. プロジェクトのビルドが成功することの確認:
   `scripts/process/build.sh`

