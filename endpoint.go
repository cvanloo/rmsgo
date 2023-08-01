package rmsgo

import (
	"encoding/json"
	"net/http"
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

var FS fileSystem = &osFileSystem{}

func Serve(w http.ResponseWriter, r *http.Request) error {
	// @todo: is it a document or a folder?
	isFolder := true

	if isFolder {
		switch r.Method {
		case http.MethodGet:
			return GetFolder(w, r)
		case http.MethodHead:
			return HeadFolder(w, r)
		}
	} else {
		switch r.Method {
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

	return writeError(w, ErrMethodNotAllowed)
}

// @todo: interceptor for authentication/authorization

func GetFolder(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, ErrNotImplemented)
}

func HeadFolder(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, ErrNotImplemented)
}

func GetDocument(w http.ResponseWriter, r *http.Request) error {
	// @todo: remove demo code
	f, err := FS.Open("test.txt")
	if err != nil {
		panic(err)
	}
	_ = f
	return writeError(w, ErrNotImplemented)
}

func HeadDocument(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, ErrNotImplemented)
}

func PutDocument(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, ErrNotImplemented)
}

func DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	return writeError(w, ErrNotImplemented)
}
