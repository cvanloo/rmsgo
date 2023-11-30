package main

import (
	"log"
	"net/http"
	"html/template"
)

func init() {
	pages = Template{template.Must(template.New("").ParseFS(templateFiles, "templates/*.html"))}
	pages.Funcs(map[string]any{
		"CallTemplate": func(name string, data any) (html template.HTML, err error) {
			buf := bytes.NewBuffer([]byte{})
			err = pages.ExecuteTemplate(buf, name, data)
			html = template.HTML(buf.String())
			return
		},
	})
}

func panicIf(err error) {
	if err != nil {
		log.Fatalf("panicIf: %w", err)
	}
}

type Template struct {
	*template.Template
}

func (t *Template) Render(w io.Writer, name string, data any) error {
	// @todo: where do we go from here?
	return t.Template.ExecuteTemplate(w, name, data)
}

var pages Template

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", routeNotFound)
	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	mux.HandleFunc("/hello", routeHello)
	log.Fatal(http.ListenAndServe(":8000", mux))
}

func routeNotFound(w http.ResponseWriter, r *http.Request) {
	errorInfo := struct{
		Status, Msg, Desc, URL string
	}{
		Status: http.StatusText(http.StatusNotFound),
		Msg: "Page " + r.URL.Path + " not found.",
		Desc: "The requested page does not exist on the server.",
		URL: "https://www.example.com/error?code=404",
	}
	w.WriteHeader(http.StatusNotFound)
	panicIf(pages.Render(w, "error.html", errorInfo))
}

func routeHello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<h1>Hello, World!"))
}
