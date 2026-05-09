package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/employee"
)

func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req request.CreateEmployeeDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	employee, err := mapper.ToEmployeeEntity(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	created, err := h.createEmployeeUC.Execute(r.Context(), employee)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "employee created succesfully")
}
