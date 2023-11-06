package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"mime"
	"net/http"
)

var (
	//go:embed templates
	templateFiles embed.FS
	htmlFiles fs.FS = templateFiles
)

type errorData struct{
	Title, Status, Msg, Desc, URL string
}

func init() {
	_ = mime.AddExtensionType(".js", "text/javascript")
}

func main() {
	tmpl := template.Must(template.New("").ParseFS(htmlFiles, "templates/*"))
	handler := func(w http.ResponseWriter, r *http.Request) {
		data := errorData{
			Title:  "You need to authenticate in order to use this service.",
			Status: http.StatusText(http.StatusUnauthorized),
			Msg:    "Permission Denied",
			Desc:   "Missing authorization token",
			URL:    "https://www.example.com/error?code=401",
		}
		err := tmpl.ExecuteTemplate(w, "main", data)
		if err != nil {
			panic(err)
		}
	}
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
