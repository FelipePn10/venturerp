package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/pdm_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/pdm"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// PDMPadraoHandler expõe o padrão descritivo: vínculo grupo×modificador, modelo
// de atributos e a prévia da descrição.
type PDMPadraoHandler struct{ uc *pdm_uc.UseCase }

func NewPDMPadraoHandler(uc *pdm_uc.UseCase) *PDMPadraoHandler { return &PDMPadraoHandler{uc: uc} }

func codigoPDM(w http.ResponseWriter, valor, rotulo string) (int32, bool) {
	v, err := strconv.ParseInt(valor, 10, 32)
	if err != nil || v <= 0 {
		security.RespondUseCaseError(w, errorsuc.NewValidationError(rotulo+" inválido"))
		return 0, false
	}
	return int32(v), true
}

func (h *PDMPadraoHandler) ModificadoresDoGrupo(w http.ResponseWriter, r *http.Request) {
	grupo, ok := codigoPDM(w, chi.URLParam(r, "code"), "grupo")
	if !ok {
		return
	}
	out, err := h.uc.ModificadoresDoGrupo(r.Context(), grupo)
	responder(w, http.StatusOK, out, err)
}

func (h *PDMPadraoHandler) SalvarModificadoresDoGrupo(w http.ResponseWriter, r *http.Request) {
	grupo, ok := codigoPDM(w, chi.URLParam(r, "code"), "grupo")
	if !ok {
		return
	}
	var dto struct {
		Modifiers []int32 `json:"modifiers"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarModificadoresDoGrupo(r.Context(), grupo, dto.Modifiers)
	responder(w, http.StatusOK, out, err)
}

func (h *PDMPadraoHandler) SalvarAbreviacao(w http.ResponseWriter, r *http.Request) {
	id, ok := codigoPDM(w, chi.URLParam(r, "id"), "modificador")
	if !ok {
		return
	}
	var dto struct {
		Abbreviation *string `json:"abbreviation"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	responder(w, http.StatusOK, map[string]bool{"ok": true}, h.uc.SalvarAbreviacao(r.Context(), id, dto.Abbreviation))
}

func (h *PDMPadraoHandler) Uso(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Uso(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *PDMPadraoHandler) Modelo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	grupo, ok := codigoPDM(w, q.Get("group_code"), "grupo")
	if !ok {
		return
	}
	modificador, ok := codigoPDM(w, q.Get("modifier_id"), "modificador")
	if !ok {
		return
	}
	out, err := h.uc.Modelo(r.Context(), grupo, modificador)
	responder(w, http.StatusOK, out, err)
}

func (h *PDMPadraoHandler) SalvarAtributo(w http.ResponseWriter, r *http.Request) {
	var dto dom.Atributo
	if !decodificar(w, r, &dto) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		v, err := strconv.ParseInt(id, 10, 64)
		if err != nil || v <= 0 {
			security.RespondUseCaseError(w, errorsuc.NewValidationError("atributo inválido"))
			return
		}
		dto.ID = v
	}
	out, err := h.uc.SalvarAtributo(r.Context(), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *PDMPadraoHandler) InativarAtributo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("atributo inválido"))
		return
	}
	responder(w, http.StatusOK, map[string]bool{"ok": true}, h.uc.InativarAtributo(r.Context(), id))
}

func (h *PDMPadraoHandler) Previa(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		GroupCode  int32                   `json:"group_code"`
		ModifierID int32                   `json:"modifier_id"`
		Attributes []valueobject.Attribute `json:"attributes"`
		Exclude    string                  `json:"exclude"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Previa(r.Context(), dto.GroupCode, dto.ModifierID, dto.Attributes, dto.Exclude)
	responder(w, http.StatusOK, out, err)
}
