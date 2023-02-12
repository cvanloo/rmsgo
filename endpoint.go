package rmsgo

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
)

// ServerConfig holds the server configuration.
var ServerConfig = struct {
	WebRoot, StorageRoot string
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

	// TODO: Handle OPTIONS requests

	// Request not handled
	return nil
}

func GetFolder(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, errors.New("not implemented"))
}

func HeadFolder(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, errors.New("not implemented"))
}

func GetDocument(w http.ResponseWriter, r *http.Request) error {
	// full document contents in body
	// Content-Type
	// Content-Length
	// ETag (strong)
	return writeError(w, errors.New("not implemented"))
}

func HeadDocument(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, errors.New("not implemented"))
}

func PutDocument(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, errors.New("not implemented"))
}

func DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, errors.New("not implemented"))
}

func writeError(w http.ResponseWriter, err error) error {
	status, ok := StatusCodes[err]
	if !ok {
		err = ServerError
		status = StatusCodes[ServerError]
	}

	data := map[string]any{
		"error": err.Error(),
	}

	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
