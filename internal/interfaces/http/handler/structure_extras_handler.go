package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// StructureExtrasHandler atende as duas conferências que a tela de estrutura
// oferece antes e depois de gravar: simular a fórmula e ver o histórico.
type StructureExtrasHandler struct {
	simulate *structure_uc.SimulateQuantityFormulaUseCase
	history  *structure_uc.ListStructureHistoryUseCase
}

func NewStructureExtrasHandler(
	simulate *structure_uc.SimulateQuantityFormulaUseCase,
	history *structure_uc.ListStructureHistoryUseCase,
) *StructureExtrasHandler {
	return &StructureExtrasHandler{simulate: simulate, history: history}
}

// SimulateFormula calcula a quantidade que a fórmula produz com as respostas
// informadas, sem gravar nada.
func (h *StructureExtrasHandler) SimulateFormula(w http.ResponseWriter, r *http.Request) {
	var in structure_uc.SimulateFormulaInput
	if !security.DecodeBody(w, r, &in) {
		return
	}
	out, err := h.simulate.Execute(r.Context(), in)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// History devolve quem alterou o quê na estrutura do item pai.
func (h *StructureExtrasHandler) History(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := h.history.Execute(r.Context(), request.TextCode(chi.URLParam(r, "itemCode")), limit)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}
