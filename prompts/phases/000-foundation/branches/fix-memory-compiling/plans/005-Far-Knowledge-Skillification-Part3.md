# 005-Far-Knowledge-Skillification-Part3

> **Source Specification**: [005-Far-Knowledge-Skillification.md](../ideas/005-Far-Knowledge-Skillification.md)

## Goal Description

Part3 では emitter の拡張 (`prompts/memory/branches/*/skills/` からの集約 compile)、E2E シナリオテストの実装、および全体統合テストを行う。

## User Review Required

> [!IMPORTANT]
> emitter 拡張は全 Coding Agent エミッター (antigravity, codex, cursor, claude-code) に影響する。各エミッターの `Emit()` メソッドに `branches/*/skills/` スキャンロジックを追加する必要がある。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R9: tt prompt update の拡張 (branches/*/skills/ からの集約) | Proposed Changes > emitter/ |
| E2E テスト (Phase 0-6) | Proposed Changes > tests/tt/ |
| R5: prompts/memory/knowledge/ ディレクトリ初期化 | Proposed Changes > prompts/memory/ |

## Proposed Changes

### 1. emitter の拡張 (R9)

#### [MODIFY] [emitter.go](file://features/tt/internal/prompt/emitter/emitter.go)
*   **Description**: `branches/*/skills/` をスキャンし、追加の capability として扱う共通関数を追加。
*   **Technical Design**:
    ```go
    // ScanBranchSkills walks prompts/memory/branches/*/skills/ and returns
    // a list of Capability objects parsed from __far-knowledge-*.md files.
    func ScanBranchSkills(memoryDir string) ([]manifest.Capability, error)
    ```
*   **Logic**:
    *   `memoryDir/branches/` 配下のディレクトリを走査
    *   各 `skills/` サブディレクトリ内の `__far-knowledge-*.md` ファイルを読み込む
    *   YAML frontmatter をパースして `manifest.Capability` に変換
    *   `user_visible: false` のフィールドを保持
    *   返却された Capability リストは通常の capability と同様に emit 処理に回す

#### [MODIFY] [emitter_test.go](file://features/tt/internal/prompt/emitter/emitter_test.go)
*   **Description**: ScanBranchSkills のテスト追加。
*   **Test Cases**:
    *   空の branches/ -> 空リスト
    *   1ブランチに2スキル -> 2件の Capability 返却
    *   不正なフロントマターのファイルはスキップ + warning
    *   `__` プレフィックスなしのファイルはスキップ

#### [MODIFY] [codex.go](file://features/tt/internal/prompt/emitter/codex.go)
*   **Description**: Emit() 内で ScanBranchSkills を呼び出し、通常の capability に追加。
*   **Logic**:
    *   `resolved.Capabilities` に `ScanBranchSkills()` の結果を append
    *   スキル配置先は通常と同じ `skillsDir` (`.agents/skills/`)
    *   `__` プレフィックス付きのスキル ID はそのまま維持

#### [MODIFY] [antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go)
*   **Description**: codex.go と同様の拡張。

#### [MODIFY] [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go)
*   **Description**: codex.go と同様の拡張。配置先は `.cursor/skills/`。

#### [MODIFY] [claude_code.go](file://features/tt/internal/prompt/emitter/claude_code.go)
*   **Description**: codex.go と同様の拡張。Claude Code のスキル配置先に合わせる。

---

### 2. 知識ディレクトリの初期化

#### [NEW] [prompts/memory/knowledge/.gitkeep](file://prompts/memory/knowledge/.gitkeep)
*   **Description**: 階層化された遠方知識ディレクトリの初期化用空ファイル。

---

### 3. E2E シナリオテスト

#### [NEW] [tests/tt/tt_far_knowledge_e2e_test.go](file://tests/tt/tt_far_knowledge_e2e_test.go)
*   **Description**: 仕様の Phase 0-6 を自動化した E2E テスト。
*   **Technical Design**:
    ```go
    package tt_test

    import (
        "testing"
    )

    // TestFarKnowledgeE2E は遠方知識の記録->体系化->スキル化->デプロイの
    // 全フローを検証する E2E テスト。
    func TestFarKnowledgeE2E(t *testing.T) {
        // Phase 0: クリーン環境セットアップ
        t.Run("Phase0_Scaffold", testPhase0Scaffold)
        // Phase 1: ワークフロー・スキルのデプロイ
        t.Run("Phase1_PromptUpdate", testPhase1PromptUpdate)
        // Phase 2: record による知識記録
        t.Run("Phase2_RecordKnowledge", testPhase2RecordKnowledge)
        // Phase 3: 新規カテゴリ作成
        t.Run("Phase3_NewCategory", testPhase3NewCategory)
        // Phase 4: 既存カテゴリ追記
        t.Run("Phase4_AppendCategory", testPhase4AppendCategory)
        // Phase 5: カテゴリ統合 (merge)
        t.Run("Phase5_MergeCategories", testPhase5MergeCategories)
        // Phase 6: スキル化とデプロイ
        t.Run("Phase6_SkillifyAndDeploy", testPhase6SkillifyAndDeploy)
    }
    ```
*   **Logic**:

##### Phase 0: クリーン環境セットアップ
```go
func testPhase0Scaffold(t *testing.T) {
    // 1. tmp/e2e-far-knowledge/ を作成
    // 2. tt scaffold go-axsh-standard を実行
    // 3. git init + git add -A + git commit
    // 検証:
    //   - exit code 0
    //   - prompts/manifest/ が存在
    //   - scripts/code/agent/record.sh が存在
}
```

##### Phase 1: ワークフローのデプロイ
```go
func testPhase1PromptUpdate(t *testing.T) {
    // 1. tt prompt update を実行
    // 検証:
    //   - .agents/workflows/execute-implementation-plan.md が存在
    //   - .agents/workflows/systematize-far-knowledge.md が存在
    //   - .agents/skills/record-far-knowledge/SKILL.md が存在
    //   - .agents/skills/pre-push-knowledge-check/SKILL.md が存在
}
```

##### Phase 2: record による知識記録
```go
func testPhase2RecordKnowledge(t *testing.T) {
    // 1. record.sh を4回実行 (design-pattern x2, convention x2)
    // 検証:
    //   - 各コマンドが exit code 0
    //   - prompts/memory/var/intake/pending/ に4件の JSON
    //   - 各 JSON の flags に適切なフラグ
}
```

##### Phase 3: 新規カテゴリ作成
```go
func testPhase3NewCategory(t *testing.T) {
    // 1. intake list --status pending で4件確認
    // 2. knowledge list で0件確認
    // 3. knowledge add x3 (error-handling, test-conventions, logging)
    //    - content-file には事前に用意したテンプレートを使用
    // 4. intake processed x4
    // 検証:
    //   - prompts/memory/knowledge/error-handling/_category.yaml が存在
    //   - prompts/memory/knowledge/error-handling/*.md が存在
    //   - knowledge list で3件
    //   - intake list --status pending で0件
    //   - intake list --status processed で4件 (存在確認)
}
```

##### Phase 4: 既存カテゴリ追記
```go
func testPhase4AppendCategory(t *testing.T) {
    // 1. record.sh で1件追加 (lesson-learned)
    // 2. knowledge append で error-handling に追記
    // 3. intake processed
    // 検証:
    //   - error-handling/ 内の知識ファイルが増えている
    //   - _category.yaml の last_updated が更新されている
}
```

##### Phase 5: カテゴリ統合
```go
func testPhase5MergeCategories(t *testing.T) {
    // 1. knowledge merge error-handling logging --into observability
    //    - plan file は事前に用意
    // 検証:
    //   - error-handling/ が削除されている
    //   - logging/ が削除されている
    //   - observability/ が存在
    //   - observability/_category.yaml の title が正しい
    //   - knowledge list で2件 (observability, test-conventions)
}
```

##### Phase 6: スキル化とデプロイ
```go
func testPhase6SkillifyAndDeploy(t *testing.T) {
    // 1. prompts/memory/branches/<branch-package-id>/skills/ に
    //    __far-knowledge-observability.md と __far-knowledge-test-conventions.md を配置
    //    (LLM相当の処理をハードコードで実行)
    // 2. tt prompt update を実行
    // 検証:
    //   - .agents/skills/__far-knowledge-observability/SKILL.md が存在
    //   - .agents/skills/__far-knowledge-test-conventions/SKILL.md が存在
    //   - SKILL.md 内に name, description が含まれる
}
```

#### [NEW] [tests/tt/testdata/e2e_far_knowledge/](file://tests/tt/testdata/e2e_far_knowledge/)
*   **Description**: E2E テスト用のテストデータ。
*   **Contents**:
    *   `content_error_handling.md` -- error-handling カテゴリの知識ファイル内容
    *   `content_test_conventions.md` -- test-conventions カテゴリの知識ファイル内容
    *   `content_logging.md` -- logging カテゴリの知識ファイル内容
    *   `content_error_handling_append.md` -- 追記用の知識ファイル内容
    *   `merge_plan.json` -- merge 操作用の計画ファイル
    *   `skill_observability.md` -- スキル化後の capability ファイル
    *   `skill_test_conventions.md` -- スキル化後の capability ファイル

---

## Step-by-Step Implementation Guide

1.  **knowledge ディレクトリの初期化**:
    *   `prompts/memory/knowledge/.gitkeep` を作成

2.  **emitter の ScanBranchSkills 実装 (TDD)**:
    *   `features/tt/internal/prompt/emitter/emitter_test.go` に `TestScanBranchSkills` テストケース追加
    *   `features/tt/internal/prompt/emitter/emitter.go` に `ScanBranchSkills()` を実装

3.  **各エミッターへの統合**:
    *   `codex.go`, `antigravity.go`, `cursor.go`, `claude_code.go` の `Emit()` メソッドに `ScanBranchSkills()` 呼び出しを追加
    *   既存テストを更新して branches/*/skills/ が存在するケースを追加

4.  **E2E テストデータの準備**:
    *   `tests/tt/testdata/e2e_far_knowledge/` にテストデータファイルを作成

5.  **E2E テストの実装**:
    *   `tests/tt/tt_far_knowledge_e2e_test.go` を作成
    *   Phase 0-6 の各テスト関数を実装

6.  **ビルド + テスト実行**:
    *   `scripts/process/build.sh --skip-frontend --skip-etc` でビルド確認
    *   `scripts/process/integration_test.sh --categories "common" --specify "FarKnowledgeE2E"` で E2E テスト実行

7.  **全体統合テスト**:
    *   `scripts/process/build.sh && scripts/process/integration_test.sh` で全テスト実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests (E2E)**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories "common" --specify "FarKnowledgeE2E"
    ```
    *   **Log Verification**:
        *   Phase 0: scaffold が成功し、git init + commit が完了
        *   Phase 1: prompt update でワークフロー/スキルがデプロイされる
        *   Phase 2: 4件の record が全て exit code 0 で成功
        *   Phase 3: 3カテゴリ作成、pending が 0件に
        *   Phase 4: 既存カテゴリに追記成功
        *   Phase 5: merge 後に2カテゴリ
        *   Phase 6: スキルがデプロイされる

3.  **Emitter 単体テスト**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```
    *   emitter パッケージのテストで ScanBranchSkills が正常動作すること

4.  **最終全体検証**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```

### GUI E2E Tests

GUI関連の変更なし。E2E テスト不要。

## Documentation

#### [MODIFY] [005-Far-Knowledge-Skillification.md](file://prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/005-Far-Knowledge-Skillification.md)
*   **更新内容**: 全 Part 完了後に検証結果を追記

#### [NEW] [prompts/memory/knowledge/.gitkeep](file://prompts/memory/knowledge/.gitkeep)
*   **更新内容**: 遠方知識ディレクトリの初期化
