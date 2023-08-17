package rmsgo

import (
	"net/http"
	"time"
)

type LoggingResponseWriter struct {
	http.ResponseWriter // compose original ResponseWriter
	Status, Size        int
}

func (lrw *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.Size += size
	return size, err
}

func (lrw *LoggingResponseWriter) WriteHeader(statusCode int) {
	lrw.ResponseWriter.WriteHeader(statusCode)
	lrw.Status = statusCode
}

func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{ResponseWriter: w}
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := NewLoggingResponseWriter(w)

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		logger.Info("Request", "method", r.Method, "uri", r.RequestURI, "headers", r.Header, "duration", duration, "status", lrw.Status, "size", lrw.Size)
	})
}

