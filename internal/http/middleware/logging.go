package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// Logging записывает параметры обработанного HTTP-запроса.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := newResponseStatsWriter(w)

		next.ServeHTTP(ww, r)

		log.Info().
			Str("uri", r.RequestURI).
			Str("method", r.Method).
			Str("duration", time.Since(start).String()).
			Int("status", ww.statusCode()).
			Int("size", ww.bytesWritten()).
			Msg("HTTP request completed")
	})
}

type responseStatsWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func newResponseStatsWriter(w http.ResponseWriter) *responseStatsWriter {
	return &responseStatsWriter{ResponseWriter: w}
}

func (w *responseStatsWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatsWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func (w *responseStatsWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}

	return w.status
}

func (w *responseStatsWriter) bytesWritten() int {
	return w.bytes
}
