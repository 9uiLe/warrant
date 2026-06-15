package check

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/glob"
	"github.com/9uiLe/warrant/internal/projection"
	"github.com/9uiLe/warrant/internal/registry"
	"github.com/9uiLe/warrant/internal/scan"
)

// TestRef はテストファイルの参照とリンク状態
type TestRef struct {
	File   string
	Linked bool
}

// Requirement は正規化された要件
type Requirement struct {
	ID       string
	Title    string
	Status   string
	SpecDoc  string
	SpecSec  string
	Tests    []string
	SpecOK   bool
	TestRefs []TestRef
}

// Result はチェック結果
type Result struct {
	Violations   []projection.Violation
	Requirements []Requirement
}

// Run はトレーサビリティチェックを実行する
func Run(root string, reg *registry.Registry, cfg *config.Config) (*Result, error) {
	tagIndex, err := scan.Run(root, cfg.TestGlobs, cfg.Tag, cfg.IDPattern)
	if err != nil {
		return nil, err
	}

	reqs := reg.Requirements()
	idRx := regexp.MustCompile(`^` + cfg.IDPattern + `$`)

	var violations []projection.Violation
	var requirements []Requirement

	// 宣言済み ID セット
	declared := make(map[string]struct{})
	for _, item := range reqs {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := m["id"].(string); id != "" {
			declared[id] = struct{}{}
		}
	}

	seen := make(map[string]struct{})

	for _, item := range reqs {
		m, ok := item.(map[string]any)
		if !ok {
			violations = append(violations, projection.Violation{
				Code:        "E-SCHEMA",
				Requirement: "?",
				Message:     fmt.Sprintf("要件が mapping ではない: %v", item),
			})
			continue
		}

		rid, _ := m["id"].(string)
		if rid == "" {
			violations = append(violations, projection.Violation{
				Code:        "E-SCHEMA",
				Requirement: "?",
				Message:     "要件に id がない",
			})
			continue
		}

		if !idRx.MatchString(rid) {
			violations = append(violations, projection.Violation{
				Code:        "E-ID-FORMAT",
				Requirement: rid,
				Message:     "id が id_pattern に一致しない: " + rid,
			})
		}

		if _, dup := seen[rid]; dup {
			violations = append(violations, projection.Violation{
				Code:        "E-ID-DUP",
				Requirement: rid,
				Message:     "id が重複している",
			})
		}
		seen[rid] = struct{}{}

		title, _ := m["title"].(string)
		if title == "" {
			violations = append(violations, projection.Violation{
				Code:        "E-SCHEMA",
				Requirement: rid,
				Message:     "title がない",
			})
		}

		status, _ := m["status"].(string)
		if status == "" {
			status = "active"
		}

		specDoc := ""
		specSec := ""

		// Python 逐語: E-SPEC-DERIVED と E-SPEC-NOFILE / E-SPEC-NOSECTION は
		// それぞれ独立した if（派生かつ実在しない doc なら両方発火する）。else if にしない。
		specMap, specIsMap := m["spec"].(map[string]any)
		doc := ""
		if specIsMap {
			doc, _ = specMap["doc"].(string)
		}
		if !specIsMap || doc == "" {
			violations = append(violations, projection.Violation{
				Code:        "E-SPEC-MISSING",
				Requirement: rid,
				Message:     "spec.doc がない（立法の根拠が宣言されていない）",
			})
		} else {
			specDoc = doc
			if derivedMatch(doc, cfg.DerivedGlobs) {
				violations = append(violations, projection.Violation{
					Code:        "E-SPEC-DERIVED",
					Requirement: rid,
					Message:     "spec.doc が派生データを指している（判断根拠にできない）: " + doc,
				})
			}
			if !isFile(filepath.Join(root, doc)) {
				violations = append(violations, projection.Violation{
					Code:        "E-SPEC-NOFILE",
					Requirement: rid,
					Message:     "spec.doc が実在しない: " + doc,
				})
			} else {
				sec, _ := specMap["section"].(string)
				if sec != "" {
					content, err := os.ReadFile(filepath.Join(root, doc))
					if err == nil && !strings.Contains(string(content), sec) {
						violations = append(violations, projection.Violation{
							Code:        "E-SPEC-NOSECTION",
							Requirement: rid,
							Message:     fmt.Sprintf("spec.section が doc 内に見つからない: %q in %s", sec, doc),
						})
					}
				}
				specSec = sec
			}
		}

		// Python 逐語: tests = req.get("tests") or []。E-NOTEST は「生リスト」が空かで判定し、
		// 実在判定後の testFiles 長では判定しない（存在しないテストの宣言でも E-NOTEST は出ない）。
		var testsList []any
		numRawTests := 0
		switch v := m["tests"].(type) {
		case []any:
			testsList = v
			numRawTests = len(v)
		case nil:
			numRawTests = 0
		default:
			// 非リストの truthy 値（稀）。Python は `or []` で保持し not tests が False になる。
			numRawTests = 1
		}

		if status == "active" && numRawTests == 0 {
			violations = append(violations, projection.Violation{
				Code:        "E-NOTEST",
				Requirement: rid,
				Message:     "active な要件にテストが 1 件もない（証明されていない機能）",
			})
		}

		var testFiles []string
		var testRefs []TestRef
		for _, t := range testsList {
			var tfile string
			switch v := t.(type) {
			case string:
				tfile = v
			case map[string]any:
				tfile, _ = v["file"].(string)
			}
			if tfile == "" {
				violations = append(violations, projection.Violation{
					Code:        "E-TEST-SCHEMA",
					Requirement: rid,
					Message:     fmt.Sprintf("tests 要素が不正: %v", t),
				})
				continue
			}
			// projection 用に宣言された全テストファイルを保持（Python の report と同様、
			// 実在しないテストも一覧に含める）。
			testFiles = append(testFiles, tfile)
			fileExists := isFile(filepath.Join(root, tfile))
			tfileSlash := filepath.ToSlash(tfile)
			_, hasTag := tagIndex[rid][tfileSlash]
			// TestRef は continue 前に追加する（ファイルが実在しない場合でも全テストファイルを収集し、
			// linked 判定はファイル実在かつタグあり の両方が必要）。
			testRefs = append(testRefs, TestRef{File: tfile, Linked: fileExists && hasTag})
			if !fileExists {
				violations = append(violations, projection.Violation{
					Code:        "E-TEST-NOFILE",
					Requirement: rid,
					Message:     "テストが実在しない: " + tfile,
				})
				continue
			}
			if !hasTag {
				violations = append(violations, projection.Violation{
					Code:        "E-TAG-MISSING",
					Requirement: rid,
					Message:     fmt.Sprintf("テスト %s に `%s %s` タグがない（要件→テストの宣言とテスト側の自己申告が不一致）", tfile, cfg.Tag, rid),
				})
			}
		}

		specOK := specDoc != "" && isFile(filepath.Join(root, specDoc)) && !derivedMatch(specDoc, cfg.DerivedGlobs)

		requirements = append(requirements, Requirement{
			ID:       rid,
			Title:    title,
			Status:   status,
			SpecDoc:  specDoc,
			SpecSec:  specSec,
			Tests:    testFiles,
			SpecOK:   specOK,
			TestRefs: testRefs,
		})
	}

	// 孤児タグ（決定論のため tag_id をソートして走査）
	orphanIDs := make([]string, 0, len(tagIndex))
	for tagID := range tagIndex {
		orphanIDs = append(orphanIDs, tagID)
	}
	sort.Strings(orphanIDs)

	for _, tagID := range orphanIDs {
		if _, ok := declared[tagID]; !ok {
			files := tagIndex[tagID]
			sortedFiles := make([]string, 0, len(files))
			for f := range files {
				sortedFiles = append(sortedFiles, f)
			}
			sort.Strings(sortedFiles)
			violations = append(violations, projection.Violation{
				Code:        "E-TAG-ORPHAN",
				Requirement: tagID,
				Message:     fmt.Sprintf("未知の要件 ID を指す @covers タグ（要件未登録）: %v", sortedFiles),
			})
		}
	}

	return &Result{
		Violations:   violations,
		Requirements: requirements,
	}, nil
}

// BuildGraph は check 結果から projection.Graph を構築する
func BuildGraph(reqs []Requirement, vs []projection.Violation, generatedAt string) projection.Graph {
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	verdict := "PASS"
	if len(vs) > 0 {
		verdict = "FAIL"
	}

	var nodes []projection.Node
	var edges []projection.Edge
	nodeSet := make(map[string]struct{})

	addNode := func(id, kind, label string) {
		if _, exists := nodeSet[id]; !exists {
			nodeSet[id] = struct{}{}
			nodes = append(nodes, projection.Node{ID: id, Kind: kind, Label: label})
		}
	}

	for _, req := range reqs {
		reqNodeID := req.ID
		addNode(reqNodeID, "requirement", req.Title)

		if req.SpecDoc != "" {
			specNodeID := "spec:" + req.SpecDoc
			addNode(specNodeID, "spec", req.SpecDoc)
			edges = append(edges, projection.Edge{From: reqNodeID, To: specNodeID, Kind: "spec"})
		}

		for _, t := range req.Tests {
			testNodeID := "test:" + t
			addNode(testNodeID, "test", t)
			edges = append(edges, projection.Edge{From: reqNodeID, To: testNodeID, Kind: "test"})
		}
	}

	if nodes == nil {
		nodes = []projection.Node{}
	}
	if edges == nil {
		edges = []projection.Edge{}
	}
	if vs == nil {
		vs = []projection.Violation{}
	}

	return projection.Graph{
		Verdict:          verdict,
		RequirementCount: len(reqs),
		Violations:       vs,
		Nodes:            nodes,
		Edges:            edges,
		GeneratedAt:      generatedAt,
	}
}

func derivedMatch(doc string, derivedGlobs []string) bool {
	for _, pattern := range derivedGlobs {
		if glob.FnMatch(doc, pattern) {
			return true
		}
	}
	return false
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
