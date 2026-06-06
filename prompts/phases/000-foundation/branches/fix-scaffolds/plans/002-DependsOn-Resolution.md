# 002-DependsOn-Resolution

> **Source Specification**: [002-DependsOn-Resolution.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/002-DependsOn-Resolution.md)

## Goal Description

`ScaffoldEntry` の `depends_on` フィールドを活用し、scaffold 間の依存関係を再帰的に解決する機能を実装する。`tt scaffold feature axsh-go-standard` 実行時に、依存チェーン（`root/default → project/axsh-go-standard → feature/axsh-go-standard`）を自動的に辿り、末端から順に各 scaffold の ZIP をダウンロード・展開・適用する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 依存チェーンの再帰的解決 | Proposed Changes > `dependency.go` (新規) |
| トポロジカル順序での適用 | Proposed Changes > `dependency.go` の `ResolveDependencies` |
| 循環依存の検出 | Proposed Changes > `dependency.go` の `ResolveDependencies` |
| 重複の排除 | Proposed Changes > `dependency.go` の `dedup` |
| ユーザーへの表示 | Proposed Changes > `scaffold.go` の `Run` / `Apply` 変更 |
| 既に適用済みの依存のスキップ | 既存の `conflict_policy: skip` で対応（変更不要） |
| オプション値の収集 | Proposed Changes > `scaffold.go` の `Run` 変更 |

## Proposed Changes

### scaffold パッケージ

---

#### [NEW] [dependency_test.go](file://features/tt/internal/scaffold/dependency_test.go)

*   **Description**: 依存解決ロジックの単体テスト
*   **Technical Design**:
    *   テスト用のモックを使い、ネットワークアクセス不要でテスト可能にする
    *   `EntryFetcher` インターフェースを利用してテスト用の `mockFetcher` を定義
    *   テーブル駆動テスト形式
*   **Logic**:
    *   `TestResolveDependencies_NoDeps`: `depends_on` が空の scaffold → 自身のみ返す
    *   `TestResolveDependencies_SingleChain`: A(depends B) → B(depends C) → C(deps なし) → 結果 `[C, B, A]`
    *   `TestResolveDependencies_CircularDependency`: A → B → A → `"circular dependency"` エラー
    *   `TestResolveDependencies_DiamondDependency`: A → {B, C}, B → D, C → D → 結果に D は1回だけ
    *   各テストケースで `mockFetcher` を設定し、`category/name` をキーとして `ScaffoldEntry` を返す

```go
// mockFetcher - テスト用
type mockFetcher struct {
    entries map[string]*ScaffoldEntry // key = "category/name"
}

func (m *mockFetcher) FetchEntry(category, name string) (*ScaffoldEntry, error) {
    key := category + "/" + name
    if e, ok := m.entries[key]; ok {
        return e, nil
    }
    return nil, fmt.Errorf("not found: %s", key)
}
```

---

#### [NEW] [dependency.go](file://features/tt/internal/scaffold/dependency.go)

*   **Description**: 依存関係の再帰的解決ロジック
*   **Technical Design**:
    *   `EntryFetcher` インターフェース: 依存先の `ScaffoldEntry` を取得する抽象化
    *   `ResolveDependencies` 関数: 再帰的に依存チェーンを走査し、トポロジカル順序で返す
    *   `dedup` ヘルパー: 重複排除

```go
// EntryFetcher は category + name から ScaffoldEntry を取得するインターフェース。
// テスト時にモックに差し替え可能。
type EntryFetcher interface {
    FetchEntry(category, name string) (*ScaffoldEntry, error)
}

// ResolveDependencies は entry の depends_on を再帰的に辿り、
// トポロジカル順序（依存なしが先頭、entry 自身が末尾）の ScaffoldEntry スライスを返す。
// 循環依存を検出した場合はエラーを返す。
func ResolveDependencies(fetcher EntryFetcher, entry *ScaffoldEntry) ([]ScaffoldEntry, error) {
    // visited: 循環検出用（探索スタック上にあるかどうか）
    // resolved: 結果リスト（トポロジカル順序）
    // seen: 重複排除用
    // 内部のヘルパー関数 resolve(entry) を再帰呼び出し
}
```

*   **Logic**:
    1. `visited` マップ (`map[string]bool`) で現在の再帰スタック上のノードを追跡
    2. `seen` マップ (`map[string]bool`) で既に結果に追加済みのノードを追跡
    3. 各 `depends_on` エントリについて:
       - `key = category + "/" + name` を生成
       - `visited[key]` が `true` → 循環依存エラー
       - `seen[key]` が `true` → スキップ（重複排除）
       - それ以外 → `fetcher.FetchEntry` でエントリを取得し、再帰呼び出し
    4. 全依存先の処理後、自身を結果に追加
    5. `visited[key]` を `false` に戻す（バックトラック）

---

#### [MODIFY] [scaffold.go](file://features/tt/internal/scaffold/scaffold.go)

*   **Description**: `Run` と `Apply` 関数を依存チェーン対応に変更
*   **Technical Design**:
    *   `githubEntryFetcher` 構造体を追加: `EntryFetcher` インターフェースの本番実装
    *   `Run` 関数: 依存解決 → 全 scaffold 分のプラン収集 → まとめて表示
    *   `Apply` 関数: 依存チェーン分を順番に適用

```go
// githubEntryFetcher は GitHub API を使って ScaffoldEntry を取得する EntryFetcher 実装。
type githubEntryFetcher struct {
    client *github.Client
}

func (f *githubEntryFetcher) FetchEntry(category, name string) (*ScaffoldEntry, error) {
    // ShardPath(category, name) でパスを算出
    // FetchFile → ParseScaffoldDetail → findEntry
}
```

*   **`Run` 関数の変更点**:
    1. `fetchAndResolveEntry` で対象エントリを取得（既存）
    2. **新規**: `ResolveDependencies(fetcher, entry)` で依存チェーンを解決 → `[]ScaffoldEntry`
    3. 依存チェーンが2件以上の場合、ユーザーに依存解決結果を表示:
       ```
       Resolving dependencies...
         [1/3] root/default
         [2/3] project/axsh-go-standard
         [3/3] feature/axsh-go-standard
       ```
    4. 各 scaffold エントリについてループ:
       - `fetchTemplateAndPlacement` で ZIP ダウンロード・展開
       - `effectiveOptions` でオプション取得
       - オプションがある場合、scaffold 名を表示してから `CollectOptionValues` 実行
       - `BuildPlan` でプラン作成
    5. 全 scaffold のプランをまとめた `CompositePlan` を返す（後述）
    6. `RunOptions` 構造体に `SkipDeps bool` フィールドを追加
*   **`Apply` 関数の変更点**:
    1. 依存チェーンの各 scaffold について順番に適用
    2. 各 scaffold の適用時に進捗を表示: `[1/3] Applying root/default...`

---

#### [MODIFY] [applier.go](file://features/tt/internal/scaffold/applier.go)

*   **Description**: `Plan` 構造体に依存チェーン情報を追加
*   **Technical Design**:

```go
// Plan 構造体に追加するフィールド:
type Plan struct {
    // ...既存フィールド...
    
    DependencyPlans []DependencyPlan // 依存チェーンの各 scaffold プラン
}

// DependencyPlan は依存チェーン中の各 scaffold のプラン情報を保持する。
type DependencyPlan struct {
    Entry        ScaffoldEntry          // scaffold エントリ情報
    Files        []DownloadedFile       // ダウンロード済みファイル
    Placement    *Placement             // 配置定義
    OptionValues map[string]string      // オプション値
    SubPlan      *Plan                  // scaffold 単位のプラン
}
```

---

#### [MODIFY] [plan.go](file://features/tt/internal/scaffold/plan.go)

*   **Description**: `PrintPlan` を依存チェーン対応に変更
*   **Logic**:
    *   `DependencyPlans` が存在する場合、各 scaffold のプランを `[1/3] root/default:` のようなヘッダ付きで表示
    *   `DependencyPlans` が空（依存なし）の場合、既存の表示ロジックをそのまま使用

---

#### [MODIFY] [scaffold.go](file://features/tt/cmd/scaffold.go)

*   **Description**: `--skip-deps` フラグの追加
*   **Technical Design**:
    *   `scaffoldFlagSkipDeps bool` を追加
    *   `scaffoldCmd.Flags().BoolVar(&scaffoldFlagSkipDeps, "skip-deps", false, "Skip dependency resolution")` を追加
    *   `RunOptions.SkipDeps` に値を設定

---

### 統合テスト

#### [MODIFY] [tt_scaffold_test.go](file://tests/integration-test/tt_scaffold_test.go)

*   **Description**: `TestScaffoldWithDependencies` テストを追加
*   **Logic**:
    1. `requireGitHubReachable(t)` でネットワーク確認
    2. `t.TempDir()` + `initGitRepo(t, tmpDir)` で環境準備
    3. `runTTInDir(t, tmpDir, "scaffold", "feature", "axsh-go-standard", "--yes", "--default")` を実行
    4. 検証項目:
       - exit code = 0
       - `root/default` のファイル群（`features/README.md`, `prompts/phases/` 等）が存在
       - `project/axsh-go-standard` のファイル群が存在
       - `feature/axsh-go-standard` のファイル群が存在
       - stderr に `[1/3]`, `[2/3]`, `[3/3]` の進捗表示が含まれる

## Step-by-Step Implementation Guide

1.  [x] **`EntryFetcher` インターフェースと `mockFetcher` の定義**
2.  [x] **依存解決テストの作成 (TDD: Red)**
3.  [x] **依存解決ロジックの実装 (TDD: Green)**
4.  [x] **`Plan` 構造体の拡張**
5.  [x] **`PrintPlan` の依存チェーン対応**
6.  [x] **`githubEntryFetcher` の実装**
7.  [x] **`Run` 関数の変更**
8.  [x] **`Apply` 関数の変更**
9.  [x] **`RunOptions` に `SkipDeps` を追加**
10. [x] **ビルド & 単体テスト実行** ✅ PASS
11. [x] **統合テスト作成・実行** ✅ PASS (全5テスト、リグレッションなし)

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: `dependency_test.go` の4つのテストケース（`TestResolveDependencies_*`）がすべて PASS すること

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestScaffold"
    ```
    *   **Log Verification**:
        *   既存の `TestScaffoldDefault`, `TestScaffoldList`, `TestScaffoldDefaultLocaleJa`, `TestScaffoldCwdFlag` が引き続き PASS
        *   新規の `TestScaffoldWithDependencies` が PASS
        *   `TestScaffoldWithDependencies` の stderr 出力に依存解決の進捗表示が含まれること

## Documentation

本計画で影響を受ける既存ドキュメントはありません。リモートリポジトリの仕様書は変更不要です。
