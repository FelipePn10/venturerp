package handler

import (
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
		respondErrStatus(w, http.StatusNotFound, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *DeliveryPromiseParamsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateDeliveryPromiseParamsDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.uc.Save(r.Context(), dto, "system")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
