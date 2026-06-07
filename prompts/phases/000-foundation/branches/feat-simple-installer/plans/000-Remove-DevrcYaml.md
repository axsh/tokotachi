# 000-Remove-DevrcYaml

> **Source Specification**: [000-Remove-DevrcYaml.md](file://prompts/phases/000-foundation/ideas/feat-simple-installer/000-Remove-DevrcYaml.md)

## Goal Description

`.devrc.yaml` のグローバル設定ファイル機能を完全に削除する。`GlobalConfig` 構造体、`LoadGlobalConfig` 関数、`tt doctor` のチェック/修正機能、および全テストから `.devrc.yaml` への参照を除去し、デフォルト値をハードコードに置換する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `GlobalConfig` 構造体と `LoadGlobalConfig` を削除 | Proposed Changes > pkg/resolve/config.go |
| デフォルト値をハードコードに置き換え | Proposed Changes > features/tt/cmd/common.go, tokotachi.go |
| `tt doctor` の `.devrc.yaml` チェック機能を削除 | Proposed Changes > pkg/doctor/checks.go, doctor.go |
| READMEの Configuration セクションを削除 | Proposed Changes > README.md |
| 関連テストコードの削除・修正 | Proposed Changes > 各 `_test.go` |
| ビルドとテストが通ること | Verification Plan |

## Proposed Changes

### pkg/resolve (設定読み込み)

#### [MODIFY] [config_test.go](file://pkg/resolve/config_test.go)
*   **Description**: `LoadGlobalConfig` 関連テスト2件を削除
*   **Technical Design**:
    *   `TestLoadGlobalConfig_Defaults` (L13-20) — 削除
    *   `TestLoadGlobalConfig_FromFile` (L22-38) — 削除
    *   `TestLoadFeatureConfig_*` テスト2件は残す
    *   import の `"os"`, `"path/filepath"` が不要になる場合は削除

---

#### [MODIFY] [config.go](file://pkg/resolve/config.go)
*   **Description**: `GlobalConfig` 構造体と `LoadGlobalConfig` 関数を削除
*   **Technical Design**:
    *   `GlobalConfig` 構造体 (L12-17) — 削除
    *   `LoadGlobalConfig` 関数 (L28-56) — 削除
    *   `FeatureConfig` 構造体と `LoadFeatureConfig` 関数はそのまま残す
    *   使われなくなった import (`"errors"`) を削除

---

### pkg/doctor (ヘルスチェック)

#### [MODIFY] [checks_test.go](file://pkg/doctor/checks_test.go)
*   **Description**: `TestCheckGlobalConfig` テスト関数全体を削除
*   **Technical Design**:
    *   `TestCheckGlobalConfig` (L126-213) — 全サブテスト含め削除
    *   他のテスト (`TestCheckExternalTools_*`, `TestCheckRepoStructure`, `TestCheckFeature`, `TestDiscoverFeatures`) は残す

---

#### [MODIFY] [checks.go](file://pkg/doctor/checks.go)
*   **Description**: `.devrc.yaml` 関連コードを削除
*   **Technical Design**:
    *   `categoryConfig` 定数 (L18) — 削除
    *   `globalConfig` 構造体 (L140-144) — 削除
    *   `checkGlobalConfig` 関数 (L146-263) — 削除
    *   `fixGlobalConfig` 関数 (L318-323) — 削除
    *   `validEditors`, `validContainerModes` (L134-137) — `checkGlobalConfig` 内でのみ使用されているため削除
    *   使われなくなった import (`"slices"`, `"gopkg.in/yaml.v3"`, `"errors"`) を整理

---

#### [MODIFY] [doctor_test.go](file://pkg/doctor/doctor_test.go)
*   **Description**: `.devrc.yaml` の作成・検証・fix テストを修正
*   **Technical Design**:
    *   `TestRun_AllPass` (L13-42):
        *   L20-21: `.devrc.yaml` ファイル作成行を削除
        *   テストは `.devrc.yaml` なしでオール PASS となることを検証
    *   `TestRun_WithFailure` (L44-69):
        *   `.devrc.yaml` のパースエラーによる FAIL を検証するテスト → 別の FAIL 条件に変更、または テスト自体を `.devrc.yaml` に依存しない形にリファクタリング
        *   L51-52: `.devrc.yaml` 作成行を削除
        *   L53: コメント削除
        *   FAIL がなくなるので `assert.True(t, report.HasFailures())` → 別の失敗条件がなければテスト自体削除
    *   `TestRun_FeatureFilter` (L71-114):
        *   L79-80: `.devrc.yaml` 作成行を削除
    *   `TestRun_FeatureFilterNotFound` (L116-138):
        *   L121-122: `.devrc.yaml` 作成行を削除
    *   `TestRun_WithFix` (L140-182):
        *   L143, L146: コメント修正（`.devrc.yaml` 言及を削除）
        *   L166-168: `.devrc.yaml` 作成確認を削除
        *   L181: fixedCount の比較を `>= 2` から `>= 1` に変更（`work/` のみ）
        *   fix 対象の `.devrc.yaml` が存在しないことを確認するアサーション追加は不要（チェック自体が削除されるため）
    *   `TestFixGlobalConfig` (L184-195) — テスト全体を削除
    *   `TestFixDirectory` (L197-206) — 残す

---

#### [MODIFY] [doctor.go](file://pkg/doctor/doctor.go)
*   **Description**: `checkGlobalConfig` 呼び出しと fix ロジックを削除
*   **Technical Design**:
    *   L28-29: `// 3. Global config` セクションとその `checkGlobalConfig` 呼び出しを削除
    *   `applyFixes` 関数内:
        *   L72-76: `.devrc.yaml` fix 分岐 (`case res.Category == categoryConfig ...`) を削除

---

#### [MODIFY] [result_test.go](file://pkg/doctor/result_test.go)
*   **Description**: テストデータ内の `.devrc.yaml` 参照を置換
*   **Technical Design**:
    *   `TestReport_PrintText` (L109): `.devrc.yaml` → 別のテストデータ名に変更（例: `"config.yaml"`）, Category `"Config"` → `"Repo"` など
    *   `TestReport_PrintText_Fixed` (L178): `.devrc.yaml` → 別名に変更
    *   `TestReport_PrintJSON_Fixed` (L210): `.devrc.yaml` → 別名に変更

---

### features/tt/cmd (CLI エントリポイント)

#### [MODIFY] [common.go](file://features/tt/cmd/common.go)
*   **Description**: `ResolveEnvironment` から `LoadGlobalConfig` を除去し、デフォルト値をハードコード
*   **Technical Design**:
    *   `ResolveEnvironment` メソッド (L112-142):
        *   L117-120: `LoadGlobalConfig` 呼び出しとエラー処理を削除
        *   L130: `globalCfg.DefaultEditor` → リテラル `"cursor"`
        *   L138: `globalCfg.DefaultContainerMode` → リテラル `"docker-local"`
        *   `import` から `"github.com/axsh/tokotachi/pkg/resolve"` を削除（`resolve` が不要になる）
    *   **修正後の `ResolveEnvironment` のロジック**:
        ```go
        func (ctx *AppContext) ResolveEnvironment(editorFlag string) (detect.OS, detect.Editor, matrix.ContainerMode, error) {
            currentOS := detect.CurrentOS()
            // ...
            featureCfg, err := resolve.LoadFeatureConfig(ctx.RepoRoot, ctx.Feature)
            // ...
            ed, err := detect.ResolveEditor(
                editorFlag,
                os.Getenv(detect.EnvKeyEditor),
                featureCfg.Dev.EditorDefault,
                "cursor",  // ← ハードコード
            )
            // ...
            containerMode := matrix.ContainerMode("docker-local")  // ← ハードコード
            // ...
        }
        ```
    *   注意: `resolve` パッケージの import は `LoadFeatureConfig` のため残る

---

### tokotachi.go (公開API)

#### [MODIFY] [tokotachi.go](file://tokotachi.go)
*   **Description**: `resolveProjectName` を簡略化し、`LoadGlobalConfig` 呼び出しを除去
*   **Technical Design**:
    *   `resolveProjectName` メソッド (L83-90):
        ```go
        func (c *Client) resolveProjectName() string {
            return "tt"
        }
        ```
    *   `Up` メソッド内 (L166-167):
        *   `LoadGlobalConfig` 呼び出しを削除
        *   `containerMode` をハードコード: `containerMode := matrix.ContainerMode("docker-local")`
    *   `Open` メソッド内 (L422-423):
        *   同様に `LoadGlobalConfig` 呼び出しを削除してハードコード
    *   import: `"github.com/axsh/tokotachi/pkg/resolve"` は他の `resolve.*` 関数（`ContainerName`, `Worktree` 等）で使われるため残す

---

### pkg/detect (エディタ検出)

#### [MODIFY] [editor.go](file://pkg/detect/editor.go)
*   **Description**: コメントの `.devrc.yaml` 言及を修正
*   **Technical Design**:
    *   L39: `//  4. Global config (globalConfig, from .devrc.yaml)` → `//  4. Default value ("cursor")` に変更

---

### pkg/scaffold (テンプレート)

#### [MODIFY] [catalog_test.go](file://pkg/scaffold/catalog_test.go)
*   **Description**: テストデータの `.devrc.yaml` 参照を別名に変更
*   **Technical Design**:
    *   L237: `Files: []string{".devrc.yaml"}` → `Files: []string{"nonexistent.yaml"}`
    *   L242: `assert.Contains(t, err.Error(), ".devrc.yaml")` → `assert.Contains(t, err.Error(), "nonexistent.yaml")`

---

### README.md (ドキュメント)

#### [MODIFY] [README.md](file://README.md)
*   **Description**: Configuration セクションの `.devrc.yaml` 部分を削除
*   **Technical Design**:
    *   `### Configuration` 見出しと `#### Project-level (.devrc.yaml)` サブセクション + コードブロックを削除
    *   `#### Feature-level (feature.yaml)` は残すが、見出しレベルを `### Configuration` 直下に調整

## Step-by-Step Implementation Guide

1.  **テスト削除 (pkg/resolve)**:
    *   `pkg/resolve/config_test.go` から `TestLoadGlobalConfig_Defaults` と `TestLoadGlobalConfig_FromFile` を削除

2.  **本体コード削除 (pkg/resolve)**:
    *   `pkg/resolve/config.go` から `GlobalConfig` 構造体と `LoadGlobalConfig` 関数を削除
    *   不要な import を整理

3.  **テスト修正 (pkg/doctor)**:
    *   `pkg/doctor/checks_test.go` から `TestCheckGlobalConfig` を削除
    *   `pkg/doctor/doctor_test.go` の各テストから `.devrc.yaml` 作成行を削除、`TestFixGlobalConfig` を削除、`TestRun_WithFix` のアサーション修正
    *   `pkg/doctor/result_test.go` のテストデータ名を変更

4.  **本体コード削除 (pkg/doctor)**:
    *   `pkg/doctor/checks.go` から `categoryConfig`, `globalConfig`, `checkGlobalConfig`, `fixGlobalConfig`, `validEditors`, `validContainerModes` を削除
    *   `pkg/doctor/doctor.go` から `checkGlobalConfig` 呼び出しと `.devrc.yaml` fix 分岐を削除

5.  **呼び出し元修正 (features/tt/cmd)**:
    *   `features/tt/cmd/common.go` の `ResolveEnvironment` をデフォルト値ハードコードに修正

6.  **呼び出し元修正 (tokotachi.go)**:
    *   `resolveProjectName` を `return "tt"` に簡略化
    *   `Up`, `Open` 内の `LoadGlobalConfig` 呼び出しをデフォルト値に置換

7.  **コメント修正**:
    *   `pkg/detect/editor.go` L39 のコメント修正
    *   `pkg/scaffold/catalog_test.go` のテストデータ修正

8.  **README修正**:
    *   `README.md` の Configuration セクションから `.devrc.yaml` 部分を削除

9.  **ビルド・テスト実行**:
    *   `./scripts/process/build.sh` でビルド + 単体テスト
    *   `grep -r "devrc" --include="*.go" .` で残留参照の確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **残留参照チェック**:
    ```bash
    grep -r "devrc" --include="*.go" .
    ```
    *   出力が0件であること

## Documentation

#### [MODIFY] [README.md](file://README.md)
*   **更新内容**: `### Configuration` セクション内の `#### Project-level (.devrc.yaml)` サブセクションを削除
