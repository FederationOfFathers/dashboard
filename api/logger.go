package api

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseWriter struct {
	status int
	http.ResponseWriter
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

type httpLogger struct {
}

func (h *httpLogger) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	var nw = &responseWriter{status: 200, ResponseWriter: w}
	next(nw, r)
	remoteAddress := r.RemoteAddr
	if addr := r.Header.Get("X-Real-IP"); addr != "" {
		remoteAddress = addr
	}
	Logger.Info(
		"HTTP Request",
		zap.String("uri", r.RequestURI),
		zap.Int("http_status", nw.status),
		zap.String("username", getSlackUserName(r)),
		zap.String("remote_address", remoteAddress),
		zap.String("method", r.Method),
		zap.Int64("content_length", r.ContentLength),
		zap.Float64("response_time", time.Now().Sub(start).Seconds()))
}
