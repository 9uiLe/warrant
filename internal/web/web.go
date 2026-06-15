package web

import (
	"embed"
	"html/template"
)

//go:embed report.tmpl.html
var tmplFS embed.FS

// Template はサーバサイドレンダリング用のテンプレート。
var Template = template.Must(template.New("report.tmpl.html").Funcs(template.FuncMap{}).ParseFS(tmplFS, "report.tmpl.html"))
