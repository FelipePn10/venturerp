package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// PreviaNFeHandler expõe a conferência da nota antes da emissão.
type PreviaNFeHandler struct {
	previaUC *fiscal_uc.PreviaNFeUseCase
}

func NewPreviaNFeHandler(previaUC *fiscal_uc.PreviaNFeUseCase) *PreviaNFeHandler {
	return &PreviaNFeHandler{previaUC: previaUC}
}

// Previa monta a NF-e como ela será transmitida e lista as pendências.
// GET /api/fiscal/exits/{id}/previa
func (h *PreviaNFeHandler) Previa(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		security.RespondError(w, http.StatusBadRequest, "nota de saída inválida")
		return
	}
	result, err := h.previaUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// PreviaPDF entrega a prévia em PDF (layout do DANFE, sem valor fiscal).
// GET /api/fiscal/exits/{id}/previa/pdf
func (h *PreviaNFeHandler) PreviaPDF(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		security.RespondError(w, http.StatusBadRequest, "nota de saída inválida")
		return
	}
	nome, pdf, err := h.previaUC.PDF(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, nome))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

// PreviaXML devolve o XML da prévia (layout da NF-e, marcado como prévia sem
// validade fiscal) com o nome do arquivo, para a tela mostrar, copiar e baixar.
// GET /api/fiscal/exits/{id}/previa/xml
func (h *PreviaNFeHandler) PreviaXML(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		security.RespondError(w, http.StatusBadRequest, "nota de saída inválida")
		return
	}
	nome, xml, err := h.previaUC.XML(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"arquivo": nome, "xml": string(xml)})
}

// DestinatarioNotaHandler entrega à nota avulsa os dados do cliente já cadastrados.
type DestinatarioNotaHandler struct {
	uc *fiscal_uc.DestinatarioDaNotaUseCase
}

func NewDestinatarioNotaHandler(uc *fiscal_uc.DestinatarioDaNotaUseCase) *DestinatarioNotaHandler {
	return &DestinatarioNotaHandler{uc: uc}
}

func (h *DestinatarioNotaHandler) DoCliente(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do cliente inválido")
		return
	}
	result, err := h.uc.DoCliente(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *DestinatarioNotaHandler) SugerirCfop(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	result, err := h.uc.SugerirCfop(r.Context(), q.Get("uf"), q.Get("ie"), q.Get("tipo_pessoa"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
