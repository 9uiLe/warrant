// @warrant-covers WARRANT-ADVISE
package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/9uiLe/warrant/internal/semantic"
)

// TestDecideVerdict_Deterministic は決定論的に uncertain を返し、
// rationale に入力の rule_id/criterion/targets件数 が反映されることを確認する。
func TestDecideVerdict_Deterministic(t *testing.T) {
	req := request{RuleID: "RULE-X", Criterion: "基準", Targets: []string{"a.go", "b.go"}}
	v := decideVerdict(req)
	if v.Verdict != "uncertain" {
		t.Errorf("verdict = %q, want uncertain", v.Verdict)
	}
	for _, want := range []string{"RULE-X", "基準", "targets件数=2"} {
		if !strings.Contains(v.Rationale, want) {
			t.Errorf("rationale に %q が含まれない: %s", want, v.Rationale)
		}
	}
}

// TestContractParity_Request は examples の request が internal/semantic.Request と
// JSON フィールド名で一致することを双方向の往復で保証する。
// このスタブは置換性のデモのため struct を意図的に複製しているが、本テストにより
// 本体の契約とドリフトした瞬間に CI が失敗する（腐らない参照実装）。
func TestContractParity_Request(t *testing.T) {
	sem := semantic.Request{
		RuleID:    "RULE-X",
		Title:     "t",
		Basis:     "b",
		Criterion: "c",
		Targets:   []string{"a.go", "b.go"},
	}

	// semantic.Request -> JSON -> example request
	var ex request
	roundTrip(t, sem, &ex)
	if ex.RuleID != sem.RuleID || ex.Title != sem.Title || ex.Basis != sem.Basis ||
		ex.Criterion != sem.Criterion || len(ex.Targets) != len(sem.Targets) {
		t.Errorf("semantic.Request -> example request でフィールドが欠落: %+v", ex)
	}

	// example request -> JSON -> semantic.Request
	var back semantic.Request
	roundTrip(t, ex, &back)
	if back.RuleID != sem.RuleID || back.Title != sem.Title || back.Basis != sem.Basis ||
		back.Criterion != sem.Criterion || len(back.Targets) != len(sem.Targets) {
		t.Errorf("example request -> semantic.Request でフィールドが欠落: %+v", back)
	}
}

// TestContractParity_Verdict は examples の verdict が internal/semantic.Verdict と
// JSON フィールド名で一致することを双方向の往復で保証する。
func TestContractParity_Verdict(t *testing.T) {
	ex := verdict{Verdict: "uncertain", Rationale: "r", ProposedAssertion: "p"}

	// example verdict -> JSON -> semantic.Verdict
	var sem semantic.Verdict
	roundTrip(t, ex, &sem)
	if sem.Verdict != ex.Verdict || sem.Rationale != ex.Rationale || sem.ProposedAssertion != ex.ProposedAssertion {
		t.Errorf("example verdict -> semantic.Verdict でフィールドが欠落: %+v", sem)
	}

	// semantic.Verdict -> JSON -> example verdict
	var back verdict
	roundTrip(t, sem, &back)
	if back.Verdict != ex.Verdict || back.Rationale != ex.Rationale || back.ProposedAssertion != ex.ProposedAssertion {
		t.Errorf("semantic.Verdict -> example verdict でフィールドが欠落: %+v", back)
	}
}

// roundTrip は src を JSON にして dst へデコードする
func roundTrip(t *testing.T, src, dst any) {
	t.Helper()
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
