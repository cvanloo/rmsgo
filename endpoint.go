package rmsgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	. "github.com/cvanloo/rmsgo/mock"
)

// @todo: https://datatracker.ietf.org/doc/html/draft-dejong-remotestorage-21#section-6
// keep multiple versions of files around, option to restore deleted files
// > A provider MAY offer version rollback functionality to its users,
// > but this specification does not define the interface for that.

type (
	Middleware func(next http.Handler) http.Handler

	HandlerWithError func(w http.ResponseWriter, r *http.Request) error

	MuxWithError struct {
		http.ServeMux
	}

	ErrorResponder interface {
		RespondError(w http.ResponseWriter, r *http.Request) (wasHandled bool)
	}
)

func MiddlewareStack(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			m := middlewares[i]
			next = m(next)
		}
		return next
	}
}

func (h HandlerWithError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		if err, ok := err.(ErrorResponder); ok {
			if err.RespondError(w, r) {
				return
			}
		}
		status := http.StatusInternalServerError
		http.Error(w, http.StatusText(status), status)
		g.unhandled(err)
	}
}

func (m *MuxWithError) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request) error) {
	m.ServeMux.Handle(pattern, HandlerWithError(handler))
}

func RMSRouter() http.Handler {
	folderMux := &MuxWithError{}
	folderMux.HandleFunc("GET /", getFolder)
	folderMux.Handle("/", HandlerWithError(func(http.ResponseWriter, *http.Request) error {
		return BadRequest("method not allowed on folders")
	}))

	documentMux := &MuxWithError{}
	documentMux.HandleFunc("GET /", getDocument)
	documentMux.HandleFunc("PUT /", putDocument)
	documentMux.HandleFunc("DELETE /", deleteDocument)
	documentMux.Handle("/", HandlerWithError(func(http.ResponseWriter, *http.Request) error {
		return BadRequest("method not allowed on documents")
	}))

	mux := &MuxWithError{}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) error {
		path := r.URL.Path
		isFolder := path[len(path)-1] == '/' // @fixme: is this the path without query parameters and stuff?
		if isFolder {
			folderMux.ServeHTTP(w, r)
		} else {
			documentMux.ServeHTTP(w, r)
		}
		return nil
	})

	return mux
}

func getFolder(w http.ResponseWriter, r *http.Request) error {
	n, err := Retrieve(r.URL.Path)
	if err != nil {
		return MaybeNotFound(err)
	}
	if !n.isFolder {
		return NotAFolder(n.rname)
	}

	etag, err := n.Version()
	if err != nil {
		return err // internal server error
	}

	if condStr := r.Header.Get("If-None-Match"); condStr != "" { // @todo: extract into its own type/functionality?
		conds := strings.Split(condStr, ",")
		for _, cond := range conds {
			cond = strings.TrimSpace(cond)
			rev, err := ParseETag(cond)
			if err != nil {
				// @note(#orig_error): it's ok to lose the original error,
				// since it can only be caused by a malformed ETag.
				return InvalidIfNonMatch(cond) // @todo: pass original err?
			}
			if rev.Equal(etag) {
				return NotModified()
			}
		}
	}

	items := LDjson{}
	for _, child := range n.children {
		desc := LDjson{}
		etag, err := child.Version()
		if err != nil {
			return err // internal server error
		}
		desc["ETag"] = etag.String()
		if !child.isFolder {
			desc["Content-Type"] = child.mime
			desc["Content-Length"] = child.length
			desc["Last-Modified"] = child.lastMod.Format(timeFormat)
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

func getDocument(w http.ResponseWriter, r *http.Request) error {
	n, err := Retrieve(r.URL.Path)
	if err != nil {
		return MaybeNotFound(err)
	}
	if n.isFolder {
		return NotADocument(n.rname)
	}

	etag, err := n.Version()
	if err != nil {
		return err // internal server error
	}

	if condStr := r.Header.Get("If-None-Match"); condStr != "" {
		conds := strings.Split(condStr, ",")
		for _, cond := range conds {
			cond = strings.TrimSpace(cond)
			rev, err := ParseETag(cond)
			if err != nil {
				// @note(#orig_error): it's ok to lose the original error,
				// since it can only have been caused by a malformed ETag.
				return InvalidIfNonMatch(cond) // @todo: parse all and report all errors instead of only the first?
			}
			if rev.Equal(etag) {
				return NotModified()
			}
		}
	}

	fd, err := FS.Open(n.sname)
	if err != nil {
		return err // internal server error
	}

	hs := w.Header()
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", etag.String())
	hs.Set("Content-Type", n.mime)
	hs.Set("Content-Length", fmt.Sprintf("%d", n.length))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, err = io.Copy(w, fd)
	}
	return err
}

func putDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := r.URL.Path

	n, err := Retrieve(rpath)
	found := !errors.Is(err, ErrNotExist)

	if found { // err is /not/ ErrNotExist
		if err != nil {
			return err // internal server error
		}
		if n.isFolder {
			return Conflict(n.rname)
		}
	}

	if cond := r.Header.Get("If-None-Match"); cond == "*" && found {
		etag, err := n.Version()
		if err != nil {
			return err // internal server error
		}
		w.Header().Set("ETag", etag.String())
		return DocExists(n.rname)
	}

	if cond := r.Header.Get("If-Match"); cond != "" {
		rev, err := ParseETag(cond)
		if err != nil {
			// @note(#orig_error): it's ok to lose the original error, since
			// it can only be caused by a malformed ETag.
			return InvalidIfMatch(cond)
		}
		etag, err := n.Version()
		if err != nil {
			return err // internal server error
		}
		w.Header().Set("ETag", etag.String())
		if !etag.Equal(rev) {
			return VersionMismatch(rev, etag)
		}
	}

	mime := r.Header.Get("Content-Type")

	if found {
		fd, err := FS.Create(n.sname)
		if err != nil {
			return err // internal server error
		}

		fsize, err := io.Copy(fd, r.Body)
		if err != nil {
			return err // internal server error
		}

		if mime == "" {
			_, err := fd.Seek(0, io.SeekStart)
			if err != nil {
				return err // internal server error
			}
			bs := make([]byte, 512)
			_, err = fd.Read(bs)
			if err != nil {
				return err // internal server error
			}
			mime = http.DetectContentType(bs)
		}

		UpdateDocument(n, mime, fsize)

		err = fd.Close()
		if err != nil {
			return err // internal server error
		}
	} else {
		u, err := UUID()
		if err != nil {
			return err // internal server error
		}
		sname := filepath.Join(g.sroot, u.String())

		fd, err := FS.Create(sname)
		if err != nil {
			return err // internal server error
		}

		fsize, err := io.Copy(fd, r.Body)
		if err != nil {
			return err // internal server error
		}

		if mime == "" {
			_, err := fd.Seek(0, io.SeekStart)
			if err != nil {
				return err // internal server error
			}
			bs := make([]byte, 512)
			_, err = fd.Read(bs)
			if err != nil {
				return err // internal server error
			}
			mime = http.DetectContentType(bs)
		}

		n, err = AddDocument(rpath, sname, fsize, mime)
		if err != nil {
			return MaybeAncestorConflict(err, rpath)
		}

		err = fd.Close()
		if err != nil {
			return err // internal server error
		}
	}

	etag, err := n.Version()
	if err != nil {
		return err // internal server error
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	w.WriteHeader(http.StatusCreated)
	return nil
}

func deleteDocument(w http.ResponseWriter, r *http.Request) error {
	rpath := r.URL.Path

	n, err := Retrieve(rpath)
	if err != nil {
		return MaybeNotFound(err)
	}
	if n.isFolder {
		return NotADocument(n.rname)
	}

	etag, err := n.Version()
	if err != nil {
		return err // internal server error
	}

	if cond := r.Header.Get("If-Match"); cond != "" {
		rev, err := ParseETag(cond)
		if err != nil {
			// @note(#orig_error): it's ok to lose the original error, since
			// it can only be caused by a malformed ETag.
			return InvalidIfMatch(cond)
		}
		if !etag.Equal(rev) {
			w.Header().Set("ETag", etag.String())
			return VersionMismatch(rev, etag)
		}
	}

	RemoveDocument(n)
	err = FS.Remove(n.sname)
	if err != nil {
		return err // internal server error
	}

	hs := w.Header()
	hs.Set("ETag", etag.String())
	w.WriteHeader(http.StatusOK)
	return nil
}
