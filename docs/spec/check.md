# 仕様: トレーサビリティ司法ゲート（check）

`warrant check` は `.warrant/requirements.yaml` に宣言された要件が、要件・仕様・テストの三者間リンクの不変条件を満たすかを検証する。`internal/check` がこの判定ロジックの中核である。

## 不変条件

- 各要件は一次情報ソース（`spec.doc`）へのリンクを持たなければならない。
- `spec.doc` は SSOT（Single Source of Truth）であり、派生データ（generated / dist 配下など）を指してはならない。
- `status: active` な要件は、必ず 1 件以上のテストで証明されていなければならない。
- 宣言されたテストファイルは `tag`（既定 `@covers`、本リポジトリでは `@warrant-covers`）で自身がカバーする要件 ID を自己申告しなければならない。

## 判定の決定性

判定は毎回 SSOT（registry / config / 仕様・テスト本体）からゼロ計算で行い、キャッシュを持たない。孤児タグの走査順などは決定論的になるようソートしてから処理する。

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
| `E-TAG-MISSING` | テストファイルに `tag <ID>` タグがない（要件側の宣言とテスト側の自己申告が不一致） |
| `E-TAG-ORPHAN` | テストファイルに `tag <ID>` タグはあるが、その ID がレジストリに登録されていない |

`E-SPEC-DERIVED` と `E-SPEC-NOFILE` / `E-SPEC-NOSECTION` は独立に評価する。派生データかつ実在しない doc を指した場合は両方が発火する。

## exit コード

| コード | 意味 |
|---|---|
| `0` | 違反なし（司法ゲート PASS） |
| `1` | 1 件以上の違反あり（司法ゲート FAIL） |
| `2` | 実行エラー（設定・レジストリの読み込み失敗など） |
