package rmsgo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cvanloo/rmsgo.git/isdelve"
)

var mfs fileSystem = &osFileSystem{}

type Server struct {
	Rroot, Sroot string
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
	rpath := strings.TrimPrefix(r.URL.Path, s.Rroot)

	n, err := Node(rpath)
	if err != nil {
		return writeError(w, err)
	}

	etag, err := n.ETag()
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
		etag, err := child.ETag()
		if err != nil {
			return writeError(w, err)
		}
		desc["ETag"] = etag.String()
		if !child.isFolder {
			desc["Content-Type"] = child.mime
			desc["Content-Length"] = child.length
			desc["Last-Modified"] = child.lastMod.Format(rmsTimeFormat)
		}
		items[child.name] = desc
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
	rpath := strings.TrimPrefix(r.URL.Path, s.Rroot)

	n, err := Node(rpath)
	if err != nil {
		return writeError(w, err)
	}

	etag, err := n.ETag()
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

	fd, err := mfs.Open(n.sname)
	if err != nil {
		return writeError(w, err)
	}

	hs := w.Header()
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", etag.String())
	hs.Set("Content-Type", n.mime)
	_, err = io.Copy(w, fd)
	return err
}

func (s Server) PutDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, s.Rroot)

	n, err := Node(rpath)
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
		etag, err := n.ETag()
		if err != nil {
			return writeError(w, err)
		}
		if !etag.Equal(rev) {
			return writeError(w, ErrPreconditionFailed) // @todo(#desc_error): version mismatch
		}
	}

	// @todo: we could also automatically determine the mime type
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return writeError(w, ErrBadRequest) // @todo(#desc_error): provide a content type
	}

	// @todo: funny...
	//   merge Create and Update into one function?
	var fun func(Server, string, io.Reader, string) (*node, error)
	if notFound {
		fun = CreateDocument
	} else {
		fun = UpdateDocument
	}
	n, err = fun(s, rpath, r.Body, contentType)
	if err != nil {
		return writeError(w, err)
	}

	etag, err := n.ETag()
	if err != nil {
		return writeError(w, err)
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	return nil
}

func (s Server) DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, s.Rroot)

	n, err := Node(rpath)
	if err != nil {
		return writeError(w, err)
	}

	etag, err := n.ETag()
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

	n, err = RemoveDocument(s, rpath)
	if err != nil {
		return writeError(w, err)
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	return nil
}

func init() {
	if isdelve.Enabled {
		mfs = CreateMockFS()
		log.Println("Debugger detected, using mock filesystem")
	}
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
