// Package main は warrant advise の semantic judge ポートの参照スタブ実装。
// LLM は呼ばず、受信した Request を echo して verdict=uncertain を返す。
// 目的: 外部コマンド契約の動作確認と置換性のデモ。実運用では LLM 等を呼ぶ実装に差し替えること。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// request は外部 judge コマンドが stdin から受け取る JSON ペイロード。
// フィールド名は internal/semantic.Request と完全一致させること。
type request struct {
	RuleID    string   `json:"rule_id"`
	Title     string   `json:"title"`
	Basis     string   `json:"basis"`
	Criterion string   `json:"criterion"`
	Targets   []string `json:"targets"`
}

// verdict は外部 judge コマンドが stdout へ出力する JSON ペイロード。
// フィールド名は internal/semantic.Verdict と完全一致させること。
type verdict struct {
	Verdict           string `json:"verdict"`
	Rationale         string `json:"rationale"`
	ProposedAssertion string `json:"proposed_assertion,omitempty"`
}

// decideVerdict は Request から決定論的に Verdict を返す参照スタブの中核。
// LLM は呼ばず、常に uncertain を返して受信内容を rationale に echo する。
func decideVerdict(req request) verdict {
	rationale := fmt.Sprintf(
		"[参照スタブ] rule_id=%q criterion=%q targets件数=%d — "+
			"これは参照スタブであり実際の意味判定はしていない。実運用では LLM 等に置き換えること。",
		req.RuleID, req.Criterion, len(req.Targets),
	)
	return verdict{
		Verdict:           "uncertain",
		Rationale:         rationale,
		ProposedAssertion: "",
	}
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stdin 読み込み失敗: %v\n", err)
		os.Exit(1)
	}

	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		fmt.Fprintf(os.Stderr, "Request JSON デコード失敗: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(decideVerdict(req)); err != nil {
		fmt.Fprintf(os.Stderr, "Verdict JSON エンコード失敗: %v\n", err)
		os.Exit(1)
	}
}
