package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representative_uc"
	reprepo "github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/go-chi/chi/v5"
)

type RepresentativeHandler struct {
	uc *representative_uc.UseCase
}

func NewRepresentativeHandler(uc *representative_uc.UseCase) *RepresentativeHandler {
	return &RepresentativeHandler{uc: uc}
}

func (h *RepresentativeHandler) CreateType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRepresentativeTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateType(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *RepresentativeHandler) UpdateType(w http.ResponseWriter, r *http.Request) {
	code, ok := parseRepresentativeCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateRepresentativeTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.uc.UpdateType(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RepresentativeHandler) GetType(w http.ResponseWriter, r *http.Request) {
	code, ok := parseRepresentativeCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.GetType(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RepresentativeHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("active") != "false"
	result, err := h.uc.ListTypes(r.Context(), onlyActive)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RepresentativeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRepresentativeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *RepresentativeHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, ok := parseRepresentativeCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateRepresentativeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.uc.Update(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RepresentativeHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, ok := parseRepresentativeCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.Get(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RepresentativeHandler) List(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.List(r.Context(), parseRepresentativeFilter(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RepresentativeHandler) Block(w http.ResponseWriter, r *http.Request) {
	code, ok := parseRepresentativeCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.BlockRepresentativeDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	if err := h.uc.Block(r.Context(), code, dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RepresentativeHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	code, ok := parseRepresentativeCode(w, r, "code")
	if !ok {
		return
	}
	if err := h.uc.Unblock(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RepresentativeHandler) AddEnterprise(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeEnterpriseDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddEnterprise(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddAccounting(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeAccountingDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddAccounting(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddRegion(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeRegionDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddRegion(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddSegment(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeSegmentDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddSegment(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddSalesPlan(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeSalesPlanDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddSalesPlan(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddInterest(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeInterestDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddInterest(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddPhone(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativePhoneDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddPhone(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddEmail(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeEmailDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddEmail(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddCorrespondenceAddress(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeCorrespondenceAddressDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddCorrespondenceAddress(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) AddContact(w http.ResponseWriter, r *http.Request) {
	var dto request.RepresentativeContactDTO
	if decodeRepresentative(w, r, &dto) {
		result, err := h.uc.AddContact(r.Context(), dto)
		respondRepresentative(w, result, err)
	}
}

func (h *RepresentativeHandler) Report(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.Report(r.Context(), parseRepresentativeFilter(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *RepresentativeHandler) FollowUp(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.FollowUp(r.Context(), parseFollowUpFilter(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func parseRepresentativeCode(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return 0, false
	}
	return code, true
}

func parseRepresentativeFilter(r *http.Request) reprepo.RepresentativeFilter {
	q := r.URL.Query()
	filter := reprepo.RepresentativeFilter{
		Codes:        parseIntList(q.Get("codes")),
		ActiveStatus: q.Get("active_status"),
		SortBy:       q.Get("sort_by"),
		WithAccounts: q.Get("with_accounts") == "true",
	}
	if description := q.Get("description"); description != "" {
		filter.Description = &description
	}
	if raw := q.Get("type_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.TypeCode = &v
		}
	}
	if state := q.Get("state"); state != "" {
		state = strings.ToUpper(state)
		filter.State = &state
	}
	if raw := q.Get("region_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.RegionCode = &v
		}
	}
	return filter
}

func parseFollowUpFilter(r *http.Request) reprepo.FollowUpFilter {
	q := r.URL.Query()
	filter := reprepo.FollowUpFilter{
		RepresentativeCodes: parseIntList(q.Get("representative_codes")),
		CustomerCodes:       parseIntList(q.Get("customer_codes")),
	}
	if from := q.Get("from"); from != "" {
		filter.From = datetime.ParseDatePtr(&from)
	}
	if to := q.Get("to"); to != "" {
		filter.To = datetime.ParseDatePtr(&to)
	}
	return filter
}

func parseIntList(raw string) []int64 {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		if v, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func decodeRepresentative(w http.ResponseWriter, r *http.Request, dto any) bool {
	if err := json.NewDecoder(r.Body).Decode(dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func respondRepresentative(w http.ResponseWriter, result any, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}
