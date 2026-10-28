package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// LoteHandler expõe o lote de inspeção: abertura manual, consulta,
// resultados, decisão de uso, cancelamento e laudos do fornecedor.
type LoteHandler struct{ uc *lote_uc.UseCase }

func NewLoteHandler(uc *lote_uc.UseCase) *LoteHandler { return &LoteHandler{uc: uc} }

func (h *LoteHandler) Listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	item, ok1 := inteiroDaConsulta(r, "item")
	fornecedor, ok2 := inteiroDaConsulta(r, "fornecedor")
	ordem, ok3 := inteiroDaConsulta(r, "ordem")
	if !ok1 || !ok2 || !ok3 {
		security.RespondError(w, http.StatusBadRequest, "filtro inválido")
		return
	}
	out, err := h.uc.Listar(r.Context(), dom.Filtro{Status: q.Get("status"), Origem: q.Get("origem"), ItemCode: item, SupplierCode: fornecedor, ProductionOrderID: ordem})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *LoteHandler) Abrir(w http.ResponseWriter, r *http.Request) {
	var dto lote_uc.AberturaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Abrir(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

func (h *LoteHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "lote inválido")
		return
	}
	out, err := h.uc.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *LoteHandler) GravarResultados(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "lote inválido")
		return
	}
	var dto struct {
		Resultados []lote_uc.ResultadoDTO `json:"resultados"`
	}
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.GravarResultados(r.Context(), id, dto.Resultados)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *LoteHandler) Decidir(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "lote inválido")
		return
	}
	var dto lote_uc.DecisaoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Decidir(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *LoteHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "lote inválido")
		return
	}
	var dto struct {
		Motivo string `json:"motivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Cancelar(r.Context(), id, dto.Motivo)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *LoteHandler) VincularLaudo(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "lote inválido")
		return
	}
	var dto struct {
		QualityReportID int64 `json:"quality_report_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil || dto.QualityReportID <= 0 {
		security.RespondError(w, http.StatusBadRequest, "informe o laudo (quality_report_id)")
		return
	}
	if err := h.uc.VincularRelatorio(r.Context(), id, dto.QualityReportID); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out, err := h.uc.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *LoteHandler) DesvincularLaudo(w http.ResponseWriter, r *http.Request) {
	id, ok1 := idDaRota(r, "id")
	laudo, ok2 := idDaRota(r, "laudo")
	if !ok1 || !ok2 {
		security.RespondError(w, http.StatusBadRequest, "lote ou laudo inválido")
		return
	}
	if err := h.uc.DesvincularRelatorio(r.Context(), id, laudo); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
