package middleware_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/http/middleware"
)

func TestGunzipRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       io.Reader
		encoding   string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "plain body",
			body:       strings.NewReader("12345678903"),
			wantStatus: http.StatusOK,
			wantBody:   "12345678903",
		},
		{
			name:       "gzip body",
			body:       bytes.NewReader(gzipData(t, "12345678903")),
			encoding:   "gzip",
			wantStatus: http.StatusOK,
			wantBody:   "12345678903",
		},
		{
			name:       "unsupported encoding",
			body:       strings.NewReader("12345678903"),
			encoding:   "deflate",
			wantStatus: http.StatusBadRequest,
			wantBody:   "bad request\n",
		},
		{
			name:       "broken gzip",
			body:       strings.NewReader("broken"),
			encoding:   "gzip",
			wantStatus: http.StatusBadRequest,
			wantBody:   "bad request\n",
		},
		{
			name:       "multiple encodings",
			body:       strings.NewReader("12345678903"),
			encoding:   "gzip, deflate",
			wantStatus: http.StatusBadRequest,
			wantBody:   "bad request\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.GunzipRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				_, err = w.Write(body)
				require.NoError(t, err)
			}))
			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", tt.body)
			if tt.encoding != "" {
				req.Header.Set("Content-Encoding", tt.encoding)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantBody, recorder.Body.String())
		})
	}
}

func gzipData(t *testing.T, data string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write([]byte(data))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buf.Bytes()
}
