# 仕様: 派生レポート生成（report コマンド）

`warrant report` は、SSOT（`requirements.yaml` と各仕様・テスト）からトレーサビリティの可視化レポートを Markdown で生成する。`internal/cli/report.go` と `internal/report` が実装の中核である。

## 入力

| フラグ | 既定値 | 意味 |
|---|---|---|
| `--repo-root` | `.` | リポジトリ root |
| `--config` | `<root>/.warrant/config.yaml` | 設定ファイルパス |
| `--registry` | `<root>/.warrant/requirements.yaml` | レジストリファイルパス |

## 出力

- 出力先は `config.yaml` の `report_path`（既定 `.warrant/traceability.generated.md`）。
- 出力ディレクトリが無ければ作成する。
- レポートは毎回 SSOT から `check` と同じ計算（`check.Run`）を行って生成する。キャッシュは持たない。

レポートの構成:

1. **派生データである旨の警告**（先頭 HTML コメントと引用ブロック）。判断の根拠にしてはならない正本は `requirements.yaml` と各仕様・テスト本体である。
2. **機能カバレッジ表** — 各要件の `ID / タイトル / ステータス / 仕様（doc とセクション）/ テスト`。
3. **未解決の違反一覧** — 違反があればコード・要件・メッセージの表、なければ「違反なし（PASS）」。

## 不変条件

- レポートは**派生データ**であり、自身を `spec.doc` に指定すると `E-SPEC-DERIVED` の対象となる（`derived_globs` に `*.generated.*` を含めること）。
- レポート生成は判定ロジックを変えない。違反の有無は `check` と一致する。

## exit コード

| コード | 意味 |
|---|---|
| `0` | 生成成功かつ違反なし |
| `1` | 生成は成功したが違反あり（`check` と同じく違反検出で 1） |
| `2` | 実行エラー（設定・レジストリ読み込み失敗、レポート書き込み失敗など） |
