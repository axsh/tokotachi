# プロンプトデータフロー解説書

本書は、`prompts/manifest` および `prompts/memory` ディレクトリ配下の各 Markdown ファイルが、コンパイルを経て各 Coding Agent（Antigravity, Cursor, Claude Code, Codex）向けのファイル（rules, skills など）へどのように変換・配置されるか、そのデータフローを説明します。

## 全体データフロー

以下は、Markdownソースファイルから解析、統合を経て、各エージェント向けの出力先へと配信されるデータフローの概要です。

```mermaid
graph TD
    subgraph 入力ファイル (Source Files)
        Policies["policies/*.md<br>(kind: policy)"]
        Procedures["procedures/*.md<br>(kind: procedure)"]
        Capabilities["capabilities/*.md<br>(kind: capability)"]
        MemoryDocs["memory/**/*.md<br>(Memory Documents)"]
    end

    subgraph コンパイル (Compile / Resolve)
        Parser["Parser (ParseAllEntities / ParseAllMemoryDocs)"]
        Resolved["ResolvedManifest<br>(manifest.resolved.yaml)"]
    end

    subgraph 各エージェントへの配信 (Emitter & Deploy)
        AntigravityEmitter["Antigravity Emitter"]
        CursorEmitter["Cursor Emitter"]
        ClaudeCodeEmitter["Claude Code Emitter"]
        CodexEmitter["Codex Emitter"]
    end

    subgraph 出力ファイル (Output Files)
        %% Antigravity
        AG_Rules[".agents/rules/<br>- instructions.md<br>- {id}.md"]
        AG_Skills[".agents/skills/{id}/SKILL.md"]

        %% Cursor
        CS_Rules[".cursor/rules/{id}.mdc"]
        CS_Skills[".cursor/skills/{id}/SKILL.md"]

        %% Claude Code
        CC_Rules[".claude/rules/{id}.md"]
        CC_Skills[".claude/skills/{id}/SKILL.md"]

        %% Codex
        CX_Rules[".agents/rules/{id}.md"]
        CX_Skills[".agents/skills/{id}/SKILL.md"]
        CX_Index["AGENTS.md (インデックス自動更新)"]
    end

    Policies --> Parser
    Procedures --> Parser
    Capabilities --> Parser
    MemoryDocs --> Parser

    Parser --> Resolved

    Resolved --> AntigravityEmitter
    Resolved --> CursorEmitter
    Resolved --> ClaudeCodeEmitter
    Resolved --> CodexEmitter

    AntigravityEmitter --> AG_Rules
    AntigravityEmitter --> AG_Skills

    CursorEmitter --> CS_Rules
    CursorEmitter --> CS_Skills

    ClaudeCodeEmitter --> CC_Rules
    ClaudeCodeEmitter --> CC_Skills

    CodexEmitter --> CX_Rules
    CodexEmitter --> CX_Skills
    CodexEmitter --> CX_Index
```

## Frontmatter による出力先の判定ルール

入力ファイルに指定された Frontmatter（メタデータ）の条件によって、生成されるファイルの場所や形式、Frontmatterの内容が決定されます。

### 1. Policies (kind: policy) -> 各エージェントの Rules
主に各エージェントの動作を規定するルール（ポリシー）として出力されます。

| ターゲットエージェント | 出力パス | Frontmatter 変換ルール | 備考 |
|---|---|---|---|
| **Antigravity** | `.agents/rules/{id}.md` | `activation.mode == "always"` の場合、`trigger: always_on` を付与。それ以外は Frontmatter なし。 | `id == "project-instructions"` の場合は `instructions.md` として出力される。 |
| **Cursor** | `.cursor/rules/{id}.mdc` | `description: {title}`<br>`globs: {paths}`<br>`alwaysApply: {activation.mode == "always"}` を持つ Frontmatter を生成。 | ファイル拡張子は `.mdc`。 |
| **Claude Code** | `.claude/rules/{id}.md` | `paths` が指定されている場合、`paths: [...]` を持つ Frontmatter を生成。指定がない場合は Frontmatter なし。 | |
| **Codex** | `.agents/rules/{id}.md` | Frontmatter なし（純粋な Markdown）。 | `AGENTS.md` の Rules リストに自動追加される。 |

### 2. Capabilities (kind: capability) -> 各エージェントの Skills
エージェントが実行可能なスキルとして出力されます。

*   **全エージェント共通の出力形式**: 各エージェントの skills ディレクトリ配下に `{id}/SKILL.md` という構成で出力されます。
*   **Frontmatter 変換ルール**:
    すべてのエージェントで共通して以下の `SkillFrontmatter` 形式にシリアライズされて出力されます。
    ```yaml
    name: {id}
    description: {description}
    paths: {paths} (Antigravity のみ出力)
    disable-model-invocation: {manual_only}
    ```

| ターゲットエージェント | 出力パス |
|---|---|
| **Antigravity** | `.agents/skills/{id}/SKILL.md` |
| **Cursor** | `.cursor/skills/{id}/SKILL.md` |
| **Claude Code** | `.claude/skills/{id}/SKILL.md` |
| **Codex** | `.agents/skills/{id}/SKILL.md` |

### 3. Procedures (kind: procedure) -> 各エージェント의 Skills
作業手順（プロシージャ）は、各エージェントにおいて **スキル（Skills）** として出力・展開されます。

*   **全エージェント共通の出力形式**: `capabilities` と同様に `{id}/SKILL.md` として出力されます。
*   **Frontmatter 変換ルール**:
    ```yaml
    name: {id}
    description: {title}
    disable-model-invocation: {trigger.manual_only}
    ```
*   **本文の自動構成**:
    Markdown ソースファイルの本文部分がそのまま使用されますが、Frontmatter 内に `steps`（リスト形式）が定義されている場合は、本文の末尾に `## Steps` というセクションが自動生成され、ステップ番号付きリストとして出力されます。

### 4. Memory Docs (Memory Documents)
`prompts/memory/**/*.md` 配下の Memory Docs は、特定のディレクトリに直接ファイルとしてコピーされることはありません。
代わりに、他の Markdown テンプレート本文中に記述された `{{kind:id}}` などの変数を解決するためのデータソースとして利用されます。

---

## Emitter ごとの特別処理

### 1. Codex Emitter (AGENTS.md の更新)
Codex Emitter は、出力ファイルを生成するだけでなく、プロジェクトのインデックスファイル（デフォルトは `AGENTS.md`）にある `AGENT-MANAGED:BEGIN` から `AGENT-MANAGED:END` のマーカーセクションを動的に書き換えます。
このセクションには、今回コンパイル・デプロイされたポリシーおよびスキルの一覧が自動的に挿入されます。

### 2. Branch Skills
すべてのエージェントエミッターは、共通処理として `branches/*/skills/` ディレクトリ配下にある分岐（ブランチ）固有のスキルファイルをスキャンし、各エージェントの skills ディレクトリへマージして出力します。
