package rmsgo

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Server struct{}

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

func (s *Server) GetFolder(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func (s *Server) HeadFolder(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func (s *Server) GetDocument(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func (s *Server) HeadDocument(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func (s *Server) PutDocument(w http.ResponseWriter, r *http.Request) error {
	return errors.New("not implemented")
}

func (s *Server) DeleteDocument(w http.ResponseWriter, r *http.Request) error {
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
