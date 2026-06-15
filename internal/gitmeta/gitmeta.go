package gitmeta

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/9uiLe/warrant/internal/projection"
)

// GitMeta はリポジトリルートから Repository/Branch/Commit を取得する。
// エラー・git 不在・非 git ディレクトリでは全て空文字で degrade する。
func GitMeta(root string) projection.ReportMeta {
	run := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	branch := run("rev-parse", "--abbrev-ref", "HEAD")
	commit := run("rev-parse", "--short", "HEAD")
	repo := filepath.Base(run("rev-parse", "--show-toplevel"))
	if repo == "." || repo == "/" {
		repo = ""
	}
	return projection.ReportMeta{
		Repository: repo,
		Branch:     branch,
		Commit:     commit,
	}
}
