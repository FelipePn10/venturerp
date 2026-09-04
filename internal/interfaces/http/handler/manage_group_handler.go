package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/go-chi/chi/v5"
)

func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(chi.URLParam(r, "code"))
	if err != nil {
		h.BadRequest(w, "invalid code")
		return
	}
	result, err := h.getGroupUC.Execute(r.Context(), code)
	if err != nil {
		h.NotFound(w, "grupo PDM não encontrado")
		return
	}
	h.OK(w, result)
}

func (h *GroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	results, err := h.listGroupsUC.Execute(r.Context())
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	page, pageSize := 1, 100
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 {
			h.BadRequest(w, "page deve ser positivo")
			return
		}
		page = value
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 || value > 200 {
			h.BadRequest(w, "page_size deve estar entre 1 e 200")
			return
		}
		pageSize = value
	}
	total := len(results)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Page", strconv.Itoa(page))
	w.Header().Set("X-Page-Size", strconv.Itoa(pageSize))
	results = results[start:end]
	h.OK(w, results)
}

type updateGroupBody struct {
	Description string `json:"description"`
}

func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(chi.URLParam(r, "code"))
	if err != nil {
		h.BadRequest(w, "invalid code")
		return
	}
	var body updateGroupBody
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		h.BadRequest(w, "corpo inválido; enterprise_id e created_by não são aceitos")
		return
	}
	result, err := h.updateGroupUC.Execute(r.Context(), code, body.Description)
	if err != nil {
		if v, ok := errorsuc.AsValidation(err); ok {
			h.UnprocessableEntity(w, v.Error())
			return
		}
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, result, "group updated successfully")
}
