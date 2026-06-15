// @warrant-covers WARRANT-CHECK
package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/projection"
	"github.com/9uiLe/warrant/internal/registry"
)

// violationKey は (Code, Requirement) のペアを表す
type violationKey struct {
	Code        string
	Requirement string
}

// collectViolationKeys は result.Violations から (Code, Requirement) ペアの集合を返す
func collectViolationKeys(result *Result) map[violationKey]struct{} {
	keys := make(map[violationKey]struct{})
	for _, v := range result.Violations {
		keys[violationKey{v.Code, v.Requirement}] = struct{}{}
	}
	return keys
}

// hasViolation は指定のペアが結果に含まれるか確認する
func hasViolation(keys map[violationKey]struct{}, code, requirement string) bool {
	_, ok := keys[violationKey{code, requirement}]
	return ok
}

// writeFile はテスト用ヘルパー: ディレクトリを作成してファイルを書く
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// defaultCfg は DerivedGlobs が空の既定設定を返す
func defaultCfg() *config.Config {
	return config.Default()
}

// makeReg は requirements スライスを持つ Registry を返す
func makeReg(reqs []any) *registry.Registry {
	return &registry.Registry{
		Raw: map[string]any{
			"requirements": reqs,
		},
	}
}

// TestRun_ESchema_NotMapping: mapping でない要件 → E-SCHEMA / "?"
func TestRun_ESchema_NotMapping(t *testing.T) {
	dir := t.TempDir()

	reg := makeReg([]any{
		"not-a-map",
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-SCHEMA", "?") {
		t.Errorf("expected E-SCHEMA/? violation, got: %v", result.Violations)
	}
}

// TestRun_ESchema_NoID: id がない要件 → E-SCHEMA / "?"
func TestRun_ESchema_NoID(t *testing.T) {
	dir := t.TempDir()

	reg := makeReg([]any{
		map[string]any{
			"title": "No ID requirement",
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-SCHEMA", "?") {
		t.Errorf("expected E-SCHEMA/? violation, got: %v", result.Violations)
	}
}

// TestRun_ESchema_NoTitle: title がない要件 → E-SCHEMA / rid
func TestRun_ESchema_NoTitle(t *testing.T) {
	dir := t.TempDir()

	reg := makeReg([]any{
		map[string]any{
			"id":     "FEAT-001",
			"status": "draft",
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
		},
	})
	cfg := defaultCfg()

	// spec.doc ファイルを作成しておく
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "content")

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-SCHEMA", "FEAT-001") {
		t.Errorf("expected E-SCHEMA/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_EIDFormat: id がパターン違反 → E-ID-FORMAT / rid
func TestRun_EIDFormat(t *testing.T) {
	dir := t.TempDir()

	reg := makeReg([]any{
		map[string]any{
			"id":     "invalid_id",
			"title":  "Bad ID",
			"status": "draft",
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
		},
	})
	cfg := defaultCfg()
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "content")

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-ID-FORMAT", "invalid_id") {
		t.Errorf("expected E-ID-FORMAT/invalid_id violation, got: %v", result.Violations)
	}
}

// TestRun_EIDDup: id 重複 → E-ID-DUP / rid
func TestRun_EIDDup(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "content")

	reg := makeReg([]any{
		map[string]any{
			"id":     "FEAT-001",
			"title":  "First",
			"status": "draft",
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
		},
		map[string]any{
			"id":     "FEAT-001",
			"title":  "Duplicate",
			"status": "draft",
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-ID-DUP", "FEAT-001") {
		t.Errorf("expected E-ID-DUP/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ESpecMissing: spec.doc がない → E-SPEC-MISSING / rid
func TestRun_ESpecMissing(t *testing.T) {
	dir := t.TempDir()

	reg := makeReg([]any{
		map[string]any{
			"id":     "FEAT-001",
			"title":  "No Spec",
			"status": "draft",
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-SPEC-MISSING", "FEAT-001") {
		t.Errorf("expected E-SPEC-MISSING/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ESpecDerived: spec.doc が derived_globs に一致 → E-SPEC-DERIVED / rid
func TestRun_ESpecDerived(t *testing.T) {
	dir := t.TempDir()
	// derived_globs にマッチするパス
	derivedDoc := "dist/generated.md"
	writeFile(t, filepath.Join(dir, "dist", "generated.md"), "content")

	reg := makeReg([]any{
		map[string]any{
			"id":     "FEAT-001",
			"title":  "Derived Spec",
			"status": "draft",
			"spec": map[string]any{
				"doc": derivedDoc,
			},
		},
	})
	cfg := defaultCfg()
	cfg.DerivedGlobs = []string{"dist/**"}

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-SPEC-DERIVED", "FEAT-001") {
		t.Errorf("expected E-SPEC-DERIVED/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ESpecNoFile: spec.doc のファイルが存在しない → E-SPEC-NOFILE / rid
func TestRun_ESpecNoFile(t *testing.T) {
	dir := t.TempDir()

	reg := makeReg([]any{
		map[string]any{
			"id":     "FEAT-001",
			"title":  "Missing Spec File",
			"status": "draft",
			"spec": map[string]any{
				"doc": "docs/spec/nonexistent.md",
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-SPEC-NOFILE", "FEAT-001") {
		t.Errorf("expected E-SPEC-NOFILE/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ESpecNoSection: spec.section が doc に存在しない → E-SPEC-NOSECTION / rid
func TestRun_ESpecNoSection(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "# Overview\n\nsome content here")

	reg := makeReg([]any{
		map[string]any{
			"id":     "FEAT-001",
			"title":  "Missing Section",
			"status": "draft",
			"spec": map[string]any{
				"doc":     "docs/spec/req.md",
				"section": "## Nonexistent Section",
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-SPEC-NOSECTION", "FEAT-001") {
		t.Errorf("expected E-SPEC-NOSECTION/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ENoTest: active な要件にテストがない → E-NOTEST / rid
func TestRun_ENoTest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "content")

	reg := makeReg([]any{
		map[string]any{
			"id":    "FEAT-001",
			"title": "No Tests",
			// status 省略 → "active"
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-NOTEST", "FEAT-001") {
		t.Errorf("expected E-NOTEST/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ETestNoFile: tests に存在しないファイルを指定 → E-TEST-NOFILE / rid
func TestRun_ETestNoFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "content")

	reg := makeReg([]any{
		map[string]any{
			"id":    "FEAT-001",
			"title": "Test No File",
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
			"tests": []any{
				"tests/nonexistent_test.go",
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-TEST-NOFILE", "FEAT-001") {
		t.Errorf("expected E-TEST-NOFILE/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ETagMissing: テストファイルは存在するがタグがない → E-TAG-MISSING / rid
func TestRun_ETagMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "content")
	// タグなしのテストファイル
	writeFile(t, filepath.Join(dir, "tests", "feat_test.go"), "package tests\n\n// no covers tag here\n")

	reg := makeReg([]any{
		map[string]any{
			"id":    "FEAT-001",
			"title": "Tag Missing",
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
			"tests": []any{
				"tests/feat_test.go",
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-TAG-MISSING", "FEAT-001") {
		t.Errorf("expected E-TAG-MISSING/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ETestSchema: tests 要素が不正（nil や不正な型）→ E-TEST-SCHEMA / rid
func TestRun_ETestSchema(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "content")

	reg := makeReg([]any{
		map[string]any{
			"id":    "FEAT-001",
			"title": "Test Schema",
			"spec": map[string]any{
				"doc": "docs/spec/req.md",
			},
			"tests": []any{
				42, // 不正な型（string でも map でもない）
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-TEST-SCHEMA", "FEAT-001") {
		t.Errorf("expected E-TEST-SCHEMA/FEAT-001 violation, got: %v", result.Violations)
	}
}

// TestRun_ETagOrphan: 未知の要件 ID を指すタグ → E-TAG-ORPHAN / tagID
func TestRun_ETagOrphan(t *testing.T) {
	dir := t.TempDir()
	// テストファイルに未知の要件 ID のタグを書く
	writeFile(t, filepath.Join(dir, "tests", "orphan_test.go"), "package tests\n\n// @covers UNKNOWN-999\n")

	// 要件リストは空（UNKNOWN-999 が宣言されていない）
	reg := makeReg([]any{})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-TAG-ORPHAN", "UNKNOWN-999") {
		t.Errorf("expected E-TAG-ORPHAN/UNKNOWN-999 violation, got: %v", result.Violations)
	}
}

// TestRun_Pass: 全条件を満たす → violations なし
func TestRun_Pass(t *testing.T) {
	dir := t.TempDir()

	// spec ファイルを作成
	writeFile(t, filepath.Join(dir, "docs", "spec", "req.md"), "# Requirements\n\n## Feature Section\n\nsome content")

	// テストファイルを作成（@covers タグあり）
	writeFile(t, filepath.Join(dir, "tests", "feat_test.go"), "package tests\n\n// @covers FEAT-001\nfunc TestFeat() {}\n")

	reg := makeReg([]any{
		map[string]any{
			"id":    "FEAT-001",
			"title": "Feature 001",
			// status 省略 → "active"
			"spec": map[string]any{
				"doc":     "docs/spec/req.md",
				"section": "## Feature Section",
			},
			"tests": []any{
				"tests/feat_test.go",
			},
		},
	})
	cfg := defaultCfg()

	result, err := Run(dir, reg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Violations) != 0 {
		t.Errorf("expected no violations, got: %v", result.Violations)
	}
	if len(result.Requirements) != 1 {
		t.Errorf("expected 1 requirement, got: %d", len(result.Requirements))
	}
	req := result.Requirements[0]
	if req.ID != "FEAT-001" {
		t.Errorf("expected ID FEAT-001, got: %s", req.ID)
	}
	if req.SpecDoc != "docs/spec/req.md" {
		t.Errorf("expected SpecDoc docs/spec/req.md, got: %s", req.SpecDoc)
	}
	if req.SpecSec != "## Feature Section" {
		t.Errorf("expected SpecSec '## Feature Section', got: %s", req.SpecSec)
	}
	if len(req.Tests) != 1 || req.Tests[0] != "tests/feat_test.go" {
		t.Errorf("expected Tests [tests/feat_test.go], got: %v", req.Tests)
	}
}

// TestBuildGraph_Pass: violations なし → verdict PASS
func TestBuildGraph_Pass(t *testing.T) {
	reqs := []Requirement{
		{
			ID:      "FEAT-001",
			Title:   "Feature 001",
			Status:  "active",
			SpecDoc: "docs/spec/req.md",
			SpecSec: "## Feature Section",
			Tests:   []string{"tests/feat_test.go"},
		},
	}

	g := BuildGraph(reqs, nil, "2026-01-01T00:00:00Z")

	if g.Verdict != "PASS" {
		t.Errorf("expected PASS, got: %s", g.Verdict)
	}
	if g.RequirementCount != 1 {
		t.Errorf("expected RequirementCount 1, got: %d", g.RequirementCount)
	}
	if len(g.Violations) != 0 {
		t.Errorf("expected no violations, got: %v", g.Violations)
	}
	if g.GeneratedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("expected GeneratedAt 2026-01-01T00:00:00Z, got: %s", g.GeneratedAt)
	}
	// ノード: FEAT-001 (requirement) + spec:docs/spec/req.md (spec) + test:tests/feat_test.go (test)
	if len(g.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got: %d", len(g.Nodes))
	}
	// エッジ: req→spec + req→test
	if len(g.Edges) != 2 {
		t.Errorf("expected 2 edges, got: %d", len(g.Edges))
	}
}

// TestBuildGraph_Fail: violations あり → verdict FAIL
func TestBuildGraph_Fail(t *testing.T) {
	vs := []projection.Violation{
		{Code: "E-NOTEST", Requirement: "FEAT-001", Message: "no test"},
	}
	g := BuildGraph(nil, vs, "2026-01-01T00:00:00Z")

	if g.Verdict != "FAIL" {
		t.Errorf("expected FAIL, got: %s", g.Verdict)
	}
	if len(g.Violations) != 1 {
		t.Errorf("expected 1 violation, got: %d", len(g.Violations))
	}
}
