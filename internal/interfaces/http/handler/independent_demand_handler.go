package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// Demanda independente (VPLA0102): cadastro, programação, acompanhamento e as
// ações de encerrar, atender a finalidade e gerar OF.

// Create devolve as demandas criadas (uma ou as entregas da programação).
func (h *IndependentDemandHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateIndependentDemandDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.uc.Criar(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

// CreateUma é o /create de antes: uma demanda, devolvida como objeto. O
// desktop até a 1.8.0 lê a resposta assim; a programação vai pelo POST /.
func (h *IndependentDemandHandler) CreateUma(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateIndependentDemandDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	if dto.Entregas > 1 {
		security.RespondError(w, http.StatusUnprocessableEntity, "programação de entregas: use POST /api/independent-demand")
		return
	}
	result, err := h.uc.Criar(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result[0])
}

func (h *IndependentDemandHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDaDemanda(w, r)
	if !ok {
		return
	}
	var dto request.UpdateIndependentDemandDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	dto.CodeDemand = code
	result, err := h.uc.Alterar(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// List: ?item=<código de negócio>&motivo=&situacao=&de=AAAA-MM-DD&ate=AAAA-MM-DD
func (h *IndependentDemandHandler) List(w http.ResponseWriter, r *http.Request) {
	f, ok := filtroDaDemanda(w, r)
	if !ok {
		return
	}
	h.buscar(w, r, f, strings.TrimSpace(r.URL.Query().Get("item")))
}

func (h *IndependentDemandHandler) ListByItem(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código de item inválido")
		return
	}
	h.buscar(w, r, entity.Filtro{ItemCode: &itemCode, Situacao: entity.SituacaoAberta}, "")
}

func (h *IndependentDemandHandler) ListFromDate(w http.ResponseWriter, r *http.Request) {
	date, err := time.Parse("2006-01-02", chi.URLParam(r, "date"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "formato de data inválido")
		return
	}
	h.buscar(w, r, entity.Filtro{De: &date, Situacao: entity.SituacaoAberta}, "")
}

func (h *IndependentDemandHandler) buscar(w http.ResponseWriter, r *http.Request, f entity.Filtro, item string) {
	result, err := h.uc.Buscar(r.Context(), f, item)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndependentDemandHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDaDemanda(w, r)
	if !ok {
		return
	}
	result, err := h.uc.Detalhe(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// Delete cancela a demanda (?motivo=). É a exclusão do desktop até a 1.8.0,
// que não manda motivo: sem ele, o cancelamento registra de onde veio.
func (h *IndependentDemandHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDaDemanda(w, r)
	if !ok {
		return
	}
	motivo := strings.TrimSpace(r.URL.Query().Get("motivo"))
	if motivo == "" {
		motivo = "Excluída pela tela de demanda independente (sem motivo informado)"
	}
	if err := h.uc.Cancelar(r.Context(), code, motivo); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IndependentDemandHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	h.encerrar(w, r, h.uc.Cancelar)
}

func (h *IndependentDemandHandler) Encerrar(w http.ResponseWriter, r *http.Request) {
	h.encerrar(w, r, h.uc.Encerrar)
}

func (h *IndependentDemandHandler) encerrar(w http.ResponseWriter, r *http.Request, acao func(ctx context.Context, code int64, motivo string) error) {
	code, ok := codigoDaDemanda(w, r)
	if !ok {
		return
	}
	var dto request.EncerrarDemandaDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	if err := acao(r.Context(), code, dto.Motivo); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	result, err := h.uc.Detalhe(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndependentDemandHandler) Atender(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDaDemanda(w, r)
	if !ok {
		return
	}
	var dto request.AtenderDemandaDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.uc.Atender(r.Context(), code, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *IndependentDemandHandler) GerarOF(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDaDemanda(w, r)
	if !ok {
		return
	}
	var dto request.GerarOFDaDemandaDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.uc.GerarOF(r.Context(), code, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func codigoDaDemanda(w http.ResponseWriter, r *http.Request) (int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código da demanda inválido")
		return 0, false
	}
	return code, true
}

func filtroDaDemanda(w http.ResponseWriter, r *http.Request) (entity.Filtro, bool) {
	q := r.URL.Query()
	f := entity.Filtro{Motivo: strings.TrimSpace(q.Get("motivo")), Situacao: strings.TrimSpace(q.Get("situacao"))}
	for _, c := range []struct {
		nome string
		alvo **time.Time
	}{{"de", &f.De}, {"ate", &f.Ate}} {
		if v := strings.TrimSpace(q.Get(c.nome)); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				security.RespondError(w, http.StatusBadRequest, "data inválida em "+c.nome+": use AAAA-MM-DD")
				return f, false
			}
			*c.alvo = &t
		}
	}
	if f.Motivo != "" && !entity.MotivoValido(f.Motivo) {
		security.RespondError(w, http.StatusBadRequest, "motivo inválido")
		return f, false
	}
	// Sem situação, só as abertas: é o que o /list devolvia ao desktop até a
	// 1.8.0, que não filtra e mostraria a cancelada como ativa. TODAS traz as
	// três.
	switch f.Situacao {
	case "":
		f.Situacao = entity.SituacaoAberta
	case "TODAS":
		f.Situacao = ""
	case entity.SituacaoAberta, entity.SituacaoEncerrada, entity.SituacaoCancelada:
	default:
		security.RespondError(w, http.StatusBadRequest, "situação inválida")
		return f, false
	}
	return f, true
}

// Consumo: ?item=<código de negócio> — a visão do consumo do item (previsão,
// demandas e vendas, com quem consumiu o quê).
func (h *IndependentDemandHandler) Consumo(w http.ResponseWriter, r *http.Request) {
	item := strings.TrimSpace(r.URL.Query().Get("item"))
	if item == "" {
		security.RespondError(w, http.StatusBadRequest, "informe o item")
		return
	}
	out, err := h.uc.ConsumoDoItem(r.Context(), request.TextCode(item))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
