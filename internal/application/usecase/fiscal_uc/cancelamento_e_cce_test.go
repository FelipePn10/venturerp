package fiscal_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type authFiscalLiberado struct{ ports.AuthService }

func (authFiscalLiberado) CanCancelFiscalExit(context.Context) bool { return true }
func (authFiscalLiberado) CanCreateFiscalExit(context.Context) bool { return true }
func (authFiscalLiberado) CanGetFiscalExit(context.Context) bool    { return true }
func (authFiscalLiberado) UserID(context.Context) (uuid.UUID, error) {
	return uuid.New(), nil
}

type notaParaCancelar struct {
	fiscalrepo.FiscalRepository
	exit      *fiscalentity.FiscalExit
	cancelada bool
}

func (n *notaParaCancelar) GetExitByID(context.Context, int64) (*fiscalentity.FiscalExit, error) {
	return n.exit, nil
}
func (n *notaParaCancelar) GetFiscalConfig(context.Context) (*fiscalentity.FiscalConfig, error) {
	return &fiscalentity.FiscalConfig{}, nil
}
func (n *notaParaCancelar) CancelExitWithMotivo(_ context.Context, _ int64, _ string, _ uuid.UUID) (*fiscalentity.FiscalExit, error) {
	n.cancelada = true
	return n.exit, nil
}

const justificativaValida = "Pedido cancelado a pedido do cliente"

// A nota que foi à SEFAZ não pode ser cancelada só no sistema: sem token, ela
// continuaria valendo para a fiscalização.
func TestCancelar_NotaTransmitidaSemTokenRecusa(t *testing.T) {
	ref := "ref-1"
	repo := &notaParaCancelar{exit: &fiscalentity.FiscalExit{ID: 1, Status: fiscalentity.ExitStatusAuthorized, FocusRef: &ref, DataEmissao: time.Now()}}
	uc := &CancelFiscalExitUseCase{Repo: repo, Auth: authFiscalLiberado{}}

	_, err := uc.Execute(context.Background(), CancelFiscalExitParams{ID: 1, Motivo: justificativaValida})

	if _, ok := errorsuc.AsValidation(err); !ok || !strings.Contains(err.Error(), "VFIS0100") {
		t.Fatalf("esperava recusa apontando para os parâmetros fiscais, veio %v", err)
	}
	if repo.cancelada {
		t.Fatal("a nota foi cancelada só no sistema")
	}
}

func TestCancelar_MensagemNaoExpoeOEnum(t *testing.T) {
	repo := &notaParaCancelar{exit: &fiscalentity.FiscalExit{ID: 1, Status: fiscalentity.ExitStatusDraft}}
	uc := &CancelFiscalExitUseCase{Repo: repo, Auth: authFiscalLiberado{}}

	_, err := uc.Execute(context.Background(), CancelFiscalExitParams{ID: 1, Motivo: justificativaValida})

	if err == nil || strings.Contains(err.Error(), "DRAFT") || !strings.Contains(err.Error(), "em rascunho") {
		t.Fatalf("mensagem deveria falar a situação em português: %v", err)
	}
}

func TestPrazoDeCancelamento_ContaODiaInteiroDaEmissao(t *testing.T) {
	emissao := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	casos := []struct {
		nome    string
		agora   time.Time
		vencido bool
	}{
		{"mesmo dia", time.Date(2026, 10, 30, 23, 0, 0, 0, time.UTC), false},
		{"dia seguinte à noite, onde 24h a partir da meia-noite já recusaria", time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), false},
		{"fim do dia seguinte em Rio Branco", time.Date(2026, 11, 1, 4, 59, 0, 0, time.UTC), false},
		{"dois dias depois", time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC), true},
		{"um mês depois", time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC), true},
	}
	for _, c := range casos {
		if got := prazoDeCancelamentoVencido(emissao, c.agora); got != c.vencido {
			t.Errorf("%s: vencido=%v, esperava %v", c.nome, got, c.vencido)
		}
	}
}

func TestCCe_TextoContaCaracteresENaoBytes(t *testing.T) {
	ref := "ref-1"
	repo := &notaParaCancelar{exit: &fiscalentity.FiscalExit{ID: 1, Status: fiscalentity.ExitStatusAuthorized, FocusRef: &ref}}
	uc := &EmitirCCeUseCase{Repo: repo, Auth: authFiscalLiberado{}}

	// 14 caracteres acentuados têm mais de 15 bytes: a contagem por byte deixava passar.
	if _, err := uc.Execute(context.Background(), EmitirCCeParams{FiscalExitID: 1, TextoCorrecao: strings.Repeat("ç", 14)}); err == nil || !strings.Contains(err.Error(), "15 a 1000") {
		t.Fatalf("texto curto deveria ser recusado: %v", err)
	}
	if _, err := uc.Execute(context.Background(), EmitirCCeParams{FiscalExitID: 1, TextoCorrecao: strings.Repeat("a", 1001)}); err == nil || !strings.Contains(err.Error(), "15 a 1000") {
		t.Fatalf("texto longo deveria ser recusado: %v", err)
	}
	// Dentro do limite segue até a checagem seguinte (token ausente).
	if _, err := uc.Execute(context.Background(), EmitirCCeParams{FiscalExitID: 1, TextoCorrecao: strings.Repeat("ç", 15)}); err == nil || !strings.Contains(err.Error(), "VFIS0100") {
		t.Fatalf("texto válido deveria chegar à checagem do token: %v", err)
	}
}

func TestErroDaEmissora_NaoExpoeDetalheTecnico(t *testing.T) {
	ctx := context.Background()
	casos := []struct {
		nome      string
		err       error
		contem    string
		validacao bool
	}{
		{"recusa da SEFAZ", &focusnfe.RejeicaoError{Status: "erro_cancelamento", Motivo: "Rejeição: prazo de cancelamento superior ao previsto."}, "a SEFAZ não aceitou o cancelamento: Rejeição: prazo de cancelamento superior ao previsto", true},
		{"token recusado", &focusnfe.HTTPError{StatusCode: 401, Codigo: "nao_autorizado", Mensagem: "Token inválido"}, "VFIS0100", true},
		{"recusa da emissora", &focusnfe.HTTPError{StatusCode: 400, Codigo: "requisicao_invalida", Mensagem: "Justificativa inválida"}, "a Focus NF-e não aceitou o cancelamento: Justificativa inválida", true},
		{"emissora fora do ar", &focusnfe.HTTPError{StatusCode: 503, Corpo: "<html>Bad gateway</html>"}, "tente de novo", false},
		{"rede", fmt.Errorf("http request: %w", errors.New("dial tcp 1.2.3.4:443: i/o timeout")), "tente de novo", false},
	}
	for _, c := range casos {
		got := erroDaEmissora(ctx, "o cancelamento", 1, c.err)
		if !strings.Contains(got.Error(), c.contem) {
			t.Errorf("%s: %q não contém %q", c.nome, got, c.contem)
		}
		for _, tecnico := range []string{"HTTP", "dial tcp", "<html>", "erro_cancelamento", "nao_autorizado", "requisicao_invalida"} {
			if strings.Contains(got.Error(), tecnico) {
				t.Errorf("%s: mensagem expõe %q: %q", c.nome, tecnico, got)
			}
		}
		if _, ok := errorsuc.AsValidation(got); ok != c.validacao {
			t.Errorf("%s: validação=%v, esperava %v", c.nome, ok, c.validacao)
		}
	}
}
