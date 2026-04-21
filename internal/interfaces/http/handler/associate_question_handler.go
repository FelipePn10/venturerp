package handler

import (
	"encoding/json"
	"net/http"

	applicationreq "github.com/FelipePn10/panossoerp/internal/application/dto/request"
	internalreq "github.com/FelipePn10/panossoerp/internal/infrastructure/http/request"
)

func (h *AssociateByQuestionItemHandler) AssociateQuestions(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req internalreq.AssociateProductQuestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ItemCode <= 0 || len(req.Questions) == 0 {
		http.Error(w, "questions cannot be empty", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	// one product N questions
	for _, q := range req.Questions {
		dto := applicationreq.AssociateByQuestionItemRequestDTO{
			ItemCode:   req.ItemCode,
			QuestionID: q.QuestionID,
			Position:   q.Position,
		}
		if err := h.associateByQuestionProductUC.Execute(ctx, dto); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}
