package rmsgo

import (
	"encoding/json"
	"fmt"
	"framagit.org/attaboy/rmsgo/storage"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
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

	name, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}

	node, ok := storage.Retrieve(name)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Header().Set("ETag", node.Version().Base64())
	w.WriteHeader(http.StatusOK)
	return storage.WriteDescription(w, node.Folder())
}

func (srv Server) HeadFolder(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access)
	_ = user

	name, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	node, ok := storage.Retrieve(name)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Header().Set("ETag", node.Version().Base64())
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

	name, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	node, ok := storage.Retrieve(name)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	doc := node.Document()
	reader, err := doc.Reader()
	if err != nil {
		return writeError(w, err)
	}

	headers := w.Header()
	headers.Set("Content-Type", doc.Mime())
	headers.Set("Content-Length", fmt.Sprintf("%d", doc.Length()))
	headers.Set("ETag", doc.Version().Base64())
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

	name, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	node, ok := storage.Retrieve(name)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	doc := node.Document()

	headers := w.Header()
	headers.Set("Content-Type", doc.Mime())
	headers.Set("Content-Length", fmt.Sprintf("%d", doc.Length()))
	headers.Set("ETag", doc.Version().Base64())
	w.WriteHeader(http.StatusOK)
	return nil
}

func (srv Server) PutDocument(w http.ResponseWriter, r *http.Request) error {
	user, err := srv.auth(r)
	if err != nil {
		return writeError(w, err)
	}

	// what now? (storage access (r/w), quota)
	_ = user

	contentLengthStr := r.Header.Get("Content-Length")
	contentLength, err := strconv.ParseUint(contentLengthStr, 10, 64)
	if err != nil {
		// TODO: wrap errors to provide more info to the client
		return writeError(w, ErrBadRequest)
	}
	contentType := r.Header.Get("Content-Type")
	if len(contentType) == 0 {
		contentType = "text/plain"
	}

	name, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}

	// TODO: put logging into an interceptor / middle service
	log.Printf("request: %s, remote: %s", r.URL.Path, name)

	node, err := storage.Store(name, r.Body, contentType, contentLength)
	if err != nil {
		return writeError(w, err)
	}

	w.Header().Set("ETag", node.Version().Base64())
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

	name, err := filepath.Rel(srv.webRoot, r.URL.Path)
	if err != nil {
		return writeError(w, err)
	}
	err = storage.Remove(name)
	if err != nil {
		return writeError(w, err)
	}

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
