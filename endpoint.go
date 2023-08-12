package rmsgo

import (
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
	. "github.com/cvanloo/rmsgo.git/mock"
)

func init() {
	if !isdelve.Enabled {
		FS = &RealFileSystem{}
	}
}

// Any errors that the remoteStorage server doesn't know how to handle itself
// are passed to the ErrorHandler.
type ErrorHandler func(err error)

// ServeMux implements http.Handler and can therefore be passed directly to a
// http.ServeMux.Handle or http.ListenAndServe(TLS).
type ServeMux struct{}

func (ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := Serve(w, r)
	if err != nil {
		unhandled(err)
	}
}

const rmsTimeFormat = time.RFC1123

var (
	rroot, sroot string
	unhandled    ErrorHandler = func(err error) {
		log.Printf("rmsgo: unhandled error: %v\n", err)
	}
)

// Configure creates a remoteStorage server configuration.
// remoteRoot is the root folder of the storage tree (used in the URL),
// storageRoot is a folder on the server's file system where remoteStorage
// documents are written to and read from.
func Configure(remoteRoot, storageRoot string, errHandler ErrorHandler) error {
	rroot = filepath.Clean(remoteRoot)
	sroot = filepath.Clean(storageRoot)

	if errHandler != nil {
		unhandled = errHandler
	}

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

// Serve responds to a remoteStorage request.
func Serve(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path
	if !strings.HasPrefix(path, rroot) {
		return writeError(w, ErrNotFound)
	}
	isFolder := false
	if path[len(path)-1] == '/' {
		isFolder = true
	}

	switch r.Method {
	case http.MethodHead:
		fallthrough
	case http.MethodGet:
		if isFolder {
			return GetFolder(w, r)
		} else {
			return GetDocument(w, r)
		}
	case http.MethodPut:
		if isFolder {
			return writeError(w, HttpError{
				Msg:   "put to folder disallowed",
				Desc:  "PUT requests only need to be made to documents, and never to folders.",
				Cause: ErrBadRequest,
			})
		} else {
			return PutDocument(w, r)
		}
	case http.MethodDelete:
		if isFolder {
			return writeError(w, HttpError{
				Msg:   "delete to folder disallowed",
				Desc:  "DELETE requests only need to be made to documents, and never to folders.",
				Cause: ErrBadRequest,
			})
		} else {
			return DeleteDocument(w, r)
		}
	}

	return writeError(w, ErrMethodNotAllowed)
}

/*
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
*/

// @todo: OPTIONS/cors
// @todo: https://datatracker.ietf.org/doc/html/draft-dejong-remotestorage-21#section-6
// keep multiple versions of files around, option to restore deleted files
// > A provider MAY offer version rollback functionality to its users,
// > but this specification does not define the interface for that.

func GetFolder(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return writeError(w, HttpError{
			Msg:  "folder not found",
			Desc: "The requested folder does not exist on the server.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrNotFound,
		})
	}

	etag, err := n.Version()
	if err != nil {
		return writeError(w, err) // internal server error
	}

	if condStr := r.Header.Get("If-Non-Match"); condStr != "" {
		conds := strings.Split(condStr, ",")
		for _, cond := range conds {
			cond = strings.TrimSpace(cond)
			rev, err := ParseETag(cond)
			if err != nil {
				return writeError(w, HttpError{
					Msg:  "invalid etag",
					Desc: "Failed to parse the ETag contained in the If-Non-Match header.",
					Data: LDjson{
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

	items := LDjson{}
	for _, child := range n.children {
		desc := LDjson{}
		etag, err := child.Version()
		if err != nil {
			return writeError(w, err) // internal server error
		}
		desc["ETag"] = etag.String()
		if !child.isFolder {
			desc["Content-Type"] = child.mime
			desc["Content-Length"] = child.length
			desc["Last-Modified"] = child.lastMod.Format(rmsTimeFormat)
		}
		items[child.name] = desc
	}

	desc := LDjson{
		"@context": "http://remotestorage.io/spec/folder-description",
		"items":    items,
	}

	hs := w.Header()
	hs.Set("Content-Type", "application/ld+json")
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", etag.String())
	return json.NewEncoder(w).Encode(desc)
}

func GetDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return writeError(w, HttpError{
			Msg:  "document not found",
			Desc: "The requested document does not exist on the server.",
			Data: LDjson{
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
					Data: LDjson{
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

	fd, err := FS.Open(n.sname)
	if err != nil {
		return writeError(w, err) // internal server error
	}

	hs := w.Header()
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", etag.String())
	hs.Set("Content-Type", n.mime)
	_, err = io.Copy(w, fd) // @perf: is this efficient for HEAD requests?
	return err
}

func PutDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, rroot)

	n, err := Retrieve(rpath)
	found := !errors.Is(err, ErrNotExist)
	if err != nil && found { // err is NOT ErrNotExist
		return writeError(w, err) // internal server error
	}

	if found && n.isFolder {
		return writeError(w, HttpError{
			Msg:  "conflicting path names",
			Desc: "The document conflicts with an already existing folder of the same name.",
			Data: map[string]any{
				"rname":    rpath,
				"conflict": rpath,
			},
			Cause: ErrConflict,
		})
	}

	if cond := r.Header.Get("If-Non-Match"); cond == "*" && found {
		return writeError(w, HttpError{
			Msg:  fmt.Sprintf("document already exists: %s", rpath),
			Desc: "The request was rejected because the requested document already exists, but `If-Non-Match: *' was specified.",
			Data: LDjson{
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
				Data: LDjson{
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
				Desc: "The version provided in the If-Match header does not match the document's current version.",
				Data: LDjson{
					"rname":    rpath,
					"if_match": cond,
					"etag":     etag.String(),
				},
				Cause: ErrPreconditionFailed,
			})
		}
	}

	mime := r.Header.Get("Content-Type")

	// @todo: merge Create and Update into one function?
	if found {
		fd, err := FS.Create(n.sname)
		if err != nil {
			return writeError(w, err) // internal server error
		}

		fsize, err := io.Copy(fd, r.Body)
		if err != nil {
			return writeError(w, err) // internal server error
		}

		if mime == "" {
			mime, err = DetectMime(fd)
			if err != nil {
				return writeError(w, err) // internal server error
			}
		}

		UpdateDocument(n, mime, fsize)

		err = fd.Close()
		if err != nil {
			return writeError(w, err) // internal server error
		}
	} else {
		u, err := UUID()
		if err != nil {
			return writeError(w, err) // internal server error
		}
		sname := filepath.Join(sroot, u.String())

		fd, err := FS.Create(sname)
		if err != nil {
			return writeError(w, err) // internal server error
		}

		fsize, err := io.Copy(fd, r.Body)
		if err != nil {
			return writeError(w, err) // internal server error
		}

		if mime == "" {
			mime, err = DetectMime(fd)
			if err != nil {
				return writeError(w, err) // internal server error
			}
		}

		n, err = AddDocument(rpath, sname, fsize, mime)
		var conflictErr ConflictError
		if err != nil && errors.As(err, &conflictErr) {
			return writeError(w, HttpError{
				Msg:  "conflicting path names",
				Desc: "The name of an ancestor collides with the name of an existing document.",
				Data: LDjson{
					"rname":    rpath,
					"conflict": conflictErr.ConflictPath,
				},
				Cause: ErrConflict,
			})
		}
		assert(err == nil, "ConflictError is the only kind of error returned by AddDocument")

		err = fd.Close()
		if err != nil {
			return writeError(w, err) // internal server error
		}
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

func DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return writeError(w, HttpError{
			Msg:  "document not found",
			Desc: "The requested document does not exist on the server.",
			Data: LDjson{
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
				Data: LDjson{
					"rname":    rpath,
					"if_match": cond,
				},
				Cause: ErrBadRequest,
			})
		}
		if !etag.Equal(rev) {
			return writeError(w, HttpError{
				Msg:  "version mismatch",
				Desc: "The version provided in the If-Match header does not match the document's current version.",
				Data: LDjson{
					"rname":    rpath,
					"if_match": cond,
					"etag":     etag.String(),
				},
				Cause: ErrPreconditionFailed,
			})
		}
	}

	RemoveDocument(n)
	err = FS.Remove(n.sname)
	if err != nil {
		return writeError(w, err) // internal server error
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	return nil
}

func writeError(w http.ResponseWriter, err error) error {
	var (
		httpErr   HttpError
		unhandled error
	)
	if errors.As(err, &httpErr) {
		status, isSentinel := StatusCodes[httpErr.Cause]
		if !isSentinel {
			unhandled = httpErr.Cause
			status = StatusCodes[ErrServerError]
		}
		data := LDjson{
			"message":     httpErr.Msg,
			"description": httpErr.Desc,
			"url":         httpErr.URL,
			"data":        httpErr.Data,
		}
		hs := w.Header()
		hs.Set("Content-Type", "application/ld+json")
		w.WriteHeader(status)
		encErr := json.NewEncoder(w).Encode(data)
		if encErr != nil {
			unhandled = errors.Join(unhandled, encErr)
		}
		w.WriteHeader(status)
		return unhandled
	} else {
		status, isSentinel := StatusCodes[err]
		if !isSentinel {
			unhandled = err
			status = StatusCodes[ErrServerError]
		}
		w.WriteHeader(status)
		return unhandled
	}
}
