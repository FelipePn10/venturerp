package middleware

import "testing"

func TestRoleHasPermission(t *testing.T) {
	cases := []struct {
		role string
		perm string
		want bool
	}{
		{"ADMIN", PermAdmin, true},
		{"ADMIN", PermPlanningRun, true},
		{"USER", PermPlanningRun, true},
		{"USER", PermAdmin, false},
		{"VIEWER", PermRead, true},
		{"VIEWER", PermWrite, false},
		{"VIEWER", PermFiscalAuthorize, false},
		{"UNKNOWN", PermRead, false},
		{"", PermRead, false},
	}
	for _, tc := range cases {
		if got := RoleHasPermission(tc.role, tc.perm); got != tc.want {
			t.Errorf("RoleHasPermission(%q, %q) = %v, want %v", tc.role, tc.perm, got, tc.want)
		}
	}
}
