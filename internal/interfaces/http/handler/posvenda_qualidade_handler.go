package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/indicadores_pv_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/recuperacao_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/recuperacao"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// PosVendaQualidadeHandler expõe os indicadores do pós-venda, a CAPA aberta
// pelo chamado e a recuperação do custo junto ao fornecedor.
type PosVendaQualidadeHandler struct {
	indicadores *indicadores_pv_uc.UseCase
	rec         *recuperacao_uc.UseCase
}

func NewPosVendaQualidadeHandler(i *indicadores_pv_uc.UseCase, r *recuperacao_uc.UseCase) *PosVendaQualidadeHandler {
	return &PosVendaQualidadeHandler{indicadores: i, rec: r}
}

func (h *PosVendaQualidadeHandler) Indicadores(w http.ResponseWriter, r *http.Request) {
	out, err := h.indicadores.Painel(r.Context(), r.URL.Query().Get("de"), r.URL.Query().Get("ate"))
	responder(w, http.StatusOK, out, err)
}

func (h *PosVendaQualidadeHandler) AbrirCAPA(w http.ResponseWriter, r *http.Request) {
	code, ok := idDaRota(r, "code")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "chamado inválido")
		return
	}
	var dto recuperacao_uc.CAPADTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.rec.AbrirCAPA(r.Context(), code, dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *PosVendaQualidadeHandler) Listar(w http.ResponseWriter, r *http.Request) {
	out, err := h.rec.Listar(r.Context(), dom.Filtro{Situacao: r.URL.Query().Get("situacao"), Fornecedor: int64DaQuery(r, "fornecedor"), OSID: int64DaQuery(r, "os")})
	responder(w, http.StatusOK, out, err)
}

func (h *PosVendaQualidadeHandler) Abrir(w http.ResponseWriter, r *http.Request) {
	var dto recuperacao_uc.AberturaDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.rec.Abrir(r.Context(), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *PosVendaQualidadeHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "reclamação inválida")
		return
	}
	out, err := h.rec.Obter(r.Context(), id)
	responder(w, http.StatusOK, out, err)
}

func (h *PosVendaQualidadeHandler) Mudar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "reclamação inválida")
		return
	}
	var dto recuperacao_uc.MudancaDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.rec.Mudar(r.Context(), id, dto)
	responder(w, http.StatusOK, out, err)
}
