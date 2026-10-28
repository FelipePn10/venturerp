package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/amostragem_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// AmostragemHandler expõe a amostragem NBR 5426: calculadora, procedimentos,
// plano do lote, estados de severidade e liberação de suspensão.
type AmostragemHandler struct{ uc *amostragem_uc.UseCase }

func NewAmostragemHandler(uc *amostragem_uc.UseCase) *AmostragemHandler {
	return &AmostragemHandler{uc: uc}
}

func inteiroDaConsulta(r *http.Request, nome string) (int64, bool) {
	s := strings.TrimSpace(r.URL.Query().Get(nome))
	if s == "" {
		return 0, true
	}
	v, err := strconv.ParseInt(s, 10, 64)
	return v, err == nil && v >= 0
}

// Tabelas devolve os valores válidos para a tela montar as listas.
func (h *AmostragemHandler) Tabelas(w http.ResponseWriter, _ *http.Request) {
	security.RespondJSON(w, http.StatusOK, map[string]any{
		"nqas":        dom.NQAs,
		"niveis":      []dom.Nivel{dom.NivelS1, dom.NivelS2, dom.NivelS3, dom.NivelS4, dom.NivelI, dom.NivelII, dom.NivelIII},
		"severidades": []dom.Severidade{dom.Normal, dom.Apertada, dom.Atenuada},
	})
}

func (h *AmostragemHandler) Calcular(w http.ResponseWriter, r *http.Request) {
	lote, ok := inteiroDaConsulta(r, "lote")
	if !ok || lote == 0 {
		security.RespondError(w, http.StatusBadRequest, "informe o tamanho do lote em unidades (lote=)")
		return
	}
	q := r.URL.Query()
	out, err := h.uc.Calcular(lote, q.Get("nivel"), q.Get("nqa"), q.Get("severidade"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AmostragemHandler) ListarProcedimentos(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarProcedimentos(r.Context(), r.URL.Query().Get("ativos") == "true")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AmostragemHandler) ObterProcedimento(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "procedimento inválido")
		return
	}
	out, err := h.uc.ObterProcedimento(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AmostragemHandler) salvar(w http.ResponseWriter, r *http.Request, id int64, status int) {
	var dto amostragem_uc.ProcedimentoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Salvar(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, status, out)
}

func (h *AmostragemHandler) CriarProcedimento(w http.ResponseWriter, r *http.Request) {
	h.salvar(w, r, 0, http.StatusCreated)
}

func (h *AmostragemHandler) AtualizarProcedimento(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "procedimento inválido")
		return
	}
	h.salvar(w, r, id, http.StatusOK)
}

// PlanoDoLote: ?procedimento=&item=&chave=&quantidade= — o plano pela
// severidade corrente, sem registrar nada.
func (h *AmostragemHandler) PlanoDoLote(w http.ResponseWriter, r *http.Request) {
	proc, ok1 := inteiroDaConsulta(r, "procedimento")
	item, ok2 := inteiroDaConsulta(r, "item")
	if !ok1 || !ok2 || proc == 0 || item == 0 {
		security.RespondError(w, http.StatusBadRequest, "informe procedimento e item")
		return
	}
	q := r.URL.Query()
	out, err := h.uc.PlanoDoLote(r.Context(), proc, item, q.Get("chave"), q.Get("quantidade"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AmostragemHandler) Estados(w http.ResponseWriter, r *http.Request) {
	item, ok := inteiroDaConsulta(r, "item")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "item inválido")
		return
	}
	out, err := h.uc.Estados(r.Context(), item)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AmostragemHandler) Eventos(w http.ResponseWriter, r *http.Request) {
	item, ok := inteiroDaConsulta(r, "item")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "item inválido")
		return
	}
	out, err := h.uc.Eventos(r.Context(), item)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AmostragemHandler) LiberarSuspensao(w http.ResponseWriter, r *http.Request) {
	var dto amostragem_uc.LiberacaoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.LiberarSuspensao(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
