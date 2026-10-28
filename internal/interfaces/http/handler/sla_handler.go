package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/sla_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// SLAHandler expõe as regras de SLA da assistência e o painel dos chamados.
type SLAHandler struct{ uc *sla_uc.UseCase }

func NewSLAHandler(uc *sla_uc.UseCase) *SLAHandler { return &SLAHandler{uc: uc} }

func (h *SLAHandler) Regras(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Regras(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *SLAHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var dto sla_uc.RegraDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarRegra(r.Context(), 0, dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *SLAHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "regra inválida")
		return
	}
	var dto sla_uc.RegraDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarRegra(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *SLAHandler) Excluir(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "regra inválida")
		return
	}
	responder(w, http.StatusOK, map[string]bool{"excluida": true}, h.uc.ExcluirRegra(r.Context(), id))
}

func (h *SLAHandler) Painel(w http.ResponseWriter, r *http.Request) {
	dias, _ := strconv.Atoi(r.URL.Query().Get("dias"))
	out, err := h.uc.Painel(r.Context(), r.URL.Query().Get("situacao"), int64DaQuery(r, "cliente"), dias)
	responder(w, http.StatusOK, out, err)
}

func (h *SLAHandler) Expediente(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Expediente(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *SLAHandler) SalvarExpediente(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		Intervalos []sla_uc.HorarioDTO `json:"intervalos"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarExpediente(r.Context(), dto.Intervalos)
	responder(w, http.StatusOK, out, err)
}
