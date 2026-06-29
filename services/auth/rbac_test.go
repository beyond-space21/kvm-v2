package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kvm_v2/internal/models"
	"kvm_v2/services/auth"
)

func TestRequireRole(t *testing.T) {
	svc, err := auth.NewService("test-secret-key-for-rbac")
	if err != nil {
		t.Fatal(err)
	}

	token, err := svc.SignToken("user-1", models.RoleStaff, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	handler := svc.Middleware(auth.RequireAdmin()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Staff should be forbidden for admin-only
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	adminToken, _ := svc.SignToken("admin-1", models.RoleSuperAdmin, time.Hour)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d", rec.Code)
	}
}
