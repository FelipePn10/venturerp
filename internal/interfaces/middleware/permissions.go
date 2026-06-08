package middleware

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Permission scopes — finer-grained than the coarse ADMIN/USER role check.
// New sensitive endpoints can gate on a specific scope while reusing the role
// the JWT already carries, giving real granularity without a per-route rewrite.
const (
	PermRead            = "read"             // any read endpoint
	PermWrite           = "write"            // create/update of operational data
	PermPlanningRun     = "planning:run"     // trigger MRP/CRP/APS
	PermPurchaseApprove = "purchase:approve" // approve purchase suggestions/orders
	PermFiscalAuthorize = "fiscal:authorize" // authorize/cancel NF-e
	PermFinancialManage = "financial:manage" // accounts payable/receivable, CNAB
	PermItemActivate    = "item:activate"    // activate items (engineering gate)
	PermAdmin           = "admin"            // administrative configuration
)

// rolePermissions maps a JWT role to the scopes it is granted. ADMIN gets
// everything; USER gets all operational scopes but not admin; VIEWER is
// read-only. Unknown roles get nothing.
var rolePermissions = map[string]map[string]struct{}{
	"ADMIN":  asSet(PermRead, PermWrite, PermPlanningRun, PermPurchaseApprove, PermFiscalAuthorize, PermFinancialManage, PermItemActivate, PermAdmin),
	"USER":   asSet(PermRead, PermWrite, PermPlanningRun, PermPurchaseApprove, PermFiscalAuthorize, PermFinancialManage, PermItemActivate),
	"VIEWER": asSet(PermRead),
}

func asSet(perms ...string) map[string]struct{} {
	s := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		s[p] = struct{}{}
	}
	return s
}

// RoleHasPermission reports whether the role is granted the scope.
func RoleHasPermission(role, perm string) bool {
	if perms, ok := rolePermissions[role]; ok {
		_, has := perms[perm]
		return has
	}
	return false
}

// RequirePermission gates a route on a specific scope derived from the user's
// role, returning 403 when the role lacks it.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !RoleHasPermission(user.Role, perm) {
				http.Error(w, "forbidden: missing scope "+perm, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
