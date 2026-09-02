package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// StructureConfiguratorHandler atende o botão "Configurador" da Estrutura de
// Produto (VENT0210). O configurador não tem tela própria: estes dois endpoints
// carregam o painel e aplicam a configuração escolhida.
type StructureConfiguratorHandler struct {
	uc *structure_uc.StructureConfiguratorUseCase
}

func NewStructureConfiguratorHandler(uc *structure_uc.StructureConfiguratorUseCase) *StructureConfiguratorHandler {
	return &StructureConfiguratorHandler{uc: uc}
}

// Panel devolve perguntas, respostas possíveis, configurações já geradas e as
// fórmulas de quantidade que a configuração alimenta.
func (h *StructureConfiguratorHandler) Panel(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(chi.URLParam(r, "itemCode"))
	if code == "" {
		jsonError(w, http.StatusBadRequest, "informe o código do item")
		return
	}
	result, err := h.uc.Panel(r.Context(), request.TextCode(code))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// Apply valida as respostas contra as restrições, gera a máscara e devolve a
// estrutura resolvida para a configuração.
func (h *StructureConfiguratorHandler) Apply(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(chi.URLParam(r, "itemCode"))
	if code == "" {
		jsonError(w, http.StatusBadRequest, "informe o código do item")
		return
	}
	var dto request.ApplyStructureConfigurationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.Apply(r.Context(), request.TextCode(code), dto)
	if err != nil {
		// Restrição violada volta com a lista para a tela destacar as perguntas.
		var violation *structure_uc.RestrictionViolationError
		if errors.As(err, &violation) {
			jsonResponse(w, http.StatusUnprocessableEntity, map[string]any{
				"error":      violation.Error(),
				"code":       "RESTRICAO_DE_CONFIGURACAO",
				"violations": violation.Violations,
			})
			return
		}
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
