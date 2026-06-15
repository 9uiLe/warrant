// @warrant-covers WARRANT-CHECK
package check

import (
	"testing"

	"github.com/9uiLe/warrant/internal/projection"
)

func TestBuildReport_Verdict(t *testing.T) {
	reqs := []Requirement{
		{ID: "REQ-1", Title: "Req1", Status: "active"},
		{ID: "REQ-2", Title: "Req2", Status: "active"},
		{ID: "REQ-3", Title: "Req3", Status: "draft"},
	}
	vs := []projection.Violation{
		{Code: "E-TEST", Requirement: "REQ-1", Message: "fail"},
	}
	rep := BuildReport(reqs, vs, projection.ReportMeta{}, "2024-01-01T00:00:00Z")

	// 違反あり active → fail
	if rep.Requirements[0].Verdict != "fail" {
		t.Errorf("REQ-1 verdict = %q, want fail", rep.Requirements[0].Verdict)
	}
	// 違反なし active → pass
	if rep.Requirements[1].Verdict != "pass" {
		t.Errorf("REQ-2 verdict = %q, want pass", rep.Requirements[1].Verdict)
	}
	// draft status → draft
	if rep.Requirements[2].Verdict != "draft" {
		t.Errorf("REQ-3 verdict = %q, want draft", rep.Requirements[2].Verdict)
	}
}

func TestBuildReport_Summary(t *testing.T) {
	reqs := []Requirement{
		{ID: "A", Title: "A", Status: "active"},
		{ID: "B", Title: "B", Status: "active"},
		{ID: "C", Title: "C", Status: "active"},
		{ID: "D", Title: "D", Status: "draft"},
	}
	vs := []projection.Violation{
		{Code: "E-X", Requirement: "A", Message: "x"},
	}
	rep := BuildReport(reqs, vs, projection.ReportMeta{}, "")

	if rep.Summary.Total != 4 {
		t.Errorf("Total = %d, want 4", rep.Summary.Total)
	}
	if rep.Summary.Passed != 2 {
		t.Errorf("Passed = %d, want 2", rep.Summary.Passed)
	}
	if rep.Summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", rep.Summary.Failed)
	}
	if rep.Summary.Draft != 1 {
		t.Errorf("Draft = %d, want 1", rep.Summary.Draft)
	}
	// coverage = round(2/4*100) = 50
	if rep.Summary.CoveragePercent != 50 {
		t.Errorf("CoveragePercent = %d, want 50", rep.Summary.CoveragePercent)
	}
}

func TestBuildReport_EmptyInput(t *testing.T) {
	rep := BuildReport(nil, nil, projection.ReportMeta{}, "")
	if rep.Requirements == nil {
		t.Error("Requirements should not be nil")
	}
	if len(rep.Requirements) != 0 {
		t.Errorf("Requirements length = %d, want 0", len(rep.Requirements))
	}
	if rep.Summary.CoveragePercent != 0 {
		t.Errorf("CoveragePercent = %d, want 0", rep.Summary.CoveragePercent)
	}
}

func TestBuildReport_TestRefs(t *testing.T) {
	reqs := []Requirement{
		{
			ID:     "R1",
			Title:  "R1",
			Status: "active",
			TestRefs: []TestRef{
				{File: "a_test.go", Linked: true},
				{File: "b_test.go", Linked: false},
			},
		},
	}
	rep := BuildReport(reqs, nil, projection.ReportMeta{}, "")
	tests := rep.Requirements[0].Tests
	if len(tests) != 2 {
		t.Fatalf("Tests len = %d, want 2", len(tests))
	}
	if tests[0].Status != "linked" {
		t.Errorf("tests[0].Status = %q, want linked", tests[0].Status)
	}
	if tests[1].Status != "broken" {
		t.Errorf("tests[1].Status = %q, want broken", tests[1].Status)
	}
}
