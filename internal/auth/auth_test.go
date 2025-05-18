package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestSignAndValidate(t *testing.T) {
	a := auth.New("test-secret")
	userID := "user123"
	signed := a.SignCookieValue(userID)

	parts := strings.SplitN(signed, ":", 2)
	assert.Len(t, parts, 2)
	assert.Equal(t, userID, parts[0])
	assert.Equal(t, a.SignCookieValue(userID), signed)
}

func TestIssueCookie(t *testing.T) {
	a := auth.New("test-secret")
	rec := httptest.NewRecorder()
	userID := a.GetOrSetUserID(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.NotEmpty(t, userID)

	resp := rec.Result()
	defer resp.Body.Close()

	cookies := resp.Cookies()
	assert.NotEmpty(t, cookies)
	assert.Equal(t, "auth_token", cookies[0].Name)
}

func TestGetOrSetUserID_Valid(t *testing.T) {
	a := auth.New("test-secret")
	userID := "test-user"
	signed := a.SignCookieValue(userID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: signed,
	})

	rec := httptest.NewRecorder()
	gotID := a.GetOrSetUserID(rec, req)
	assert.Equal(t, userID, gotID)
}

func TestGetOrSetUserID_Invalid(t *testing.T) {
	a := auth.New("test-secret")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "invalidformat",
	})

	rec := httptest.NewRecorder()
	userID := a.GetOrSetUserID(rec, req)
	assert.NotEmpty(t, userID)
	assert.NotEqual(t, "invalidformat", userID)
}

func TestValidateUserID(t *testing.T) {
	a := auth.New("test-secret")
	userID := "valid-user"
	signed := a.SignCookieValue(userID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: signed,
	})

	id, ok := a.ValidateUserID(req)
	assert.True(t, ok)
	assert.Equal(t, userID, id)
}

func TestValidateUserID_Invalid(t *testing.T) {
	a := auth.New("test-secret")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "someuser:bad-signature",
	})

	id, ok := a.ValidateUserID(req)
	assert.False(t, ok)
	assert.Empty(t, id)
}
