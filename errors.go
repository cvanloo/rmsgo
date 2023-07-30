package rmsgo

import "errors"

// Sentinel error values
var (
	ErrServerError         = errors.New("internal server error")
	ErrNotImplemented      = errors.New("not implemented")
	ErrNotModified         = errors.New("not modified")
	ErrUnauthorized        = errors.New("missing or invalid bearer token")
	ErrForbidden           = errors.New("insufficient scope")
	ErrNotFound            = errors.New("resource not found")
	ErrConflict            = errors.New("conflicting document/folder names")
	ErrPreconditionFailed  = errors.New("precondition failed")
	ErrTooLarge            = errors.New("request entity too large")
	ErrUriTooLong          = errors.New("request uri too long")
	ErrRangeNotSatisfiable = errors.New("request range not satisfiable")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrMethodNotAllowed    = errors.New("method not allowed")
	ErrInsufficientStorage = errors.New("insufficient storage")
	ErrBadRequest          = errors.New("bad request")
)

// StatusCodes maps errors to their respective HTTP status codes
var StatusCodes = map[error]int{
	ErrServerError:         500,
	ErrNotImplemented:      501,
	ErrNotModified:         304,
	ErrUnauthorized:        401,
	ErrForbidden:           403,
	ErrNotFound:            404,
	ErrConflict:            409,
	ErrPreconditionFailed:  412,
	ErrTooLarge:            413,
	ErrUriTooLong:          414,
	ErrRangeNotSatisfiable: 416,
	ErrTooManyRequests:     429,
	ErrMethodNotAllowed:    405,
	ErrInsufficientStorage: 507,
	ErrBadRequest:          400,
}
