# 003-Download-History

> **Source Specification**: [003-Download-History.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/003-Download-History.md)

## Goal Description

scaffold のダウンロード履歴を `.kotoshiro/tokotachi/downloaded.yaml` に記録し、依存チェーン内で既に適用済みの静的 scaffold をスキップする。`base_dir` に `{{...}}` を含む動的 scaffold は履歴管理の対象外とし、常にダウンロード・適用する。`--force` フラグで履歴を無視して全再適用を可能にする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point |
| :--- | :--- |
| 1. ダウンロード履歴ファイル (`.kotoshiro/tokotachi/downloaded.yaml`) | Proposed Changes > `download_history.go` |
| 2. 重複ダウンロードのスキップ | Proposed Changes > `scaffold.go` (`applyDependencyChain`, `applySingleScaffold`) |
| 3. 記録タイミング (ApplyFiles+PostActions成功後) | Proposed Changes > `scaffold.go` |
| 4. 動的 `base_dir` の除外 | Proposed Changes > `download_history.go` (`IsDynamic`) および `scaffold.go` |
| 5. `--force` フラグ | Proposed Changes > `scaffold.go` (`RunOptions.Force`), `cmd/scaffold.go` |

## Proposed Changes

### scaffold パッケージ

#### [NEW] [download_history_test.go](file://features/tt/internal/scaffold/download_history_test.go)
*   **Description**: ダウンロード履歴管理の単体テスト（TDD: Red → Green）
*   **Technical Design**:
    ```go
    func TestLoadHistory_NotFound(t *testing.T)
    // tmpDir にファイルなし → Load() が空の DownloadHistory を返す

    func TestSaveAndLoad_Roundtrip(t *testing.T)
    // Save で記録 → Load で読み込み → 一致を assert

    func TestIsDownloaded_Found(t *testing.T)
    // history に "root"/"default" が存在 → IsDownloaded("root", "default") == true

    func TestIsDownloaded_NotFound(t *testing.T)
    // history に "feature" がない → IsDownloaded("feature", "x") == false

    func TestRecordDownload_NewEntry(t *testing.T)
    // 空 history → RecordDownload("root", "default") → Load して確認

    func TestRecordDownload_ExistingCategory(t *testing.T)
    // "root"/"default" 存在 → RecordDownload("root", "another") → 両方存在

    func TestIsDynamic_WithTemplate(t *testing.T)
    // Placement{BaseDir: "features/{{feature_name}}"} → IsDynamic == true

    func TestIsDynamic_WithoutTemplate(t *testing.T)
    // Placement{BaseDir: "."} → IsDynamic == false
    ```

---

#### [NEW] [download_history.go](file://features/tt/internal/scaffold/download_history.go)
*   **Description**: ダウンロード履歴の読み書きと判定ロジック
*   **Technical Design**:
    ```go
    const (
        DownloadHistoryDir      = ".kotoshiro/tokotachi"
        DownloadHistoryFileName = "downloaded.yaml"
    )

    type DownloadRecord struct {
        DownloadedAt string `yaml:"downloaded_at"`
    }

    type DownloadHistory struct {
        History map[string]map[string]DownloadRecord `yaml:"history"`
    }

    type DownloadHistoryStore struct {
        repoRoot string
    }

    func NewDownloadHistoryStore(repoRoot string) *DownloadHistoryStore
    ```
*   **Logic**:
    *   `Load() (*DownloadHistory, error)`:
        - ファイル読み込み → `os.IsNotExist` なら空の `DownloadHistory{History: make(...)}` を返す
    *   `Save(history *DownloadHistory) error`:
        - ディレクトリ作成 (`os.MkdirAll`) → `yaml.Marshal` → `os.WriteFile`
    *   `IsDownloaded(category, name string) bool`:
        - `Load()` → `history.History[category][name]` の存在チェック
    *   `RecordDownload(category, name string) error`:
        - `Load()` → カテゴリ map がなければ作成 → `DownloadRecord{DownloadedAt: time.Now().UTC().Format(time.RFC3339)}` を設定 → `Save()`
    *   `IsDynamic(placement *Placement) bool`:
        - `strings.Contains(placement.BaseDir, "{{")` を返す

---

#### [MODIFY] [scaffold.go](file://features/tt/internal/scaffold/scaffold.go)
*   **Description**: `RunOptions` に `Force` フィールド追加、`Apply`/`applyDependencyChain`/`applySingleScaffold` に履歴チェック・記録ロジックを統合
*   **Technical Design**:
    ```go
    type RunOptions struct {
        // ... 既存フィールド ...
        Force bool // --force flag: ignore download history
    }
    ```
*   **Logic (`applyDependencyChain` の変更)**:
    ```
    historyStore := NewDownloadHistoryStore(opts.RepoRoot)

    for i, dp := range plan.DependencyPlans {
        // 1. placement を取得（fetchTemplateAndPlacement 後）
        // 2. IsDynamic チェック
        isDynamic := historyStore.IsDynamic(placement)

        // 3. 静的 scaffold の履歴チェック（--force でなければ）
        if !isDynamic && !opts.Force && historyStore.IsDownloaded(category, name) {
            logger.Info("Skipping %s/%s (already downloaded)", category, name)
            continue
        }

        // 4. ダウンロード & 適用（既存フロー）
        // ... fetchTemplateAndPlacement, ApplyFiles, ApplyPostActions ...

        // 5. 静的 scaffold のみ履歴記録
        if !isDynamic {
            historyStore.RecordDownload(category, name)
        }
    }
    ```
*   **Logic (`applySingleScaffold` の変更)**: 同様のロジック
    ```
    historyStore := NewDownloadHistoryStore(opts.RepoRoot)
    // fetchTemplateAndPlacement で placement を取得後
    isDynamic := historyStore.IsDynamic(placement)

    if !isDynamic && !opts.Force && historyStore.IsDownloaded(category, name) {
        logger.Info("Skipping %s/%s (already downloaded)", category, name)
        // checkpoint を削除して終了
        return nil
    }

    // ... 既存の apply フロー ...

    if !isDynamic {
        historyStore.RecordDownload(category, name)
    }
    ```

> [!IMPORTANT]
> `applyDependencyChain` ではループ内の `continue` でスキップする。`applySingleScaffold` ではスキップ時に `RemoveCheckpoint` → `return nil` で正常終了する。

---

#### [MODIFY] [scaffold.go (cmd)](file://features/tt/cmd/scaffold.go)
*   **Description**: `--force` フラグを追加し `RunOptions.Force` に渡す
*   **Technical Design**:
    ```go
    var scaffoldFlagForce bool

    // init() に追加:
    scaffoldCmd.Flags().BoolVar(&scaffoldFlagForce, "force", false,
        "Force re-download all scaffolds, ignoring download history")

    // runScaffold() の opts に追加:
    Force: scaffoldFlagForce,
    ```

---

### 統合テスト

#### [MODIFY] [tt_scaffold_test.go](file://tests/integration-test/tt_scaffold_test.go)
*   **Description**: ダウンロード履歴の統合テストを追加
*   **Technical Design**:
    ```go
    func TestScaffoldDownloadHistory(t *testing.T)
    // 1. 空の git repo で `tt scaffold --yes` を実行（root/default）
    // 2. .kotoshiro/tokotachi/downloaded.yaml が存在することを assert
    // 3. ファイルを読み込み、"root"/"default" エントリが存在を assert
    // 4. downloaded_at が空でないことを assert

    func TestScaffoldSkipAlreadyDownloaded(t *testing.T)
    // 1. 空の git repo で `tt scaffold --yes` を実行（初回）
    // 2. 同じコマンドを再実行
    // 3. stderr 出力に "Skipping root/default" が含まれることを assert
    ```

## Step-by-Step Implementation Guide

1.  **テスト作成 (TDD Red)**:
    *   `download_history_test.go` に8テストケースを作成
    *   この時点では `DownloadHistoryStore` 等が未定義のため**コンパイルエラー**

2.  **`download_history.go` の実装 (TDD Green)**:
    *   `DownloadRecord`, `DownloadHistory`, `DownloadHistoryStore` 構造体を定義
    *   `NewDownloadHistoryStore`, `Load`, `Save`, `IsDownloaded`, `RecordDownload`, `IsDynamic` を実装
    *   テストがパスすることを確認

3.  **ビルド & 単体テスト**:
    *   `./scripts/process/build.sh` を実行

4.  **`RunOptions` に `Force` フィールドを追加**:
    *   `scaffold.go` の `RunOptions` に `Force bool` フィールドを追加

5.  **`applyDependencyChain` の変更**:
    *   ループの先頭に `IsDynamic` → `IsDownloaded` チェックを追加
    *   `continue` でスキップ、適用成功後に `RecordDownload` で記録

6.  **`applySingleScaffold` の変更**:
    *   `fetchTemplateAndPlacement` の後に同様のチェックを追加
    *   スキップ時は `RemoveCheckpoint` → `return nil`

7.  **`cmd/scaffold.go` に `--force` フラグを追加**:
    *   `scaffoldFlagForce` 変数と `BoolVar` 登録、`opts.Force` への受け渡し

8.  **ビルド & 単体テスト**:
    *   `./scripts/process/build.sh` を実行

9.  **統合テスト作成・実行**:
    *   `TestScaffoldDownloadHistory`, `TestScaffoldSkipAlreadyDownloaded` を追加
    *   `bin/tt` を `bin/tt.exe` にコピー（Windows 対応）
    *   統合テストを実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestScaffoldDownloadHistory"
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestScaffoldSkipAlreadyDownloaded"
    ```
    *   **Log Verification**:
        - `TestScaffoldDownloadHistory`: `downloaded.yaml` が存在し `root/default` エントリが記録されている
        - `TestScaffoldSkipAlreadyDownloaded`: stderr に `Skipping root/default` が含まれる

3.  **Regression Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestScaffold"
    ```
    *   既存の `TestScaffoldDefault`, `TestScaffoldWithDependencies` 等がパスすることを確認

## Documentation

#### [MODIFY] [003-Download-History.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/003-Download-History.md)
*   **更新内容**: 実装完了後、検証結果に基づいてステータスを更新
