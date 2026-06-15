package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/9uiLe/warrant/internal/authority"
	"github.com/9uiLe/warrant/internal/config"
	"gopkg.in/yaml.v3"
)

const ratifyConsentNote = "これはルール本文への人間の承認(content_hash の確定)を記録する操作です (ref: P17)"

// runRatify は ratify サブコマンドのフラグをパースして実行する
func runRatify(args []string) int {
	fs := flag.NewFlagSet("ratify", flag.ContinueOnError)
	rootFlag := fs.String("repo-root", ".", "repo root directory")
	rulesPath := fs.String("rules", "", "rules.yaml path")
	cfgPath := fs.String("config", "", "config file path")
	ruleID := fs.String("rule", "", "このルール ID のみ承認する")
	all := fs.Bool("all", false, "全ルールを承認する")
	write := fs.Bool("write", false, "rules.yaml に書き込む（既定は dry-run）")
	approvedBy := fs.String("approved-by", "", "ratification.approved_by に記録する承認者名")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root := *rootFlag
	if *rulesPath == "" {
		*rulesPath = filepath.Join(root, ".warrant", "rules.yaml")
	}
	if *cfgPath == "" {
		*cfgPath = filepath.Join(root, ".warrant", "config.yaml")
	}

	return ratifyRun(root, *rulesPath, *cfgPath, *ruleID, *all, *write, *approvedBy, os.Stdout)
}

// ratifyRun は ratify の中核。テストから直接呼べるよう out を注入する。
// 終了コード: 0=成功(dry-run/書き込み), 2=使用方法・IO エラー
func ratifyRun(root, rulesPath, cfgPath, ruleID string, all, write bool, approvedBy string, out io.Writer) int {
	// 強制機能(forcing function): --write は対象を明示させ、意図しない全件承認を構造的に封じる
	if write && ruleID == "" && !all {
		fmt.Fprintln(out, "ratify: --write には --rule <ID> または --all が必要です（意図しない全件承認を防ぐため）")
		return 2
	}

	var cfg *config.Config
	if c, err := config.Load(cfgPath); err == nil {
		cfg = c
	} else {
		cfg = config.Default()
	}

	// 正規化済みルールと期待ハッシュを得る。CanonicalHash は check/advise と同一の出所(ドリフト不能)。
	res, err := authority.Run(root, rulesPath, cfg)
	if err != nil {
		fmt.Fprintf(out, "ratify: rules.yaml 読み込みエラー: %v\n", err)
		return 2
	}
	expected := make(map[string]string, len(res.Rules))
	for _, r := range res.Rules {
		expected[r.ID] = authority.CanonicalHash(r.ID, r.Title, r.Status, r.Basis, r.Scope, r.EnforcedBy)
	}

	// rules.yaml を yaml.Node として読み、content_hash のみ外科的に更新する(コメント・構造を保全)
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		fmt.Fprintf(out, "ratify: rules.yaml 読み込みエラー: %v\n", err)
		return 2
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		fmt.Fprintf(out, "ratify: rules.yaml パースエラー: %v\n", err)
		return 2
	}

	ruleNodes := ruleMappingNodes(&doc)

	// 対象選択: --rule 指定なら単一(無ければエラー)、それ以外は全件
	found := false
	changed := false

	type status struct {
		id    string
		state string
		old   string
		new   string
	}
	var statuses []status

	for _, rn := range ruleNodes {
		idNode := mapValueNode(rn, "id")
		if idNode == nil || idNode.Value == "" {
			continue
		}
		id := idNode.Value
		if ruleID != "" && id != ruleID {
			continue
		}
		found = true

		exp, ok := expected[id]
		if !ok {
			// authority.Run が正規化できなかったルール(id 形式不正など)はスキップ
			statuses = append(statuses, status{id: id, state: "SKIP"})
			continue
		}

		stored := ""
		if rat := mapValueNode(rn, "ratification"); rat != nil {
			if ch := mapValueNode(rat, "content_hash"); ch != nil {
				stored = ch.Value
			}
		}

		if stored == exp {
			statuses = append(statuses, status{id: id, state: "MATCH", new: exp})
			// approved_by だけ更新したいケースに対応
			if write && approvedBy != "" && setApprovedBy(rn, approvedBy) {
				changed = true
				statuses[len(statuses)-1].state = "UPDATED"
			}
			continue
		}

		st := status{id: id, old: stored, new: exp}
		if write {
			setContentHash(rn, exp)
			if approvedBy != "" {
				setApprovedBy(rn, approvedBy)
			}
			st.state = "UPDATED"
			changed = true
		} else {
			st.state = "WILL-UPDATE"
		}
		statuses = append(statuses, st)
	}

	if ruleID != "" && !found {
		fmt.Fprintf(out, "ratify: ルール %q が rules.yaml に見つかりません\n", ruleID)
		return 2
	}

	// 出力
	if write {
		fmt.Fprintf(out, "ratify: %s。\n", ratifyConsentNote)
	} else {
		fmt.Fprintf(out, "ratify (dry-run): 変更は書き込まれません。--write で rules.yaml を更新します。\n")
		fmt.Fprintf(out, "%s。\n", ratifyConsentNote)
	}
	for _, s := range statuses {
		switch s.state {
		case "MATCH":
			fmt.Fprintf(out, "  %-28s MATCH        %s\n", s.id, s.new)
		case "SKIP":
			fmt.Fprintf(out, "  %-28s SKIP         (正規化不可: id 形式などを確認)\n", s.id)
		case "WILL-UPDATE", "UPDATED":
			oldDisp := s.old
			if oldDisp == "" {
				oldDisp = "(未承認)"
			}
			fmt.Fprintf(out, "  %-28s %-12s %s -> %s\n", s.id, s.state, oldDisp, s.new)
		}
	}

	if write {
		if !changed {
			fmt.Fprintln(out, "ratify: 更新対象はありません（すべて承認済み）。")
			return 0
		}
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(&doc); err != nil {
			fmt.Fprintf(out, "ratify: rules.yaml エンコードエラー: %v\n", err)
			return 2
		}
		_ = enc.Close()
		if err := os.WriteFile(rulesPath, buf.Bytes(), 0644); err != nil {
			fmt.Fprintf(out, "ratify: rules.yaml 書き込みエラー: %v\n", err)
			return 2
		}
		fmt.Fprintln(out, "ratify: rules.yaml を更新しました。")
	}

	return 0
}

// ruleMappingNodes は doc から rules シーケンス配下の各ルール mapping ノードを返す
func ruleMappingNodes(doc *yaml.Node) []*yaml.Node {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	top := doc.Content[0]
	rulesVal := mapValueNode(top, "rules")
	if rulesVal == nil || rulesVal.Kind != yaml.SequenceNode {
		return nil
	}
	var out []*yaml.Node
	for _, item := range rulesVal.Content {
		if item.Kind == yaml.MappingNode {
			out = append(out, item)
		}
	}
	return out
}

// mapValueNode は mapping ノードから key に対応する value ノードを返す（無ければ nil）
func mapValueNode(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// setContentHash は rule mapping の ratification.content_hash を val に設定する。
// ratification ブロックや content_hash キーが無ければ生成する。
func setContentHash(rule *yaml.Node, val string) {
	rat := mapValueNode(rule, "ratification")
	if rat == nil {
		rat = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		appendScalarKV(rat, "content_hash", val)
		appendKV(rule, "ratification", rat)
		return
	}
	if rat.Kind != yaml.MappingNode {
		rat.Kind = yaml.MappingNode
		rat.Tag = "!!map"
		rat.Value = ""
		rat.Content = nil
	}
	setScalar(rat, "content_hash", val)
}

// setApprovedBy は ratification.approved_by を設定する。変更があれば true を返す。
func setApprovedBy(rule *yaml.Node, name string) bool {
	rat := mapValueNode(rule, "ratification")
	if rat == nil {
		rat = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		appendScalarKV(rat, "approved_by", name)
		appendKV(rule, "ratification", rat)
		return true
	}
	if cur := mapValueNode(rat, "approved_by"); cur != nil {
		if cur.Value == name {
			return false
		}
		cur.Kind = yaml.ScalarNode
		cur.Tag = "!!str"
		cur.Value = name
		cur.Style = yaml.DoubleQuotedStyle
		return true
	}
	appendScalarKV(rat, "approved_by", name)
	return true
}

// setScalar は mapping の key を val(文字列スカラ)に設定。無ければ追加する。
func setScalar(m *yaml.Node, key, val string) {
	if v := mapValueNode(m, key); v != nil {
		v.Kind = yaml.ScalarNode
		v.Tag = "!!str"
		v.Value = val
		v.Style = yaml.DoubleQuotedStyle
		return
	}
	appendScalarKV(m, key, val)
}

// appendScalarKV は mapping に key: "val"(double-quoted スカラ)を追加する
func appendScalarKV(m *yaml.Node, key, val string) {
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	v := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: val, Style: yaml.DoubleQuotedStyle}
	m.Content = append(m.Content, k, v)
}

// appendKV は mapping に key: valueNode を追加する
func appendKV(m *yaml.Node, key string, value *yaml.Node) {
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	m.Content = append(m.Content, k, value)
}
