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
	Option func(*Server)

	Server struct {
		rroot, sroot    string
		allowAllOrigins bool
		allowedOrigins  []string
		allowOrigin     AllowOriginFunc
		middleware      Middleware
		unhandled       ErrorHandlerFunc
		defaultUser     User
		authenticate    AuthenticateFunc
	}

	// @todo: domain name (needed eg., for rfc9457 errors)

	// ErrorHandlerFunc is passed any errors that the remoteStorage server
	// doesn't know how to handle itself.
	ErrorHandlerFunc func(err error)

	// AllowOriginFunc decides whether the origin of request r is allowed
	// (returns true) or forbidden (returns false).
	AllowOriginFunc func(r *http.Request, origin string) bool

	// AuthenticateFunc authenticates a request (usually based on the bearer token).
	// If the request is correctly authenticated a valid User and true are returned.
	// Is the authentication invalid, the returned values are nil and false.
	AuthenticateFunc func(r *http.Request, bearer string) (User, bool)
)

const timeFormat = time.RFC1123

var g *Server

// Configure initializes the remote storage server.
// remoteRoot is the URL path below which remote storage is accessible.
// storageRoot is a folder on the server's file system where remoteStorage
// documents are written to and read from.
// It is recommended to properly configure authentication by using the
// WithAuthentication option.
func Configure(remoteRoot, storageRoot string, opts ...Option) error {
	rroot := filepath.Clean(remoteRoot)
	if rroot == "/" {
		rroot = ""
	}

	sroot := filepath.Clean(storageRoot)
	fi, err := FS.Stat(sroot)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("storage root is not a directory: %s", sroot)
	}

	s := &Server{
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

	for _, opt := range opts {
		opt(s)
	}

	g = s
	return nil
}

// WithErrorHandler configures the error handler to use.
func WithErrorHandler(h ErrorHandlerFunc) Option {
	return func(s *Server) {
		s.unhandled = h
	}
}

// WithMiddleware configures middleware (e.g., for logging) in front of the
// remote storage server.
// The middleware is responsible for passing the request on to the rms server
// using next.ServeHTTP(w, r).
// If this is not done correctly, rmsgo won't be able to handle requests.
func WithMiddleware(m Middleware) Option {
	return func(s *Server) {
		s.middleware = m
	}
}

// WithAllowAnyReadWrite allows even unauthenticated requests to create, read,
// and delete any documents on the server.
// This option has no effect if WithAuthentication is specified.
// Per default, i.e, if neither this nor any other auth related option
// is configured, read-only (GET and HEAD) requests are allowed for the
// unauthenticated user.
func WithAllowAnyReadWrite() Option {
	return func(s *Server) {
		s.defaultUser = UserReadWrite{}
	}
}

// WithAuthentication configures the function to use for authenticating requests.
// The AuthenticateFunc authenticates a request and returns the associated user
// and true, or nil and false (unauthenticated).
// In the latter case, access is forbidden unless it is a read request going to
// a public document (a document whose path starts with "/public/").
// In case of an authenticated user, access rights are determined based on the
// user's Permission method.
func WithAuthentication(a AuthenticateFunc) Option {
	return func(s *Server) {
		s.authenticate = a
	}
}

// WithAllowedOrigins configures a list of allowed origins.
// By default all origins are allowed.
// This option is ignored if WithAllowOrigin is called.
func WithAllowedOrigins(origins []string) Option {
	return func(s *Server) {
		s.allowAllOrigins = false
		s.allowedOrigins = origins
	}
}

// WithAllowOrigin configures the remote storage server to use f to decide
// whether an origin is allowed or not.
// If this option is set up, the list of origins set by WithAllowOrigins is ignored.
func WithAllowOrigin(f AllowOriginFunc) Option {
	return func(s *Server) {
		s.allowAllOrigins = false
		s.allowOrigin = f
	}
}

func WithCondition(cond bool, opt Option) Option {
	return func(s *Server) {
		if cond {
			opt(s)
		}
	}
}

func Options(opts ...Option) Option {
	return func(s *Server) {
		for _, opt := range opts {
			opt(s)
		}
	}
}

func handlePanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch t := r.(type) {
				case error:
					err = fmt.Errorf("recovered panic: %w", t)
				default:
					err = fmt.Errorf("recovered panic: %v", t)
				}
				g.unhandled(err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func stripRoot(next http.Handler) http.Handler {
	return http.StripPrefix(g.rroot /* don't strip slash */, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}))
}

// Register the remote storage server (with middleware if configured via
// Options.UseMiddleware) to the mux using g.Rroot() + '/' as pattern.
// If mux is nil the http.DefaultServeMux is used.
func Register(mux *http.ServeMux) {
	if mux == nil {
		mux = http.DefaultServeMux
	}
	stack := MiddlewareStack(
		handlePanic,
		g.middleware,
		stripRoot,
		handleCORS,
		handleAuthorization,
	)
	mux.Handle(g.rroot+"/", stack(RMSRouter()))
}
