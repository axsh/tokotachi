# 001-Reorganize-Command-Structure

> **Source Specification**: [001-Reorganize-Command-Structure.md](file://prompts/phases/000-foundation/ideas/fix-naming/001-Reorganize-Command-Structure.md)

## Goal Description

`tt` CLIのコマンド体系を再構成する。現行の `up`/`open`/`close` の責務を分離し、単機能コマンド（`create`, `delete`, `editor`, `up`, `down`）とSyntax Sugarコマンド（`open`, `close`）に整理する。

## User Review Required

> [!IMPORTANT]
> **`up` コマンドの `--editor` フラグ廃止**: 既存の統合テストで `--editor` フラグを使用しているテストがある場合、テストの修正が必要です。

> [!WARNING]
> **`action.Close` ロジックの分割**: 現行の `action.Close` は複雑な再帰処理を含んでおり、これを `action.Delete` と `action.Close`（Syntax Sugar）に分割します。分割後の責務境界が適切か確認をお願いします。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `create <branch>` — worktree作成 | Proposed Changes > `cmd/create.go` [NEW] |
| R1: `delete <branch>` — worktree削除、安全ガード | Proposed Changes > `cmd/delete.go` [NEW], `action/delete.go` [NEW] |
| R1: `status`, `list` — 変更なし | 変更不要 |
| R2: `up` — feature必須化、worktree自動作成除去 | Proposed Changes > `cmd/up.go` [MODIFY] |
| R2: `up` — `--editor` フラグ廃止 | Proposed Changes > `cmd/up.go` [MODIFY] |
| R2: `down`, `exec`, `shell` — 変更なし | 変更不要 |
| R3: `pr` — 変更なし | 変更不要 |
| R4: `editor <branch> [feature]` | Proposed Changes > `cmd/editor.go` [NEW] |
| R5: `open` — Syntax Sugar: create→up→editor | Proposed Changes > `cmd/open.go` [MODIFY] |
| R5: `close` — Syntax Sugar: down→delete | Proposed Changes > `cmd/close.go` [MODIFY], `action/close.go` [MODIFY] |
| R6: `scaffold`, `doctor`, `_update-code-status` — 変更なし | 変更不要 |

---

## Close 再帰ロジックの等価性分析

### 現行実装（仮想言語）

```
function Close(opts, wm):
    // ── Phase 1: ネストworktree検出 ──
    if opts.depth > 0:
        nested = wm.FindNestedWorktrees(opts.branch)
        if depth == 1:
            depthWarning = any child has grandchildren
    else:
        warn("depth limit reached, nested worktrees will NOT be closed")

    // ── Phase 2: 確認プロンプト ──
    if not opts.yes:
        show preview(nested, depthWarning)
        if user says no: return (abort)

    // ── Phase 3: 再帰的にネストworktreeをクローズ ──
    if nested is not empty AND depth > 0:
        for each child in nested:
            Close({branch: child, force, depth: depth-1, yes: true}, wm)  // ← 再帰呼び出し

    // ── Phase 4: 自ブランチの処理 ──
    if opts.feature != "":
        // ── 分岐A: feature指定あり ──
        Down(containerName)                          // コンテナ停止
        state.RemoveFeature(feature)
        if state.features is empty:
            wm.Remove(branch)                        // worktree削除
            wm.DeleteBranch(branch)                  // ブランチ削除
            state.Remove()                           // ステートファイル削除
        else:
            state.Save()                             // 残りfeatureの状態保存
        pruneIfForce()
    else:
        // ── 分岐B: feature省略（全feature対象） ──
        failCount = 0
        for each feature in state.ActiveFeatureNames():
            if Down(containerName) fails:
                failCount++
        if failCount > 0:
            warn("skipping worktree removal")
            return                                    // ← worktreeは残す
        wm.Remove(branch)                            // worktree削除
        wm.DeleteBranch(branch)                      // ブランチ削除
        state.Remove()                               // ステートファイル削除
        pruneIfForce()
```

### 新しい実装（仮想言語）— Delete + Close に分割

```
function Delete(opts, wm):
    // ── 安全ガード（新規） ──
    state = state.Load(branch)
    if state exists AND state.HasActiveFeatures():
        return ERROR("active containers exist, stop them first")

    // ── Phase 1: ネストworktree検出 ──
    // (現行 Close Phase 1 と同一ロジック)
    if opts.depth > 0:
        nested = wm.FindNestedWorktrees(opts.branch)
        if depth == 1:
            depthWarning = any child has grandchildren
    else:
        warn("depth limit reached")

    // ── Phase 2: 確認プロンプト ──
    // (現行 Close Phase 2 と同一ロジック)
    if not opts.yes:
        show preview(nested, depthWarning)
        if user says no: return (abort)

    // ── Phase 3: 再帰的にネストworktreeを削除 ──
    if nested is not empty AND depth > 0:
        for each child in nested:
            Delete({branch: child, force, depth: depth-1, yes: true}, wm)  // ← 再帰

    // ── Phase 4: worktree/ブランチ/ステート削除 ──
    // (現行 Close 分岐B の末尾ロジックと同一)
    wm.Remove(branch)
    wm.DeleteBranch(branch)
    state.Remove()
    pruneIfForce()
```

```
function Close(opts, wm):                    // Syntax Sugar
    if opts.feature != "":
        // ── 分岐A: feature指定 ──
        Down(containerName)                  // コンテナ停止
        state.RemoveFeature(feature)
        if NOT state.HasActiveFeatures():    // 全container停止済み
            Delete({branch, force, depth, yes, ...}, wm)
        else:
            state.Save()                     // 残りfeatureの状態保存
    else:
        // ── 分岐B: feature省略（全feature対象） ──
        failCount = 0
        for each feature in state.ActiveFeatureNames():
            if Down(containerName) fails:
                failCount++
        if failCount > 0:
            warn("skipping delete")
            return
        Delete({branch, force, depth, yes: true, ...}, wm)
```

### 等価性のチェック

| 処理 | 現行 Close | 新 Delete + Close | 等価? |
|------|-----------|-------------------|-------|
| **ネストworktree再帰削除** | `Close` 内で `Close` を再帰呼び出し | `Delete` 内で `Delete` を再帰呼び出し | ✅ 等価（ロジック同一） |
| **ネスト深度制限・警告** | Phase 1 で depth チェック | `Delete` Phase 1 で同一チェック | ✅ 等価 |
| **確認プロンプト** | Phase 2 で `[y/N]` 表示 | `Delete` Phase 2 で同一表示 | ✅ 等価 |
| **子worktreeの確認スキップ** | 再帰時 `Yes=true` で確認省略 | 再帰時 `Yes=true` で確認省略 | ✅ 等価 |
| **pruneIfForce** | 現行: Close 内で呼び出し | 新: Delete 内で呼び出し | ✅ 等価 |
| **分岐B: feature省略時** | Down 全feature → (失敗なければ) worktree/branch/state 削除 | Close: Down 全feature → Delete 呼び出し | ✅ 等価 |
| **分岐B: Down 失敗時** | `failCount > 0` → worktree残す | Close: `failCount > 0` → Delete スキップ | ✅ 等価 |
| **分岐A: feature指定時** | Down → RemoveFeature → features空なら worktree/branch/state 削除 | Close: Down → RemoveFeature → HasActiveFeatures == false なら Delete | ⚠️ **差異あり**（下記参照） |
| **Delete: 安全ガード** | なし | `HasActiveFeatures()` チェック → エラー | 🆕 **新規追加** |

### 非等価な点の詳細

#### 差異1: feature指定時の worktree/branch 削除条件

**現行**: `sf.RemoveFeature()` 後に `len(sf.Features) == 0` で判定
**新**: `Close` が `sf.RemoveFeature()` 後に `HasActiveFeatures()` で判定し、false なら `Delete` を呼ぶ

- `len(sf.Features) == 0` と `HasActiveFeatures() == false` は **セマンティクスが異なる**
  - `len(sf.Features) == 0` = featuresマップが空（featureが一つも無い）
  - `HasActiveFeatures() == false` = `status == "active"` のfeatureが無い（`"stopped"` のfeatureが残っている可能性）
- **仕様書の要件**: 「全Dev Containerが停止していなければ、`delete` をスキップする」
- **判定**: 仕様書に沿えば `HasActiveFeatures()` が適切。ただし、`stopped` 状態のfeatureが残っている場合でもworktree削除を実行するかどうかは、`Delete` 内の安全ガード（`HasActiveFeatures()` チェック）が判定する。
- **結論**: 新実装では `Close(feature指定)` → `RemoveFeature` → `HasActiveFeatures() == false` → `Delete` の流れとなり、`Delete` 内で再度 `HasActiveFeatures()` をチェックする。安全ガードが二重にかかるため、**安全側に移行**する。現行とは厳密に等価ではないが、仕様書の要件に合致する。

#### 差異2: Delete の安全ガード（新規）

**現行**: `close` で feature省略時、Down 失敗でも worktree は残すが、明示的なエラーは返さない（`return nil`）

**新**: `delete` コマンド単独呼び出し時、起動中コンテナがあれば **エラーを返す**（`return error`）

- これは仕様書の要件「対象ブランチに紐づくDev Containerが一つでも起動中であれば、エラーを返して終了する」に準拠
- 従来の `close` Syntax Sugar 経由では `Down` → `Delete` の順で呼ばれるため、`Delete` に到達する時点では通常コンテナは停止済み。安全ガードは `delete` 単独呼び出しの保護が主目的。

---

## Proposed Changes

### action パッケージ（`internal/action/`）

#### [NEW] [delete.go](file://features/tt/internal/action/delete.go)

*   **Description**: worktree削除 + ブランチ削除の単機能アクション。現行の `action.Close` からworktree/ブランチ削除ロジックを抽出。
*   **Technical Design**:
    ```go
    // DeleteOptions holds parameters for the delete action.
    type DeleteOptions struct {
        Branch      string
        Force       bool
        RepoRoot    string
        ProjectName string
        Depth       int       // max recursion depth for nested worktrees
        Yes         bool      // skip [y/N] confirmation
        Stdin       io.Reader // input source for confirmation
    }
    
    // Delete removes worktree, deletes branch, and cleans up state.
    // Returns error if any active containers are found (safety guard).
    func (r *Runner) Delete(opts DeleteOptions, wm *worktree.Manager) error
    ```
*   **Logic**:
    1. `state.Load()` でステートファイルを読み込み
    2. `sf.HasActiveFeatures()` で起動中コンテナを確認 → 起動中ならエラーを返す（安全ガード）
    3. 現行 `Close` の Phase 1 (ネストworktree検出) を移植
    4. 現行 `Close` の Phase 2 (確認プロンプト) を移植
    5. 再帰的にネストworktreeを削除（`DeleteOptions` で子呼び出し、`Yes=true`）
    6. `wm.Remove()` でworktree削除
    7. `wm.DeleteBranch()` でブランチ削除
    8. `state.Remove()` でステートファイル削除
    9. `pruneIfForce()` で `--force` 時に prune

---

#### [NEW] [delete_test.go](file://features/tt/internal/action/delete_test.go)

*   **Description**: `Delete` アクションの単体テスト。
*   **Technical Design**: テーブル駆動テスト。
    ```go
    func TestDelete(t *testing.T) {
        tests := []struct {
            name           string
            hasActive      bool    // 起動中コンテナあり
            worktreeExists bool
            force          bool
            wantErr        bool
            wantErrMsg     string  // エラーメッセージのサブストリング
        }{...}
    }
    ```
*   テストケース:
    - 正常系: worktree削除成功
    - 異常系: 起動中コンテナがある場合はエラー
    - `--force` 時の prune 実行
    - ステートファイルが存在しない場合も正常に動作

---

#### [MODIFY] [close.go](file://features/tt/internal/action/close.go)

*   **Description**: Syntax Sugar版 `close` のロジックに書き直す。worktree/ブランチ削除ロジックは `Delete` に委譲。
*   **Technical Design**:
    ```go
    // CloseOptions holds parameters for the close action (Syntax Sugar).
    type CloseOptions struct {
        Feature     string    // empty = close all features + delete
        Branch      string
        Force       bool
        RepoRoot    string
        ProjectName string
        Depth       int
        Yes         bool
        Stdin       io.Reader
    }
    
    // Close performs the close sequence (Syntax Sugar).
    // feature指定あり: down(指定feature) → delete(全container停止時のみ)
    // feature省略: down(全feature) → delete
    func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error
    ```
*   **Logic** (feature省略時):
    1. ステートファイルを読み込み
    2. `ActiveFeatureNames()` で全活性featureを取得
    3. 各featureに対して `Down(containerName)` を呼び出し
    4. 全成功したら `Delete(deleteOpts, wm)` を呼び出し
    5. 1つでも失敗したら `Delete` をスキップしてログ出力
*   **Logic** (feature指定時):
    1. 指定featureのコンテナ名を解決
    2. `Down(containerName)` を呼び出し
    3. ステートファイルから該当featureを `RemoveFeature()` で削除
    4. 残りのfeatureが全て停止済みかを判定
    5. 全停止済みなら `Delete(deleteOpts, wm)` を呼び出し
    6. 起動中featureがあれば `Delete` をスキップし、更新済みステートを保存

---

#### [MODIFY] [close_test.go](file://features/tt/internal/action/close_test.go)

*   **Description**: 新しいSyntax Sugar版 `Close` に合わせてテストケースを更新。
*   テストケース:
    - feature省略: 全containerを `Down` → `Delete` 呼び出し
    - feature指定: 指定containerを `Down` → 全停止時のみ `Delete`
    - feature指定 + 他container起動中: `Delete` スキップ

---

### cmd パッケージ（`cmd/`）

#### [NEW] [create.go](file://features/tt/cmd/create.go)

*   **Description**: `create <branch>` コマンド。worktree作成の単機能コマンド。
*   **Technical Design**:
    ```go
    var createCmd = &cobra.Command{
        Use:   "create <branch>",
        Short: "Create a branch and worktree",
        Long:  "Create a new git branch and set up a worktree for development.",
        Args:  cobra.ExactArgs(1),
        RunE:  runCreate,
    }
    
    func runCreate(cmd *cobra.Command, args []string) error
    ```
*   **Logic**:
    1. `InitContext(args)` でコンテキスト初期化（featureなし）
    2. `worktree.Manager` を作成
    3. `wm.Exists(branch)` でworktreeの存在確認 → 既存なら `"worktree already exists"` を表示して正常終了
    4. `wm.Create(branch)` でworktree作成
    5. レポートにステップ追加

---

#### [NEW] [delete.go](file://features/tt/cmd/delete.go)

*   **Description**: `delete <branch>` コマンド。worktree/ブランチ削除の単機能コマンド。
*   **Technical Design**:
    ```go
    var (
        deleteFlagForce bool
        deleteFlagDepth int
        deleteFlagYes   bool
    )
    
    var deleteCmd = &cobra.Command{
        Use:   "delete <branch>",
        Short: "Delete worktree and branch",
        Long:  "Remove worktree and delete branch. Fails if any Dev Container is still running.",
        Args:  cobra.ExactArgs(1),
        RunE:  runDelete,
    }
    
    func init() {
        deleteCmd.Flags().BoolVar(&deleteFlagForce, "force", false, "Force delete even if branch is not merged")
        deleteCmd.Flags().IntVar(&deleteFlagDepth, "depth", 10, "Maximum depth for recursive nested worktree close")
        deleteCmd.Flags().BoolVar(&deleteFlagYes, "yes", false, "Skip [y/N] confirmation")
    }
    
    func runDelete(cmd *cobra.Command, args []string) error
    ```
*   **Logic**:
    1. `InitContext(args)` でコンテキスト初期化（branchのみ）
    2. `action.Delete(opts, wm)` を呼び出し
    3. レポートにステップ追加

---

#### [NEW] [editor.go](file://features/tt/cmd/editor.go)

*   **Description**: `editor <branch> [feature]` コマンド。現行の `open.go` からエディタ起動ロジックを移植。
*   **Technical Design**:
    ```go
    var (
        editorFlagEditor string
        editorFlagAttach bool
    )
    
    var editorCmd = &cobra.Command{
        Use:   "editor <branch> [feature]",
        Short: "Open the editor",
        Long:  "Open the editor for the given branch. Use --attach to reconnect to a running container.",
        Args:  cobra.RangeArgs(1, 2),
        RunE:  runEditor,
    }
    
    func init() {
        editorCmd.Flags().StringVar(&editorFlagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
        editorCmd.Flags().BoolVar(&editorFlagAttach, "attach", false, "Attempt DevContainer attach to running container")
    }
    
    func runEditor(cmd *cobra.Command, args []string) error
    ```
*   **Logic**: 現行 `open.go` の L175-L201 のエディタ起動ロジックをそのまま移植。
    1. `InitContext(args)` でコンテキスト初期化
    2. `ResolveEnvironment(editorFlagEditor)` で環境解決
    3. `resolve.Worktree()` でworktreeパス解決
    4. `editor.NewLauncher(ed)` でランチャー作成
    5. `plan.Build()` で `TryDevcontainerAttach` 判定
    6. `ctx.ActionRunner.Open(launcher, launchOpts)` で起動
    7. `--up` フラグは **削除**（`open` コマンドで代替）

---

#### [MODIFY] [up.go](file://features/tt/cmd/up.go)

*   **Description**: worktree自動作成ロジックの除去、feature必須化、`--editor` フラグ廃止。
*   **変更点**:
    1. `Use` を `"up <branch> <feature>"` に変更
    2. `Args` を `cobra.ExactArgs(2)` に変更
    3. `upFlagEditor` フラグ定義と `init()` 内の登録を削除
    4. `runUp` 内:
        - worktree自動作成ブロック（L68-L78）を削除
        - worktreeが存在しない場合はエラーを返すチェックを追加: `if !wm.Exists(ctx.Branch) { return error }`
        - feature未指定時のスキップ分岐（L200-L202）を削除（常にfeatureあり）
        - `p.ShouldOpenEditor` によるエディタ起動ブロック（L209-L231）を削除

---

#### [MODIFY] [open.go](file://features/tt/cmd/open.go)

*   **Description**: Syntax Sugarとして書き直し: `create → up → editor` の一連操作。
*   **Technical Design**:
    ```go
    var (
        openFlagEditor string
    )
    
    var openCmd = &cobra.Command{
        Use:   "open <branch> [feature]",
        Short: "Create worktree, start container, and open editor",
        Long:  "Syntax sugar: runs create → up → editor in sequence. " +
               "If feature is omitted, skips container start.",
        Args:  cobra.RangeArgs(1, 2),
        RunE:  runOpen,
    }
    
    func init() {
        openCmd.Flags().StringVar(&openFlagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
    }
    
    func runOpen(cmd *cobra.Command, args []string) error
    ```
*   **Logic**:
    1. `InitContext(args)` でコンテキスト初期化
    2. **Step 1 (create)**: `wm.Exists(branch)` → 存在しなければ `wm.Create(branch)`
    3. **Step 2 (up)**: `ctx.HasFeature()` が true の場合のみ:
        - `resolve.ContainerName()` / `resolve.ImageName()` でコンテナ名解決
        - `ctx.ActionRunner.Status()` で起動チェック → 起動中ならスキップ
        - 現行 `up.go` のコンテナ起動ロジック（L117-L199）を使用
    4. **Step 3 (editor)**: `ResolveEnvironment(openFlagEditor)` でエディタ解決 → `editor.NewLauncher()` → `ActionRunner.Open()`
    5. `--attach` および `--up` フラグは削除（`open` 自体が create+up+editor のため不要）

---

#### [MODIFY] [close.go](file://features/tt/cmd/close.go)

*   **Description**: `action.Close` を呼び出すだけのSyntax Sugarコマンド。
*   **変更点**:
    - `closeFlagForce`, `closeFlagDepth`, `closeFlagYes` フラグはそのまま維持
    - `Short` / `Long` の説明文を更新
    - `runClose` 内のロジックは現行とほぼ同じ（`action.Close` を呼ぶだけ）
    - `action.Close` が新しいSyntax Sugar版に置き換わるため、cmd側の変更は最小限

---

#### [MODIFY] [root.go](file://features/tt/cmd/root.go)

*   **Description**: 新コマンドの登録追加。
*   **変更点**:
    - `init()` に以下を追加:
      ```go
      rootCmd.AddCommand(createCmd)
      rootCmd.AddCommand(deleteCmd)
      rootCmd.AddCommand(editorCmd)
      ```

---

## Step-by-Step Implementation Guide

### Phase 1: action レイヤー — テスト先行

- [x] **Step 1**: `action/delete_test.go` を作成
    - テーブル駆動テストで `Delete` のテストケースを定義
    - この時点ではコンパイルエラー（`Delete` 関数未実装）が発生する
- [x] **Step 2**: `action/delete.go` を作成
    - `DeleteOptions` 構造体と `Delete` メソッドを実装
    - 現行 `action/close.go` からworktree/ブランチ削除ロジックを抽出
    - 安全ガード（`HasActiveFeatures` チェック）を追加
- [x] **Step 3**: `action/close_test.go` を更新
    - 新しいSyntax Sugar版 `Close` のテストケースに書き換え
    - feature省略/指定の分岐テスト追加
- [x] **Step 4**: `action/close.go` を書き換え
    - worktree/ブランチ削除ロジックを `Delete` に委譲
    - feature省略時: `Down(全feature)` → `Delete`
    - feature指定時: `Down(指定feature)` → 全停止なら `Delete`

### Phase 2: cmd レイヤー

- [x] **Step 5**: `cmd/create.go` を作成
    - `createCmd` のcobra定義と `runCreate` を実装
- [x] **Step 6**: `cmd/delete.go` を作成
    - `deleteCmd` のcobra定義と `runDelete` を実装
    - `action.Delete` を呼び出すだけのシンプルなコマンド
- [x] **Step 7**: `cmd/editor.go` を作成
    - 現行 `open.go` のエディタ起動ロジック（L175-L201相当）を移植
    - `--up` フラグは含めない
- [x] **Step 8**: `cmd/up.go` を変更
    - feature必須化: `Args: cobra.ExactArgs(2)`
    - worktree自動作成ブロック削除
    - worktree存在チェック追加
    - `--editor` フラグ削除、エディタ起動ブロック削除
- [x] **Step 9**: `cmd/open.go` を書き換え
    - Syntax Sugar: create → up → editor
    - `--attach`, `--up` フラグ削除
- [x] **Step 10**: `cmd/close.go` を更新
    - 説明文の更新のみ（action レイヤーの変更に依存）
- [x] **Step 11**: `cmd/root.go` に新コマンド登録
    - `createCmd`, `deleteCmd`, `editorCmd` を追加

### Phase 3: ビルド検証

- [x] **Step 12**: ビルドと単体テスト実行
    ```bash
    ./scripts/process/build.sh
    ```

### Phase 4: 統合テストの修正・追加

> [!IMPORTANT]
> 以下の統合テストは `tests/integration-test/` 配下に配置する。
> 全テストファイルの影響分析を実施済み。

#### 既存テストの影響分析

| ファイル | 影響 | 修正内容 |
|---------|------|---------|
| `helpers_test.go` | なし | `branchName`, `featureName` 定数、`runTT`, `cleanupTTDown` は変更不要 |
| `tt_up_test.go` | なし | 既に `runTT(t, "up", branchName, featureName)` で feature引数付きで呼んでいるため影響なし |
| `tt_up_git_test.go` | なし | 同上。`runTT(t, "up", branchName, featureName, "--verbose")` で呼んでおり影響なし |
| `tt_down_test.go` | なし | `up` のsetup呼び出しは feature引数付き。`down` コマンド自体は変更なし |
| `tt_status_test.go` | なし | `up` のsetup呼び出しは feature引数付き。`status` コマンド自体は変更なし |
| `tt_list_code_test.go` | なし | `list` コマンドは変更対象外 |
| `tt_doctor_test.go` | なし | `doctor` コマンドは変更対象外 |
| `tt_scaffold_test.go` | なし | `scaffold` コマンドは変更対象外 |
| `docker_build_test.go` | なし | Dockerfileビルドの直接テストのため影響なし |
| `tt_env_option_test.go` | **あり** | `open` コマンドのフラグ変更（`--up`, `--attach` 廃止）に影響。`open` の動作が Syntax Sugar に変わるため、テスト内容を見直す必要あり |

#### Step 13: 既存統合テストの修正

- [x] **`tt_env_option_test.go` を修正**
    - 現行テスト `TestTtOpen_NoEnvByDefault`:
      ```go
      // 現行: open コマンドで --dry-run テスト
      runTT(t, "open", "--dry-run", "fix-env-option")
      ```
    - `open` が Syntax Sugar（create→up→editor）に変わるため:
      - `--dry-run` での動作が変わる可能性がある
      - テストの意図（`--env` フラグの挙動確認）は `open` でも `editor` でも検証可能
      - `open` コマンドのまま維持し、Syntax Sugar版で同じ `--env` 挙動が動作するか検証

#### Step 14: 新規統合テストの追加

- [x] **`tt_create_delete_test.go` を新規作成**
    - `TestTtCreate_CreatesWorktree`: `tt create <branch>` でworktreeが作成されること
    - `TestTtCreate_IdempotentIfExists`: 既存worktreeで再実行しても正常終了すること
    - `TestTtDelete_RemovesWorktree`: `tt delete <branch> --yes` でworktreeとブランチが削除されること
    - `TestTtDelete_BlockedByRunningContainer`: 起動中コンテナがある場合にエラーを返すこと
    - `TestTtDelete_DryRun`: `--dry-run` で実際の削除が行われないこと

- [x] **`tt_editor_test.go` を新規作成**
    - `TestTtEditor_DryRun`: `tt editor <branch> --dry-run` が正常終了すること
    - `TestTtEditor_NoWorktreeError`: worktreeが存在しないbranchでエラーを返すこと

- [x] **`tt_open_close_syntax_test.go` を新規作成** (Syntax Sugar テスト)
    - `TestTtOpen_CreateAndEditor`: `tt open <branch> --dry-run` でworktree作成＋エディタ起動の流れが実行されること
    - `TestTtClose_DownAndDelete`: `tt close <branch> --yes` でコンテナ停止→worktree削除の流れが実行されること

#### Step 15: 統合テスト実行

- [x] 統合テスト実行
    ```bash
    ./scripts/process/integration_test.sh
    ```

---

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    - 全単体テストがパスすること
    - 新規テスト `action/delete_test.go` がパスすること
    - 更新テスト `action/close_test.go` がパスすること

2. **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    - 既存の統合テスト 全10ファイルがパスすること
    - 新規統合テスト 3ファイルがパスすること:
      - `tt_create_delete_test.go`
      - `tt_editor_test.go`
      - `tt_open_close_syntax_test.go`
    - **Log Verification**: 各テストの stderr に `PASS` が含まれ、`FAIL` が含まれないこと

---

## Documentation

#### [MODIFY] [README.md](file://README.md)
*   **更新内容**: コマンド使用例に `create`, `delete`, `editor` を追加。`up` のUsageを更新。
