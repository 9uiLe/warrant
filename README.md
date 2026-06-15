# warrant

Go 製トレーサビリティ統治ゲート。要件・仕様・テストの三者間リンクをチェックし、証明されていない機能や根拠のない要件を CI で検出する。

## 概要

`warrant` は `.warrant/registry.yaml` に宣言された要件が、以下の不変条件を満たしているかを検証する。

- 各要件は一次情報ソース（spec.doc）へのリンクを持つ
- spec.doc は SSOT（Single Source of Truth）であり、派生データ（generated / dist 配下など）を指してはならない
- active な要件は必ず 1 件以上のテストで証明されていなければならない
- テストファイルは `@covers <ID>` タグで自身がカバーする要件 ID を自己申告しなければならない

違反があれば exit 1 で終了し、CI を停止する。

## インストール（CI でのビルド方法）

`go.sum` はリポジトリに含めず、CI の `go mod tidy` ステップで生成する。スタティックバイナリは CGO を無効にしてビルドする。

```yaml
- name: go mod tidy
  run: go mod tidy

- name: Build
  run: CGO_ENABLED=0 go build -o warrant .
```

ローカルでは `go build` または `go install` で取得できる。

```sh
CGO_ENABLED=0 go build -o warrant .
./warrant check
```

## .warrant/ ディレクトリレイアウト

```
.warrant/
├── config.yaml          # ツール設定（省略時は既定値を使用）
├── registry.yaml        # 要件レジストリ（SSOT）
└── traceability.generated.md  # report サブコマンドが生成するトレーサビリティマトリクス（派生物）
```

`traceability.generated.md` は派生物であるため、`config.yaml` の `derived_globs` に登録しておくこと。これにより誤って `spec.doc` に指定した場合に `E-SPEC-DERIVED` で検出される。

### config.yaml の例

```yaml
spec_root: docs/spec
test_globs:
  - "**/*_test.go"
  - "tests/**/*.py"
tag: "@covers"
id_pattern: '[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+'
derived_globs:
  - ".warrant/traceability.generated.md"
  - "dist/**"
report_path: .warrant/traceability.generated.md
```

### registry.yaml の例

```yaml
requirements:
  - id: FEAT-001
    title: ユーザーがログインできる
    status: active          # active（既定）| draft | deprecated
    spec:
      doc: docs/spec/auth.md
      section: "## ログイン仕様"
    tests:
      - tests/auth_test.go
```

## サブコマンド

### `warrant check`

トレーサビリティチェックを実行する。違反があれば exit 1。

```sh
warrant check
warrant --config path/to/config.yaml check
```

違反がない場合は exit 0 で終了する。

### `warrant report`

トレーサビリティマトリクスを生成する。出力先は `config.yaml` の `report_path`（既定: `.warrant/traceability.generated.md`）。

```sh
warrant report
```

生成されたファイルは派生物であるため、`derived_globs` に登録し `spec.doc` として参照できないようにしておくこと。

### `warrant serve`

トレーサビリティグラフを HTTP で配信する（実装予定）。

```sh
warrant serve --port 8080
```

### `warrant init`

`.warrant/` ディレクトリと設定ファイルの雛形を生成する（実装予定）。

```sh
warrant init
```

## SSOT / 派生分離の不変条件

`spec.doc` には必ず一次情報ソース（人間が書いた仕様書）を指定しなければならない。生成物・ビルド成果物・ツールが出力したファイルを指定してはならない。

`config.yaml` の `derived_globs` に派生データのパターンを列挙しておくと、誤って `spec.doc` に指定した場合に `E-SPEC-DERIVED` で CI が落ちる。

```yaml
# 悪い例: warrant report が生成したファイルを spec.doc に指定している
spec:
  doc: .warrant/traceability.generated.md  # → E-SPEC-DERIVED で FAIL

# 良い例: 人間が書いた仕様書を指定している
spec:
  doc: docs/spec/auth.md
```

## パリティ方針

`warrant` は Python 実装と Go 実装の 2 つのリファレンスを持つ。両実装は以下の条件を満たすことを CI で照合する想定。

- 同一の `registry.yaml` と `config.yaml` に対して、**違反コードの集合**が一致する
- **exit コード**（0 / 1 / 2）が一致する

実装言語が異なってもセマンティクスが乖離しないことを保証するため、テストフィクスチャを共有し、両実装を同一 CI ジョブで実行して diff を取る。

## 違反コード一覧

| コード | 意味 |
|---|---|
| `E-SCHEMA` | 要件が mapping でない、または `id` / `title` が欠落している |
| `E-ID-FORMAT` | `id` が `id_pattern` に一致しない |
| `E-ID-DUP` | `id` がレジストリ内で重複している |
| `E-SPEC-MISSING` | `spec.doc` が宣言されていない（立法根拠なし） |
| `E-SPEC-DERIVED` | `spec.doc` が `derived_globs` に一致する派生データを指している |
| `E-SPEC-NOFILE` | `spec.doc` に指定したファイルが存在しない |
| `E-SPEC-NOSECTION` | `spec.section` に指定した文字列が `spec.doc` 内に見つからない |
| `E-NOTEST` | `status: active` の要件にテストが 1 件もない（証明されていない機能） |
| `E-TEST-SCHEMA` | `tests` リストの要素形式が不正（string でも `{file: ...}` でもない） |
| `E-TEST-NOFILE` | `tests` に指定したテストファイルが存在しない |
| `E-TAG-MISSING` | テストファイルに `@covers <ID>` タグがない（要件側の宣言とテスト側の自己申告が不一致） |
| `E-TAG-ORPHAN` | テストファイルに `@covers <ID>` タグはあるが、その ID がレジストリに登録されていない |
