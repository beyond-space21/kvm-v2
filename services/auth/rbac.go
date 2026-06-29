package auth

import (
	"net/http"

	"kvm_v2/internal/httpx"
	"kvm_v2/internal/models"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				httpx.WriteError(w, httpx.Unauthorized("authentication required"))
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				httpx.WriteError(w, httpx.Forbidden("insufficient permissions for this action"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole(models.RoleSuperAdmin)
}

func RequireStaffOrAdmin() func(http.Handler) http.Handler {
	return RequireRole(models.RoleSuperAdmin, models.RoleStaff)
}
