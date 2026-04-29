package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type DeliveryPromiseParamsHandler struct {
	uc *delivery_promise_params_uc.ManageDeliveryPromiseParamsUseCase
}

func (h *DeliveryPromiseParamsHandler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.Get(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *DeliveryPromiseParamsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateDeliveryPromiseParamsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Save(r.Context(), dto, "system")
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
