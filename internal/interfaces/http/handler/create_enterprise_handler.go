package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise"
)

func (h *EnterpriseHandler) CreateEnterprise(w http.ResponseWriter, r *http.Request) {
	var req request.CreateEnterpriseDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	enterprise, err := mapper.ToEnterpriseEntity(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	created, err := h.createEnterpriseUC.Execute(r.Context(), enterprise)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "enterprise created succesfully")
}
