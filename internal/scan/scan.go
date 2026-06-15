package scan

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/9uiLe/warrant/internal/glob"
)

// TagIndex は id → そのタグを含むファイルの相対パス集合（forward-slash）
type TagIndex map[string]map[string]struct{}

// Run は tag と id_pattern で test_globs を走査し TagIndex を返す
func Run(root string, testGlobs []string, tag, idPattern string) (TagIndex, error) {
	// タグ正規表現: regexp.QuoteMeta(tag) + \s+(id_pattern)
	tagRx := regexp.MustCompile(regexp.QuoteMeta(tag) + `\s+(` + idPattern + `)`)

	index := make(TagIndex)
	seen := make(map[string]struct{})

	for _, pattern := range testGlobs {
		files, err := glob.Glob(root, pattern)
		if err != nil {
			continue
		}
		for _, relPath := range files {
			if _, dup := seen[relPath]; dup {
				continue
			}
			seen[relPath] = struct{}{}

			absPath := filepath.Join(root, filepath.FromSlash(relPath))
			data, err := os.ReadFile(absPath)
			if err != nil {
				continue // skip unreadable
			}
			matches := tagRx.FindAllSubmatch(data, -1)
			for _, m := range matches {
				id := string(m[1])
				if index[id] == nil {
					index[id] = make(map[string]struct{})
				}
				index[id][relPath] = struct{}{}
			}
		}
	}
	return index, nil
}
