# semantic-judge スタブ

## 目的

`warrant advise` の semantic judge ポート(外部コマンド契約)の参照実装。LLM を呼ばずに決定論的な `uncertain` を返すスタブであり、以下の用途を想定する。

- judge ポートの契約(Request/Verdict JSON)の動作確認
- ローカル開発・CI での `warrant advise` の結合テスト
- 置換性のデモ: 標準ライブラリのみで第三者が再実装できることを示す

このスタブは置換性のデモのため `internal/semantic` の型を意図的に複製している。`main_test.go` の契約テスト(`WARRANT-ADVISE` のカバレッジ)が、複製した struct と本体 `internal/semantic.Request`/`Verdict` の JSON フィールドが双方向で一致することを検証する。契約がドリフトした瞬間に `go test ./...` が失敗するため、参照実装が腐らない。

## JSON 契約

### Request (stdin)

| フィールド | 型 | 説明 |
|---|---|---|
| `rule_id` | string | ルール識別子 |
| `title` | string | ルールタイトル |
| `basis` | string | 判定根拠テキスト |
| `criterion` | string | 判定基準 |
| `targets` | []string | 判定対象ファイル一覧 |

### Verdict (stdout)

| フィールド | 型 | 説明 |
|---|---|---|
| `verdict` | string | `"pass"` / `"fail"` / `"uncertain"` |
| `rationale` | string | 判定理由 |
| `proposed_assertion` | string | 提案アサーション(省略可) |

## 配線手順

`.warrant/config.yaml` の `semantic_command` にこのスタブを指定する。

### go run で直接使う

```yaml
semantic_command: "go run ./examples/semantic-judge"
```

### ビルドして使う

```bash
go build -o /tmp/warrant-judge ./examples/semantic-judge
```

```yaml
semantic_command: "/tmp/warrant-judge"
```

## 動作確認

```bash
echo '{"rule_id":"RULE-X","title":"t","basis":"b","criterion":"c","targets":["a.go"]}' \
  | go run ./examples/semantic-judge
```

期待出力(整形は環境依存):

```json
{"verdict":"uncertain","rationale":"[参照スタブ] rule_id=\"RULE-X\" criterion=\"c\" targets件数=1 — これは参照スタブであり実際の意味判定はしていない。実運用では LLM 等に置き換えること。"}
```

## 実運用での差し替え

実運用では `semantic_command` を LLM API を呼ぶ実装に差し替える。契約(Request/Verdict JSON)を満たす任意の言語・ランタイムで実装できる。
