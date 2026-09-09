package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type ItemConversionHandler struct {
	uc *item_conversion_uc.ItemConversionUseCase
}

func NewItemConversionHandler(uc *item_conversion_uc.ItemConversionUseCase) *ItemConversionHandler {
	return &ItemConversionHandler{uc: uc}
}

func (h *ItemConversionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateItemConversionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

// ListByItem recebe o código de negócio do item (texto), como a tela o conhece.
func (h *ItemConversionHandler) ListByItem(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(chi.URLParam(r, "itemCode"))
	if raw == "" {
		jsonError(w, http.StatusBadRequest, "informe o código do item")
		return
	}
	itemCode, err := h.uc.ResolveItem(r.Context(), request.TextCode(raw))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	res, err := h.uc.ListByItem(r.Context(), itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	if err := h.uc.Delete(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Convert resolves a quantity conversion: GET ?item=&from=&to=&qty=
// conversionResult é o retorno da conversão. Era um map anônimo: o contrato não
// aparecia em lugar nenhum e a auditoria de drift acusava as chaves como
// inexistentes no backend.
type conversionResult struct {
	ItemCode       string  `json:"item_code"`
	LegacyItemCode int64   `json:"legacy_item_code"`
	FromUOM        string  `json:"from_uom"`
	ToUOM          string  `json:"to_uom"`
	Factor         float64 `json:"factor"`
	Quantity       float64 `json:"quantity"`
	ConvertedQty   float64 `json:"converted_qty"`
}

func (h *ItemConversionHandler) Convert(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := strings.TrimSpace(q.Get("from"))
	to := strings.TrimSpace(q.Get("to"))
	qty, _ := strconv.ParseFloat(q.Get("qty"), 64)
	mask := q.Get("mask")
	rawItem := strings.TrimSpace(q.Get("item"))
	if rawItem == "" || from == "" || to == "" {
		jsonError(w, http.StatusBadRequest, "informe o item, a unidade de origem e a de destino")
		return
	}
	itemCode, err := h.uc.ResolveItem(r.Context(), request.TextCode(rawItem))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	factor, found, err := h.uc.FactorConfigured(r.Context(), itemCode, mask, from, to)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	if !found {
		jsonError(w, http.StatusNotFound, item_conversion_uc.ErrNoConversion.Error())
		return
	}
	converted, convertedFound, convertErr := h.uc.ConvertQuantityConfigured(r.Context(), itemCode, mask, qty, from, to)
	if convertErr != nil {
		jsonError(w, http.StatusUnprocessableEntity, convertErr.Error())
		return
	}
	if !convertedFound {
		jsonError(w, http.StatusNotFound, item_conversion_uc.ErrNoConversion.Error())
		return
	}
	jsonResponse(w, http.StatusOK, conversionResult{
		ItemCode:       rawItem,
		LegacyItemCode: itemCode,
		FromUOM:        from,
		ToUOM:          to,
		Factor:         factor,
		Quantity:       qty,
		ConvertedQty:   converted,
	})
}
