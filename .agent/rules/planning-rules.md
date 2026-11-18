# 計画立案規範 (Planning Rules)

本規範は、計画立案プロセス、品質管理、およびビルド/テストの実行基準を定め、プロジェクトの円滑な進行を確保することを目的とします。

## 1. テストと品質に対する共通哲学 (General Philosophy)

コンポーネントと言語に関わらず、以下の原則を遵守してください。

### 1.1 TDD (Test Driven Development)
*   **Failed First**: いかなる機能も、「その機能が正しく動作することを確認するテスト」を先に書き、それが失敗することを確認してから実装を開始してください。
*   **E2E First for GUI**: 画面（GUI）を作る・変える計画では、E2E テストシナリオの実装と、その失敗の確認（レッド）を、画面と API の実装より前の手順として置いてください。画面が使う API やロジックの単体テストも、それぞれの実装より前に置きます。
*   **Verification Oriented**: 「どう実装するか」の前に「どう検証するか（テストするか）」を計画してください。検証不可能な機能は実装してはいけません。

### 1.2 Fail Fast
*   フィードバックループを最小化するため、単体テスト（Unit Test）は高速に実行可能であることを維持してください。
*   統合テスト等の重いテストを回す前に、単体テストで論理的な誤りを排除してください。

### 1.3 Concretization (具体化)
*   **Abstraction is Logic Loss**: 仕様書から実装計画への変換プロセスは、抽象化（要約）ではなく、**具体化（詳細化）**のプロセスでなければなりません。
    *   **Data Structure is Mandatory**: 仕様書に記載された構造体定義やデータモデルは、必ず実装計画の `Proposed Changes` に含めてください。これらを「自明な既存コード」や「コンテキスト情報」として省略してはいけません。
*   **Traceability**: 前段のドキュメント（仕様書）に記載された具体的な手順、条件、シナリオ（例: 「(1) Aして (2) Bする」）は、実装計画においても個別の検証項目または実装手順として明確に追跡可能である必要があります。

## 2. 計画策定における要件 (Planning Requirements)

実装計画 (`prompts/phases/.../plans/...`) を作成する際は、**「これを読めば実装の実現方法が明確にわかる」** レベルの詳細さが求められます。

- どのファイルを新規に作成すべきか、もしくはどのファイルをどのように修正すべきかを明確に記述してください。
- コーディングする際に必要な情報を、具体的に記載してください。

### 2.1 Backend Development (Go)

バックエンド機能の計画においては、以下の要素を具体的に記述してください。

*   **詳細な実現手順**:
    *   どのパッケージの、どの構造体/インターフェースに変更を加えるか。
    *   追加するAPIエンドポイントの定義（パス、メソッド、リクエスト/レスポンス）。
*   **テスト計画 (Unit & Integration)**:
    *   **Unit Tests**: テーブル駆動テスト (`tests := []struct{...}`) のケース設計。モックが必要な依存関係（DB, 外部API）。
    *   **Integration Tests**: `tests/` 配下のどのファイルに追加するか。実際のDBやDockerコンテナとの連携確認手順。
    *   **統合テスト必要性の判断基準**: 以下のいずれかに該当する場合、統合テストは**必須**です。
        *   ファイルシステムへの書き込み/読み取りを行う
        *   SQLite/DB 操作を行う
        *   外部コマンド (git 等) を呼び出す
        *   CLI サブコマンドとして利用者に提供される
        *   上記のいずれにも該当しない純粋なロジックのみの場合は、単体テストのみで可。ただし Verification Plan にその理由を明記すること。
    *   記述順序: `Proposed Changes` では必ず `_test.go` を先に記述してください。

### 2.2 Frontend Development (Web 画面 / React)

Web 画面（SPA）の計画においては、画面特有の検証容易性を確保するため、以下の要素が必須です。

*   **Component Design**:
    *   ReactコンポーネントのProps定義とState管理。
    *   **Test IDの定義**: 文言で探せない要素と、企業ごとのカスタマイズで文言が変わりうる要素には、`data-testid` を定義するよう計画に含めてください（`.agent/rules/testing-rules.md` §10.3）。
*   **E2E Scenarios**:
    *   **主たる検証手段**: 画面の可視的な変更やインタラクションの追加は、必ず E2E テストシナリオとして定義します。手動確認はあくまで補助的な位置付けです。
    *   Web 画面の E2E は、`.agent/rules/testing-rules.md` §10 の 3 つの層とペルソナで計画してください。
    *   計画には、追加・変更する画面オブジェクト（画面 ID ごと）、目的別操作オブジェクト、シナリオ（前提のペルソナと、ペルソナごとの手順）を書いてください。
        *   Bad: 「招待をテストする」
        *   Good: 「シナリオ『招待した人がチームに加わる』を追加。前提: ペルソナ A（企業 A のテナント管理者）、ペルソナ B（企業 A への招待先）。A が目的別操作 `inviteUser` で B を招待し、B が `acceptInvitation` で受諾する。A が開いたままの ADM-02（画面オブジェクト `UserListPage`）で、B の状態が『有効』に変わることを確かめる。」
    *   **記述順序**: 3 つの層の追加・変更とレッドの確認を、画面と API の実装手順より前に置いてください（§1.1 の E2E First for GUI）。




## 3. ビルドとテスト方針

本プロジェクトは procedure（ワークフロー）および `scripts/` を活用した標準化されたプロセスを採用しています。

### 3.1 標準化された実行手段

全ての検証は `scripts/` 配下のスクリプトを通じて行ってください。

*   **Backend Track**:
    *   Unit: `scripts/process/build.sh --skip-frontend`
    *   Integration: `scripts/process/integration_test.sh`
*   **Frontend Track**:
    *   Unit (画面): `./scripts/process/build.sh` (Prohibit direct `npm test` in plans)
    *   E2E (Web 画面): `.agent/rules/testing-rules.md` §10 に従う。実行は `./scripts/process/integration_test.sh --categories e2e`

    **Note**: Do not use `cd` commands in plans. Use paths relative to the project root.

    > [!WARNING]
    > **Prohibition of Raw Toolchain Commands**:
    > You are **PROHIBITED** from suggesting raw `go build`, `go test`, or `npm run build` commands in implementation plans.
    > You **MUST** use the provided scripts (`scripts/process/build.sh`, `scripts/process/integration_test.sh`).
    > *Reason*: These scripts handle critical build steps (binary relocation, environment variables, incremental build logic) that raw commands miss.

### 3.2 テストファイルの配置と命名

*   **Backend**: `package_test.go` (Unit), `tests/{cat}_{func}_test.go` (Integration)
*   **Frontend**: `Component.test.tsx` (Unit)。E2E の置き場所と名前は `.agent/rules/testing-rules.md` §10.5 に従う


### 3.3 ビルドと検証要件 (Build & Verification Requirement)

実装計画の「Verification Plan」には、検証に行うための**スクリプト実行コマンド**を必ず明記してください。
GUIの変更であっても、手動検証ではなく自動化された E2E テストを使用することを原則とします。

*   **推奨記述例**:
    1.  **Automated Verification**: Run `./scripts/process/build.sh && ./scripts/process/integration_test.sh --specify "New Feature"` to verify the UI changes.

    > [!WARNING]
    > **Prohibition of Manual Verification**:
    > You are **PROHIBITED** from planning "Manual Verification" as the primary verification method.
    > Do not write steps like "Open the browser and click the button". Instead, write "Create E2E test scenario to click the button".
    > *Exception*: Strictly visual aesthetics (e.g. "Check if the color is correct") can be manual, but functional logic MUST be automated.
    
    *   **Prohibited Commands**:
        *   ❌ `npm run build`, `npm test`
        *   ❌ `go build`, `go test`
        *   ❌ `cd ... && ...`
        *   ❌ `integration_test.sh` (STRICTLY PROHIBITED without preceding `build.sh`)
    *   **Required Commands**:
        *   ✅ `./scripts/process/build.sh && ./scripts/process/integration_test.sh`

    > [!WARNING]
    > **Partial Build Flags in Final Verification**:
    > `--skip-frontend` は開発中の高速フィードバック用です。
    > Verification Plan の最終検証コマンドには `./scripts/process/build.sh` (フラグなし) を使用してください。
