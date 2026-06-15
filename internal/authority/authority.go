package authority

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/glob"
	"github.com/9uiLe/warrant/internal/projection"
	"github.com/9uiLe/warrant/internal/scan"
	"gopkg.in/yaml.v3"
)

// Rule は正規化されたルール
type Rule struct {
	ID         string
	Title      string
	Status     string
	Basis      string
	Scope      []string
	EnforcedBy []EnforceEntry
	Ratified   bool // content_hash が正しい場合 true
}

// EnforceEntry は enforced_by の一要素
type EnforceEntry struct {
	Kind      string
	Ref       string
	Governs   []string
	Criterion string
}

// Result は Authority チェック結果
type Result struct {
	Violations []projection.Violation
	Rules      []Rule
}

// CanonicalHash はルール本文の正規ハッシュを計算する
// canonical =
//
//	"id:" + id + "\n" +
//	"title:" + title + "\n" +
//	"status:" + status(空なら"active") + "\n" +
//	"basis:" + basis + "\n" +
//	"scope:" + strings.Join(sortedScope, ",") + "\n" +
//	"enforced_by:" + strings.Join(sortedEntries, ",") + "\n"
//
// sortedScope: scope グロブを昇順ソートして結合
// sortedEntries: 各 enforced_by を "kind|ref|governs1;governs2;..." の文字列にし昇順ソート
func CanonicalHash(id, title, status, basis string, scope []string, enforcedBy []EnforceEntry) string {
	if status == "" {
		status = "active"
	}
	sortedScope := make([]string, len(scope))
	copy(sortedScope, scope)
	sort.Strings(sortedScope)

	entries := make([]string, 0, len(enforcedBy))
	for _, e := range enforcedBy {
		sortedGoverns := make([]string, len(e.Governs))
		copy(sortedGoverns, e.Governs)
		sort.Strings(sortedGoverns)
		entries = append(entries, e.Kind+"|"+e.Ref+"|"+strings.Join(sortedGoverns, ";")+"|"+e.Criterion)
	}
	sort.Strings(entries)

	canonical := "id:" + id + "\n" +
		"title:" + title + "\n" +
		"status:" + status + "\n" +
		"basis:" + basis + "\n" +
		"scope:" + strings.Join(sortedScope, ",") + "\n" +
		"enforced_by:" + strings.Join(entries, ",") + "\n"
	h := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(h[:])
}

// loadRules は rules.yaml をロードしてルールの生リストを返す
func loadRules(path string) ([]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	v, ok := raw["rules"]
	if !ok || v == nil {
		return nil, nil
	}
	rules, ok := v.([]any)
	if !ok {
		return nil, nil
	}
	return rules, nil
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Run は Authority チェックを実行する
func Run(root string, rulesPath string, cfg *config.Config) (*Result, error) {
	rules, err := loadRules(rulesPath)
	if err != nil {
		return nil, err
	}

	// enforceTag でチェックファイルをスキャン (既存 test_globs を流用)
	enforceIndex, err := scan.Run(root, cfg.TestGlobs, cfg.EnforceTag, cfg.IDPattern)
	if err != nil {
		return nil, err
	}

	idRx := regexp.MustCompile(`^` + cfg.IDPattern + `$`)

	var violations []projection.Violation
	var resultRules []Rule

	// 宣言済み ID セット
	declared := make(map[string]struct{})
	for _, item := range rules {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := m["id"].(string); id != "" {
			declared[id] = struct{}{}
		}
	}

	seen := make(map[string]struct{})

	for _, item := range rules {
		m, ok := item.(map[string]any)
		if !ok {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-SCHEMA",
				Requirement: "?",
				Message:     fmt.Sprintf("ルールが mapping ではない: %v", item),
			})
			continue
		}

		rid, _ := m["id"].(string)
		if rid == "" {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-SCHEMA",
				Requirement: "?",
				Message:     "ルールに id がない",
			})
			continue
		}

		if !idRx.MatchString(rid) {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-ID-FORMAT",
				Requirement: rid,
				Message:     "id が id_pattern に一致しない: " + rid,
			})
		}

		if _, dup := seen[rid]; dup {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-ID-DUP",
				Requirement: rid,
				Message:     "id が重複している",
			})
		}
		seen[rid] = struct{}{}

		title, _ := m["title"].(string)
		if title == "" {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-SCHEMA",
				Requirement: rid,
				Message:     "title がない",
			})
		}

		status, _ := m["status"].(string)
		if status == "" {
			status = "active"
		}

		basis, _ := m["basis"].(string)

		// basis 検査
		if basis == "" {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-NOBASIS",
				Requirement: rid,
				Message:     "basis が宣言されていない（立法の根拠がない）",
			})
		} else {
			// basis は "file#anchor" 形式
			parts := strings.SplitN(basis, "#", 2)
			basisFile := parts[0]
			basisAnchor := ""
			if len(parts) == 2 {
				basisAnchor = parts[1]
			}
			basisAbsPath := filepath.Join(root, basisFile)
			if !isFile(basisAbsPath) {
				violations = append(violations, projection.Violation{
					Code:        "E-RULE-BASIS-NOFILE",
					Requirement: rid,
					Message:     "basis のファイルが存在しない: " + basisFile,
				})
			} else if basisAnchor != "" {
				content, err := os.ReadFile(basisAbsPath)
				if err == nil && !strings.Contains(string(content), "{#"+basisAnchor+"}") {
					violations = append(violations, projection.Violation{
						Code:        "E-RULE-BASIS-NOANCHOR",
						Requirement: rid,
						Message:     fmt.Sprintf("basis のアンカーが憲法ファイル内に見つからない: {#%s} in %s", basisAnchor, basisFile),
					})
				}
			}
		}

		// scope パース
		var scope []string
		if sv, ok := m["scope"]; ok {
			if scopeList, ok := sv.([]any); ok {
				for _, s := range scopeList {
					if ss, ok := s.(string); ok && ss != "" {
						scope = append(scope, ss)
					}
				}
			}
		}

		// enforced_by パース
		var enforcedByRaw []any
		switch v := m["enforced_by"].(type) {
		case []any:
			enforcedByRaw = v
		}

		var enforcedBy []EnforceEntry
		hasDeterministicValid := false
		for _, e := range enforcedByRaw {
			em, ok := e.(map[string]any)
			if !ok {
				violations = append(violations, projection.Violation{
					Code:        "E-RULE-ENFORCE-SCHEMA",
					Requirement: rid,
					Message:     fmt.Sprintf("enforced_by 要素が不正: %v", e),
				})
				continue
			}
			kind, _ := em["kind"].(string)
			ref, _ := em["ref"].(string)
			if kind == "" || ref == "" {
				violations = append(violations, projection.Violation{
					Code:        "E-RULE-ENFORCE-SCHEMA",
					Requirement: rid,
					Message:     fmt.Sprintf("enforced_by 要素の kind/ref が取れない: %v", e),
				})
				continue
			}

			// governs パース
			var governs []string
			if gv, ok := em["governs"]; ok {
				if governsList, ok := gv.([]any); ok {
					for _, g := range governsList {
						if gs, ok := g.(string); ok && gs != "" {
							governs = append(governs, gs)
						}
					}
				}
			}

			criterion, _ := em["criterion"].(string)
			enforcedBy = append(enforcedBy, EnforceEntry{Kind: kind, Ref: ref, Governs: governs, Criterion: criterion})

			// deterministic 種別のみ検査
			if kind != "deterministic" {
				continue
			}
			refAbsPath := filepath.Join(root, ref)
			if !isFile(refAbsPath) {
				violations = append(violations, projection.Violation{
					Code:        "E-ENFORCE-NOFILE",
					Requirement: rid,
					Message:     "enforced_by の ref が存在しない: " + ref,
				})
				continue
			}
			refSlash := filepath.ToSlash(ref)
			_, hasTag := enforceIndex[rid][refSlash]
			if !hasTag {
				violations = append(violations, projection.Violation{
					Code:        "E-ENFORCE-TAG-MISSING",
					Requirement: rid,
					Message:     fmt.Sprintf("チェックファイル %s に `%s %s` タグがない", ref, cfg.EnforceTag, rid),
				})
			} else {
				hasDeterministicValid = true
			}

			// E-CHECK-OUTOFSCOPE: governs ⊆ scope の検証
			// scope と governs が両方非空のときのみ評価する（後方互換）
			if len(scope) > 0 && len(governs) > 0 {
				// governs の全パターンがマッチするファイル集合 G を列挙
				governsFileSet := make(map[string]struct{})
				for _, gp := range governs {
					matched, err := glob.Glob(root, gp)
					if err != nil {
						continue
					}
					for _, f := range matched {
						governsFileSet[f] = struct{}{}
					}
				}

				// G の各ファイルが scope のいずれかにマッチするか確認
				if len(governsFileSet) > 0 {
					var outOfScope []string
					for f := range governsFileSet {
						inScope := false
						for _, sp := range scope {
							if glob.FnMatch(f, sp) {
								inScope = true
								break
							}
						}
						if !inScope {
							outOfScope = append(outOfScope, f)
						}
					}
					sort.Strings(outOfScope)
					if len(outOfScope) > 0 {
						// 代表ファイルは先頭数件（最大3件）
						preview := outOfScope
						if len(preview) > 3 {
							preview = preview[:3]
						}
						violations = append(violations, projection.Violation{
							Code:        "E-CHECK-OUTOFSCOPE",
							Requirement: rid,
							Message:     fmt.Sprintf("チェック %s がルール %s の管轄(scope)外を裁いている。管轄外: %v", ref, rid, preview),
						})
					}
				}
			}
		}

		// E-RULE-UNENFORCED: active なルールに有効な deterministic enforced_by が1件もない
		if status == "active" && !hasDeterministicValid {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-UNENFORCED",
				Requirement: rid,
				Message:     "active なルールに kind=deterministic かつ実在・タグ付きの enforced_by が1件もない",
			})
		}

		// E-RULE-UNRATIFIED: content_hash 検証
		ratMap, _ := m["ratification"].(map[string]any)
		storedHash := ""
		if ratMap != nil {
			storedHash, _ = ratMap["content_hash"].(string)
		}
		expectedHash := CanonicalHash(rid, title, status, basis, scope, enforcedBy)
		ratified := storedHash == expectedHash && storedHash != ""
		if !ratified {
			violations = append(violations, projection.Violation{
				Code:        "E-RULE-UNRATIFIED",
				Requirement: rid,
				Message:     fmt.Sprintf("ルール本文の承認ハッシュが不一致。本文を確認の上 ratification.content_hash を %s に更新して承認してください", expectedHash),
			})
		}

		resultRules = append(resultRules, Rule{
			ID:         rid,
			Title:      title,
			Status:     status,
			Basis:      basis,
			Scope:      scope,
			EnforcedBy: enforcedBy,
			Ratified:   ratified,
		})
	}

	// 孤児タグ E-CHECK-ORPHAN（ソートで決定論化）
	orphanIDs := make([]string, 0, len(enforceIndex))
	for tagID := range enforceIndex {
		orphanIDs = append(orphanIDs, tagID)
	}
	sort.Strings(orphanIDs)

	for _, tagID := range orphanIDs {
		if _, ok := declared[tagID]; !ok {
			files := enforceIndex[tagID]
			sortedFiles := make([]string, 0, len(files))
			for f := range files {
				sortedFiles = append(sortedFiles, f)
			}
			sort.Strings(sortedFiles)
			violations = append(violations, projection.Violation{
				Code:        "E-CHECK-ORPHAN",
				Requirement: tagID,
				Message:     fmt.Sprintf("未知のルール ID を指す @warrant-enforces タグ（ルール未登録）: %v", sortedFiles),
			})
		}
	}

	return &Result{
		Violations: violations,
		Rules:      resultRules,
	}, nil
}

// BuildGraph は authority 結果から projection.Graph にノード/エッジを追加する（既存 Graph にマージ）
func BuildGraph(g *projection.Graph, rules []Rule) {
	nodeSet := make(map[string]struct{})
	for _, n := range g.Nodes {
		nodeSet[n.ID] = struct{}{}
	}
	addNode := func(id, kind, label string) {
		if _, exists := nodeSet[id]; !exists {
			nodeSet[id] = struct{}{}
			g.Nodes = append(g.Nodes, projection.Node{ID: id, Kind: kind, Label: label})
		}
	}

	for _, rule := range rules {
		ruleNodeID := "rule:" + rule.ID
		addNode(ruleNodeID, "rule", rule.Title)

		if rule.Basis != "" {
			constNodeID := "const:" + rule.Basis
			parts := strings.SplitN(rule.Basis, "#", 2)
			addNode(constNodeID, "constitution", parts[0])
			g.Edges = append(g.Edges, projection.Edge{From: ruleNodeID, To: constNodeID, Kind: "basis"})
		}

		for _, e := range rule.EnforcedBy {
			if e.Kind == "deterministic" {
				checkNodeID := "check:" + e.Ref
				addNode(checkNodeID, "check", e.Ref)
				g.Edges = append(g.Edges, projection.Edge{From: ruleNodeID, To: checkNodeID, Kind: "enforces"})
			}
		}
	}
}
