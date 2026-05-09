package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func (h *QuestionOptionHandler) CreateQuestionOptionHandler(w http.ResponseWriter, r *http.Request) {
	var req request.CreateQuestionOptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	questionOption, err := h.createQuestionOptionUC.Execute(r.Context(), req)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, questionOption, "Created question option success")
}
