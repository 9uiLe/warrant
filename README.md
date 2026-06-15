# warrant

Go 製トレーサビリティ統治ゲート。要件・仕様・テストの三者間リンクをチェックし、証明されていない機能や根拠のない要件を CI で検出する。

## 概要

`warrant` は `.warrant/requirements.yaml` に宣言された要件が、以下の不変条件を満たしているかを検証する。

- 各要件は一次情報ソース（spec.doc）へのリンクを持つ
- spec.doc は SSOT（Single Source of Truth）であり、派生データ（generated / dist 配下など）を指してはならない
- active な要件は必ず 1 件以上のテストで証明されていなければならない
- テストファイルは `@covers <ID>` タグで自身がカバーする要件 ID を自己申告しなければならない

違反があれば exit 1 で終了し、CI を停止する。

## インストール

前提: Go 1.23 以降。依存は標準ライブラリ＋ `gopkg.in/yaml.v3`（唯一の例外依存）のみ。

> **注記:** 現在このリポジトリは private です。公開までは取得に GitHub 認証（SSH 鍵またはトークン）が必要です。public 化後は以下の手順がそのまま使えます。

### ローカルへ導入

リポジトリを取得し、単一バイナリをビルドする。`CGO_ENABLED=0` でスタティックリンクにしておくと、同一 OS/アーキテクチャの別マシンへそのまま配布できる。

```sh
git clone git@github.com:9uiLe/warrant.git    # public 化後は https://github.com/9uiLe/warrant.git も可
cd warrant
CGO_ENABLED=0 go build -o warrant .

# PATH の通った場所へ配置（例）
mv warrant /usr/local/bin/warrant
warrant init      # 利用したいプロジェクトの root で実行
```

`go install` でも取得できる（`$GOBIN`／`$GOPATH/bin` に配置される）。**public 化後に利用可能。**

```sh
go install github.com/9uiLe/warrant@latest
```

バージョンタグ（`vX.Y.Z`）を発行していない間は `@latest` が `master` の最新コミット（擬似バージョン）を指す。安定版を配布する場合はタグを発行すること。

private の間に `go install` する場合は、モジュールプロキシを迂回して SSH で直接取得する設定が必要:

```sh
export GOPRIVATE=github.com/9uiLe/*
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

### CI でのビルド

スタティックバイナリを CI でビルドし、成果物（artifact）としてプロジェクトに同梱する運用を想定している。

`go.sum` はリポジトリに含めているため、CI で `go mod tidy` を実行する必要はない。

```yaml
- name: Build
  run: CGO_ENABLED=0 go build -o warrant .

- name: Gate
  run: ./warrant check      # 違反があれば exit 1 で CI を停止
```

## クイックスタート

```sh
# 1. 対象プロジェクトの root で雛形を生成
warrant init
#   → .warrant/{config.yaml, requirements.yaml, requirements.schema.json, README.md} を作成
#     （既存ファイルは上書きせずスキップ。再実行してもべき等）

# 2. .warrant/requirements.yaml を編集し、要件→仕様→テストを宣言する
#    対応するテストファイルに `@covers <ID>` タグを書く

# 3. ゲートチェック（CI でも実行）
warrant check                # PASS なら exit 0 / 違反があれば exit 1

# 4. （任意）トレーサビリティ表を生成・可視化
warrant report               # .warrant/traceability.generated.md を生成
warrant serve                # http://127.0.0.1:7777 で可視化（読み取り専用）
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

### requirements.yaml の例

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

トレーサビリティチェックを実行する。違反があれば exit 1、実行エラーは exit 2。フラグはサブコマンドの**後ろ**に置く。

```sh
warrant check
warrant check --repo-root path/to/project   # 既定: カレントディレクトリ
warrant check --config path/to/config.yaml --registry path/to/requirements.yaml
warrant check --json                         # 機械可読な JSON で出力（日本語は非エスケープ）
```

違反がない場合は exit 0 で終了する。共通フラグ:

| フラグ | 既定値 | 意味 |
|---|---|---|
| `--repo-root` | `.` | 検証対象プロジェクトの root |
| `--config` | `<repo-root>/.warrant/config.yaml` | 設定ファイル（無い場合は既定値で動作） |
| `--registry` | `<repo-root>/.warrant/requirements.yaml` | 要件レジストリ（SSOT） |

### `warrant report`

トレーサビリティマトリクスを生成する。出力先は `config.yaml` の `report_path`（既定: `.warrant/traceability.generated.md`）。

```sh
warrant report
```

生成されたファイルは派生物であるため、`derived_globs` に登録し `spec.doc` として参照できないようにしておくこと。

### `warrant serve`

トレーサビリティグラフを HTTP で可視化する。ローカル開発用。`127.0.0.1` 固定でバインドし、**読み取り専用**（`GET /` で埋め込み HTML、`GET /api/graph` で SSOT から再計算した projection JSON を返す。それ以外のメソッド・パスは拒否）。

```sh
warrant serve                # http://127.0.0.1:7777
warrant serve --port 7799    # ポート変更（既定: 7777）
```

配信するグラフはリクエストごとに SSOT から再計算するため、派生データをキャッシュ・永続化しない。

### `warrant init`

`.warrant/` ディレクトリと設定ファイルの雛形を生成する。**既存ファイルは上書きせずスキップ**するため、再実行してもべき等。

```sh
warrant init
warrant init --repo-root path/to/project   # 既定: カレントディレクトリ
```

生成されるファイル:

| ファイル | 役割 |
|---|---|
| `config.yaml` | ツール設定 |
| `requirements.yaml` | 要件レジストリ（SSOT、サンプル要件入り） |
| `requirements.schema.json` | requirements.yaml のスキーマ（参考） |
| `README.md` | `.warrant/` の説明 |

### `warrant advise`

`kind: semantic` の `enforced_by` エントリを持つルールに対して、外部 Judge コマンドを呼び出し意味的評価を **advisory** として提供する。**CI ゲートではなく、常に exit 0 で終了する。**

```sh
warrant advise                         # semantic_command が未設定なら即終了
warrant advise --json                  # 機械可読 JSON 出力
warrant advise --repo-root path/to/project
warrant advise --rules path/to/rules.yaml
```

Judge コマンドは `config.yaml` の `semantic_command` で指定する。未設定の場合はスキップして正常終了する。

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

## Authority 軸（立法）の違反コード

| コード | 意味 |
|---|---|
| `E-RULE-SCHEMA` | ルールが mapping でない、または `id` / `title` が欠落している |
| `E-RULE-ID-FORMAT` | `id` が `id_pattern` に一致しない |
| `E-RULE-ID-DUP` | `id` がレジストリ内で重複している |
| `E-RULE-NOBASIS` | `basis` が宣言されていない（立法根拠なし） |
| `E-RULE-BASIS-NOFILE` | `basis` のファイルが存在しない |
| `E-RULE-BASIS-NOANCHOR` | `basis` のアンカーが憲法ファイル内に見つからない |
| `E-RULE-UNENFORCED` | `status: active` のルールに `kind: deterministic` かつ実在・タグ付きの `enforced_by` が 1 件もない |
| `E-RULE-ENFORCE-SCHEMA` | `enforced_by` 要素の形式が不正 |
| `E-ENFORCE-NOFILE` | `enforced_by` の `ref` が存在しない |
| `E-ENFORCE-TAG-MISSING` | チェックファイルに `@warrant-enforces <ID>` タグがない |
| `E-RULE-UNRATIFIED` | ルール本文の正規ハッシュが承認ハッシュと不一致 |
| `E-CHECK-ORPHAN` | `@warrant-enforces <ID>` タグの ID がルール登録されていない |
| `E-CHECK-OUTOFSCOPE` | チェックの `governs` がルールの `scope` 外のファイルを裁いている（越境司法） |

## セマンティックチェック (advisory)

`warrant advise` は LLM などの外部 Judge コマンドを使って `kind: semantic` ルールに対するセマンティック評価を行う。決定論的ゲート（`warrant check`）とは完全に分離されており、advise の結果が CI を落とすことはない。

### config.yaml での設定

```yaml
# Judge コマンド（未設定なら warrant advise は即終了）
semantic_command: "claude -p 'あなたはコードレビュアーです...' --output-format json"
# タイムアウト秒数（既定: 30）
semantic_timeout_sec: 30
```

### rules.yaml での semantic エントリ例

```yaml
rules:
  - id: RULE-EXAMPLE
    title: "サンプルルール"
    status: active
    basis: ".warrant/constitution.md#example"
    enforced_by:
      - kind: deterministic
        ref: "internal/example_test.go"
      - kind: semantic
        ref: "LLM"
        governs:
          - "internal/**/*.go"
        criterion: "関数は単一責任原則に従っているか"
```

### Judge コマンドの入出力契約

Judge コマンドは stdin で Request JSON を受け取り、stdout に Verdict JSON を出力する。

Request:
```json
{"rule_id": "RULE-EXAMPLE", "title": "...", "basis": "...", "criterion": "...", "targets": ["path/to/file.go"]}
```

Verdict:
```json
{"verdict": "pass", "rationale": "理由", "proposed_assertion": "（任意）"}
```

`verdict` は `"pass"` / `"fail"` / `"uncertain"` のいずれか。advise は常に exit 0 で終了し、verdict の値にかかわらず CI を停止しない。
