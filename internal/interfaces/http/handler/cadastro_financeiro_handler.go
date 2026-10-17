package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// CadastroFinanceiroHandler: manutenção (listar com inativos, ver uso, alterar,
// excluir) do plano de contas e do centro de custo financeiros. As regras de
// quando cada coisa é permitida ficam nos casos de uso.
type CadastroFinanceiroHandler struct {
	ListPlanos   *financial_uc.ListPlanoContasCadastroUseCase
	UsoPlano     *financial_uc.UsoPlanoContasUseCase
	UpdatePlano  *financial_uc.UpdatePlanoContasUseCase
	DeletePlano  *financial_uc.DeletePlanoContasUseCase
	ListCentros  *financial_uc.ListCentrosCustoCadastroUseCase
	UsoCentro    *financial_uc.UsoCentroCustoUseCase
	UpdateCentro *financial_uc.UpdateCentroCustoUseCase
	DeleteCentro *financial_uc.DeleteCentroCustoUseCase
}

func idDoCadastro(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "id inválido")
	}
	return id, ok
}

func (h *CadastroFinanceiroHandler) ListPlanoContas(w http.ResponseWriter, r *http.Request) {
	out, err := h.ListPlanos.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CadastroFinanceiroHandler) UsoPlanoContas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoCadastro(w, r)
	if !ok {
		return
	}
	out, err := h.UsoPlano.Execute(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CadastroFinanceiroHandler) UpdatePlanoContas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoCadastro(w, r)
	if !ok {
		return
	}
	var dto request.UpdatePlanoContasDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.UpdatePlano.Execute(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CadastroFinanceiroHandler) DeletePlanoContas(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoCadastro(w, r)
	if !ok {
		return
	}
	if err := h.DeletePlano.Execute(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CadastroFinanceiroHandler) ListCentrosCusto(w http.ResponseWriter, r *http.Request) {
	out, err := h.ListCentros.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CadastroFinanceiroHandler) UsoCentroCusto(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoCadastro(w, r)
	if !ok {
		return
	}
	out, err := h.UsoCentro.Execute(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CadastroFinanceiroHandler) UpdateCentroCusto(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoCadastro(w, r)
	if !ok {
		return
	}
	var dto request.UpdateCentroCustoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.UpdateCentro.Execute(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CadastroFinanceiroHandler) DeleteCentroCusto(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoCadastro(w, r)
	if !ok {
		return
	}
	if err := h.DeleteCentro.Execute(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
