package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/capa_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/capa"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// CapaHandler expõe CAPA/8D e a lista de não conformidades com lote e CAPA.
type CapaHandler struct{ uc *capa_uc.UseCase }

func NewCapaHandler(uc *capa_uc.UseCase) *CapaHandler { return &CapaHandler{uc: uc} }

func decodificar[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return false
	}
	return true
}

func responder(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, status, out)
}

func (h *CapaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := h.uc.Listar(r.Context(), q.Get("status"), q.Get("minhas") == "true", q.Get("atrasadas") == "true")
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) Abrir(w http.ResponseWriter, r *http.Request) {
	var dto capa_uc.AberturaDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Abrir(r.Context(), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *CapaHandler) capa(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "CAPA inválida")
	}
	return id, ok
}

func (h *CapaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.capa(w, r); ok {
		out, err := h.uc.Obter(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *CapaHandler) Analisar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.capa(w, r)
	var dto capa_uc.AnaliseDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Analisar(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) Avancar(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.capa(w, r); ok {
		out, err := h.uc.Avancar(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *CapaHandler) Encerrar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.capa(w, r)
	var dto capa_uc.EncerramentoDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Encerrar(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.capa(w, r)
	var dto struct {
		Motivo string `json:"motivo"`
	}
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Cancelar(r.Context(), id, dto.Motivo)
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) AdicionarAcao(w http.ResponseWriter, r *http.Request) {
	id, ok := h.capa(w, r)
	var dto capa_uc.AcaoDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.AdicionarAcao(r.Context(), id, dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *CapaHandler) AtualizarAcao(w http.ResponseWriter, r *http.Request) {
	id, ok := h.capa(w, r)
	acao, ok2 := idDaRota(r, "acao")
	if ok && !ok2 {
		security.RespondError(w, http.StatusBadRequest, "ação inválida")
		return
	}
	var dto capa_uc.AndamentoDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.AtualizarAcao(r.Context(), id, acao, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) VincularNC(w http.ResponseWriter, r *http.Request) {
	id, ok := h.capa(w, r)
	var dto struct {
		NCID int64 `json:"nc_id"`
	}
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.VincularNC(r.Context(), id, dto.NCID)
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) AcoesPendentes(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.AcoesPendentes(r.Context(), r.URL.Query().Get("todas") != "true")
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) Usuarios(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Usuarios(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *CapaHandler) NCs(w http.ResponseWriter, r *http.Request) {
	item, ok1 := inteiroDaConsulta(r, "item")
	fornecedor, ok2 := inteiroDaConsulta(r, "fornecedor")
	if !ok1 || !ok2 {
		security.RespondError(w, http.StatusBadRequest, "filtro inválido")
		return
	}
	q := r.URL.Query()
	out, err := h.uc.NCs(r.Context(), dom.FiltroNC{SoAbertas: q.Get("abertas") == "true", SemCapa: q.Get("sem_capa") == "true", ItemCode: item, SupplierCode: fornecedor})
	responder(w, http.StatusOK, out, err)
}
