# 統合テスト強制ガードレールの追加

## 背景 (Background)

調査レポート ([investigation-integration-test-gap.md](file:///C:/Users/yamya/.gemini/antigravity-ide/brain/7976e93e-7fc0-4c5d-98a0-b386d07619c2/investigation-integration-test-gap.md)) により、統合テストが作成・実行されない3つの構造的原因が特定された:

1. **計画テンプレートの統合テスト項目がオプショナルに読める**: `create-implementation-plan` テンプレートの Verification Plan セクションに統合テストの項目はあるが、「いつ必須でいつ省略可か」の判断基準がない。結果としてAIが「この機能には統合テストは不要」と自己判断して省略できる。
2. **実行ワークフローが計画の Verification Plan に完全依存**: `execute-implementation-plan` は計画に記載されたテストのみを実行する設計。計画で統合テストが省略されると、実行時にも合法的にスキップされる。GUI E2E テストには「省略禁止」の CAUTION があるが、Go統合テストには同等のガードレールがない。
3. **`build.sh --backend-only` の抜け道**: planning-rules では `build.sh && integration_test.sh` の連結実行が推奨されているが、最終検証で `build.sh --backend-only` のみが使われ、連結が切られた。

### 影響

- 000 Part1/Part2 の実装計画には統合テスト (`tests/common/`) が記載されていたが、実際には作成されなかった
- 001 の実装計画では統合テストの記載自体が省略された
- 単体テストのみで検証が完了しているため、エンドツーエンドの動作保証がない

## 要件 (Requirements)

### R1: 統合テスト必要性の判断基準を明文化 (planning-rules)

`planning-rules.md` の 2.1 Backend Development セクションに、統合テストが必要なケースの判断基準を追加する。

判断基準:
- ファイルシステムへの書き込み/読み取りを行う -> 統合テスト必要
- SQLite/DB 操作を行う -> 統合テスト必要
- 外部コマンド (git 等) を呼び出す -> 統合テスト必要
- CLI サブコマンドとして利用者に提供される -> 統合テスト必要
- 上記のいずれにも該当しない純粋なロジック -> 単体テストのみで可

統合テストを省略する場合は、その理由を Verification Plan に明記することを義務付ける。

### R2: 統合テスト省略禁止の CAUTION を追加 (execute-implementation-plan)

`execute-implementation-plan.md` の 3.1 セクションの「統合テストの実施」(Step 2) に、GUI E2E テストと同等の省略禁止 CAUTION を追加する。

追加すべき内容:
```
> [!CAUTION]
> **統合テストの省略禁止**: 実装計画の Verification Plan に
> `integration_test.sh` の実行コマンドが記載されている場合、
> **テストコードの作成と実行の両方を省略してはならない**。
> 計画に記載された統合テストファイルが存在しない場合は、
> 先に作成してから実行すること。
```

### R3: 計画テンプレートの統合テスト項目を条件付き必須にする (create-implementation-plan)

`create-implementation-plan.md` の Verification Plan テンプレート内の「Integration Tests」項目に、条件付き必須であることを明記する。

変更内容:
- 現在の `2. **Integration Tests**:` のヘッダーに「(ファイルI/O, DB, 外部コマンド, CLI を含む場合は必須)」を追加
- 統合テストが不要な場合は理由を明記する旨の注釈を追加

### R4: `build.sh` の最終検証制約を明記 (planning-rules)

`planning-rules.md` の 3.3 セクションに、最終検証での `--backend-only` 使用禁止を明記する。

追加すべき内容:
```
> [!WARNING]
> **Partial Build Flags in Final Verification**:
> `--backend-only`, `--skip-frontend`, `--skip-etc` は開発中の高速フィードバック用です。
> Verification Plan の最終検証コマンドには `./scripts/process/build.sh` (フラグなし)
> を使用してください。
```

### R5: セルフレビューに統合テスト存在確認を追加 (create-implementation-plan)

`create-implementation-plan.md` のセルフレビューチェック項目 (4番と5番の間) に、統合テストの必要性判定チェックを追加する。

追加内容:
```
*   (Go) 本実装が ファイルI/O, DB操作, 外部コマンド呼び出し, CLI サブコマンド
    のいずれかを含む場合、**統合テストが Verification Plan に含まれているか**。
    含まない場合、その理由が明記されているか。
```

## 実現方針 (Implementation Approach)

修正対象はすべて `prompts/manifest/code_content/` 配下のテンプレートファイル。テンプレートを修正後、`tt prompt compile --apply` でデプロイ先 (`.agent/rules/`, `.agents/skills/`) に反映する。

### 修正対象ファイル

| 要件 | テンプレートファイル |
|---|---|
| R1, R4 | `prompts/manifest/code_content/policies/planning-rules.md` |
| R2 | `prompts/manifest/code_content/procedures/execute-implementation-plan.md` |
| R3, R5 | `prompts/manifest/code_content/procedures/create-implementation-plan.md` |

### 修正方針

- 既存の構造を維持し、必要な箇所にテキストを挿入する
- GUI E2E テストで既に確立されているパターン (CAUTION ブロック) を流用する
- 変更はすべてドキュメント (マークダウン) のみ。コード変更なし。

## 検証シナリオ (Verification Scenarios)

1. テンプレート修正後、`./bin/tt.exe prompt compile --apply` を実行し、エラーなくコンパイルされることを確認
2. デプロイ先のファイルに修正内容が反映されていることを目視確認:
   - `.agent/rules/planning-rules.md` に R1, R4 の内容が含まれるか
   - `.agents/skills/execute-implementation-plan/SKILL.md` に R2 の CAUTION が含まれるか
   - `.agents/skills/create-implementation-plan/SKILL.md` に R3, R5 の内容が含まれるか

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```

2. プロンプトコンパイル:
   ```
   ./bin/tt.exe prompt compile --apply
   ```

本仕様はドキュメント (プロンプトテンプレート) のみの変更であり、Go/TypeScript コードへの変更は発生しない。統合テストの対象外。
