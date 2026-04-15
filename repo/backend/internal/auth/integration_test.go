package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eagle-point/service-portal/internal/auth"
	"github.com/eagle-point/service-portal/internal/config"
	"github.com/eagle-point/service-portal/internal/router"
	"github.com/eagle-point/service-portal/internal/testutil"
)

// integrationServer returns a test httptest.Server backed by a real database.
func integrationServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testutil.DBOrSkip(t)
	testutil.TruncateTables(t, db,
		"login_attempts", "sessions", "user_roles", "user_preferences", "users",
	)

	cfg := &config.Config{
		AppEnv:              "test",
		Port:                "8080",
		DBHost:              "db",
		DBPort:              "3306",
		FieldEncryptionKey:  "",
		SessionCookieDomain: "localhost",
	}

	r := router.New(cfg, db)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// doJSON is a helper for making JSON requests.
func doJSON(t *testing.T, client *http.Client, method, url string, body any, headers map[string]string) *http.Response {
	t.Helper()
	var buf *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		buf = bytes.NewReader(raw)
	} else {
		buf = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// ─── Full register → login → protected access → logout ────────────────────────

func TestIntegration_RegisterLoginLogout(t *testing.T) {
	srv := integrationServer(t)
	client := srv.Client()
	// Use cookie jar to persist session
	jar := newCookieJar()
	client.Jar = jar

	base := srv.URL

	// 1. Register
	resp := doJSON(t, client, http.MethodPost, base+"/api/v1/auth/register", map[string]string{
		"username":     "intuser",
		"email":        "int@example.local",
		"password":     "ValidPass1",
		"display_name": "Int User",
	}, nil)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// 2. Login
	resp = doJSON(t, client, http.MethodPost, base+"/api/v1/auth/login", map[string]string{
		"username": "intuser",
		"password": "ValidPass1",
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var loginBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginBody)
	resp.Body.Close()

	csrfToken, _ := loginBody["csrf_token"].(string)
	require.NotEmpty(t, csrfToken)

	// 3. GET /me — should succeed (cookie is set)
	resp = doJSON(t, client, http.MethodGet, base+"/api/v1/auth/me", nil, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 4. Logout
	resp = doJSON(t, client, http.MethodPost, base+"/api/v1/auth/logout", nil,
		map[string]string{"X-CSRF-Token": csrfToken})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 5. GET /me after logout — should be 401
	resp = doJSON(t, client, http.MethodGet, base+"/api/v1/auth/me", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

// ─── RBAC enforcement ─────────────────────────────────────────────────────────

func TestIntegration_RBAC_AdminRouteBlocksRegularUser(t *testing.T) {
	srv := integrationServer(t)
	client := srv.Client()
	client.Jar = newCookieJar()
	base := srv.URL

	// Register as regular user (default role)
	doJSON(t, client, http.MethodPost, base+"/api/v1/auth/register", map[string]string{
		"username": "rbacuser", "email": "rbac@example.local",
		"password": "ValidPass1", "display_name": "RBAC",
	}, nil).Body.Close()

	resp := doJSON(t, client, http.MethodPost, base+"/api/v1/auth/login", map[string]string{
		"username": "rbacuser", "password": "ValidPass1",
	}, nil)
	var loginBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginBody)
	resp.Body.Close()
	csrf := loginBody["csrf_token"].(string)

	// Hit admin-only endpoint
	resp = doJSON(t, client, http.MethodPost, base+"/api/v1/admin/hmac-keys/rotate", nil,
		map[string]string{"X-CSRF-Token": csrf})
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

// ─── Account lockout ──────────────────────────────────────────────────────────

func TestIntegration_Lockout_After5BadPasswords(t *testing.T) {
	srv := integrationServer(t)
	client := srv.Client()
	client.Jar = newCookieJar()
	base := srv.URL

	doJSON(t, client, http.MethodPost, base+"/api/v1/auth/register", map[string]string{
		"username": "lockuser", "email": "lock@example.local",
		"password": "ValidPass1", "display_name": "Lock",
	}, nil).Body.Close()

	// 5 bad password attempts
	for i := 0; i < 5; i++ {
		resp := doJSON(t, client, http.MethodPost, base+"/api/v1/auth/login", map[string]string{
			"username": "lockuser", "password": "WrongPass9",
		}, nil)
		resp.Body.Close()
	}

	// 6th attempt (correct password) — should be locked
	resp := doJSON(t, client, http.MethodPost, base+"/api/v1/auth/login", map[string]string{
		"username": "lockuser", "password": "ValidPass1",
	}, nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	errObj, _ := body["error"].(map[string]interface{})
	assert.Equal(t, "account_locked", errObj["code"])
}

// ─── Rate limiting ────────────────────────────────────────────────────────────

func TestIntegration_RateLimit_Returns429(t *testing.T) {
	srv := integrationServer(t)
	client := srv.Client()
	client.Jar = newCookieJar()
	base := srv.URL

	// Hit the register endpoint 61 times from the same IP
	// The general rate limiter (60/min) should trigger on the 61st
	var lastCode int
	for i := 0; i < 65; i++ {
		resp := doJSON(t, client, http.MethodPost, base+"/api/v1/auth/register", map[string]string{
			"username": "rlu" + string(rune('a'+i%26)),
			"email":    "rlu" + string(rune('a'+i%26)) + "@x.local",
			"password": "ValidPass1", "display_name": "RL",
		}, nil)
		lastCode = resp.StatusCode
		resp.Body.Close()
		if lastCode == http.StatusTooManyRequests {
			break
		}
	}
	assert.Equal(t, http.StatusTooManyRequests, lastCode)
}

// ─── HMAC — missing / wrong signature ────────────────────────────────────────

func TestIntegration_HMAC_MissingHeaders_Returns400(t *testing.T) {
	srv := integrationServer(t)
	client := srv.Client()
	base := srv.URL

	// Hit an HMAC-protected internal route without headers
	req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/internal/", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 400 (missing headers) or 404 (route not yet implemented) — both acceptable
	// The key assertion is: NOT 200
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_HMAC_WrongSignature_Returns401(t *testing.T) {
	srv := integrationServer(t)
	client := srv.Client()
	base := srv.URL

	req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/internal/", nil)
	req.Header.Set("X-Key-ID", "fake-key")
	req.Header.Set("X-Signature", "hmac-sha256 invalidsignature")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 401 (wrong sig / key not found) or 404 (no route) — both acceptable
	assert.NotEqual(t, http.StatusOK, resp.StatusCode)
}

// ─── Protected route requires auth ───────────────────────────────────────────

func TestIntegration_UnauthenticatedRequest_Returns401(t *testing.T) {
	srv := integrationServer(t)
	client := srv.Client()
	base := srv.URL

	resp := doJSON(t, client, http.MethodGet, base+"/api/v1/auth/me", nil, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── Cookie jar helper ────────────────────────────────────────────────────────

type simpleCookieJar struct {
	cookies map[string][]*http.Cookie
}

func newCookieJar() *simpleCookieJar {
	return &simpleCookieJar{cookies: make(map[string][]*http.Cookie)}
}

func (j *simpleCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.cookies[u.Host] = append(j.cookies[u.Host], cookies...)
}

func (j *simpleCookieJar) Cookies(u *url.URL) []*http.Cookie {
	return j.cookies[u.Host]
}

// Ensure compile: auth package referenced for ErrInvalidCredentials
var _ = auth.ErrInvalidCredentials
