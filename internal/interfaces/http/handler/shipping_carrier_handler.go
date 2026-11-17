package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipping_carrier_uc"
	carrierentity "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/entity"
	carrierrepo "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type ShippingCarrierHandler struct {
	uc *shipping_carrier_uc.UseCase
}

func NewShippingCarrierHandler(uc *shipping_carrier_uc.UseCase) *ShippingCarrierHandler {
	return &ShippingCarrierHandler{uc: uc}
}

func (h *ShippingCarrierHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := carrierrepo.Filtro{
		Busca:         q.Get("q"),
		UF:            q.Get("uf"),
		Modal:         q.Get("modal"),
		SomenteAtivas: q.Get("only_active") == "true",
	}
	lista, err := h.uc.Listar(r.Context(), f)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, lista)
}

func (h *ShippingCarrierHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "id")
	if !ok {
		return
	}
	out, err := h.uc.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// GetBySupplier serve a tela de fornecedor: "este fornecedor já é transportadora?"
func (h *ShippingCarrierHandler) GetBySupplier(w http.ResponseWriter, r *http.Request) {
	code, ok := parseCarrierID(w, r, "supplierCode")
	if !ok {
		return
	}
	out, err := h.uc.ObterPorFornecedor(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ShippingCarrierHandler) Create(w http.ResponseWriter, r *http.Request) {
	dto, ok := decodeCarrier(w, r)
	if !ok {
		return
	}
	out, err := h.uc.Salvar(r.Context(), 0, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

func (h *ShippingCarrierHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "id")
	if !ok {
		return
	}
	dto, ok := decodeCarrier(w, r)
	if !ok {
		return
	}
	out, err := h.uc.Salvar(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ShippingCarrierHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "id")
	if !ok {
		return
	}
	var corpo struct {
		IsActive *bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.IsActive == nil {
		security.RespondError(w, http.StatusBadRequest, "informe se a transportadora fica ativa ou inativa")
		return
	}
	if err := h.uc.DefinirSituacao(r.Context(), id, *corpo.IsActive); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]any{"id": id, "is_active": *corpo.IsActive})
}

func (h *ShippingCarrierHandler) ListOccurrences(w http.ResponseWriter, r *http.Request) {
	if !queryValida(w, r, filtrosDaQuery{Inteiros: []string{"limit"}}) {
		return
	}
	id, ok := parseCarrierID(w, r, "id")
	if !ok {
		return
	}
	limite, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	lista, err := h.uc.ListarOcorrencias(r.Context(), id, limite)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, lista)
}

func (h *ShippingCarrierHandler) CreateOccurrence(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "id")
	if !ok {
		return
	}
	var dto request.RegistrarOcorrenciaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "não foi possível ler a ocorrência enviada")
		return
	}
	out, err := h.uc.RegistrarOcorrencia(r.Context(), id, &dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

// Quote compara o frete das transportadoras que atendem o destino.
func (h *ShippingCarrierHandler) Quote(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	carga := carrierentity.Carga{
		UF:     strings.ToUpper(strings.TrimSpace(q.Get("uf"))),
		Cidade: strings.TrimSpace(q.Get("city")),
		CEP:    strings.TrimSpace(q.Get("cep")),
	}
	var err error
	if carga.PesoKg, err = decimalDaQuery(q.Get("weight_kg")); err != nil {
		security.RespondError(w, http.StatusBadRequest, "peso inválido")
		return
	}
	if carga.VolumeM3, err = decimalDaQuery(q.Get("volume_m3")); err != nil {
		security.RespondError(w, http.StatusBadRequest, "volume inválido")
		return
	}
	if carga.ValorMercadoria, err = decimalDaQuery(q.Get("cargo_value")); err != nil {
		security.RespondError(w, http.StatusBadRequest, "valor da carga inválido")
		return
	}
	carga.ProdutoPerigoso = q.Get("dangerous") == "true"
	if base := strings.TrimSpace(q.Get("base_date")); base != "" {
		d, err := time.Parse("2006-01-02", base)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "data base inválida: use AAAA-MM-DD")
			return
		}
		carga.Base = d
	}
	out, err := h.uc.Cotar(r.Context(), carga)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func decodeCarrier(w http.ResponseWriter, r *http.Request) (*request.SalvarTransportadoraDTO, bool) {
	var dto request.SalvarTransportadoraDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "não foi possível ler o cadastro enviado")
		return nil, false
	}
	return &dto, true
}

func parseCarrierID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id <= 0 {
		security.RespondError(w, http.StatusBadRequest, "identificador inválido")
		return 0, false
	}
	return id, true
}

func decimalDaQuery(v string) (decimal.Decimal, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(v)
}

// Usage diz onde a transportadora está gravada: a tela usa antes de oferecer
// excluir ou inativar.
func (h *ShippingCarrierHandler) Usage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "id")
	if !ok {
		return
	}
	out, err := h.uc.Uso(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ShippingCarrierHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "id")
	if !ok {
		return
	}
	if err := h.uc.Excluir(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShippingCarrierHandler) ListVehicleTypes(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarTiposVeiculo(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ShippingCarrierHandler) SaveVehicleType(w http.ResponseWriter, r *http.Request) {
	var id int64
	if chi.URLParam(r, "typeId") != "" {
		var ok bool
		if id, ok = parseCarrierID(w, r, "typeId"); !ok {
			return
		}
	}
	var dto request.SalvarTipoVeiculoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "não foi possível ler o tipo de veículo enviado")
		return
	}
	out, err := h.uc.SalvarTipoVeiculo(r.Context(), id, &dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ShippingCarrierHandler) DeleteVehicleType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "typeId")
	if !ok {
		return
	}
	if err := h.uc.ExcluirTipoVeiculo(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShippingCarrierHandler) ListOccurrenceTypes(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarTiposOcorrencia(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ShippingCarrierHandler) SaveOccurrenceType(w http.ResponseWriter, r *http.Request) {
	var id int64
	if chi.URLParam(r, "typeId") != "" {
		var ok bool
		if id, ok = parseCarrierID(w, r, "typeId"); !ok {
			return
		}
	}
	var dto request.SalvarTipoOcorrenciaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "não foi possível ler o tipo de ocorrência enviado")
		return
	}
	out, err := h.uc.SalvarTipoOcorrencia(r.Context(), id, &dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ShippingCarrierHandler) DeleteOccurrenceType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCarrierID(w, r, "typeId")
	if !ok {
		return
	}
	if err := h.uc.ExcluirTipoOcorrencia(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
