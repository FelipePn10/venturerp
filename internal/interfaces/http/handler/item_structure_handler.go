package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/go-chi/chi/v5"
)

func (h *ItemStructureHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateStructureComponentDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}

	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, result)
}

func (h *ItemStructureHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateStructureComponentDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}

	if dto.ParentCode == 0 || dto.ChildCode == 0 {
		jsonError(w, http.StatusBadRequest, "parent_code and child_code are required")
		return
	}

	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// Delete removes a structure component identified by code.
func (h *ItemStructureHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code, err := parseCode(r, "code")
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = code

	w.WriteHeader(http.StatusNoContent)
}

// GetTree returns the BOM tree for a root item.
func (h *ItemStructureHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	rootItemCode, err := parseCode(r, "rootItemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	dto := request.GetStructureTreeDTO{
		RootItemCode: rootItemCode,
	}

	result, err := h.treeUC.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// GetAllDirectChildren returns direct children of a structure component.
func (h *ItemStructureHandler) GetAllDirectChildren(w http.ResponseWriter, r *http.Request) {
	parentItemCode, err := parseCode(r, "parentItemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	dto := request.GetAllDirectChildrenDTO{
		ParentItemCode: parentItemCode,
	}

	result, err := h.getAllStructure.Execute(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

//// ResolveForMask resolves BOM tree using a mask configuration.
//func (h *ItemStructureHandler) ResolveForMask(w http.ResponseWriter, r *http.Request) {
//	rootItemCode, err := parseCode(r, "rootItemCode")
//	if err != nil {
//		jsonError(w, http.StatusBadRequest, err.Error())
//		return
//	}
//
//	maskValue := r.URL.Query().Get("mask")
//	if maskValue == "" {
//		jsonError(w, http.StatusBadRequest, "query param 'mask' is required (example: ?mask=100%23100%2350)")
//		return
//	}
//
//	dto := request.ResolveStructureForMaskDTO{
//		RootItemCode:  rootItemCode,
//		RootMaskValue: maskValue,
//	}
//
//	result, err := h.resolveUC.Execute(r.Context(), dto)
//	if err != nil {
//		jsonError(w, http.StatusUnprocessableEntity, err.Error())
//		return
//	}
//
//	jsonResponse(w, http.StatusOK, result)
//}

// parseCode extracts and validates a numeric code from URL params.
func parseCode(r *http.Request, param string) (int64, error) {
	raw := chi.URLParam(r, param)

	code, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || code <= 0 {
		return 0, fmt.Errorf("parameter '%s' must be a positive integer", param)
	}

	return code, nil
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	if status >= http.StatusInternalServerError {
		msg = "internal server error"
	}
	jsonResponse(w, status, map[string]string{"error": msg})
}
