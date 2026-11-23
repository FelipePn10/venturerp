package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// CobrancaHandler: retorno bancário (baixa automática) e PIX do título.
type CobrancaHandler struct {
	Retorno *financial_uc.RetornoBancarioUseCase
	Pix     *financial_uc.CobrancaPixUseCase
	Fluxo   *financial_uc.FluxoComPedidosUseCase
	DRE     *financial_uc.DREGerencialUseCase
	Aviso   *financial_uc.AvisoDeCobrancaUseCase
}

// EnviarAviso: POST /api/financial/contas-receber/{id}/aviso {channel, to}.
func (h *CobrancaHandler) EnviarAviso(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "título inválido")
		return
	}
	var dto financial_uc.AvisoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.Aviso.Enviar(r.Context(), id, dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// DREGerencial: GET /api/financial/dre-gerencial?de=&ate=&plano_id=&prefixos=3,4
func (h *CobrancaHandler) DREGerencial(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hoje := time.Now()
	de, ate := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, hoje.Location()), hoje
	for _, p := range []struct {
		nome string
		dst  *time.Time
	}{{"de", &de}, {"ate", &ate}} {
		if v := q.Get(p.nome); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				jsonError(w, http.StatusBadRequest, "data inválida em "+p.nome+" (use AAAA-MM-DD)")
				return
			}
			*p.dst = t
		}
	}
	var plano *int64
	if v := q.Get("plano_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "plano inválido")
			return
		}
		plano = &n
	}
	var prefixos []string
	for _, p := range strings.Split(q.Get("prefixos"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			prefixos = append(prefixos, p)
		}
	}
	out, err := h.DRE.Gerar(r.Context(), plano, de, ate, prefixos)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// FluxoComPedidos: GET /api/financial/fluxo-projetado-pedidos?dias=90 — títulos e
// pedidos de compra/venda em aberto, por semana.
func (h *CobrancaHandler) FluxoComPedidos(w http.ResponseWriter, r *http.Request) {
	dias := 90
	if v := r.URL.Query().Get("dias"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "dias inválido")
			return
		}
		dias = n
	}
	out, err := h.Fluxo.Projetar(r.Context(), dias)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// ProcessarRetorno: POST /api/financial/retorno-bancario {conta_bancaria_id, content}.
func (h *CobrancaHandler) ProcessarRetorno(w http.ResponseWriter, r *http.Request) {
	var dto financial_uc.RetornoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.Retorno.Processar(r.Context(), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// GerarPix: POST /api/financial/contas-receber/{id}/pix.
func (h *CobrancaHandler) GerarPix(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "título inválido")
		return
	}
	out, err := h.Pix.Gerar(r.Context(), id)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, out)
}
