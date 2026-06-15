# 011-Prompt-Source-Alignment-To-Arctic-Tern

## 背景 (Background)

本プロジェクト (tokotachi) の `prompts/manifest/code_content/` 配下のプロンプトソースファイル群は、もともと `axsh/arctic-tern` リポジトリの `.agent/` 配下と同一内容だったが、両リポジトリで独立して進化した結果、差分が生じている。

`arctic-tern` 側はより洗練された記述（冗長な説明の削除、`testing-rules.md` への詳細委譲、構造の簡潔化）に改善されている。一方、tokotachi 側にはメモリ (far-knowledge) 機能に関する独自の追加記述がある。

この仕様は、8ファイルについて arctic-tern 側の記述に寄せつつ、tokotachi 独自のメモリ関連記述を保持する書き換え方針を定義する。

## 要件 (Requirements)

### 必須要件

1. **arctic-tern ベースへの書き換え**: 以下の8ファイルについて、arctic-tern 側の記述をベースとして書き換える
2. **メモリ (far-knowledge) 記述の保持**: tokotachi 側にのみ存在するメモリ関連の記述は書き換え後も保持する
3. **frontmatter の保持**: tokotachi 側の frontmatter (`apiVersion`, `id`, `kind`, `title`, `trigger` 等) は独自のものなので保持する
4. **テンプレート参照の保持**: `{{policy:coding-rules}}` や `{{policy:testing-rules}}` 等のテンプレート変数参照は tokotachi 側の形式を維持する

### 対象ファイルと方針

---

## ファイル 1: project-instructions.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/policies/project-instructions.md` |
| **arctic-tern** | `.agent/rules/instructions.md` |

### 差分分析

**arctic-tern 側にない tokotachi 独自の記述:**

1. **frontmatter**: `apiVersion`, `id`, `kind`, `scope`, `title`, `applies_when` (tokotachi のマニフェストシステム用)
2. **ワークフロー参照パスの違い**: arctic-tern は `.agent/workflows/xxx.md` を参照、tokotachi は `{{policy:xxx}}` テンプレート変数で参照
3. **Git コミットメッセージのクォーティング注意**: tokotachi 側の L177-180 にシングルクォート使用の注意書きがある (Windows PowerShell 経由問題の回避)
4. **テスト順序の記述の違い**: tokotachi 側は「単体テスト → 統合テスト → その他のテスト」(L144) だが arctic-tern 側は「統合テスト → 単体テスト → その他のテスト」(L134)

**両方で同じ内容:**
- 大部分の構造とテキストは同一

### 書き換え方針

- **基本方針**: arctic-tern 側をベースにそのまま使う
- **保持するもの**:
  - tokotachi の frontmatter
  - ルール参照は `{{policy:xxx}}` テンプレート記法に統一 (例: `prompts/rules/coding-rules.md` → `{{policy:coding-rules}}`)
  - Git コミットメッセージのクォーティング注意 (L177-180) を追加
  - テスト順序は「単体テスト → 統合テスト」に修正 (arctic-tern 側の記述は誤記)
  - コミット実行例のシングルクォート使用

```diff
 --- tokotachi project-instructions.md ---

 ## 変更点サマリー:
 - arctic-tern 側のテキストをベースにする
 - frontmatter は tokotachi のものを保持
+- Git操作ルールにクォーティング注意を追加 (tokotachi独自)
 - ルール参照は {{policy:xxx}} テンプレート記法を使用
+- テスト順序を「単体テスト → 統合テスト → その他のテスト」に修正
```

---

## ファイル 2: build-pipeline.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/procedures/build-pipeline.md` |
| **arctic-tern** | `.agent/workflows/build-pipeline.md` |

### 差分分析

**arctic-tern 側の改善点:**

1. **冒頭の簡潔化**: 説明文を短く整理
2. **`{{policy:testing-rules}}` への委譲**: Linux/Remote-SSH の詳細ルールを `{{policy:testing-rules}}` Section 1 への参照に置き換え (tokotachi は本文中にインライン展開)
3. **「準備: ステータスの確認」セクションの削除**: arctic-tern にはない
4. **「Fix Loop」の簡潔化**: `{{policy:testing-rules}}` Section 3 を参照する形に
5. **セクション番号の変更**: 5ステップ構成 (tokotachi は6ステップ)

**tokotachi 独自の記述 (メモリ関連):**
- なし。メモリ関連の追加記述はこのファイルにはない。

### 書き換え方針

- **基本方針**: arctic-tern 側をほぼそのまま採用
- **保持するもの**: tokotachi の frontmatter のみ

```diff
 --- build-pipeline.md 変更方針 ---

-# 6ステップ構成 (準備/Build/Setup/Test/FixLoop/Final)
+# 5ステップ構成 (Build/Setup/Test/FixLoop/Final) - arctic-tern準拠
-# Linux/Remote-SSH の詳細説明をインライン展開
+# {{policy:testing-rules}} への参照に委譲
-# 「準備: ステータスの確認」セクション
+# (削除)
```

---

## ファイル 3: create-implementation-plan.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/procedures/create-implementation-plan.md` |
| **arctic-tern** | `.agent/workflows/create-implementation-plan.md` |

### 差分分析

**arctic-tern 側の改善点:**

1. **ルール参照の形式**: arctic-tern は `prompts/rules/testing-rules.md` (直接パス参照) だが、tokotachi では `{{policy:testing-rules}}` テンプレート記法を使用する
2. **E2E テストセクションのテンプレート**: arctic-tern は Go E2E テスト (`tests/` 配下) に特化した記述を採用
3. **セルフレビューの簡潔化**: arctic-tern 側は GUI E2E 関連チェック (Scenario Consolidation) を持たず、代わりに Go E2E テストコード化チェックがある
4. **`git commit` セクションの追加**: arctic-tern 側の Section 4.1 にドキュメント Git コミットの記述がある (tokotachi にもある)

**tokotachi 独自の記述:**
- `{{policy:xxx}}` テンプレート変数参照形式 (保持する)

### 書き換え方針

- **基本方針**: arctic-tern 側をベースにする
- **保持するもの**:
  - tokotachi の frontmatter
  - ルール参照は `{{policy:xxx}}` テンプレート記法を保持 (arctic-tern の直接パス参照をテンプレート記法に変換)
  - GUI E2E テスト関連のセルフレビュー項目は削除 (arctic-tern に合わせる)
  - E2E テストテンプレートは arctic-tern の Go E2E テスト版を採用

```diff
 --- create-implementation-plan.md 変更方針 ---

 # テンプレート: Verification Plan 内 E2E Tests セクション
-# GUI E2E テスト (Playwright/VSCode Extension) 向けの記述
+# Go E2E テスト (tests/ 配下) 向けの記述 (arctic-tern準拠)

 # セルフレビュー
-# GUI E2Eテスト計画チェック (Scenario Consolidation) の項目群
+# E2Eテストコード化チェック (arctic-tern準拠)

 # ルール参照
-# prompts/rules/testing-rules.md (arctic-tern の直接パス参照)
+# {{policy:testing-rules}} (tokotachi テンプレート記法に変換)
```

---

## ファイル 4: execute-implementation-plan.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/procedures/execute-implementation-plan.md` |
| **arctic-tern** | `.agent/workflows/execute-implementation-plan.md` |

### 差分分析

**arctic-tern 側の改善点:**

1. **全体的な簡潔化**: 冗長な注釈や警告文を削減
2. **`{{policy:testing-rules}}` への委譲**: テスト実施の詳細を `{{policy:testing-rules}}` 参照に
3. **ルール読み込み**: `{{policy:logging-rules}}` も読み込み対象に追加 (arctic-tern から移植済み)
4. **E2E テスト**: arctic-tern は `tests/` 配下の Go E2E テストに特化、tokotachi は GUI E2E テスト (Playwright) に特化
5. **Section 2.5**: arctic-tern に「E2Eテストの実装」セクションが追加されている (テスト実行前にテストコードを実装する指示)

**tokotachi 独自の記述 (メモリ関連):**

- **Section 3.3 「遠方知識の記録 (Far-Knowledge Recording)」**: `git push` の前に record-far-knowledge スキルに従って遠方知識の記録を行う指示。**これは保持する必要がある**。

### 書き換え方針

- **基本方針**: arctic-tern 側をベースにする
- **保持するもの**:
  - tokotachi の frontmatter
  - **Section 3.3「遠方知識の記録」** を Section 3 と Section 4 (Git Push) の間に挿入
  - E2E テストは arctic-tern の Go E2E テスト版を採用
  - `{{policy:logging-rules}}` の読み込みを追加 (arctic-tern から `prompts/manifest/code_content/policies/logging-rules.md` として移植済み)

```diff
 --- execute-implementation-plan.md 変更方針 ---

 # ベース: arctic-tern 版を採用

 # メモリ関連 (tokotachi 独自) を追加:
+## 3.5 遠方知識の記録 (Far-Knowledge Recording)
+
+> [!CAUTION]
+> **省略禁止**: `git push` の前に、このステップを**必ず**実行してください。
+> 遠方知識に該当する変更がない場合でも、判定プロセスは実行し、
+> 「no update」の報告を出してから次に進んでください。
+
+全てのビルドとテストが成功し、コミットが完了した後、`git push` の**前に**、
+**record-far-knowledge** スキルに従って遠方知識の記録を行ってください。
+
+体系化・スキル化は別途 **systematize-far-knowledge** ワークフローで実施します。
```

---

## ファイル 5: investigate.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/procedures/investigate.md` |
| **arctic-tern** | `.agent/workflows/investigate.md` |

### 差分分析

**差分は frontmatter のみ:**
- tokotachi 側に `apiVersion`, `id`, `kind`, `title`, `trigger` の frontmatter がある
- arctic-tern 側は `description` のみ (より具体的な説明文)
- 本文は完全に同一

### 書き換え方針

- **変更なし** (本文は同一)
- frontmatter は tokotachi のものを保持
- `description` は arctic-tern 側の方が具体的なので、description 行のみ更新を検討

---

## ファイル 6: review-point.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/procedures/review-point.md` |
| **arctic-tern** | `.agent/workflows/review-point.md` |

### 差分分析

**arctic-tern 側の改善点:**
- `description` がより具体的

**tokotachi 独自の記述:**
- **Section 4「ドキュメントの Git コミット」** (L33-41): レビュー中に成果物を修正した場合の `git commit` 手順。arctic-tern にはない。

### 書き換え方針

- **基本方針**: arctic-tern 側をベースにする
- **保持するもの**:
  - tokotachi の frontmatter
  - **Section 4「ドキュメントの Git コミット」を保持する**。これはレビューフロー中のコミット忘れを防ぐ有用な記述。

```diff
 --- review-point.md 変更方針 ---

 # ベース: arctic-tern 版を採用

 # tokotachi 独自記述を追加:
+4. **ドキュメントの Git コミット**
+   - レビュー中に成果物を修正した場合は、修正内容を `git add` → `git commit` してください。
+   - コミットメッセージ例: `docs: revise specification XXX-Name per review`
+   - 修正がなかった場合はこのステップをスキップして構いません。
```

---

## ファイル 7: run-all-tests.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/procedures/run-all-tests.md` |
| **arctic-tern** | `.agent/workflows/run-all-tests.md` |

### 差分分析

**arctic-tern 側の改善点:**

1. **全体的な大幅簡潔化**: tokotachi 版は非常に冗長 (224行) だが、arctic-tern 版は簡潔 (115行)
2. **動的カテゴリ発見**: 両方にあるが、arctic-tern はより簡潔な記述
3. **テストファイルの調査 (Section 2.2)**: tokotachi にはあるが arctic-tern では削除
4. **タイムアウト時の分割戦略**: tokotachi にある詳細な分割戦略セクションが arctic-tern では削除
5. **実行コマンド例**: tokotachi の詳細な例が arctic-tern では削除
6. **Mermaid フローチャート**: 両方にあるが arctic-tern はラベルを簡潔化
7. **長時間実行への対応セクション**: tokotachi にのみ存在、arctic-tern では削除
8. **Phase 5**: arctic-tern は「結果レポート & Push」に統合

### 書き換え方針

- **基本方針**: arctic-tern 側をベースにする (大幅に簡潔になる)
- **保持するもの**: tokotachi の frontmatter のみ

```diff
 --- run-all-tests.md 変更方針 ---

-# 224行の冗長版
+# 115行の簡潔版 (arctic-tern準拠)
-# 詳細なタイムアウト分割戦略
+# (削除 - testing-rules.md に委譲)
-# 長時間実行への対応セクション
+# (削除)
-# Phase 5: 結果レポート (Git Push 別セクション)
+# Phase 5: 結果レポート & Push (統合)
```

---

## ファイル 8: test-generator.md

| 項目 | 値 |
|------|-----|
| **tokotachi** | `prompts/manifest/code_content/procedures/test-generator.md` |
| **arctic-tern** | `.agent/workflows/test-generator.md` |

### 差分分析

**arctic-tern 側の改善点:**

1. **全体的な大幅簡潔化**: tokotachi 版は 388行、arctic-tern 版はずっと短い
2. **テンプレート内の記述**: arctic-tern はプレースホルダーを簡潔化 (冗長な例示を削除)
3. **Section 9 の統合**: tokotachi の Section 9.1 (実装手順) + 9.2 (テスト実行手順) が arctic-tern では Section 5「Verification Plan」に統合
4. **セルフレビュー**: arctic-tern は 7 項目に簡潔化 (tokotachi は 8 項目)
5. **セクション番号**: arctic-tern は Section 9-11 → Section 9-11 (番号が1つずつ若い)

**tokotachi 独自の記述 (メモリ関連):**
- なし

### 書き換え方針

- **基本方針**: arctic-tern 側をベースにする
- **保持するもの**: tokotachi の frontmatter のみ

```diff
 --- test-generator.md 変更方針 ---

-# 388行の冗長版
+# 簡潔版 (arctic-tern準拠)
-# テンプレート内の冗長な例示
+# 簡潔なプレースホルダー
-# Section 9.1/9.2 の分離
+# Section 5 Verification Plan に統合
-# Section 10 (テンプレート)
+# Section 9 (テンプレート、番号修正)
```

---

## 変更サマリー

| ファイル | 変更の大きさ | メモリ関連の保持 | 主な変更内容 |
|---------|-------------|-----------------|-------------|
| project-instructions.md | 小 | なし | クォーティング注意を追加、テスト順序修正 |
| build-pipeline.md | 中 | なし | `{{policy:testing-rules}}` への委譲、準備セクション削除 |
| create-implementation-plan.md | 中 | なし | Go E2E テスト版テンプレート採用、`{{policy:xxx}}` 記法保持 |
| execute-implementation-plan.md | 中 | **Section 3.3 遠方知識記録を保持** | 全体簡潔化、`{{policy:logging-rules}}` 追加(移植済み) |
| investigate.md | 極小 | なし | description のみ更新検討 |
| review-point.md | 小 | なし | Git コミットセクションを保持 |
| run-all-tests.md | 大 | なし | 224行 → 115行程度に大幅簡潔化 |
| test-generator.md | 大 | なし | 388行 → 大幅簡潔化 |

## 検証シナリオ (Verification Scenarios)

1. 書き換え後の各ファイルについて、arctic-tern 側の対応ファイルとの `diff` を取り、差分が frontmatter・メモリ関連記述・テンプレート記法変換のみであることを確認する
2. tokotachi 側の `execute-implementation-plan.md` に遠方知識の記録セクション (Section 3.3 相当) が含まれていることを確認する
3. `review-point.md` に Git コミットセクションが含まれていることを確認する
4. 全ファイルの frontmatter が tokotachi 形式 (`apiVersion`, `id`, `kind`, `title`, `trigger`) を保持していることを確認する
5. 全ファイルにおいてルール参照が `{{policy:xxx}}` テンプレート記法で統一されていることを確認する
6. `prompts/manifest/code_content/policies/logging-rules.md` が正しく作成されていることを確認する

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド+単体テスト:
   `scripts/process/build.sh`
   (プロンプトファイルの変更のためビルドには影響しないが、念のため確認)

### 手動検証

- 書き換え後の各ファイルを目視で確認し、意図しない記述の欠落がないことを確認する
- `prompts/manifest/` 配下のマニフェストコンパイルが正常に動作することを確認する
