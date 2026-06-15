# 仕様: セマンティック advisory (advise コマンド)

## 概要

`warrant advise` は LLM などの外部 Judge コマンドを利用した意味的チェックを advisory として提供する。**CI ゲートではない**。

`kind: semantic` の `enforced_by` エントリを持つルールに対し、外部 Judge コマンドを呼び出してセマンティック評価を行う。

## 不変条件

- 常に exit 0 で終了する
- judge コマンドが失敗・タイムアウトしても続行する
- `semantic_command` が未設定でも正常終了する
- CI ゲートではないため、`fail` や `uncertain` の verdict が出ても exit 1 にならない

## Request / Verdict JSON 契約

### Request (stdin として Judge コマンドへ渡す)

| フィールド | 型 | 説明 |
|---|---|---|
| `rule_id` | string | ルール ID |
| `title` | string | ルールタイトル |
| `basis` | string | 立法根拠（constitution アンカー） |
| `criterion` | string | 判定基準（enforced_by.criterion） |
| `targets` | []string | governs グロブで展開されたファイルパス一覧 |

### Verdict (Judge コマンドの stdout から受け取る)

| フィールド | 型 | 説明 |
|---|---|---|
| `verdict` | string | `"pass"` / `"fail"` / `"uncertain"` |
| `rationale` | string | 判定理由 |
| `proposed_assertion` | string | （任意）提案するアサーション文字列 |

## 外部コマンド契約

- `semantic_command` に指定したコマンドを `sh -c <command>` で起動する
- Request JSON を stdin に渡す
- Judge は Verdict JSON を stdout に出力する
- タイムアウトは `semantic_timeout_sec`（既定: 30秒）

## 設計理由

「AI が提案・人間が承認」原則に基づき、LLM の意味判定はあくまで advisory として扱う。決定論的なゲートとは分離し、セマンティックチェックの結果は参考情報として提示する。

決定論ゲート（`kind: deterministic`）と semantic advisory（`kind: semantic`）を明確に分離することで、CI の安定性を保ちながら LLM によるセマンティック評価の恩恵を受けられる。
