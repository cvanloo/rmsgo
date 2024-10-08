package rmsgo_test

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/cvanloo/rmsgo"
)

// ExampleRegister demonstrates how to register the remote storage endpoints
// to a serve mux.
func ExampleRegister() {
	const (
		remoteRoot  = "/storage/"
		storageRoot = "/var/rms/storage/"
	)

	// [!] TODO: Use a real file
	persistFile := &bytes.Buffer{}

	// Restore server state at startup
	err := rmsgo.Load(persistFile)
	if err != nil {
		log.Fatal(err)
	}

	err = rmsgo.Configure(remoteRoot, storageRoot)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	// TODO: Other mux.Handle setup

	rmsgo.Register(mux)
	http.ListenAndServe(":8080", mux) // [!] TODO: Use TLS

	// Persist server state at shutdown
	err = rmsgo.Persist(persistFile)
	if err != nil {
		log.Fatal(err)
	}
}

// Alternatively, the endpoints can be registered to the http.DefaultServeMux
// by passing nil to Register.
func ExampleRegister_usingDefaultServeMux() {
	const (
		remoteRoot  = "/storage/"
		storageRoot = "/var/rms/storage/"
	)

	// [!] TODO: Use a real file
	persistFile := &bytes.Buffer{}

	// Restore server state at startup
	err := rmsgo.Load(persistFile)
	if err != nil {
		log.Fatal(err)
	}

	err = rmsgo.Configure(remoteRoot, storageRoot)
	if err != nil {
		log.Fatal(err)
	}

	rmsgo.Register(nil)
	http.ListenAndServe(":8080", nil) // [!] TODO: Use TLS

	// Persist server state at shutdown
	err = rmsgo.Persist(persistFile)
	if err != nil {
		log.Fatal(err)
	}
}

// Configure accepts a variable number of Option arguments.
// These can be used to overwrite the default configuration, e.g., to configure
// CORS, setup authentication, additional middleware, and more.
func ExampleOptions() {
	const (
		remoteRoot  = "/storage/"
		storageRoot = "/var/rms/storage/"
	)

	err := rmsgo.Configure(remoteRoot, storageRoot,

		rmsgo.WithErrorHandler(func(err error) {
			log.Panicf("remote storage: unhandled error: %v", err)
		}),

		rmsgo.WithMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				lrw := rmsgo.NewLoggingResponseWriter(w)

				// [!] Pass request on to remote storage server
				next.ServeHTTP(lrw, r)

				duration := time.Since(start)

				// maybe use an actual library for structured logging
				log.Printf("%v", map[string]any{
					"method":   r.Method,
					"uri":      r.RequestURI,
					"duration": duration,
					"status":   lrw.Status,
					"size":     lrw.Size,
				})
			})
		}),

		rmsgo.WithAuthentication(func(r *http.Request, bearer string) (rmsgo.User, bool) {
			// [!] TODO: Your authentication logic here...
			//       Return one of your own users.
			return rmsgo.UserReadWrite{}, true
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	rmsgo.Register(nil)
	http.ListenAndServe(":8080", nil) // [!] TODO: Use TLS
}
