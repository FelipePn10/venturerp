package handler

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	appsec "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/portal_uc"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// PortalHandler: os links do portal (rotas do ERP) e o próprio portal (rotas
// públicas, autenticadas pelo token do link).
type PortalHandler struct{ UC *portal_uc.UseCase }

// CriarLink: POST /api/customers/{code}/portal-links {days, description}.
func (h *PortalHandler) CriarLink(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "cliente inválido")
		return
	}
	var dto struct {
		Dias      int    `json:"days"`
		Descricao string `json:"description"`
	}
	if !security.DecodeBodyOptional(w, r, &dto) {
		return
	}
	out, err := h.UC.CriarLink(r.Context(), code, dto.Dias, dto.Descricao)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, out)
}

// ListarLinks: GET /api/customers/{code}/portal-links.
func (h *PortalHandler) ListarLinks(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "cliente inválido")
		return
	}
	out, err := h.UC.ListarLinks(r.Context(), code)
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// RevogarLink: DELETE /api/customers/portal-links/{id}.
func (h *PortalHandler) RevogarLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "link inválido")
		return
	}
	if err := h.UC.RevogarLink(r.Context(), id); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sessao abre o token e devolve o contexto da empresa do link, assinado pelo
// ADMIN dela (quem grava o rascunho), como no aviso do banco.
func (h *PortalHandler) sessao(w http.ResponseWriter, r *http.Request) (context.Context, *portal_uc.Sessao, bool) {
	s, err := h.UC.Abrir(r.Context(), chi.URLParam(r, "token"), r.RemoteAddr, r.UserAgent())
	if err != nil {
		if errors.Is(err, portal_uc.ErrNaoEncontrado) {
			jsonError(w, http.StatusNotFound, "link inválido ou vencido: peça um novo à empresa")
		} else {
			respondErr(w, err)
		}
		return nil, nil, false
	}
	ctx := context.WithValue(r.Context(), contextkey.UserKey, &appsec.AuthUser{ID: s.Ator.String(), Role: "ADMIN", EnterpriseID: s.EnterpriseID, EnterpriseCode: s.EnterpriseCode})
	return ctx, s, true
}

// Resumo: GET /api/portal/{token}.
func (h *PortalHandler) Resumo(w http.ResponseWriter, r *http.Request) {
	ctx, s, ok := h.sessao(w, r)
	if !ok {
		return
	}
	out, err := h.UC.Resumo(ctx, s, r.RemoteAddr, r.UserAgent())
	if err != nil {
		respondErr(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, http.StatusOK, out)
}

// PedirDeNovo: POST /api/portal/{token}/pedidos/{code}/repetir {note}.
func (h *PortalHandler) PedirDeNovo(w http.ResponseWriter, r *http.Request) {
	ctx, s, ok := h.sessao(w, r)
	if !ok {
		return
	}
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "pedido inválido")
		return
	}
	var dto struct {
		Observacao string `json:"note"`
	}
	if !security.DecodeBodyOptional(w, r, &dto) {
		return
	}
	res, err := h.UC.PedirDeNovo(ctx, s, code, dto.Observacao, r.RemoteAddr, r.UserAgent())
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]any{"order_number": res.Pedido.OrderNumber, "lines": res.Linhas, "skipped": res.Falhas})
}

var baixadorDoPortal = &http.Client{Timeout: 30 * time.Second}

// Documento: GET /api/portal/{token}/notas/{id}/{tipo} (tipo = danfe | xml).
// O arquivo oficial vem da emissora e passa pela API: o cliente não recebe o
// endereço de lá e a nota só sai se for de um pedido dele.
func (h *PortalHandler) Documento(w http.ResponseWriter, r *http.Request) {
	ctx, s, ok := h.sessao(w, r)
	if !ok {
		return
	}
	nota, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	tipo := chi.URLParam(r, "tipo")
	if err != nil || (tipo != "danfe" && tipo != "xml") {
		jsonError(w, http.StatusBadRequest, "documento inválido")
		return
	}
	if tipo == "xml" {
		xml, nome, err := h.UC.XMLDaNota(ctx, s, nota, r.RemoteAddr, r.UserAgent())
		if errors.Is(err, portal_uc.ErrNaoEncontrado) {
			jsonError(w, http.StatusNotFound, "nota não encontrada")
			return
		}
		if err != nil {
			respondErr(w, err)
			return
		}
		if xml != "" {
			w.Header().Set("Content-Type", "application/xml")
			w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(nome, `"`, "")+`"`)
			w.Header().Set("Cache-Control", "no-store")
			_, _ = io.WriteString(w, xml)
			return
		}
	}
	url, nome, err := h.UC.Documento(ctx, s, nota, tipo, r.RemoteAddr, r.UserAgent())
	if err != nil {
		if errors.Is(err, portal_uc.ErrNaoEncontrado) {
			jsonError(w, http.StatusNotFound, "nota não encontrada")
			return
		}
		respondErr(w, err)
		return
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		jsonError(w, http.StatusUnprocessableEntity, "documento da nota indisponível")
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		respondErr(w, err)
		return
	}
	resp, err := baixadorDoPortal.Do(req)
	if err != nil {
		jsonError(w, http.StatusBadGateway, "a emissora da nota não respondeu: tente de novo em instantes")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		jsonError(w, http.StatusBadGateway, "a emissora da nota não entregou o documento: tente de novo em instantes")
		return
	}
	// O arquivo é o oficial da emissora (DANFE da Focus e XML autorizado com o
	// protocolo da SEFAZ). Se vier outra coisa (página de erro com 200), o
	// cliente não pode receber um "PDF" quebrado.
	corpo := bufio.NewReaderSize(io.LimitReader(resp.Body, 20<<20), 512)
	inicio, _ := corpo.Peek(512)
	if !documentoValido(tipo, inicio) {
		jsonError(w, http.StatusBadGateway, "a emissora da nota não entregou o documento: tente de novo em instantes")
		return
	}
	ct := "application/pdf"
	if tipo == "xml" {
		ct = "application/xml"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(nome, `"`, "")+`"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, corpo)
}

// documentoValido confere a assinatura do arquivo: PDF começa com "%PDF" e o XML
// da NF-e traz <nfeProc> ou <NFe>.
func documentoValido(tipo string, inicio []byte) bool {
	if tipo == "danfe" {
		return bytes.HasPrefix(inicio, []byte("%PDF"))
	}
	t := bytes.TrimLeft(inicio, "\ufeff \t\r\n")
	return bytes.HasPrefix(t, []byte("<")) && (bytes.Contains(inicio, []byte("<nfeProc")) || bytes.Contains(inicio, []byte("<NFe")))
}
