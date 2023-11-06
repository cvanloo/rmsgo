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
	handler := func(w http.ResponseWriter, r *http.Request) {
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
			"Title": "Permission Denied",
			"Data": errorData{
				Status: http.StatusText(http.StatusUnauthorized),
				Msg:    "Missing authorization token",
				Desc:   "You need to authenticate in order to use this service.",
				URL:    "https://www.example.com/error?code=401",
			},
		}
		panicIf(tmpl.Execute(w, data))
	}
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
