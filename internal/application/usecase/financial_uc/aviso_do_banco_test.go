package financial_uc

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

type bancoFake map[string]*ports.PagamentoConsultado

func (b bancoFake) GerarPix(context.Context, ports.CobrancaPix) (*ports.PixGerado, error) {
	return nil, nil
}
func (bancoFake) ConsultarCobranca(context.Context, string) (*ports.PixGerado, error) {
	return nil, nil
}
func (bancoFake) CancelarCobranca(context.Context, string, string) error { return nil }
func (bancoFake) RegistrarWebhook(context.Context, string) error         { return nil }
func (b bancoFake) ConsultarPagamento(_ context.Context, c string) (*ports.PagamentoConsultado, error) {
	return b[c], nil
}

type titulosFake struct{}

func (titulosFake) TituloPorSeuNumero(_ context.Context, s string) (int64, bool, decimal.Decimal, error) {
	if s == "CR9" {
		return 9, false, decimal.Zero, nil // já baixado
	}
	return 0, false, decimal.Zero, nil
}

// Aviso de cobrança não paga ou de título já baixado não chama a baixa.
func TestAvisoDoBancoSoBaixaOQueOBancoConfirma(t *testing.T) {
	uc := &AvisoDoBancoUseCase{Titulos: titulosFake{}, Cobranca: bancoFake{
		"a": {SeuNumero: "CR1", Situacao: "A_RECEBER"},
		"b": {SeuNumero: "CR9", Pago: true, Valor: decimal.NewFromInt(10), Data: time.Now()},
	}}
	r, err := uc.Processar(context.Background(), []string{"a", "b"})
	if err != nil || len(r.Baixados) != 0 || len(r.Ignorados) != 2 {
		t.Fatalf("%+v err=%v", r, err)
	}
}

// Título travado e baixado por outro aviso: conta como ignorado, não como erro.
func TestAvisoConcorrenteNaoViraErro(t *testing.T) {
	err := errorsuc.NewConflictError("o título 7 já está RECEBIDO")
	var conflito *errorsuc.ConflictError
	if !errors.As(fmt.Errorf("embrulhado: %w", err), &conflito) {
		t.Fatal("o conflito precisa ser reconhecido mesmo embrulhado")
	}
}
