package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// IntegracoesHandler: o que está ligado (as telas escondem o que não está) e o
// cadastro do webhook no banco.
type IntegracoesHandler struct {
	Banco, WhatsApp, Frete, Portal bool
	Treino                         bool
	WebhookToken                   string
	Cobranca                       ports.CobrancaBancaria
}

// Status: GET /api/integracoes/status.
func (h *IntegracoesHandler) Status(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{
		"bank": h.Banco, "whatsapp": h.WhatsApp, "freight": h.Frete, "portal": h.Portal,
		"bank_webhook_ready": h.Banco && h.WebhookToken != "", "training": h.Treino,
	})
}

// RegistrarWebhookInter: POST /api/integracoes/inter/webhook-registrar {base_url}
// — base_url é o endereço público da API (https://api.empresa…).
func (h *IntegracoesHandler) RegistrarWebhookInter(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		BaseURL string `json:"base_url"`
	}
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	if h.WebhookToken == "" {
		jsonError(w, http.StatusUnprocessableEntity, "defina INTER_WEBHOOK_TOKEN no ambiente da API antes de cadastrar o webhook")
		return
	}
	base := strings.TrimRight(strings.TrimSpace(dto.BaseURL), "/")
	if !strings.HasPrefix(base, "https://") {
		jsonError(w, http.StatusUnprocessableEntity, "informe o endereço público da API em HTTPS (ex.: https://api.suaempresa.com.br)")
		return
	}
	url := base + "/api/integracoes/inter/webhook/" + h.WebhookToken
	if err := h.Cobranca.RegistrarWebhook(r.Context(), url); err != nil {
		if errors.Is(err, ports.ErrIntegracaoDesligada) {
			jsonError(w, http.StatusUnprocessableEntity, "a integração com o Banco Inter não está configurada")
			return
		}
		respondErr(w, err)
		return
	}
	// O segredo não volta na resposta.
	jsonResponse(w, http.StatusOK, map[string]string{"registered": base + "/api/integracoes/inter/webhook/…"})
}
