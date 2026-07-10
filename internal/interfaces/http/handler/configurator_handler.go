package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/configurator_uc"
	"github.com/go-chi/chi/v5"
)

// ConfiguratorHandler serves the Product Configurator (Fase 1): Conjuntos/
// Variáveis, Características e Características do Item + geração de máscara.
type ConfiguratorHandler struct {
	uc *configurator_uc.ConfiguratorUseCase
}

func NewConfiguratorHandler(uc *configurator_uc.ConfiguratorUseCase) *ConfiguratorHandler {
	return &ConfiguratorHandler{uc: uc}
}

func cfgID(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, key), 10, 64)
}

// ─── Conjuntos ────────────────────────────────────────────────────────────────

func (h *ConfiguratorHandler) CreateSet(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCfgSetDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.CreateSet(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListSets(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListSets(r.Context(), r.URL.Query().Get("only_active") == "true")
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) GetSet(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}
	res, err := h.uc.GetSet(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateSet(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}
	var dto request.UpdateCfgSetDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.UpdateSet(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeactivateSet(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}
	if err := h.uc.DeactivateSet(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Variáveis ────────────────────────────────────────────────────────────────

func (h *ConfiguratorHandler) CreateVariable(w http.ResponseWriter, r *http.Request) {
	setID, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}
	var dto request.CreateCfgVariableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.SetID = setID
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.CreateVariable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListVariables(w http.ResponseWriter, r *http.Request) {
	setID, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}
	res, err := h.uc.ListVariablesBySet(r.Context(), setID, r.URL.Query().Get("only_active") == "true")
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) GetVariable(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "varId")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid variable id")
		return
	}
	res, err := h.uc.GetVariable(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateVariable(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "varId")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid variable id")
		return
	}
	var dto request.UpdateCfgVariableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.UpdateVariable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeactivateVariable(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "varId")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid variable id")
		return
	}
	if err := h.uc.DeactivateVariable(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConfiguratorHandler) SetVariableLanguage(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "varId")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid variable id")
		return
	}
	var dto request.CfgVariableLanguageDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.SetVariableLanguage(r.Context(), id, dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeleteVariableLanguage(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "langId")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid language id")
		return
	}
	if err := h.uc.DeleteVariableLanguage(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Características ───────────────────────────────────────────────────────────

func (h *ConfiguratorHandler) CreateCharacteristic(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCfgCharacteristicDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.CreateCharacteristic(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListCharacteristics(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListCharacteristics(r.Context(), r.URL.Query().Get("only_active") == "true")
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) GetCharacteristic(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid characteristic id")
		return
	}
	res, err := h.uc.GetCharacteristic(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateCharacteristic(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid characteristic id")
		return
	}
	var dto request.UpdateCfgCharacteristicDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.UpdateCharacteristic(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeactivateCharacteristic(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid characteristic id")
		return
	}
	if err := h.uc.DeactivateCharacteristic(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConfiguratorHandler) SetCharacteristicLanguage(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid characteristic id")
		return
	}
	var dto request.CfgCharacteristicLanguageDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.SetCharacteristicLanguage(r.Context(), id, dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeleteCharacteristicLanguage(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "langId")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid language id")
		return
	}
	if err := h.uc.DeleteCharacteristicLanguage(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Características do Item ───────────────────────────────────────────────────

func (h *ConfiguratorHandler) AddItemCharacteristic(w http.ResponseWriter, r *http.Request) {
	itemCode, err := cfgID(r, "itemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	var dto request.AddCfgItemCharacteristicDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ItemCode = itemCode
	res, err := h.uc.AddItemCharacteristic(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListItemCharacteristics(w http.ResponseWriter, r *http.Request) {
	itemCode, err := cfgID(r, "itemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	res, err := h.uc.ListItemCharacteristics(r.Context(), itemCode)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateItemCharacteristic(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateCfgItemCharacteristicDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.UpdateItemCharacteristic(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) RemoveItemCharacteristic(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.RemoveItemCharacteristic(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Geração de máscara ───────────────────────────────────────────────────────

func (h *ConfiguratorHandler) GenerateMask(w http.ResponseWriter, r *http.Request) {
	var dto request.CfgGenerateMaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.GenerateMask(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// GenerateMasks explodes the item's ESCOLHA characteristics into all valid
// combinations (produto cartesiano + restrições/dependências).
func (h *ConfiguratorHandler) GenerateMasks(w http.ResponseWriter, r *http.Request) {
	var dto request.CfgGenerateMasksDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.GenerateMasks(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// ─── Tipos de Descrição ───────────────────────────────────────────────────────

func (h *ConfiguratorHandler) CreateDescriptionType(w http.ResponseWriter, r *http.Request) {
	var dto request.CfgDescriptionTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.CreateDescriptionType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListDescriptionTypes(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListDescriptionTypes(r.Context(), r.URL.Query().Get("only_active") == "true")
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) GetDescriptionType(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	res, err := h.uc.GetDescriptionType(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateDescriptionType(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.CfgDescriptionTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.UpdateDescriptionType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeactivateDescriptionType(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeactivateDescriptionType(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Descrição de Itens Configurados ──────────────────────────────────────────

func (h *ConfiguratorHandler) CreateItemDescription(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCfgItemDescriptionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.CreateItemDescription(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListItemDescriptions(w http.ResponseWriter, r *http.Request) {
	itemCode, err := cfgID(r, "itemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	res, err := h.uc.ListItemDescriptionsByItem(r.Context(), itemCode)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) GetItemDescription(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	res, err := h.uc.GetItemDescription(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateItemDescriptionLines(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateCfgItemDescriptionLinesDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.UpdateItemDescriptionLines(r.Context(), id, dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) ReloadItemDescription(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	res, err := h.uc.ReloadLines(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) RenderItemDescription(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.CfgRenderDescriptionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.RenderItemDescription(r.Context(), id, dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeleteItemDescription(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeleteItemDescription(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Regras de Variáveis Equivalentes ─────────────────────────────────────────

func (h *ConfiguratorHandler) CreateEquivalentRule(w http.ResponseWriter, r *http.Request) {
	var dto request.CfgEquivalentRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.CreateEquivalentRule(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListEquivalentRules(w http.ResponseWriter, r *http.Request) {
	parentItemCode, err := cfgID(r, "parentItemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid parent item code")
		return
	}
	res, err := h.uc.ListEquivalentRulesByParent(r.Context(), parentItemCode, r.URL.Query().Get("only_active") == "true")
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) GetEquivalentRule(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	res, err := h.uc.GetEquivalentRule(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateEquivalentRule(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.CfgEquivalentRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.UpdateEquivalentRule(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeactivateEquivalentRule(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeactivateEquivalentRule(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConfiguratorHandler) ApplyEquivalent(w http.ResponseWriter, r *http.Request) {
	var dto request.CfgApplyEquivalentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.ApplyEquivalent(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// ─── Regras de Itens Configurados ─────────────────────────────────────────────

func (h *ConfiguratorHandler) CreateItemRule(w http.ResponseWriter, r *http.Request) {
	var dto request.CfgItemRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.CreateItemRule(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListItemRules(w http.ResponseWriter, r *http.Request) {
	itemCode, err := cfgID(r, "itemCode")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	res, err := h.uc.ListItemRulesByItem(r.Context(), itemCode, r.URL.Query().Get("only_active") == "true")
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) GetItemRule(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	res, err := h.uc.GetItemRule(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) UpdateItemRule(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.CfgItemRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.UpdateItemRule(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeleteItemRule(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeleteItemRule(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConfiguratorHandler) EvaluateItemRules(w http.ResponseWriter, r *http.Request) {
	var dto request.CfgEvaluateItemRulesDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.EvaluateItemRules(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// ListCharacteristicItems lists the items that use a characteristic (Botão Itens Vinculados).
func (h *ConfiguratorHandler) ListCharacteristicItems(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid characteristic id")
		return
	}
	res, err := h.uc.ListItemsByCharacteristic(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"characteristic_id": id, "item_codes": res})
}

// ─── Botão Itens do Tipo Recebimento ──────────────────────────────────────────

func (h *ConfiguratorHandler) AddReceivingItem(w http.ResponseWriter, r *http.Request) {
	charID, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid characteristic id")
		return
	}
	var dto request.CfgReceivingItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.AddReceivingItem(r.Context(), charID, dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ConfiguratorHandler) ListReceivingItems(w http.ResponseWriter, r *http.Request) {
	charID, err := cfgID(r, "id")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid characteristic id")
		return
	}
	res, err := h.uc.ListReceivingItems(r.Context(), charID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ConfiguratorHandler) DeleteReceivingItem(w http.ResponseWriter, r *http.Request) {
	id, err := cfgID(r, "recvId")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeleteReceivingItem(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
