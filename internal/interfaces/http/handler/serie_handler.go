package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/serie_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/serie"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// SerieHandler expõe o perfil de pós-venda do item, as séries (base
// instalada) e as séries da linha da NF de saída.
type SerieHandler struct{ uc *serie_uc.UseCase }

func NewSerieHandler(uc *serie_uc.UseCase) *SerieHandler { return &SerieHandler{uc: uc} }

func int64DaQuery(r *http.Request, nome string) *int64 {
	v, err := strconv.ParseInt(r.URL.Query().Get(nome), 10, 64)
	if err != nil || v <= 0 {
		return nil
	}
	return &v
}

func (h *SerieHandler) Perfis(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Perfis(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) Perfil(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Perfil(r.Context(), chi.URLParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) SalvarPerfil(w http.ResponseWriter, r *http.Request) {
	var dto serie_uc.PerfilDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarPerfil(r.Context(), chi.URLParam(r, "item"), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) Listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limite, _ := strconv.Atoi(q.Get("limite"))
	out, err := h.uc.Listar(r.Context(), q.Get("item"), dom.Filtro{CustomerCode: int64DaQuery(r, "cliente"),
		Situacao: q.Get("situacao"), Busca: q.Get("busca"), Limite: limite})
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) Cadastrar(w http.ResponseWriter, r *http.Request) {
	var dto serie_uc.CadastroDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Cadastrar(r.Context(), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *SerieHandler) serie(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "série inválida")
	}
	return id, ok
}

func (h *SerieHandler) Obter(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.serie(w, r); ok {
		out, err := h.uc.Obter(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *SerieHandler) Instalar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.serie(w, r)
	var dto serie_uc.InstalacaoDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Instalar(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) Transferir(w http.ResponseWriter, r *http.Request) {
	id, ok := h.serie(w, r)
	var dto serie_uc.TransferenciaDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Transferir(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) Baixar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.serie(w, r)
	var dto struct {
		Motivo string `json:"motivo"`
	}
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Baixar(r.Context(), id, dto.Motivo)
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) LinhasDaNota(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "nota inválida")
		return
	}
	out, err := h.uc.LinhasDaNota(r.Context(), id)
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) DefinirSeriesDaLinha(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	item, okItem := idDaRota(r, "itemId")
	if !ok || !okItem {
		security.RespondError(w, http.StatusBadRequest, "nota ou linha inválida")
		return
	}
	var dto serie_uc.SeriesDaLinhaDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.DefinirSeriesDaLinha(r.Context(), id, item, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) ConsultarGarantia(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dto := serie_uc.ConsultaGarantiaDTO{Item: q.Get("item"), Serie: q.Get("serie"), Cliente: int64DaQuery(r, "cliente"), NumeroNF: int64DaQuery(r, "nf")}
	if d := q.Get("data_compra"); d != "" {
		dto.DataCompra = &d
	}
	out, err := h.uc.ConsultarGarantia(r.Context(), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *SerieHandler) EstenderGarantia(w http.ResponseWriter, r *http.Request) {
	id, ok := h.serie(w, r)
	var dto serie_uc.GarantiaEstendidaDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.EstenderGarantia(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}
