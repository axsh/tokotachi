# tt prompt updateの変更検出廃止とビルドクリーンアップ設計仕様書

## 背景 (Background)

`tt prompt update` および `tt prompt deploy` コマンドは、プロンプトソースファイルに変更があった場合のみコンパイル・デプロイを実行する差分検出機能を持っていました。しかし、未コミットの変更が検出されない問題や、ダイジェスト対象外のファイルが存在する問題により、想定通りにコンパイルが実行されない現象が発生していました。

また、ビルド時の中間生成物出力先である `tmp/dist/` がクリーンアップされないため、古い中間ファイルが残る問題も指摘されていました。

これらの問題を解決するため、変更検出ロジック（Gitログおよびダイジェスト比較）を完全に廃止して常にコンパイルとデプロイを実行するようにし、ビルド処理の開始前に `tmp/dist/` をクリーンアップする設計へと移行します。

## 要件 (Requirements)

1. **変更検出ロジックの完全廃止**
   - `tt prompt update` および `tt prompt deploy` を実行した際、変更の有無にかかわらず常にコンパイルおよびデプロイを実行する。
   - `last_update.yaml` メタデータの読み書き、および Git 履歴に基づく変更検出 (`CheckForChanges`)、ドリフト検出 (`CheckDrift`) によるスキップ処理を廃止する。
   - ソースファイルの SHA-256 ダイジェスト比較によるスキップ判定を廃止する。
   - `--force` フラグは後方互換性のために残す（何もしない no-op フラグとする）。
   - コマンドの結果において、スキップされたことを示す `Skipped` フィールドは常に `false` になるようにする。

2. **中間生成物ディレクトリ（tmp/dist/）のクリーンアップ**
   - コンパイル (`Compile`) 処理の中で、バリデーションエラーがないことが確認された後、実際の生成処理を行う直前に、中間生成物ディレクトリ（`tmp/dist/`）を完全に削除して再作成する。
   - ドライラン (`DryRun = true`) の場合は、ファイルの書き出しを行わないため、クリーンアップも実行しない。

3. **デプロイ時の EmitMode の維持**
   - `tmp/dist/` から出力先（`.agents/` など）への展開ステージでは、差分に応じた上書き・スキップ・不要ファイル削除などのオプション（`EmitMode` の `overwrite`, `skip`, `immune`）をそのまま維持し、動作するようにする。

## 実現方針 (Implementation Approach)

### 1. compiler.Update の修正 (update.go)
- `Update()` 関数において、`shouldUpdate` や `CheckForChanges`、`CheckDrift` を呼び出して `needsUpdate` を判定する処理を削除し、常に `Deploy()` を実行するように変更する。
- 戻り値の `TargetUpdateResult` の `Skipped` は常に `false` にし、`Reason` は空にする。
- メタデータファイル (`last_update.yaml`) の読み書き処理 (`ReadMetadata`, `WriteMetadata`) は呼び出さず、可能であれば関数ごと削除する。

### 2. compiler.Deploy の修正 (deploy.go)
- `Deploy()` 内のダイジェスト比較と `CheckDrift` によるスキップ判定（ステップ7）を削除する。
- `result.Skipped` は常に `false` になる。
- ダイジェストの計算・保存処理 (`ComputeSourceDigest`, `SaveDigest`) 自体は、ログや他の互換性のために残す必要がなければ削除する（今回は不要なため削除する）。

### 3. compiler.Compile の修正 (compiler.go)
- `Compile()` 関数のバリデーションチェック（ステップ8）が通過した後、実際のファイル書き出し（ステップ11）と Emitter による展開（ステップ13）の前に、`buildDir` (デフォルト `tmp/dist/`) のクリーンアップ処理を追加する。
- `!opts.DryRun` の場合のみ、`buildDir` を `os.RemoveAll` して `os.MkdirAll` で再作成する。

### 4. 既存テストの修正
- `update_test.go` および `deploy_test.go` において、`Skipped = true` をアサートしている箇所をすべて削除、または `Skipped = false` に修正する。
- メタデータ書き込みや Git 検出のテストなど、廃止された内部ロジックに対するテストケースを削除する。

## 変更ファイル

### compiler/update.go
- `Update()` 関数の変更検出・メタデータ処理を削除。
- `shouldUpdate()`, `CheckForChanges()`, `ReadMetadata()`, `WriteMetadata()` 関数を削除。

### compiler/deploy.go
- `Deploy()` 内のダイジェスト比較によるスキップロジックを削除。
- 不要となったダイジェスト処理の呼び出しをクリーンアップ。

### compiler/compiler.go
- `Compile()` 内のバリデーションチェック通過後に、`buildDir` の削除および再作成を行う処理を追加。

## 検証計画 (Verification Plan)

### 自動テスト
- `update_test.go` と `deploy_test.go` を修正し、常にコンパイル/デプロイが実行されることを検証する。
- 以下のコマンドを実行してテストがすべて通過することを確認する:
  ```bash
  & "C:\Program Files\Git\bin\bash.exe" -c "go test ./features/tt/internal/prompt/compiler/..."
  ```

### 手動検証
1. `tt prompt update` を連続で2回実行し、2回目もスキップされずにコンパイルとデプロイが成功することを確認する。
2. `tmp/dist/` に適当な古いファイル（例: `tmp/dist/old-junk.txt`）を手動で作成した後、`tt prompt compile` を実行し、実行後に `tmp/dist/` の中身が完全にクリアされ、新しい生成ファイルのみが存在することを確認する。
3. `tt prompt deploy --mode immune` を実行し、デプロイ先（例: `.agents/`）の不要なファイルが正しく削除されることを確認する（`EmitMode` の機能維持確認）。

