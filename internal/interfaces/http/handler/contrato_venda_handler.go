package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/contrato_venda_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/contratovenda"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// ContratoVendaHandler expõe o contrato de venda (comercial, Fase 5.2).
type ContratoVendaHandler struct{ uc *contrato_venda_uc.UseCase }

func NewContratoVendaHandler(uc *contrato_venda_uc.UseCase) *ContratoVendaHandler {
	return &ContratoVendaHandler{uc: uc}
}

func (h *ContratoVendaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Listar(r.Context(), dom.Filtro{Cliente: int64DaQuery(r, "cliente"), Situacao: r.URL.Query().Get("situacao")})
	responder(w, http.StatusOK, out, err)
}

func (h *ContratoVendaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var dto contrato_venda_uc.CriarDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Criar(r.Context(), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *ContratoVendaHandler) id(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "contrato inválido")
	}
	return id, ok
}

func (h *ContratoVendaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.id(w, r); ok {
		out, err := h.uc.Obter(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *ContratoVendaHandler) Situacao(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var dto struct {
		Situacao string `json:"situacao"`
		Motivo   string `json:"motivo"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Mudar(r.Context(), id, dto.Situacao, dto.Motivo)
	responder(w, http.StatusOK, out, err)
}

func (h *ContratoVendaHandler) Programar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var dto contrato_venda_uc.ParcelaDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Programar(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *ContratoVendaHandler) RemoverParcela(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	parcela, ok := idDaRota(r, "parcela")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "parcela inválida")
		return
	}
	out, err := h.uc.RemoverParcela(r.Context(), id, parcela)
	responder(w, http.StatusOK, out, err)
}

func (h *ContratoVendaHandler) Chamar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var dto contrato_venda_uc.ChamadaDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Chamar(r.Context(), id, dto)
	responder(w, http.StatusCreated, out, err)
}
