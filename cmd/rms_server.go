package main

import (
	"log"
	"net/http"
	"time"

	"github.com/cvanloo/rmsgo.git"
	"github.com/cvanloo/rmsgo.git/mock"
)

const (
	RemoteRoot  = "/storage/"
	StorageRoot = "/tmp/rms/storage/"
)

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := rmsgo.NewLoggingResponseWriter(w)

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		log.Printf("%v", map[string]any{
			"method":   r.Method,
			"uri":      r.RequestURI,
			"duration": duration,
			"status":   lrw.Status,
			"size":     lrw.Size,
		})
	})
}

func main() {
	log.Println("starting listener on :8080")

	mock.Mock(
		mock.WithDirectory("/tmp/rms/storage/"),
	)

	err := rmsgo.Setup(RemoteRoot, StorageRoot)
	if err != nil {
		log.Fatal(err)
	}
	rmsgo.UseErrorHandler(func(err error) {
		log.Fatalf("remote storage: unhandled error: %v", err)
	})
	rmsgo.UseMiddleware(logger)

	mux := http.NewServeMux()
	rmsgo.Register(mux)
	http.ListenAndServe(":8080", mux) // @todo: use TLS
}
