package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/metrologia_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/metrologia"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// MetrologiaHandler expõe instrumentos, calibrações e o impacto da reprovação.
type MetrologiaHandler struct{ uc *metrologia_uc.UseCase }

func NewMetrologiaHandler(uc *metrologia_uc.UseCase) *MetrologiaHandler {
	return &MetrologiaHandler{uc: uc}
}

func (h *MetrologiaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dias, _ := strconv.Atoi(q.Get("dias"))
	out, err := h.uc.Listar(r.Context(), dom.Filtro{Grupo: q.Get("grupo"), Situacao: q.Get("situacao"), Dias: dias, Inclusive: q.Get("baixados") == "true"})
	responder(w, http.StatusOK, out, err)
}

func (h *MetrologiaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var dto metrologia_uc.InstrumentoDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Salvar(r.Context(), 0, dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *MetrologiaHandler) instrumento(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "instrumento inválido")
	}
	return id, ok
}

func (h *MetrologiaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.instrumento(w, r); ok {
		out, err := h.uc.Obter(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *MetrologiaHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.instrumento(w, r)
	var dto metrologia_uc.InstrumentoDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Salvar(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}

func (h *MetrologiaHandler) Calibrar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.instrumento(w, r)
	var dto metrologia_uc.CalibracaoDTO
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.RegistrarCalibracao(r.Context(), id, dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *MetrologiaHandler) Baixar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.instrumento(w, r)
	var dto struct {
		Motivo string `json:"motivo"`
	}
	if !ok || !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Baixar(r.Context(), id, dto.Motivo)
	responder(w, http.StatusOK, out, err)
}

func (h *MetrologiaHandler) Impacto(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.instrumento(w, r); ok {
		out, err := h.uc.Impacto(r.Context(), id, r.URL.Query().Get("desde"))
		responder(w, http.StatusOK, out, err)
	}
}
