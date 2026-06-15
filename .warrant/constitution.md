# warrant 憲法

warrant 自身の設計原則を記述する。

## 決定性ゲートの原則 {#determinism}

ゲート経路（warrant check の判定ロジック）に非決定的判定を入れない。

具体的には以下を禁止する:
- LLM 推論による判定
- `time.Now()` / `time.Now` を判定分岐に使用すること（出力メタデータは除く）
- `math/rand` 等の乱数を判定に使用すること

全判定は SSOT（requirements.yaml / rules.yaml）からゼロ計算し、ソートで順序を固定する。

## serve 読み取り専用の原則 {#serve-readonly}

`warrant serve` が提供する HTTP サーバは読み取り専用でなければならない。

具体的には以下を保証する:
- `GET /` 以外のメソッド・パスは拒否（405 Method Not Allowed）
- `GET /api/graph` 以外のパスは拒否
- グラフデータはリクエストごとに SSOT から再計算し、派生データをキャッシュ・永続化しない
