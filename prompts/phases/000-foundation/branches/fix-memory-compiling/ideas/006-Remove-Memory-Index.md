# index.md の完全削除

## 背景 (Background)

`prompts/memory/index.md` は、旧メモリシステムにおいて「メモリ文書のルーティングテーブル」として設計された自動生成ファイルである。`tt prompt compile` 実行時に、`prompts/memory/` 以下のマークダウンファイルのフロントマターを走査し、ID・ステータス・トピック等のメタデータを集約したインデックスを生成していた。

しかし、005-Far-Knowledge-Skillification の実装により、知識の蓄積先が `prompts/memory/knowledge/` 階層に移行した。現在の `index.md` は以下の状態にある:

- **ルーティングテーブルが空**: メモリ文書が0件であり、Coding Agent が読んでも有用な情報が一切ない
- **フォールバック先が存在しない**: 参照している `inbox.md` や `open-questions.md` は現在のプロジェクトに存在しない
- **更新ルールが旧体系を前提**: `deploy.sh --force` を案内するなど、旧ワークフローを前提とした記述になっている
- **knowledge/ のインデックスは不要**: `tt agent knowledge list` コマンドや `find` による再帰走査で代替可能
- **ダイジェスト計算に副作用**: `deploy.go` が compile 後にダイジェストを再計算する理由が「index.md の生成がソースディレクトリを変更するため」であり、これ自体が不要な複雑性

## 要件 (Requirements)

### 必須要件

1. **ファイル削除**: `prompts/memory/index.md` を削除する
2. **Go コード削除**: 以下のコードを削除または修正する
   - `features/tt/internal/prompt/memory/indexer.go` 全体を削除
   - `features/tt/internal/prompt/memory/indexer_test.go` が存在すれば削除
   - `features/tt/internal/prompt/compiler/compiler.go` から index.md 生成ロジック (Step 10, Step 12 の index 書き出し部分) を除去
   - `features/tt/internal/prompt/compiler/compiler.go` の `CompileResult.IndexContent` フィールドを削除
   - `features/tt/internal/prompt/compiler/deploy.go` の L96-97 のコメント修正 (index.md 言及の除去)
   - `features/tt/internal/prompt/compiler/compiler_test.go` から index.md 関連のアサーションを除去
3. **設定削除**: `prompts/manifest/project.yaml` から `memory_index` 出力設定を削除
4. **設定型修正**: `features/tt/internal/prompt/manifest/types.go` の `Outputs.MemoryIndex` フィールドを削除
5. **テンプレート変数削除**: `features/tt/internal/prompt/emitter/template.go` の `memory` kind 解決ロジックを削除 (L58-59)
6. **テンプレートテスト修正**: `features/tt/internal/prompt/emitter/template_test.go` の `{{memory:index}}` テストケースを削除
7. **ガード削除**: `prompts/manifest/safety/guards/deny-direct-edit-of-index.yaml` を削除
8. **ワークフロー参照除去**: 以下のファイルから `{{memory:index}}` や `prompts/memory/index.md` への参照を削除する。「メモリの確認」ステップは代替なしで完全削除する
   - `prompts/manifest/code_content/procedures/execute-implementation-plan.md` -- Section 1.3「メモリの確認」ステップ (L25-27) を完全削除
   - `prompts/manifest/code_content/procedures/create-specification.md` -- Section 1.2「メモリの確認」ステップ (L23-25) を完全削除
   - `prompts/manifest/code_content/policies/far-knowledge-memory.md` (L12) -- `prompts/memory/index.md` への言及を削除
   - `prompts/manifest/code_content/capabilities/record-far-knowledge.md` (references 欄) -- 参照除去
   - `prompts/manifest/code_content/capabilities/pre-push-knowledge-check.md` (references 欄) -- 参照除去
   - 注: `create-implementation-plan.md` には既に `{{memory:*}}` 参照が存在しないため対象外
9. **prompt update 後のデプロイ確認**: 変更後に `tt prompt update` が正常に完了すること

### 任意要件

- `memory_docs` ソースパターン (`prompts/memory/**/*.md`) の扱いを検討する。README.md や knowledge/ 以下の md ファイルが不要にスキャンされるなら除去する

## 実現方針 (Implementation Approach)

### 削除対象の整理

```
削除するファイル:
  prompts/memory/index.md
  features/tt/internal/prompt/memory/indexer.go
  features/tt/internal/prompt/memory/indexer_test.go (存在する場合)
  prompts/manifest/safety/guards/deny-direct-edit-of-index.yaml

修正するファイル:
  features/tt/internal/prompt/compiler/compiler.go      (index生成ロジック除去)
  features/tt/internal/prompt/compiler/compiler_test.go  (indexアサーション除去)
  features/tt/internal/prompt/compiler/deploy.go         (コメント修正)
  features/tt/internal/prompt/manifest/types.go          (MemoryIndex フィールド削除)
  features/tt/internal/prompt/emitter/template.go        (memory kind 削除)
  features/tt/internal/prompt/emitter/template_test.go   (テストケース削除)
  prompts/manifest/project.yaml                          (memory_index 削除)
  prompts/manifest/code_content/procedures/*.md          (参照除去, 3ファイル)
  prompts/manifest/code_content/policies/far-knowledge-memory.md (参照除去)
  prompts/manifest/code_content/capabilities/*.md        (参照除去, 2ファイル)
```

### 修正方針

- `compiler.go` の Step 10 (GenerateIndex) と Step 12 の index 書き出しを丸ごと削除する。Step の番号をリナンバリングする
- `template.go` の `memory` case は、今後 knowledge ベースの参照に発展させる可能性があるため、削除ではなく `// TODO: remove or repurpose for knowledge` コメント付きで残す選択肢もあるが、YAGNI 原則により現時点では削除する
- ワークフローの「メモリの確認」ステップ (L25-27 相当) は、`prompts/memory/README.md` を読むよう書き換えるか、あるいは knowledge list で代替する
- `deploy.go` のダイジェスト再計算 (L96-103) は index.md 生成による副作用が唯一の理由であったため、不要になる可能性がある。ただし他の compile 生成物 (resolved manifest) が source ディレクトリに書かれるケースもあり得るので、コメントのみ修正し、再計算ロジック自体は残す

## 懸念事項

### 低リスク: memory_docs ソースパターンの影響 (本仕様の対象外)

`project.yaml` の `memory_docs: prompts/memory/**/*.md` は、旧メモリ文書 (current.md, decisions.md, invariants.md 等) を走査対象としていた。今回 `README.md` を追加したことで、README.md がメモリ文書としてパースされる可能性がある (フロントマターが無ければパースエラーにはならず無視されるが、不要な走査は行われる)。

index.md を廃止する場合、`memory_docs` パターン自体も削除して問題ないか検討する。現在メモリ文書は0件であり、compiler が memDocs を使うのは index 生成と ID ユニーク検証のみである。index 生成を廃止すれば、memDocs の用途は ID ユニーク検証だけになり、0件なら実質無害。

**判断**: `memory_docs` パターンは残しても害はないが、将来的に knowledge/ の .md ファイルが増えた際に不要なパースが発生する。削除しても安全だが、最小変更原則に従い本仕様では対象外とする。

### 無リスク: テンプレート変数 `{{memory:*}}` の消失

`{{memory:index}}` 以外の `{{memory:*}}` パターン (`{{memory:invariants}}`, `{{memory:decisions}}` 等) は、`execute-implementation-plan.md` と `create-specification.md` の「メモリの確認」ステップ内でのみ使用されていた。これらのステップ自体を完全削除するため、未解決変数が残る問題は発生しない。

**判断**: `memory` case ごと削除する。将来必要になれば再追加すればよい。

## 検証シナリオ (Verification Scenarios)

1. `prompts/memory/index.md` がファイルシステム上に存在しないこと
2. `prompts/manifest/safety/guards/deny-direct-edit-of-index.yaml` が存在しないこと
3. `features/tt/internal/prompt/memory/indexer.go` が存在しないこと
4. `tt prompt compile` が index.md を生成せずに正常完了すること
5. `tt prompt update` が全4ターゲットに対して正常完了すること
6. デプロイされた `.agents/`, `.claude/`, `.cursor/` のファイルに `{{memory:index}}` が残っていないこと
7. `prompts/memory/README.md` が影響を受けないこと (今回は修正対象外)

## テスト項目 (Testing for the Requirements)

### ビルド + 単体テスト

```bash
scripts/process/build.sh --backend-only
```

`compiler_test.go` と `template_test.go` の修正が正しいことをビルドパイプラインで検証する。

### 統合テスト (影響範囲: prompt コンパイル関連)

```bash
scripts/process/integration_test.sh --categories "common" --specify "Compile|Emit|Deploy"
```

### 手動確認 (自動化不要)

- `git grep "memory/index.md"` でプロジェクトコード (refs/ 除く) に残存参照がないことを確認
- `git grep "memory:index"` で同上
