package rmsgo

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	. "github.com/cvanloo/rmsgo.git/mock"
)

type (
	// Any errors that the remoteStorage server doesn't know how to handle itself
	// are passed to the ErrorHandlerFunc.
	ErrorHandlerFunc func(err error)

	MiddlewareFunc func(next http.Handler) http.Handler

	// AllowOriginFunc decides whether an origin is allowed (returns true) or
	// forbidden (returns false).
	AllowOriginFunc func(r *http.Request, origin string) bool

	// AuthenticateFunc authenticates a request (usually with the bearer token).
	// If the request is correctly authenticated, a User and true must be
	// returned.
	AuthenticateFunc func(r *http.Request, bearer string) (User, bool)
)

const rmsTimeFormat = time.RFC1123

var (
	rroot, sroot string

	allowAllOrigins bool = true
	allowedOrigins  []string

	allowOrigin AllowOriginFunc = func(r *http.Request, origin string) bool {
		for _, o := range allowedOrigins {
			if o == origin {
				return true
			}
		}
		return false
	}

	middleware MiddlewareFunc = func(next http.Handler) http.Handler {
		return next
	}

	unhandled ErrorHandlerFunc = func(err error) {
		log.Printf("rmsgo: unhandled error: %v\n", err)
	}

	authenticate AuthenticateFunc = func(r *http.Request, bearer string) (User, bool) {
		return anyUser{}, true
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

func UseErrorHandler(h ErrorHandlerFunc) {
	unhandled = h
}

func UseMiddleware(m MiddlewareFunc) {
	middleware = m
}

func UseAuthentication(a AuthenticateFunc) {
	authenticate = a
}

// AllowOrigins configures a list of allowed origins.
// By default, i.e if AllowOrigins is never called, all origins are allowed.
func AllowOrigins(origins []string) {
	allowAllOrigins = false
	allowedOrigins = origins
}

// UseAllowOrigin configures the remote storage server to use f to decide
// whether an origin is allowed or not.
// If this option is set up, the list of origins set by AllowOrigins is ignored.
func UseAllowOrigin(f AllowOriginFunc) {
	allowAllOrigins = false
	allowOrigin = f
}

func handleRMS() http.Handler {
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
	mux.Handle(rroot+"/", middleware(handleCORS(handleAuthorization(handleRMS()))))
}
