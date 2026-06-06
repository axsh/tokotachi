# 009-DevTools-Platform-Refactoring

> **Source Specification**: [008-DevTools-Platform-Refactoring.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/008-DevTools-Platform-Refactoring.md)

## Goal Description

`scripts/tools/` を `scripts/dist/` にリネームし、不要な `compatibility-matrix.yaml` を削除し、リリース手順 README を作成する。影響を受ける全ドキュメントの参照も更新する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req.1: `scripts/tools/` → `scripts/dist/` リネーム | Step 1: スクリプト移動＋メッセージ修正 |
| Req.1: 旧ディレクトリ削除 | Step 1: `scripts/tools/` 削除 |
| Req.2: `compatibility-matrix.yaml` 削除 | Step 2: ファイル削除 |
| Req.3: 007仕様書の参照更新 (`scripts/tools` + `compatibility-matrix`) | Step 3: ドキュメント更新 |
| Req.3: 008実装計画書の参照更新 (`scripts/tools` + `compatibility-matrix`) | Step 3: ドキュメント更新 |
| Req.3: `docs/release-process.md` の参照更新 | Step 3: ドキュメント更新 |
| Req.4: `scripts/dist/README.md` 作成 | Step 4: README作成 |

---

## Proposed Changes

### scripts/dist — スクリプト移動

#### [MODIFY] [dev](file:///c:/Users/yamya/myprog/tokotachi/scripts/tools/dev) → `scripts/dist/dev`
*   **Description**: ファイルを `scripts/dist/dev` に移動し、エコーメッセージを修正
*   **Logic**:
    *   `echo "[scripts/tools/dev]` → `echo "[scripts/dist/dev]`

#### [MODIFY] [build](file:///c:/Users/yamya/myprog/tokotachi/scripts/tools/build) → `scripts/dist/build`
*   **Description**: ファイルを `scripts/dist/build` に移動し、エコーメッセージを修正
*   **Logic**:
    *   `echo "[scripts/tools/build]` → `echo "[scripts/dist/build]`

#### [MODIFY] [release](file:///c:/Users/yamya/myprog/tokotachi/scripts/tools/release) → `scripts/dist/release`
*   **Description**: ファイルを `scripts/dist/release` に移動し、エコーメッセージを修正
*   **Logic**:
    *   `echo "[scripts/tools/release]` → `echo "[scripts/dist/release]`

#### [MODIFY] [publish](file:///c:/Users/yamya/myprog/tokotachi/scripts/tools/publish) → `scripts/dist/publish`
*   **Description**: ファイルを `scripts/dist/publish` に移動し、エコーメッセージを修正
*   **Logic**:
    *   `echo "[scripts/tools/publish]` → `echo "[scripts/dist/publish]`

#### [MODIFY] [install-tools](file:///c:/Users/yamya/myprog/tokotachi/scripts/tools/install-tools) → `scripts/dist/install-tools`
*   **Description**: ファイルを `scripts/dist/install-tools` に移動し、エコーメッセージを修正
*   **Logic**:
    *   `echo "[scripts/tools/install-tools]` → `echo "[scripts/dist/install-tools]`

#### [MODIFY] [bootstrap-tools](file:///c:/Users/yamya/myprog/tokotachi/scripts/tools/bootstrap-tools) → `scripts/dist/bootstrap-tools`
*   **Description**: ファイルを `scripts/dist/bootstrap-tools` に移動し、エコーメッセージを修正
*   **Logic**:
    *   `echo "[scripts/tools/bootstrap-tools]` → `echo "[scripts/dist/bootstrap-tools]`

#### [DELETE] `scripts/tools/` ディレクトリ
*   **Description**: 移動完了後に旧ディレクトリを削除

---

### tools/metadata — ファイル削除

#### [DELETE] [compatibility-matrix.yaml](file:///c:/Users/yamya/myprog/tokotachi/tools/metadata/compatibility-matrix.yaml)
*   **Description**: 不要なOS×エディタ互換性マトリクスファイルを削除

---

### scripts/dist — README作成

#### [NEW] [README.md](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/README.md)
*   **Description**: アプリケーションのリリース手順ドキュメントを作成
*   **Technical Design**:
    ```markdown
    # Distribution Scripts

    CLI tools distribution pipeline scripts.

    ## Overview

    This directory contains scripts for building, releasing, and publishing
    CLI tools defined in `features/`. These scripts form the distribution
    pipeline that produces cross-platform binaries and publishes them to
    package managers.

    ## Prerequisites

    - Go 1.21+
    - GoReleaser (optional, for automated releases)

    ## Scripts

    | Script | Description | Usage |
    |--------|-------------|-------|
    | `build` | Build CLI tools from features | `./scripts/dist/build <tool-id>` |
    | `release` | Create release artifacts | `./scripts/dist/release <tool-id> <version>` |
    | `publish` | Publish to GitHub Releases / Homebrew / Scoop | `./scripts/dist/publish <tool-id> <version>` |
    | `dev` | Launch development environments | `./scripts/dist/dev <feature-name>` |
    | `install-tools` | Install developer tools locally | `./scripts/dist/install-tools [--all \| <tool-id>...]` |
    | `bootstrap-tools` | Initial setup for new developers | `./scripts/dist/bootstrap-tools` |

    ## Release Workflow

    ### 1. Build

    Build a CLI tool from its feature source:

    ```bash
    ./scripts/dist/build devctl
    ```

    This reads `tools/manifests/devctl.yaml` to determine build targets,
    then compiles the Go binary for all specified platforms.

    ### 2. Release

    Create release artifacts (archives + checksums):

    ```bash
    ./scripts/dist/release devctl v1.0.0
    ```

    Artifacts are written to `dist/devctl/v1.0.0/`.

    ### 3. Publish

    Publish the release to distribution channels:

    ```bash
    ./scripts/dist/publish devctl v1.0.0
    ```

    This publishes to:
    - GitHub Releases
    - Homebrew tap (from `tools/installers/homebrew/`)
    - Scoop bucket (from `tools/installers/scoop/`)

    ## Artifact Flow

    ```
    features/
         ↓
    tools/manifests/
         ↓
    scripts/dist/build
         ↓
    dist/
         ↓
    scripts/dist/release
         ↓
    packaging/
         ↓
    scripts/dist/publish
         ↓
    Homebrew / Scoop / GitHub Releases
    ```

    ## Related Files

    - `tools/manifests/` — Tool distribution metadata
    - `packaging/` — Build packaging configuration (GoReleaser, archives, checksums)
    - `releases/` — Release history and channel definitions
    - `dist/` — Build artifacts (gitignored)
    ```

---

## Documentation

#### [MODIFY] [007-DevTools-Platform-Structure.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/007-DevTools-Platform-Structure.md)
*   **更新内容**:
    1. すべての `scripts/tools` を `scripts/dist` に置換
    2. `compatibility-matrix.yaml` に関する記述を削除:
        *   L126: `compatibility-matrix.yaml` をディレクトリツリーから削除
        *   L171-189: `compatibility-matrix.yaml` 例のブロック全体を削除

#### [MODIFY] [008-DevTools-Platform-Structure.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/plans/main/008-DevTools-Platform-Structure.md)
*   **更新内容**:
    1. すべての `scripts/tools` を `scripts/dist` に置換
    2. `compatibility-matrix.yaml` に関する記述を削除:
        *   L251-270: `compatibility-matrix.yaml` の `[NEW]` セクション全体を削除
        *   L586: Step 2の `compatibility-matrix.yaml` 作成行を削除
        *   L660: Verification Planの `compatibility-matrix.yaml` チェック行を削除

#### [MODIFY] [release-process.md](file:///c:/Users/yamya/myprog/tokotachi/docs/release-process.md)
*   **更新内容**: すべての `scripts/tools` を `scripts/dist` に置換

---

## Step-by-Step Implementation Guide

### Step 1: スクリプトの移動とメッセージ修正

1. `mkdir -p scripts/dist` でディレクトリ作成
2. 6つのスクリプトを `scripts/tools/` から `scripts/dist/` へコピー
3. 各スクリプトのエコーメッセージ内の `scripts/tools/` を `scripts/dist/` に修正
4. `chmod +x scripts/dist/*` で実行権限付与
5. `rm -rf scripts/tools/` で旧ディレクトリを削除

### Step 2: `compatibility-matrix.yaml` の削除

1. `rm tools/metadata/compatibility-matrix.yaml`

### Step 3: ドキュメントの参照更新

1. `007-DevTools-Platform-Structure.md`: `scripts/tools` → `scripts/dist` 一括置換 + `compatibility-matrix` 関連記述削除
2. `008-DevTools-Platform-Structure.md`: `scripts/tools` → `scripts/dist` 一括置換 + `compatibility-matrix` 関連記述削除
3. `docs/release-process.md`: `scripts/tools` → `scripts/dist` 一括置換

### Step 4: `scripts/dist/README.md` の作成

1. 上記 Technical Design のとおりに README.md を作成

### Step 5: 検証

1. 構造検証（ファイル存在・非存在チェック）
2. 参照残留チェック（grep）
3. ビルドテスト

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **構造検証**:
    ```bash
    # scripts/dist/ に全ファイルが存在すること
    for file in dev build release publish install-tools bootstrap-tools README.md; do
      [ -f "scripts/dist/$file" ] && echo "OK: scripts/dist/$file" || echo "FAIL: scripts/dist/$file"
    done

    # 旧ディレクトリが存在しないこと
    [ ! -d "scripts/tools" ] && echo "OK: scripts/tools/ removed" || echo "FAIL: scripts/tools/ still exists"

    # compatibility-matrix.yaml が存在しないこと
    [ ! -f "tools/metadata/compatibility-matrix.yaml" ] && echo "OK: compatibility-matrix.yaml removed" || echo "FAIL: still exists"
    ```

3.  **参照残留チェック**:
    ```bash
    # scripts/dist/ 内にscripts/toolsの参照が残っていないこと
    result=$(grep -r "scripts/tools" scripts/dist/ 2>/dev/null)
    [ -z "$result" ] && echo "OK: No old references in scripts/dist/" || echo "FAIL: $result"

    # ドキュメント内にscripts/toolsの参照が残っていないこと
    result=$(grep -r "scripts/tools" docs/ prompts/phases/000-foundation/ 2>/dev/null)
    [ -z "$result" ] && echo "OK: No old references in docs" || echo "FAIL: $result"
    ```
