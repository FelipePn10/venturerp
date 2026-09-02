package representativevalidation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
)

type representativeStub struct {
	representative *entity.Representative
	err            error
}

func (s representativeStub) Get(context.Context, int64) (*entity.Representative, error) {
	return s.representative, s.err
}

func TestValidateRejectsBlockedAndInactiveRepresentativesInPortuguese(t *testing.T) {
	reason := "inadimplência"
	tests := []struct {
		name string
		rep  *entity.Representative
		want string
	}{
		{"bloqueado", &entity.Representative{Code: 7, IsActive: true, Blocked: true, BlockReason: &reason}, "está bloqueado: inadimplência"},
		{"inativo", &entity.Representative{Code: 7}, "está inativo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := int64(7)
			err := Validate(context.Background(), representativeStub{representative: tt.rep}, &code)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("erro=%v, esperado conter %q", err, tt.want)
			}
		})
	}
}

func TestValidateRejectsRepresentativeOutsideAuthenticatedEnterprise(t *testing.T) {
	code := int64(8)
	err := Validate(context.Background(), representativeStub{err: errors.New("not found")}, &code)
	if err == nil || !strings.Contains(err.Error(), "empresa autenticada") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidateAcceptsActiveUnblockedRepresentative(t *testing.T) {
	code := int64(9)
	if err := Validate(context.Background(), representativeStub{representative: &entity.Representative{Code: code, IsActive: true}}, &code); err != nil {
		t.Fatal(err)
	}
}
