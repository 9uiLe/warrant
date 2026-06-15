# warrant

Go 製トレーサビリティ統治ゲート。要件・仕様・テストの三者間リンクをチェックし、証明されていない機能や根拠のない要件を CI で検出する。

## 概要

`warrant` は `.warrant/requirements.yaml` に宣言された要件が、以下の不変条件を満たしているかを検証する。

- 各要件は一次情報ソース（spec.doc）へのリンクを持つ
- spec.doc は SSOT（Single Source of Truth）であり、派生データ（generated / dist 配下など）を指してはならない
- active な要件は必ず 1 件以上のテストで証明されていなければならない
- テストファイルは `@covers <ID>` タグで自身がカバーする要件 ID を自己申告しなければならない

違反があれば exit 1 で終了し、CI を停止する。

## 設計思想: warrant が守るもの・守らないもの

warrant が機械保証するのは**リンクの構造的整合性**（要件↔仕様、要件↔テスト）であり、**仕様文とテストの意味的一致は守らない**。意味の一致はテストランナー（テスト↔実装の挙動）と人間レビューが担う。この三者分業を理解しないと `check` の挙動（例: 仕様の本文だけを書き換えても PASS する）が腑に落ちない。

| 守りたい整合性 | 担保するもの |
|---|---|
| 要件 ↔ 仕様 / 要件 ↔ テスト（リンク） | `warrant check`（機械保証） |
| テスト ↔ 実装（挙動） | テストランナーの実行 |
| 仕様文 ↔ テスト・実装（意味） | 人間レビュー + semantic advise（warrant は可視化のみ） |

詳細な思想・三者分業・改修時の挙動（捕まる/素通りの境界）は [整合性モデルと運用思想](docs/consistency-model.md)（Mermaid 図つき）を参照。

## インストール

> **注記:** install.sh / GitHub Action は GitHub Releases に公開されたバイナリを取得します。リリースが 1 件も存在しない場合は最新版の解決に失敗するため、`WARRANT_VERSION` で明示するか、リリース公開後に利用してください。

### 推奨: install.sh によるワンライナー

Go 不要。`$HOME/.local/bin` にバイナリを配置し、SHA256 を自動検証します。

```sh
curl -fsSL https://raw.githubusercontent.com/9uiLe/warrant/master/install.sh | sh
```

インストール後、`$HOME/.local/bin` が PATH に含まれていない場合はシェルの設定ファイルに追記します:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

バージョンを固定したい場合は環境変数で指定できます:

```sh
curl -fsSL https://raw.githubusercontent.com/9uiLe/warrant/master/install.sh | WARRANT_VERSION=v0.1.0 sh
```

### CI: GitHub Action

`setup-go` + `go build` を以下の 1 ステップに置き換えられます:

```yaml
- uses: 9uiLe/warrant@v0.1.0
  with:
    args: check
```

オプションで `working-directory` や `version` を指定できます:

```yaml
- uses: 9uiLe/warrant@v0.1.0
  with:
    version: v0.1.0
    args: check
    working-directory: path/to/project
```

### 再現可能ビルドと SHA256 検証について

配布バイナリは `-trimpath` フラグ付きで再現可能ビルドされ、checksums.txt（SHA256）と共に GitHub Releases に公開されます。install.sh は常に checksums.txt を取得してバイナリの整合性を検証し、不一致なら即終了します（fail closed）。「配布物 = ソースからのビルド」を独立に確認できる経路を保ち、warrant 自身が掲げる決定論・派生物不信の思想と整合しています。

### ソースからビルド（Go 保有者向けの経路）

最も簡単な経路は install.sh ですが、Go 環境がある場合や配布バイナリを自分で検証したい場合は以下の経路を使用します（再現可能ビルドにより install.sh が配る成果物と一致する）。

リポジトリを取得し、単一バイナリをビルドする。`CGO_ENABLED=0` でスタティックリンクにしておくと、同一 OS/アーキテクチャの別マシンへそのまま配布できる。

```sh
git clone https://github.com/9uiLe/warrant.git
cd warrant
CGO_ENABLED=0 go build -o warrant .

# PATH の通った場所へ配置（例）
mv warrant /usr/local/bin/warrant
warrant init      # 利用したいプロジェクトの root で実行
```

`go install` でも取得できる（`$GOBIN`／`$GOPATH/bin` に配置される）。

```sh
go install github.com/9uiLe/warrant@latest
```

### CI でのビルド（Go 保有者向け代替経路）

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

## 設定ファイルの例

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

## 承認の二重化(ブランチ保護)

`ratify` サブコマンド（ローカルでの承認 = content_hash 更新）は「ファイルを手元で確認した」という記録にとどまり、「第二の人間が実際にレビューした」ことを担保しない。

`.github/CODEOWNERS` により `/.warrant/`・`/docs/spec/`・`/.github/` の変更には `@9uiLe`（リポジトリオーナー）のレビューが必須化されている。

ただし CODEOWNERS はブランチ保護と組み合わせて初めて機能する。ブランチ保護はリポジトリ管理者が GitHub の設定画面でサーバ側に設定する手動作業であり、このリポジトリのコードには含められない。以下の項目を設定すること。

- (a) `master` への直接 push を禁止し、PR 経由を必須化する
- (b) マージ前に PR レビュー承認を必須化する
- (c) "Require review from Code Owners" を有効化する
- (d) 必須ステータスチェックに `warrant check` を実行する CI を指定する（任意・推奨）

これにより、`ratify`（ローカルの承認 = content_hash 更新）と CODEOWNERS + ブランチ保護（第二の人間によるレビュー）で承認ループが閉じる。

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
