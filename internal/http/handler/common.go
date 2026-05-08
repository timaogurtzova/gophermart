package handler

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	maxBodySize     = 2048
	contentTypeJSON = "application/json"
)

func hasContentType(r *http.Request, contentType string) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), contentType)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}

func writeResponse(w http.ResponseWriter, body []byte) {
	if _, err := w.Write(body); err != nil {
		log.Error().Err(err).Msg("failed to write http response")
	}
}
