package rmsgo

import "net/http"

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
