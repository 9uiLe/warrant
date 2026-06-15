# 仕様: Authority Provenance Graph（立法軸）

`warrant check` は `.warrant/rules.yaml` に宣言されたルールが、ルール・憲法根拠・チェックの三者間リンクの不変条件を満たすかを検証する。`internal/authority` がこの判定ロジックの中核である。

## 不変条件

- 各ルールは憲法ファイルへの根拠アンカー（`basis`）を持たなければならない。
- `status: active` なルールは、必ず 1 件以上の `kind: deterministic` チェックで執行されていなければならない。
- 執行チェックファイルは `@warrant-enforces <ID>` タグで自身が執行するルール ID を自己申告しなければならない。
- ルール本文の正規ハッシュが `ratification.content_hash` と一致することで人間の承認を証明する。
- チェックの実効範囲（`governs`）はルールの管轄（`scope`）の部分集合でなければならない（越境司法の禁止）。

## 違反コード一覧

| コード | 意味 |
|---|---|
| `E-RULE-SCHEMA` | ルールが mapping でない、または `id` / `title` が欠落している |
| `E-RULE-ID-FORMAT` | `id` が `id_pattern` に一致しない |
| `E-RULE-ID-DUP` | `id` がレジストリ内で重複している |
| `E-RULE-NOBASIS` | `basis` が宣言されていない（立法根拠なし） |
| `E-RULE-BASIS-NOFILE` | `basis` のファイルが存在しない |
| `E-RULE-BASIS-NOANCHOR` | `basis` のアンカーが憲法ファイル内に見つからない |
| `E-RULE-UNENFORCED` | `status: active` のルールに `kind: deterministic` かつ実在・タグ付きの `enforced_by` が 1 件もない |
| `E-RULE-ENFORCE-SCHEMA` | `enforced_by` 要素の形式が不正（`kind` / `ref` が取れない） |
| `E-ENFORCE-NOFILE` | `enforced_by` の `ref` が存在しない |
| `E-ENFORCE-TAG-MISSING` | チェックファイルに `@warrant-enforces <ID>` タグがない |
| `E-RULE-UNRATIFIED` | ルール本文の正規ハッシュが `ratification.content_hash` と一致しない（未承認） |
| `E-CHECK-ORPHAN` | チェックファイルに `@warrant-enforces <ID>` タグはあるが、その ID がルール登録されていない |
| `E-CHECK-OUTOFSCOPE` | チェックの `governs` が `scope` の範囲外のファイルを裁いている（越境司法） |

## scope と governs

ルールに `scope`（グロブのリスト）、`enforced_by` 各エントリに `governs`（グロブのリスト）を宣言することで越境司法を検知する。

- `scope`: ルールの管轄。ルールが責任を持つファイル群を表すグロブ。
- `governs`: そのチェックが実際に裁く範囲。チェックファイルが検証対象とするファイル群を表すグロブ。

不変条件: `governs ⊆ scope`。すなわち `governs` でマッチする全ファイルが `scope` にもマッチしなければならない。

判定アルゴリズム:
1. `scope` と `governs` が両方とも非空のときのみ評価する（どちらか空なら後方互換でスキップ）。
2. `governs` の全グロブパターンに `Glob(root, pattern)` でマッチするファイル集合 G を列挙する。
3. G が空なら違反なし（マッチするファイルがなければ越境不能）。
4. G の各ファイル f について `FnMatch(f, scopePattern)` でいずれかの `scope` パターンにマッチするか確認。
5. マッチしないファイルが 1 件以上あれば `E-CHECK-OUTOFSCOPE` 違反。

注意: `scope` の一致確認には `FnMatch`（`*` がスラッシュを超えてマッチ）を使用する。
`governs` の列挙には `Glob`（ファイルシステムを走査）を使用する。末尾セグメントが `**` のグロブ（例 `"pkg/**"`）は配下の全ファイルに再帰的にマッチする（`"pkg/**/*"` と同じ集合）。特定拡張子に絞るなら `"pkg/**/*.go"` のように末尾にファイル名パターンを置く。

## basis アンカー形式

`basis` は `<file>#<anchor>` 形式で指定する。`file` はリポジトリルート相対パス。アンカーの一致条件は憲法ファイル本文に `{#<anchor>}` がリテラルで含まれること。

## ハッシュ正規化

`ratification.content_hash` の対象は `ratification` ブロックを除いたルール正規 body。`sha256:` プレフィックス付き小文字 hex。

正規文字列のフォーマット:

```
id:<id>\n
title:<title>\n
status:<status(空なら"active")>\n
basis:<basis>\n
scope:<sortedScope(カンマ区切り昇順)>\n
enforced_by:<sortedEntries(カンマ区切り昇順)>\n
```

`sortedEntries` の各要素は `kind|ref|governs1;governs2;...`（governs は昇順セミコロン区切り）。`scope` や `governs` が空の場合は空文字列になる。
