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
)

const rmsTimeFormat = time.RFC1123

var (
	rroot, sroot string

	middleware Middleware = func(next http.Handler) http.Handler {
		return next
	}

	unhandled ErrorHandler = func(err error) {
		log.Printf("rmsgo: unhandled error: %v\n", err)
	}
)

// Setup creates a remoteStorage server configuration.
// remoteRoot is the root folder of the storage tree (used in the URL),
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
// E.g. if Rroot is "/storage" then a document "/Picture/Kittens.png" can
// be accessed using the URL "example.com/storage/Picture/Kittens.png".
// Rroot does not have a trailing slash.
func Rroot() string {
	return rroot
}

// Sroot is a path specifying the location on the server's file system
// where all of remoteStorage's files are stored.
// Sroot does not have a trailing slash.
func Sroot() string {
	return sroot
}

func UseErrorHandler(h ErrorHandler) {
	unhandled = h
}

func UseMiddleware(m Middleware) {
	middleware = m
}

// Handler returns an http.Handler which may be passed directly to a
// http.ServeMux.Handle or http.ListenAndServe/TLS.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := serve(w, r)
		if err != nil {
			unhandled(err)
		}
	})
}

func Register(mux *http.ServeMux) {
	mux.Handle(rroot+"/", middleware(Handler()))
}
