package rmsgo

import "errors"

// Sentinel error values
var (
	ErrServerError         = errors.New("internal server error")
	ErrNotModified         = errors.New("not modified")
	ErrUnauthorized        = errors.New("missing or invalid bearer token")
	ErrForbidden           = errors.New("insufficient scope")
	ErrNotFound            = errors.New("document does not exist")
	ErrConflict            = errors.New("conflicting document/folder names")
	ErrPreconditionFailed  = errors.New("precondition failed")
	ErrTooLarge            = errors.New("request entity too large")
	ErrUriTooLong          = errors.New("request uri too long")
	ErrRangeNotSatisfiable = errors.New("request range not satisfiable")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrMethodNotAllowed    = errors.New("method not allowed")
	ErrInsufficientStorage = errors.New("insufficient storage")
)

// StatusCodes maps errors to their respective HTTP status codes
var StatusCodes = map[error]int{
	ErrServerError:          500,
	ErrNotModified:          304,
	ErrUnauthorized:         401,
	ErrForbidden:            403,
	ErrNotFound:             404,
	ErrConflict:             409,
	ErrPreconditionFailed:   412,
	ErrTooLarge:             413,
	ErrUriTooLong:           414,
	ErrRangeNotSatisfiable:  416,
	ErrTooManyRequests:      429,
	ErrMethodNotAllowed:     405,
	ErrInsufficientStorage:  507,
}
