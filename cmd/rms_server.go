package main

import (
	"log"
	"net/http"

	"github.com/cvanloo/rmsgo.git"
)

// @todo: include those values in a server struct{}
const (
	RemoteRoot  = "/storage/"
	StorageRoot = "/tmp/rms/storage/"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// @todo: we could create a server struct{} and turn the `rmsgo.Serve` function
// into a `ServeHTTP` method.
//
//	type http.Handler interface {
//	    ServeHTTP(ResponseWriter, *Request)
//	}
func serveRms() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := rmsgo.Serve(w, r)
		if err != nil {
			log.Printf("rms-server: %s", err)
		}
	})
}

func main() {
	mux := http.NewServeMux()
	mux.Handle(RemoteRoot, loggingMiddleware(serveRms()))
	log.Println("Listening on :8080")
	http.ListenAndServe(":8080", mux)
}
