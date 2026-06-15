package check

import (
	"math"
	"time"

	"github.com/9uiLe/warrant/internal/projection"
)

// BuildReport は check 結果から projection.Report を構築する
func BuildReport(reqs []Requirement, vs []projection.Violation, meta projection.ReportMeta, generatedAt string) projection.Report {
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	meta.GeneratedAt = generatedAt

	// 違反がある要件IDのセット
	violating := make(map[string]struct{})
	for _, v := range vs {
		violating[v.Requirement] = struct{}{}
	}

	var rreqs []projection.ReportRequirement
	if rreqs == nil {
		rreqs = []projection.ReportRequirement{}
	}

	passed, failed, draft := 0, 0, 0

	for _, req := range reqs {
		var verdict string
		if req.Status != "active" {
			verdict = "draft"
			draft++
		} else if _, bad := violating[req.ID]; bad {
			verdict = "fail"
			failed++
		} else {
			verdict = "pass"
			passed++
		}

		// Tests
		tests := make([]projection.ReportTest, 0, len(req.TestRefs))
		for _, tr := range req.TestRefs {
			status := "broken"
			if tr.Linked {
				status = "linked"
			}
			tests = append(tests, projection.ReportTest{File: tr.File, Status: status})
		}

		// Spec
		var specStatus string
		if req.SpecDoc == "" {
			specStatus = "missing"
		} else if req.SpecOK {
			specStatus = "linked"
		} else {
			specStatus = "broken"
		}

		rreqs = append(rreqs, projection.ReportRequirement{
			ID:      req.ID,
			Title:   req.Title,
			Status:  req.Status,
			Verdict: verdict,
			Tests:   tests,
			Spec:    projection.ReportSpec{Doc: req.SpecDoc, Section: req.SpecSec, Status: specStatus},
		})
	}

	total := len(reqs)
	var coveragePercent int
	if total > 0 {
		coveragePercent = int(math.Round(float64(passed) / float64(total) * 100))
	}

	return projection.Report{
		Meta: meta,
		Summary: projection.ReportSummary{
			Total:           total,
			Passed:          passed,
			Failed:          failed,
			Draft:           draft,
			CoveragePercent: coveragePercent,
		},
		Requirements: rreqs,
	}
}
