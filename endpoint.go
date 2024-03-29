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

func handleRMS() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := serve(w, r); err != nil {
			if err := WriteError(w, err); err != nil {
				g.unhandled(err)
			}
		}
	})
}

func serve(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path
	isFolder := path[len(path)-1] == '/'

	switch r.Method {
	case http.MethodHead:
		fallthrough
	case http.MethodGet:
		if isFolder {
			return getFolder(w, r)
		} else {
			return getDocument(w, r)
		}
	case http.MethodPut:
		if isFolder {
			return HttpError{
				Msg:   "put to folder disallowed",
				Desc:  "PUT requests need only be made to documents, and never to folders.",
				Cause: ErrBadRequest,
			}
		} else {
			return putDocument(w, r)
		}
	case http.MethodDelete:
		if isFolder {
			return HttpError{
				Msg:   "delete to folder disallowed",
				Desc:  "DELETE requests need only be made to documents, and never to folders.",
				Cause: ErrBadRequest,
			}
		} else {
			return deleteDocument(w, r)
		}
	}

	return ErrMethodNotAllowed
}

func getFolder(w http.ResponseWriter, r *http.Request) error {
	rpath := strings.TrimPrefix(r.URL.Path, g.rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return HttpError{
			Msg:  "folder not found",
			Desc: "The requested folder does not exist on the server.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrNotFound,
		}
	}
	if !n.isFolder {
		return HttpError{
			Msg:  "requested resource is not a folder",
			Desc: "A request was made to retrieve a folder, but a document with the same path was found.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrBadRequest,
		}
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
				// since it can only be caused by a malformed ETag.
				return HttpError{
					Msg:  "invalid etag",
					Desc: "Failed to parse the ETag contained in the If-None-Match header.",
					Data: LDjson{
						"rname":         rpath,
						"if_none_match": cond,
					},
					Cause: ErrBadRequest,
				}
			}
			if rev.Equal(etag) {
				return ErrNotModified
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
	rpath := strings.TrimPrefix(r.URL.Path, g.rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return HttpError{
			Msg:  "document not found",
			Desc: "The requested document does not exist on the server.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrNotFound,
		}
	}
	if n.isFolder {
		return HttpError{
			Msg:  "requested resource is not a document",
			Desc: "A request was made to retrieve a document, but a folder with the same path was found.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrBadRequest,
		}
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
				return HttpError{
					Msg:  "invalid etag",
					Desc: "Failed to parse the ETag contained in the If-None-Match header.",
					Data: LDjson{
						"rname":         rpath,
						"if_none_match": cond,
					},
					Cause: ErrBadRequest,
				}
			}
			if rev.Equal(etag) {
				return ErrNotModified
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
	rpath := strings.TrimPrefix(r.URL.Path, g.rroot)

	n, err := Retrieve(rpath)
	found := !errors.Is(err, ErrNotExist)

	if err != nil && found { // err is NOT ErrNotExist
		return err // internal server error
	}

	if found && n.isFolder {
		return HttpError{
			Msg:  "conflicting path names",
			Desc: "The document conflicts with an already existing folder of the same name.",
			Data: map[string]any{
				"rname":    rpath,
				"conflict": rpath,
			},
			Cause: ErrConflict,
		}
	}

	if cond := r.Header.Get("If-None-Match"); cond == "*" && found {
		etag, err := n.Version()
		if err != nil {
			return err // internal server error
		}
		w.Header().Set("ETag", etag.String())
		return HttpError{
			Msg:  fmt.Sprintf("document already exists: %s", rpath),
			Desc: "The request was rejected because the requested document already exists, but `If-None-Match: *' was specified.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrPreconditionFailed,
		}
	}

	if cond := r.Header.Get("If-Match"); cond != "" {
		rev, err := ParseETag(cond)
		if err != nil {
			// @note(#orig_error): it's ok to lose the original error, since
			// it can only be caused by a malformed ETag.
			return HttpError{
				Msg:  "invalid etag",
				Desc: "Failed to parse the ETag contained in the If-Match header.",
				Data: LDjson{
					"rname":    rpath,
					"if_match": cond,
				},
				Cause: ErrBadRequest,
			}
		}
		etag, err := n.Version()
		if err != nil {
			return err // internal server error
		}
		w.Header().Set("ETag", etag.String())
		if !etag.Equal(rev) {
			return HttpError{
				Msg:  "version mismatch",
				Desc: "The version provided in the If-Match header does not match the document's current version.",
				Data: LDjson{
					"rname":           rpath,
					"if_match":        cond,
					"current_version": etag.String(),
				},
				Cause: ErrPreconditionFailed,
			}
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
		var conflictErr ConflictError
		if err != nil && errors.As(err, &conflictErr) {
			return HttpError{
				Msg:  "conflicting path names",
				Desc: "The name of an ancestor collides with the name of an existing document.",
				Data: LDjson{
					"rname":    rpath,
					"conflict": conflictErr.ConflictPath,
				},
				Cause: ErrConflict,
			}
		}
		assert(err == nil, "ConflictError is the only kind of error returned by AddDocument")

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
	rpath := strings.TrimPrefix(r.URL.Path, g.rroot)

	n, err := Retrieve(rpath)
	if err != nil {
		return HttpError{
			Msg:  "document not found",
			Desc: "The requested document does not exist on the server.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrNotFound,
		}
	}
	if n.isFolder {
		return HttpError{
			Msg:  "requested resource is not a document",
			Desc: "A request was made to retrieve a document, but a folder with the same path was found.",
			Data: LDjson{
				"rname": rpath,
			},
			Cause: ErrBadRequest,
		}
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
			return HttpError{
				Msg:  "invalid etag",
				Desc: "Failed to parse the ETag contained in the If-Match header.",
				Data: LDjson{
					"rname":    rpath,
					"if_match": cond,
				},
				Cause: ErrBadRequest,
			}
		}
		if !etag.Equal(rev) {
			w.Header().Set("ETag", etag.String())
			return HttpError{
				Msg:  "version mismatch",
				Desc: "The version provided in the If-Match header does not match the document's current version.",
				Data: LDjson{
					"rname":           rpath,
					"if_match":        cond,
					"current_version": etag.String(),
				},
				Cause: ErrPreconditionFailed,
			}
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

// WriteError formats and writes err to w.
// If err is wrapped in an HttpError, it is formatted into an ld+json map.
// The status code is decided based upon the (inner) error.
// If err or err.(HttpError).Cause is one of the sentinel errors no further handling is needed.
// Otherwise, an ErrServerError is written to w and the error returned for further handling.
func WriteError(w http.ResponseWriter, err error) error {
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
