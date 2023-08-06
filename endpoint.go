package rmsgo

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

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

var mfs fileSystem = &osFileSystem{}

// @todo: strong ETags
// - include hostname / server uri?
// - hash file contents
// - / hash file names of document contents (the entire subtree)
// - last modified time
// - ...?

type Server struct {
	Rroot, Sroot string
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := Serve(w, r)
	if err != nil {
		// @todo: allow user to configure a logging function
		log.Printf("rms-server: %s", err)
	}
}

func Serve(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path
	isFolder := false
	if path[len(path)-1] == '/' {
		isFolder = true
	}

	if isFolder {
		switch r.Method {
		case http.MethodHead:
			//return HeadFolder(w, r)
			fallthrough
		case http.MethodGet:
			return GetFolder(w, r)
		}
	} else {
		switch r.Method {
		case http.MethodHead:
			//return HeadDocument(w, r)
			fallthrough
		case http.MethodGet:
			return GetDocument(w, r)
		case http.MethodPut:
			return PutDocument(w, r)
		case http.MethodDelete:
			return DeleteDocument(w, r)
		}
	}

	return writeError(w, ErrMethodNotAllowed)
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

func GetFolder(w http.ResponseWriter, r *http.Request) error {
	// verify If-Non-Match: (revisions) header (fail with 304 if folder included in revisions)

	// empty folders: "items": {}
	// don't list empty folders in their parents!
	// a folder is non-empty if in its subtree at least one document is contained
	documentDesc := ldjson{
		"ETag":           "DEADBEEFDEADBEEFDEADBEEF",
		"Content-Type":   "image/jpeg",
		"Content-Length": 82352,
		"Last-Modified":  time.Now().Format(time.RFC1123),
	}
	folderDesc := ldjson{
		"ETag": "1337ABCD1337ABCD1337ABCD",
	}
	desc := ldjson{
		"@context": "http://remotestorage.io/spec/folder-description",
		"items": ldjson{
			"abc":  documentDesc,
			"def/": folderDesc,
		},
	}
	hs := w.Header()
	hs.Set("Content-Type", "application/ld+json")
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", "????")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(desc)
}

func HeadFolder(w http.ResponseWriter, r *http.Request) error {
	// @todo: I think this should work if we just redirect the head to GetFolder,
	// Go will omit the body automatically (?)

	// same as GetFolder, but omitting the response body
	return writeError(w, ErrNotImplemented)
}

func GetDocument(w http.ResponseWriter, r *http.Request) error {
	// verify If-Non-Match: (revisions) header (fail with 304 if document included in revisions)

	// return content-type, current version (ETag), and contents

	hs := w.Header()
	hs.Set("Cache-Control", "no-cache")
	hs.Set("ETag", "????")
	return writeError(w, ErrNotImplemented)
}

func HeadDocument(w http.ResponseWriter, r *http.Request) error {
	// @todo: I think this should work if we just redirect the head to GetDocument,
	// Go will omit the body automatically (?)

	// like GetDocument, but omitting the response body
	return writeError(w, ErrNotImplemented)
}

func PutDocument(w http.ResponseWriter, r *http.Request) error {
	// verify If-Match header (fail with 412)
	// verify If-Non-Match: * header (fail with 412 if document already exists)

	// store new version, content-type, and contents, conditional on the
	// current version

	// bs := r.Body write to new file vers

	// auto-create ancestor folders as necessary, add document to parent
	// folder, add each ancestor to its parent

	// ctype := r.Header.Get("Content-Type") store as document's content type
	// if no content type was specified, automatically determine it
	//   --> or: reject request with a descriptive error message

	// update document's ETag, as well as ETag of all ancestor folders

	hs := w.Header()
	hs.Set("ETag", "????") // new etag
	return writeError(w, ErrNotImplemented)
}

func DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	// verify If-Match header (fail with 412)

	// remove document from storage, conditional on the current version

	// remove document from storage

	// remove document from parent folder

	// auto-delete all ancestor folders that are now empty

	// update ETags of all ancestor folders

	hs := w.Header()
	hs.Set("ETag", "????") // deleted etag
	return writeError(w, ErrNotImplemented)
}
