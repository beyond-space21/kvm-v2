package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"kvm_v2/Events"
	authhandler "kvm_v2/Events/auth"
	"kvm_v2/internal/repository"
	"kvm_v2/services/auth"
	"kvm_v2/services/config"
	"kvm_v2/services/db"
)

func testDB(t *testing.T) *httptest.Server {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://lms:lms@localhost:5432/lms?sslmode=disable"
	}

	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	usersRepo := repository.NewUserRepository(database)
	_ = authhandler.BootstrapAdmin(context.Background(), usersRepo, "test-admin@lms.local", "testpass123")

	authService, err := auth.NewService("test-jwt-secret-key")
	if err != nil {
		t.Fatalf("auth: %v", err)
	}

	srv := httptest.NewServer(events.NewRouter(events.Dependencies{
		DB:   database,
		Auth: authService,
		Cfg:  config.Config{},
	}))
	t.Cleanup(srv.Close)
	return srv
}

func login(t *testing.T, srv *httptest.Server, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var result struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode login: %v", err)
		}
		return result.Token
	}

	// Bootstrap first admin if none exists
	bootstrapBody, _ := json.Marshal(map[string]string{
		"email": email, "password": password, "name": "Test Admin",
	})
	bresp, err := http.Post(srv.URL+"/api/v1/auth/bootstrap", "application/json", bytes.NewReader(bootstrapBody))
	if err != nil {
		t.Fatalf("bootstrap request: %v", err)
	}
	defer bresp.Body.Close()
	if bresp.StatusCode != http.StatusCreated {
		t.Fatalf("bootstrap/login failed: login=%d bootstrap=%d", resp.StatusCode, bresp.StatusCode)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(bresp.Body).Decode(&result); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	return result.Token
}

func authRequest(t *testing.T, method, url, token string, payload any) *http.Response {
	t.Helper()
	var body *bytes.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestHealth(t *testing.T) {
	srv := testDB(t)
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLoginAndAcademicFlow(t *testing.T) {
	srv := testDB(t)
	token := login(t, srv, "test-admin@lms.local", "testpass123")

	// List seeded classes
	resp := authRequest(t, http.MethodGet, srv.URL+"/api/v1/classes", token, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("classes status: %d", resp.StatusCode)
	}

	// Create academic year
	yearResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/academic-years", token, map[string]any{
		"name":       fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()),
		"start_date": "2026-04-01",
		"end_date":   "2027-03-31",
		"is_active":  true,
	})
	defer yearResp.Body.Close()
	if yearResp.StatusCode != http.StatusCreated {
		t.Fatalf("create year status: %d", yearResp.StatusCode)
	}

	// Create staff
	staffResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/staff", token, map[string]string{
		"email": fmt.Sprintf("teacher-%d@lms.local", time.Now().UnixNano()), "password": "teach123", "name": "Test Teacher",
	})
	defer staffResp.Body.Close()
	if staffResp.StatusCode != http.StatusCreated {
		t.Fatalf("create staff status: %d", staffResp.StatusCode)
	}

	// Create student
	studentResp := authRequest(t, http.MethodPost, srv.URL+"/api/v1/students", token, map[string]string{
		"email": fmt.Sprintf("student-%d@lms.local", time.Now().UnixNano()), "password": "stud123", "name": "Test Student",
	})
	defer studentResp.Body.Close()
	if studentResp.StatusCode != http.StatusCreated {
		t.Fatalf("create student status: %d", studentResp.StatusCode)
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	srv := testDB(t)
	resp, err := http.Get(srv.URL + "/api/v1/classes")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestPasswordHashing(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(hash, "secret") {
		t.Fatal("password should match")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatal("wrong password should not match")
	}
}
