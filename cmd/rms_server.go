package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/cvanloo/rmsgo.git"
)

const (
	// @todo: ensure this is always handled correctly with(out) a at the end /
	RemoteRoot  = "/storage/"
	StorageRoot = "/tmp/rms/storage/"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s [%s]", r.Method, r.URL.Path, strings.TrimPrefix(r.URL.Path, RemoteRoot[:len(RemoteRoot)-1]))
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	rms, err := rmsgo.New(RemoteRoot, StorageRoot)
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle(RemoteRoot, loggingMiddleware(rms))
	log.Println("starting listener on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
