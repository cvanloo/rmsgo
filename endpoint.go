package rmsgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cvanloo/rmsgo.git/isdelve"
)

func init() {
	if isdelve.Enabled {
		mfs = CreateMockFS().CreateDirectories("/tmp/rms/storage/")
	}
}

var mfs fileSystem = &osFileSystem{}

// Server configuration for the remoteStorage endpoint.
// Server implements http.Handler and can therefore be passed directly to a
// http.ServeMux.Handle or http.ListenAndServe(TLS)
type Server struct {
	rroot, sroot string
}

// New constructs a remoteStorage server configuration.
// remoteRoot is the root folder of the storage tree (used in the URL),
// storageRoot is a folder on the server's file system where remoteStorage
// documents are written to and read from.
func New(remoteRoot, storageRoot string) (Server, error) {
	rroot := filepath.Clean(remoteRoot)
	sroot := filepath.Clean(storageRoot)
	fi, err := mfs.Stat(sroot)
	if err != nil || !fi.IsDir() {
		return Server{rroot, sroot}, fmt.Errorf("storage root does not exist or is not a directory: %w", err)
	}
	return Server{rroot, sroot}, nil
}

// Rroot specifies the URL path at which remoteStorage is rooted.
// E.g. if Rroot is "/storage" then a document "/Picture/Kittens.png" can
// be accessed using the URL "example.com/storage/Picture/Kittens.png".
// Rroot does not have a trailing slash.
func (s Server) Rroot() string {
	return s.rroot
}

// Sroot is a path specifying the location on the server's file system
// where all of remoteStorage's files are stored.
// Sroot does not have a trailing slash.
func (s Server) Sroot() string {
	return s.sroot
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	isFolder := false
	if path[len(path)-1] == '/' {
		isFolder = true
	}

	var err error
	if isFolder {
		switch r.Method {
		case http.MethodHead:
			fallthrough
		case http.MethodGet:
			err = s.GetFolder(w, r)
		default:
			err = writeError(w, ErrMethodNotAllowed)
		}
	} else {
		switch r.Method {
		case http.MethodHead:
			fallthrough
		case http.MethodGet:
			err = s.GetDocument(w, r)
		case http.MethodPut:
			err = s.PutDocument(w, r)
		case http.MethodDelete:
			err = s.DeleteDocument(w, r)
		default:
			err = writeError(w, ErrMethodNotAllowed)
		}
	}

	if err != nil {
		// @todo: allow user to configure a logging function
		log.Printf("rms-server: %s", err)
	}
}

const userKey = "AUTHENTICATED_USER"

// @todo: interceptor for authentication/authorization
func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// @todo: logic to authenticate user
		user, err := "???", ErrUnauthorized
		if err != nil {
			writeError(w, err) // @fixme: error ignored
			return
		}

		nctx := context.WithValue(r.Context(), userKey, user)
		nr := r.WithContext(nctx)
		next.ServeHTTP(w, nr)
	})
}

// @todo: OPTIONS/cors
// @todo: https://datatracker.ietf.org/doc/html/draft-dejong-remotestorage-21#section-6
// keep multiple versions of files around, option to restore deleted files
// > A provider MAY offer version rollback functionality to its users,
// > but this specification does not define the interface for that.

type ldjson = map[string]any

const rmsTimeFormat = time.RFC1123

func (s Server) GetFolder(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, s.rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return writeError(w, err)
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err)
	}

	if ifMatch := r.Header["If-Non-Match"]; len(ifMatch) > 0 {
		for _, rev := range ifMatch {
			rev, err := ParseETag(rev)
			if err != nil {
				return writeError(w, err) // @todo: ErrBadRequest
			}
			if rev.Equal(etag) {
				return writeError(w, ErrNotModified)
			}
		}
	}

	items := ldjson{}
	for _, child := range n.children {
		desc := ldjson{}
		etag, err := child.Version()
		if err != nil {
			return writeError(w, err)
		}
		desc["ETag"] = etag.String()
		if !child.IsFolder {
			desc["Content-Type"] = child.Mime
			desc["Content-Length"] = child.Length
			desc["Last-Modified"] = child.LastMod.Format(rmsTimeFormat)
		}
		items[child.Name] = desc
	}

	desc := ldjson{
		"@context": "http://remotestorage.io/spec/folder-description",
		"items":    items,
	}

	hs := w.Header()
	hs.Set("Content-Type", "application/ld+json")
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", etag.String())
	return json.NewEncoder(w).Encode(desc)
}

func (s Server) GetDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, s.rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return writeError(w, err)
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err)
	}

	if ifMatch := r.Header["If-Non-Match"]; len(ifMatch) > 0 {
		for _, rev := range ifMatch {
			rev, err := ParseETag(rev)
			if err != nil {
				return writeError(w, err) // @todo: ErrBadRequest
			}
			if rev.Equal(etag) {
				return writeError(w, ErrNotModified)
			}
		}
	}

	fd, err := mfs.Open(n.Sname)
	if err != nil {
		return writeError(w, err)
	}

	hs := w.Header()
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", etag.String())
	hs.Set("Content-Type", n.Mime)
	_, err = io.Copy(w, fd)
	return err
}

func (s Server) PutDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, s.rroot)

	n, err := Retrieve(rpath)
	notFound := errors.Is(err, ErrNotFound)
	if err != nil && !notFound {
		return writeError(w, err)
	}

	if ifNonMatch := r.Header.Get("If-Non-Match"); ifNonMatch == "*" {
		if !notFound {
			return writeError(w, ErrPreconditionFailed) // @todo(#desc_error): the document already exists
		}
	}

	if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
		rev, err := ParseETag(ifMatch)
		if err != nil {
			return writeError(w, err) // @todo: ErrBadRequest when this fails
		}
		etag, err := n.Version()
		if err != nil {
			return writeError(w, err)
		}
		if !etag.Equal(rev) {
			return writeError(w, ErrPreconditionFailed) // @todo(#desc_error): version mismatch
		}
	}

	contentType := r.Header.Get("Content-Type")

	// @todo: merge Create and Update into one function?
	if notFound {
		sname, fsize, mime, err := WriteFile(s, "", r.Body)
		if err != nil {
			return err
		}
		if contentType == "" {
			contentType = mime
		}
		n, err = AddDocument(rpath, sname, fsize, contentType)
		if err != nil {
			return writeError(w, err)
		}
	} else {
		_, fsize, mime, err := WriteFile(s, n.Sname, r.Body)
		if err != nil {
			return err
		}
		if contentType == "" {
			contentType = mime
		}
		n, err = UpdateDocument(rpath, fsize, contentType)
		if err != nil {
			return writeError(w, err)
		}
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err)
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	w.WriteHeader(http.StatusCreated)
	return nil
}

func (s Server) DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, s.rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return writeError(w, err)
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err)
	}

	if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
		rev, err := ParseETag(ifMatch)
		if err != nil {
			return writeError(w, err) // @todo: ErrBadRequest when this fails
		}
		if !etag.Equal(rev) {
			return writeError(w, ErrPreconditionFailed) // @todo(#desc_error): version mismatch
		}
	}

	n, err = RemoveDocument(rpath)
	if err != nil {
		return writeError(w, err)
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	return nil
}

func writeError(w http.ResponseWriter, err error) error {
	status, ok := StatusCodes[err]
	if !ok {
		err = ErrServerError
		status = StatusCodes[ErrServerError]
	}

	data := map[string]any{
		"error": err.Error(),
		// @todo: "message": err.Message()?
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
