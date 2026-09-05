package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/go-chi/chi/v5"
)

// ConsultStructure implements  Product Structure Consultation.
// GET /api/items/structure/consult?item_code=2210&levels=1&mask=...&effectiveness_date=2026-05-20
func (h *ItemQueryStructureHandler) ConsultStructure(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	itemCode := request.TextCode(q.Get("item_code"))
	if itemCode.String() == "" {
		jsonError(w, http.StatusBadRequest, "item_code é obrigatório")
		return
	}

	levels, _ := strconv.Atoi(q.Get("levels")) // 0 = all levels

	var effectivenessDate *time.Time
	if raw := q.Get("effectiveness_date"); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "informe a data de vigência no formato ano-mês-dia")
			return
		}
		effectivenessDate = &t
	}

	dto := request.ConsultStructureDTO{
		ItemCode:          itemCode,
		Mask:              q.Get("mask"),
		EffectivenessDate: effectivenessDate,
		Levels:            levels,
	}

	result, err := h.consultUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// WhereUsed implements implosão de estrutura — dado um componente, retorna todos os produtos que o utilizam.
// GET /api/items/structure/where-used/{itemCode}?levels=0
func (h *ItemQueryStructureHandler) WhereUsed(w http.ResponseWriter, r *http.Request) {
	itemCode := request.TextCode(chi.URLParam(r, "itemCode"))
	if itemCode.String() == "" {
		jsonError(w, http.StatusBadRequest, "itemCode é obrigatório")
		return
	}
	levels, _ := strconv.Atoi(r.URL.Query().Get("levels"))

	result, err := h.whereUsedUC.Execute(r.Context(), itemCode, levels)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemQueryStructureHandler) ResolveStructure(w http.ResponseWriter, r *http.Request) {
	code := request.TextCode(chi.URLParam(r, "itemCode"))
	if code.String() == "" {
		jsonError(w, http.StatusBadRequest, "itemCode é obrigatório")
		return
	}

	dto := request.ResolveStructureQueryDTO{
		ItemCode: code,
		Mask:     r.URL.Query().Get("mask"),
	}

	result, err := h.resolveUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
