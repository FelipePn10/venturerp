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
	// PermProductionReport é o apontamento de chão de fábrica: iniciar, pausar,
	// concluir etapa, apontar produção e consumo, devolver sucata e ler código de
	// barras. Era gate de PLANEJAMENTO (`CanCreatePlannedOrder`): o operador do
	// posto precisava da permissão de criar ordem planejada — e ganhava junto
	// tudo o que ela abre. São decisões diferentes, com gente diferente.
	PermProductionReport = "production:report" // apontar produção no chão de fábrica
	PermAdmin            = "admin"             // administrative configuration
)

// rolePermissions maps a JWT role to the scopes it is granted. ADMIN gets
// everything; USER gets all operational scopes but not admin; VIEWER is
// read-only. Unknown roles get nothing.
// OPERATOR existe para o posto de trabalho: enxerga o que precisa e aponta o
// que fez, sem poder criar ordem, aprovar compra ou mexer em cadastro. Os
// perfis que já operavam continuam idênticos — ADMIN e USER recebem o escopo
// novo junto com todos os que já tinham.
var rolePermissions = map[string]map[string]struct{}{
	"ADMIN":    asSet(PermRead, PermWrite, PermPlanningRun, PermPurchaseApprove, PermFiscalAuthorize, PermFinancialManage, PermItemActivate, PermProductionReport, PermAdmin),
	"USER":     asSet(PermRead, PermWrite, PermPlanningRun, PermPurchaseApprove, PermFiscalAuthorize, PermFinancialManage, PermItemActivate, PermProductionReport),
	"OPERATOR": asSet(PermRead, PermProductionReport),
	"VIEWER":   asSet(PermRead),
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
				respondJSONErro(w, http.StatusForbidden, "ACESSO_NEGADO",
					"seu perfil não tem a permissão necessária para esta operação ("+perm+")")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
