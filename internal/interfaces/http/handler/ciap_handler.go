package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// CIAPHandler: parâmetros, bens e apuração do CIAP (VFIS0345).
type CIAPHandler struct {
	UC *fiscal_uc.CIAPUseCase
}

func responderCIAP(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, status, out)
}

func idDoBem(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("bem inválido"))
		return 0, false
	}
	return id, true
}

func anoMes(r *http.Request) (int, int) {
	ano, _ := strconv.Atoi(r.URL.Query().Get("ano"))
	mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))
	if ano == 0 {
		ano = time.Now().Year()
	}
	return ano, mes
}

func (h *CIAPHandler) Parametros(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Parametros(r.Context())
	responderCIAP(w, http.StatusOK, fiscal_uc.CIAPParametrosDTO{CodigoAjuste: out.CodigoAjuste, ContaAtivoID: out.ContaAtivoID, NumeroParcelas: out.NumeroParcelas}, err)
}

func (h *CIAPHandler) SalvarParametros(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.CIAPParametrosDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.SalvarParametros(r.Context(), dto)
	responderCIAP(w, http.StatusOK, fiscal_uc.CIAPParametrosDTO{CodigoAjuste: out.CodigoAjuste, ContaAtivoID: out.ContaAtivoID, NumeroParcelas: out.NumeroParcelas}, err)
}

func (h *CIAPHandler) Bens(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Bens(r.Context())
	responderCIAP(w, http.StatusOK, out, err)
}

func (h *CIAPHandler) CriarBem(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.CIAPBemDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.CriarBem(r.Context(), dto)
	responderCIAP(w, http.StatusCreated, out, err)
}

func (h *CIAPHandler) AtualizarBem(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoBem(w, r)
	if !ok {
		return
	}
	var dto fiscal_uc.CIAPBemDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.AtualizarBem(r.Context(), id, dto)
	responderCIAP(w, http.StatusOK, out, err)
}

func (h *CIAPHandler) BaixarBem(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoBem(w, r)
	if !ok {
		return
	}
	var dto fiscal_uc.CIAPBaixaDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	if err := h.UC.Baixar(r.Context(), id, dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CIAPHandler) ExcluirBem(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoBem(w, r)
	if !ok {
		return
	}
	if err := h.UC.ExcluirBem(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Previa: GET ?ano=&mes=&saidas_trib_exp=&saidas_total= (sem gravar).
func (h *CIAPHandler) Previa(w http.ResponseWriter, r *http.Request) {
	ano, mes := anoMes(r)
	q := r.URL.Query()
	out, err := h.UC.Previa(r.Context(), fiscal_uc.CIAPApurarDTO{Ano: ano, Mes: mes, SaidasTribExp: q.Get("saidas_trib_exp"), SaidasTotal: q.Get("saidas_total")})
	responderCIAP(w, http.StatusOK, out, err)
}

func (h *CIAPHandler) Apurar(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.CIAPApurarDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.Apurar(r.Context(), dto)
	responderCIAP(w, http.StatusOK, out, err)
}

func (h *CIAPHandler) Fechar(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		Ano    int  `json:"ano"`
		Mes    int  `json:"mes"`
		Fechar bool `json:"fechar"`
	}
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	if err := h.UC.Fechar(r.Context(), dto.Ano, dto.Mes, dto.Fechar); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CIAPHandler) Apuracoes(w http.ResponseWriter, r *http.Request) {
	ano, _ := anoMes(r)
	out, err := h.UC.Apuracoes(r.Context(), ano)
	responderCIAP(w, http.StatusOK, out, err)
}
