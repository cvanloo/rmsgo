package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"
)

func init() {
	pages = Template{template.New("")}
	pages.Funcs(map[string]any{
		"CallTemplate": func(name string, data any) (html template.HTML, err error) {
			buf := bytes.NewBuffer([]byte{})
			err = pages.ExecuteTemplate(buf, name, data)
			html = template.HTML(buf.String())
			return
		},
	})
	template.Must(pages.ParseFS(templateFiles, "templates/*.html"))
}

func panicIf(err error) {
	if err != nil {
		log.Fatalf("panicIf: %v", err)
	}
}

type File struct {
	IsFolder    bool
	Name        string
	Version     ETag
	Mime        string
	Length      FileSize
	LastModText string
}

type FileSize uint64

func (s FileSize) String() string {
	if s < (2 << 9) {
		return fmt.Sprintf("%d B", s)
	} else if s < (2 << 19) {
		return fmt.Sprintf("%d KB", s/(2<<9))
	} else if s < (2 << 29) {
		return fmt.Sprintf("%d MB", s/(2<<19))
	} else if s < (2 << 39) {
		return fmt.Sprintf("%d GB", s/(2<<29))
	} else if s < (2 << 49) {
		return fmt.Sprintf("%d TB", s/(2<<39))
	} else if s < (2 << 59) {
		return fmt.Sprintf("%d PB", s/(2<<49))
	} else {
		// that ought to be future proof enough
		return fmt.Sprintf("%d EB", s/(2<<59))
	}
}

type Template struct {
	*template.Template
}

func (t *Template) Render(w io.Writer, name string, data any) error {
	return t.Template.ExecuteTemplate(w, name, data)
}

type ETag string

func (e ETag) String() string {
	return fmt.Sprintf("ETag: %s", string(e))
}

//go:embed templates
var templateFiles embed.FS

var pages Template

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", routeNotFound)
	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	mux.HandleFunc("/storage/", routeStorage)
	log.Fatal(http.ListenAndServe(":8000", mux))
}

func routeNotFound(w http.ResponseWriter, r *http.Request) {
	errorInfo := struct {
		Status, Msg, Desc, URL string
	}{
		Status: http.StatusText(http.StatusNotFound),
		Msg:    "Page " + r.URL.Path + " not found.",
		Desc:   "The requested page does not exist on the server.",
		URL:    "https://www.example.com/error?code=404",
	}
	renderData := struct {
		Title, Body string
		Data        any
	}{
		Title: "Error",
		Body:  "error.html",
		Data:  errorInfo,
	}
	w.WriteHeader(http.StatusNotFound)
	panicIf(pages.Render(w, "main.html", renderData))
}

func routeStorage(w http.ResponseWriter, r *http.Request) {
	dateText := time.Now().Format(time.RFC1123)
	storageData := struct {
		Files []File
	}{
		Files: []File{
			{
				IsFolder: true,
				Name:     "Documents",
				Version:  "VERSIONSTRING1",
			},
			{
				IsFolder:    false,
				Name:        "Kittens.avif",
				Version:     "VERSIONSTRING1.1",
				Mime:        "image/avif",
				Length:      128,
				LastModText: dateText,
			},
			{
				IsFolder:    false,
				Name:        "ProofSantaIsntReal.png",
				Version:     "VERSIONSTRING5",
				Mime:        "image/png",
				Length:      1024,
				LastModText: dateText,
			},
			{
				IsFolder:    false,
				Name:        "SecretStockOptions.csv",
				Version:     "VERSIONSTRING6",
				Mime:        "text/csv",
				Length:      65535,
				LastModText: dateText,
			},
		},
	}
	renderData := struct {
		Title, Body string
		Data        any
	}{
		Title: "Files",
		Body:  "files.html",
		Data:  storageData,
	}
	panicIf(pages.Render(w, "main.html", renderData))
}
