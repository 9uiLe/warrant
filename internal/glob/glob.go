package glob

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// translateFnMatch は Python の fnmatch.translate 相当の変換を行う。
// * → (?s:.*) （DOTALL で / を含め何でもマッチ）
// ? → .
// [...] → 文字クラス（! を ^ に変換）
// その他は regexp.QuoteMeta でエスケープ
// 全体を ^ と $ で囲む
func translateFnMatch(pattern string) string {
	var sb strings.Builder
	sb.WriteString("^")
	i := 0
	for i < len(pattern) {
		ch := pattern[i]
		switch ch {
		case '*':
			sb.WriteString("(?s:.*)")
			i++
		case '?':
			sb.WriteString(".")
			i++
		case '[':
			// 文字クラスの終端 ']' を探す
			j := i + 1
			if j < len(pattern) && pattern[j] == '!' {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) {
				// ']' が見つからない場合はリテラルとして扱う
				sb.WriteString(regexp.QuoteMeta(string(ch)))
				i++
			} else {
				// [...] を変換
				inner := pattern[i+1 : j]
				sb.WriteString("[")
				if len(inner) > 0 && inner[0] == '!' {
					sb.WriteString("^")
					inner = inner[1:]
				}
				sb.WriteString(inner)
				sb.WriteString("]")
				i = j + 1
			}
		default:
			sb.WriteString(regexp.QuoteMeta(string(ch)))
			i++
		}
	}
	sb.WriteString("$")
	return sb.String()
}

// FnMatch は Python fnmatch 互換のマッチング。
// * は / を含め何でもマッチ（DOTALL）。大文字小文字は区別する。
func FnMatch(name, pattern string) bool {
	rx, err := regexp.Compile(translateFnMatch(pattern))
	if err != nil {
		return false
	}
	return rx.MatchString(name)
}

// Glob は root を起点に pattern をマッチするファイルのリストを返す。
// 返すパスは forward-slash の root 相対パス。
// ** はフルセグメントのとき再帰（0個以上のディレクトリにマッチ）。
// ワイルドカードセグメントは先頭ドットの名前にマッチしない。
func Glob(root, pattern string) ([]string, error) {
	parts := strings.Split(pattern, "/")
	var results []string
	err := matchParts(root, root, parts, &results)
	return results, err
}

func matchParts(root, current string, parts []string, results *[]string) error {
	if len(parts) == 0 {
		return nil
	}
	part := parts[0]
	rest := parts[1:]

	if part == "**" {
		if len(rest) == 0 {
			// 末尾セグメントが ** のとき: 現在ディレクトリ配下の全ファイルに再帰的にマッチする。
			// （以前はファイル 0 件を返し、governs グロブが何にもマッチせず越境司法を
			//  沈黙でスキップする偽陰性を生んでいた。PR #4 残課題2 の修正。）
			entries, err := os.ReadDir(current)
			if err != nil {
				return nil // 握りつぶし
			}
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, ".") {
					continue // leading-dot を除外
				}
				full := filepath.Join(current, name)
				if e.IsDir() {
					if err := matchParts(root, full, parts, results); err != nil {
						return err
					}
				} else {
					rel, _ := filepath.Rel(root, full)
					*results = append(*results, filepath.ToSlash(rel))
				}
			}
			return nil
		}

		// 中間 ** は 0 個以上のディレクトリにマッチ
		// まず 0 個（現在のディレクトリで残りの parts をマッチ）
		if err := matchParts(root, current, rest, results); err != nil {
			return err
		}
		// 1 個以上（サブディレクトリを再帰）
		entries, err := os.ReadDir(current)
		if err != nil {
			return nil // 握りつぶし
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue // leading-dot を除外
			}
			if e.IsDir() {
				if err := matchParts(root, filepath.Join(current, name), parts, results); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// 通常セグメント（ワイルドカードなし or */?/[...]）
	isWild := strings.ContainsAny(part, "*?[")
	if len(rest) == 0 {
		// 最終セグメント → ファイルをマッチ
		entries, err := os.ReadDir(current)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			name := e.Name()
			if isWild && strings.HasPrefix(name, ".") {
				continue
			}
			if e.IsDir() {
				continue
			}
			matched := false
			if isWild {
				matched, _ = path.Match(part, name)
			} else {
				matched = (name == part)
			}
			if matched {
				rel, _ := filepath.Rel(root, filepath.Join(current, name))
				*results = append(*results, filepath.ToSlash(rel))
			}
		}
	} else {
		// 中間セグメント → ディレクトリをマッチ
		entries, err := os.ReadDir(current)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() {
				continue
			}
			if isWild && strings.HasPrefix(name, ".") {
				continue
			}
			matched := false
			if isWild {
				matched, _ = path.Match(part, name)
			} else {
				matched = (name == part)
			}
			if matched {
				if err := matchParts(root, filepath.Join(current, name), rest, results); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
