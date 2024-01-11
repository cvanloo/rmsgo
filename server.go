// Package rmsgo implements the remoteStorage protocol.
//
// Remote root refers to the URL path below which remote storage resources can be accessed.
// For example, with remote root set to "/storage", a document named
// "/Pictures/Kitten.avif" can be found via the URL
// "https://example.com/storage/Pictures/Kitten.avif".
//
// Storage root refers to the location on disk where the actual "Kitten.avif"
// file is stored (for example "/var/storage").
//
// Use the Configure function and the methods on the returned Options struct
// to setup the server.
//
//	opts, err := rmsgo.Configure(RemoteRoot, StorageRoot)
//	if err != nil {
//		log.Fatal(err)
//	}
//	opts.UseXXX(...)
package rmsgo

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	. "github.com/cvanloo/rmsgo/mock"
)

type (
	// Options' methods configure the remote storage server.
	Options struct {
		rroot, sroot    string
		allowAllOrigins bool
		allowedOrigins  []string
		allowOrigin     AllowOriginFunc
		middleware      MiddlewareFunc
		unhandled       ErrorHandlerFunc
		defaultUser     User
		authenticate    AuthenticateFunc
	}

	// ErrorHandlerFunc is passed any errors that the remoteStorage server
	// doesn't know how to handle itself.
	ErrorHandlerFunc func(err error)

	// A MiddlewareFunc is inserted into a chain of other http.Handler.
	// This way, different parts of handling a request can be separated each
	// into its own handler.
	// The handler inserted here will receive the request first, before any
	// remote storage handler are executed.
	MiddlewareFunc func(next http.Handler) http.Handler

	// AllowOriginFunc decides whether the origin of request r is allowed
	// (returns true) or forbidden (returns false).
	AllowOriginFunc func(r *http.Request, origin string) bool

	// AuthenticateFunc authenticates a request (usually based on the bearer token).
	// If the request is correctly authenticated a valid User and true are returned.
	// Is the authentication invalid, the returned values are nil and false.
	AuthenticateFunc func(r *http.Request, bearer string) (User, bool)
)

const timeFormat = time.RFC1123

// Global g holds essential configuration values.
var g *Options

// Configure initializes the remote storage server with the default configuration.
// remoteRoot is the URL path below which remote storage is accessible, and
// storageRoot is a folder on the server's file system where remoteStorage
// documents are written to and read from.
// A pointer to the Options object is returned and allows for further
// configuration beyond the default settings.
func Configure(remoteRoot, storageRoot string) (*Options, error) {
	rroot := filepath.Clean(remoteRoot)
	if rroot == "/" {
		rroot = ""
	}
	sroot := filepath.Clean(storageRoot)
	fi, err := FS.Stat(sroot)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("storage root is not a directory: %s", sroot)
	}

	g = &Options{
		rroot:           rroot,
		sroot:           sroot,
		allowAllOrigins: true,
		allowedOrigins:  []string{},
		allowOrigin: func(r *http.Request, origin string) bool {
			for _, o := range g.allowedOrigins {
				if o == origin {
					return true
				}
			}
			return false
		},
		middleware: func(next http.Handler) http.Handler {
			return next
		},
		unhandled: func(err error) {
			log.Printf("rmsgo: unhandled error: %v\n", err)
		},
		defaultUser: UserReadOnly{},
		authenticate: func(r *http.Request, bearer string) (User, bool) {
			return g.defaultUser, true
		},
	}
	return g, nil
}

// Rroot specifies the URL path at which remoteStorage is rooted.
// E.g., if Rroot is "/storage" then a document "/Picture/Kittens.png" can
// be accessed using the URL "https://example.com/storage/Picture/Kittens.png".
// Rroot does not have a trailing slash.
func (o *Options) Rroot() string {
	return o.rroot
}

// Sroot is a path specifying the location on the server's file system where
// all of remoteStorage's files are stored. Sroot does not have a trailing
// slash.
func (o *Options) Sroot() string {
	return o.sroot
}

// UseErrorHandler configures the error handler to use.
func (o *Options) UseErrorHandler(h ErrorHandlerFunc) {
	o.unhandled = h
}

// UseMiddleware configures middleware (e.g., for logging) in front of the
// remote storage server.
// The middleware is responsible for passing the request on to the rms server
// using next.ServeHTTP(w, r).
func (o *Options) UseMiddleware(m MiddlewareFunc) {
	o.middleware = m
}

// AllowAnyReadWrite allows even unauthenticated requests to create, read, and
// delete any documents on the server.
// This option has no effect if UseAuthentication is used.
// Per default, i.e if neither this nor any other auth related option
// is configured, read-only (GET and HEAD) requests are allowed for the
// unauthenticated user.
func (o *Options) AllowAnyReadWrite() {
	o.defaultUser = UserReadWrite{}
}

// UseAuthentication configures the function to use for authenticating requests.
// The AuthenticateFunc authenticates a request and returns the associated user,
// or nil (unauthenticated).
// In the latter case, access is forbidden unless it is a read request going to
// a public document (a document whose path starts with "/public/").
// In case of an authenticated user, access rights are determined based on the
// user's Permission method.
func (o *Options) UseAuthentication(a AuthenticateFunc) {
	o.authenticate = a
}

// UseAllowedOrigins configures a list of allowed origins.
// By default all origins are allowed.
func (o *Options) UseAllowedOrigins(origins []string) {
	o.allowAllOrigins = false
	o.allowedOrigins = origins
}

// UseAllowOrigin configures the remote storage server to use f to decide
// whether an origin is allowed or not.
// If this option is set up, the list of origins set by AllowOrigins is ignored.
func (o *Options) UseAllowOrigin(f AllowOriginFunc) {
	o.allowAllOrigins = false
	o.allowOrigin = f
}

// Register the remote storage server (with middleware if configured) to the
// mux using g.Rroot + '/' as pattern.
// If mux is nil the http.DefaultServeMux is used.
func Register(mux *http.ServeMux) {
	if mux == nil {
		mux = http.DefaultServeMux
	}
	mux.Handle(g.rroot+"/", g.middleware(handleCORS(handleAuthorization(handleRMS()))))
}
