# 004: 統合テスト失敗の原因分析と修正

## 背景 (Background)

統合テストを全カテゴリ (`release-note`, `tt`) で実行した結果、多数のテストが失敗した。
ビルド (`scripts/process/build.sh`) は成功しているため、テストコード自体、またはテスト前提条件（ディレクトリ構造・外部依存・CLI インターフェース変更）に問題がある。

### 実行環境
- OS: Windows (Git Bash)
- Docker Desktop: インストール済みだがエンジン未起動 (`dockerDesktopLinuxEngine` への接続失敗)
- ネットワーク: GitHub API にはアクセス可能

### 実行結果サマリ

| カテゴリ | 合計テスト数 | 成功 | 失敗 | 失敗率 |
|:---------|:----------:|:----:|:----:|:------:|
| release-note | 5 | 3 | 2 (非Docker) + 1 (credential) | 60% |
| tt | 30+ | 18 | 14 | ~46% |

## 要件 (Requirements)

### 必須要件

1. **テストコード側のバグを修正し、Docker未起動やcredential未配置などの環境要因を除いた全テストが成功すること**
2. **テストの前提条件が現在のディレクトリ構造と一致すること**
3. **CLI インターフェース変更に追従したテストコードに更新すること**

### 任意要件

4. Docker 依存テストについて、Docker未起動時の振る舞い方針を明確にする（現状は `t.Fatalf` で即失敗）
5. credential ファイル未配置テストの扱いの検討

## 失敗テスト分析 (Failure Analysis)

### カテゴリ1: ディレクトリ構造の不一致 (release-note)

#### TestScanner_RealPhaseStructure (FAIL)

- **原因**: テストが `prompts/phases/000-foundation/ideas/` の存在を検証しているが、現在のプロジェクト構造では `ideas/` は `prompts/phases/000-foundation/branches/{branch-name}/ideas/` に移動している。
- **該当コード**: [scanner_test.go:51-53](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/scanner_test.go#L51-L53)
- **修正方針**: テストを現在のディレクトリ構造 (`branches/` 配下) に合わせて更新する。

#### TestScanner_FindBranchFolder (FAIL)

- **原因**: テストが `prompts/phases/000-foundation/ideas/fix-module-versioning` を探しているが、(1) ディレクトリ構造が変わっている (2) ブランチ名が `fix-module-versioning` ではなく `fix-memory-compiling` である。
- **該当コード**: [scanner_test.go:57-66](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/scanner_test.go#L57-L66)
- **修正方針**: 現在のブランチ構造 (`branches/fix-memory-compiling/ideas/`) に合わせてパスを修正する。

#### TestConfigLoad_CredentialFileExists (FAIL)

- **原因**: `features/release-note/settings/secrets/credential.yaml` が存在しない。これは機密ファイルであり `.gitignore` で除外されている可能性が高い。
- **該当コード**: [config_load_test.go:69-94](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/config_load_test.go#L69-L94)
- **修正方針**: credential ファイルが存在しない環境でも通るように、テストの前提条件チェックを改善する。credential ファイルがないことを `t.Fatalf` で報告しつつ、テスト実行を仕組みで制御する方法を検討する（ただし `t.Skip` は禁止）。

### カテゴリ2: Docker 未起動 (tt)

以下のテストは全て `requireDockerAvailable(t)` を呼んでおり、Docker Desktop エンジンが起動していない環境で `t.Fatalf` で即失敗する。

| テスト名 | ファイル |
|:---------|:---------|
| TestIntegrationTestDockerfileBuild | [docker_build_test.go:13](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/docker_build_test.go#L13) |
| TestTtDockerfileBuild | [docker_build_test.go:31](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/docker_build_test.go#L31) |
| TestTtDelete_BlockedByRunningContainer | [tt_create_delete_test.go:43](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_create_delete_test.go#L43) |
| TestTtDownStopsContainer | [tt_down_test.go:10](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_down_test.go#L10) |
| TestTtDownNoopWhenNotRunning | [tt_down_test.go:32](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_down_test.go#L32) |
| TestTtStatusWhenRunning | [tt_status_test.go:11](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_status_test.go#L11) |
| TestTtStatusWhenStopped | [tt_status_test.go:33](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_status_test.go#L33) |
| TestTtUpGitWorktree | [tt_up_git_test.go:11](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_up_git_test.go#L11) |
| TestTtUpStartsContainer | [tt_up_test.go:10](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_up_test.go#L10) |
| TestTtUpIdempotent | [tt_up_test.go:31](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_up_test.go#L31) |

- **原因**: Docker Desktop エンジンが起動していない（`//./pipe/dockerDesktopLinuxEngine` が見つからない）。
- **修正方針**: これらのテストは Docker 環境がなければ実行不可能であるため、テストコードの修正ではなく実行条件の整備（Docker Desktop の起動）で対応する。ただし、プロジェクトルールで `t.Skip` は禁止されているため、Docker 未起動時に `t.Fatalf` で落ちるのは正しい動作である。

> **方針決定が必要**: Docker 依存テストを通常のテスト実行から分離するために、Go build tags (`//go:build docker`) やカテゴリ分離 (例: `tests/tt-docker/`) を導入すべきか？

### カテゴリ3: GitHub API 認証エラー (tt, scaffold系)

以下のテストは全て GitHub API から `HTTP 401 (Unauthorized)` エラーを受け取って失敗している。

| テスト名 | エラー内容 |
|:---------|:----------|
| TestScaffoldDefault | `HTTP 401` - catalog/scaffolds/6/j/v/n.yaml の取得失敗 |
| TestScaffoldList | `HTTP 401` - meta.yaml の取得失敗 |
| TestScaffoldDefaultLocaleJa | `HTTP 401` - catalog/scaffolds/6/j/v/n.yaml の取得失敗 |
| TestScaffoldRootFlag | `HTTP 401` - catalog/scaffolds/6/j/v/n.yaml の取得失敗 |
| TestScaffoldWithDependencies | `HTTP 401` - catalog/scaffolds/b/i/b/l.yaml の取得失敗 |
| TestScaffoldDownloadHistory | `HTTP 401` - catalog/scaffolds/6/j/v/n.yaml の取得失敗 |
| TestScaffoldSkipAlreadyDownloaded | `HTTP 401` - catalog/scaffolds/6/j/v/n.yaml の取得失敗 |

- **原因**: **Antigravity IDE がプロセス環境変数 `GITHUB_TOKEN` にダミー値 `github_pat_antigravitydummytoken` を注入している。** この無効なトークンが GitHub API リクエスト時に送信され、`HTTP 401 (Unauthorized)` を引き起こしている。Windows のシステム/ユーザー環境変数には存在せず、シェルプロファイルにも設定がないため、IDE プロセスが直接注入していると判断。
- **該当コード**: [tt_scaffold_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/tt/tt_scaffold_test.go)
- **修正方針**:
  - (A) **テスト実行前に `unset GITHUB_TOKEN` を行う。** `integration_test.sh` に環境変数クリア処理を追加するか、テストヘルパー内で `os.Unsetenv("GITHUB_TOKEN")` を呼ぶ。
  - (B) scaffold テストの前提条件として、`GITHUB_TOKEN` が無効なダミー値でないことを検証するヘルパーを追加する。

### カテゴリ4: その他のテスト失敗 (tt)

`TestScaffoldCreateWithEnvOption` と `TestListCommand` についてはテスト実行ログに出力がありましたが、現在のソースコードに該当するテスト関数が存在しません。これは統合テストスクリプトの実行順の問題で前回の実行の残留か、あるいは別の worktree のテストが混入した可能性があります。最終実行のログからは実際のソースに存在するテストのみが実行されたことを確認済みです。

## 実現方針 (Implementation Approach)

### Phase 1: テストコードの修正（ソースに起因する問題）

1. **release-note テストのディレクトリパス修正**
   - [scanner_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/scanner_test.go) のテストパスを `branches/` 構造に更新
   - 動的なブランチ名の取得方法を検討（`git` コマンドや環境変数からの取得）

2. **release-note credential テストの改善**
   - [config_load_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/fix-memory-compiling/tests/release-note/config_load_test.go) の `TestConfigLoad_CredentialFileExists` を credential ファイル必須のエラーメッセージを明確にした上で維持

3. **scaffold テストの環境変数クリーンアップ**
   - テスト実行前に `GITHUB_TOKEN` をクリアする処理を追加（`unset GITHUB_TOKEN` または `os.Unsetenv`）
   - Antigravity IDE が注入するダミートークン (`github_pat_antigravitydummytoken`) が存在する場合に検知・クリアするロジック
   - `integration_test.sh` のテスト実行前処理として `unset GITHUB_TOKEN` を追加する案（テストコード修正不要）

### Phase 2: Docker 依存テストの環境整備

- Docker Desktop の起動を前提条件として文書化
- 必要に応じて Docker 依存テストのカテゴリ分離を検討

## 検証シナリオ (Verification Scenarios)

1. Phase 1 のテストコード修正後、`release-note` カテゴリの非Docker・非credential テストが全て成功する
2. `GITHUB_TOKEN` を unset した状態で scaffold テストが全て成功する
3. Docker Desktop を起動した状態で、Docker 依存テストが全て成功する

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```

2. release-note 統合テスト（ディレクトリ構造修正の確認）:
   ```
   scripts/process/integration_test.sh --categories "release-note"
   ```

3. tt 統合テスト（GITHUB_TOKEN unset後の scaffold テスト確認）:
   ```
   unset GITHUB_TOKEN && scripts/process/integration_test.sh --categories "tt" --specify "TestScaffold"
   ```

4. tt 統合テスト（Docker不要テストのみ、全体リグレッション確認）:
   ```
   scripts/process/integration_test.sh --categories "tt" --specify "TestTtCreate_|TestTtDelete_DryRun|TestTtDelete_ReservedBranch|TestTtDelete_PendingChanges|TestTtDoctor|TestTtEditor|TestTtOpen|TestTtListCode|TestTtClose|TestTtUp_RequiresFeature"
   ```

## 未解決事項 (Open Questions)

> [!IMPORTANT]
> **Docker 依存テストの分離方針**: 現在 Docker 依存テストと非依存テストが同じディレクトリ (`tests/tt/`) に混在しています。Docker 未起動環境でも CI を回せるように、build tags やカテゴリ分離を導入すべきでしょうか？

> [!NOTE]
> **GitHub 認証トークンの問題 (解決済み)**: Antigravity IDE がプロセス環境変数 `GITHUB_TOKEN` にダミー値 `github_pat_antigravitydummytoken` を注入していることが原因。テスト実行前に `unset GITHUB_TOKEN` することで解決する。永続的な修正としては `integration_test.sh` にクリア処理を追加する。

> [!IMPORTANT]
> **credential テストの扱い**: `TestConfigLoad_CredentialFileExists` は機密ファイルに依存しています。CI/CD 環境ではこのファイルは通常存在しません。このテストの失敗を許容するか、テスト環境にダミーの credential ファイルを配置する仕組みを作るか、方針を決める必要があります。
