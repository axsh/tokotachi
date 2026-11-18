# 001-GitLab-Docker-Live-Test

> **Source Specification**: [prompts/phases/000-foundation/branches/feat-adjust-gitlab/ideas/002-GitLab-Docker-Live-Test.md](file://prompts/phases/000-foundation/branches/feat-adjust-gitlab/ideas/002-GitLab-Docker-Live-Test.md)

## Goal Description

ローカルで GitLab CE を Docker 起動し、`tt doctor` / `tt list --update` / `tt pr` を実 `glab` 経由でスモークできる任意ライブ検証基盤を追加する。リポジトリには Compose（イメージピン留め）・起動／待機／シード／停止スクリプト・README・オプトイン統合テストのみを保存し、トークン実値・data volume・既定 CI／通常の全カテゴリ実行への組込は行わない。001（GitLab MR Support）の機能受け入れ（単体 Recorder＋doctor 統合）は変更しない。

## User Review Required

None.（ポート `8929`、イメージタグ、オプトイン発見除外、非対話 `glab` ラッパは本計画で確定する。）

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: Compose＋up/wait/down（`--help`）、既定統合から自動起動しない | Proposed Changes > `docker/gitlab/` / `scripts/dev/gitlab-live.sh` / `integration_test.sh` discover 除外 |
| R1: ポート・RAM を README に明記 | Proposed Changes > `docker/gitlab/README.md` |
| R2: seed（プロジェクト・トークン・初期 repo）、トークンを git 管理下に残さない | Proposed Changes > `gitlab-live.sh seed` / `.gitignore` / `.env.example` |
| R3.1: オプトイン時 `tt doctor` が `glab` を含む | Proposed Changes > `tests/live-gitlab/live_doctor_test.go` |
| R3.2: open MR で `tt list --update` が `pr` 相当 | Proposed Changes > `tests/live-gitlab/live_list_test.go` |
| R3.3: `tt pr` が MR 作成（非対話） | Proposed Changes > `gitlab-glab-ni.sh` + `live_pr_test.go` |
| R3: 未設定時スキップ／既定経路に載せない（`t.Skip` 禁止のためカテゴリ分離） | Proposed Changes > `tests/live-gitlab/.opt-in-only` + discover 除外 |
| R4: 保存する／しないの明確化 | Proposed Changes > README / `.gitignore` / artifacts 常時テスト |
| R5: 001 境界・forge 本番変更なし | forge パッケージは触らない。検証ハーネスのみ |
| 検証シナリオ 1–9 | Step-by-Step / Verification Plan / live-gitlab テスト |
| CI 手動トリガ（任意） | 本計画では実装しない（先送り。既定パイプライン非ブロックは R4 で担保） |

## Proposed Changes

### 常時検証（GitLab を起動しない）

#### [NEW] [tt_gitlab_docker_artifacts_test.go](file://tests/tt/tt_gitlab_docker_artifacts_test.go)

*   **Description**: ライブ基盤の成果物がリポジトリに存在し、スクリプトが `--help` で exit 0 すること（R1/R4 の常時証拠）。GitLab コンテナは起動しない。
*   **Technical Design**:
    ```go
    package integration_test

    func TestGitLabDockerArtifacts_Exist(t *testing.T) {
        // require files:
        //   docker/gitlab/docker-compose.yml
        //   docker/gitlab/README.md
        //   docker/gitlab/.env.example
        //   docker/gitlab/.gitignore  (or root .gitignore entries)
        //   scripts/dev/gitlab-live.sh
        //   scripts/dev/gitlab-glab-ni.sh
        //   tests/live-gitlab/.opt-in-only
    }

    func TestGitLabDockerScripts_Help(t *testing.T) {
        // run: bash scripts/dev/gitlab-live.sh --help  → exit 0, stdout/stderr に Usage
        // run: bash scripts/dev/gitlab-live.sh help    → exit 0
    }

    func TestGitLabLiveCategory_IsOptInOnly(t *testing.T) {
        // tests/live-gitlab/.opt-in-only が存在すること
    }
    ```
*   **Logic**:
    *   `projectRoot()` 基準の絶対パスで `os.Stat`
    *   Windows でも動かすためスクリプト実行は `bash` を試し、無ければ Git Bash パスまたは `t.Fatalf` で明示（`t.Skip` 禁止）
    *   Compose 内にピン留めイメージ文字列 `gitlab/gitlab-ce:` が含まれることをアサート

---

### Docker Compose と秘密情報の除外

#### [NEW] [docker/gitlab/docker-compose.yml](file://docker/gitlab/docker-compose.yml)

*   **Description**: GitLab CE の再現起動定義（イメージタグピン留め）。
*   **Technical Design**:
    ```yaml
    services:
      gitlab:
        image: gitlab/gitlab-ce:17.7.0-ce.0
        container_name: tokotachi-gitlab
        hostname: gitlab
        environment:
          GITLAB_OMNIBUS_CONFIG: |
            external_url 'http://localhost:8929'
            gitlab_rails['gitlab_shell_ssh_port'] = 2224
            nginx['listen_port'] = 80
            nginx['listen_https'] = false
        ports:
          - "8929:80"
          - "2224:22"
        volumes:
          - tokotachi_gitlab_config:/etc/gitlab
          - tokotachi_gitlab_logs:/var/log/gitlab
          - tokotachi_gitlab_data:/var/opt/gitlab
        shm_size: "256m"
        healthcheck:
          test: ["CMD-SHELL", "curl -fsS http://localhost/-/readiness || exit 1"]
          interval: 30s
          timeout: 10s
          retries: 30
          start_period: 300s
    volumes:
      tokotachi_gitlab_config:
      tokotachi_gitlab_logs:
      tokotachi_gitlab_data:
    ```
*   **Logic**:
    *   イメージは `gitlab/gitlab-ce:17.7.0-ce.0` に固定（更新時は本ファイルと README を同時更新）
    *   HTTP のみ（ホスト `8929`）。SSH は `2224`
    *   名前付き volume のみ使用し、リポジトリ配下に data ディレクトリを作らない
    *   `shm_size` は GitLab 推奨に合わせる

#### [NEW] [docker/gitlab/.env.example](file://docker/gitlab/.env.example)

*   **Description**: ローカル専用設定の雛形（実値はコミットしない）。
*   **Technical Design**:
    ```bash
    # Copy to docker/gitlab/.env (gitignored) and edit.
    GITLAB_ROOT_PASSWORD=  # optional; first boot may auto-generate into container logs
    GITLAB_HTTP_URL=http://localhost:8929
    GITLAB_PROJECT_PATH=root/tokotachi-live
    # Token is written by seed to tmp/gitlab-live/token (gitignored via tmp/)
    ```
*   **Logic**: 実 `.env` は seed/up が参照。example のみコミット。

#### [NEW] [docker/gitlab/.gitignore](file://docker/gitlab/.gitignore)

*   **Description**: 秘密情報とローカル生成物を除外。
*   **Technical Design**:
    ```
    .env
    *.token
    ```
*   **Logic**: volume は Docker 管理のためリポジトリに現れない。追加で root `.gitignore` に `docker/gitlab/.env` を明記してもよい（冗長可）。

#### [NEW] [docker/gitlab/README.md](file://docker/gitlab/README.md)

*   **Description**: 前提・手順・保存方針・トラブルシュートの正本。
*   **Logic（必須記載）**:
    *   前提: Docker Desktop（または同等）、目安 RAM **8GB+**（推奨 8–16GB）、ディスク数 GB、初回起動 **5–15 分**
    *   ポート: HTTP `8929`、SSH `2224`
    *   コマンド:
        ```bash
        ./scripts/dev/gitlab-live.sh up
        ./scripts/dev/gitlab-live.sh wait-ready
        ./scripts/dev/gitlab-live.sh seed
        # glab auth (see below)
        TT_LIVE_GITLAB=1 ./scripts/dev/gitlab-live.sh smoke
        ./scripts/dev/gitlab-live.sh down
        ./scripts/dev/gitlab-live.sh down --volumes   # wipe data
        ```
    *   `glab auth login --hostname localhost --api-host localhost:8929`（または token を `glab config` / `GITLAB_TOKEN` に設定）。HTTPS 無効のため `--insecure` 等が必要なら明記
    *   **保存する／しない**（仕様 R4 をそのまま表で再掲）
    *   root 初期パスワード: コンテナ内 `/etc/gitlab/initial_root_password`（24h 有効）の取得コマンド例
    *   001 との関係: 機能受け入れは単体＋doctor。本基盤は任意信頼度
    *   既定の `integration_test.sh`（全カテゴリ／`tt`）は本 GitLab を起動しない

#### [MODIFY] [.gitignore](file://.gitignore)

*   **Description**: ライブ検証の秘密・生成物を明示除外。
*   **Logic**: 追記:
    ```
    # GitLab live verification (local only)
    docker/gitlab/.env
    ```
    （`tmp/` は既存でカバー。`tmp/gitlab-live/` は追加コメント可）

---

### オーケストレーションスクリプト

#### [NEW] [scripts/dev/gitlab-live.sh](file://scripts/dev/gitlab-live.sh)

*   **Description**: up / wait-ready / seed / down / smoke の単一エントリ（`--help` 必須）。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    # source scripts/dist/shared/_lib.sh
    # COMPOSE_FILE=docker/gitlab/docker-compose.yml
    # TOKEN_FILE=tmp/gitlab-live/token
    # ENV_FILE=docker/gitlab/.env
    # DEFAULT_URL=http://localhost:8929
    # PROJECT_PATH=root/tokotachi-live
    # WAIT_TIMEOUT_SEC=900

    usage() { ... }  # up|wait-ready|seed|down|smoke|help

    cmd_up() {
      # docker compose -f "$COMPOSE_FILE" up -d
    }
    cmd_wait_ready() {
      # loop until curl -fsS "$URL/-/readiness" succeeds OR timeout → exit 1
      # print elapsed seconds
    }
    cmd_seed() {
      # 1. Ensure wait-ready already ok (or call it)
      # 2. Obtain root password:
      #    docker exec tokotachi-gitlab cat /etc/gitlab/initial_root_password
      #    OR from .env GITLAB_ROOT_PASSWORD if set
      # 3. OAuth token via API:
      #    POST /oauth/token  OR create PAT via API with root session
      #    Preferred: personal access token API after password grant / private_token login
      # 4. Create project root/tokotachi-live if missing (POST /api/v4/projects)
      # 5. Write token ONLY to tmp/gitlab-live/token (mkdir -p, chmod 600)
      # 6. Print: URL, project path, token file path (not the token value on success summary optional; printing once is ok but never write into docker/gitlab/.env committed paths)
      # 7. Optionally push an initial README via API so the project is non-empty
    }
    cmd_down() {
      # docker compose -f ... down
      # if --volumes: down -v
    }
    cmd_smoke() {
      # require TT_LIVE_GITLAB=1 else exit 1 with message (do not silently no-op)
      # require token file + wait-ready
      # require glab on PATH
      # export GITLAB_TOKEN from token file, TT_FORGE=gitlab, TT_CMD_GLAB=scripts/dev/gitlab-glab-ni.sh (abs path)
      # run: ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories live-gitlab --specify "LiveGitLab"
    }
    ```
*   **Logic**:
    *   `docker compose` を優先し、無ければ `docker-compose`
    *   seed は冪等（既存プロジェクト・既存トークンファイルがあっても再実行可。トークン再発行は `--rotate-token` 明示時のみ推奨）
    *   smoke は **必ず** 先に `build.sh` を実行してから `integration_test.sh`（planning-rules）
    *   スクリプト自体は GitLab を既定の `integration_test.sh` から呼ばない

#### [NEW] [scripts/dev/gitlab-glab-ni.sh](file://scripts/dev/gitlab-glab-ni.sh)

*   **Description**: `TT_CMD_GLAB` 用の非対話ラッパ。`tt pr` → `glab mr create` に `--yes` 等を足し、ライブで TTY 無しでも MR 作成できるようにする。forge 本体は変更しない。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    # Resolve real glab: if TT_CMD_GLAB_REAL set use it, else "glab"
    REAL="${TT_CMD_GLAB_REAL:-glab}"
    if [[ "${1:-}" == "mr" && "${2:-}" == "create" ]]; then
      shift 2
      # If caller already passed --yes/--fill, do not duplicate
      exec "$REAL" mr create --yes --fill "$@"
    fi
    exec "$REAL" "$@"
    ```
*   **Logic**:
    *   `mr list` 等はそのままパススルー
    *   ライブテストは `TT_CMD_GLAB=<repo>/scripts/dev/gitlab-glab-ni.sh` をセット
    *   無限再帰防止: ラッパは `glab` 実体を直接 exec（自分自身を `TT_CMD_GLAB` にしない）

---

### オプトイン統合テストカテゴリ

#### [MODIFY] [integration_test.sh](file://scripts/process/integration_test.sh)

*   **Description**: `.opt-in-only` マーカーがあるカテゴリを「全カテゴリ実行」から除外する。`--categories` で明示指定された場合は実行する。
*   **Technical Design**:
    ```bash
    discover_categories() {
      ...
      for dir in "$tests_dir"/*/; do
        ...
        # Skip opt-in-only categories unless explicitly requested via --categories
        if [[ -f "$dir/.opt-in-only" ]]; then
          continue
        fi
        ...
      done
    }
    ```
*   **Logic**:
    *   `get_target_categories` が `--categories live-gitlab` を受け取ったときは discover を使わずユーザー指定をそのまま使う既存分岐を維持 → 明示実行は可能
    *   ヘルプ文言に「`.opt-in-only` カテゴリは明示指定時のみ」と一行追記

#### [NEW] [tests/live-gitlab/.opt-in-only](file://tests/live-gitlab/.opt-in-only)

*   **Description**: 空ファイル（または一行説明）。discover 除外マーカー。
*   **Logic**: 内容は `# Opt-in only; requires ./scripts/dev/gitlab-live.sh` でよい。

#### [NEW] [tests/live-gitlab/go.mod](file://tests/live-gitlab/go.mod)

*   **Description**: `tests/tt` と同様の独立モジュール。
*   **Technical Design**:
    ```go
    module github.com/axsh/tokotachi/tests/live-gitlab

    go 1.24.0

    require github.com/stretchr/testify v1.11.1
    ```

#### [NEW] [tests/live-gitlab/helpers_test.go](file://tests/live-gitlab/helpers_test.go)

*   **Description**: ライブ前提のヘルパ。欠落時は **`t.Fatalf`（`t.Skip` 禁止）**。
*   **Technical Design**:
    ```go
    package integration_test

    const (
        envLiveGitLab = "TT_LIVE_GITLAB" // documented; smoke script sets it
        defaultHTTP   = "http://localhost:8929"
        projectPath   = "root/tokotachi-live"
    )

    func projectRoot() string { /* same pattern as tests/tt */ }
    func ttBinary(t *testing.T) string { /* bin/tt(.exe) or Fatalf */ }

    func requireLiveGitLab(t *testing.T) (baseURL, token, tokenFile string) {
        // 1. TT_LIVE_GITLAB must be "1" else Fatalf with hint to run gitlab-live.sh smoke
        // 2. token file tmp/gitlab-live/token must exist and be non-empty else Fatalf
        // 3. curl/HTTP GET baseURL/-/readiness must succeed else Fatalf
        // 4. glab must be on PATH (or TT_CMD_GLAB_REAL) else Fatalf
        // 5. return URL, token, path
    }

    func glabNIPath(t *testing.T) string {
        // abs path to scripts/dev/gitlab-glab-ni.sh
    }

    func runTTInDirEnv(...) (stdout, stderr string, code int) { /* copy from tt_forge_host_test */ }

    func setupLiveClone(t *testing.T, baseURL, token, projectPath string) (repoDir, branch string) {
        // git clone http://oauth2:<token>@localhost:8929/root/tokotachi-live.git
        // create feature branch live-gitlab-<pid>, commit empty/readme, push -u
        // return repoDir, branch
    }
    ```
*   **Logic**:
    *   前提不足はすべて Fatalf（スキップ禁止）
    *   clone URL にトークンを埋め込む場合、ログへ URL を出すときはトークンをマスク

#### [NEW] [tests/live-gitlab/live_doctor_test.go](file://tests/live-gitlab/live_doctor_test.go)

*   **Description**: R3.1 — 実 origin がローカル GitLab のとき doctor が `glab` を診断。
*   **Technical Design**:
    ```go
    func TestLiveGitLab_Doctor_ChecksGlab(t *testing.T) {
        baseURL, token, _ := requireLiveGitLab(t)
        dir := t.TempDir()
        // git init + remote add origin <baseURL>/<projectPath>.git
        // features/demo を作成（doctor が features を要求する場合）
        stdout, stderr, code := runTTInDirEnv(t, dir,
            []string{"TT_FORGE=gitlab", "GITLAB_TOKEN="+token, "TT_CMD_GLAB="+glabNIPath(t)},
            "doctor", "--json")
        require.Equal(t, 0, code, ...)
        // External Tools に "glab" が含まれる（tt_forge_host_test と同型の JSON パース）
    }
    ```
*   **Logic**: ホスト自動検出を使う場合 origin hostname が `localhost` で `gitlab` を含まない点に注意 → **必ず `TT_FORGE=gitlab`** を付与（仕様 R3「または TT_FORGE=gitlab」）。

#### [NEW] [tests/live-gitlab/live_list_test.go](file://tests/live-gitlab/live_list_test.go)

*   **Description**: R3.2 — open MR があるブランチで CodeStatus が `pr` 相当。
*   **Technical Design**:
    ```go
    func TestLiveGitLab_ListUpdate_DetectsOpenMR(t *testing.T) {
        baseURL, token, _ := requireLiveGitLab(t)
        repoDir, branch := setupLiveClone(t, baseURL, token, projectPath)
        // Create open MR via glab (real), non-interactive:
        //   TT_CMD_GLAB_REAL=glab, run glabNIPath: mr create --yes --fill
        //   OR API POST /merge_requests
        // Ensure tt worktree/feature layout if list requires it:
        //   minimal: features/<name>/ + worktree registration as existing list tests do
        // Prefer invoking codestatus path via:
        //   tt list --update --json  from a repo layout list understands
        // Assert JSON/table: CODE/status indicates pr (match existing list JSON schema)
        // Also assert glab mr list --source-branch returns iid > 0 as cross-check
    }
    ```
*   **Logic**:
    *   `tt list` がリポジトリルートの worktree 一覧前提なら、既存 `tests/tt` の list セットアップ手法を踏襲（必要なら最小 features＋branch worktree を作成）
    *   最低証拠: (1) API/`glab` で open MR が存在 (2) `tt list --update` 後の当該ブランチ status が `pr`（または JSON フィールド同等）
    *   MR はテスト終了時に close してよい（ベストエフォート）

#### [NEW] [tests/live-gitlab/live_pr_test.go](file://tests/live-gitlab/live_pr_test.go)

*   **Description**: R3.3 — `tt pr` がラッパ経由で MR を作成する。
*   **Technical Design**:
    ```go
    func TestLiveGitLab_PR_CreatesMR(t *testing.T) {
        baseURL, token, _ := requireLiveGitLab(t)
        repoDir, branch := setupLiveClone(t, ...) // branch WITHOUT existing MR
        // Prepare worktree layout expected by `tt pr` (same as production: worktree path)
        // env: TT_FORGE=gitlab, GITLAB_TOKEN=..., TT_CMD_GLAB=glabNIPath
        // run: tt pr <branch> <feature>  (match actual CLI: tt pr <branch> [feature])
        stdout, stderr, code := runTTInDirEnv(...)
        require.Equal(t, 0, code, ...)
        // Verify MR exists: glab mr list --source-branch <branch> --output json → iid >= 1
    }
    ```
*   **Logic**:
    *   forge は触らず、`TT_CMD_GLAB` ラッパで `--yes --fill` を付与
    *   `tt pr` の引数・cwd（worktree）は現行 `features/tt/cmd/pr.go` / `pkg/action.PR` に合わせる
    *   失敗時は stdout/stderr を Fatalf に含める

---

## Step-by-Step Implementation Guide

1. [x] **TDD（常時成果物テストを先に書く）**:
    *   Add `tests/tt/tt_gitlab_docker_artifacts_test.go`（当初は FAIL）。
2. [x] **Compose / ignore / README**:
    *   Add `docker/gitlab/docker-compose.yml`（image `gitlab/gitlab-ce:17.7.0-ce.0`）、`.env.example`、`.gitignore`、`README.md`。
    *   Modify root `.gitignore`。
3. [x] **スクリプト**:
    *   Add `scripts/dev/gitlab-live.sh`（up/wait-ready/seed/down/smoke + `--help`）。
    *   Add `scripts/dev/gitlab-glab-ni.sh`。
    *   `chmod +x`（Git 上で実行ビット）。
4. [x] **discover 除外**:
    *   Modify `scripts/process/integration_test.sh` の `discover_categories` で `.opt-in-only` をスキップ。
    *   Add `tests/live-gitlab/.opt-in-only`。
5. [x] **常時テストを GREEN に**:
    *   Re-run artifacts/help テストが通ることを確認（`build.sh` + `integration_test.sh --categories tt --specify "GitLabDocker"`）。
6. [x] **オプトイン live テスト（TDD）**:
    *   Add `tests/live-gitlab/go.mod`、`helpers_test.go`、`live_doctor_test.go`、`live_list_test.go`、`live_pr_test.go`。
    *   ローカルで `gitlab-live.sh up && wait-ready && seed` 後、`TT_LIVE_GITLAB=1 ./scripts/dev/gitlab-live.sh smoke` で RED→GREEN。
7. [x] **ドキュメント最終確認**:
    *   `docker/gitlab/README.md` に R4 表・ポート・RAM・コマンド・001 境界を揃える。
8. [x] **Verification Plan を実行**する（下記）。総合判定（§12）まで実施する。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **常時 tt 統合（GitLab CE 非起動・001 回帰含む）**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories tt --specify "Doctor|List|Forge|GitLab|Host|GitLabDocker"
    ```
    *   **Log Verification**:
        *   `TestGitLabDockerArtifacts_Exist` / `TestGitLabDockerScripts_Help` / `TestGitLabLiveCategory_IsOptInOnly` が PASS
        *   既存 `TestTtDoctor_*GitLab*` が PASS
        *   実行中に `tokotachi-gitlab` コンテナの新規 create が走っていないこと（Docker 未操作）

3. **全カテゴリ discover が live-gitlab を拾わないこと**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories tt --specify "GitLabDocker"
    ```
    *   併せて実装時に `discover_categories` 相当の動作確認: オプションなし実行は重いため、計画上は「`.opt-in-only` があること＋ discover ロジック変更のコードレビュー」と、明示カテゴリ実行（次項）の対比で証明する。
    *   Nightly 等でオプションなしを回す場合も `live-gitlab` が走らないこと。

4. **オプトイン・ライブ E2E（Docker GitLab 起動後のみ）**:
    ```bash
    ./scripts/dev/gitlab-live.sh up
    ./scripts/dev/gitlab-live.sh wait-ready
    ./scripts/dev/gitlab-live.sh seed
    TT_LIVE_GITLAB=1 ./scripts/dev/gitlab-live.sh smoke
    ```
    `smoke` 内部は次と等価であること:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories live-gitlab --specify "LiveGitLab"
    ```
    *   **Log Verification**:
        *   `TestLiveGitLab_Doctor_ChecksGlab` — JSON External Tools に `glab`
        *   `TestLiveGitLab_ListUpdate_DetectsOpenMR` — status `pr`（または同等 JSON）
        *   `TestLiveGitLab_PR_CreatesMR` — exit 0 かつ `glab mr list` で iid 確認
        *   `TT_LIVE_GITLAB` 未設定で smoke を呼ぶと **exit ≠ 0**（黙って成功しない）

5. **E2E Tests（新規）**:

    #### [NEW] [tests/tt/tt_gitlab_docker_artifacts_test.go](file://tests/tt/tt_gitlab_docker_artifacts_test.go)
    *   **テストケース**: `TestGitLabDockerArtifacts_Exist` / `TestGitLabDockerScripts_Help` / `TestGitLabLiveCategory_IsOptInOnly`
    *   **検証ポイント**: 成果物・help・opt-in マーカー（GitLab 非起動）

    #### [NEW] [tests/live-gitlab/live_doctor_test.go](file://tests/live-gitlab/live_doctor_test.go)
    *   **テストケース**: `TestLiveGitLab_Doctor_ChecksGlab`
    *   **検証ポイント**: 実機＋`TT_FORGE=gitlab` で doctor が `glab` を含む

    #### [NEW] [tests/live-gitlab/live_list_test.go](file://tests/live-gitlab/live_list_test.go)
    *   **テストケース**: `TestLiveGitLab_ListUpdate_DetectsOpenMR`
    *   **検証ポイント**: open MR → `tt list --update` が `pr` 相当

    #### [NEW] [tests/live-gitlab/live_pr_test.go](file://tests/live-gitlab/live_pr_test.go)
    *   **テストケース**: `TestLiveGitLab_PR_CreatesMR`
    *   **検証ポイント**: `tt pr`＋`gitlab-glab-ni.sh` で MR 作成

    **既定経路にライブ E2E を入れない理由**: 仕様 R3/R4/R5 および 001 方針。`t.Skip` 禁止のため、未準備時スキップではなく **別カテゴリ＋`.opt-in-only`** で既定実行から除外する。

### テスト項目設計（§11）

ボトムアップ順序:

1. **成果物・スクリプト help**（末端・常時）— Compose/README/scripts の存在
2. **discover 除外** — `.opt-in-only` により既定実行に乗らない
3. **seed／readiness** — `gitlab-live.sh wait-ready` / `seed`（smoke 前提）
4. **doctor ライブ** — `TT_FORGE=gitlab` + 実 token
5. **list ライブ** — 実 open MR → CodeStatus `pr`
6. **pr ライブ** — `tt pr` + 非対話ラッパ → MR 作成

観点チェックリスト:

| # | 観点 | 本計画での担保 |
|---|---|---|
| 1 | 正常系 | up→wait→seed→doctor/list/pr |
| 2 | 異常系 | smoke で `TT_LIVE_GITLAB` 未設定→失敗。wait timeout→非0。token 欠落→Fatalf |
| 3 | 外部連携 | 実 GitLab CE + 実 `glab` |
| 4 | データ一貫性 | seed の project/token ファイル、MR iid ↔ list status |
| 5 | 状態遷移 | branch push → MR open → status `pr` |
| 6 | 設定反映 | `TT_FORGE=gitlab`、`TT_CMD_GLAB` ラッパ、イメージピン |
| 7 | 副作用 | `.env`/token が git status に出ない、volume を repo に置かない、既定 tt が GitLab を起動しない |

セルフレビュー（§11.4）:

1. **網羅性**: R1–R5 と検証シナリオ 1–9 を常時成果物テスト＋オプトイン LiveGitLab 三本＋スクリプトでカバー。CI 手動ジョブは任意要件のため先送り（既定非組込は満たす）。
2. **証拠の十分性**: doctor JSON のツール名、MR iid、list の `pr`、help exit 0、イメージ文字列ピンをアサート。
3. **迂回排除**: `TT_FORGE=gitlab` 明示、`TT_CMD_GLAB` が `gitlab-glab-ni.sh` であること、MR 作成後に `glab mr list` で実在確認（Recorder のみに頼らない）。
4. **依存関係**: 成果物 → discover 除外 → seed/ready → doctor → list → pr の順。上位成功だけでは下位未検証としない。

### 総合判定プロセス（§12）

全テスト成功後、実装完了報告に以下を記録する:

1. スキップ有無（`t.Skip` がコードに無いこと、smoke 未設定が誤って成功していないこと）
2. ログの ERROR/panic
3. doctor/list/pr が GitHub/`gh` 経路に落ちていないこと（`TT_FORGE` / ツール名 / MR API）
4. イメージタグがピン留めのままか
5. `git status` に `docker/gitlab/.env` や token が現れていないか
6. `--categories tt` 単独で `LiveGitLab` が実行されていないか
7. 外部 GitLab コンテナが smoke 前に healthy だったか

判定フォーマットは testing-rules §12.3 に従う。

## Documentation

`prompts/specifications` は存在しないため更新対象外。本計画のユーザー向け正本は次:

#### [NEW] [docker/gitlab/README.md](file://docker/gitlab/README.md)

*   **更新内容**: 前提（RAM/ポート/起動時間）、コマンド、認証、保存方針、001 との境界、トラブルシュート

#### [MODIFY] [README.md](file://README.md)（任意・最小）

*   **更新内容**: 開発者向けに「GitLab ライブ検証は `docker/gitlab/README.md` を参照（任意）」を一行追加してもよい。必須ではない（詳細は `docker/gitlab/README.md`）。
