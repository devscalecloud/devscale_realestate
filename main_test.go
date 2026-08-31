package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

var testTmplOnce sync.Once

func init() {
	appLog = log.New(ioutil.Discard, "", 0)
}

// loadTestTemplates parses templates/*.html once so handlers that render
// HTML (login, register, dashboard) can be exercised directly in tests.
func loadTestTemplates(t *testing.T) {
	t.Helper()
	testTmplOnce.Do(func() {
		var err error
		tmpl, err = template.New("").Funcs(template.FuncMap{
			"initials": initials,
			"fullName": fullName,
		}).ParseGlob("templates/*.html")
		if err != nil {
			t.Fatalf("failed to parse templates: %v", err)
		}
	})
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{name: "large amount", input: 450000, want: "$450,000.00"},
		{name: "small amount", input: 850.5, want: "$850.50"},
		{name: "zero", input: 0, want: "$0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMoney(tt.input)
			if got != tt.want {
				t.Fatalf("formatMoney(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPropertyByID(t *testing.T) {
	p, ok := propertyByID("p1")
	if !ok {
		t.Fatal("propertyByID(\"p1\") not found")
	}
	if p.Title != "Sunset Villa" {
		t.Fatalf("propertyByID(\"p1\").Title = %q, want %q", p.Title, "Sunset Villa")
	}

	_, ok = propertyByID("does-not-exist")
	if ok {
		t.Fatal("propertyByID(\"does-not-exist\") should not be found")
	}
}

func TestSeededUsersSharePassword(t *testing.T) {
	for _, username := range []string{"dapo", "iyiola", "abayomi", "chimezie", "peter", "juwon", "fisayo"} {
		mu.RLock()
		user, ok := users[username]
		mu.RUnlock()
		if !ok {
			t.Fatalf("expected seeded user %q to exist", username)
		}
		if user.Password != demoPassword {
			t.Fatalf("user %q password = %q, want %q", username, user.Password, demoPassword)
		}
	}
}

func TestBuyHandlerUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/buy", strings.NewReader("property_id=p1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	buyHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestBuyHandlerSuccess(t *testing.T) {
	token := "test-buy-token"
	mu.Lock()
	sessions[token] = Session{Username: "dapo", ExpiresAt: time.Now().Add(time.Hour)}
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(sessions, token)
		mu.Unlock()
	}()

	req := httptest.NewRequest(http.MethodPost, "/buy", strings.NewReader("property_id=p1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	buyHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); !strings.Contains(body, "Sunset Villa has been bought") {
		t.Fatalf("body = %q, want it to mention the purchased property", body)
	}
}

func TestRentHandlerSuccess(t *testing.T) {
	token := "test-rent-token"
	mu.Lock()
	sessions[token] = Session{Username: "dapo", ExpiresAt: time.Now().Add(time.Hour)}
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(sessions, token)
		mu.Unlock()
	}()

	req := httptest.NewRequest(http.MethodPost, "/rent", strings.NewReader("property_id=p2"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	rentHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if body := rr.Body.String(); !strings.Contains(body, "Maple Court Apartment has been rented successfully") {
		t.Fatalf("body = %q, want it to mention the rented property", body)
	}
}

func TestBuyHandlerUnknownProperty(t *testing.T) {
	token := "test-unknown-token"
	mu.Lock()
	sessions[token] = Session{Username: "dapo", ExpiresAt: time.Now().Add(time.Hour)}
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(sessions, token)
		mu.Unlock()
	}()

	req := httptest.NewRequest(http.MethodPost, "/buy", strings.NewReader("property_id=does-not-exist"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rr := httptest.NewRecorder()

	buyHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestLoginHandlerInvalidCredentials(t *testing.T) {
	loadTestTemplates(t)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("username=dapo&password=wrong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	loginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == "session_token" {
			t.Fatal("expected no session cookie on failed login, got one")
		}
	}
}

func TestLoginHandlerValidCredentials(t *testing.T) {
	loadTestTemplates(t)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("username=dapo&password=welcome%2B1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	loginHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}

	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "session_token" && c.Value != "" {
			found = true
			mu.Lock()
			delete(sessions, c.Value)
			mu.Unlock()
		}
	}
	if !found {
		t.Fatal("expected a session_token cookie to be set on successful login")
	}
}
