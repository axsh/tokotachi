---
apiVersion: agent.meta/v1
kind: capability
id: knowledge-database
uid: d065f2b1-b1f6-463f-ab61-0d749f13afff
title: 知識データベース
description: >-
  知識データベース（知識DB、KDB）を新しく設計するときに使う。
  何の知識を、どの形のデータベースと、どの CLI と、どのタイミングのスキルで扱うかを決める。
body: inline
---

# 知識データベース

知識データベース（知識DB、KDB）とは、次の四つをひとまとめにした仕組みである。

1. 知識を置くデータベース。表、文書、グラフ、埋め込み検索のどれにするかは、知識の形に合わせて選ぶ。
2. そこへ入るゲートである目的別 CLI。エージェントはデータベースへ直に入らない。
3. CLI の `help` と、そのサブコマンドが返す AI 向けの取り扱い説明。データベースが止まっていても読める。
4. その CLI を、いつ、何のために呼ぶかを書いたスキル。呼ぶ契機をスキルに書き、CLI の呼び出しに触れることで、エージェントと知識DBが連動する。

この四つを合わせて知識データベースと呼ぶ。画面 ID や画面遷移は、その一例である。将来の知識DBは、この例の表やグラフに限らない。新しい知識が必要になったら、新しい知識DBとして設計する。

## フレームワーク核

他リポジトリへ持ち出すときも揃える契約である。個別の知識ドメイン（画面在庫、表定義、Tern、製品 RLS、Embedding キー、フェーズ番号）は核に含めない。

1. 知識DB = 知識ストア + 目的別 CLI + help + タイミングスキル
2. 段階 = 蓄積 → 蒸留 → 退避 → 再構築
3. 置き場 = `{kdb-root}/<kdb-id>/data/`（gitignore）・`{kdb-root}/<kdb-id>/corpus/`（Git 正本）・`tmp/kdb/<kdb-id>/`（現状ダンプ）。既定の kdb-root は `shared/kdb`
4. 退避 = distill → dump(tmp) → 単純マージ(tmp, data) → pack(data→corpus)
5. 再構築 = unpack(corpus→data) → restore(data→DB)。鮮度で corpus が新しければ実施する
6. 単純マージは結合のみ。重複整理は各知識の distill
7. help は英語・固定見出し・DB 停止時も読める。操作成功は stdout の JSON 一つ
8. 遠方知識の systematize とは別系統。sync 前に両方を片付ける
9. 製品 DB の RLS / 保持は自動適用しない
10. **レコード UUID（テーブル ID と並列）** — 下節「レコード UUID の設計意図」
このリポジトリが追跡する知識DBの一覧は registry（既定は `shared/kdb/registry.yaml`、`--registry` / `KDB_REGISTRY`、または `--kdb-root` / `KDB_ROOT` で変更できる）である。registry が無くても **backlog は内蔵として常に登録**される。yaml に同 id があれば yaml 優先。全知識DBの追跡は `tt kdb-track`（`list` / `unregistered` / `status-cmds` / `status-all` / `systematize` / `skeleton`）が行い、一つの知識DBへの操作は `tt kdb <id> <operation>` が行う。pre-sync と `systematize-knowledge-database` は `tt kdb-track systematize` を呼ぶ。bash スクリプトは使わない。使い方の説明は `tt help kdb-track` と `tt help kdb` で読む。registry の `*_cmd` は引数付きコマンドライン文字列でよい。

backlog は `tt` に内蔵された知識DBである。操作は `tt kdb backlog <op>`（`add` / `list` / `get` / `update` / `resolve` / `status` / `distill` / `dump` / `merge` / `pack` / `unpack` / `restore` / `import-md` / `migrate-uuids`）で、説明は `tt help kdb backlog` と `tt help kdb backlog <op>` で読む。埋め込み SQLite で動くので、Python と外部の `sqlite3` は要らない。既定の置き場は `shared/kdb/backlog/`（corpus / data）。registry の `module_path` / `corpus_path` で上書きできる。

### レコード UUID の設計意図

知識DBの**永続レコード**（共有正本に載せる行・節・辺など）は、グローバル一意の **UUID を必須**とする。レコード間の相互参照や、dump／corpus／restore の突き合わせは **UUID を正**にする。

これは、テーブル都合の ID（連番 PRIMARY KEY、複合キー、局所コードなど）の設計を UUID に置き換える要件**ではない**。テーブル ID の形は各知識DBの表設計に任せ、SQL 上の処理は一般的な ID で行ってよい。UUID はそれと**並列**に持つ。

意図の核心:

1. 退避したデータを DB に復元するとき、**以前と同じテーブル ID である必要はない**。保存データに UUID があるので、それを正として同一物を対応付け、復元先では ID を改めて振ってよい。
2. 連番 ID を共有正本の正にしてしまうと、マージなどの加工のたびに「連番のまま整合するか」を検討しなければならなくなる。UUID が関連を維持していれば、**新たに振られた ID でも関係は続く**。
3. したがって、スキーマを「PRIMARY KEY = UUID」に縛る必要はない。エージェントや文書が指すのも UUID である。

example（designdb）へいつ列を足すかは、その知識DBの仕様／バックログで決める。核として揃えるのは「UUID を持つ・参照と退避の正にする・テーブル ID と並列」という契約である。

## 正本と遠方知識

立て付け（四構成・四段階・置き場・help 規約・設計手順・フレームワーク核）の正本は本スキル（manifest）である。遠方知識・チャット・調査メモを正本にしない。遠方知識に書くときは、本スキルへのポインタとプロジェクト固有の束縛だけにする。設計全文の再述は禁止する。既存の `__far-knowledge-agent-tooling-tern-kdb` 内の四構成再述は、体系化時にポインタへ縮約する（本スキル更新だけでは全文削除しない）。

## 共有化メモとの用語対応

| 説明メモの言い方 | 実装・スキルでの言い方 |
| --- | --- |
| ブランチ終了時に YAML/JSON を `corpus/` へ | dump は `tmp/kdb/<kdb-id>/` → 単純マージで `data/` → pack で `corpus/` |
| corpus と直接マージして取り込み | unpack(corpus→data) → restore(data→DB)。作業中のマージ相手は主に tmp と data |
| マージは CLI の責任 | 共通層は単純結合。重複畳み込みは各知識の distill |
| 正本は corpus | Git 共有正本は corpus。ロード元は常に（マージ後の）data |
| （メモに無い）蒸留 | 蓄積のあとに distill を必ず概念として置く |
| （メモに無い）遠方知識 | systematize-far-knowledge と知識DB退避は別。pre-sync で両方を片付ける |

## アーキテクチャ

共有の正本は、リポジトリ上の `{kdb-root}/<kdb-id>/corpus/`（既定 `shared/kdb/`）である。作業中の展開は `{kdb-root}/<kdb-id>/data/`（gitignore、JSON）である。知識を置くデータベースは作業用のキャッシュである。チャットの記憶や、遠方知識のメモを正本にしない。

現状ダンプは `tmp/kdb/<kdb-id>/` へ出し、既存の `data/` と**単純マージ**する。重複の畳み込みは共通マージでは行わず、各知識 CLI の蒸留（distill）に任せる。PR 前の退避は distill → dump(tmp) → 単純マージ → pack(data→corpus) である。

ゲートは目的別の CLI にする。探す、足す、直す、消す、関係を結ぶ、蒸留する、退避する、再構築する、といった目的ごとに入口を分ける。エージェントに生の SQL や、データベースの接続文字列を渡して操作させない。

説明は CLI の `help` に置く。いつ呼ぶかはスキルに置く。スキルは操作マニュアルの全文を写さない。コマンド名と、呼ぶ契機だけを書き、操作の詳細は `help` を読むよう指示する。

すべての知識DBは、概念上次の四段階を持つ。

1. **蓄積**: 作業中に CLI でキャッシュへ書く。
2. **蒸留**: 矛盾・古い値の上書き・相反する知識セットを整理する。やることが無くてもフェーズとして残し、報告を出す。
3. **退避**: 蒸留後の集合を tmp へ dump し、data と単純マージしたうえで corpus へ pack し、Git で共有する。
4. **再構築**: corpus を data へ unpack し、data を正としてキャッシュへ入れ直す。鮮度チェックで corpus 側が新しければ入れ直す。

一つの知識DBにつき、次のスキルをセットで設計する。

- このスキル。知識DBを新しく作るときの共通の手順である。個別の知識の操作手順は書かない。
- タイミングのスキル。その知識を参照する契機と、蓄える契機を、実際の CLI コマンドで書く。知識の種類ごとに分ける。
- 退避の手順（procedure `systematize-knowledge-database`）。PR や同期の前に、蒸留と dump／マージ／pack を行い、未退避を残さない。

画面 ID の一例（example）では、キャッシュは `designdb`、正本は registry の `corpus_path`（例: `shared/kdb/designdb/corpus/`）、展開は対になる `data/`、入口は `scripts/design/id.sh` と `scripts/design/flow.sh` に加え `distill.sh` / `dump.sh` / `restore.sh` / `status.sh` / `pack.sh` / `unpack.sh` / `merge.sh` / `sync.sh`、タイミングのスキルは `design-screen-id` と `design-screen-flow`、退避は `systematize-knowledge-database` である。これらのパスは一例であり、追跡対象かどうかは `shared/kdb/registry.yaml`（または `--registry`）が決める。新しい知識を、この一例へ自動では足さない。足すか、別の知識DBにするかは、設計のときに決める。

製品データベースの RLS と保持期間は、知識DBへ自動では適用しない。適用するか、別のデータベースに分けるかは、その知識DBの仕様で決める。

## CLI の help の規約

知識DBの CLI は、次の規約で `help` を書く。

- 本文は英語にする。
- `help` と、空白で掘るサブコマンドを持つ。例: `help`、`help id`、`help id search`。
- 各ページの見出しは、この順にする。Purpose、When to use、When not to use、Preconditions、Input and output、Exit codes、Read next。
- Read next には、実在する `help` トピックだけを書く。
- `help` はデータベースが止まっていても、終了コード 0 で読める。
- 操作の成功は、標準出力の JSON 一つである。
- `help` の成功は、標準出力の Markdown である。
- 失敗は JSON である（bash の CLI は標準エラー。`tt kdb` / `tt kdb-track` は標準出力の先頭行）。終了コードの意味を Exit codes に書き、次に読む `help` を失敗 JSON から辿れるようにする。
- ログに、API キー、接続文字列のパスワード、認可ヘッダを出さない。

個別のコマンドの引数、閾値、終了コードの番号は、その知識DBの `help` に書く。このスキルへ写さない。

## 設計の手順

知識DBを作る依頼が来たら、実装の前に、次の順で設計する。仕様に書けない点は、仮定で埋めず、依頼主であるユーザに問う。

1. 管理したい知識を一文で書く。誰が、何を、参照し、何を蓄えるか。既存の知識DBのどれかに属するか、新しい知識かも書く。
2. 知識の構成を洗い出す。識別子、呼び名、関係、文書、履歴、古い値を残すか、何を一意にするか。**各永続レコードの UUID**（作成時に発行し dump で保持）と、テーブル都合の ID（連番など）を分けて書く。参照と退避の正は UUID である（「レコード UUID の設計意図」）。
3. 構成に合うデータベースの形を選ぶ。一つに限らなくてよい。
    - 表。項目が安定し、一意や関連を制約で守るとき。表の PK は一般的な ID でよく、UUID 列を並列に持つ。
    - 非構造化ドキュメント。形が毎回違う文書を、そのまま残すとき。
    - グラフ。節と辺そのものが知識であるとき。画面遷移は、この形の一例である。辺の端点も UUID で指せるようにする。
    - RAG。言い回しから、意味の近い知識を引くとき。Embedding モデル、次元、保存するモデル名、キーが無いときの動きを決める。
4. 形が決まってから CLI を決める。目的別の入口と、`help` のトピックを、上の規約で設計する。データベースと連携するだけでなく、知識をどう扱うかも決める。読み書きだけで足りるか。Tern を介したコーディングエージェントに、取り込みや整形をさせるか。Embedding や LLM で、検索、要約、分類をするか。取得・参照の入口は UUID でも辿れるようにする。
5. 初期データを決める。空の台帳から、これから蓄えるのか。すでにいる仕様書、一覧、文書から、初期データを作るのか。初期データを作る処理から、外部の API を呼ぶのか。移行時もレコードごとに UUID を一度付け、以降は保持する。
6. タイミングのスキルを設計する。参照する契機と、蓄える契機を、CLI のコマンド名で書く。登録の前に探す、など、誤った重複を防ぐ契機を先に書く。
7. Git 共有の契約を決める。正本の置き場（`{kdb-root}/<id>/corpus/`）、展開（`data/`）、tmp ダンプ、単純マージ、**UUID による突き合わせ**（テーブル ID の保存必須化はしない）、チャンク命名、蒸留の検査、鮮度チェック、退避 procedure、pre-sync での未退避検出。
8. 本システムのデータベースと分けるか、RLS や保持期間を付けるかを、仕様に書く。

## 依頼主に問うこと

次が仕様から決まらないときは、実装へ進まずに問う。

- その知識の利用者と、参照と蓄積のどちらが主か。
- 一意にしたい値と、古い値を残すか。
- 表、文書、グラフ、RAG のどれが合うか。組み合わせるか。
- 既存の知識DBへ足すか、新しく作るか。
- 製品のデータベースと分けるか。守り（RLS、保持期間、権限）を付けるか。
- 初期データは空から蓄えるか、既存の情報から作るか。
