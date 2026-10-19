package procurement_uc

import (
	"context"
	"strings"
	"testing"

	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/shopspring/decimal"
)

type saldoFixo struct {
	stockrepo.StockRepository
	q decimal.Decimal
}

func (s saldoFixo) GetBalance(context.Context, int64, string, int64) (*stockentity.StockBalance, error) {
	return &stockentity.StockBalance{Quantity: s.q}, nil
}

// A análise só é gravada se a inspeção tem o que ela vai tirar: com 4 casas na
// ordem (6,1856) e 6,185567 no saldo, liberar o exibido era recusado depois de a
// análise já estar registrada.
func TestConferirSaldoDaInspecao(t *testing.T) {
	uc := &UseCase{StockRepo: saldoFixo{q: decimal.RequireFromString("6.185567")}}
	if err := uc.conferirSaldoDaInspecao(context.Background(), 1, "", 2, 6.185567); err != nil {
		t.Fatalf("liberar o saldo inteiro deveria passar: %v", err)
	}
	err := uc.conferirSaldoDaInspecao(context.Background(), 1, "", 2, 6.1856)
	if err == nil || !strings.Contains(err.Error(), "6.185567") {
		t.Fatalf("tirar 6,1856 de 6,185567 deveria ser recusado com o saldo na mensagem: %v", err)
	}
	if err := uc.conferirSaldoDaInspecao(context.Background(), 1, "", 2, 0); err != nil {
		t.Fatalf("nada a tirar não confere saldo: %v", err)
	}
}
