package fiscal_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

type retornoFalso struct{ recebe decimal.Decimal }

func (r retornoFalso) ReceberRetornoDaNota(_ context.Context, _, _ int64, qtd decimal.Decimal, _ int64, _ int, _ uuid.UUID, _ string) (decimal.Decimal, []int64, error) {
	return decimal.Min(qtd, r.recebe), []int64{55}, nil
}

// Só a linha de retorno (5902) vai para a OS; sem OS ou com sobra, vira aviso.
func TestRetornoDeIndustrializacaoNaAprovacao(t *testing.T) {
	forn, item := int64(9), int64(601)
	doc := &entity.FiscalEntry{ID: 1, SupplierCode: &forn, Itens: []*entity.FiscalEntryItem{
		{Sequence: 1, ItemCode: &item, Cfop: "5902", Quantity: decimal.NewFromInt(4)},
		{Sequence: 2, ItemCode: &item, Cfop: "5124", Quantity: decimal.NewFromInt(4)},
	}}
	uc := &ApproveFiscalEntryUseCase{Terceiros: retornoFalso{recebe: decimal.NewFromInt(4)}}
	av := uc.retornarIndustrializacao(context.Background(), doc, uuid.New())
	if len(av) != 1 || !strings.Contains(av[0], "linha 1: retorno recebido") {
		t.Fatalf("avisos: %v", av)
	}
	uc.Terceiros = retornoFalso{recebe: decimal.NewFromInt(1)}
	if av := uc.retornarIndustrializacao(context.Background(), doc, uuid.New()); len(av) != 1 || !strings.Contains(av[0], "ficou sem OS") {
		t.Fatalf("sobra: %v", av)
	}
	uc.Terceiros = retornoFalso{}
	if av := uc.retornarIndustrializacao(context.Background(), doc, uuid.New()); len(av) != 1 || !strings.Contains(av[0], "registre o retorno à mão") {
		t.Fatalf("sem OS: %v", av)
	}
	uc.Terceiros = nil
	if av := uc.retornarIndustrializacao(context.Background(), doc, uuid.New()); av != nil {
		t.Fatalf("sem o repositório não faz nada: %v", av)
	}
}
