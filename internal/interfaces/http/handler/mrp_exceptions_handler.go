package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type MRPExceptionsHandler struct {
	uc     *mrp_uc.NotifyExceptionsUseCase
	handle *mrp_calculation_uc.HandleMRPExceptionUseCase
}

func NewMRPExceptionsHandler(uc *mrp_uc.NotifyExceptionsUseCase, handle *mrp_calculation_uc.HandleMRPExceptionUseCase) *MRPExceptionsHandler {
	return &MRPExceptionsHandler{uc: uc, handle: handle}
}

// Handle executa a ação do planejador sobre a exceção {code}: REPROGRAMAR,
// CANCELAR ou TRATAR.
func (h *MRPExceptionsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondErrorCode(w, http.StatusBadRequest, "MRP_EXCECAO_INVALIDA", "código da exceção inválido")
		return
	}
	var dto request.HandleMRPExceptionDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.handle.Execute(r.Context(), code, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

func (h *MRPExceptionsHandler) Notify(w http.ResponseWriter, r *http.Request) {
	var dto mrp_uc.NotifyExceptionsDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
