package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// ImpressaoHandler entrega os documentos impressos (PDF) de cada rotina.
type ImpressaoHandler struct {
	UC *impressao_uc.UseCase
}

func (h *ImpressaoHandler) responder(w http.ResponseWriter, a *impressao_uc.Arquivo, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, a.Nome))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.Bytes)
}

func (h *ImpressaoHandler) porCodigo(fn func(ctx context.Context, code int64) (*impressao_uc.Arquivo, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
		if err != nil || code <= 0 {
			security.RespondError(w, http.StatusBadRequest, "código inválido")
			return
		}
		a, err := fn(r.Context(), code)
		h.responder(w, a, err)
	}
}

func (h *ImpressaoHandler) PedidoVenda() http.HandlerFunc    { return h.porCodigo(h.UC.PedidoVenda) }
func (h *ImpressaoHandler) NotaEntrada() http.HandlerFunc    { return h.porCodigo(h.UC.NotaEntrada) }
func (h *ImpressaoHandler) PlanoMateriais() http.HandlerFunc { return h.porCodigo(h.UC.PlanoMateriais) }
func (h *ImpressaoHandler) Capacidade() http.HandlerFunc     { return h.porCodigo(h.UC.Capacidade) }
func (h *ImpressaoHandler) Romaneio() http.HandlerFunc       { return h.porCodigo(h.UC.Romaneio) }
func (h *ImpressaoHandler) Proposta() http.HandlerFunc       { return h.porCodigo(h.UC.Proposta) }
func (h *ImpressaoHandler) FichaCliente() http.HandlerFunc   { return h.porCodigo(h.UC.FichaCliente) }

// Cotacao: ?fornecedor=N imprime a solicitação para o fornecedor; sem ele, o
// mapa comparativo.
func (h *ImpressaoHandler) Cotacao() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forn, err := inteiroParam(r, "fornecedor")
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		var f int64
		if forn != nil {
			f = *forn
		}
		h.porCodigo(func(ctx context.Context, code int64) (*impressao_uc.Arquivo, error) {
			return h.UC.Cotacao(ctx, code, f)
		})(w, r)
	}
}

// ─── parâmetros de consulta ─────────────────────────────────────────────────

type erroParametro string

func (e erroParametro) Error() string { return string(e) }

func dataParam(r *http.Request, nome string) (*time.Time, error) {
	v := strings.TrimSpace(r.URL.Query().Get(nome))
	if v == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006"} {
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, erroParametro(fmt.Sprintf("data inválida em %q: use AAAA-MM-DD", nome))
}

func inteiroParam(r *http.Request, nome string) (*int64, error) {
	v := strings.TrimSpace(r.URL.Query().Get(nome))
	if v == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, erroParametro(fmt.Sprintf("número inválido em %q", nome))
	}
	return &n, nil
}

func (h *ImpressaoHandler) titulos(receber bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parte := "fornecedor"
		if receber {
			parte = "cliente"
		}
		var f impressao_uc.FiltroTitulos
		var err error
		if f.ParteID, err = inteiroParam(r, parte); err == nil {
			if f.NotaFiscalID, err = inteiroParam(r, "nota"); err == nil {
				if f.De, err = dataParam(r, "de"); err == nil {
					f.Ate, err = dataParam(r, "ate")
				}
			}
		}
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		f.SomenteAbertos = r.URL.Query().Get("abertos") == "true"
		a, err := h.UC.Titulos(r.Context(), receber, f)
		h.responder(w, a, err)
	}
}

func (h *ImpressaoHandler) ContasReceber() http.HandlerFunc { return h.titulos(true) }
func (h *ImpressaoHandler) ContasPagar() http.HandlerFunc   { return h.titulos(false) }

// periodo lê de/ate; sem eles, o mês corrente até hoje + dias.
func periodoImpressao(r *http.Request, diasDepois int) (time.Time, time.Time, error) {
	de, err := dataParam(r, "de")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	ate, err := dataParam(r, "ate")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	hoje := time.Now()
	ini := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, time.Local)
	fim := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, diasDepois)
	if de != nil {
		ini = *de
	}
	if ate != nil {
		fim = *ate
	}
	return ini, fim, nil
}

func (h *ImpressaoHandler) Programacao() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		de, ate, err := periodoImpressao(r, 60)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		grupo, err := inteiroParam(r, "grupo")
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		a, err := h.UC.Programacao(r.Context(), de, ate, grupo)
		h.responder(w, a, err)
	}
}

func (h *ImpressaoHandler) Kardex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		de, ate, err := periodoImpressao(r, 0)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		dep, err := inteiroParam(r, "deposito")
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		a, err := h.UC.Kardex(r.Context(), impressao_uc.FiltroKardex{Item: r.URL.Query().Get("item"), Deposito: dep, De: de, Ate: ate})
		h.responder(w, a, err)
	}
}

func (h *ImpressaoHandler) CustoProducao() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ordens []int64
		for _, s := range strings.Split(r.URL.Query().Get("ordens"), ",") {
			if s = strings.TrimSpace(s); s == "" {
				continue
			}
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				security.RespondError(w, http.StatusBadRequest, "lista de ordens inválida")
				return
			}
			ordens = append(ordens, n)
		}
		de, err := dataParam(r, "de")
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		ate, err := dataParam(r, "ate")
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if ordens == nil {
			ordens = []int64{}
		}
		a, err := h.UC.CustoProducao(r.Context(), ordens, de, ate)
		h.responder(w, a, err)
	}
}
