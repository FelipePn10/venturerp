package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// InterWebhookHandler recebe o aviso de pagamento do Banco Inter. Rota pública:
// o segredo vai no caminho (INTER_WEBHOOK_TOKEN) e o aviso só dispara a
// CONSULTA no Inter — quem baixa é a resposta do banco, não o corpo recebido.
type InterWebhookHandler struct {
	Token     string
	EmpresaID int64
	Sessao    func(ctx context.Context, empresa int64) (int64, uuid.UUID, error)
	UC        *financial_uc.AvisoDoBancoUseCase
}

type avisoInter struct {
	CodigoSolicitacao string `json:"codigoSolicitacao"`
}

// codigosDoAviso aceita a lista de cobranças ou o envelope {"payload": [...]}.
func codigosDoAviso(corpo []byte) []string {
	var lista []avisoInter
	if err := json.Unmarshal(corpo, &lista); err != nil {
		var env struct {
			Payload []avisoInter `json:"payload"`
			avisoInter
		}
		if json.Unmarshal(corpo, &env) != nil {
			return nil
		}
		lista = env.Payload
		if len(lista) == 0 && env.CodigoSolicitacao != "" {
			lista = []avisoInter{env.avisoInter}
		}
	}
	var out []string
	vistos := map[string]bool{}
	for _, a := range lista {
		if a.CodigoSolicitacao != "" && !vistos[a.CodigoSolicitacao] {
			vistos[a.CodigoSolicitacao] = true
			out = append(out, a.CodigoSolicitacao)
		}
	}
	return out
}

// Receber: POST /api/integracoes/inter/webhook/{token}.
func (h *InterWebhookHandler) Receber(w http.ResponseWriter, r *http.Request) {
	if h.Token == "" || subtle.ConstantTimeCompare([]byte(chi.URLParam(r, "token")), []byte(h.Token)) != 1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	corpo, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	codigos := codigosDoAviso(corpo)
	if len(codigos) == 0 {
		w.WriteHeader(http.StatusOK) // nada a fazer (teste de cadastro do webhook)
		return
	}
	code, ator, err := h.Sessao(r.Context(), h.EmpresaID)
	if err != nil {
		slog.Error("aviso do Inter sem sessão do sistema", "erro", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	ctx := context.WithValue(r.Context(), contextkey.UserKey, &security.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: h.EmpresaID, EnterpriseCode: code})
	res, err := h.UC.Processar(ctx, codigos)
	if err != nil {
		// 503: o Inter reenvia o aviso.
		slog.Error("aviso do Inter não processado", "cobrancas", codigos, "erro", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	slog.Info("aviso do Inter processado", "baixados", res.Baixados, "ignorados", res.Ignorados)
	w.WriteHeader(http.StatusOK)
}
