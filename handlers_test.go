package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
)

func TestLoginHandler(t *testing.T) {
	// Test successful login
	data := url.Values{}
	data.Set("username", "testadmin")
	data.Set("password", "testpassword")
	t.Logf("TestLoginHandler: app.config.defaultAdminUser=%s, app.config.defaultAdminPassword=%s", app.config.defaultAdminUser, app.config.defaultAdminPassword)
	t.Logf("Attempting login with username: %s, password: %s", data.Get("username"), data.Get("password"))
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	app.LoginHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if rr.Header().Get("Location") != "/shortlinks" {
		t.Errorf("Expected redirect to /shortlinks, got %s", rr.Header().Get("Location"))
	}
	cookie := rr.Result().Cookies()[0]
	if cookie.Name != "session_token" || cookie.Value == "" {
		t.Error("Session cookie not set or empty")
	}

	// Test failed login
	data.Set("password", "wrongpassword")
	t.Logf("Attempting failed login with username: %s, password: %s", data.Get("username"), data.Get("password"))
	req, _ = http.NewRequest("POST", "/login", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()
	app.LoginHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if rr.Header().Get("Location") != "/login" {
		t.Errorf("Expected redirect to /login, got %s", rr.Header().Get("Location"))
	}
}

func TestLogoutHandler(t *testing.T) {
	req, _ := http.NewRequest("GET", "/logout", nil)
	rr := httptest.NewRecorder()

	app.LogoutHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if rr.Header().Get("Location") != "/login" {
		t.Errorf("Expected redirect to /login, got %s", rr.Header().Get("Location"))
	}
	cookie := rr.Result().Cookies()[0]
	if cookie.Name != "session_token" || cookie.MaxAge != -1 {
		t.Error("Session cookie not cleared")
	}
}

func TestFollowShortURL(t *testing.T) {
	clearTable()
	link := Link{ShortCode: "testshort", LongLink: "http://example.com"}
	_, _ = insertLink(&link)

	// Test successful redirect
	req, _ := http.NewRequest("GET", "/testshort", nil)
	rr := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Get("/{shortURL}", app.FollowShortURL)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected status %d, got %d", http.StatusTemporaryRedirect, rr.Code)
	}
	if rr.Header().Get("Location") != link.LongLink {
		t.Errorf("Expected redirect to %s, got %s", link.LongLink, rr.Header().Get("Location"))
	}

	// Test not found
	req, _ = http.NewRequest("GET", "/nonexistent", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestShortenURL(t *testing.T) {
	clearTable()

	// Test successful shorten
	data := url.Values{}
	data.Set("url", "http://new.com")
	data.Set("shortURL", "newshort")
	req, _ := http.NewRequest("POST", "/shorten", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	app.ShortenURL(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if rr.Header().Get("Location") != "/shortlinks" {
		t.Errorf("Expected redirect to /shortlinks, got %s", rr.Header().Get("Location"))
	}

	// Test invalid URL
	data.Set("url", "invalid-url")
	req, _ = http.NewRequest("POST", "/shorten", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()
	app.ShortenURL(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestShortenURLApi(t *testing.T) {
	clearTable()

	r := chi.NewRouter()
	r.With(ApiKeyAuthMiddleware).Post("/api/shorten", app.ShortenURLApi)

	// Test successful shorten
	payload := map[string]string{"url": "http://api.com", "short_code": "apishort"}
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/shorten", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "testapikey")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	// Test invalid URL
	payload = map[string]string{"url": "invalid-api-url", "short_code": "invalidapi"}
	jsonPayload, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/api/shorten", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "testapikey")

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// Test invalid JSON
	req, _ = http.NewRequest("POST", "/api/shorten", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "testapikey")

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Test unauthorized (missing API key)
	payload = map[string]string{"url": "http://unauth.com", "short_code": "unauth"}
	jsonPayload, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/api/shorten", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestDeleteShortLink(t *testing.T) {
	clearTable()
	link := Link{ShortCode: "deleteme", LongLink: "http://deleteme.com"}
	_, _ = insertLink(&link)

	// Test successful delete
	req, _ := http.NewRequest("POST", "/deleteme/delete", nil)
	rr := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/{shortURL}/delete", app.DeleteShortLink)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if rr.Header().Get("Location") != "/shortlinks" {
		t.Errorf("Expected redirect to /shortlinks, got %s", rr.Header().Get("Location"))
	}

	// Verify it's deleted
	_, err := getLinkByShortLink("deleteme")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Error("Expected link to be deleted, but it was found")
	}
}

func TestShortLinksHandler(t *testing.T) {
	clearTable()
	_, _ = insertLink(&Link{ShortCode: "page1", LongLink: "http://page1.com"})
	_, _ = insertLink(&Link{ShortCode: "page2", LongLink: "http://page2.com"})

	req, _ := http.NewRequest("GET", "/shortlinks", nil)
	rr := httptest.NewRecorder()

	app.ShortLinksHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "page1") || !strings.Contains(rr.Body.String(), "page2") {
		t.Error("Response body does not contain expected shortlinks")
	}
}
