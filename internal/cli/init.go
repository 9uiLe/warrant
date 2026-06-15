package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	var root string
	fs.StringVar(&root, "repo-root", ".", "repo root directory")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	warrantDir := filepath.Join(absRoot, ".warrant")
	if err := os.MkdirAll(warrantDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	// テンプレートファイルを展開（既存は上書きしない）
	files := map[string]string{
		"config.yaml":              configYAMLTemplate,
		"requirements.yaml":        requirementsYAMLTemplate,
		"requirements.schema.json": requirementsSchemaTemplate,
		"README.md":                warrantReadmeTemplate,
	}

	for name, content := range files {
		path := filepath.Join(warrantDir, name)
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  スキップ（既存）: .warrant/%s\n", name)
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "実行エラー: %s の書き込みに失敗: %v\n", name, err)
			return 2
		}
		fmt.Printf("  作成: .warrant/%s\n", name)
	}

	return 0
}

// テンプレート文字列定数
const configYAMLTemplate = `# warrant 設定ファイル
# 詳細は .warrant/README.md を参照

spec_root: "docs/spec"
test_globs:
  - "**/*_test.*"
tag: "@covers"
id_pattern: "[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+"
derived_globs:
  - "*.generated.*"
  - ".warrant/*.generated.*"
report_path: ".warrant/traceability.generated.md"
`

const requirementsYAMLTemplate = `# 要件登録簿 (SSOT)
# このファイルが判断の唯一の正本です。
# 派生データ（*.generated.*）を spec.doc に指定すると E-SPEC-DERIVED で失敗します。

requirements:
  - id: FEAT-001
    title: "サンプル機能"
    status: active
    spec:
      doc: "docs/spec/feat-001.md"
      section: "## 仕様"
    tests:
      - "tests/feat001_test.go"
`

const requirementsSchemaTemplate = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "warrant requirements",
  "type": "object",
  "required": ["requirements"],
  "properties": {
    "requirements": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "title", "spec"],
        "properties": {
          "id": {
            "type": "string",
            "pattern": "^[A-Z][A-Z0-9]*(-[A-Z0-9]+)+$",
            "description": "要件 ID（例: FEAT-001）"
          },
          "title": { "type": "string" },
          "status": {
            "type": "string",
            "enum": ["active", "deprecated", "draft"],
            "default": "active"
          },
          "spec": {
            "type": "object",
            "required": ["doc"],
            "properties": {
              "doc": { "type": "string", "description": "仕様ドキュメントのパス（派生データ不可）" },
              "section": { "type": "string", "description": "参照セクション文字列（部分一致）" }
            }
          },
          "tests": {
            "type": "array",
            "items": {
              "oneOf": [
                { "type": "string" },
                {
                  "type": "object",
                  "required": ["file"],
                  "properties": {
                    "file": { "type": "string" }
                  }
                }
              ]
            }
          }
        }
      }
    }
  }
}
`

const warrantReadmeTemplate = `# .warrant/

warrant のデータディレクトリ。

## ファイル構成

| ファイル | 役割 |
|---------|------|
| ` + "`config.yaml`" + ` | warrant の設定（spec_root, test_globs 等） |
| ` + "`requirements.yaml`" + ` | **SSOT（唯一の正本）** 機能→仕様→テストの宣言 |
| ` + "`requirements.schema.json`" + ` | requirements.yaml のスキーマ（参考） |
| ` + "`*.generated.md`" + ` | 派生レポート（` + "`warrant report`" + ` が生成、判断根拠にしてはならない） |

## 使い方

` + "```" + `sh
# ゲートチェック（CI で実行）
warrant check

# 派生レポート生成
warrant report

# 可視化サーバ（ローカル開発用）
warrant serve

# 初期化（初回のみ）
warrant init
` + "```" + `

## .gitignore 推奨設定

` + "`*.generated.md`" + ` を .gitignore に追加することを推奨します:

` + "```" + `
# .gitignore
.warrant/*.generated.md
` + "```" + `

## 不変条件

- ` + "`requirements.yaml`" + ` と各仕様・テスト本体が SSOT（Single Source of Truth）
- ` + "`*.generated.md`" + ` 等の派生データを ` + "`spec.doc`" + ` に指定すると ` + "`E-SPEC-DERIVED`" + ` で失敗する
- warrant は毎回 SSOT からゼロ計算で判定する（キャッシュなし）
`
