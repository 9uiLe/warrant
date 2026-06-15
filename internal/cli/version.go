package cli

import (
	"fmt"

	"github.com/9uiLe/warrant/internal/version"
)

// runVersion はバージョン情報を標準出力に出力する。
func runVersion(_ []string) int {
	fmt.Printf("warrant %s\n", version.Version)
	return 0
}
