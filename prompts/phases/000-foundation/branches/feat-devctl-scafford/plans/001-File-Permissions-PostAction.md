# 001-File-Permissions-PostAction

> **Source Specification**: [001-File-Permissions-PostAction.md](file://prompts/phases/000-foundation/ideas/feat-devctl-scafford/001-File-Permissions-PostAction.md)

## Goal Description

`post_actions` に `file_permissions` を追加し、scaffold 適用後にファイルパーミッション（実行権限やシークレットファイル用の制限パーミッション）を宣言的に設定できるようにする。`executable: true`（`mode: "0755"` の糖衣構文）と `mode` による明示的なパーミッション指定の両方をサポートする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| FP-R1: placement.yaml での file_permissions 定義 | `placement.go` (FilePermission 構造体 + PostActions 拡張 + バリデーション) |
| FP-R2: グロブパターンによるファイルマッチング | `applier.go` (applyFilePermissions 関数内の doublestar マッチング) |
| FP-R3: パーミッションの適用 | `applier.go` (applyFilePermissions: os.Chmod + git update-index) |
| FP-R4: 実行計画への表示 | `plan.go` (PrintPlan に file_permissions 表示を追加) + `applier.go` (BuildPlan に PermissionActions 追加) |
| FP-R5: チェックポイント情報への記録 | `checkpoint.go` (CheckpointInfo に PermissionsApplied 追加) |
| FP-R6: 冪等性 | `applier.go` (os.Chmod は冪等、エラーをハンドリング) |

## Proposed Changes

### scaffold パッケージ (internal/scaffold)

---

#### [MODIFY] [placement_test.go](file://features/devctl/internal/scaffold/placement_test.go)

*   **Description**: `FilePermissions` のパース・バリデーションテストを追加。
*   **Technical Design**:
    ```go
    // 追加テストケース:
    // - "valid placement with file_permissions (executable)":
    //     YAML に executable: true を含む → 正しくパースされる
    // - "valid placement with file_permissions (mode)":
    //     YAML に mode: "0600" を含む → 正しくパースされる
    // - "file_permissions with both executable and mode (mode takes precedence)":
    //     YAML に executable: true + mode: "0700" → mode が優先、ResolvedMode() == 0o700
    // - "file_permissions with neither executable nor mode":
    //     YAML に pattern のみ → バリデーションエラー
    // - "file_permissions with invalid mode":
    //     mode: "abc" → バリデーションエラー
    // - "file_permissions with out-of-range mode":
    //     mode: "9999" → バリデーションエラー
    ```

#### [MODIFY] [placement.go](file://features/devctl/internal/scaffold/placement.go)

*   **Description**: `PostActions` に `FilePermissions` フィールドを追加。`FilePermission` 構造体と `ResolvedMode()` メソッド、バリデーションロジックを追加。
*   **Technical Design**:
    ```go
    // FilePermission defines a file permission rule.
    type FilePermission struct {
        Pattern    string `yaml:"pattern"`
        Executable *bool  `yaml:"executable,omitempty"` // ポインタで未指定検出
        Mode       string `yaml:"mode,omitempty"`       // 8進数文字列 e.g. "0755"
    }

    // ResolvedMode returns the resolved os.FileMode for this permission rule.
    // If Mode is set, parse and return it. If Executable is true, return 0o755.
    func (fp FilePermission) ResolvedMode() (os.FileMode, error)
    // ロジック:
    //   1. fp.Mode != "" → strconv.ParseUint(fp.Mode, 8, 32) で変換
    //   2. fp.Executable != nil && *fp.Executable → 0o755
    //   3. それ以外 → エラー

    // IsExecutable returns true if the resolved mode has any execute bit set.
    func (fp FilePermission) IsExecutable() bool
    // ロジック: mode & 0o111 != 0

    // PostActions を拡張:
    type PostActions struct {
        GitignoreEntries []string         `yaml:"gitignore_entries"`
        FilePermissions  []FilePermission `yaml:"file_permissions"`
    }
    ```
*   **Logic**:
    - `ParsePlacement` 内に `validateFilePermissions(p.PostActions.FilePermissions)` を追加。
    - バリデーション: 各エントリについて、`Pattern` が空 → エラー、`Mode` も `Executable` も未指定 → エラー、`Mode` が不正な 8 進数 → エラー。

---

#### [MODIFY] [applier_test.go](file://features/devctl/internal/scaffold/applier_test.go)

*   **Description**: `applyFilePermissions` のテストケースを追加。
*   **Technical Design**:
    ```go
    // 追加テスト:
    // - TestApplyFilePermissions_Executable:
    //     tmpDir に scripts/setup.sh を作成
    //     FilePermissions: [{Pattern: "scripts/**/*.sh", Executable: boolPtr(true)}]
    //     → os.Stat で 0o755 確認
    //
    // - TestApplyFilePermissions_Mode0600:
    //     tmpDir に secrets/key.txt を作成
    //     FilePermissions: [{Pattern: "secrets/**/*", Mode: "0600"}]
    //     → os.Stat で 0o600 確認
    //
    // - TestApplyFilePermissions_NoMatch:
    //     FilePermissions: [{Pattern: "*.py", Executable: boolPtr(true)}]
    //     但し .py ファイルなし → エラーなし
    //
    // - TestApplyFilePermissions_Idempotent:
    //     2回呼んでもエラーなし
    //
    // - TestApplyPostActions_WithFilePermissions:
    //     GitignoreEntries + FilePermissions の両方を含む PostActions → 両方実行
    //
    // - TestBuildPlan_WithPermissionActions:
    //     FilePermissions 付きの BuildPlan → Plan.PermissionActions に反映
    ```

#### [MODIFY] [applier.go](file://features/devctl/internal/scaffold/applier.go)

*   **Description**: `applyFilePermissions` 関数を追加、`ApplyPostActions` と `BuildPlan` を拡張。`Plan` に `PermissionActions` フィールドを追加。
*   **Technical Design**:
    ```go
    import "github.com/bmatcuk/doublestar/v4"

    // PermissionAction describes a planned permission change.
    type PermissionAction struct {
        Path string
        Mode string // 表示用: "0755", "0600" 等
    }

    // Plan に追加:
    type Plan struct {
        // ... 既存フィールド ...
        PermissionActions []PermissionAction // file_permissions による変更
    }

    // applyFilePermissions applies file permission settings based on glob patterns.
    func applyFilePermissions(perms []FilePermission, repoRoot, baseDir string) error
    // ロジック:
    //   1. 各 FilePermission について:
    //      a. ResolvedMode() で os.FileMode を取得
    //      b. fullBaseDir = filepath.Join(repoRoot, baseDir)
    //      c. filepath.WalkDir(fullBaseDir, ...) で全ファイルを走査
    //      d. 各ファイルの相対パス (baseDir からの) を doublestar.Match(pattern, relPath) で判定
    //      e. マッチしたら os.Chmod(fullPath, mode)
    //   2. マッチなしでもエラーにしない
    //   3. Windows 環境では追加で git update-index --chmod=+x/-x (runtime.GOOS == "windows")

    // ApplyPostActions を拡張:
    func ApplyPostActions(actions PostActions, repoRoot string, baseDir string) error
    // 既存の gitignore 処理の後に:
    //   applyFilePermissions(actions.FilePermissions, repoRoot, baseDir) を呼ぶ
    ```
*   **Logic**:
    - `ApplyPostActions` のシグネチャに `baseDir string` を追加（パターンマッチングのルート特定用）。
    - `BuildPlan` でも `PermissionActions` をプレビュー生成する（実ファイルに対するグロブマッチング）。

> [!IMPORTANT]
> `ApplyPostActions` のシグネチャ変更は `scaffold.go` の `Apply` 関数の呼び出し箇所にも影響する。

---

#### [MODIFY] [plan_test.go](file://features/devctl/internal/scaffold/plan_test.go)

*   **Description**: `PrintPlan` に `PermissionActions` の表示テストを追加。
*   **Technical Design**:
    ```go
    // 追加テスト:
    // - TestPrintPlan_WithPermissions:
    //     plan.PermissionActions に:
    //       {Path: "scripts/build.sh", Mode: "0755"}
    //       {Path: "secrets/key.txt", Mode: "0600"}
    //     → 出力に "[CHMOD 0755] scripts/build.sh" と
    //       "[CHMOD 0600] secrets/key.txt" が含まれる
    ```

#### [MODIFY] [plan.go](file://features/devctl/internal/scaffold/plan.go)

*   **Description**: `PrintPlan` に `PermissionActions` の表示セクションを追加。
*   **Technical Design**:
    ```go
    // PrintPlan の Post-actions セクションに追加:
    //   if len(plan.PermissionActions) > 0 {
    //       for _, pa := range plan.PermissionActions {
    //           fmt.Fprintf(w, "  [CHMOD %s] %s\n", pa.Mode, pa.Path)
    //       }
    //   }
    //
    // Summary 行も更新:
    //   postActionCount = len(plan.PostActions.GitignoreEntries) + len(plan.PermissionActions)
    ```

---

#### [MODIFY] [checkpoint.go](file://features/devctl/internal/scaffold/checkpoint.go)

*   **Description**: `CheckpointInfo` に `PermissionsApplied` フィールドを追加。
*   **Technical Design**:
    ```go
    type CheckpointInfo struct {
        // ... 既存フィールド ...
        PermissionsApplied []PermissionRecord `yaml:"permissions_applied,omitempty"`
    }

    type PermissionRecord struct {
        Path string `yaml:"path"`
        Mode string `yaml:"mode"`
    }
    ```
*   **Logic**:
    - `BuildCheckpointFromPlan` で `plan.PermissionActions` を `PermissionsApplied` にコピー。
    - ロールバック時: scaffold で作成されたファイルが削除されるため、パーミッション復元は不要。

---

#### [MODIFY] [scaffold.go](file://features/devctl/internal/scaffold/scaffold.go)

*   **Description**: `Apply` 関数内の `ApplyPostActions` 呼び出しに `baseDir` 引数を追加。
*   **Technical Design**:
    ```diff
    -if err := ApplyPostActions(placement.PostActions, opts.RepoRoot); err != nil {
    +if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir); err != nil {
    ```

---

### 外部依存

#### [MODIFY] [go.mod](file://features/devctl/go.mod)

*   **Description**: `github.com/bmatcuk/doublestar/v4` を追加（`**` グロブパターンサポート用）。
*   **Technical Design**:
    ```bash
    cd features/devctl && go get github.com/bmatcuk/doublestar/v4
    ```

---

### ドキュメント

#### [MODIFY] [000-Reference-Manual.md](file://prompts/phases/000-foundation/refs/tokotachi-scaffolds/000-Reference-Manual.md)

*   **更新内容**:
    - 後処理アクション (`post_actions`) セクションの `file_permissions` を「将来の拡張ポイント」から正式なフィールドとして記載。
    - スキーマ例に `file_permissions` を追加（`executable` と `mode` の両方）。

#### [MODIFY] [001-Default-Template-Spec.md](file://prompts/phases/000-foundation/refs/tokotachi-scaffolds/001-Default-Template-Spec.md)

*   **更新内容**:
    - `placements/default.yaml` の例に `file_permissions: []` を追加（デフォルトテンプレートではスクリプトが `.gitkeep` のみのため空配列）。

## Step-by-Step Implementation Guide

> [!NOTE]
> TDD 方式: 各ステップで `_test.go` を先に作成し、失敗を確認してから実装する。

### Step 1: FilePermission 構造体 + PostActions 拡張 + バリデーション (FP-R1)

1. `internal/scaffold/placement_test.go` にテストケースを追加
   - `file_permissions` の正常パース（executable / mode）
   - `executable` と `mode` の排他制御（mode 優先）
   - `pattern` 空 / `executable` も `mode` も未指定 → バリデーションエラー
   - `mode` が不正値 → バリデーションエラー
2. `internal/scaffold/placement.go` を修正
   - `FilePermission` 構造体・`ResolvedMode()`・`IsExecutable()` を追加
   - `PostActions` に `FilePermissions` フィールドを追加
   - `ParsePlacement` に `validateFilePermissions` を追加
3.  ビルド & テスト: `./scripts/process/build.sh`

### Step 2: doublestar 依存追加 (FP-R2)

1. `go.mod` に `github.com/bmatcuk/doublestar/v4` を追加
   ```bash
   cd features/devctl && go get github.com/bmatcuk/doublestar/v4
   ```
2. ビルド確認: `./scripts/process/build.sh`

### Step 3: applyFilePermissions + ApplyPostActions 拡張 (FP-R2, FP-R3, FP-R6)

1. `internal/scaffold/applier_test.go` にテストケースを追加
   - `TestApplyFilePermissions_Executable`: 0755 確認
   - `TestApplyFilePermissions_Mode0600`: 0600 確認
   - `TestApplyFilePermissions_NoMatch`: エラーなし
   - `TestApplyFilePermissions_Idempotent`: 2回呼び出し
   - `TestApplyPostActions_WithFilePermissions`: gitignore + chmod の両方実行
2. `internal/scaffold/applier.go` を修正
   - `PermissionAction` 構造体を追加
   - `Plan` に `PermissionActions` フィールドを追加
   - `applyFilePermissions` 関数を実装
   - `ApplyPostActions` のシグネチャに `baseDir` を追加し `applyFilePermissions` を呼び出し
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 4: BuildPlan の PermissionActions プレビュー (FP-R4)

1. `internal/scaffold/applier_test.go` にテストケースを追加
   - `TestBuildPlan_WithPermissionActions`: Plan に PermissionActions が含まれることを確認
2. `internal/scaffold/applier.go` の `BuildPlan` を修正
   - `PermissionActions` のプレビュー生成を追加
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 5: PrintPlan の表示拡張 (FP-R4)

1. `internal/scaffold/plan_test.go` にテストケースを追加
   - `TestPrintPlan_WithPermissions`: `[CHMOD]` 表示確認
2. `internal/scaffold/plan.go` の `PrintPlan` を修正
   - `PermissionActions` セクションの表示を追加
   - Summary 行のカウントを更新
3. ビルド & テスト: `./scripts/process/build.sh`

### Step 6: チェックポイント拡張 (FP-R5)

1. `internal/scaffold/checkpoint.go` を修正
   - `PermissionRecord` 構造体と `CheckpointInfo.PermissionsApplied` を追加
   - `BuildCheckpointFromPlan` で `PermissionActions` → `PermissionsApplied` を変換
2. ビルド & テスト: `./scripts/process/build.sh`

### Step 7: scaffold.go の Apply 呼び出し修正

1. `internal/scaffold/scaffold.go` の `Apply` 関数を修正
   - `ApplyPostActions` の呼び出しに `placement.BaseDir` を引数追加
2. ビルド & テスト: `./scripts/process/build.sh`

### Step 8: ドキュメント更新

1. `000-Reference-Manual.md` の後処理セクションを更新
2. `001-Default-Template-Spec.md` に `file_permissions: []` を追加

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    全単体テストが PASS することを確認。特に以下のテストファイル:
    - `internal/scaffold/placement_test.go` — FilePermission のパース・バリデーション
    - `internal/scaffold/applier_test.go` — applyFilePermissions、ApplyPostActions
    - `internal/scaffold/plan_test.go` — PrintPlan の CHMOD 表示

    **ログ確認**: `PASS` が表示され、`FAIL` がないことを確認。

## Documentation

#### [MODIFY] [000-Reference-Manual.md](file://prompts/phases/000-foundation/refs/tokotachi-scaffolds/000-Reference-Manual.md)
*   **更新内容**: `post_actions` セクションの `file_permissions` を正式フィールドとして記載。スキーマ例を追加。

#### [MODIFY] [001-Default-Template-Spec.md](file://prompts/phases/000-foundation/refs/tokotachi-scaffolds/001-Default-Template-Spec.md)
*   **更新内容**: `placements/default.yaml` 例に `file_permissions: []` を追加。
