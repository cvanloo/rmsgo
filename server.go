package rmsgo

import (
	"fmt"
	. "github.com/cvanloo/rmsgo.git/mock"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type (
	// Any errors that the remoteStorage server doesn't know how to handle itself
	// are passed to the ErrorHandler.
	ErrorHandler func(err error)

	Middleware func(next http.Handler) http.Handler

	// AllowOrigin decides whether an origin is allowed (returns true) or
	// forbidden (returns false).
	AllowOrigin func(r *http.Request, origin string) bool
)

const rmsTimeFormat = time.RFC1123

var (
	rroot, sroot string

	allowAllOrigins bool = true
	allowedOrigins  []string

	allowOriginFunc AllowOrigin = func(r *http.Request, origin string) bool {
		for _, o := range allowedOrigins {
			if o == origin {
				return true
			}
		}
		return false
	}

	middleware Middleware = func(next http.Handler) http.Handler {
		return next
	}

	unhandled ErrorHandler = func(err error) {
		log.Printf("rmsgo: unhandled error: %v\n", err)
	}
)

// Setup initializes the remote storage server.
// remoteRoot is the URL path below which remote storage is accessible, and
// storageRoot is a folder on the server's file system where remoteStorage
// documents are written to and read from.
func Setup(remoteRoot, storageRoot string) error {
	rroot = filepath.Clean(remoteRoot)
	sroot = filepath.Clean(storageRoot)

	fi, err := FS.Stat(sroot)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("storage root is not a directory: %s", sroot)
	}
	return nil
}

// Rroot specifies the URL path at which remoteStorage is rooted.
// E.g., if Rroot is "/storage" then a document "/Picture/Kittens.png" can
// be accessed using the URL "https://example.com/storage/Picture/Kittens.png".
// Rroot does not have a trailing slash.
func Rroot() string {
	return rroot
}

// Sroot is a path specifying the location on the server's file system where
// all of remoteStorage's files are stored. Sroot does not have a trailing
// slash.
func Sroot() string {
	return sroot
}

func UseErrorHandler(h ErrorHandler) {
	unhandled = h
}

func UseMiddleware(m Middleware) {
	middleware = m
}

// AllowOrigins configures a list of allowed origins.
// By default, i.e if AllowOrigins is never called, all origins are allowed.
func AllowOrigins(origins []string) {
	allowAllOrigins = false
	allowedOrigins = origins
}

// AllowOriginFunc configures the remote storage server to use f to decide
// whether an origin is allowed or not.
// If this option is set up, the list of origins set by AllowOrigins is ignored.
func AllowOriginFunc(f AllowOrigin) {
	allowAllOrigins = false
	allowOriginFunc = f
}

// Handler returns an http.Handler which may be passed directly to a
// http.ServeMux.Handle or http.ListenAndServe/TLS.
// Usually you would want to use Register instead.
// If using Handler directly, make sure that it is accessible at Rroot+'/' or '/'.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := serve(w, r)
		if err != nil {
			unhandled(err)
		}
	})
}

// Register the remote storage server (with middleware if configured) to the
// mux using Rroot + '/' as pattern.
func Register(mux *http.ServeMux) {
	mux.Handle(rroot+"/", middleware(handleCORS(Handler())))
}
