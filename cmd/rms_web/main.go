package main

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
)

var (
	//go:embed templates
	embedFiles embed.FS

	htmlFiles fs.FS = embedFiles
)

func main() {
	var data struct {
		Title                  string
		Status, Msg, Desc, URL string
	}
	data.Title = "Hello, World!"
	data.Status = http.StatusText(http.StatusBadRequest)
	data.Msg = "Permission Denied"
	data.Desc = "Missing authorization token."
	data.URL = "https://www.example.com/error?code=401"

	tmpl, err := template.New("").ParseFS(htmlFiles, "templates/*")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err = tmpl.ExecuteTemplate(w, "main", data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print(err)
		}
	})
	log.Fatal(http.ListenAndServe(":8888", nil))
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
