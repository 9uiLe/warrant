// @warrant-covers WARRANT-AUTHORITY
package authority

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9uiLe/warrant/internal/config"
)

// violationKey は (Code, Requirement) のペアを表す
type violationKey struct {
	Code        string
	Requirement string
}

func collectViolationKeys(result *Result) map[violationKey]struct{} {
	keys := make(map[violationKey]struct{})
	for _, v := range result.Violations {
		keys[violationKey{v.Code, v.Requirement}] = struct{}{}
	}
	return keys
}

func hasViolation(keys map[violationKey]struct{}, code, requirement string) bool {
	_, ok := keys[violationKey{code, requirement}]
	return ok
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func defaultCfg() *config.Config {
	return config.Default()
}

// TestCanonicalHash_Fixed: sha256: プレフィックスと冪等性を検証
func TestCanonicalHash_Fixed(t *testing.T) {
	got := CanonicalHash("RULE-TEST", "Test Rule", "active", ".warrant/constitution.md#test", nil, []EnforceEntry{
		{Kind: "deterministic", Ref: "internal/authority/authority_test.go"},
	})
	if !strings.HasPrefix(got, "sha256:") {
		t.Errorf("expected sha256: prefix, got: %s", got)
	}
	got2 := CanonicalHash("RULE-TEST", "Test Rule", "active", ".warrant/constitution.md#test", nil, []EnforceEntry{
		{Kind: "deterministic", Ref: "internal/authority/authority_test.go"},
	})
	if got != got2 {
		t.Errorf("CanonicalHash is not deterministic: %s != %s", got, got2)
	}
	got3 := CanonicalHash("RULE-TEST", "Test Rule", "active", ".warrant/constitution.md#test", nil, []EnforceEntry{
		{Kind: "deterministic", Ref: "internal/authority/authority_test.go"},
	})
	if got != got3 {
		t.Errorf("CanonicalHash changed unexpectedly: %s != %s", got, got3)
	}
}

// TestCanonicalHash_SortOrder: enforced_by の順序が違っても同じハッシュが返る
func TestCanonicalHash_SortOrder(t *testing.T) {
	h1 := CanonicalHash("RULE-X", "X", "active", "basis", nil, []EnforceEntry{
		{Kind: "deterministic", Ref: "a.go"},
		{Kind: "deterministic", Ref: "b.go"},
	})
	h2 := CanonicalHash("RULE-X", "X", "active", "basis", nil, []EnforceEntry{
		{Kind: "deterministic", Ref: "b.go"},
		{Kind: "deterministic", Ref: "a.go"},
	})
	if h1 != h2 {
		t.Errorf("CanonicalHash order-sensitive: %s != %s", h1, h2)
	}
}

// TestCanonicalHash_WithScopeAndGoverns: scope/governs あり/なし両方で固定値テスト
func TestCanonicalHash_WithScopeAndGoverns(t *testing.T) {
	// scope/governs なし（後方互換）
	h1 := CanonicalHash("RULE-X", "X", "active", "basis", nil, []EnforceEntry{
		{Kind: "deterministic", Ref: "a_test.go"},
	})
	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("expected sha256: prefix, got: %s", h1)
	}

	// scope/governs あり
	h2 := CanonicalHash("RULE-X", "X", "active", "basis",
		[]string{"internal/foo/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: "a_test.go", Governs: []string{"internal/foo/**"}},
		},
	)
	if !strings.HasPrefix(h2, "sha256:") {
		t.Errorf("expected sha256: prefix, got: %s", h2)
	}

	// scope/governs の有無でハッシュが変わること（内容が異なる）
	if h1 == h2 {
		t.Errorf("hash should differ between scope-less and scope-ful rules")
	}

	// scope/governs の順序が違っても同じハッシュ
	h3 := CanonicalHash("RULE-X", "X", "active", "basis",
		[]string{"internal/foo/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: "a_test.go", Governs: []string{"internal/foo/**"}},
		},
	)
	if h2 != h3 {
		t.Errorf("CanonicalHash with scope/governs is not deterministic: %s != %s", h2, h3)
	}

	// governs の複数要素の順序が違っても同じハッシュ
	h4 := CanonicalHash("RULE-X", "X", "active", "basis",
		[]string{"b/**", "a/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: "a_test.go", Governs: []string{"b/**", "a/**"}},
		},
	)
	h5 := CanonicalHash("RULE-X", "X", "active", "basis",
		[]string{"a/**", "b/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: "a_test.go", Governs: []string{"a/**", "b/**"}},
		},
	)
	if h4 != h5 {
		t.Errorf("CanonicalHash scope/governs order-sensitive: %s != %s", h4, h5)
	}
}

// TestRun_ERuleSchema_NotMapping: mapping でないルール → E-RULE-SCHEMA / "?"
func TestRun_ERuleSchema_NotMapping(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules:\n  - not-a-map\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-SCHEMA", "?") {
		t.Errorf("expected E-RULE-SCHEMA/? violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleSchema_NoID: id がないルール → E-RULE-SCHEMA / "?"
func TestRun_ERuleSchema_NoID(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules:\n  - title: No ID\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-SCHEMA", "?") {
		t.Errorf("expected E-RULE-SCHEMA/? violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleSchema_NoTitle: title がないルール → E-RULE-SCHEMA / rid
func TestRun_ERuleSchema_NoTitle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules:\n  - id: RULE-TEST\n    basis: file.md\n")
	writeFile(t, filepath.Join(dir, "file.md"), "content")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-SCHEMA", "RULE-TEST") {
		t.Errorf("expected E-RULE-SCHEMA/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleIDFormat: id がパターン違反 → E-RULE-ID-FORMAT / rid
func TestRun_ERuleIDFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules:\n  - id: invalid_id\n    title: Bad ID\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-ID-FORMAT", "invalid_id") {
		t.Errorf("expected E-RULE-ID-FORMAT/invalid_id violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleIDDup: id 重複 → E-RULE-ID-DUP / rid
func TestRun_ERuleIDDup(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), `rules:
  - id: RULE-TEST
    title: First
  - id: RULE-TEST
    title: Duplicate
`)
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-ID-DUP", "RULE-TEST") {
		t.Errorf("expected E-RULE-ID-DUP/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleNoBasis: basis なし → E-RULE-NOBASIS / rid
func TestRun_ERuleNoBasis(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules:\n  - id: RULE-TEST\n    title: No Basis\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-NOBASIS", "RULE-TEST") {
		t.Errorf("expected E-RULE-NOBASIS/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleBasisNoFile: basis のファイルが存在しない → E-RULE-BASIS-NOFILE / rid
func TestRun_ERuleBasisNoFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules:\n  - id: RULE-TEST\n    title: No Basis File\n    basis: nonexistent.md\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-BASIS-NOFILE", "RULE-TEST") {
		t.Errorf("expected E-RULE-BASIS-NOFILE/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleBasisNoAnchor: basis のアンカーが見つからない → E-RULE-BASIS-NOANCHOR / rid
func TestRun_ERuleBasisNoAnchor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules:\n  - id: RULE-TEST\n    title: No Anchor\n    basis: const.md#missing\n")
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n\nno anchors here\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-BASIS-NOANCHOR", "RULE-TEST") {
		t.Errorf("expected E-RULE-BASIS-NOANCHOR/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleUnenforced: active なルールに deterministic enforced_by がない → E-RULE-UNENFORCED / rid
func TestRun_ERuleUnenforced(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), `rules:
  - id: RULE-TEST
    title: Unenforced
    basis: const.md
`)
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-UNENFORCED", "RULE-TEST") {
		t.Errorf("expected E-RULE-UNENFORCED/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_EEnforceNoFile: enforced_by の ref が存在しない → E-ENFORCE-NOFILE / rid
func TestRun_EEnforceNoFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), `rules:
  - id: RULE-TEST
    title: Enforce No File
    basis: const.md
    enforced_by:
      - kind: deterministic
        ref: nonexistent_test.go
`)
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-ENFORCE-NOFILE", "RULE-TEST") {
		t.Errorf("expected E-ENFORCE-NOFILE/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_EEnforceTagMissing: チェックファイルにタグがない → E-ENFORCE-TAG-MISSING / rid
func TestRun_EEnforceTagMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), `rules:
  - id: RULE-TEST
    title: Tag Missing
    basis: const.md
    enforced_by:
      - kind: deterministic
        ref: check_test.go
`)
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n")
	writeFile(t, filepath.Join(dir, "check_test.go"), "package authority_test\n\n// no enforces tag\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-ENFORCE-TAG-MISSING", "RULE-TEST") {
		t.Errorf("expected E-ENFORCE-TAG-MISSING/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleEnforceSchema: enforced_by 要素が不正 → E-RULE-ENFORCE-SCHEMA / rid
func TestRun_ERuleEnforceSchema(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), `rules:
  - id: RULE-TEST
    title: Enforce Schema
    basis: const.md
    enforced_by:
      - not-a-map
`)
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-ENFORCE-SCHEMA", "RULE-TEST") {
		t.Errorf("expected E-RULE-ENFORCE-SCHEMA/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ERuleUnratified: content_hash が不一致 → E-RULE-UNRATIFIED / rid
func TestRun_ERuleUnratified(t *testing.T) {
	dir := t.TempDir()
	checkFile := "check_test.go"
	writeFile(t, filepath.Join(dir, "rules.yaml"), `rules:
  - id: RULE-TEST
    title: Unratified
    basis: const.md
    enforced_by:
      - kind: deterministic
        ref: check_test.go
    ratification:
      content_hash: sha256:wronghash
`)
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n")
	enforceTag := "@warrant-" + "enforces RULE-TEST"
	writeFile(t, filepath.Join(dir, checkFile), "// "+enforceTag+"\npackage authority_test\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-RULE-UNRATIFIED", "RULE-TEST") {
		t.Errorf("expected E-RULE-UNRATIFIED/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ECheckOrphan: 未知の ID を指す @warrant-enforces タグ → E-CHECK-ORPHAN / tagID
func TestRun_ECheckOrphan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yaml"), "rules: []\n")
	orphanTag := "@warrant-" + "enforces UNKNOWN-999"
	writeFile(t, filepath.Join(dir, "orphan_test.go"), "// "+orphanTag+"\npackage authority_test\n")
	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-CHECK-ORPHAN", "UNKNOWN-999") {
		t.Errorf("expected E-CHECK-ORPHAN/UNKNOWN-999 violation, got: %v", result.Violations)
	}
}

// TestRun_Pass: 全条件を満たす → violations なし
func TestRun_Pass(t *testing.T) {
	dir := t.TempDir()
	checkFile := "check_test.go"

	// 正しいハッシュを計算
	expectedHash := CanonicalHash("RULE-TEST", "Pass Rule", "active", "const.md#section", nil, []EnforceEntry{
		{Kind: "deterministic", Ref: checkFile},
	})

	rulesYaml := `rules:
  - id: RULE-TEST
    title: Pass Rule
    basis: const.md#section
    enforced_by:
      - kind: deterministic
        ref: ` + checkFile + `
    ratification:
      content_hash: ` + expectedHash + `
`
	writeFile(t, filepath.Join(dir, "rules.yaml"), rulesYaml)
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n\n## Section {#section}\n\nContent here.\n")
	passEnforceTag := "@warrant-" + "enforces RULE-TEST"
	writeFile(t, filepath.Join(dir, checkFile), "// "+passEnforceTag+"\npackage authority_test\n")

	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Violations) != 0 {
		t.Errorf("expected no violations, got: %v", result.Violations)
	}
	if len(result.Rules) != 1 {
		t.Errorf("expected 1 rule, got: %d", len(result.Rules))
	}
}

// TestRun_ECheckOutOfScope_Violation: governs が scope 外のファイルにマッチ → E-CHECK-OUTOFSCOPE
func TestRun_ECheckOutOfScope_Violation(t *testing.T) {
	dir := t.TempDir()
	checkFile := "internal/domain/check_test.go"
	// scope 内ファイル
	writeFile(t, filepath.Join(dir, "internal/domain/handler.go"), "package domain\n")
	// scope 外ファイル（governs はマッチするが scope はマッチしない）
	// governs: "internal/**/*.go" → Glob でマッチ。scope: "internal/domain/**" → FnMatch で不一致
	writeFile(t, filepath.Join(dir, "internal/other/service.go"), "package other\n")

	enforceTag := "@warrant-" + "enforces RULE-TEST"
	writeFile(t, filepath.Join(dir, checkFile), "// "+enforceTag+"\npackage domain_test\n")
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n\n## Section {#section}\n\n")

	hash := CanonicalHash("RULE-TEST", "Out Of Scope", "active", "const.md#section",
		[]string{"internal/domain/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: checkFile, Governs: []string{"internal/**/*.go"}},
		},
	)

	rulesYaml := `rules:
  - id: RULE-TEST
    title: "Out Of Scope"
    basis: const.md#section
    scope:
      - "internal/domain/**"
    enforced_by:
      - kind: deterministic
        ref: ` + checkFile + `
        governs:
          - "internal/**/*.go"
    ratification:
      content_hash: ` + hash + `
`
	writeFile(t, filepath.Join(dir, "rules.yaml"), rulesYaml)

	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if !hasViolation(keys, "E-CHECK-OUTOFSCOPE", "RULE-TEST") {
		t.Errorf("expected E-CHECK-OUTOFSCOPE/RULE-TEST violation, got: %v", result.Violations)
	}
}

// TestRun_ECheckOutOfScope_NoViolation: governs ⊆ scope → 違反なし
func TestRun_ECheckOutOfScope_NoViolation(t *testing.T) {
	dir := t.TempDir()
	checkFile := "internal/domain/check_test.go"
	writeFile(t, filepath.Join(dir, "internal/domain/handler.go"), "package domain\n")

	enforceTag := "@warrant-" + "enforces RULE-TEST"
	writeFile(t, filepath.Join(dir, checkFile), "// "+enforceTag+"\npackage domain_test\n")
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n\n## Section {#section}\n\n")

	// governs に Glob でマッチするパターンを使う（"internal/domain/**/*.go"）
	// Glob で列挙される handler.go と check_test.go はいずれも scope "internal/domain/**" に FnMatch する
	hash := CanonicalHash("RULE-TEST", "In Scope", "active", "const.md#section",
		[]string{"internal/domain/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: checkFile, Governs: []string{"internal/domain/**/*.go"}},
		},
	)

	rulesYaml := `rules:
  - id: RULE-TEST
    title: "In Scope"
    basis: const.md#section
    scope:
      - "internal/domain/**"
    enforced_by:
      - kind: deterministic
        ref: ` + checkFile + `
        governs:
          - "internal/domain/**/*.go"
    ratification:
      content_hash: ` + hash + `
`
	writeFile(t, filepath.Join(dir, "rules.yaml"), rulesYaml)

	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if hasViolation(keys, "E-CHECK-OUTOFSCOPE", "RULE-TEST") {
		t.Errorf("unexpected E-CHECK-OUTOFSCOPE violation: %v", result.Violations)
	}
}

// TestRun_ECheckOutOfScope_ScopeEmpty: scope が空なら越境判定スキップ → 違反なし
func TestRun_ECheckOutOfScope_ScopeEmpty(t *testing.T) {
	dir := t.TempDir()
	checkFile := "check_test.go"
	writeFile(t, filepath.Join(dir, "internal/other/service.go"), "package other\n")

	enforceTag := "@warrant-" + "enforces RULE-TEST"
	writeFile(t, filepath.Join(dir, checkFile), "// "+enforceTag+"\npackage authority_test\n")
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n\n## Section {#section}\n\n")

	// scope なし、governs あり → スキップ（Glob でマッチするパターンを使う）
	hash := CanonicalHash("RULE-TEST", "No Scope", "active", "const.md#section",
		nil,
		[]EnforceEntry{
			{Kind: "deterministic", Ref: checkFile, Governs: []string{"internal/**/*.go"}},
		},
	)

	rulesYaml := `rules:
  - id: RULE-TEST
    title: "No Scope"
    basis: const.md#section
    enforced_by:
      - kind: deterministic
        ref: ` + checkFile + `
        governs:
          - "internal/**/*.go"
    ratification:
      content_hash: ` + hash + `
`
	writeFile(t, filepath.Join(dir, "rules.yaml"), rulesYaml)

	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if hasViolation(keys, "E-CHECK-OUTOFSCOPE", "RULE-TEST") {
		t.Errorf("unexpected E-CHECK-OUTOFSCOPE when scope is empty: %v", result.Violations)
	}
}

// TestRun_ECheckOutOfScope_GovernEmpty: governs が空なら越境判定スキップ → 違反なし
func TestRun_ECheckOutOfScope_GovernEmpty(t *testing.T) {
	dir := t.TempDir()
	checkFile := "check_test.go"

	enforceTag := "@warrant-" + "enforces RULE-TEST"
	writeFile(t, filepath.Join(dir, checkFile), "// "+enforceTag+"\npackage authority_test\n")
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n\n## Section {#section}\n\n")

	// scope あり、governs なし → スキップ
	hash := CanonicalHash("RULE-TEST", "No Governs", "active", "const.md#section",
		[]string{"internal/domain/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: checkFile},
		},
	)

	rulesYaml := `rules:
  - id: RULE-TEST
    title: "No Governs"
    basis: const.md#section
    scope:
      - "internal/domain/**"
    enforced_by:
      - kind: deterministic
        ref: ` + checkFile + `
    ratification:
      content_hash: ` + hash + `
`
	writeFile(t, filepath.Join(dir, "rules.yaml"), rulesYaml)

	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if hasViolation(keys, "E-CHECK-OUTOFSCOPE", "RULE-TEST") {
		t.Errorf("unexpected E-CHECK-OUTOFSCOPE when governs is empty: %v", result.Violations)
	}
}

// TestRun_ECheckOutOfScope_GovernMatchesZeroFiles: governs がマッチするファイル0件 → 違反なし
func TestRun_ECheckOutOfScope_GovernMatchesZeroFiles(t *testing.T) {
	dir := t.TempDir()
	checkFile := "check_test.go"

	enforceTag := "@warrant-" + "enforces RULE-TEST"
	writeFile(t, filepath.Join(dir, checkFile), "// "+enforceTag+"\npackage authority_test\n")
	writeFile(t, filepath.Join(dir, "const.md"), "# Constitution\n\n## Section {#section}\n\n")

	// internal/domain/ 配下にファイルが存在しないため governs パターンがマッチするファイルが0件 → 越境なし
	hash := CanonicalHash("RULE-TEST", "No Match", "active", "const.md#section",
		[]string{"internal/domain/**"},
		[]EnforceEntry{
			{Kind: "deterministic", Ref: checkFile, Governs: []string{"internal/domain/**/*.go"}},
		},
	)

	rulesYaml := `rules:
  - id: RULE-TEST
    title: "No Match"
    basis: const.md#section
    scope:
      - "internal/domain/**"
    enforced_by:
      - kind: deterministic
        ref: ` + checkFile + `
        governs:
          - "internal/domain/**/*.go"
    ratification:
      content_hash: ` + hash + `
`
	writeFile(t, filepath.Join(dir, "rules.yaml"), rulesYaml)

	cfg := defaultCfg()
	result, err := Run(dir, filepath.Join(dir, "rules.yaml"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	keys := collectViolationKeys(result)
	if hasViolation(keys, "E-CHECK-OUTOFSCOPE", "RULE-TEST") {
		t.Errorf("unexpected E-CHECK-OUTOFSCOPE when governs matches 0 files: %v", result.Violations)
	}
}
