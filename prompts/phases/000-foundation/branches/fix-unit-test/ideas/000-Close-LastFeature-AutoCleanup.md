# close コマンド: 最後の feature 閉鎖時の自動クリーンアップ

## 背景 (Background)

`devctl close --feature <name>` は、指定された feature のコンテナを停止し、state ファイルから該当 feature エントリを削除する。
しかし、ブランチ上の **最後の feature** を閉じた場合でも、worktree の削除・ブランチの削除・state ファイル自体の削除が行われない。

これにより、以下の単体テストが FAIL している:

| テスト名 | 期待する動作 |
|---|---|
| `TestClose_WithFeature_LastFeature_CleansUpWorktree` | 最後の feature 閉鎖後に state ファイル削除 + worktree 削除 |
| `TestClose_WithFeature_Force_PropagatedToCleanup` | `Force=true` 時にworktree 削除に `-f`、branch 削除に `-D` が伝播 |

### 問題箇所

[close.go](file://features/devctl/internal/action/close.go) の feature-specific close パス（26〜49行目）は、`RemoveFeature()` で feature エントリを削除した後、残りの feature 数を確認せずに即 `return nil` している。

```go
// 現在の実装（簡略化）
if opts.Feature != "" {
    // コンテナ停止 ...
    sf.RemoveFeature(opts.Feature)
    state.Save(statePath, sf)
    return nil  // ← ここで終了。最後の feature だった場合の処理がない
}
```

一方、feature なし（全閉鎖）パス（52〜107行目）には、worktree 削除・ブランチ削除・state ファイル削除のロジックがすでに存在している。

## 要件 (Requirements)

### 必須要件

1. **最後の feature 閉鎖時の自動クリーンアップ**: `RemoveFeature()` 後に feature が 0 件になった場合、以下を順に実行する:
   - worktree の削除（`wm.Remove(branch, force)`)
   - ブランチの削除（`wm.DeleteBranch(branch, force)`)
   - state ファイルの削除（`state.Remove(statePath)`)

2. **Force フラグの伝播**: `CloseOptions.Force = true` の場合、worktree 削除とブランチ削除にフラグが正しく伝播されること:
   - `wm.Remove(branch, true)` → `git worktree remove -f <path>`
   - `wm.DeleteBranch(branch, true)` → `git branch -D <branch>`

3. **既存テストの維持**: 現在 PASS しているテスト（`TestClose_WithFeature_OtherFeaturesRemain_KeepsWorktree` と `TestClose_WithoutFeature_Unchanged`）が引き続き PASS すること。

### 任意要件

- 自動クリーンアップのログメッセージ出力（worktree 削除やブランチ削除の開始/完了を表示）

## 実現方針 (Implementation Approach)

### 修正対象ファイル

- [close.go](file://features/devctl/internal/action/close.go)（1ファイルのみ）

### 修正方針

`close.go` の feature-specific close パス（26〜49行目）において、`RemoveFeature()` + `Save()` の後に「残り feature 数が 0 か」をチェックするロジックを追加する。

```go
// 修正後の疑似コード
if opts.Feature != "" {
    // コンテナ停止 ...

    sf.RemoveFeature(opts.Feature)
    
    if len(sf.Features) == 0 {
        // 最後の feature → 全クリーンアップ
        // 1. worktree 削除
        if wm.Exists(opts.Branch) {
            wm.Remove(opts.Branch, opts.Force)
        }
        // 2. ブランチ削除
        wm.DeleteBranch(opts.Branch, opts.Force)
        // 3. state ファイル削除
        state.Remove(statePath)
    } else {
        // feature がまだ残っている → state を保存して終了
        state.Save(statePath, sf)
    }
    
    return nil
}
```

> [!NOTE]
> feature なしパス（52〜107行目）の既存クリーンアップロジックと同様の処理を、feature-specific パスにも条件付きで追加する形。重複を避けるために内部関数への切り出しも検討可能だが、テスト修正が主目的のためスコープ外とする。

## 検証シナリオ (Verification Scenarios)

1. feature が 1 件のみ → `close --feature myfeature` → worktree 削除 + ブランチ削除 + state ファイル削除が実行される
2. feature が 2 件以上 → `close --feature feature-a` → feature-a のみ削除、worktree とブランチはそのまま残る
3. `Force=true` で最後の feature を閉じる → worktree 削除に `-f`、ブランチ削除に `-D` が渡される
4. feature なし（全閉鎖）→ 既存動作と変わらない

## テスト項目 (Testing for the Requirements)

### 自動テスト

以下のコマンドですべてのテストが PASS することを確認する:

```bash
./scripts/process/build.sh
```

#### 対応表

| 要件 | テストケース | ファイル |
|---|---|---|
| 最後の feature 閉鎖時の自動クリーンアップ | `TestClose_WithFeature_LastFeature_CleansUpWorktree` | [close_test.go](file://features/devctl/internal/action/close_test.go):63 |
| Force フラグの伝播 | `TestClose_WithFeature_Force_PropagatedToCleanup` | [close_test.go](file://features/devctl/internal/action/close_test.go):140 |
| 既存テスト（複数 feature 時の動作） | `TestClose_WithFeature_OtherFeaturesRemain_KeepsWorktree` | [close_test.go](file://features/devctl/internal/action/close_test.go):98 |
| 既存テスト（全閉鎖） | `TestClose_WithoutFeature_Unchanged` | [close_test.go](file://features/devctl/internal/action/close_test.go):174 |

> [!IMPORTANT]
> テストコード（`close_test.go`）の修正は不要。実装コード（`close.go`）のみの修正で、既存テストが PASS するようになることが期待される。
