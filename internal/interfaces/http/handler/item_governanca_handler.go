package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_governanca_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
)

// ItemGovernancaHandler expõe a governança do cadastro de item: parâmetros,
// regras de campo, painel por item, liberação das visões, status, histórico,
// uso e duplicidade. O item vem pelo código de negócio.
type ItemGovernancaHandler struct{ uc *item_governanca_uc.UseCase }

func NewItemGovernancaHandler(uc *item_governanca_uc.UseCase) *ItemGovernancaHandler {
	return &ItemGovernancaHandler{uc: uc}
}

func (h *ItemGovernancaHandler) Parametros(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Parametros(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) SalvarParametros(w http.ResponseWriter, r *http.Request) {
	var dto dom.Parametros
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarParametros(r.Context(), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Regras(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Regras(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) SalvarRegra(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		Obrigatorio *bool `json:"obrigatorio"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarRegra(r.Context(), chi.URLParam(r, "campo"), dto.Obrigatorio)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Politicas(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Politicas(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) SalvarPoliticas(w http.ResponseWriter, r *http.Request) {
	var dto []dom.RegraPolitica
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarPoliticas(r.Context(), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) PreviaFiscal(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.PreviaFiscal(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Embalagens(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Embalagens(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) SalvarEmbalagens(w http.ResponseWriter, r *http.Request) {
	var dto []dom.Embalagem
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarEmbalagens(r.Context(), itemCodePathParam(r, "item"), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) AtributosLivres(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.AtributosLivres(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) SalvarAtributosLivres(w http.ResponseWriter, r *http.Request) {
	var dto []dom.AtributoLivre
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarAtributosLivres(r.Context(), itemCodePathParam(r, "item"), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) NomesDeAtributosLivres(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.NomesDeAtributosLivres(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) ResumoDeAtributosLivres(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ResumoDeAtributosLivres(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) CuidadosDosItens(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.CuidadosDosItens(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Revisoes(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Revisoes(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) CriarRevisao(w http.ResponseWriter, r *http.Request) {
	var dto item_governanca_uc.NovaRevisaoDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.CriarRevisao(r.Context(), itemCodePathParam(r, "item"), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *ItemGovernancaHandler) Saldo(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.SaldoDuasUnidades(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Prontidao(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Prontidao(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Pendentes(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Pendentes(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Duplicados(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := h.uc.Duplicados(r.Context(), q.Get("nome"), q.Get("ncm"), q.Get("excluir"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Painel(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Painel(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Liberar(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		Visao string `json:"visao"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Liberar(r.Context(), itemCodePathParam(r, "item"), dto.Visao)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Estornar(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		Visao  string `json:"visao"`
		Motivo string `json:"motivo"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.EstornarLiberacao(r.Context(), itemCodePathParam(r, "item"), dto.Visao, dto.Motivo)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) MudarStatus(w http.ResponseWriter, r *http.Request) {
	var dto item_governanca_uc.StatusDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.MudarStatus(r.Context(), itemCodePathParam(r, "item"), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Historico(w http.ResponseWriter, r *http.Request) {
	limite, _ := strconv.Atoi(r.URL.Query().Get("limite"))
	out, err := h.uc.Historico(r.Context(), itemCodePathParam(r, "item"), limite)
	responder(w, http.StatusOK, out, err)
}

func (h *ItemGovernancaHandler) Uso(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Uso(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}
