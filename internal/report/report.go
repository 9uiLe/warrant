package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/9uiLe/warrant/internal/check"
	"github.com/9uiLe/warrant/internal/projection"
)

// Write は派生レポートを生成する
func Write(root, reportPath string, reqs []check.Requirement, vs []projection.Violation) error {
	absPath := filepath.Join(root, reportPath)
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ディレクトリ作成エラー: %w", err)
	}

	var sb strings.Builder

	sb.WriteString("<!-- これは派生データです。判断の根拠にしてはなりません。正本は .warrant/requirements.yaml と各仕様・テスト本体です。 -->\n\n")
	sb.WriteString("# トレーサビリティ レポート（派生）\n\n")
	sb.WriteString("> **注意**: このファイルは `warrant report` によって自動生成された派生データです。  \n")
	sb.WriteString("> 判断の根拠にしてはなりません。正本は `.warrant/requirements.yaml` と各仕様・テスト本体です。\n\n")

	// 機能→仕様→テスト表
	sb.WriteString("## 機能カバレッジ\n\n")
	sb.WriteString("| ID | タイトル | ステータス | 仕様 | テスト |\n")
	sb.WriteString("|-----|---------|-----------|------|-------|\n")

	for _, req := range reqs {
		specCell := ""
		if req.SpecDoc != "" {
			if req.SpecSec != "" {
				specCell = fmt.Sprintf("`%s` § %s", req.SpecDoc, req.SpecSec)
			} else {
				specCell = fmt.Sprintf("`%s`", req.SpecDoc)
			}
		}

		testCells := make([]string, len(req.Tests))
		for i, t := range req.Tests {
			testCells[i] = fmt.Sprintf("`%s`", t)
		}
		testCell := strings.Join(testCells, "<br>")
		if testCell == "" {
			testCell = "—"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			req.ID, req.Title, req.Status, specCell, testCell))
	}

	// 未解決違反一覧
	sb.WriteString("\n## 未解決の違反\n\n")
	if len(vs) == 0 {
		sb.WriteString("違反なし（PASS）\n")
	} else {
		sb.WriteString("| コード | 要件 | メッセージ |\n")
		sb.WriteString("|--------|------|----------|\n")
		for _, v := range vs {
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", v.Code, v.Requirement, v.Message))
		}
	}

	return os.WriteFile(absPath, []byte(sb.String()), 0644)
}
