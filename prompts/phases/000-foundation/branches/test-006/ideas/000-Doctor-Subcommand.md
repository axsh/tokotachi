# devctl doctor サブコマンド仕様書

## 背景 (Background)

`devctl` は、フィーチャーベースのモノレポにおいて、開発環境のオーケストレーションを行うCLIツールである。正しく動作するためには、リポジトリ内の特定のフォルダ構成・設定ファイルが期待通りに存在し、正しいフォーマットで記述されている必要がある。

現状、設定に不備がある場合はサブコマンド実行時（`devctl up` など）に初めてエラーとして表面化する。事前にまとめてチェックする手段がないため、問題の切り分けが困難になることがある。

`devctl doctor` コマンドを新設することで、利用者が環境全体の健全性を事前に一覧確認でき、問題がある場合は修正方法まで提示されるようになる。

## 要件 (Requirements)

### 必須要件

1. **新しいサブコマンド `devctl doctor`**
   - 引数なしで実行でき、リポジトリルートの検出から自動的にチェックを行う。
   - すべてのチェック結果を一覧表示する（✅ PASS / ❌ FAIL / ⚠️ WARN）。
   - FAILまたはWARNの項目には、期待する状態と修正方法を併記する。
   - 終了コード: すべてPASSなら `0`、1つでもFAILがあれば `1`。

2. **チェックカテゴリとチェック項目**

   #### A. 外部ツール依存チェック (External Tools)
   | チェック項目 | 判定方法 | 期待状態 |
   |---|---|---|
   | `git` が利用可能 | `git --version` 実行 | 正常終了 |
   | `docker` が利用可能 | `docker --version` 実行 | 正常終了 |
   | `gh` (GitHub CLI) が利用可能 | `gh --version` 実行 | 正常終了（WARN扱い：`pr`コマンドでのみ必要） |

   #### B. リポジトリ構造チェック (Repository Structure)
   | チェック項目 | 判定方法 | 期待状態 |
   |---|---|---|
   | Gitリポジトリのルートにいるか | `git rev-parse --show-toplevel` | カレントディレクトリまたは祖先がgitリポジトリのルート |
   | `features/` ディレクトリ存在 | ディレクトリ存在確認 | 存在する |
   | `work/` ディレクトリ存在 | ディレクトリ存在確認 | 存在する（WARN扱い：worktree未作成時は無い場合がある） |
   | `scripts/` ディレクトリ存在 | ディレクトリ存在確認 | 存在する |

   #### C. グローバル設定チェック (Global Config)
   | チェック項目 | 判定方法 | 期待状態 |
   |---|---|---|
   | `.devrc.yaml` の存在 | ファイル存在確認 | 存在する（WARN扱い：デフォルト値が使用される） |
   | `.devrc.yaml` の YAML パース | `resolve.LoadGlobalConfig` でロード | パースエラーなし |
   | `project_name` 設定値 | ロードした値を確認 | 空文字でないこと（WARN扱い：デフォルト `devctl` が使用される） |
   | `default_editor` 設定値 | ロードした値を確認 | サポートされるエディタ値か（空の場合はデフォルト `cursor`） |
   | `default_container_mode` 設定値 | ロードした値を確認 | サポートされるモード値か |

   #### D. フィーチャーチェック (Feature Validation)
   各 `features/<name>/` ディレクトリに対して以下をチェック:

   | チェック項目 | 判定方法 | 期待状態 |
   |---|---|---|
   | `feature.yaml` の存在 | ファイル存在確認 | 存在する |
   | `feature.yaml` の YAML パース | `resolve.LoadFeatureConfig` でロード | パースエラーなし |
   | `.devcontainer/devcontainer.json` の存在 | ファイル存在確認 | 存在する（WARN扱い：必須ではない） |
   | `.devcontainer/devcontainer.json` の JSON パース | JSONとして読み取り | パースエラーなし |
   | `go.mod` の存在（Go言語フィーチャーの場合） | ファイル存在確認 | 存在する |

### 任意要件

3. **`--feature <name>` フラグ**
   - 指定されたフィーチャーのみチェックを行う。省略時はすべてのフィーチャーをチェック。

4. **`--json` フラグ**
   - チェック結果を JSON 形式で出力する。CI/CD パイプラインやスクリプトからの利用を想定。

5. **`--verbose` フラグ（既存）**
   - 各チェックの詳細情報（検索パス、パース結果の中身など）を表示する。

## 実現方針 (Implementation Approach)

### アーキテクチャ

既存の `cmd/` パッケージにサブコマンド `doctor.go` を追加し、チェックロジックは新規 `internal/doctor/` パッケージに分離する。

```
features/devctl/
├── cmd/
│   ├── doctor.go       # [NEW] cobra サブコマンド定義
│   └── root.go         # [MODIFY] AddCommand(doctorCmd) 追加
└── internal/
    └── doctor/
        ├── doctor.go   # [NEW] チェック実行エンジン
        ├── checks.go   # [NEW] 各チェック項目の実装
        └── result.go   # [NEW] チェック結果の型定義と出力フォーマット
```

### 主要コンポーネント

1. **`result.go`**: チェック結果の型定義
   ```go
   type Status int  // Pass, Fail, Warn
   type CheckResult struct {
       Category    string
       Name        string
       Status      Status
       Message     string   // 結果メッセージ
       Expected    string   // 期待する状態
       FixHint     string   // 修正方法
   }
   ```

2. **`checks.go`**: 各チェック項目の実装
   - `checkExternalTools()`: git, docker, gh の存在チェック
   - `checkRepoStructure()`: ディレクトリ構造チェック
   - `checkGlobalConfig()`: `.devrc.yaml` のチェック
   - `checkFeatures()`: 全フィーチャーのチェック

3. **`doctor.go`**: チェック実行エンジン
   - すべてのチェックを順番に実行し、結果を集約
   - 既存の `resolve` パッケージ、`detect` パッケージのロジックを再活用

4. **`cmd/doctor.go`**: cobra コマンド定義
   - `--feature` フラグ、`--json` フラグの処理
   - 結果の出力とプロセス終了コード設定

### 出力フォーマット（テキスト）

```
🏥 devctl doctor
================

📦 External Tools
  ✅ git          git version 2.43.0
  ✅ docker       Docker version 24.0.7
  ⚠️  gh           not found
                   → Install GitHub CLI: https://cli.github.com/
                   → Note: only required for 'devctl pr' command

📂 Repository Structure
  ✅ Git repository root detected
  ✅ features/     directory exists
  ✅ work/         directory exists
  ✅ scripts/      directory exists

⚙️  Global Config (.devrc.yaml)
  ✅ File exists and is valid YAML
  ✅ project_name = "tokotachi"
  ✅ default_editor = "cursor"
  ✅ default_container_mode = "docker-local"

🔧 Feature: devctl
  ✅ feature.yaml       exists and valid
  ✅ .devcontainer/     devcontainer.json found and valid
  ✅ go.mod             exists

================
Result: 14 passed, 0 failed, 1 warning
```

### 出力フォーマット（JSON）

```json
{
  "results": [
    {
      "category": "External Tools",
      "name": "git",
      "status": "pass",
      "message": "git version 2.43.0"
    }
  ],
  "summary": {
    "total": 15,
    "passed": 14,
    "failed": 0,
    "warnings": 1
  }
}
```

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: 正常な環境でのチェック実行
1. 正常に構成されたリポジトリルートで `devctl doctor` を実行する。
2. すべてのチェック項目が ✅ PASS または ⚠️ WARN で表示される。
3. 終了コードが `0` であることを確認する。

### シナリオ 2: 不正な設定ファイルでのチェック実行
1. `.devrc.yaml` に不正な YAML を記述する（例: タブとスペースの混在）。
2. `devctl doctor` を実行する。
3. `.devrc.yaml` のパースチェックが ❌ FAIL になり、修正方法が表示される。
4. 終了コードが `1` であることを確認する。

### シナリオ 3: フィーチャー指定でのチェック実行
1. `devctl doctor --feature devctl` を実行する。
2. `devctl` フィーチャーのチェック結果のみが表示される。
3. 他のフィーチャーのチェック結果は表示されない。

### シナリオ 4: JSON 出力
1. `devctl doctor --json` を実行する。
2. 出力が有効な JSON であることを確認する。
3. `results` 配列と `summary` オブジェクトが含まれていることを確認する。

### シナリオ 5: 外部ツールが存在しない場合
1. `PATH` に `gh` が含まれていない環境で `devctl doctor` を実行する。
2. `gh` チェックが ⚠️ WARN になり、インストール方法が表示される。
3. `gh` は WARN のため、終了コードは `0` のまま。

## テスト項目 (Testing for the Requirements)

### 単体テスト

| 対象 | テスト内容 | 検証コマンド |
|---|---|---|
| `internal/doctor/checks.go` | 各チェック関数が正しいStatusを返す | `scripts/process/build.sh` |
| `internal/doctor/result.go` | テキスト出力・JSON出力のフォーマット検証 | `scripts/process/build.sh` |
| `internal/doctor/doctor.go` | チェック実行エンジンの集約ロジック | `scripts/process/build.sh` |

テスト方針:
- 外部ツールのチェックは、`cmdexec.Runner` をインターフェースでモック化して注入する。
- ファイルシステムのチェックは、`testing.TempDir()` で一時ディレクトリを作成し、テスト用の構造を構築する。
- 設定ファイルのチェックは、正常・異常の YAML/JSON ファイルを一時ディレクトリに配置してテストする。

### 統合テスト

| テスト内容 | 検証コマンド |
|---|---|
| 正常なリポジトリで `devctl doctor` 実行の end-to-end テスト | `scripts/process/integration_test.sh` |
| `--json` フラグ付き実行で有効な JSON が出力されること | `scripts/process/integration_test.sh` |

### 検証コマンド

```bash
# 全体ビルドと単体テスト
scripts/process/build.sh

# 統合テスト
scripts/process/integration_test.sh
```
