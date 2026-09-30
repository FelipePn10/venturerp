package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *StandardCostHandler) UpsertWorkCenterCost(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertWorkCenterCostDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	dto.UpdatedBy = actingUser(r).String()
	result, err := h.uc.UpsertWorkCenterCost(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) ListWorkCenterCosts(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListWorkCenterCosts(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) ListWorkCenters(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(queryDefault(r, "limit", "100"))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, "limit inválido")
		return
	}
	offset, err := strconv.Atoi(queryDefault(r, "offset", "0"))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, "offset inválido")
		return
	}
	result, err := h.uc.ListWorkCenters(r.Context(), r.URL.Query().Get("search"), limit, offset)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func queryDefault(r *http.Request, name, fallback string) string {
	if value := r.URL.Query().Get(name); value != "" {
		return value
	}
	return fallback
}

func (h *StandardCostHandler) UpsertItemPurchaseCost(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertItemPurchaseCostDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	result, err := h.uc.UpsertItemPurchaseCost(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) GetItemPurchaseCost(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	result, err := h.uc.GetItemPurchaseCost(r.Context(), itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) RollUp(w http.ResponseWriter, r *http.Request) {
	var dto request.CostRollupDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	// Quem apurou vem do token, nunca do corpo: com `calculated_by` vindo do
	// cliente, qualquer um podia atribuir a apuração de custo a outro usuário —
	// e, omitindo o campo, a tela levava "invalid calculated_by UUID".
	if id, ok := actor(r); ok {
		dto.CalculatedBy = id.String()
	}
	result, err := h.uc.RollUp(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *StandardCostHandler) GetStandardCost(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	mask := r.URL.Query().Get("mask")
	result, err := h.uc.GetStandardCost(r.Context(), itemCode, mask)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── esquema de rateio de indiretos e histórico (migração 000373) ─────────────

func (h *StandardCostHandler) ListarRegrasDeRateio(w http.ResponseWriter, r *http.Request) {
	regras, err := h.uc.ListarRegrasDeRateio(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, regras)
}

func (h *StandardCostHandler) CriarRegraDeRateio(w http.ResponseWriter, r *http.Request) {
	dto, ok := lerRegraDeRateio(w, r)
	if !ok {
		return
	}
	criada, err := h.uc.CriarRegraDeRateio(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, criada)
}

func (h *StandardCostHandler) AtualizarRegraDeRateio(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código da regra inválido")
		return
	}
	dto, ok := lerRegraDeRateio(w, r)
	if !ok {
		return
	}
	atualizada, err := h.uc.AtualizarRegraDeRateio(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, atualizada)
}

func (h *StandardCostHandler) DesativarRegraDeRateio(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código da regra inválido")
		return
	}
	if err := h.uc.DesativarRegraDeRateio(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "regra de rateio desativada"})
}

// HistoricoDeCusto devolve as apurações passadas do item, para comparar o custo
// componente a componente ao longo do tempo.
func (h *StandardCostHandler) HistoricoDeCusto(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	limite := 0
	if bruto := r.URL.Query().Get("limit"); bruto != "" {
		n, err := strconv.Atoi(bruto)
		if err != nil || n <= 0 {
			security.RespondError(w, http.StatusUnprocessableEntity, "limit inválido")
			return
		}
		limite = n
	}
	linhas, err := h.uc.HistoricoDeCusto(r.Context(), itemCode, r.URL.Query().Get("mask"), limite)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, linhas)
}

// lerRegraDeRateio decodifica o corpo e carimba o autor a partir do token. O autor
// nunca vem do corpo: quem cadastra a taxa que entra no custo de todo produto não
// pode ser escolhido pelo cliente.
func lerRegraDeRateio(w http.ResponseWriter, r *http.Request) (request.CostOverheadRuleDTO, bool) {
	var dto request.CostOverheadRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return dto, false
	}
	id, ok := actor(r)
	if !ok {
		security.RespondError(w, http.StatusUnauthorized, "não foi possível identificar o usuário da sessão")
		return dto, false
	}
	dto.CreatedBy = id.String()
	return dto, true
}
