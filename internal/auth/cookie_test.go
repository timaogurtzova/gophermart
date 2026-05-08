package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/auth"
)

func TestNewCookieForRequestCreatesPersistentCookie(t *testing.T) {
	authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	cookie, err := authenticator.NewCookieForRequest(42, req)
	require.NoError(t, err)

	assert.Equal(t, "user_id", cookie.Name)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.False(t, cookie.Secure)
	assert.Equal(t, 365*24*60*60, cookie.MaxAge)
	assert.WithinDuration(t, time.Now().Add(365*24*time.Hour), cookie.Expires, 2*time.Second)
}

func TestNewCookieForRequestMarksHTTPSCookieAsSecure(t *testing.T) {
	authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	cookie, err := authenticator.NewCookieForRequest(42, req)
	require.NoError(t, err)

	assert.True(t, cookie.Secure)
}

func TestSetUserIDAndUserID(t *testing.T) {
	authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	rec := httptest.NewRecorder()

	require.NoError(t, authenticator.SetUserID(rec, req, 42))

	res := rec.Result()
	defer res.Body.Close()
	require.NotEmpty(t, res.Cookies())

	reqWithCookie := httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	reqWithCookie.AddCookie(res.Cookies()[0])

	userID, err := authenticator.UserID(reqWithCookie)
	require.NoError(t, err)
	assert.Equal(t, int64(42), userID)
}

func TestUserIDReturnsErrorForInvalidCookie(t *testing.T) {
	authenticator, err := auth.NewAuthenticator([]byte("01234567890123456789012345678901"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "broken"})

	userID, err := authenticator.UserID(req)
	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrUserIDMissing)
	assert.Zero(t, userID)
}
