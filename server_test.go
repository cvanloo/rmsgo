package rmsgo_test

import (
	"log"
	"net/http"
	"time"

	"github.com/cvanloo/rmsgo.git"
)

func ExampleRegister() {
	const (
		remoteRoot  = "/storage/"
		storageRoot = "/var/rms/storage/"
	)

	err := rmsgo.Setup(remoteRoot, storageRoot)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	rmsgo.Register(mux)
	http.ListenAndServe(":8080", mux) // [!] TODO: Use TLS
}

func ExampleSetup() {
	const (
		remoteRoot  = "/storage/"
		storageRoot = "/var/rms/storage/"
	)

	err := rmsgo.Setup(remoteRoot, storageRoot)
	if err != nil {
		log.Fatal(err)
	}
	rmsgo.UseErrorHandler(func(err error) {
		log.Panicf("remote storage: unhandled error: %v", err)
	})
	rmsgo.UseMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := rmsgo.NewLoggingResponseWriter(w)

			// [!] pass request on to remote storage server
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
	})

	mux := http.NewServeMux()
	rmsgo.Register(mux)
	http.ListenAndServe(":8080", mux) // [!] TODO: Use TLS
}
