package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// ConciliacaoHandler expõe o extrato importado e a conciliação manual (VFIN0620).
type ConciliacaoHandler struct {
	uc *financial_uc.ConciliacaoUseCase
}

func NewConciliacaoHandler(uc *financial_uc.ConciliacaoUseCase) *ConciliacaoHandler {
	return &ConciliacaoHandler{uc: uc}
}

func idDaRotaConciliacao(w http.ResponseWriter, r *http.Request, param, nome string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código "+nome+" inválido")
		return 0, false
	}
	return id, true
}

// Extrato: GET /conciliacao/{conta_id}/extrato?situacao=&data_inicio=&data_fim=
func (h *ConciliacaoHandler) Extrato(w http.ResponseWriter, r *http.Request) {
	contaID, ok := idDaRotaConciliacao(w, r, "conta_id", "da conta")
	if !ok {
		return
	}
	q := r.URL.Query()
	out, err := h.uc.ListarExtrato(r.Context(), contaID, q.Get("situacao"), q.Get("data_inicio"), q.Get("data_fim"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// AutoConciliar: POST /conciliacao/{conta_id}/auto
func (h *ConciliacaoHandler) AutoConciliar(w http.ResponseWriter, r *http.Request) {
	contaID, ok := idDaRotaConciliacao(w, r, "conta_id", "da conta")
	if !ok {
		return
	}
	out, err := h.uc.AutoConciliar(r.Context(), contaID)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Candidatos: GET /conciliacao/extrato/{id}/candidatos
func (h *ConciliacaoHandler) Candidatos(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaConciliacao(w, r, "id", "do lançamento")
	if !ok {
		return
	}
	out, err := h.uc.Candidatos(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// Conciliar: POST /conciliacao/extrato/{id}/conciliar {"fluxo_caixa_id": N}
func (h *ConciliacaoHandler) Conciliar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaConciliacao(w, r, "id", "do lançamento")
	if !ok {
		return
	}
	var body struct {
		FluxoCaixaID int64 `json:"fluxo_caixa_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.uc.Conciliar(r.Context(), id, body.FluxoCaixaID); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]bool{"conciliado": true})
}

// Lancar: POST /conciliacao/extrato/{id}/lancar {"descricao": "..."} (corpo opcional)
func (h *ConciliacaoHandler) Lancar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaConciliacao(w, r, "id", "do lançamento")
	if !ok {
		return
	}
	var body struct {
		Descricao string `json:"descricao"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Lancar(r.Context(), id, body.Descricao)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

// Desfazer: POST /conciliacao/extrato/{id}/desfazer
func (h *ConciliacaoHandler) Desfazer(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRotaConciliacao(w, r, "id", "do lançamento")
	if !ok {
		return
	}
	if err := h.uc.Desfazer(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]bool{"conciliado": false})
}
