package rmsgo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"framagit.org/attaboy/rmsgo/filetree"
	"framagit.org/attaboy/rmsgo/storage"
)

// Serve HTTP requests. Unhandled errors are returned non-nil.
func (srv Server) Serve(w http.ResponseWriter, r *http.Request) error {
	rPath := r.URL.Path
	isFolder := false
	l := len(rPath)
	if rPath[l-1] == '/' {
		isFolder = true
	}

	// TODO: Handle OPTIONS/CORS requests

	rMethod := r.Method
	if isFolder {
		switch rMethod {
		case http.MethodGet:
			return srv.GetFolder(w, r)
		case http.MethodHead:
			return srv.HeadFolder(w, r)
		case http.MethodOptions:
			return writeError(w, ErrNotImplemented)
		}
	} else {
		switch rMethod {
		case http.MethodGet:
			return srv.GetDocument(w, r)
		case http.MethodHead:
			return srv.HeadDocument(w, r)
		case http.MethodPut:
			return srv.PutDocument(w, r)
		case http.MethodDelete:
			return srv.DeleteDocument(w, r)
		case http.MethodOptions:
			return writeError(w, ErrNotImplemented)
		}
	}

	return writeError(w, ErrMethodNotAllowed)
}

func (srv Server) GetFolder(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access, validate quota, ...?)
	_ = user

	path, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}

	node, ok := filetree.Get(path)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Header().Set("ETag", string(node.Version()))
	w.WriteHeader(http.StatusOK)
	return filetree.WriteDescription(w, node.Folder())
}

func (srv Server) HeadFolder(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access)
	_ = user

	path, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	_, ok := filetree.Get(path)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.WriteHeader(http.StatusOK)
	return nil
}

func (srv Server) GetDocument(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access)
	_ = user

	path, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	node, ok := filetree.Get(path)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	reader, err := storage.Retrieve(path)
	if err != nil {
		return writeError(w, err)
	}

	doc := node.Document()

	headers := w.Header()
	headers.Set("Content-Type", doc.Mime)
	headers.Set("Content-Length", fmt.Sprintf("%d", doc.Length))
	headers.Set("ETag", string(doc.Version()))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
	return nil
}

func (srv Server) HeadDocument(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access)
	_ = user

	path, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	node, ok := filetree.Get(path)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	doc := node.Document()

	headers := w.Header()
	headers.Set("Content-Type", doc.Mime)
	headers.Set("Content-Length", fmt.Sprintf("%d", doc.Length))
	headers.Set("ETag", string(doc.Version()))
	w.WriteHeader(http.StatusOK)
	return nil
}

func (srv Server) PutDocument(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access)
	_ = user

	contentLengthStr := r.Header.Get("Content-Length")
	contentLength, err := strconv.ParseUint(contentLengthStr, 10, 64)
	if err != nil {
		// TODO: wrap errors to provide more info to the client
		return writeError(w, ErrBadRequest)
	}
	contentType := r.Header.Get("Content-Type")
	if len(contentType) == 0 {
		// if request without Content-Type, server MAY refuse request
		// (or we just figure out the content type ourselves?)
		// TODO: go get github.com/gabriel-vasile/mimetype
	}

	path, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	err = storage.Store(path, r.Body)
	if err != nil {
		return writeError(w, err)
	}

	doc := filetree.NewDocument(path, contentType, uint(contentLength), time.Now())
	if err != nil {
		return writeError(w, err)
	}
	filetree.Add(doc)

	w.Header().Set("ETag", string(doc.Version()))
	w.WriteHeader(http.StatusCreated)
	return nil
}

func (srv Server) DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access)
	_ = user

	// TODO: remove web root from path
	// delete document from storage
	path, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	err = storage.Remove(path)
	if err != nil {
		return writeError(w, err)
	}

	// delete document from parent folder
	// deletion of any ancestors left empty by this action
	// update version (ETag) of all ancestors
	filetree.Remove(path)

	w.WriteHeader(http.StatusOK)
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
	}

	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
