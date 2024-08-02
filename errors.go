package rmsgo

import (
	"errors"
	"fmt"
	"net/http"
	"encoding/json"
)

type (
	// @todo: to be extended as a RFC 9457 compliant error
	//   (?maybe as its own library?)
	HttpError struct {
		Type string `json:"type"`
		Status int `json:"status"`
		Title string `json:"title"`
		Detail string `json:"detail"`
		Instance string `json:"instance"`
	}

	ErrBadRequest struct {
		HttpError
	}

	ErrMethodNotAllowed struct {
		HttpError
	}

	ErrForbidden struct {
		HttpError
	}

	ErrNotFound struct {
		HttpError
	}

	ErrMaybeNotFound struct {
		HttpError
		Cause error
	}

	ErrNotAFolder struct {
		HttpError
	}

	ErrNotADocument struct {
		HttpError
	}

	ErrInvalidIfNonMatch struct {
		HttpError
	}

	ErrInvalidIfMatch struct {
		HttpError
	}

	ErrNotModified struct{} // not an RFC9457 error: (1) 304 does not allow a body, (2) also not really an error, just an (expected) condition

	ErrConflict struct {
		HttpError
	}

	ErrMaybeAncestorConflict struct {
		HttpError
		Cause error
	}

	ErrDocExists struct {
		HttpError
	}

	ErrVersionMismatch struct {
		HttpError
	}
)

func (e HttpError) Error() string {
	return fmt.Sprintf("%#v", e)
}

func (e HttpError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "application/problem+json") // respond with problem+json even if the client didn't request it (rfc9457#section-3-11)

	//text := http.StatusText(e.Status) // @todo: replace with RFC 9457 formatted error response
	//http.Error(w, text, e.Status)

	// @todo: do we want any validation that the HttpError is valid? (eg., valid Status value, type, ...?)

	w.WriteHeader(e.Status)
	encErr := json.NewEncoder(w).Encode(e)
	if encErr != nil {
		// at this point we probably can't respond (eg., with internal server error) anymore
		g.unhandled(errors.Join(encErr, e))
	}
}

func (e HttpError) RespondError(w http.ResponseWriter, r *http.Request) bool {
	e.ServeHTTP(w, r)
	return true
}

func BadRequest(msg string) error {
	s := http.StatusBadRequest
	return ErrBadRequest{
		HttpError: HttpError{
			Status: s,
			Title: http.StatusText(s),
			Detail: msg,
		},
	}
}

func MethodNotAllowed(msg string) error {
	s := http.StatusMethodNotAllowed
	return ErrMethodNotAllowed{
		HttpError: HttpError{
			Status: s,
			Title: http.StatusText(s),
		},
	}
}

func Forbidden(msg string) error {
	s := http.StatusForbidden
	return ErrForbidden{
		HttpError: HttpError{
			Status: s,
			Title: http.StatusText(s),
		},
	}
}

func NotFound(msg string) error {
	s := http.StatusNotFound
	return ErrNotFound{
		HttpError: HttpError{
			Status: s,
			Title: http.StatusText(s),
		},
	}
}

func MaybeNotFound(err error) error {
	s := http.StatusNotFound
	return ErrMaybeNotFound{
		HttpError: HttpError{
			Status: s,
			Title: http.StatusText(s),
		},
		Cause: err,
	}
}

func (e ErrMaybeNotFound) Unwrap() error {
	return e.Cause
}

func (e ErrMaybeNotFound) IsNotFound() bool {
	return errors.Is(e.Cause, ErrNotExist) // @todo: other possible not founds?
}

func (e ErrMaybeNotFound) RespondError(w http.ResponseWriter, r *http.Request) bool {
	if e.IsNotFound() {
		return e.HttpError.RespondError(w, r)
	}
	return false
}

func NotAFolder(path string) error {
	return ErrNotAFolder{
		HttpError: HttpError{
			Status: http.StatusBadRequest,
			Title: "requested resource is not a folder",
			Detail: "a request was made to retrieve a folder, but a document with the same path was found",
			// @todo: fmt.Sprintf("%s", path),
		},
	}
}

func NotADocument(path string) error {
	return ErrNotADocument{
		HttpError: HttpError{
			Status: http.StatusBadRequest,
			Title: "requested resource is not a document",
			Detail: "a request was made to retrieve a document, but a folder with the same path was found",
			// @todo: fmt.Sprintf("%s", path),
		},
	}
}

func InvalidIfNonMatch(cond string) error {
	return ErrInvalidIfNonMatch{
		HttpError: HttpError {
			Status: http.StatusBadRequest,
			Title: "invalid etag",
			Detail: "the etag contained in the If-None-Match header could not be parsed",
			// @todo: fmt.Sprintf("%s", cond),
		},
	}
}

func InvalidIfMatch(cond string) error {
	return ErrInvalidIfMatch{
		HttpError: HttpError {
			Status: http.StatusBadRequest,
			Title: "invalid etag",
			Detail: "the etag contained in the If-Match header could not be parsed",
			// @todo: fmt.Sprintf("%s", cond),
		},
	}
}

func NotModified() error {
	return ErrNotModified{}
}

func (e ErrNotModified) Error() string {
	return http.StatusText(http.StatusNotModified)
}

func (e ErrNotModified) RespondError(w http.ResponseWriter, r *http.Request) bool {
	s := http.StatusNotModified
	http.Error(w, http.StatusText(s), s)
	return true
}

func Conflict(path string) error {
	return ErrConflict{
		HttpError: HttpError{
			Status: http.StatusConflict,
			Title: "conflicting path names",
			Detail: "the document conflicts with an already existing folder of the same name",
			// @todo: fmt.Sprintf("%s", cond),
		},
	}
}

func MaybeAncestorConflict(err error, path string) error {
	return ErrMaybeAncestorConflict{
		HttpError: HttpError{
			Status: http.StatusConflict,
			Title: "conflicting path names while creating ancestors",
			Detail: "the name of an ancestor collides with the name of an existing document",
			// @todo: fmt.Sprintf("%s", cond),
			//   err.(ConflictError).ConflictPath
		},
		Cause: err,
	}
}

func (e ErrMaybeAncestorConflict) Unwrap() error {
	return e.Cause
}

func (e ErrMaybeAncestorConflict) AsConflict(conflict *ConflictError) bool {
	return errors.As(e.Cause, conflict)
}

func (e ErrMaybeAncestorConflict) RespondError(w http.ResponseWriter, r *http.Request) bool {
	var errConflict ConflictError
	if isConflict := e.AsConflict(&errConflict); isConflict {
		return e.HttpError.RespondError(w, r)
	}
	return false
}

func DocExists(path string) error {
	return ErrDocExists{
		HttpError: HttpError{
			Status: http.StatusPreconditionFailed,
			Title: "document already exists",
			Detail: "the request was rejected because the requested document already exists, but If-None-Match with a value of * was specified",
			// @todo: fmt.Sprintf("%s", cond),
		},
	}
}

func VersionMismatch(expected, actual ETag) error {
	return ErrVersionMismatch{
		HttpError: HttpError{
			Status: http.StatusPreconditionFailed,
			Title: "version mismatch",
			Detail: "the version provided in the If-Match header does not match the document's current version",
			// @todo: expected, actual
		},
	}
}
