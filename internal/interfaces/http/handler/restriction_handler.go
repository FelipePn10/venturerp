package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	restrictionEntity "github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *RestrictionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRestrictionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *RestrictionHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *RestrictionHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RestrictionHandler) GetByItem(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	results, err := h.getByItemUC.Execute(r.Context(), itemCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *RestrictionHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.UpdateRestrictionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	res := &restrictionEntity.Restriction{
		Code:                 dto.Code,
		Situation:            restrictionEntity.RestrictionSituation(dto.Situation),
		CustomerCode:         dto.CustomerCode,
		ItemCode:             dto.ItemCode,
		ReasonCode:           dto.ReasonCode,
		ClassificationType:   dto.ClassificationType,
		ClassificationOrigin: dto.ClassificationOrigin,
		DivisionID:           dto.DivisionID,
		Weight:               restrictionEntity.CalcWeight(dto.CustomerCode, dto.ItemCode, dto.ClassificationType, dto.DivisionID),
	}
	result, err := h.updateUC.Execute(r.Context(), res)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RestrictionHandler) GetByCustomer(w http.ResponseWriter, r *http.Request) {
	customerCode, err := strconv.ParseInt(chi.URLParam(r, "customerCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid customer code")
		return
	}
	results, err := h.getByCustomerUC.Execute(r.Context(), customerCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Evaluate evaluates all active restrictions against the provided answers and
// returns a ready-to-use response that the frontend can apply directly:
//   - invalid_question_ids: questions to hide and whose answers must be discarded
//   - locked_values:        question_id → value that must be forced on the field
//   - cleaned_answers:      the input answers after all determinants are applied
//
// The frontend should call this endpoint on every answer change to keep the
// visible question set and locked values in sync, regardless of answer order.
func (h *RestrictionHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var dto request.EvaluateRestrictionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if dto.Answers == nil {
		dto.Answers = map[int64]string{}
	}
	result, err := h.evaluateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RestrictionHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.deactivateUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
