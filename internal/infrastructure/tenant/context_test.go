package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestIDRequiresAuthenticatedEnterprise(t *testing.T) {
	if _, err := ID(context.Background()); !errors.Is(err, ErrMissingEnterprise) {
		t.Fatalf("expected ErrMissingEnterprise, got %v", err)
	}
}

func TestIDReturnsSelectedEnterprise(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseID: 42})
	id, err := ID(ctx)
	if err != nil || id != 42 {
		t.Fatalf("expected enterprise 42, got %d, %v", id, err)
	}
}

func TestCodeRequiresAndReturnsAuthenticatedEnterpriseCode(t *testing.T) {
	if _, err := Code(context.Background()); !errors.Is(err, ErrMissingEnterprise) {
		t.Fatalf("expected ErrMissingEnterprise, got %v", err)
	}
	ctx := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{EnterpriseCode: 77})
	code, err := Code(ctx)
	if err != nil || code != 77 {
		t.Fatalf("Code() = %d, %v", code, err)
	}
}
