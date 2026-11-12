package financial_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
)

func pagar(repo *fakeFinRepo, conta int64) error {
	uc := BaixarContaPagarUseCase{Repo: repo, FiscalRepo: fakeFiscalRepo{}, Auth: fakeFinAuth{canPagar: true}}
	return uc.Execute(context.Background(), 1, request.BaixarContaPagarDTO{
		ContaBancariaID: conta, DataPagamento: "2026-01-10", ValorPago: decimal.NewFromInt(100)})
}

// Título vencido é dívida em aberto: tem de aceitar pagamento.
func TestBaixarContaPagar_AceitaTituloVencido(t *testing.T) {
	repo := &fakeFinRepo{cp: contaPagar(entity.ContaPagarStatusVencido, 100, time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))}
	if err := pagar(repo, 2); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.baixaParams == nil {
		t.Fatal("a baixa do título vencido não foi gravada")
	}
}

// A conta da baixa tem de existir na empresa e estar ativa.
func TestBaixarContaPagar_RecusaContaBancariaInvalida(t *testing.T) {
	casos := map[string]func(*fakeFinRepo) int64{
		"não informada": func(*fakeFinRepo) int64 { return 0 },
		"inexistente": func(r *fakeFinRepo) int64 {
			r.contaErr = errorsuc.NewNotFoundError("conta bancária 99 não encontrada")
			return 99
		},
		"inativa": func(r *fakeFinRepo) int64 { r.contaInativa = true; return 2 },
	}
	for nome, preparar := range casos {
		t.Run(nome, func(t *testing.T) {
			repo := &fakeFinRepo{cp: contaPagar(entity.ContaPagarStatusPendente, 100, time.Now())}
			err := pagar(repo, preparar(repo))
			if _, ok := errorsuc.AsValidation(err); !ok {
				t.Fatalf("erro = %v, esperado validação", err)
			}
			if repo.baixaParams != nil {
				t.Fatal("a baixa foi gravada em conta inválida")
			}
		})
	}
}

// Aprovar o pago ou o cancelado o devolvia para "aprovado": a dívida quitada
// voltava a constar em aberto.
func TestAprovarContaPagar_SoOTituloQueAguardaAprovacao(t *testing.T) {
	for status, pode := range map[entity.ContaPagarStatus]bool{
		entity.ContaPagarStatusPendente:  true,
		entity.ContaPagarStatusVencido:   true,
		entity.ContaPagarStatusAprovado:  false,
		entity.ContaPagarStatusPago:      false,
		entity.ContaPagarStatusCancelado: false,
	} {
		t.Run(string(status), func(t *testing.T) {
			repo := &fakeFinRepo{cp: contaPagar(status, 100, time.Now())}
			uc := ApproveContaPagarUseCase{Repo: repo, Auth: fakeFinAuth{canPagar: true}}
			err := uc.Execute(context.Background(), 1)
			if pode && (err != nil || !repo.aprovou) {
				t.Fatalf("deveria aprovar: err=%v aprovou=%v", err, repo.aprovou)
			}
			if !pode && (err == nil || repo.aprovou) {
				t.Fatalf("não deveria aprovar título %s", status)
			}
		})
	}
}

// Cancelar só o que está em aberto e sem baixa: o título quitado cancelado
// deixava o pagamento no caixa sem dívida que o explicasse.
func TestImpedeCancelarTitulo(t *testing.T) {
	zero, parte := decimal.Zero, decimal.NewFromInt(50)
	for _, s := range []string{"PENDENTE", "APROVADO", "VENCIDO"} {
		if msg := impedeCancelarTitulo(s, zero, "pago", "pagamento"); msg != "" {
			t.Fatalf("título %s sem baixa deveria poder ser cancelado: %s", s, msg)
		}
		if msg := impedeCancelarTitulo(s, parte, "pago", "pagamento"); msg == "" {
			t.Fatalf("título %s com baixa não pode ser cancelado", s)
		}
	}
	for _, s := range []string{"PAGO", "RECEBIDO", "CANCELADO", "BAIXADO_PERDA", "RENEGOCIADO"} {
		if msg := impedeCancelarTitulo(s, zero, "recebido", "recebimento"); msg == "" {
			t.Fatalf("título %s não pode ser cancelado", s)
		}
	}
}

func TestCancelarContaPagar_RecusaTituloPago(t *testing.T) {
	pago := contaPagar(entity.ContaPagarStatusPago, 100, time.Now())
	pago.ValorPago = decimal.NewFromInt(100)
	repo := &crudRepoPago{cp: pago}
	err := (&CancelContaPagarUseCase{Repo: repo, Auth: crudAuth{ok: true}}).Execute(context.Background(), 1)
	if _, ok := errorsuc.AsValidation(err); !ok || repo.cancelou {
		t.Fatalf("erro = %v, cancelou = %v", err, repo.cancelou)
	}
}

type crudRepoPago struct {
	crudRepo
	cp       *entity.ContaPagar
	cancelou bool
}

func (r *crudRepoPago) GetContaPagar(context.Context, int64) (*entity.ContaPagar, error) {
	return r.cp, nil
}
func (r *crudRepoPago) CancelContaPagar(context.Context, int64) error { r.cancelou = true; return nil }
