# Features

このディレクトリには、プロジェクトの各機能（feature）が格納されます。

各 feature は独立したモジュールで、コードベース・テスト・設定を持ちます。

## Feature の追加

`devctl scaffold features <template>` で新しい feature 構成を生成します。

## ディレクトリ規約

```
features/
  <feature-name>/
    cmd/           # CLI エントリポイント
    internal/      # 内部パッケージ
    go.mod         # Go モジュール定義
```
