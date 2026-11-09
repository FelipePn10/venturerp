package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type ItemConversionHandler struct {
	uc *item_conversion_uc.ItemConversionUseCase
}

func NewItemConversionHandler(uc *item_conversion_uc.ItemConversionUseCase) *ItemConversionHandler {
	return &ItemConversionHandler{uc: uc}
}

func (h *ItemConversionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateItemConversionDTO
	if !security.DecodeBody(w, r, &dto) {
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
	if err := h.uc.DeleteWithReason(r.Context(), id, r.URL.Query().Get("reason")); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Update altera uma conversão (PUT /{id}): corrige o fator ou abre nova vigência.
func (h *ItemConversionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	var dto request.UpdateItemConversionDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	res, err := h.uc.Update(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// itemDaRota resolve o {itemCode} (código de negócio) da rota.
func (h *ItemConversionHandler) itemDaRota(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := strings.TrimSpace(chi.URLParam(r, "itemCode"))
	if raw == "" {
		jsonError(w, http.StatusBadRequest, "informe o código do item")
		return 0, false
	}
	itemCode, err := h.uc.ResolveItem(r.Context(), request.TextCode(raw))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return 0, false
	}
	return itemCode, true
}

// Versions: todas as versões, inclusive as encerradas.
func (h *ItemConversionHandler) Versions(w http.ResponseWriter, r *http.Request) {
	itemCode, ok := h.itemDaRota(w, r)
	if !ok {
		return
	}
	res, err := h.uc.ListVersions(r.Context(), itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) History(w http.ResponseWriter, r *http.Request) {
	itemCode, ok := h.itemDaRota(w, r)
	if !ok {
		return
	}
	res, err := h.uc.History(r.Context(), itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) Coverage(w http.ResponseWriter, r *http.Request) {
	itemCode, ok := h.itemDaRota(w, r)
	if !ok {
		return
	}
	res, err := h.uc.Coverage(r.Context(), itemCode, r.URL.Query().Get("mask"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) Impact(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	res, err := h.uc.Impact(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// Simulate: GET ?item=&from=&to=&qty=&mask=&lot=&date= — devolve o caminho.
func (h *ItemConversionHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	if !queryValida(w, r, filtrosDaQuery{Decimais: []string{"qty"}, Datas: []string{"date"}}) {
		return
	}
	q := r.URL.Query()
	rawItem, from, to := strings.TrimSpace(q.Get("item")), strings.TrimSpace(q.Get("from")), strings.TrimSpace(q.Get("to"))
	if rawItem == "" || from == "" || to == "" {
		jsonError(w, http.StatusBadRequest, "informe o item, a unidade de origem e a de destino")
		return
	}
	qty := decimal.NewFromInt(1)
	if s := strings.TrimSpace(q.Get("qty")); s != "" {
		qty, _ = decimal.NewFromString(s)
	}
	itemCode, err := h.uc.ResolveItem(r.Context(), request.TextCode(rawItem))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	res, err := h.uc.Simulate(r.Context(), itemCode, q.Get("mask"), q.Get("lot"), q.Get("date"), from, to, qty)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) Lots(w http.ResponseWriter, r *http.Request) {
	itemCode, ok := h.itemDaRota(w, r)
	if !ok {
		return
	}
	res, err := h.uc.ListLots(r.Context(), itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) SaveLot(w http.ResponseWriter, r *http.Request) {
	var dto request.LotConversionDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	res, err := h.uc.SaveLot(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) DeleteLot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	if err := h.uc.DeleteLot(r.Context(), id, r.URL.Query().Get("reason")); err != nil {
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
	if !queryValida(w, r, filtrosDaQuery{Decimais: []string{"qty"}}) {
		return
	}
	q := r.URL.Query()
	from := strings.TrimSpace(q.Get("from"))
	to := strings.TrimSpace(q.Get("to"))
	qty, _ := decimal.NewFromString(q.Get("qty")) // queryValida já recusou o inválido
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
		Factor:         factor.InexactFloat64(),
		Quantity:       qty.InexactFloat64(),
		ConvertedQty:   converted.InexactFloat64(),
	})
}
