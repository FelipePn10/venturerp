package delivery_promise_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/service"
)

type leitorFixo struct{ s service.SituacaoDoItem }

func (l leitorFixo) SituacaoParaPromessa(context.Context, int64) (service.SituacaoDoItem, error) {
	return l.s, nil
}
func (leitorFixo) ComponentesParaPromessa(context.Context, int64) ([]service.Componente, error) {
	return nil, nil
}

type prazoFixo int

func (p prazoFixo) PrazoDeFabricacao(context.Context, int64, decimal.Decimal) (int, error) {
	return int(p), nil
}

// Risco do V: com o prazo do MRP (roteiro do lote, dias no terceiro inclusos)
// a etapa no terceiro não é somada duas vezes.
func TestPromessaUsaOPrazoDeFabricacaoDoMRP(t *testing.T) {
	segunda := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	s := service.SituacaoDoItem{ItemCode: 1, Fabricado: true, PrazoFabricacaoDias: 5, DiasNoTerceiro: 5}
	uc := &PromessaUseCase{Repo: leitorFixo{s}, Prazos: prazoFixo(7)}
	p, err := uc.prometer(context.Background(), segunda, 1, decimal.NewFromInt(50), map[int64]decimal.Decimal{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if quer := service.DiasUteis(segunda, 7); !p.Data.Equal(quer) {
		t.Fatalf("prometeu %s, quer %s (7 dias úteis do MRP)", p.Data.Format("02/01"), quer.Format("02/01"))
	}
	// Sem o MRP: cadastro (5) + terceiro (5).
	uc.Prazos = nil
	p, _ = uc.prometer(context.Background(), segunda, 1, decimal.NewFromInt(50), map[int64]decimal.Decimal{}, 0)
	if quer := service.DiasUteis(segunda, 10); !p.Data.Equal(quer) {
		t.Fatalf("sem MRP prometeu %s, quer %s", p.Data.Format("02/01"), quer.Format("02/01"))
	}
}
