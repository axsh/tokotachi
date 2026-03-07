---
trigger: always_on
---

# Antigravity Critical Issues

Windows版Antigravityの既知の問題をここに記載する。Mac版は該当しないので、無視してください。

## コマンド実行に関する重要な注意

[!IMPORTANT]
標準のコマンド実行ツール`run_command`は、実行を開始せずフリーズ/ハングアップする問題が発生しています。
そのため、**`run_command`ツールの利用は絶対に避けてください**。
**代替手段として、確実に動作するMCPコマンドランナー (`mcp_mcp-command-runner_execute_bash_command`) を利用してください**。

### コマンド実行時のセキュリティ制限

**重要**: すべてのパスはカレントワークディレクトリ(`cwd`)内に制限されます。CWD外へのアクセスはセキュリティガードレールにより拒否されます。

- ✅ プロジェクト内: `cat ./README.md`
- ❌ プロジェクト外: `cat ../../etc/passwd` → エラー