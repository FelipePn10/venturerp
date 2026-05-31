package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc"
	"github.com/go-chi/chi/v5"
)

type ICMSReductionHandler struct {
	reductionUC  *fiscal_params_uc.ICMSReductionSubstitutionUseCase
	additionalUC *fiscal_params_uc.ICMSSummaryAdditionalUseCase
	stRestUC     *fiscal_params_uc.ICMSSTRestitutionUseCase
	specialNoteUC *fiscal_params_uc.SpecialAdjustmentNoteUseCase
}

func NewICMSReductionHandler(
	reductionUC *fiscal_params_uc.ICMSReductionSubstitutionUseCase,
	additionalUC *fiscal_params_uc.ICMSSummaryAdditionalUseCase,
	stRestUC *fiscal_params_uc.ICMSSTRestitutionUseCase,
	specialNoteUC *fiscal_params_uc.SpecialAdjustmentNoteUseCase,
) *ICMSReductionHandler {
	return &ICMSReductionHandler{
		reductionUC:   reductionUC,
		additionalUC:  additionalUC,
		stRestUC:      stRestUC,
		specialNoteUC: specialNoteUC,
	}
}

// ─── ICMS Reduction / Substitution ───────────────────────────────────────────

func (h *ICMSReductionHandler) CreateReduction(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.ICMSReductionSubstitution
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.reductionUC.Create(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ICMSReductionHandler) UpdateReduction(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.ICMSReductionSubstitution
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.reductionUC.Update(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) GetReduction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.reductionUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) ListReductions(w http.ResponseWriter, r *http.Request) {
	uf := r.URL.Query().Get("uf")
	onlyActive := r.URL.Query().Get("active") != "false"
	var itemID *int64
	if raw := r.URL.Query().Get("item_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			itemID = &v
		}
	}
	result, err := h.reductionUC.List(r.Context(), uf, itemID, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) FindReduction(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	uf := q.Get("uf")
	opType := fiscalEntity.ICMSOperationType(q.Get("op_type"))
	var itemID, customerID *int64
	if raw := q.Get("item_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			itemID = &v
		}
	}
	if raw := q.Get("customer_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			customerID = &v
		}
	}
	result, err := h.reductionUC.Find(r.Context(), uf, itemID, customerID, opType)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── ICMS Summary Entry Additionals (Aba Adicionais) ─────────────────────────

func (h *ICMSReductionHandler) AddSummaryAdditional(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.ICMSSummaryEntryAdditional
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.additionalUC.Add(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ICMSReductionHandler) ListSummaryAdditionals(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.additionalUC.List(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── ICMS ST Restitution ──────────────────────────────────────────────────────

func (h *ICMSReductionHandler) CreateSTRestitution(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.ICMSSTRestitution
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.stRestUC.Create(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ICMSReductionHandler) UpdateSTRestitution(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.ICMSSTRestitution
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.stRestUC.Update(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) GetSTRestitution(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.stRestUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) ListSTRestitutions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	empresaID, _ := strconv.Atoi(q.Get("empresa_id"))
	period := q.Get("period")
	uf := q.Get("uf")
	result, err := h.stRestUC.List(r.Context(), empresaID, period, uf)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Special Adjustment Notes ─────────────────────────────────────────────────

func (h *ICMSReductionHandler) CreateSpecialNote(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.SpecialAdjustmentNote
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.specialNoteUC.Create(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ICMSReductionHandler) UpdateSpecialNote(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.SpecialAdjustmentNote
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.specialNoteUC.Update(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) GetSpecialNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.specialNoteUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) ListSpecialNotes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	empresaID, _ := strconv.Atoi(q.Get("empresa_id"))
	period := q.Get("period")
	result, err := h.specialNoteUC.List(r.Context(), empresaID, period)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ICMSReductionHandler) AddSpecialNoteItem(w http.ResponseWriter, r *http.Request) {
	var body fiscalEntity.SpecialAdjustmentNoteItem
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64); err == nil {
		body.NoteID = id
	}
	result, err := h.specialNoteUC.AddItem(r.Context(), &body)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ICMSReductionHandler) ListSpecialNoteItems(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.specialNoteUC.ListItems(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
