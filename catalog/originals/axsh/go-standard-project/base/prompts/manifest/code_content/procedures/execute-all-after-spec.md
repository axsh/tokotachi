---
apiVersion: agent.meta/v1
id: execute-all-after-spec
kind: procedure
title: Execute All After Spec
trigger:
    command: execute-all-after-spec
description: After a confirmed specification, create the implementation plan, implement it, and write the reproducible report without stopping between those steps.
tags:
  - baseline
steps:
  - create-implementation-plan
  - execute-implementation-plan
  - create-reproducible-report

---

# 仕様確認後の連続実行

確認済みの仕様書から、実装計画、実装、再現手順までを一度に終わらせる。各手順の中身は、それぞれの手順書に従う。この手順は、その三つを止めずに順番へつなぐ。

## 1. 入力

1. ユーザーが指定した仕様書を入力にする。指定が無ければ、エディタで開いている `ideas/` のファイルを仕様書にする。
2. それが仕様書でなければ、実装へ進まず、どの仕様書かを確認する。
3. 仕様書は変更しない。調査や新しい仕様書の作成は、この手順の中で始めない。

## 2. 順番

次の順で、各手順書を最初から最後まで実行する。

1. **create-implementation-plan**
    * 入力は、§1 の仕様書である。
    * 計画書を作り、その手順書どおりにセルフレビューし、計画書だけをコミットする。
2. **execute-implementation-plan**
    * 入力は、直前に作った計画書である。分割されている場合は、そのすべての計画書である。
    * 実装、テスト、遠方知識の記録、`git push` まで、その手順書どおりに行う。
3. **create-reproducible-report**
    * 入力は、§1 の仕様書と、§2 の計画書である。
    * レポートを実行して書き、レポートだけをコミットする。

各手順書に書いてある判断、テンプレート、検証、コミットの単位は省略しない。この手順は、それらを要約して置き換えない。

## 3. 止めないこと

仕様書の確認が、この連続実行の承認である。計画書ができたところで、ユーザーのレビューを待たない。計画の `User Review Required` は `None.` とし、実装へ進む。

仕様書だけでは実装の判断が足りないときだけ止まる。足りない判断を計画の `User Review Required` に書き、実装と再現手順には進まない。

途中の手順が終わったことだけでは、ユーザーへの完了報告を区切らない。三つの手順がすべて終わったあと、一度だけ結果を報告する。

## 4. 失敗したとき

ある手順が失敗したら、次の手順を始めない。

* 計画書を書けないときは、実装しない。
* ビルドまたはテストが失敗しているときは、再現手順を書かない。失敗は、execute-implementation-plan の修正ループで直す。
* 再現手順の結果が仕様と違うときは、ソース、テスト、仕様書、計画書を直さない。実際の結果をレポートに書き、ユーザーに報告する。

## 5. Git

* 計画書のコミットは、create-implementation-plan の完了時に行う。
* 実装のコミット、遠方知識の記録、`git push` は、execute-implementation-plan に従う。この push は、再現手順のレポートを作る前に行う。
* 再現手順のコミットは、create-reproducible-report に従う。このレポートは push しない。
* 関係のない変更は、どのコミットにも含めない。
