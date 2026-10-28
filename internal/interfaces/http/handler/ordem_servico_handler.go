package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/ordem_servico_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/ordem"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// OrdemServicoHandler expõe a ordem de serviço da assistência técnica.
type OrdemServicoHandler struct{ uc *ordem_servico_uc.UseCase }

func NewOrdemServicoHandler(uc *ordem_servico_uc.UseCase) *OrdemServicoHandler {
	return &OrdemServicoHandler{uc: uc}
}

type motivoDTO struct {
	Motivo string `json:"motivo"`
}

type inicioDTO struct {
	TecnicoCode *int64 `json:"tecnico_code,omitempty"`
}

func (h *OrdemServicoHandler) Parametros(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Parametros(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *OrdemServicoHandler) SalvarParametros(w http.ResponseWriter, r *http.Request) {
	var dto ordem_servico_uc.ParametrosDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.SalvarParametros(r.Context(), dto)
	responder(w, http.StatusOK, out, err)
}

func (h *OrdemServicoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Listar(r.Context(), dom.Filtro{Situacao: r.URL.Query().Get("situacao"), Cliente: int64DaQuery(r, "cliente"),
		Chamado: int64DaQuery(r, "chamado"), Tecnico: int64DaQuery(r, "tecnico")})
	responder(w, http.StatusOK, out, err)
}

func (h *OrdemServicoHandler) Abrir(w http.ResponseWriter, r *http.Request) {
	var dto ordem_servico_uc.AberturaDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Abrir(r.Context(), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *OrdemServicoHandler) os(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "ordem de serviço inválida")
	}
	return id, ok
}

func (h *OrdemServicoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.os(w, r); ok {
		out, err := h.uc.Obter(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

// acao lê o corpo da ação e responde com a OS atualizada.
func acao[T any](h *OrdemServicoHandler, f func(*http.Request, int64, T) (*ordem_servico_uc.OSResponse, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := h.os(w, r)
		if !ok {
			return
		}
		var dto T
		if !decodificar(w, r, &dto) {
			return
		}
		out, err := f(r, id, dto)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *OrdemServicoHandler) Agendar() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto ordem_servico_uc.AgendaDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.Agendar(r.Context(), id, dto)
	})
}

func (h *OrdemServicoHandler) Iniciar() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto inicioDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.Iniciar(r.Context(), id, dto.TecnicoCode)
	})
}

func (h *OrdemServicoHandler) Concluir() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto ordem_servico_uc.ConclusaoDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.Concluir(r.Context(), id, dto)
	})
}

func (h *OrdemServicoHandler) Reabrir() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto motivoDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.Reabrir(r.Context(), id, dto.Motivo)
	})
}

func (h *OrdemServicoHandler) Cancelar() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto motivoDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.Cancelar(r.Context(), id, dto.Motivo)
	})
}

func (h *OrdemServicoHandler) Faturar() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto ordem_servico_uc.FaturamentoDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.Faturar(r.Context(), id, dto)
	})
}

func (h *OrdemServicoHandler) AddHora() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto ordem_servico_uc.HoraDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.AddHora(r.Context(), id, dto)
	})
}

func (h *OrdemServicoHandler) AddPeca() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto ordem_servico_uc.PecaDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.AddPeca(r.Context(), id, dto)
	})
}

func (h *OrdemServicoHandler) AddDespesa() http.HandlerFunc {
	return acao(h, func(r *http.Request, id int64, dto ordem_servico_uc.DespesaDTO) (*ordem_servico_uc.OSResponse, error) {
		return h.uc.AddDespesa(r.Context(), id, dto)
	})
}

// Remover: DELETE /os/{id}/{tipo}/{lancamento}, tipo horas|pecas|despesas.
func (h *OrdemServicoHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := h.os(w, r)
	if !ok {
		return
	}
	lanc, ok := idDaRota(r, "lancamento")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "lançamento inválido")
		return
	}
	out, err := h.uc.Remover(r.Context(), id, chi.URLParam(r, "tipo"), lanc)
	responder(w, http.StatusOK, out, err)
}
