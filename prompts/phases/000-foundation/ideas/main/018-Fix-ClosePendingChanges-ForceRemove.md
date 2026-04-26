# 018: close/delete で未コミット変更あり時に確認承認しても worktree 削除が失敗する問題の修正

## 背景 (Background)

`tt` で worktree を削除する際、対象 worktree に未コミット変更や未トラックファイルがあると警告と確認プロンプトが表示される。

しかし現状では、確認プロンプトで `y` / `yes` を入力して承認しても、内部で実行される `git worktree remove` に強制削除フラグが付与されず、以下のような失敗が発生して削除が完了しない。

- `git worktree remove` が `--force`（または `-f`）なしで実行される
- `contains modified or untracked files, use --force to delete` 系のエラーで停止する

このため、ユーザーが「強制実行を承認した」のに削除できない UX 不整合が起きている。

## 要件 (Requirements)

### 必須要件

1. 未コミット変更・未トラックファイル検出後の確認プロンプトで `y` / `yes` が入力された場合、worktree 削除処理に強制削除フラグを確実に伝播すること
2. 強制伝播時、`git worktree remove` が `-f`（または同等の `--force`）付きで実行されること
3. `close` と `delete` の双方で、同一の確認結果に対して同一の強制削除動作となること
4. `n` / 空入力 / その他入力時は既存どおり中断し、削除を実行しないこと
5. `--force` を明示指定したケース、`--yes` 指定ケース、通常ケースの既存動作を壊さないこと

### 任意要件

- 確認承認時のログに、強制削除へ切り替わったことが分かるメッセージを追加し、ユーザーが挙動を把握しやすくする

## 実現方針 (Implementation Approach)

1. pending changes 確認ロジックを「承認可否」だけでなく「強制削除の要否」も扱える形へ見直す
2. `close` 側で pending changes が検出され、ユーザーが承認した場合は、`DeleteOptions.Force` に反映したうえで `Delete` を呼び出す
3. `delete` 直実行パスでも同じ判定・反映ロジックを使い、`wm.Remove(branch, force)` まで確実に伝播させる
4. ユニットテストを追加/更新し、以下を固定化する
  - pending changes + `yes` 承認時に `worktree remove -f` が記録される
  - pending changes + 非承認時は `worktree remove` が呼ばれない
  - 既存 `--force` / `--yes` シナリオが回帰しない

主な修正対象想定:

- `pkg/action/pending_changes.go`
- `pkg/action/close.go`
- `pkg/action/delete.go`
- `pkg/action/close_test.go`
- `pkg/action/delete_test.go`

## 検証シナリオ (Verification Scenarios)

### シナリオ1: pending changes ありで承認した場合に強制削除される

1. 任意の worktree ブランチを用意し、未コミット変更または未トラックファイルを作る
2. `tt close <branch>`（または `tt delete <branch>`）を実行する
3. pending changes の警告と確認プロンプトが表示されることを確認する
4. プロンプトで `yes`（または `y`）を入力する
5. `git worktree remove -f ...` 相当で削除が進み、エラー終了しないことを確認する

### シナリオ2: pending changes ありで非承認なら中断される

1. pending changes を含む worktree で `tt close <branch>` を実行する
2. プロンプトで `n` を入力する
3. worktree / branch が削除されず、処理が中断されることを確認する

### シナリオ3: `--force` 明示指定の既存動作を維持する

1. pending changes の有無に関わらず `tt close --force <branch>` を実行する
2. `git worktree remove -f ...` が実行され、従来どおり削除できることを確認する

### シナリオ4: `--yes` の既存動作を維持する

1. pending changes を含む worktree で `tt close --yes <branch>` を実行する
2. 確認プロンプトをスキップした場合でも、仕様に沿った force 伝播で削除が完了することを確認する

## テスト項目 (Testing for the Requirements)


| 要件            | テスト方法                                                   | 検証コマンド                                        |
| ------------- | ------------------------------------------------------- | --------------------------------------------- |
| 承認時の force 伝播 | `close` / `delete` のユニットテストで `worktree remove -f` 記録を検証 | `go test ./pkg/action -run 'TestClose         |
| 非承認時の中断       | `n` 入力時に `worktree remove` 未実行を検証                       | `go test ./pkg/action -run 'TestClose_.*Abort |
| 既存挙動の回帰防止     | 既存 close/delete テスト一式の再実行                               | `go test ./pkg/action`                        |
| 全体整合性         | プロジェクト標準ビルドで回帰確認                                        | `./scripts/process/build.sh`                  |
| 統合テスト回帰       | プロジェクト標準統合テストで回帰確認                                      | `./scripts/process/integration_test.sh`       |
