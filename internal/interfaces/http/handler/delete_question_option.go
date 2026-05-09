package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *QuestionOptionHandler) DeleteQuestionOption(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.deleteQuestionOptionUC.Execute(r.Context(), id); err != nil {
		h.InternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
