package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// Recusa do provedor fiscal precisa chegar ao usuário COM O MOTIVO. Embrulhada
// em fmt.Errorf ela virava 500 "erro interno do servidor": quem tentava faturar
// não tinha como saber que o problema era o CNPJ do emitente, e a nota ficava
// parada sem explicação.
func TestRecusaDeServicoExternoRespondeMotivo(t *testing.T) {
	w := httptest.NewRecorder()
	erro := fmt.Errorf("autorizar nota: %w",
		errorsuc.NewExternalServiceError("Focus NF-e", "HTTP 403 (permissao_negada): CNPJ do emitente não autorizado."))

	RespondUseCaseError(w, erro)

	if w.Code != 502 {
		t.Fatalf("status = %d, esperado 502", w.Code)
	}
	var corpo struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatal(err)
	}
	if corpo.Code != "SERVICO_EXTERNO_RECUSOU" {
		t.Fatalf("code = %s", corpo.Code)
	}
	for _, trecho := range []string{"Focus NF-e", "CNPJ do emitente não autorizado"} {
		if !contains(corpo.Error, trecho) {
			t.Fatalf("a mensagem perdeu %q: %s", trecho, corpo.Error)
		}
	}
}

// O tipo é reconhecido mesmo embrulhado — é assim que o caso de uso devolve.
func TestErroDeServicoExternoSobreviveAoEmbrulho(t *testing.T) {
	base := errorsuc.NewExternalServiceError("Focus CT-e", "certificado vencido")
	if _, ok := errorsuc.AsExternalService(fmt.Errorf("camada 1: %w", fmt.Errorf("camada 2: %w", base))); !ok {
		t.Fatal("AsExternalService não atravessou dois embrulhos")
	}
	var alvo *errorsuc.ExternalServiceError
	if !errors.As(base, &alvo) || alvo.Servico != "Focus CT-e" {
		t.Fatal("o serviço não foi preservado")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
