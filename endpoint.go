package rmsgo

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Server struct{
	WebRoot, StorageRoot string
}

// Serve HTTP requests. Unhandled errors are returned non-nil.
func (s *Server) Serve(w http.ResponseWriter, r *http.Request) error {
	// GET folder
	// HEAD folder
	// GET document
	// HEAD document
	// PUT document
	// DELETE document
	// OPTIONS

	// Request not handled
	return nil
}

func GetFolder(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func HeadFolder(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func GetDocument(w http.ResponseWriter, r *http.Request) error {
	// full document contents in body
	// Content-Type
	// Content-Length
	// ETag (strong)
	return errors.New("not implemented")
}

func HeadDocument(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func PutDocument(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func DeleteDocument(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
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
