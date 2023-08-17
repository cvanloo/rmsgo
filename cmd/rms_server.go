package main

import (
	"log"
	"net/http"
	"time"

	"github.com/cvanloo/rmsgo"
	//"github.com/cvanloo/rmsgo/mock"
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

		rmsgo.Logger().Info("Request", "method", r.Method, "uri", r.RequestURI, "duration", duration, "status", lrw.Status, "size", lrw.Size)
	})
}

func main() {
	log.Println("starting listener on :8080")

	//mock.Mock(
	//	mock.WithDirectory("/tmp/rms/storage/"),
	//)

	opts, err := rmsgo.Configure(RemoteRoot, StorageRoot)
	if err != nil {
		log.Fatal(err)
	}
	opts.UseErrorHandler(func(err error) {
		log.Fatalf("remote storage: unhandled error: %v", err)
	})
	opts.UseMiddleware(logger)
	opts.AllowAnyReadWrite()

	rmsgo.Register(nil)
	http.ListenAndServe(":8080", nil) // @todo: use TLS
}
