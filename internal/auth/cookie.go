package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultCookieName = "user_id"
const cookieTTL = 365 * 24 * time.Hour

// ErrUserIDMissing возвращается, когда запрос не содержит корректный user ID.
var ErrUserIDMissing = errors.New("user id is missing")

type cookieState int

const (
	cookieStateValid cookieState = iota
	cookieStateMissing
	cookieStateInvalid
	cookieStateEmptyUserID
)

// Authenticator управляет подписью и валидацией пользовательской cookie.
type Authenticator struct {
	cookieName string
	secret     []byte
}

// NewAuthenticator создаёт новый helper для пользовательской cookie.
func NewAuthenticator(secret []byte) (*Authenticator, error) {
	if len(secret) == 0 {
		return nil, errors.New("auth secret is empty")
	}

	secretCopy := make([]byte, len(secret))
	copy(secretCopy, secret)

	return &Authenticator{
		cookieName: defaultCookieName,
		secret:     secretCopy,
	}, nil
}

// NewRandomSecret генерирует симметричный ключ для подписи cookie.
func NewRandomSecret(size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("invalid secret size")
	}

	secret := make([]byte, size)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}

	return secret, nil
}

// SetUserID устанавливает подписанную cookie с ID пользователя.
func (a *Authenticator) SetUserID(w http.ResponseWriter, r *http.Request, userID int64) error {
	cookie, err := a.NewCookieForRequest(userID, r)
	if err != nil {
		return err
	}

	http.SetCookie(w, cookie)
	return nil
}

// UserID возвращает ID пользователя из подписанной cookie.
func (a *Authenticator) UserID(r *http.Request) (int64, error) {
	userID, state, err := a.resolveUserID(r)
	if err != nil {
		return 0, err
	}

	if state != cookieStateValid {
		return 0, ErrUserIDMissing
	}

	return userID, nil
}

// NewCookie создаёт подписанную cookie для указанного user ID.
func (a *Authenticator) NewCookie(userID int64) (*http.Cookie, error) {
	return a.newCookie(userID, false)
}

// NewCookieForRequest создаёт подписанную cookie для запроса с учётом transport security.
func (a *Authenticator) NewCookieForRequest(userID int64, r *http.Request) (*http.Cookie, error) {
	return a.newCookie(userID, isSecureRequest(r))
}

func (a *Authenticator) resolveUserID(r *http.Request) (int64, cookieState, error) {
	cookie, err := r.Cookie(a.cookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return 0, cookieStateMissing, nil
		}
		return 0, cookieStateInvalid, nil
	}

	userID, state := a.decode(cookie.Value)
	return userID, state, nil
}

func (a *Authenticator) newCookie(userID int64, secure bool) (*http.Cookie, error) {
	encodedValue, err := a.encode(userID)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(cookieTTL)

	return &http.Cookie{
		Name:     a.cookieName,
		Value:    encodedValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   int(cookieTTL.Seconds()),
		Expires:  expiresAt,
	}, nil
}

func (a *Authenticator) encode(userID int64) (string, error) {
	if userID <= 0 {
		return "", ErrUserIDMissing
	}

	payload := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(userID, 10)))

	signature, err := a.sign(payload)
	if err != nil {
		return "", err
	}

	return payload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (a *Authenticator) decode(value string) (int64, cookieState) {
	payload, signature, ok := strings.Cut(value, ".")
	if !ok {
		return 0, cookieStateInvalid
	}

	expectedSignature, err := a.sign(payload)
	if err != nil {
		return 0, cookieStateInvalid
	}

	actualSignature, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return 0, cookieStateInvalid
	}

	if !hmac.Equal(actualSignature, expectedSignature) {
		return 0, cookieStateInvalid
	}

	userIDBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return 0, cookieStateInvalid
	}

	if len(userIDBytes) == 0 {
		return 0, cookieStateEmptyUserID
	}

	userID, err := strconv.ParseInt(string(userIDBytes), 10, 64)
	if err != nil || userID <= 0 {
		return 0, cookieStateInvalid
	}

	return userID, cookieStateValid
}

func (a *Authenticator) sign(payload string) ([]byte, error) {
	mac := hmac.New(sha256.New, a.secret)
	if _, err := mac.Write([]byte(payload)); err != nil {
		return nil, err
	}

	return mac.Sum(nil), nil
}

func isSecureRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	if r.TLS != nil {
		return true
	}

	if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}

	return strings.EqualFold(r.Header.Get("X-Forwarded-Ssl"), "on")
}
