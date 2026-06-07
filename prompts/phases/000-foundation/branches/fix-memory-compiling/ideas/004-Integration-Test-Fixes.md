# 004: 統合テスト失敗の原因分析と修正

## 背景 (Background)

統合テストを全カテゴリ (`release-note`, `tt`) で実行した結果、テストが失敗した。
ビルド (`scripts/process/build.sh`) は成功しているため、テストコード自体、またはテスト前提条件（ディレクトリ構造・外部依存）に問題がある。

### 実行環境
- OS: Windows (Git Bash)
- Docker Desktop: 起動済み (Server Version 28.0.1)
- ネットワーク: GitHub API にアクセス可能
- `GITHUB_TOKEN`: Antigravity IDE が注入するダミートークンを `unset` して実行

### 再実行結果サマリ (Docker起動 + GITHUB_TOKEN unset)

| カテゴリ | 合計テスト | 成功 | 失敗 | 修正対象 |
|:---------|:---------:|:----:|:----:|:--------:|
| release-note | 6 | 3 | 3 | テストコードバグ2件 + credential未配置1件 |
| tt | 34 | 32 | 2 | テストコードバグ1件 + 残留コンテナ1件 |

> 初回実行時は Docker 未起動 + ダミー `GITHUB_TOKEN` により tt カテゴリで 14 テストが失敗していたが、環境を整備した再実行で **32/34 成功**まで改善した。

## 要件 (Requirements)

### 必須要件

1. **テストコードのバグを修正し、全テストが成功すること**
2. **テストの期待値が現在のディレクトリ構造・テンプレート出力と一致すること**
3. **`integration_test.sh` で `GITHUB_TOKEN` のダミー値をクリアし、IDE環境でも安定実行できること**

### 任意要件

4. `TestTtDownStopsContainer` の残留コンテナ問題への対処（テスト前のクリーンアップ強化）
5. credential ファイル未配置テストの扱いの検討

## 失敗テスト分析 (Failure Analysis)

### 失敗1: TestScanner_RealPhaseStructure (release-note)

- **原因**: テストが `prompts/phases/000-foundation/ideas/` の存在を検証しているが、現在のプロジェクト構造では `ideas/` は `prompts/phases/000-foundation/branches/{branch-name}/ideas/` に移動している。
- **該当コード**: [scanner_test.go:50-54](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/scanner_test.go#L50-L54)
- **修正方針**: テストを現在のディレクトリ構造 (`branches/` 配下) に合わせて更新する。

### 失敗2: TestScanner_FindBranchFolder (release-note)

- **原因**: テストが `prompts/phases/000-foundation/ideas/fix-module-versioning` を探しているが、(1) ディレクトリ構造が `branches/` 方式に変わっている (2) ブランチ名が `fix-module-versioning` ではなく `fix-memory-compiling` である。
- **該当コード**: [scanner_test.go:57-66](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/scanner_test.go#L57-L66)
- **修正方針**: 現在のブランチ構造 (`branches/fix-memory-compiling/ideas/`) に合わせてパスを修正する。ブランチ名のハードコードを避け、`branches/` 配下の最初のディレクトリを動的に探索する方式を検討する。

### 失敗3: TestConfigLoad_CredentialFileExists (release-note)

- **原因**: `features/release-note/settings/secrets/credential.yaml` が存在しない。これは機密ファイルであり `.gitignore` で除外されている。
- **該当コード**: [config_load_test.go:69-94](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/config_load_test.go#L69-L94)
- **修正方針**: credential ファイルの存在はテスト環境依存。このテストは credential ファイルが配置された環境での動作検証を目的としているため、ファイルが存在しない場合のエラーメッセージを明確にした上で現状維持する。

### 失敗4: TestScaffoldDefault/CreatesExpectedStructure (tt)

- **原因**: scaffold テンプレートの出力ディレクトリ構造が変更され、テストの期待するファイルが生成されなくなった。具体的には以下の3ファイルが存在しない:
  - `prompts/phases/000-foundation/ideas/.gitkeep`
  - `prompts/phases/000-foundation/plans/.gitkeep`
  - `prompts/rules/.gitkeep`
- **該当コード**: [tt_scaffold_test.go:116-131](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_scaffold_test.go#L116-L131)
- **修正方針**: scaffold テンプレートの実際の出力を確認し、期待するファイルリストを更新する。

### 失敗5: TestTtDownStopsContainer (tt)

- **原因**: 前回のテスト実行で残ったコンテナ `tt-integration-test` が存在し、セットアップ時の `docker run` でコンテナ名の競合が発生。
  ```
  docker: Error response from daemon: Conflict. The container name "/tt-integration-test" is already in use
  ```
- **該当コード**: [tt_down_test.go:10-30](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_down_test.go#L10-L30)
- **修正方針**: テストのセットアップで既存コンテナを事前にクリーンアップする処理を追加する。または `TestMain` のセットアップで全テスト開始前に残留コンテナを除去する。

### 環境起因の失敗（修正済み）

#### Docker 未起動 (10テスト) -- Docker Desktop 起動で解決
Docker Desktop を起動することで、`requireDockerAvailable(t)` で落ちていた全10テストが成功した。

#### GitHub API 認証エラー (7テスト) -- `unset GITHUB_TOKEN` で解決
Antigravity IDE がプロセス環境変数 `GITHUB_TOKEN` にダミー値 `github_pat_antigravitydummytoken` を注入していた。`unset GITHUB_TOKEN` することで全7テスト中6テストが成功した（残り1テストは期待ファイルリストの不一致で別原因の失敗）。

## 実現方針 (Implementation Approach)

### Phase 1: テストコードの修正（3件）

1. **release-note テストのディレクトリパス修正**
   - [scanner_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/scanner_test.go) のテストパスを `branches/` 構造に更新
   - `TestScanner_RealPhaseStructure`: `000-foundation/ideas/` → `000-foundation/branches/` 配下を検証
   - `TestScanner_FindBranchFolder`: パスとブランチ名を動的に取得する方式に変更

2. **scaffold テストの期待ファイルリスト修正**
   - [tt_scaffold_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_scaffold_test.go) の `TestScaffoldDefault/CreatesExpectedStructure` で期待するファイルリストを実際のテンプレート出力に合わせて更新

3. **Docker テストの前処理クリーンアップ追加**
   - [tt_down_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_down_test.go) の `TestTtDownStopsContainer` のセットアップで `docker rm -f tt-integration-test` を実行

### Phase 2: integration_test.sh の改善（1件）

4. **`GITHUB_TOKEN` のダミー値クリア**
   - [integration_test.sh](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/scripts/process/integration_test.sh) のテスト実行前に `unset GITHUB_TOKEN` を追加
   - Antigravity IDE 環境でもテストが安定実行されるようにする

## 検証シナリオ (Verification Scenarios)

1. Phase 1 のテストコード修正後、`release-note` カテゴリのテスト（credential テスト除く）が全て成功する
2. Phase 1 のテストコード修正後、`tt` カテゴリの全テストが成功する
3. `integration_test.sh` の `GITHUB_TOKEN` クリア追加後、`unset` なしで scaffold テストが成功する

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```

2. release-note 統合テスト（ディレクトリ構造修正の確認）:
   ```
   scripts/process/integration_test.sh --categories "release-note" --specify "TestScanner"
   ```

3. tt 統合テスト（scaffold テスト + Docker テストの確認）:
   ```
   scripts/process/integration_test.sh --categories "tt" --specify "TestScaffoldDefault|TestTtDownStopsContainer"
   ```

4. 全統合テスト（リグレッション確認）:
   ```
   scripts/process/integration_test.sh
   ```

## 未解決事項 (Open Questions)

> [!IMPORTANT]
> **credential テストの扱い**: `TestConfigLoad_CredentialFileExists` は機密ファイル (`credential.yaml`) に依存しています。CI/CD 環境ではこのファイルは通常存在しません。このテストの失敗を許容するか、テスト環境にダミーの credential ファイルを配置する仕組みを作るか、方針を決める必要があります。
