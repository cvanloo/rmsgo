package main

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed templates
var embedFiles embed.FS

var Files fs.FS = embedFiles

func main() {
}

func TemplateHandler(fs fs.FS, route string, funcs template.FuncMap) (http.HandlerFunc, error) {
	tmpl, err := template.New(route).Funcs(funcs).ParseFS(fs)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		name := RouteId(r)
		if t := tmpl.Lookup(name); t != nil {
			var tdata struct {
				Title string
			}
			tdata.Title = "Test Title"
			var buf bytes.Buffer
			err := t.Execute(&buf, tdata)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}, nil
}

func RouteId(r *http.Request) string {
	//name := strings.TrimSuffix(filepath.Base(route), filepath.Ext(route))
	panic("not implemented")
}
