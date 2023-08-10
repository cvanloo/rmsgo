package rmsgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cvanloo/rmsgo.git/isdelve"
	. "github.com/cvanloo/rmsgo.git/mock"
)

func init() {
	if !isdelve.Enabled {
		FS = &RealFileSystem{}
	}
}

// Server configuration for the remoteStorage endpoint.
// Server implements http.Handler and can therefore be passed directly to a
// http.ServeMux.Handle or http.ListenAndServe(TLS)
type Server struct {
	rroot, sroot string
	unhandled    ErrorHandler
}

type ErrorHandler func(err error)

// New constructs a remoteStorage server configuration.
// remoteRoot is the root folder of the storage tree (used in the URL),
// storageRoot is a folder on the server's file system where remoteStorage
// documents are written to and read from.
func New(remoteRoot, storageRoot string, errHandler ErrorHandler) (srv Server, err error) {
	rroot := filepath.Clean(remoteRoot)
	sroot := filepath.Clean(storageRoot)
	srv = Server{rroot, sroot, errHandler}

	fi, err := FS.Stat(sroot)
	if err != nil {
		return srv, err
	}
	if !fi.IsDir() {
		return srv, fmt.Errorf("storage root is not a directory: %s", sroot)
	}
	return srv, nil
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
	var err error
	path := r.URL.Path
	if !strings.HasPrefix(path, s.rroot) {
		err = writeError(w, ErrNotFound)
	}
	isFolder := false
	if path[len(path)-1] == '/' {
		isFolder = true
	}

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
		s.unhandled(err)
	}
}

const userKey = "AUTHENTICATED_USER"

// @todo: interceptor for authentication/authorization
func authenticator(next http.Handler) http.Handler {
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
		return writeError(w, HttpError{
			Msg:  "folder not found",
			Desc: "The requested folder does not exist on the server.",
			Data: ldjson{
				"rname": rpath,
			},
			Cause: ErrNotFound,
		})
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err) // internal server error
	}

	if conds := r.Header["If-Non-Match"]; len(conds) > 0 {
		for _, cond := range conds {
			rev, err := ParseETag(cond)
			if err != nil {
				return writeError(w, HttpError{
					Msg:  "invalid etag",
					Desc: "Failed to parse the ETag contained in the If-Non-Match header.",
					Data: ldjson{
						"rname":        rpath,
						"if_non_match": cond,
					},
					Cause: ErrBadRequest,
				})
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
			return writeError(w, err) // internal server error
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
		return writeError(w, HttpError{
			Msg:  "document not found",
			Desc: "The requested document does not exist on the server.",
			Data: ldjson{
				"rname": rpath,
			},
			Cause: ErrNotFound,
		})
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err) // internal server error
	}

	if conds := r.Header["If-Non-Match"]; len(conds) > 0 {
		for _, cond := range conds {
			rev, err := ParseETag(cond)
			if err != nil {
				return writeError(w, HttpError{
					Msg:  "invalid etag",
					Desc: "Failed to parse the ETag contained in the If-Non-Match header.",
					Data: ldjson{
						"rname":        rpath,
						"if_non_match": cond,
					},
					Cause: ErrBadRequest,
				})
			}
			if rev.Equal(etag) {
				return writeError(w, ErrNotModified)
			}
		}
	}

	fd, err := FS.Open(n.Sname)
	if err != nil {
		return writeError(w, err) // internal server error
	}

	hs := w.Header()
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", etag.String())
	hs.Set("Content-Type", n.Mime)
	_, err = io.Copy(w, fd) // @perf: is this efficient for HEAD requests?
	return err
}

func (s Server) PutDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, s.rroot)

	n, err := Retrieve(rpath)
	found := !errors.Is(err, ErrNotExist)
	if err != nil && found { // err is NOT ErrNotExist
		return writeError(w, err) // internal server error
	}

	if found && n.IsFolder {
		return writeError(w, HttpError{
			Msg:  "conflicting path names",
			Desc: "The document conflicts with an already existing folder of the same name.",
			Data: map[string]any{
				"rname": rpath,
			},
			Cause: ErrConflict,
		})
	}

	if cond := r.Header.Get("If-Non-Match"); cond == "*" && found {
		return writeError(w, HttpError{
			Msg:  fmt.Sprintf("document already exists: %s", rpath),
			Desc: "The request was rejected because the requested document already exists, but `If-Non-Match: *' was specified.",
			Data: ldjson{
				"rname": rpath,
			},
			Cause: ErrPreconditionFailed,
		})
	}

	if cond := r.Header.Get("If-Match"); cond != "" {
		rev, err := ParseETag(cond)
		if err != nil {
			// @todo(#ignore_orig): ignore original err?
			return writeError(w, HttpError{
				Msg:  "invalid etag",
				Desc: "Failed to parse the ETag contained in the If-Match header.",
				Data: ldjson{
					"rname":    rpath,
					"if_match": cond,
				},
				Cause: ErrBadRequest,
			})
		}
		etag, err := n.Version()
		if err != nil {
			return writeError(w, err) // internal server error
		}
		if !etag.Equal(rev) {
			return writeError(w, HttpError{
				Msg:  "version mismatch",
				Desc: "The version provided in the If-Match header does not match the documents current version.",
				Data: ldjson{
					"rname":    rpath,
					"if_match": cond,
					"etag":     etag.String(),
				},
				Cause: ErrPreconditionFailed,
			})
		}
	}

	contentType := r.Header.Get("Content-Type")

	// @todo: merge Create and Update into one function?
	if found {
		_, fsize, mime, err := WriteFile(s, n.Sname, r.Body)
		if err != nil {
			return writeError(w, err) // internal server error
		}
		if contentType == "" {
			contentType = mime // @todo(#mime): save doing this work in WriteFile if a contentType was provided?
		}
		UpdateDocument(n, contentType, fsize)
	} else {
		sname, fsize, mime, err := WriteFile(s, "", r.Body)
		if err != nil {
			return writeError(w, err) // internal server error
		}
		if contentType == "" {
			contentType = mime // @todo(#mime)
		}
		n, err = AddDocument(rpath, sname, fsize, contentType)
		var conflictErr *ConflictError
		if err != nil && errors.As(err, &conflictErr) {
			return writeError(w, HttpError{
				Msg:  "conflicting path names",
				Desc: "The name of an ancestor collides with the name of an existing document.",
				Data: ldjson{
					"rname":    rpath,
					"conflict": conflictErr.ConflictPath,
				},
				Cause: ErrConflict,
			})
		}
		assert(err == nil, "ConflictError is the only kind of error returned by AddDocument")
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err) // internal server error
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
		return writeError(w, HttpError{
			Msg:  "document not found",
			Desc: "The requested document does not exist on the server.",
			Data: ldjson{
				"rname": rpath,
			},
			Cause: ErrNotFound,
		})
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err) // internal server error
	}

	if cond := r.Header.Get("If-Match"); cond != "" {
		rev, err := ParseETag(cond)
		if err != nil {
			// @todo(#ignore_orig): ignore original err?
			return writeError(w, HttpError{
				Msg:  "invalid etag",
				Desc: "Failed to parse the ETag contained in the If-Match header.",
				Data: ldjson{
					"rname":    rpath,
					"if_match": cond,
				},
				Cause: ErrBadRequest,
			})
		}
		if !etag.Equal(rev) {
			return writeError(w, HttpError{
				Msg:  "version mismatch",
				Desc: "The version provided in the If-Match header does not match the documents current version.",
				Data: ldjson{
					"rname":    rpath,
					"if_match": cond,
					"etag":     etag.String(),
				},
				Cause: ErrPreconditionFailed,
			})
		}
	}

	RemoveDocument(n)
	err = FS.Remove(n.Sname)
	if err != nil {
		return writeError(w, err) // internal server error
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	return nil
}

func writeError(w http.ResponseWriter, err error) error {
	var (
		unhandled          error
		status, isSentinel = 500, false
		httpErr            HttpError
		data               ldjson
	)
	if errors.As(err, &httpErr) {
		status, isSentinel = StatusCodes[httpErr.Cause]
		if !isSentinel {
			unhandled = httpErr.Cause
			status = StatusCodes[ErrServerError]
		}
		data = ldjson{
			"message":     httpErr.Msg,
			"description": httpErr.Desc,
			"url":         httpErr.URL,
			"data":        httpErr.Data,
		}
	} else {
		status, isSentinel = StatusCodes[err]
		if !isSentinel {
			unhandled = err
			err = ErrServerError
			status = StatusCodes[ErrServerError]
		}
		data = ldjson{
			"message": err.Error(),
		}
	}
	hs := w.Header()
	hs.Set("Content-Type", "application/ld+json")
	w.WriteHeader(status)
	encErr := json.NewEncoder(w).Encode(data)
	if encErr != nil {
		unhandled = errors.Join(unhandled, encErr)
	}
	return unhandled
}
