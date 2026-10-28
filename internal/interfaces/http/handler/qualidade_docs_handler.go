package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/certificado_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/documento_uc"
	certdom "github.com/FelipePn10/panossoerp/internal/domain/quality/certificado"
	docdom "github.com/FelipePn10/panossoerp/internal/domain/quality/documento"
	certpdf "github.com/FelipePn10/panossoerp/internal/infrastructure/export/certificado"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// QualidadeDocsHandler expõe certificados de qualidade e documentos controlados.
type QualidadeDocsHandler struct {
	cert *certificado_uc.UseCase
	doc  *documento_uc.UseCase
}

func NewQualidadeDocsHandler(cert *certificado_uc.UseCase, doc *documento_uc.UseCase) *QualidadeDocsHandler {
	return &QualidadeDocsHandler{cert: cert, doc: doc}
}

// ─── certificados ─────────────────────────────────────────────────────────────

func (h *QualidadeDocsHandler) ListarCertificados(w http.ResponseWriter, r *http.Request) {
	item, ok1 := inteiroDaConsulta(r, "item")
	cliente, ok2 := inteiroDaConsulta(r, "cliente")
	if !ok1 || !ok2 {
		security.RespondError(w, http.StatusBadRequest, "filtro inválido")
		return
	}
	out, err := h.cert.Listar(r.Context(), certdom.Filtro{ItemCode: item, CustomerCode: cliente, Status: r.URL.Query().Get("status")})
	responder(w, http.StatusOK, out, err)
}

func (h *QualidadeDocsHandler) EmitirCertificado(w http.ResponseWriter, r *http.Request) {
	var dto certificado_uc.EmissaoDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.cert.Emitir(r.Context(), dto)
	responder(w, http.StatusCreated, out, err)
}

func (h *QualidadeDocsHandler) ObterCertificado(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "certificado inválido")
		return
	}
	out, err := h.cert.Obter(r.Context(), id)
	responder(w, http.StatusOK, out, err)
}

func (h *QualidadeDocsHandler) PDFCertificado(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "certificado inválido")
		return
	}
	c, err := h.cert.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	pdf := certpdf.Gerar(c, time.Now())
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="certificado-%d.pdf"`, c.Numero))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	_, _ = w.Write(pdf)
}

func (h *QualidadeDocsHandler) CancelarCertificado(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	var dto struct {
		Motivo string `json:"motivo"`
	}
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "certificado inválido")
		return
	}
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.cert.Cancelar(r.Context(), id, dto.Motivo)
	responder(w, http.StatusOK, out, err)
}

// ─── documentos ───────────────────────────────────────────────────────────────

func (h *QualidadeDocsHandler) ListarDocumentos(w http.ResponseWriter, r *http.Request) {
	item, ok := inteiroDaConsulta(r, "item")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "filtro inválido")
		return
	}
	q := r.URL.Query()
	out, err := h.doc.Listar(r.Context(), docdom.Filtro{Tipo: q.Get("tipo"), ItemCode: item, Busca: q.Get("busca"), Todos: q.Get("todos") == "true"})
	responder(w, http.StatusOK, out, err)
}

func (h *QualidadeDocsHandler) salvarDocumento(w http.ResponseWriter, r *http.Request, id int64, status int) {
	var dto documento_uc.DocumentoDTO
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.doc.Salvar(r.Context(), id, dto)
	responder(w, status, out, err)
}

func (h *QualidadeDocsHandler) CriarDocumento(w http.ResponseWriter, r *http.Request) {
	h.salvarDocumento(w, r, 0, http.StatusCreated)
}

func (h *QualidadeDocsHandler) documento(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "documento inválido")
	}
	return id, ok
}

func (h *QualidadeDocsHandler) AtualizarDocumento(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.documento(w, r); ok {
		h.salvarDocumento(w, r, id, http.StatusOK)
	}
}

func (h *QualidadeDocsHandler) ObterDocumento(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.documento(w, r); ok {
		out, err := h.doc.Obter(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *QualidadeDocsHandler) GravarRascunho(w http.ResponseWriter, r *http.Request) {
	id, ok := h.documento(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	var dto struct {
		DescricaoMudanca string  `json:"descricao_mudanca"`
		Conteudo         *string `json:"conteudo,omitempty"`
		ArquivoNome      *string `json:"arquivo_nome,omitempty"`
		ArquivoTipo      *string `json:"arquivo_tipo,omitempty"`
		ArquivoBase64    string  `json:"arquivo_base64,omitempty"`
	}
	if !decodificar(w, r, &dto) {
		return
	}
	in := documento_uc.RascunhoDTO{DescricaoMudanca: dto.DescricaoMudanca, Conteudo: dto.Conteudo, ArquivoNome: dto.ArquivoNome, ArquivoTipo: dto.ArquivoTipo}
	if s := strings.TrimSpace(dto.ArquivoBase64); s != "" {
		if i := strings.Index(s, ","); strings.HasPrefix(s, "data:") && i > 0 {
			s = s[i+1:]
		}
		bin, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "arquivo inválido (base64)")
			return
		}
		in.Arquivo = bin
	}
	out, err := h.doc.GravarRascunho(r.Context(), id, in)
	responder(w, http.StatusOK, out, err)
}

func (h *QualidadeDocsHandler) AprovarDocumento(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.documento(w, r); ok {
		out, err := h.doc.Aprovar(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *QualidadeDocsHandler) DescartarRascunho(w http.ResponseWriter, r *http.Request) {
	if id, ok := h.documento(w, r); ok {
		out, err := h.doc.DescartarRascunho(r.Context(), id)
		responder(w, http.StatusOK, out, err)
	}
}

func (h *QualidadeDocsHandler) ArquivoDaRevisao(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "rev")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "revisão inválida")
		return
	}
	rev, err := h.doc.Arquivo(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	tipo, nome := "application/octet-stream", fmt.Sprintf("revisao-%d", rev.Revisao)
	if rev.ArquivoTipo != nil && *rev.ArquivoTipo != "" {
		tipo = *rev.ArquivoTipo
	}
	if rev.ArquivoNome != nil && *rev.ArquivoNome != "" {
		nome = strings.ReplaceAll(*rev.ArquivoNome, `"`, "")
	}
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, nome))
	_, _ = w.Write(rev.Arquivo)
}
