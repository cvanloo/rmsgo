package rmsgo

import "errors"

// Sentinel error values
var (
	ServerError = errors.New("internal server error")
)

// StatusCodes maps errors to their respective HTTP status codes
var StatusCodes = map[error]int{
	ServerError: 500,
}
