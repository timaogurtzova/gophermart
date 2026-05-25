package middleware

import (
	"compress/gzip"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

const gzipEncoding = "gzip"

var compressibleContentTypes = map[string]struct{}{
	"application/json": {},
	"text/html":        {},
}

// gzipBodyReadCloser закрывает и gzip.Reader, и исходное тело запроса.
type gzipBodyReadCloser struct {
	reader *gzip.Reader
	body   io.ReadCloser
}

func (g *gzipBodyReadCloser) Read(p []byte) (int, error) {
	return g.reader.Read(p)
}

func (g *gzipBodyReadCloser) Close() error {
	if err := g.reader.Close(); err != nil {
		if closeErr := g.body.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}

		return err
	}

	return g.body.Close()
}

// GunzipRequest распаковывает тело HTTP-запроса с Content-Encoding: gzip.
func GunzipRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoding, ok := requestContentEncoding(r.Header.Values("Content-Encoding"))
		if !ok {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if encoding == "" {
			next.ServeHTTP(w, r)
			return
		}

		body, err := newGzipBody(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		r.Body = body
		r.Header.Del("Content-Encoding")
		r.Header.Del("Content-Length")
		r.ContentLength = -1

		next.ServeHTTP(w, r)
	})
}

// GzipResponse сжимает HTTP-ответы, если клиент поддерживает gzip.
func GzipResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !acceptsGzip(r.Header.Values("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		ww := &gzipResponseWriter{ResponseWriter: w}
		defer func() {
			if err := ww.Close(); err != nil {
				log.Error().Err(err).Msg("failed to close gzip response writer")
			}
		}()

		next.ServeHTTP(ww, r)
	})
}

func requestContentEncoding(values []string) (string, bool) {
	encodings := parseContentEncodings(values)
	if len(encodings) == 0 {
		return "", true
	}

	if len(encodings) == 1 && encodings[0] == gzipEncoding {
		return gzipEncoding, true
	}

	return "", false
}

func newGzipBody(body io.ReadCloser) (io.ReadCloser, error) {
	reader, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}

	return &gzipBodyReadCloser{
		reader: reader,
		body:   body,
	}, nil
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	compress    bool
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.wroteHeader = true
	if isCompressibleContentType(w.Header().Get("Content-Type")) {
		w.compress = true
		w.writer = gzip.NewWriter(w.ResponseWriter)
		w.Header().Set("Content-Encoding", gzipEncoding)
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if !w.compress {
		return w.ResponseWriter.Write(data)
	}

	return w.writer.Write(data)
}

func (w *gzipResponseWriter) Close() error {
	if w.writer == nil {
		return nil
	}

	return w.writer.Close()
}

func parseContentEncodings(values []string) []string {
	var encodings []string

	for _, value := range values {
		for _, encoding := range strings.Split(value, ",") {
			encoding = strings.TrimSpace(strings.ToLower(encoding))
			if encoding == "" {
				continue
			}
			encodings = append(encodings, encoding)
		}
	}

	return encodings
}

func acceptsGzip(values []string) bool {
	for _, encoding := range parseContentEncodings(values) {
		if encoding == gzipEncoding {
			return true
		}
	}

	return false
}

func isCompressibleContentType(value string) bool {
	contentType, _, err := mime.ParseMediaType(value)
	if err != nil {
		contentType = value
	}

	_, ok := compressibleContentTypes[strings.ToLower(contentType)]
	return ok
}
