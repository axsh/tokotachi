# 024: ターゲット非依存エンティティ参照 (`{{kind:id}}`) の全ターゲット解決

## 背景 (Background)

マニフェスト本文では、エージェント固有パスを書かずに他エンティティを指せるよう、テンプレート参照 `{{kind:id}}` を使える設計になっている（例: `{{policy:testing-rules}}`, `{{procedure:test-generator}}`, `{{capability:portable-file-references}}`）。

Emit 時にターゲット固有パスへ展開されることを期待して、ソースパス表記を次のように置換してみた:

> `prompts/manifest/code_content/procedures/test-generator.md` → `{{procedure:test-generator}}`

しかし `tt prompt update` 後の成果物では `{{procedure:test-generator}}`（および既存の `{{policy:...}}` 等）がそのまま残り、エージェントが読むパスとして解釈されなかった。

### 現状の実装ギャップ

| 箇所 | 現状 |
| :--- | :--- |
| `features/tt/internal/prompt/emitter/template.go` | `policy` / `procedure` / `capability` / `target` の解決ロジックは存在する |
| `antigravity.go` | `ResolveTemplateVars` を policy / capability / procedure の body に適用している |
| `cursor.go` / `codex.go` / `claude_code.go` | **`ResolveTemplateVars` を呼んでいない**（本文に `{{...}}` が残る） |
| `resolvePolicyPath` | Antigravity 前提（拡張子 `.md`、`project-instructions` → `instructions.md`）。Cursor の実ファイルは `.mdc` かつ rename なし |

実例:

- Antigravity 成果物では `{{policy:coding-rules}}` → `.agent/rules/coding-rules.md` に解決済み
- Cursor 成果物（例: `.cursor/skills/test-generator/SKILL.md`）では `{{policy:testing-rules}}` が未解決のまま残存

ドキュメント側も `prompt-manifest-update` が「policy 間参照」のみ言及しており、procedure / capability の利用が公式に揃っていない。

## 要件 (Requirements)

### 必須要件

#### R1: 全ターゲット emitter で本文の `{{kind:id}}` を解決する

次の emitter すべてで、policy / capability / procedure の body を書き出す直前に `ResolveTemplateVars` を適用すること。

- Antigravity（既存・維持）
- Cursor
- Codex
- Claude Code

適用タイミングは frontmatter 除去・改行正規化の後、ターゲット固有 frontmatter 組み立ておよびファイル書き込みの前とする。capability の `RewriteBundlePaths` がある場合は、テンプレート解決の後にパス書き換えを行う（既存計画 `024-Capability-Skill-Bundle` と整合）。

#### R2: デプロイ対象の全 kind を参照解決の対象とする

本文中で使える参照 kind と解決先の対応:

| kind | 解決先の意味 | 備考 |
| :--- | :--- | :--- |
| `policy` | 当該ターゲットの rules 配下のポリシーファイル | 拡張子・ファイル名は R3 |
| `procedure` | 当該ターゲットの procedure 出力パス | workflows があれば `{workflows}/{id}.md`、なければ `{skills}/{id}/SKILL.md`（現行ロジック） |
| `capability` | `{skills}/{id}/SKILL.md` | |
| `target` | `name` / `meta_dir` 等の既存特殊変数 | 現状維持 |

「procedure の他も全て」の意図は、上記のデプロイ対象 kind を漏れなく同じ仕組みで解決すること。`skip` / `bundle` / `guard` / `worker` など **成果物パスを持たない kind** は参照解決の対象外とし、未知 kind は現行どおり置換せず残す。

`memory` kind は過去に削除済みであり、再導入しない。

#### R3: 解決パスを各ターゲットの実レイアウトに一致させる

`TemplateContext`（または同等）がターゲットごとの出力規約を持ち、解決結果が実際の emit ファイルパスと一致すること。

少なくとも次を満たす:

| ターゲット | policy の例 (`coding-rules`) | `project-instructions` | procedure の例 (`test-generator`) |
| :--- | :--- | :--- | :--- |
| Cursor | `.cursor/rules/coding-rules.mdc` | `.cursor/rules/project-instructions.mdc`（rename なし） | `.cursor/skills/test-generator/SKILL.md` |
| Claude Code | `.claude/rules/coding-rules.md` | `.claude/rules/project-instructions.md` | `.claude/skills/test-generator/SKILL.md` |
| Codex | `.codex/rules/coding-rules.md` | `.codex/rules/project-instructions.md` | `.codex/skills/test-generator/SKILL.md` |
| Antigravity | `.agent/rules/coding-rules.md` | `.agent/rules/instructions.md`（rename あり） | `.agent/workflows/test-generator.md`（workflows 使用時） |

rules / skills / workflows のベースパスは各 target YAML の `paths` を用いる（現行どおり上書き可能）。

#### R4: カタログ・作業用マニフェストで procedure / capability 参照を使えることを保証する

ソース本文に `{{procedure:id}}` / `{{capability:id}}` / `{{policy:id}}` を書いた場合、`tt prompt update --target <各ターゲット>` 後の成果物ではリテラル `{{...}}` が残らず、R3 のパスに置換されていること。

本仕様の範囲では、既存ソース内のハードコードパスを一括置換すること自体は必須としない（利用可能にすることが主目的）。ただし検証用に少なくとも 1 箇所の `{{procedure:...}}` 参照が解決されることを確認する。

#### R5: ドキュメントを更新する

`prompt-manifest-update`（カタログ原本および作業ツリー）の「policy 間参照」記述を拡張し、`{{policy:id}}` / `{{procedure:id}}` / `{{capability:id}}` / `{{target:...}}` がターゲット非依存参照であり、emit 時に各エージェントパスへ展開されることを明記する。

### 任意要件

#### O1: 未解決参照の検知（warn）

既知 kind なのにパスコンテキスト不足で空文字になる、あるいは置換されなかった `{{kind:id}}` を verbose / dry-run 時に警告する。必須ではないが、再発防止に有用。

#### O2: 共通ヘルパーへの集約

各 emitter が個別に `TemplateContext` を組み立てる重複を避け、`resolveTargetPaths` 相当を共通化する。可読性向上のためのリファクタであり、挙動変更を伴わない範囲で行う。

## 実現方針 (Implementation Approach)

### コンポーネント

1. **`template.go`**
   - `TargetPaths` または `TemplateContext` に policy ファイル命名規約（拡張子、`project-instructions` の rename 有無）を持たせる。
   - `resolvePolicyPath` / `resolveRef` をターゲット非依存の規約テーブルに基づく実装へ更新する。
2. **各 emitter (`cursor.go` / `codex.go` / `claude_code.go` / `antigravity.go`)**
   - Emit 開始時に当該ターゲット用 `TemplateContext` を構築。
   - policy / capability / procedure の body 処理パイプラインに `ResolveTemplateVars` を挿入。
3. **テスト**
   - `template_test.go`: ターゲット別の期待パス（特に Cursor の `.mdc` と AG の `instructions.md`）。
   - 各 emitter テスト: body に `{{policy:...}}` / `{{procedure:...}}` / `{{capability:...}}` を含むフィクスチャで、成果物に解決済みパスが含まれること・リテラルが残らないことを検証。
4. **ドキュメント**
   - `capabilities/prompt-manifest-update.md`（catalog + prompts）を更新。

### 設計上の決定

- 解決は **compile 時ではなく emit 時** に行う（現行設計を維持。ターゲットパスは emit コンテキストでのみ確定するため）。
- 未知 kind / 未知 `target:` サブキーは置換せず残す（破壊的変更を避ける）。
- procedure の workflows vs skills 分岐は既存 `resolveRef` ロジックを維持し、コンテキストの `Paths.Workflows` の有無で切り替える。

```mermaid
flowchart LR
  src["Manifest body<br/>{{procedure:test-generator}}"]
  emit["Emitter + TemplateContext"]
  cursor[".cursor/skills/test-generator/SKILL.md"]
  ag[".agent/workflows/test-generator.md"]
  codex[".codex/skills/test-generator/SKILL.md"]
  src --> emit
  emit --> cursor
  emit --> ag
  emit --> codex
```

## 検証シナリオ (Verification Scenarios)

1. マニフェスト本文（例: いずれかの procedure / capability / policy）に次を含める:
   - `{{procedure:test-generator}}`
   - `{{policy:testing-rules}}`
   - `{{capability:portable-file-references}}`
2. `tt prompt update --target cursor` を実行する。
3. 当該成果物を開き、上記 3 参照がそれぞれ次に置換されていることを確認する:
   - `.cursor/skills/test-generator/SKILL.md`
   - `.cursor/rules/testing-rules.mdc`
   - `.cursor/skills/portable-file-references/SKILL.md`
   - リテラル `{{procedure:test-generator}}` 等が残っていないこと。
4. 同様に `--target ag` を実行し、少なくとも次を確認する:
   - `{{procedure:test-generator}}` → `.agent/workflows/test-generator.md`
   - `{{policy:testing-rules}}` → `.agent/rules/testing-rules.md`
   - `{{policy:project-instructions}}` を使う場合 → `.agent/rules/instructions.md`
5. `--target codex` および `--target claude-code` でも、各 `paths.rules` / `paths.skills` に沿った拡張子 `.md` のパスへ解決されることを確認する。
6. 解決対象外の表記（例: `{{unknown:foo}}`）は置換されず残ることを確認する。

## テスト項目 (Testing for the Requirements)

### 要件と検証の対応

| 要件 | 検証方法 |
| :--- | :--- |
| R1 | 各 emitter の単体テストで body 内テンプレートが解決されること |
| R2 | `TestResolveTemplateVars` で policy / procedure / capability / target をカバー |
| R3 | ターゲット別コンテキストでの期待パス（`.mdc` / `instructions.md` / workflows）を単体テスト |
| R4 | emitter フィクスチャに `{{procedure:...}}` を含め、成果物にリテラルが残らないこと |
| R5 | ドキュメント更新の存在確認（レビュー）＋必要なら文字列アサーションは任意 |

### ビルド・全体検証

1. ビルド＋単体テスト（emitter / template を含む）:

   ```
   scripts/process/build.sh
   ```

2. prompt emitter 関連の絞り込み確認（開発中）:

   ```
   go test ./features/tt/internal/prompt/emitter/... -count=1
   ```

3. 統合テストは本変更の主影響外のため必須カテゴリは設けない。リグレッションが疑われる場合のみ:

   ```
   scripts/process/integration_test.sh --categories "common" --specify "Prompt|Manifest|tt"
   ```

   （該当テストが存在しない場合はスキップ可。本体検証は 1・2 を正とする。）
)
