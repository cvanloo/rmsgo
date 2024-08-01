package rmsgo

import (
	"errors"
	"fmt"
	"net/http"
)

// @nocheckin: some of these might be better of as sentinel error values, instead of constructing them anew each time

type (
	// @todo: to be extended as a RFC 9457 compliant error
	//   (?maybe as its own library?)
	HttpError struct {
		Type string
		Status int
		Title string
		Detail string
		Instance string
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

	ErrNotModified struct {
		HttpError
	}

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
	text := http.StatusText(e.Status) // @todo: replace with RFC 9457 formatted error response
	http.Error(w, text, e.Status)
}

func (e HttpError) RespondError(w http.ResponseWriter, r *http.Request) bool {
	http.Error(w, http.StatusText(e.Status), e.Status)
	return true
}

func MethodNotAllowed() error {
	return ErrMethodNotAllowed{
		HttpError: HttpError{
			Status: http.StatusMethodNotAllowed,
		},
	}
}

func Forbidden() error {
	return ErrForbidden{
		HttpError: HttpError{
			Status: http.StatusForbidden,
		},
	}
}

func NotFound() error {
	return ErrNotFound{
		HttpError: HttpError{
			Status: http.StatusNotFound,
		},
	}
}

func MaybeNotFound(err error) error {
	return ErrMaybeNotFound{
		HttpError: HttpError{
			Status: http.StatusNotFound,
		},
		Cause: err,
	}
}

func (e ErrMaybeNotFound) Unwrap() error {
	return e.Cause
}

func (e ErrMaybeNotFound) IsNotFound() bool {
	return errors.Is(e.Cause, ErrNotExist) // @nocheckin: other possible not founds?
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
	return ErrNotModified{
		HttpError: HttpError{
			Status: http.StatusNotModified,
		},
	}
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
