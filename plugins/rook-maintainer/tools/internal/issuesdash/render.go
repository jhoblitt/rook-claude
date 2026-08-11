package issuesdash

import (
	"embed"
	"html/template"
	"io"
)

//go:embed page.html.tmpl cells.html.tmpl
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "*.tmpl"))

// Render writes the whole dashboard page. The page carries no trailing
// newline, so republishing it byte-compares cleanly against earlier sweeps.
func Render(w io.Writer, p Page) error {
	return tmpl.ExecuteTemplate(w, "page", p)
}
