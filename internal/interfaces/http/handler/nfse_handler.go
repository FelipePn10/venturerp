package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// NFSeHandler exposes the NFS-e (service invoice) endpoints.
type NFSeHandler struct {
	createUC    *nfse_uc.CreateNFSeUseCase
	authorizeUC *nfse_uc.AuthorizeNFSeUseCase
	cancelUC    *nfse_uc.CancelNFSeUseCase
	listUC      *nfse_uc.ListNFSeUseCase
	getUC       *nfse_uc.GetNFSeUseCase
}

func NewNFSeHandler(
	createUC *nfse_uc.CreateNFSeUseCase,
	authorizeUC *nfse_uc.AuthorizeNFSeUseCase,
	cancelUC *nfse_uc.CancelNFSeUseCase,
	listUC *nfse_uc.ListNFSeUseCase,
	getUC *nfse_uc.GetNFSeUseCase,
) *NFSeHandler {
	return &NFSeHandler{createUC: createUC, authorizeUC: authorizeUC, cancelUC: cancelUC, listUC: listUC, getUC: getUC}
}

func (h *NFSeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateNFSeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *NFSeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.authorizeUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *NFSeHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.CancelNFSeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.cancelUC.Execute(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *NFSeHandler) List(w http.ResponseWriter, r *http.Request) {
	result, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *NFSeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
