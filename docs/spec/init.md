# 仕様: 雛形生成（init コマンド）

`warrant init` は、対象プロジェクトに warrant を導入するための雛形を `.warrant/` 配下に生成する。`internal/cli/init.go` が実装の中核である。

## 入力

| フラグ | 既定値 | 意味 |
|---|---|---|
| `--repo-root` | `.` | 雛形を生成するリポジトリの root ディレクトリ |
| `--lang` | `go` | テスト言語プリセット（`go` / `swift` / `python` / `js` / `generic`） |

`--repo-root` は絶対パスに解決してから使用する。

`--lang` には以下のプリセットを指定できる。未知の値を指定した場合は stderr に有効値一覧を出力して exit 2 で終了する。

| プリセット | test_globs |
|---|---|
| `go` | `**/*_test.go` |
| `swift` | `**/*Tests.swift` |
| `python` | `**/test_*.py`, `**/*_test.py` |
| `js` | `**/*.test.ts`, `**/*.spec.ts`, `**/*.test.js`, `**/*.spec.js` |
| `generic` | 上記すべての和集合 |

## 生成物

`<root>/.warrant/` を作成し（既存なら再利用）、以下の 4 ファイルを展開する。

| ファイル | 役割 |
|---|---|
| `config.yaml` | 設定（`spec_root` / `test_globs` / `tag` / `id_pattern` / `derived_globs` / `report_path`） |
| `requirements.yaml` | 要件登録簿（SSOT）。サンプル要件 1 件を含む |
| `requirements.schema.json` | `requirements.yaml` の JSON Schema（参考） |
| `README.md` | `.warrant/` の使い方とファイル構成の説明 |

生成される `config.yaml` の `test_globs` セクションは自己記述的になっている。選択プリセットの glob が有効行として出力され、その前に全言語の glob 例がコメントで併記される。これにより、他言語への変更方法が config ファイルを見るだけで分かる。

`config.yaml` が新規作成された場合のみ、stdout に以下のヒントを表示する（既存ファイルをスキップした場合は非表示）。

```
ヒント: 選択言語プリセット = <lang>。テストファイルが test_globs にマッチするか `warrant check` で確認してください。
      他言語へ変える場合は .warrant/config.yaml の test_globs コメント例を参照。
```

## 不変条件

- **既存ファイルは上書きしない。** 生成先に同名ファイルが既に存在する場合はスキップし、その旨を表示する（べき等。再実行しても既存の設定・要件を破壊しない）。
- 新規作成したファイルは作成した旨を表示する。
- `.warrant/` ディレクトリの作成は `MkdirAll` 相当で、既存でもエラーにしない。
- `--lang` の既定値は `go`（後方互換）。`internal/config/config.go` の `applyDefaults` は変更しない。

## exit コード

| コード | 意味 |
|---|---|
| `0` | 正常終了（生成またはスキップ） |
| `2` | 実行エラー（フラグ解析失敗、未知の `--lang` 値、ディレクトリ作成失敗、ファイル書き込み失敗） |
