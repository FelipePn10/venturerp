package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/unit_of_measure_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// UnitOfMeasureHandler: catálogo de unidades, sinônimos e conversões padrão.
type UnitOfMeasureHandler struct {
	uc *unit_of_measure_uc.UseCase
}

func NewUnitOfMeasureHandler(uc *unit_of_measure_uc.UseCase) *UnitOfMeasureHandler {
	return &UnitOfMeasureHandler{uc: uc}
}

func (h *UnitOfMeasureHandler) List(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListUnits(r.Context(), r.URL.Query().Get("include_inactive") == "true")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *UnitOfMeasureHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.UnitOfMeasureDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	res, err := h.uc.CreateUnit(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *UnitOfMeasureHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UnitOfMeasureDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	res, err := h.uc.UpdateUnit(r.Context(), chi.URLParam(r, "code"), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *UnitOfMeasureHandler) Activate(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, true)
}

func (h *UnitOfMeasureHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, false)
}

func (h *UnitOfMeasureHandler) setActive(w http.ResponseWriter, r *http.Request, active bool) {
	if err := h.uc.SetActive(r.Context(), chi.URLParam(r, "code"), active); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UnitOfMeasureHandler) ListAliases(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListAliases(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *UnitOfMeasureHandler) CreateAlias(w http.ResponseWriter, r *http.Request) {
	var dto request.UnitAliasDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	res, err := h.uc.CreateAlias(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *UnitOfMeasureHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	if err := h.uc.DeleteAlias(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UnitOfMeasureHandler) ListStandard(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListStandardConversions(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *UnitOfMeasureHandler) SaveStandard(w http.ResponseWriter, r *http.Request) {
	var dto request.StandardConversionDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	res, err := h.uc.SaveStandardConversion(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *UnitOfMeasureHandler) DeleteStandard(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	if err := h.uc.DeleteStandardConversion(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Resolve: GET ?text=UND&text=PÇ&supplier=12 — como o sistema lê cada texto.
func (h *UnitOfMeasureHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	if !queryValida(w, r, filtrosDaQuery{Inteiros: []string{"supplier"}}) {
		return
	}
	q := r.URL.Query()
	textos := []string{}
	for _, t := range q["text"] {
		if t = strings.TrimSpace(t); t != "" {
			textos = append(textos, t)
		}
	}
	var forn *int64
	if s := strings.TrimSpace(q.Get("supplier")); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil && v > 0 {
			forn = &v
		}
	}
	res, err := h.uc.Resolve(r.Context(), forn, textos)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}
