# 仕様: 雛形生成（init コマンド）

`warrant init` は、対象プロジェクトに warrant を導入するための雛形を `.warrant/` 配下に生成する。`internal/cli/init.go` が実装の中核である。

## 入力

| フラグ | 既定値 | 意味 |
|---|---|---|
| `--repo-root` | `.` | 雛形を生成するリポジトリの root ディレクトリ |

`--repo-root` は絶対パスに解決してから使用する。

## 生成物

`<root>/.warrant/` を作成し（既存なら再利用）、以下の 4 ファイルを展開する。

| ファイル | 役割 |
|---|---|
| `config.yaml` | 設定（`spec_root` / `test_globs` / `tag` / `id_pattern` / `derived_globs` / `report_path`） |
| `requirements.yaml` | 要件登録簿（SSOT）。サンプル要件 1 件を含む |
| `requirements.schema.json` | `requirements.yaml` の JSON Schema（参考） |
| `README.md` | `.warrant/` の使い方とファイル構成の説明 |

## 不変条件

- **既存ファイルは上書きしない。** 生成先に同名ファイルが既に存在する場合はスキップし、その旨を表示する（べき等。再実行しても既存の設定・要件を破壊しない）。
- 新規作成したファイルは作成した旨を表示する。
- `.warrant/` ディレクトリの作成は `MkdirAll` 相当で、既存でもエラーにしない。

## exit コード

| コード | 意味 |
|---|---|
| `0` | 正常終了（生成またはスキップ） |
| `2` | 実行エラー（フラグ解析失敗、ディレクトリ作成失敗、ファイル書き込み失敗） |
