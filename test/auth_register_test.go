package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"
	"sign_flow_project/internal/router"

	"github.com/gin-gonic/gin"
)

type authRegisterResult struct {
	User struct {
		ID     uint   `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Avatar string `json:"avatar"`
		Status string `json:"status"`
	} `json:"user"`
	AccessToken string `json:"accessToken"`
}

func TestAuthRegister_OK(t *testing.T) {
	engine := setupAuthRegisterTestEngine(t)

	users := []struct {
		name            string
		email           string
		password        string
		normalizedEmail string
	}{
		{name: "Alice Johnson", email: "Alice@Example.com", password: "password123", normalizedEmail: "alice@example.com"},
		{name: "David Miller", email: "David.Miller@Example.com", password: "davidPass123", normalizedEmail: "david.miller@example.com"},
		{name: "Emma Wilson", email: "EMMA.WILSON@EXAMPLE.COM", password: "emmaPass123", normalizedEmail: "emma.wilson@example.com"},
		{name: "Frank Lee", email: " frank.lee@example.com ", password: "frankPass123", normalizedEmail: "frank.lee@example.com"},
		{name: "Grace Chen", email: "Grace.Chen@Example.com", password: "gracePass123", normalizedEmail: "grace.chen@example.com"},
	}

	for _, u := range users {
		rec := performJSON(engine, http.MethodPost, "/api/v1/auth/register", map[string]any{
			"name":     u.name,
			"email":    u.email,
			"password": u.password,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("register %s status=%d body=%s", u.email, rec.Code, rec.Body.String())
		}

		var wrapper apiResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
			t.Fatalf("unmarshal register wrapper failed: %v", err)
		}
		if wrapper.Code != http.StatusOK {
			t.Fatalf("register %s code=%d msg=%s", u.email, wrapper.Code, wrapper.Msg)
		}

		var data authRegisterResult
		if err := json.Unmarshal(wrapper.Data, &data); err != nil {
			t.Fatalf("unmarshal register data failed: %v", err)
		}
		if data.User.ID == 0 {
			t.Fatalf("register %s expect user.id > 0", u.email)
		}
		if data.User.Email != u.normalizedEmail {
			t.Fatalf("register %s expect normalized email=%s, got %q", u.email, u.normalizedEmail, data.User.Email)
		}
		if data.AccessToken == "" {
			t.Fatalf("register %s expect accessToken not empty", u.email)
		}
	}
}

func TestAuthRegister_DuplicateEmail_BadRequest(t *testing.T) {
	engine := setupAuthRegisterTestEngine(t)

	first := performJSON(engine, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Bob",
		"email":    "bob@example.com",
		"password": "password123",
	})
	if first.Code != http.StatusOK {
		t.Fatalf("first register status=%d body=%s", first.Code, first.Body.String())
	}

	second := performJSON(engine, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Bob2",
		"email":    "  BOB@example.com  ",
		"password": "password456",
	})
	if second.Code != http.StatusBadRequest {
		t.Fatalf("duplicate email register status=%d body=%s", second.Code, second.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(second.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal duplicate wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusBadRequest {
		t.Fatalf("duplicate email expect code=400, got %d", wrapper.Code)
	}
	if wrapper.Msg != "email already registered" {
		t.Fatalf("duplicate email expect msg=email already registered, got %q", wrapper.Msg)
	}
}

func TestAuthRegister_ShortPassword_BadRequest(t *testing.T) {
	engine := setupAuthRegisterTestEngine(t)

	rec := performJSON(engine, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Charlie",
		"email":    "charlie@example.com",
		"password": "1234567",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("short password register status=%d body=%s", rec.Code, rec.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal short password wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusBadRequest {
		t.Fatalf("short password expect code=400, got %d", wrapper.Code)
	}
	if wrapper.Msg != "password must be at least 8 characters" {
		t.Fatalf("short password expect msg=password must be at least 8 characters, got %q", wrapper.Msg)
	}
}

func setupAuthRegisterTestEngine(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	gdb, err := db.PostgresSetup()
	if err != nil {
		t.Fatalf("init postgres failed: %v", err)
	}
	if err := gdb.AutoMigrate(&model.WorkflowSignerModel{}); err != nil {
		t.Fatalf("migrate workflow signer failed: %v", err)
	}
	cleanupTables(t, gdb)

	engine := gin.New()
	router.RegisterRoutes(engine)
	return engine
}
