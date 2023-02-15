package rmsgo

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
)

func authenticate(bearer string) (name, secret string, err error) {
	panic("not implemented")
}

// ServerConfig holds the server configuration.
var ServerConfig = struct {
	WebRoot, StorageRoot string
	UserStore UserStorage
}{
	WebRoot:     "/storage/",
	StorageRoot: "/tmp/storage/",
}

// useFS is delegates to the os methods.
var useFS fileSystem = osFS{}

// fileSystem implements method for working with files.
// This abstraction allows mocking the file system when testing.
type fileSystem interface {
	Open(name string) (fs.File, error)
	Stat(name string) (os.FileInfo, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
}

type osFS struct{}

func (osFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
func (osFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (osFS) Remove(name string) error {
	return os.Remove(name)
}

// Serve HTTP requests. Unhandled errors are returned non-nil.
func Serve(w http.ResponseWriter, r *http.Request) error {
	rPath := r.URL.Path
	isFolder := false
	{
		l := len(rPath)
		if rPath[l-1] == '/' {
			isFolder = true
		}
	}

	rMethod := r.Method
	if isFolder {
		switch rMethod {
		case http.MethodGet:
			return GetFolder(w, r)
		case http.MethodHead:
			return HeadFolder(w, r)
		}
	} else {
		switch rMethod {
		case http.MethodGet:
			return GetDocument(w, r)
		case http.MethodHead:
			return HeadDocument(w, r)
		case http.MethodPut:
			return PutDocument(w, r)
		case http.MethodDelete:
			return DeleteDocument(w, r)
		}
	}

	// TODO: Handle OPTIONS/CORS requests

	// Request not handled
	return nil
}

func GetFolder(w http.ResponseWriter, r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	userName, userSecret, err := authenticate(authHeader)
	if err != nil {
		return writeError(w, err)
	}
	user, err := ServerConfig.UserStore.Find(userName, userSecret)
	if err != nil {
		return writeError(w, err)
	}

	// TODO: check user access permissions
	_ = user

	// TODO: remove web root from path
	name := r.URL.Path
	node, ok := Get(name)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Header().Set("ETag", string(node.ETag()))
	w.WriteHeader(http.StatusOK)
	return WriteDescription(w, node)
}

func HeadFolder(w http.ResponseWriter, r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	userName, userSecret, err := authenticate(authHeader)
	if err != nil {
		return writeError(w, err)
	}
	user, err := ServerConfig.UserStore.Find(userName, userSecret)
	if err != nil {
		return writeError(w, err)
	}

	// TODO: check user access permissions
	_ = user

	// TODO: remove web root from path
	name := r.URL.Path
	_, ok := Get(name)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	// Respond with JSON-LD document
	w.Header().Set("Content-Type", "application/ld+json")
	w.WriteHeader(http.StatusOK)
	return nil
}

func GetDocument(w http.ResponseWriter, r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	userName, userSecret, err := authenticate(authHeader)
	if err != nil {
		return writeError(w, err)
	}
	user, err := ServerConfig.UserStore.Find(userName, userSecret)
	if err != nil {
		return writeError(w, err)
	}

	// TODO: check access permissions
	_ = user

	// TODO: remove web root from path
	name := r.URL.Path
	node, ok := Get(name)
	if !ok {
		return writeError(w, ErrNotFound)
	}

	reader, err := storage.Retrieve(name)
	if err != nil {
		return writeError(w, err)
	}

	headers := w.Header()
	headers.Set("Content-Type", node.Mime())
	headers.Set("Content-Length", node.Length())
	headers.Set("ETag", node.ETag())
	w.WriteHeader(http.StatusOK)
	w.Write(reader)
	return nil
}

func HeadDocument(w http.ResponseWriter, r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	userName, userSecret, err := authenticate(authHeader)
	if err != nil {
		return writeError(w, err)
	}
	user, err := UserStore.Find(userName, userSecret)
	if err != nil {
		return writeError(w, err)
	}

	// TODO: remove web root from path
	name := r.URL.Path
	node, err := Get(name)
	if err != nil {
		return writeError(w, err)
	}

	headers := w.Header()
	headers.Set("Content-Type", node.mime)
	headers.Set("Content-Length", node.size)
	headers.Set("ETag", node.ETag())
	w.WriteHeader(http.StatusOK)
	return nil
}

func PutDocument(w http.ResponseWriter, r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	userName, userSecret, err := authenticate(authHeader)
	if err != nil {
		return writeError(w, err)
	}
	user, err := UserStore.Find(userName, userSecret)
	if err != nil {
		return writeError(w, err)
	}

	contentType := r.Header.Get("Content-Type")
	if len(contentType) == 0 {
		// if request without Content-Type, server MAY refuse request
		// (or we just figure out the content type ourselves?)
		// TODO: go get github.com/gabriel-vasile/mimetype
	}

	// TODO: remove web root from path
	// store request body as document contents
	// silently create parent/ancestor folders
	name := r.URL.Path
	node, err := storage.Store(user, name, r.Body)
	if err != nil {
		return writeError(w, err)
	}

	// update filetree, add document to its folder, add each folder to its parent
	// update etags of document and all its ancestor folders
	Add(node)

	w.Header().Set("ETag", node.ETag())
	w.WriteHeader(http.StatusCreated)
	return nil
}

func DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	userName, userSecret, err := authenticate(authHeader)
	if err != nil {
		return writeError(w, err)
	}
	user, err := UserStore.Find(userName, userSecret)
	if err != nil {
		return writeError(w, err)
	}

	// TODO: remove web root from path
	// delete document from storage
	name := r.URL.Path
	err = storage.Remove(name)
	if err != nil {
		return writeError(w, err)
	}

	// delete document from parent folder
	// deletion of any ancestors left empty by this action
	// update version (ETag) of all ancestors
	Remove(name)

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
