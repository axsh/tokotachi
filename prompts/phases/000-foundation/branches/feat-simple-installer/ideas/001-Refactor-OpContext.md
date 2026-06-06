# tokotachi.go コンテキスト外部化と合成関数リファクタリング

## 背景

`tokotachi.go` の各公開メソッド (`Create`, `Up`, `Down`, `Open`, `Close`, `Status`, `List` 等) は、共通のコンテキスト (`logger`, `runner`, `worktree.Manager`, `actionRunner`) を**メソッド内で毎回独立に作成**している。

この設計には以下の問題がある:

1. **コード重複**: `newContext()` + `cmdexec.NewRecorder()` + `cmdexec.Runner` + `worktree.Manager` の組み立てが全メソッドで重複
2. **合成困難**: `Open` は `Create → Up → Editor` を呼ぶべきだが、各メソッドが独自コンテキストを生成するため**単純に呼び出せず**、ロジックがインラインでコピーされている（約150行の重複）
3. **テスト困難**: コンテキストがメソッド内部に閉じているため、外部からモックを差し込みにくい

## 要件

### 必須

1. **コンテキスト構造体の導入**: `newContext()` が返す logger, runner, actionRunner, worktree.Manager をまとめた構造体 `opContext` を導入する
2. **各メソッドのシグネチャ変更**: 各プリミティブメソッド (`Create`, `Up`, `Down`) に `opContext` を受け取る内部版メソッドを用意し、公開版は `opContext` を生成してそれを呼ぶだけにする
3. **`Open` のリファクタリング**: `opContext` を1つ作成し、`create → up → editor` の各内部メソッドを順番に呼ぶシンプルな構造にする
4. **`Close` のリファクタリング**: 同様に `opContext` を共有して内部メソッドを組み合わせる（現在は `actionRunner.Close` 1回の呼び出しなので単純化の余地は小さいが、コンテキスト生成は統一する）
5. **既存テスト + ビルドが通ること**

### 任意

- 将来的にメソッドチェーンや Pipeline パターンに対応できる拡張性の確保
- `Recorder` の記録を合成操作全体で統一する

## 実現方針

### 1. `opContext` 構造体の導入

```go
// opContext holds shared objects for a single operation.
type opContext struct {
    logger       *log.Logger
    runner       *cmdexec.Runner
    actionRunner *action.Runner
    wm           *worktree.Manager
    projectName  string
}
```

`Client.newOpContext()` メソッドで一括生成:

```go
func (c *Client) newOpContext() *opContext {
    logger, runner, actionRunner := c.newContext()
    wm := &worktree.Manager{CmdRunner: runner, RepoRoot: c.RepoRoot}
    return &opContext{
        logger:       logger,
        runner:       runner,
        actionRunner: actionRunner,
        wm:           wm,
        projectName:  c.resolveProjectName(),
    }
}
```

### 2. 各メソッドの分割

**パターン**: 公開メソッド → `opContext` 生成 → 内部メソッド呼び出し

```go
// Public (外部API)
func (c *Client) Create(branch string, opts CreateOptions) error {
    ctx := c.newOpContext()
    return c.doCreate(ctx, branch, opts)
}

// Internal (合成可能)
func (c *Client) doCreate(ctx *opContext, branch string, opts CreateOptions) error {
    // 実装ロジック（コンテキスト生成なし）
}
```

同様に `doUp`, `doDown`, `doEditor` を作成する。

### 3. `Open` のリファクタリング

```go
func (c *Client) Open(branch, feature string, opts OpenOptions) error {
    ctx := c.newOpContext()

    // Step 1: Create worktree
    if err := c.doCreate(ctx, branch, CreateOptions{}); err != nil {
        return err
    }

    // Step 2: Up container (if feature specified)
    if feature != "" {
        if err := c.doUp(ctx, branch, feature, UpOptions{}); err != nil {
            return err
        }
    }

    // Step 3: Open editor
    return c.doEditor(ctx, branch, feature, opts.Editor)
}
```

### 4. 各 `doXxx` メソッドでの差異吸収

`Open` 内の `Up` ステップは、通常の `Up` と微妙に異なる（コンテナ running チェックがある）。この差異は `UpOptions` にフラグを追加するか、`doUp` 内で `SkipIfRunning` のような制御を入れる:

```go
type UpOptions struct {
    SSH           bool
    Rebuild       bool
    NoBuild       bool
    SkipIfRunning bool // Open から呼ぶ時は true
}
```

### 5. `projectName` の扱い

各 cmd ファイル (`up.go`, `status.go` 等) にある以下のパターン:

```go
projectName := "tt"
if projectName == "" {
    projectName = "tt"
}
```

は冗長（常に `"tt"`）。`opContext.projectName` から取るように統一できるが、cmd 側は `tokotachi.go` の Client API を直接使っているわけではないため、この仕様書の対象外とし、`tokotachi.go` 内のみを対象にする。

## 検証シナリオ

1. `Open("my-branch", "tt", ...)` を呼ぶと、Create → Up → Editor が順番に実行される
2. 既に worktree が存在する場合、Create ステップはスキップされる
3. feature が空の場合、Up ステップはスキップされ、Editor のみ実行される
4. 各プリミティブメソッド (`Create`, `Up`, `Down`, `Close`) を個別に呼んだ場合も、従来と同じ挙動をする
5. `DryRun` モードで `Open` を呼ぶと、全ステップが DryRun で実行される

## テスト項目

| 要件 | 検証方法 |
|------|----------|
| リファクタリング後にビルドが通る | `scripts/process/build.sh` |
| 既存テストがパスする | `scripts/process/build.sh` (含: 単体テスト) |
| `Open` が Create → Up → Editor の合成になっている | コードレビュー + 単体テスト |
