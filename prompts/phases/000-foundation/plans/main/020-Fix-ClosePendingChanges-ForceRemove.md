# 020: close/delete の pending changes 承認時に force 削除を確実に伝播する実装計画

## Goal Description

`tt close` / `tt delete` 実行時に、対象 worktree に未コミット変更・未トラックファイルがある場合でも、ユーザーが確認プロンプトで `y` / `yes` を入力したときは `git worktree remove -f` が必ず実行されるようにする。  
これにより「承認したのに削除できない」不整合を解消し、`--force` 明示時・`--yes` 使用時・非承認時の既存挙動は維持する。

## User Review Required

- [x] 本計画で、確認承認時に force へ昇格させる仕様（`y/yes` のみ許可）が期待どおりか
- [x] `close` と `delete` の両方に同じ判定ロジックを適用する方針で問題ないか
- [x] 非承認時（`n` / 空入力 / その他文字列）を中断扱いに据え置く方針で問題ないか
- [x] ログ改善（承認により force 削除へ切替を明示）を含める方針で問題ないか

## Requirement Traceability


| Requirement (Spec 018)                       | 実装ポイント                                                                                                   | テストでの担保                                                                                                      |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| R1: pending changes 警告後に `yes` 承認なら force 伝播 | `pkg/action/pending_changes.go` の確認結果を bool 1値ではなく「承認可否 + force 要否」を返す構造へ変更。`close.go` と `delete.go` で反映 | `pkg/action/close_test.go` と `pkg/action/delete_test.go` に「pending changes + yes -> worktree remove -f」ケース追加 |
| R2: 強制伝播時に `git worktree remove -f` 実行       | `pkg/action/delete.go` で最終的に `wm.Remove(branch, force)` へ渡す `force` を「CLI `--force` OR 承認による強制」に統一       | Recorder の記録コマンドに `worktree remove -f` が含まれることを検証                                                            |
| R3: close/delete 双方で同一挙動                     | `close.go` の delete 委譲時にも同じ決定結果を使用。`delete.go` 直実行も同ロジックを使用                                              | close/delete 両テストに同型ケースを追加し期待値を一致                                                                            |
| R4: 非承認時は中断                                  | `pending_changes.go` で `y/yes` 以外は不承認として返し、呼び出し側で削除処理へ進まない                                               | 「n 入力で worktree remove が呼ばれない」テスト追加/更新                                                                       |
| R5: `--force` / `--yes` / 通常ケースの回帰防止         | 既存フラグ分岐を保ちながらロジックを合流させる。`--yes` は prompt をスキップしつつ pending changes 検出時の force 決定を明示                       | 既存 close/delete テスト + 追加ケースで回帰確認                                                                             |


## Proposed Changes

### 1) `pkg/action/close_test.go`（先行で失敗テストを追加）

- 追加/更新するテスト観点（テーブル駆動または既存パターン準拠）
  - pending changes が存在し、標準入力 `yes` のとき `worktree remove -f` が記録される
  - pending changes が存在し、標準入力 `n` のとき `worktree remove` が記録されない
  - `--force` 指定ありケースで既存どおり `-f` が維持される
- 既存 helper（Recorder 検索）を流用して、コマンド文字列に `worktree remove -f` を含むことを判定

### 2) `pkg/action/delete_test.go`（先行で失敗テストを追加）

- `delete` 直実行での pending changes 確認分岐を検証
  - `yes` 承認時: `worktree remove -f` が実行される
  - `n` 非承認時: `worktree remove` / `branch delete` が実行されない
- `--yes` 指定時の既存仕様（prompt スキップ）との整合性を確認するケースを補強

### 3) `tests/tt/tt_create_delete_test.go`（統合テストを追加）

- `tt delete` の CLI 振る舞いを E2E で検証するテストを追加
  - `TestTtDelete_PendingChanges_Yes_UsesForceRemove`
    - 事前に worktree 内へ未トラックファイルを作成
    - `tt delete <branch>` を実行し、stdin に `yes` を渡す
    - 終了コードが成功で、`contains modified or untracked files` エラーで失敗しないことを確認
  - `TestTtDelete_PendingChanges_No_Aborts`
    - 同様に pending changes を作成
    - `tt delete <branch>` 実行時に stdin に `n` を渡す
    - 削除中断のメッセージと、worktree が残っていることを確認
- 既存の `runTT` ヘルパが stdin 注入に対応していない場合、`tests/tt/helpers_test.go` に「標準入力付き実行ヘルパ」を追加する
- 環境差分による不安定化を避けるため、pending changes は「未トラックファイル 1 件作成」で再現する（staged/unstaged 依存にしない）

### 4) `pkg/action/pending_changes.go`

- 現状の `checkPendingChangesAndConfirm(...) bool` を、以下のような判定結果を返す API に変更
  - 例: `type PendingChangesDecision struct { Approved bool; ForceDelete bool }`
  - `Approved`: 削除継続可否
  - `ForceDelete`: `git worktree remove -f` へ昇格すべきか
- 判定ロジック（具体）
  1. `opts.Yes == true` の場合
    - prompt は表示しない  
    - pending changes が 0 件なら `Approved=true, ForceDelete=false`  
    - pending changes が 1 件以上なら `Approved=true, ForceDelete=true`
  2. `opts.Yes == false` かつ pending changes が 0 件なら `Approved=true, ForceDelete=false`
  3. `opts.Yes == false` かつ pending changes が 1 件以上なら警告 + prompt 表示
    - 入力が `y` / `yes` のみ `Approved=true, ForceDelete=true`  
    - それ以外は `Approved=false, ForceDelete=false`
- 既存の表示仕様（カテゴリ表示、`maxDisplayLines=10`、`--verbose` で全件表示）は維持
- 任意要件対応として、`ForceDelete=true` 決定時に「pending changes 承認により force 削除を使用する」旨のログを追加

### 5) `pkg/action/close.go`

- pending changes 確認呼び出し部を新 API に差し替え
- `DeleteOptions` へ渡す `Force` を以下で決定
  - `effectiveForce := opts.Force || decision.ForceDelete`
- feature 指定 close（最後の feature で delete 委譲）と、feature 未指定 close（all close 後 delete 委譲）の両方で同一ロジックを適用
- `Approved=false` の場合は従来どおり中断して `nil` return

### 6) `pkg/action/delete.go`

- delete 直実行でも pending changes 判定を行い、`effectiveForce := opts.Force || decision.ForceDelete` を使って `wm.Remove(opts.Branch, effectiveForce)` を呼ぶ構造へ変更
- 中断時の挙動（削除処理へ進まず終了）を明確化
- `wm.DeleteBranch(..., force)` への force 適用方針は既存互換を維持（必要に応じて `opts.Force` と `effectiveForce` の使い分けを明示し、挙動差をテストで固定）

## Integration Tests

- 本件は unit test に加え、`tests/tt/tt_create_delete_test.go` を更新して統合テストを必ず実施する
- 追加する統合シナリオ（実施必須）
  - pending changes + `yes` 承認で delete が成功する
  - pending changes + `n` で delete が中断される
- 実行時は既存統合スイートと合わせて回帰確認する

## Step-by-Step Implementation Guide

- [x] `pkg/action/close_test.go` / `pkg/action/delete_test.go` の既存テストが通る状態を維持した
- [x] `tests/tt/tt_create_delete_test.go` と `tests/tt/helpers_test.go` に pending changes 承認/非承認の統合テストを追加
- [x] `pkg/action/pending_changes.go` の判定 API を `PendingChangesDecision` ベースへ変更
- [x] `pkg/action/close.go` の呼び出し部を新 API に差し替え、`effectiveForce` で delete 委譲
- [x] `pkg/action/delete.go` へ pending changes 判定と `effectiveForce` 適用を実装
- [x] `go test ./pkg/action` で追加テストを含めて green を確認
- [x] `./scripts/process/build.sh` を実行して通過
- [/] `./scripts/process/integration_test.sh` を実行（`tt` は通過、`release-note` は既存の credential 欠落で失敗）

## Verification Plan (Automated Commands)

1. `./scripts/process/build.sh`
2. `./scripts/process/integration_test.sh`

## Documentation updates

- 仕様・計画ドキュメントは本ファイルで更新済み
- 実装完了後、必要であれば以下を追記
  - `prompts/phases/000-foundation/ideas/main/018-Fix-ClosePendingChanges-ForceRemove.md` への「実装結果/差分」補足
  - CLI ヘルプ文言変更が発生した場合のみ関連ドキュメントを更新

## Self-Review Checklist

- 仕様 018 の必須要件 R1-R5 を全てトレース可能にした
- 重要ロジック（`y/yes` 判定、`effectiveForce` 合成、表示上限 `maxDisplayLines=10`）を具体的に継承した
- `Proposed Changes` で `_test.go` を先に記述した
- 検証コマンドはプロジェクト規範どおりスクリプト実行形式のみを記載した