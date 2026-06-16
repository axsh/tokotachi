# 011-Prompt-Source-Alignment-To-Arctic-Tern

> **Source Specification**: [011-Prompt-Source-Alignment-To-Arctic-Tern.md](file://prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/011-Prompt-Source-Alignment-To-Arctic-Tern.md)

## Goal Description

`prompts/manifest/code_content/` 配下の8つのプロンプトソースファイルを、`axsh/arctic-tern` リポジトリの `.agent/` 配下の対応ファイルに寄せて書き換える。arctic-tern 側のより簡潔で洗練された記述を採用しつつ、tokotachi 独自の以下の要素を保持する:

- frontmatter (マニフェストシステム用)
- `{{policy:xxx}}` テンプレート記法
- 遠方知識 (far-knowledge) 関連の記述
- review-point の Git コミットセクション
- Git コミットメッセージのクォーティング注意

## User Review Required

> [!IMPORTANT]
> 本計画はソースコードの変更を含まず、プロンプトファイル (マークダウン) のみの書き換えです。
> ビルドやテストへの影響はありませんが、エージェントの振る舞いに影響するため内容の確認をお願いします。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| arctic-tern ベースへの書き換え (8ファイル) | Proposed Changes > 全ファイル |
| メモリ (far-knowledge) 記述の保持 | Proposed Changes > execute-implementation-plan.md |
| frontmatter の保持 | 全ファイルで保持 (各ファイルの変更方針に記載) |
| `{{policy:xxx}}` テンプレート記法の統一 | 全ファイルでルール参照をテンプレート記法に変換 |
| テスト順序「単体テスト → 統合テスト」に修正 | Proposed Changes > project-instructions.md |
| review-point の Git コミットセクション保持 | Proposed Changes > review-point.md |
| logging-rules.md の移植 | 既に完了済み (前回のレビューで対応) |

## Proposed Changes

### プロンプトソースファイル群

以下、各ファイルの変更内容を記述する。変更のベースは `prompts/phases/000-foundation/refs/repos/arctic-tern/` 配下の対応ファイル。

---

#### [MODIFY] [project-instructions.md](file://prompts/manifest/code_content/policies/project-instructions.md)

*   **Description**: arctic-tern の `instructions.md` をベースに書き換え。tokotachi 独自の frontmatter、クォーティング注意、テスト順序修正を保持。
*   **ベースファイル**: [instructions.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/rules/instructions.md)
*   **変更内容**:
    1. arctic-tern 版の本文をそのまま採用
    2. frontmatter は tokotachi 版を保持:
       ```yaml
       ---
       activation:
           mode: trigger
       apiVersion: agent.meta/v1
       id: project-instructions
       kind: policy
       scope: project
       title: Project Instructions
       applies_when: Applies when starting work, understanding project workflows, or running scripts
       ---
       ```
    3. 2つ目の frontmatter (`trigger: always_on`) は削除 (tokotachi のマニフェストシステムが activation を管理するため不要)
    4. ルール参照を `{{policy:xxx}}` テンプレート記法に変換:
       - `prompts/rules/coding-rules.md` → `{{policy:coding-rules}}`
       - `prompts/rules/testing-rules.md` → `{{policy:testing-rules}}`
    5. Git 操作ルールにクォーティング注意を追加 (L177-180 相当):
       ```markdown
       > [!IMPORTANT]
       > **コミットメッセージのクォーティング**: Windows 環境では、PowerShell 経由で bash を呼び出す際にコミットメッセージが途切れる問題があります。
       > **必ず `-m` の引数にはシングルクォートを使用**してください（例: `git commit -m 'feat: add feature'`）。
       > Mac / Linux 環境ではこの問題は発生しませんが、統一のためシングルクォートの使用を推奨します。
       ```
    6. テスト順序を「単体テスト → 統合テスト → その他のテスト」に修正 (arctic-tern の「統合テスト → 単体テスト」は誤記)
    7. コミット実行例のクォートをシングルクォートに統一:
       ```bash
       git commit -m 'feat: implement rate limiter struct'
       ```

---

#### [MODIFY] [build-pipeline.md](file://prompts/manifest/code_content/procedures/build-pipeline.md)

*   **Description**: arctic-tern の `build-pipeline.md` をベースに書き換え。大幅に簡潔化。
*   **ベースファイル**: [build-pipeline.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/workflows/build-pipeline.md)
*   **変更内容**:
    1. arctic-tern 版の本文 (59行) をそのまま採用
    2. frontmatter は tokotachi 版を保持:
       ```yaml
       ---
       apiVersion: agent.meta/v1
       id: build-pipeline
       kind: procedure
       title: Build, Test, and Verify Pipeline
       trigger:
           command: build-pipeline
       ---
       ```
    3. 2つ目の frontmatter (`description: ...`) は削除
    4. arctic-tern の `prompts/rules/testing-rules.md` 参照を `{{policy:testing-rules}}` に変換
    5. 削除される要素:
       - 「準備: ステータスの確認」セクション (Section 1)
       - Linux/Remote-SSH のインライン詳細説明 (testing-rules.md に委譲)
       - 「Analyze Results & Feedback Loop」の詳細手順 (testing-rules.md に委譲)
       - 「Git Push」の独立セクション (「Final Check & Push」に統合)

---

#### [MODIFY] [create-implementation-plan.md](file://prompts/manifest/code_content/procedures/create-implementation-plan.md)

*   **Description**: arctic-tern の `create-implementation-plan.md` をベースに書き換え。
*   **ベースファイル**: [create-implementation-plan.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/workflows/create-implementation-plan.md)
*   **変更内容**:
    1. arctic-tern 版の本文をそのまま採用
    2. frontmatter は tokotachi 版を保持
    3. 2つ目の frontmatter は削除
    4. ルール参照を `{{policy:xxx}}` テンプレート記法に変換:
       - `prompts/rules/testing-rules.md` → `{{policy:testing-rules}}`
       - `prompts/rules/planning-rules.md` → `{{policy:planning-rules}}`
    5. E2E テストテンプレート: arctic-tern の Go E2E テスト版を採用 (`tests/` 配下)
    6. セルフレビュー: GUI E2E 関連チェック (Scenario Consolidation) を削除し、Go E2E テストコード化チェックを採用

---

#### [MODIFY] [execute-implementation-plan.md](file://prompts/manifest/code_content/procedures/execute-implementation-plan.md)

*   **Description**: arctic-tern の `execute-implementation-plan.md` をベースに書き換え。遠方知識記録セクションを保持。
*   **ベースファイル**: [execute-implementation-plan.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/workflows/execute-implementation-plan.md)
*   **変更内容**:
    1. arctic-tern 版の本文をそのまま採用
    2. frontmatter は tokotachi 版を保持
    3. 2つ目の frontmatter は削除
    4. ルール参照を `{{policy:xxx}}` テンプレート記法に変換:
       - `prompts/rules/coding-rules.md` → `{{policy:coding-rules}}`
       - `prompts/rules/testing-rules.md` → `{{policy:testing-rules}}`
       - `prompts/rules/logging-rules.md` → `{{policy:logging-rules}}`
    5. **遠方知識の記録 (Far-Knowledge Recording) セクションを追加** (Section 3 と Section 4 の間):
       ```markdown
       ### 3.3 遠方知識の記録 (Far-Knowledge Recording)

       > [!CAUTION]
       > **省略禁止**: `git push` の前に、このステップを**必ず**実行してください。
       > 遠方知識に該当する変更がない場合でも、判定プロセスは実行し、
       > 「no update」の報告を出してから次に進んでください。

       全てのビルドとテストが成功し、コミットが完了した後、`git push` の**前に**、
       **record-far-knowledge** スキルに従って遠方知識の記録を行ってください。

       体系化・スキル化は別途 **systematize-far-knowledge** ワークフローで実施します。
       ```
    6. E2E テスト: arctic-tern の Go E2E テスト版 (Section 2.5) を採用

---

#### [MODIFY] [investigate.md](file://prompts/manifest/code_content/procedures/investigate.md)

*   **Description**: frontmatter のみ変更。本文は同一のため変更不要。
*   **ベースファイル**: [investigate.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/workflows/investigate.md)
*   **変更内容**:
    1. 本文は変更なし (arctic-tern と同一)
    2. frontmatter は tokotachi 版を保持

---

#### [MODIFY] [review-point.md](file://prompts/manifest/code_content/procedures/review-point.md)

*   **Description**: arctic-tern の `review-point.md` をベースに書き換え。Git コミットセクションを保持。
*   **ベースファイル**: [review-point.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/workflows/review-point.md)
*   **変更内容**:
    1. arctic-tern 版の本文をそのまま採用
    2. frontmatter は tokotachi 版を保持
    3. 2つ目の frontmatter は削除
    4. **Git コミットセクション (Section 4) を保持**:
       ```markdown
       4. **ドキュメントの Git コミット**
          - レビュー中に成果物（仕様書、実装計画書など）を修正した場合は、修正内容を `git add` → `git commit` してください。
          - コミットメッセージ例: `docs: revise specification XXX-Name per review`, `docs: update implementation plan YYY-Name per review`
          - 修正がなかった場合はこのステップをスキップして構いません。
          ```bash
          git add <修正したドキュメントファイル>
          git commit -m 'docs: revise <対象ドキュメント名> per review'
          ```
       ```

---

#### [MODIFY] [run-all-tests.md](file://prompts/manifest/code_content/procedures/run-all-tests.md)

*   **Description**: arctic-tern の `run-all-tests.md` をベースに書き換え。224行 → 115行程度に大幅簡潔化。
*   **ベースファイル**: [run-all-tests.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/workflows/run-all-tests.md)
*   **変更内容**:
    1. arctic-tern 版の本文をそのまま採用
    2. frontmatter は tokotachi 版を保持
    3. 2つ目の frontmatter は削除
    4. 削除される要素:
       - テストファイルの調査セクション (Section 2.2)
       - タイムアウト時の分割戦略セクション
       - 実行コマンド例の詳細セクション
       - 長時間実行への対応セクション
       - Mermaid フローチャートの冗長なラベル (簡潔化)
       - Git Push の独立セクション (Phase 5 に統合)

---

#### [MODIFY] [test-generator.md](file://prompts/manifest/code_content/procedures/test-generator.md)

*   **Description**: arctic-tern の `test-generator.md` をベースに書き換え。大幅簡潔化。
*   **ベースファイル**: [test-generator.md](file://prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/workflows/test-generator.md)
*   **変更内容**:
    1. arctic-tern 版の本文をそのまま採用
    2. frontmatter は tokotachi 版を保持
    3. 2つ目の frontmatter は削除
    4. ルール参照を `{{policy:xxx}}` テンプレート記法に変換:
       - `prompts/rules/testing-rules.md` → `{{policy:testing-rules}}`
    5. 削除される要素:
       - テンプレート内の冗長な例示
       - Section 9.1/9.2 の分離 (Section 5 Verification Plan に統合)
       - セルフレビューの冗長な記述 (簡潔化)

## Step-by-Step Implementation Guide

> [!NOTE]
> 各ステップでは「arctic-tern 版の本文をコピー → frontmatter 差し替え → テンプレート記法変換 → tokotachi 独自記述の追加」の手順で進める。

1.  **[x] Step 1: project-instructions.md の書き換え**
    *   `prompts/phases/000-foundation/refs/repos/arctic-tern/.agent/rules/instructions.md` の本文を取得
    *   tokotachi の frontmatter を先頭に配置 (2つ目の frontmatter は削除)
    *   ルール参照を `{{policy:xxx}}` に変換
    *   クォーティング注意を Git 操作ルールに追加
    *   テスト順序を「単体テスト → 統合テスト → その他のテスト」に修正
    *   コミット実行例をシングルクォートに修正
    *   `git add && git commit`

2.  **[x] Step 2: build-pipeline.md の書き換え**
    *   arctic-tern 版をコピーし、frontmatter 差し替え、テンプレート記法変換
    *   `git add && git commit`

3.  **[x] Step 3: create-implementation-plan.md の書き換え**
    *   arctic-tern 版をコピーし、frontmatter 差し替え、テンプレート記法変換
    *   `git add && git commit`

4.  **[x] Step 4: execute-implementation-plan.md の書き換え**
    *   arctic-tern 版をコピーし、frontmatter 差し替え、テンプレート記法変換
    *   遠方知識の記録セクションを追加 (Section 3 と Section 4 の間)
    *   `git add && git commit`

5.  **[x] Step 5: investigate.md の確認** (本文同一のため変更なし)
    *   本文が同一であることを確認。変更不要であればスキップ。
    *   変更があれば `git add && git commit`

6.  **[x] Step 6: review-point.md の書き換え**
    *   arctic-tern 版をコピーし、frontmatter 差し替え
    *   Git コミットセクション (Section 4) を追加
    *   `git add && git commit`

7.  **[x] Step 7: run-all-tests.md の書き換え**
    *   arctic-tern 版をコピーし、frontmatter 差し替え
    *   `git add && git commit`

8.  **[x] Step 8: test-generator.md の書き換え**
    *   arctic-tern 版をコピーし、frontmatter 差し替え、テンプレート記法変換
    *   `git add && git commit`

9.  **[x] Step 9: 最終差分確認**
    *   各ファイルについて arctic-tern 版との `diff` を取り、差分が意図通りであることを確認
    *   全ファイルの frontmatter、テンプレート記法、独自セクションが正しいことを確認

10. **[x] Step 10: Verification Plan の実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests** (プロンプトファイルの変更のためビルドには影響しないが、念のため確認):
    ```bash
    ./scripts/process/build.sh
    ```

> [!NOTE]
> 本計画はプロンプトファイル (マークダウン) の書き換えのみであり、Go コードの変更を含まないため、統合テストの実行は不要です。ビルドの成功をもって検証完了とします。

### 差分検証

各ファイルについて arctic-tern 版との `diff` を取得し、以下を確認する:

1. 差分が frontmatter、テンプレート記法変換 (`prompts/rules/xxx.md` → `{{policy:xxx}}`)、tokotachi 独自記述のみであること
2. `execute-implementation-plan.md` に遠方知識の記録セクションが含まれていること
3. `review-point.md` に Git コミットセクションが含まれていること
4. `project-instructions.md` のテスト順序が「単体テスト → 統合テスト」であること
5. `project-instructions.md` にクォーティング注意が含まれていること

## Documentation

変更対象は仕様書・プロンプトファイル自体であるため、追加のドキュメント更新は不要。
