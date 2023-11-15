package main

import (
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
	http.HandleFunc("/", handlerNotFound)
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<h1>Hello, World!"))
	})
	log.Fatal(http.ListenAndServe(":8000", nil))
}

type templates struct {
	*template.Template
}

func (t *templates) Render(w io.Writer, name string, data any) error {
	return t.Template.ExecuteTemplate(w , name, data)
}

//var pages = templates{template.Must(template.New("").ParseGlob("templates/*.html"))}
var pages = templates{template.Must(template.New("").ParseFS(templateFiles, "templates/*.html"))}

func handlerNotFound(w http.ResponseWriter, r *http.Request) {
	errorInfo := struct {
		Status, Msg, Desc, URL string
	}{
		Status: http.StatusText(http.StatusNotFound),
		Msg:    "Page " + r.URL.Path + " not found.",
		Desc:   "The requested page does not exist on the server.",
		URL:    "https://www.example.com/error?code=404",
	}
	w.WriteHeader(http.StatusNotFound)
	panicIf(pages.Render(w, "error.html", errorInfo))
}
