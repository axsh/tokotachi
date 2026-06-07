# 002-Integration-Test-Guardrails

> **Source Specification**: [002-Integration-Test-Guardrails.md](file:///prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/002-Integration-Test-Guardrails.md)

## Goal Description

統合テストが計画・作成・実行されずにスキップされる構造的問題を解消するため、プロンプトテンプレート3ファイルにガードレールを追加する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: 統合テスト必要性の判断基準を明文化 | Proposed Changes > planning-rules.md (2.1 セクション) |
| R2: 統合テスト省略禁止の CAUTION を追加 | Proposed Changes > execute-implementation-plan.md (3.1 Step 2) |
| R3: 計画テンプレートの統合テスト項目を条件付き必須にする | Proposed Changes > create-implementation-plan.md (テンプレート) |
| R4: `build.sh` の最終検証制約を明記 | Proposed Changes > planning-rules.md (3.3 セクション) |
| R5: セルフレビューに統合テスト存在確認を追加 | Proposed Changes > create-implementation-plan.md (5. セルフレビュー) |

## Proposed Changes

### prompts/manifest/code_content/policies

#### [MODIFY] [planning-rules.md](file:///prompts/manifest/code_content/policies/planning-rules.md)

*   **Description**: R1 (統合テスト判断基準) と R4 (最終検証制約) を追加

*   **R1: L48 の後に統合テスト必要性判断基準を挿入**

    現在の L46-49:
    ```markdown
    *   **テスト計画 (Unit & Integration)**:
        *   **Unit Tests**: テーブル駆動テスト (`tests := []struct{...}`) のケース設計。モックが必要な依存関係（DB, 外部API）。
        *   **Integration Tests**: `tests/` 配下のどのファイルに追加するか。実際のDBやDockerコンテナとの連携確認手順。
        *   記述順序: `Proposed Changes` では必ず `_test.go` を先に記述してください。
    ```

    L48 (`Integration Tests` の行) の後に以下を挿入:

    ```markdown
        *   **統合テスト必要性の判断基準**: 以下のいずれかに該当する場合、統合テストは**必須**です。
            *   ファイルシステムへの書き込み/読み取りを行う
            *   SQLite/DB 操作を行う
            *   外部コマンド (git 等) を呼び出す
            *   CLI サブコマンドとして利用者に提供される
            *   上記のいずれにも該当しない純粋なロジックのみの場合は、単体テストのみで可。ただし Verification Plan にその理由を明記すること。
    ```

*   **R4: L118 の後に最終検証制約の WARNING を挿入**

    現在の L117-118 (ファイル末尾):
    ```markdown
        *   **Required Commands**:
            *   ✅ `./scripts/process/build.sh && ./scripts/process/integration_test.sh`
    ```

    L118 の後に以下を追加:

    ```markdown

        > [!WARNING]
        > **Partial Build Flags in Final Verification**:
        > `--backend-only`, `--skip-frontend`, `--skip-etc` は開発中の高速フィードバック用です。
        > Verification Plan の最終検証コマンドには `./scripts/process/build.sh` (フラグなし) を使用してください。
    ```

---

### prompts/manifest/code_content/procedures

#### [MODIFY] [execute-implementation-plan.md](file:///prompts/manifest/code_content/procedures/execute-implementation-plan.md)

*   **Description**: R2 (統合テスト省略禁止の CAUTION) を追加

*   **R2: L59 の「統合テストの実施」セクション先頭に CAUTION を挿入**

    現在の L59-60:
    ```markdown
    2.  **統合テストの実施**:
        *   Step 1が成功した場合のみ、統合テストを実行します。
    ```

    L59 の後、L60 の前に以下を挿入:

    ```markdown

        > [!CAUTION]
        > **統合テストの省略禁止**: 実装計画の Verification Plan に `integration_test.sh` の実行コマンドが記載されている場合、**テストコードの作成と実行の両方を省略してはならない**。
        > 計画に記載された統合テストファイルが存在しない場合は、先に作成してから実行すること。
        > 「単体テストで通っているから」「変更が小さいから」は省略の理由にならない。

    ```

---

#### [MODIFY] [create-implementation-plan.md](file:///prompts/manifest/code_content/procedures/create-implementation-plan.md)

*   **Description**: R3 (テンプレートの統合テスト条件付き必須) と R5 (セルフレビュー項目追加) を適用

*   **R3: L115 のテンプレート内「Integration Tests」ヘッダーを条件付き必須に変更**

    現在の L115-120:
    ```markdown
    2.  **Integration Tests**:
        Run integration tests.
        ```bash
        ./scripts/process/integration_test.sh --specify "[Unique Test Case Name]"
        ```
        *   **Log Verification**: [ログで何を確認すべきか具体的に記述]
    ```

    以下に変更:
    ```markdown
    2.  **Integration Tests** (ファイルI/O, DB, 外部コマンド, CLI を含む場合は必須):

        > [!CAUTION]
        > 本実装がファイルI/O、DB操作、外部コマンド呼び出し、CLI サブコマンドのいずれかを含む場合、
        > **統合テストの計画を省略してはならない**。
        > 統合テストが不要な場合は、その理由をこのセクションに明記すること。

        Run integration tests.
        ```bash
        ./scripts/process/integration_test.sh --specify "[Unique Test Case Name]"
        ```
        *   **Log Verification**: [ログで何を確認すべきか具体的に記述]
    ```

*   **R5: L185 (テスト網羅性チェック) の後にセルフレビュー項目を追加**

    現在の L183-189:
    ```markdown
    4.  **テスト網羅性チェック (Platform Specific)**:
        *   (Go) 単体テストと統合テストが計画されているか。また単体か統合かについて、テスト内容による区分けは適切か。
        *   TDDで計画されているか。
    5.  **統合テストの実行プランチェック**:
        *   `./scripts/process/integration_test.sh` は全てを実行すると非常に長い時間がかかりますので、関係のあるテストを選択的に実行すべきです。
            *   `--categories` 及び `--specify` を組み合わせたテスト実行コマンドを必ず明記してあるか。
        *   テスト範囲が適切かどうか、テストシナリオなどを分析して検証すること。
    ```

    L185 の後に以下を挿入 (既存 5. は 6. に繰り下げ、以降の番号もすべて +1):
    ```markdown
    5.  **統合テスト必要性チェック (Backend)**:
        *   (Go) 本実装がファイルI/O、DB操作、外部コマンド呼び出し、CLI サブコマンドのいずれかを含む場合、**統合テストが Verification Plan に含まれているか**。
        *   含まない場合、その理由が Verification Plan に明記されているか。
    ```

    結果として番号は:
    - 4. テスト網羅性チェック (Platform Specific) -- 変更なし
    - 5. **統合テスト必要性チェック (Backend)** -- 新規追加
    - 6. 統合テストの実行プランチェック -- 旧5
    - 7. GUI E2Eテスト計画チェック -- 旧6
    - 8. テスト項目設計のセルフレビュー -- 旧7
    - 9. 総合判定プロセスの計画 -- 旧8

## Step-by-Step Implementation Guide

### Step 1: R1 + R4 - planning-rules.md の修正

1. `prompts/manifest/code_content/policies/planning-rules.md` を編集:
    - L48 の後に統合テスト必要性の判断基準 (5項目) を挿入
    - L118 の後に Partial Build Flags の WARNING を追加
2. `git add && git commit`

### Step 2: R2 - execute-implementation-plan.md の修正

1. `prompts/manifest/code_content/procedures/execute-implementation-plan.md` を編集:
    - L59 の後に統合テスト省略禁止の CAUTION を挿入
2. `git add && git commit`

### Step 3: R3 + R5 - create-implementation-plan.md の修正

1. `prompts/manifest/code_content/procedures/create-implementation-plan.md` を編集:
    - L115-120 のテンプレート内 Integration Tests セクションを条件付き必須に変更
    - L185 の後にセルフレビュー項目を挿入し、後続の番号を繰り下げ
2. `git add && git commit`

### Step 4: プロンプトコンパイルとデプロイ確認

1. Verification Plan を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Prompt Compile & Deploy**:
    ```bash
    ./bin/tt.exe prompt compile --apply
    ```

3.  **デプロイ先の反映確認**:
    ```bash
    # R1: planning-rules.md に判断基準が含まれるか
    grep -c "統合テスト必要性の判断基準" .agent/rules/planning-rules.md

    # R4: planning-rules.md に Partial Build Flags が含まれるか
    grep -c "Partial Build Flags" .agent/rules/planning-rules.md

    # R2: execute-implementation-plan に省略禁止 CAUTION が含まれるか
    grep -c "統合テストの省略禁止" .agents/skills/execute-implementation-plan/SKILL.md

    # R3: create-implementation-plan に条件付き必須が含まれるか
    grep -c "ファイルI/O, DB, 外部コマンド, CLI" .agents/skills/create-implementation-plan/SKILL.md

    # R5: セルフレビューに統合テスト必要性チェックが含まれるか
    grep -c "統合テスト必要性チェック" .agents/skills/create-implementation-plan/SKILL.md
    ```
    全て `1` 以上が返ることを確認。

本仕様はドキュメント (プロンプトテンプレート) のみの変更であり、Go/TypeScript コードへの変更は発生しない。統合テストの対象外 (理由: コード変更なし、プロンプトテンプレートのマークダウン修正のみ)。

### テスト項目セルフレビュー

**1. 網羅性**: R1-R5 の全要件が Proposed Changes に対応しており、デプロイ先への反映を grep で検証する。

**2. 証拠の十分性**: 各 grep コマンドが固有のキーワードを検索するため、追加した文言が確実に含まれていることを確認できる。

**3. 迂回排除**: `prompt compile --apply` を通してデプロイされた先のファイルを検証するため、テンプレート修正がデプロイに反映されないケースも検出可能。

**4. 依存関係**: テンプレート修正 -> compile -> deploy の順でボトムアップに検証。

## Documentation

本計画で影響を受ける `prompts/specifications` 配下のドキュメントはなし。
