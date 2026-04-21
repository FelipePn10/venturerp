package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func (h *ItemQueryStructureHandler) ResolveStructure(w http.ResponseWriter, r *http.Request) {
	code, err := parseCode(r, "itemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
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
