package handler

import (
	"errors"
	"net/http"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

func (h *QuestionHandler) FindQuestionByName(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		h.BadRequest(w, "Thw 'name' query parameter is required")
		return
	}

	question, err := h.findQuestionByNameUC.Execute(r.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, errorsuc.ErrInvalidQuestionName):
			h.BadRequest(w, "The 'name' query parameter is required")
			return
		case errors.Is(err, errorsuc.ErrQuestionNotFound):
			h.NotFound(w, "No question found with the given name")
			return
		default:
			h.InternalError(w, r, err)
			return
		}
	}

	h.OK(w, question, "Question found")
}
