package main

import (
	"log"
	"net/http"

	"github.com/cvanloo/rmsgo.git"
)

const (
	RemoteRoot  = "/storage" // @todo: ensure this is always ends without a /
	StorageRoot = "/tmp/rms/storage/"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	rms := rmsgo.Server{
		Rroot: RemoteRoot,
		Sroot: StorageRoot,
	}
	mux.Handle(RemoteRoot, loggingMiddleware(rms))
	log.Println("starting listener on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
