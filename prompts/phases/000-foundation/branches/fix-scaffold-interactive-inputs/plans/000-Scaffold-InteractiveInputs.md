# 000-Scaffold-InteractiveInputs

> **Source Specification**: [000-Scaffold-InteractiveInputs.md](file://prompts/phases/000-foundation/ideas/fix-scaffold-interactive-inputs/000-Scaffold-InteractiveInputs.md)

## Goal Description

`tt scaffold` コマンドのインタラクティブ入力を改善する。具体的には:

1. オプション値の重複問い合わせを解消する
2. `required` の値に関わらず全オプションをインタラクティブに問い合わせる
3. デフォルト値の表示・空Enter時の自動適用を実装する
4. `--v key=value` オプションによる直接値指定を追加する
5. `--default` オプションによる非必須項目のデフォルト自動適用を追加する

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 重複質問の解消 | Proposed Changes > scaffold.go: `Plan` に `OptionValues` 追加、`Apply()` 内の再問い合わせ削除 |
| 全オプションのインタラクティブ問い合わせ | Proposed Changes > template.go: `CollectOptionValues` 改修 |
| デフォルト値の表示と適用 | Proposed Changes > template.go: プロンプト表示ロジック変更 |
| `--v key=value` オプション追加 | Proposed Changes > scaffold.go (cmd): フラグ追加、パースロジック |
| `--default` オプション追加 | Proposed Changes > scaffold.go (cmd): フラグ追加、`RunOptions` へ反映 |
| `--yes` との組み合わせ | Proposed Changes > scaffold.go (cmd): 既存動作を維持 |
| `required: true` + default なし + 空入力 → エラー | Proposed Changes > template.go: エラーハンドリング |
| `required: true` + default あり + 空入力 → デフォルト適用 | Proposed Changes > template.go: デフォルト適用ロジック |

## Proposed Changes

### scaffold パッケージ (internal/scaffold)

---

#### [MODIFY] [template_test.go](file://features/tt/internal/scaffold/template_test.go)

*   **Description**: `CollectOptionValues` のテストを、新しいシグネチャ（`useDefaults` パラメータ追加）に対応させ、新規テストケースを追加する。
*   **Technical Design**:
    ```go
    // 新シグネチャに合わせて既存テストを更新
    func CollectOptionValues(options []Option, provided map[string]string,
        reader io.Reader, writer io.Writer, useDefaults bool) (map[string]string, error)
    ```
*   **Logic**:
    *   既存5テストの `CollectOptionValues` 呼び出しに第5引数 `false` を追加する
    *   `TestCollectOptionValues_DefaultApplied`: `useDefaults: true` の場合に `required: false` のデフォルト値が自動適用されることを検証するよう変更
    *   新規テスト `TestCollectOptionValues_InteractiveNonRequired`: `useDefaults: false` の場合、`required: false` の項目もインタラクティブに問い合わせされることを検証。入力を与え、その値が使われることを確認
    *   新規テスト `TestCollectOptionValues_InteractiveEmptyEnterWithDefault`: デフォルト値がある項目で空Enter入力（`\n`）した場合にデフォルト値が適用されることを検証
    *   新規テスト `TestCollectOptionValues_RequiredWithDefaultEmptyEnter`: `required: true` + `default` ありの項目で空Enter（`\n`）入力した場合にデフォルト値が適用されることを検証
    *   新規テスト `TestCollectOptionValues_RequiredNoDefaultEmptyEnter`: `required: true` + `default` なしの項目で空Enter入力した場合にエラーになることを検証（既存の `RequiredMissing` テストに類似だが、明示的にケースを区別）
    *   新規テスト `TestCollectOptionValues_UseDefaultsSkipsNonRequired`: `useDefaults: true` の場合、`required: false` の項目はプロンプトが表示されず、`required: true` の項目は表示されることを確認
    *   新規テスト `TestCollectOptionValues_PromptFormat`: プロンプト出力にデフォルト値のヒント表示 `(default-value)` が含まれることを検証
    *   新規テスト `TestParseOptionOverrides`: `--v key=value` 文字列のパースロジックのテスト

    テストケース一覧表:

    | テストケース | options | provided | useDefaults | stdin | 期待結果 |
    |---|---|---|---|---|---|
    | `AllProvided` | required + non-required | 両方指定 | false | - | provided の値が使われる |
    | `DefaultApplied` (改修) | required + non-required(default あり) | required のみ | true | - | non-required はデフォルト適用 |
    | `InteractiveNonRequired` | non-required(default あり) | 空 | false | "custom\n" | プロンプト表示、入力値が使われる |
    | `InteractiveEmptyEnterWithDefault` | non-required(default あり) | 空 | false | "\n" | デフォルト値が適用 |
    | `RequiredWithDefaultEmptyEnter` | required(default あり) | 空 | false | "\n" | デフォルト値が適用 |
    | `RequiredNoDefaultEmptyEnter` | required(default なし) | 空 | false | "\n" | エラー |
    | `InteractiveInput` (既存) | required | 空 | false | "my-feature\n" | 入力値が使われる |
    | `RequiredMissing` (既存) | required(default なし) | 空 | false | "\n" | エラー |
    | `NoOptions` (既存) | nil | nil | false | - | 空map |
    | `UseDefaultsSkipsNonRequired` | required + non-required | 空 | true | "val\n" | required は問い合わせ、non-required はスキップ |
    | `PromptFormat` | default あり | 空 | false | "val\n" | 出力に "(default)" 含む |
    | `ParseOptionOverrides` | - | - | - | - | "a=b" → {"a":"b"} |

---

#### [MODIFY] [template.go](file://features/tt/internal/scaffold/template.go)

*   **Description**: `CollectOptionValues` 関数を改修し、全オプションのインタラクティブ問い合わせ、デフォルト値表示、`useDefaults` フラグ対応を追加する。また、`--v` オプション値のパース関数を追加する。
*   **Technical Design**:
    ```go
    // シグネチャ変更: useDefaults パラメータ追加
    func CollectOptionValues(options []Option, provided map[string]string,
        reader io.Reader, writer io.Writer, useDefaults bool) (map[string]string, error)

    // 新規関数: --v key=value のパース
    func ParseOptionOverrides(args []string) (map[string]string, error)
    ```
*   **Logic**:

    `CollectOptionValues` の新しいロジック:
    ```
    for each option:
      1. provided に値があれば → その値を使用 (変更なし)
      2. useDefaults == true && required == false && default != "" → デフォルト値を自動適用
      3. 上記以外 → インタラクティブに問い合わせ:
         - default がある場合: "? {Description} ({name}) ({default}): " と表示
         - default がない場合: "? {Description} ({name}): " と表示
         - 空Enter の場合:
           - default があれば → デフォルト値を適用
           - default がなく required == true → エラー
           - default がなく required == false → 空文字を設定
         - 入力があれば → その入力値を使用
    ```

    `ParseOptionOverrides` のロジック:
    ```
    for each arg in args:
      "key=value" の形式を分割
      "=" がない場合はエラー
      key が空の場合はエラー
      結果の map に key → value を設定
    return map, nil
    ```

---

#### [MODIFY] [applier.go](file://features/tt/internal/scaffold/applier.go)

*   **Description**: `Plan` 構造体に `OptionValues` フィールドを追加する。
*   **Technical Design**:
    ```go
    type Plan struct {
        ScaffoldName      string
        FilesToCreate     []FileAction
        FilesToSkip       []FileAction
        FilesToModify     []FileAction
        PostActions       PostActions
        PermissionActions []PermissionAction
        Warnings          []string
        OptionValues      map[string]string // 追加: 収集済みオプション値
    }
    ```
*   **Logic**: フィールド追加のみ。`BuildPlan` 関数の `optionValues` 引数を `plan.OptionValues` にも保持する。

---

#### [MODIFY] [scaffold.go](file://features/tt/internal/scaffold/scaffold.go)

*   **Description**: `RunOptions` に `OptionOverrides` と `UseDefaults` を追加。`Run()` で収集した値を `Plan.OptionValues` に保持し、`Apply()` では `Plan.OptionValues` を再利用して再問い合わせを廃止する。
*   **Technical Design**:
    ```go
    type RunOptions struct {
        Pattern         []string
        RepoURL         string
        RepoRoot        string
        DryRun          bool
        Yes             bool
        Lang            string
        Logger          *log.Logger
        Stdout          io.Writer
        Stdin           io.Reader
        OptionOverrides map[string]string // 追加: --v で指定された値
        UseDefaults     bool              // 追加: --default フラグ
    }
    ```
*   **Logic**:

    `Run()` の変更:
    ```
    // L105: CollectOptionValues の呼び出し変更
    optionValues, err = CollectOptionValues(
        entry.Options, opts.OptionOverrides, opts.Stdin, opts.Stdout, opts.UseDefaults)
    // plan にオプション値を保持
    plan.OptionValues = optionValues
    ```

    `Apply()` の変更:
    ```
    // L226-229: CollectOptionValues の再呼び出しを削除
    // 代わりに plan.OptionValues を使用
    optionValues := plan.OptionValues
    ```

---

### cmd パッケージ (cmd)

#### [MODIFY] [scaffold.go](file://features/tt/cmd/scaffold.go)

*   **Description**: `--v` と `--default` フラグを追加し、`RunOptions` に値を渡す。
*   **Technical Design**:
    ```go
    var (
        scaffoldFlagYes      bool
        scaffoldFlagRollback bool
        scaffoldFlagList     bool
        scaffoldFlagRepo     string
        scaffoldFlagLang     string
        scaffoldFlagCwd      bool
        scaffoldFlagValues   []string // 追加: --v key=value
        scaffoldFlagDefault  bool     // 追加: --default
    )
    ```
*   **Logic**:

    `init()` に追加:
    ```go
    scaffoldCmd.Flags().StringArrayVar(&scaffoldFlagValues, "v", nil, "Set option value directly (key=value), repeatable")
    scaffoldCmd.Flags().BoolVar(&scaffoldFlagDefault, "default", false, "Use default values for non-required options without prompting")
    ```

    `runScaffold()` の変更:
    ```
    // --v の値をパースして OptionOverrides に渡す
    overrides, err := scaffold.ParseOptionOverrides(scaffoldFlagValues)
    if err != nil { return err }

    opts := scaffold.RunOptions{
        // ... 既存フィールド ...
        OptionOverrides: overrides,
        UseDefaults:     scaffoldFlagDefault,
    }
    ```

## Step-by-Step Implementation Guide

### フェーズ1: テスト作成 (TDD - テストファースト)

1. **`template_test.go` の更新**:
   - 既存テスト5件の `CollectOptionValues` 呼び出しに第5引数 `false` を追加（コンパイルエラーは期待通り）
   - `TestCollectOptionValues_DefaultApplied` を `useDefaults: true` に変更
   - 新規テスト7件を追加:
     - `TestCollectOptionValues_InteractiveNonRequired`
     - `TestCollectOptionValues_InteractiveEmptyEnterWithDefault`
     - `TestCollectOptionValues_RequiredWithDefaultEmptyEnter`
     - `TestCollectOptionValues_RequiredNoDefaultEmptyEnter`
     - `TestCollectOptionValues_UseDefaultsSkipsNonRequired`
     - `TestCollectOptionValues_PromptFormat`
     - `TestParseOptionOverrides`
   - この時点で `build.sh` はコンパイルエラーになることを確認

### フェーズ2: 本体実装

2. **`template.go` の改修**:
   - `CollectOptionValues` のシグネチャに `useDefaults bool` を追加
   - 全オプションのインタラクティブ問い合わせロジックを実装
   - デフォルト値のプロンプト表示と空Enter時の適用ロジックを実装
   - `ParseOptionOverrides` 関数を新規実装

3. **`applier.go` の改修**:
   - `Plan` 構造体に `OptionValues map[string]string` フィールドを追加
   - `BuildPlan` 関数で `plan.OptionValues = optionValues` を設定

4. **`scaffold.go` (internal) の改修**:
   - `RunOptions` に `OptionOverrides` と `UseDefaults` を追加
   - `Run()` の `CollectOptionValues` 呼び出しを更新 (`opts.OptionOverrides`, `opts.UseDefaults` を渡す)
   - `Run()` で `plan.OptionValues = optionValues` を設定
   - `Apply()` から `CollectOptionValues` 呼び出しを削除し、`plan.OptionValues` を使用するよう変更

5. **`scaffold.go` (cmd) の改修**:
   - `scaffoldFlagValues` と `scaffoldFlagDefault` の変数定義を追加
   - `init()` にフラグ登録を追加
   - `runScaffold()` で `ParseOptionOverrides` を呼び出し、`RunOptions` に値を設定

### フェーズ3: ビルドと検証

6. **ビルドと単体テスト実行**: `build.sh` を実行し、全テストが通ることを確認
7. **統合テスト実行**: `integration_test.sh` を実行し、既存テストが壊れていないことを確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```
   - `template_test.go` の全テストケース（既存5件 + 新規7件）がPASSすること
   - コンパイルエラーがないこと

2. **Integration Tests**:
   ```bash
   ./scripts/process/integration_test.sh --categories "scaffold" --specify "TestScaffoldDefault"
   ```
   - `TestScaffoldDefault`: `--yes` 使用のため、今回の変更に影響しない。PASSすること
   - `TestScaffoldList`: 変更なし。PASSすること
   - `TestScaffoldCwdFlag`: `--yes` 使用のため影響なし。PASSすること
   - **Log Verification**: 統合テストの出力に `FAIL` が含まれないこと

## Documentation

本計画では既存ドキュメントへの変更は不要と判断する。CLI ヘルプ（cobra の `Short`/`Long` フィールド）は自動的にフラグ情報を含むため、追加のドキュメント更新は不要。
