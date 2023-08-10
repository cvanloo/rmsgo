package main

import (
	"log"
	"net/http"

	"github.com/cvanloo/rmsgo.git"
	"github.com/cvanloo/rmsgo.git/mock"
)

const (
	RemoteRoot  = "/storage/"
	StorageRoot = "/tmp/rms/storage/"
)

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s [%s]", r.Method, r.URL.Path, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	mfs := mock.Mock()
	mfs.CreateDirectories("/tmp/rms/storage/")

	err := rmsgo.Configure(RemoteRoot, StorageRoot, nil)
	if err != nil {
		log.Fatal(err)
	}

	rms := rmsgo.ServeMux{}

	// Option 1:
	log.Fatal(http.ListenAndServe(":8080", logger(rms)))

	// Option 2:
	mux := http.NewServeMux()
	mux.Handle(RemoteRoot, logger(rms))
	log.Println("starting listener on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
