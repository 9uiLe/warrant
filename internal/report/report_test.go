// @warrant-covers WARRANT-REPORT
package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9uiLe/warrant/internal/check"
	"github.com/9uiLe/warrant/internal/projection"
)

// TestWrite_PassNoViolations: 違反なしのレポートを生成し、派生警告と PASS が含まれる
func TestWrite_PassNoViolations(t *testing.T) {
	dir := t.TempDir()
	reportPath := ".warrant/traceability.generated.md"

	reqs := []check.Requirement{
		{
			ID:      "FEAT-001",
			Title:   "サンプル機能",
			Status:  "active",
			SpecDoc: "docs/spec/feat.md",
			SpecSec: "## 仕様",
			Tests:   []string{"tests/feat_test.go"},
		},
	}

	if err := Write(dir, reportPath, reqs, nil); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, reportPath))
	if err != nil {
		t.Fatalf("生成ファイルが読めない: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"これは派生データです", // 先頭の警告コメント
		"## 機能カバレッジ",
		"FEAT-001",
		"docs/spec/feat.md",
		"tests/feat_test.go",
		"違反なし（PASS）",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("レポートに %q が含まれていない", want)
		}
	}
}

// TestWrite_WithViolations: 違反ありのレポートに違反表が出る
func TestWrite_WithViolations(t *testing.T) {
	dir := t.TempDir()
	reportPath := ".warrant/traceability.generated.md"

	vs := []projection.Violation{
		{Code: "E-NOTEST", Requirement: "FEAT-001", Message: "テストがない"},
	}

	if err := Write(dir, reportPath, nil, vs); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, reportPath))
	if err != nil {
		t.Fatalf("生成ファイルが読めない: %v", err)
	}
	content := string(data)

	for _, want := range []string{"## 未解決の違反", "E-NOTEST", "テストがない"} {
		if !strings.Contains(content, want) {
			t.Errorf("レポートに %q が含まれていない", want)
		}
	}
}

// TestWrite_CreatesDir: 出力先のディレクトリが無ければ作成する
func TestWrite_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	reportPath := "nested/sub/out.generated.md"

	if err := Write(dir, reportPath, nil, nil); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, reportPath)); err != nil {
		t.Errorf("ネストした出力先が作られていない: %v", err)
	}
}
