// Package version はビルド時に埋め込まれるバージョン情報を提供する。
package version

// Version はビルド時に ldflags で上書きされる（例: -X github.com/9uiLe/warrant/internal/version.Version=v1.2.3）。
var Version = "dev"
