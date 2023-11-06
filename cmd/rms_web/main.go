package main

import (
	"bytes"
	"embed"
	"html/template"
	"io"
	"log"
	"net/http"
)

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

//go:embed templates
var templateFiles embed.FS

type errorData struct {
	Status, Msg, Desc, URL string
}

func main() {
	http.HandleFunc("/", notFoundHandler)
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<h1>Hello, World!"))
	})
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.New("main")
	tmpl.Funcs(map[string]any{
		"CallTemplate": func(name string, data any) (html template.HTML, err error) {
			buf := bytes.NewBuffer([]byte{})
			err = tmpl.ExecuteTemplate(buf, name, data)
			html = template.HTML(buf.String())
			return
		},
	})

	{
		fd := must(templateFiles.Open("templates/main.html"))
		defer panicIf(fd.Close())
		bs := must(io.ReadAll(fd))
		must(tmpl.Parse(string(bs)))
	}

	{
		fd := must(templateFiles.Open("templates/error.html"))
		defer panicIf(fd.Close())
		bs := must(io.ReadAll(fd))
		must(tmpl.New("body").Parse(string(bs)))
	}

	data := map[string]any{
		"Title": "Not Found",
		"Data": errorData{
			Status: http.StatusText(http.StatusNotFound),
			Msg:    "Page not found.",
			Desc:   "The requested page does not exist on the server.",
			URL:    "https://www.example.com/error?code=404",
		},
	}
	w.WriteHeader(http.StatusNotFound)
	panicIf(tmpl.Execute(w, data))
}
