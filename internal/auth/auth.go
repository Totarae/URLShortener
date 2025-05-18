package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	cookieName   = "auth_token"
	cookieMaxAge = 365 * 24 * 60 * 60 // 1 год
)

type Auth struct {
	SecretKey string
}

func New(secret string) *Auth {
	return &Auth{SecretKey: secret}
}

// Создать подпись
func (a *Auth) sign(userID string) string {
	mac := hmac.New(sha256.New, []byte(a.SecretKey))
	mac.Write([]byte(userID))
	return hex.EncodeToString(mac.Sum(nil))
}

// Создать кукми типа: auth_token=userID:signature
func (a *Auth) issueCookie(w http.ResponseWriter) string {
	userID := uuid.NewString()
	sig := a.sign(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    fmt.Sprintf("%s:%s", userID, sig),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   cookieMaxAge,
	})
	return userID
}

// Проверим наличие и корректность куки auth
func (a *Auth) GetOrSetUserID(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return a.issueCookie(w)
	}

	parts := strings.SplitN(cookie.Value, ":", 2)
	if len(parts) != 2 || a.sign(parts[0]) != parts[1] {
		return a.issueCookie(w)
	}

	return parts[0]
}

// проверить, авторизован ли пользователь
func (a *Auth) ValidateUserID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}

	parts := strings.SplitN(cookie.Value, ":", 2)
	if len(parts) != 2 || a.sign(parts[0]) != parts[1] {
		return "", false
	}

	return parts[0], true
}

// Имитация валидной куки для тестов
func (a *Auth) SignCookieValue(userID string) string {
	sig := a.sign(userID)
	return fmt.Sprintf("%s:%s", userID, sig)
}
