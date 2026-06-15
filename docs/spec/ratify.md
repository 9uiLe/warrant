# 仕様: 承認補助（ratify コマンド）

`warrant ratify` は、人間がレビュー・編集したルール本文に対する**承認**を `.warrant/rules.yaml` の `ratification.content_hash` として確定する補助コマンドである。`internal/cli/ratify.go` が中核。

承認ハッシュは `internal/authority` の `CanonicalHash` を唯一の出所として計算する。これは `warrant check`(司法ゲート)が `E-RULE-UNRATIFIED` の判定に使う計算と同一であり、ratify で書いた値はそのまま check を通過する（計算のドリフトが起きない）。

## 役割と非役割

- **役割**: 人間がルール本文(id/title/status/basis/scope/enforced_by)を編集した後、その本文への同意を `content_hash` として記録する。`warrant advise` が出す `proposed_assertion` を人間が参考にして本文を編集する運用を想定する。
- **非役割**: ルール本文そのものを自動生成・自動昇格しない。本文を著すのは人間であり、ratify は同意の記録に徹する（承認の意味を保つため）。

## 使い方

```sh
# dry-run（既定）: 何も書き込まず、各ルールの承認状態を表示する
warrant ratify

# 単一ルールを承認してファイルに書き込む
warrant ratify --rule RULE-SERVE-READONLY --write

# 全ルールを承認して書き込む
warrant ratify --all --write

# 承認者名も記録する
warrant ratify --rule RULE-X --write --approved-by "name@example.com"
```

主なフラグ:

| フラグ | 意味 |
|---|---|
| `--repo-root` | リポジトリルート（既定 `.`） |
| `--rules` | rules.yaml のパス（既定 `.warrant/rules.yaml`） |
| `--config` | config.yaml のパス（既定 `.warrant/config.yaml`） |
| `--rule <ID>` | このルール ID のみ対象にする |
| `--all` | 全ルールを対象にする |
| `--write` | rules.yaml に書き込む（無指定なら dry-run） |
| `--approved-by <name>` | `ratification.approved_by` に承認者を記録する |

## 不変条件

- 既定は **dry-run**（書き込みなし）。`--write` を付けたときのみ rules.yaml を変更する。
- `--write` は `--rule <ID>` か `--all` のいずれかを必須とする。対象未指定の `--write` は終了コード 2 で拒否する（意図しない全件承認を構造的に防ぐ）。
- `--rule <ID>` で指定した ID が rules.yaml に存在しないときは終了コード 2。
- 書き込みは `ratification.content_hash`（必要なら `approved_by`）のみを外科的に更新し、他のフィールド・コメント・構造を保全する。
- 承認ハッシュは `CanonicalHash` で計算し、`warrant check` の `E-RULE-UNRATIFIED` 判定と一致する。
- 冪等である。既に承認済み（stored == expected）のルールは MATCH と表示し、書き込みは行わない。
- 終了コード: 0=成功（dry-run または書き込み）、2=使用方法・IO エラー。

## 承認の二重化との関係

ratify が更新する `content_hash` は「ローカルで人間が承認した」という記録にすぎない。第二の人間によるレビューを担保するため、`.github/CODEOWNERS` と GitHub のブランチ保護で `.warrant/` の変更にオーナーレビューを必須化する（README「承認の二重化(ブランチ保護)」を参照）。ratify（ローカルの同意）と CODEOWNERS（レビュー必須化）の両輪で承認ループが閉じる。
