package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func (h *QuestionHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	var req request.CreateQuestionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	question, err := h.createQuestionUC.Execute(r.Context(), req)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, question, "Created question success")
}
