# 仕様: 可視化サーバ（serve コマンド）

`warrant serve` は、トレーサビリティグラフをブラウザで閲覧するためのローカル HTTP サーバを起動する。`internal/cli/serve.go`、`internal/serve`、`internal/web` が実装の中核である。

## 入力

| フラグ | 既定値 | 意味 |
|---|---|---|
| `--repo-root` | `.` | リポジトリ root |
| `--config` | `<root>/.warrant/config.yaml` | 設定ファイルパス |
| `--registry` | `<root>/.warrant/requirements.yaml` | レジストリファイルパス |
| `--port` | `7777` | リッスンするポート番号 |

## エンドポイント

サーバは以下の 2 つの GET エンドポイントのみを公開する。

| メソッド・パス | 応答 |
|---|---|
| `GET /` | 埋め込み（`go:embed`）された `index.html`（可視化 UI） |
| `GET /api/graph` | SSOT から再計算したトレーサビリティグラフの JSON |

- `/api/graph` はリクエストの**都度** `check.Run` を実行し、`check.BuildGraph` でグラフ（`verdict` / `requirement_count` / `violations` / `nodes` / `edges` / `generated_at`）を構築して返す。キャッシュは持たない。

## 不変条件（安全性）

- **127.0.0.1 に固定**してリッスンする（`127.0.0.1:<port>`）。外部インターフェースには公開しない。
- **読み取り専用**。GET 以外のメソッドには `405 Method Not Allowed` を返す。ファイルへの書き込み経路や状態変更エンドポイントを持たない。
- 提供データは毎回 SSOT から再計算するため、サーバ内に可変状態を保持しない。

## exit コード

| コード | 意味 |
|---|---|
| `0` | 正常終了 |
| `2` | 実行エラー（フラグ解析失敗、設定・レジストリ読み込み失敗、サーバ起動失敗） |
