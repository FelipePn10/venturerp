package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/devolucao_venda_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// DevolucaoVendaHandler expõe a devolução de venda (comercial, Fase 5.1).
type DevolucaoVendaHandler struct{ uc *devolucao_venda_uc.UseCase }

func NewDevolucaoVendaHandler(uc *devolucao_venda_uc.UseCase) *DevolucaoVendaHandler {
	return &DevolucaoVendaHandler{uc: uc}
}

func (h *DevolucaoVendaHandler) Previa(w http.ResponseWriter, r *http.Request) {
	nota := int64DaQuery(r, "nota")
	if nota == nil {
		security.RespondError(w, http.StatusBadRequest, "informe a NF de saída (nota)")
		return
	}
	out, err := h.uc.Previa(r.Context(), *nota)
	responder(w, http.StatusOK, out, err)
}

func (h *DevolucaoVendaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var dto devolucao_venda_uc.CriarDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Criar(r.Context(), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *DevolucaoVendaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Listar(r.Context(), int64DaQuery(r, "nota"), r.URL.Query().Get("situacao"))
	responder(w, http.StatusOK, out, err)
}

func (h *DevolucaoVendaHandler) id(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "devolução inválida")
	}
	return id, ok
}

func (h *DevolucaoVendaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.id(w, r); ok {
		out, err := h.uc.Obter(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *DevolucaoVendaHandler) GerarNF(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.id(w, r); ok {
		out, err := h.uc.GerarNF(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *DevolucaoVendaHandler) Efetivar(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.id(w, r); ok {
		d, avisos, err := h.uc.Efetivar(r.Context(), id)
		responder(w, http.StatusOK, map[string]any{"devolucao": d, "avisos": avisos}, err)
	}
}

func (h *DevolucaoVendaHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var dto struct {
		Motivo string `json:"motivo"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Cancelar(r.Context(), id, dto.Motivo)
	responder(w, http.StatusOK, out, err)
}
