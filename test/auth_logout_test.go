package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthLogout_OK_ThenTokenRevoked(t *testing.T) {
	engine := setupAuthRegisterTestEngine(t)
	token := registerAndGetToken(t, engine, "Logout User", uniqueEmail("logout-user@test.local"), "password123")

	logoutRes := performJSONWithAuth(engine, http.MethodPost, "/api/v1/auth/logout", map[string]any{}, token)
	if logoutRes.Code != http.StatusOK {
		t.Fatalf("logout status=%d body=%s", logoutRes.Code, logoutRes.Body.String())
	}

	var logoutWrapper apiResponse
	if err := json.Unmarshal(logoutRes.Body.Bytes(), &logoutWrapper); err != nil {
		t.Fatalf("unmarshal logout wrapper failed: %v", err)
	}
	if logoutWrapper.Code != http.StatusOK {
		t.Fatalf("logout expect code=200, got %d", logoutWrapper.Code)
	}
	if logoutWrapper.Msg != "logout success" {
		t.Fatalf("logout expect msg=logout success, got %q", logoutWrapper.Msg)
	}

	meRes := performAuthNoBody(engine, http.MethodGet, "/api/v1/auth/me", token)
	if meRes.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout expect status=401, got %d body=%s", meRes.Code, meRes.Body.String())
	}

	var meWrapper apiResponse
	if err := json.Unmarshal(meRes.Body.Bytes(), &meWrapper); err != nil {
		t.Fatalf("unmarshal me wrapper failed: %v", err)
	}
	if meWrapper.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout expect code=401, got %d", meWrapper.Code)
	}
	if meWrapper.Msg != "token has been revoked" {
		t.Fatalf("me after logout expect msg=token has been revoked, got %q", meWrapper.Msg)
	}

	logoutAgainRes := performJSONWithAuth(engine, http.MethodPost, "/api/v1/auth/logout", map[string]any{}, token)
	if logoutAgainRes.Code != http.StatusUnauthorized {
		t.Fatalf("logout again expect status=401, got %d body=%s", logoutAgainRes.Code, logoutAgainRes.Body.String())
	}
}

func TestAuthLogout_MissingAuthorization_Unauthorized(t *testing.T) {
	engine := setupAuthRegisterTestEngine(t)

	rec := performJSON(engine, http.MethodPost, "/api/v1/auth/logout", map[string]any{})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("logout without token expect status=401, got %d body=%s", rec.Code, rec.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal logout without token wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusUnauthorized {
		t.Fatalf("logout without token expect code=401, got %d", wrapper.Code)
	}
	if wrapper.Msg != "missing authorization" {
		t.Fatalf("logout without token expect msg=missing authorization, got %q", wrapper.Msg)
	}
}

func registerAndGetToken(t *testing.T, engine http.Handler, name, email, password string) string {
	t.Helper()

	rec := performJSON(engine, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     name,
		"email":    email,
		"password": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("register for logout test status=%d body=%s", rec.Code, rec.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal register wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("register for logout test code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data authResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal register data failed: %v", err)
	}
	if data.AccessToken == "" {
		t.Fatalf("register for logout test got empty accessToken")
	}
	return data.AccessToken
}

func performAuthNoBody(r http.Handler, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}
