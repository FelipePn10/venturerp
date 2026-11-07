package fiscal_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type regraDaEntradaFixa repository.RegraDaEntrada

func (r regraDaEntradaFixa) RegraDaEntrada(context.Context, int64) (repository.RegraDaEntrada, error) {
	return repository.RegraDaEntrada(r), nil
}

type rotaFixa struct {
	InspecaoDeRecebimento
	wh int64
}

func (r rotaFixa) ResolveInspectionRoute(context.Context, int64, string) (int64, bool) {
	return r.wh, r.wh > 0
}

func TestEntradaPorNotaSegueORoteiroDoCadastro(t *testing.T) {
	ctx := context.Background()
	direto := &ApproveFiscalEntryUseCase{Regras: regraDaEntradaFixa{Roteiro: "DIRETO"}, Inspecao: rotaFixa{wh: 9}}
	if _, ok, err := direto.inspecaoDoItem(ctx, 1); ok || err != nil {
		t.Fatalf("DIRETO não passa pela inspeção: %v %v", ok, err)
	}
	semRota := &ApproveFiscalEntryUseCase{Regras: regraDaEntradaFixa{Roteiro: "INSPECAO"}, Inspecao: rotaFixa{}}
	if _, _, err := semRota.inspecaoDoItem(ctx, 1); err == nil {
		t.Fatal("INSPECAO sem rota não entra no estoque disponível")
	}
	padrao := &ApproveFiscalEntryUseCase{Regras: regraDaEntradaFixa{Roteiro: "PADRAO"}, Inspecao: rotaFixa{wh: 9}}
	if wh, ok, err := padrao.inspecaoDoItem(ctx, 1); !ok || wh != 9 || err != nil {
		t.Fatalf("PADRAO segue a rota ativa: %d %v %v", wh, ok, err)
	}
	semCadastro := &ApproveFiscalEntryUseCase{Inspecao: rotaFixa{wh: 9}}
	if wh, ok, err := semCadastro.inspecaoDoItem(ctx, 1); !ok || wh != 9 || err != nil {
		t.Fatalf("sem regras, o comportamento anterior: %d %v %v", wh, ok, err)
	}
}
