package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

const (
	// Um XML de NF-e grande tem algumas centenas de KB; 5 MB por arquivo é
	// folga de sobra sem abrir a porta para upload de qualquer coisa.
	maxXMLNotaBytes    = 5 << 20
	maxArquivosPorLote = 50
)

// FiscalEntryHandler é a nota de entrada completa: importação do XML pelo
// arquivo, conciliação dos itens com o cadastro, plano de contas e parcelas.
type FiscalEntryHandler struct {
	upload    *fiscal_uc.UploadNFEEntryUseCase
	byKey     *fiscal_uc.ImportNFeByKeyUseCase
	conciliar *fiscal_uc.SaveFiscalEntryConciliationUseCase
	sugerir   *fiscal_uc.SuggestFiscalEntryItemUseCase
	docs      repository.FiscalEntryDocumentRepository
	auth      ports.AuthService
}

func NewFiscalEntryHandler(
	upload *fiscal_uc.UploadNFEEntryUseCase,
	byKey *fiscal_uc.ImportNFeByKeyUseCase,
	conciliar *fiscal_uc.SaveFiscalEntryConciliationUseCase,
	sugerir *fiscal_uc.SuggestFiscalEntryItemUseCase,
	docs repository.FiscalEntryDocumentRepository,
	auth ports.AuthService,
) *FiscalEntryHandler {
	return &FiscalEntryHandler{upload: upload, byKey: byKey, conciliar: conciliar, sugerir: sugerir, docs: docs, auth: auth}
}

// UploadXMLResult é o resultado por arquivo de um lote de XMLs.
type UploadXMLResult struct {
	Arquivo string                        `json:"arquivo"`
	Entrada *response.FiscalEntryResponse `json:"entrada,omitempty"`
	Erro    string                        `json:"erro,omitempty"`
}

// UploadXML recebe um ou mais arquivos XML (multipart, campo "files" ou
// "file"). Cada arquivo é uma nota independente: um XML inválido no lote não
// impede os demais, e o resultado diz o que aconteceu com cada um.
func (h *FiscalEntryHandler) UploadXML(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxArquivosPorLote)*maxXMLNotaBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		security.RespondError(w, http.StatusBadRequest, "envie os arquivos XML como formulário (multipart/form-data) no campo \"files\"")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	arquivos := append(r.MultipartForm.File["files"], r.MultipartForm.File["file"]...)
	if len(arquivos) == 0 {
		security.RespondError(w, http.StatusBadRequest, "nenhum arquivo XML enviado")
		return
	}
	if len(arquivos) > maxArquivosPorLote {
		security.RespondError(w, http.StatusBadRequest, fmt.Sprintf("envie no máximo %d arquivos por vez", maxArquivosPorLote))
		return
	}

	dto := request.UploadNFEDTO{}
	if v := strings.TrimSpace(r.FormValue("purchase_order_code")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "pedido de compra inválido")
			return
		}
		dto.PurchaseOrderCode = &n
	}
	if v := strings.TrimSpace(r.FormValue("data_entrada")); v != "" {
		dto.DataEntrada = &v
	}

	resultados := make([]UploadXMLResult, 0, len(arquivos))
	sucesso := 0
	for _, fh := range arquivos {
		res := UploadXMLResult{Arquivo: filepath.Base(fh.Filename)}
		if !strings.EqualFold(filepath.Ext(fh.Filename), ".xml") {
			res.Erro = "o arquivo não é .xml — envie o XML da nota (o DANFE em PDF não serve para importar)"
			resultados = append(resultados, res)
			continue
		}
		if fh.Size > maxXMLNotaBytes {
			res.Erro = "arquivo maior que 5 MB: não parece um XML de NF-e"
			resultados = append(resultados, res)
			continue
		}
		f, err := fh.Open()
		if err != nil {
			res.Erro = "não foi possível ler o arquivo"
			resultados = append(resultados, res)
			continue
		}
		conteudo, err := io.ReadAll(io.LimitReader(f, maxXMLNotaBytes+1))
		_ = f.Close()
		if err != nil {
			res.Erro = "não foi possível ler o arquivo"
			resultados = append(resultados, res)
			continue
		}
		entrada, err := h.upload.ExecuteFile(r.Context(), conteudo, dto)
		if err != nil {
			if errors.Is(err, errorsuc.ErrUnauthorized) {
				security.RespondUseCaseError(w, err)
				return
			}
			res.Erro = mensagemDeErro(err)
		} else {
			res.Entrada = entrada
			sucesso++
		}
		resultados = append(resultados, res)
	}

	status := http.StatusCreated
	if sucesso == 0 {
		status = http.StatusUnprocessableEntity
	}
	security.RespondJSON(w, status, map[string]any{
		"importadas": sucesso,
		"com_erro":   len(resultados) - sucesso,
		"resultados": resultados,
	})
}

// mensagemDeErro devolve a mensagem que o usuário pode ler: erros de regra
// (validação, conflito, não encontrado) dizem o que fazer; falha interna não
// expõe detalhe de banco.
func mensagemDeErro(err error) string {
	var v *errorsuc.ValidationError
	var c *errorsuc.ConflictError
	var n *errorsuc.NotFoundError
	switch {
	case errors.As(err, &v), errors.As(err, &c), errors.As(err, &n):
		return err.Error()
	default:
		return "falha interna ao importar a nota; tente novamente ou fale com o suporte"
	}
}

func (h *FiscalEntryHandler) ImportByKey(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.ImportNFeByKeyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	result, err := h.byKey.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FiscalEntryHandler) SaveConciliation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto request.SaveFiscalEntryConciliationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.conciliar.Execute(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalEntryHandler) SuggestItems(w http.ResponseWriter, r *http.Request) {
	id, err1 := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	itemID, err2 := strconv.ParseInt(chi.URLParam(r, "itemId"), 10, 64)
	if err1 != nil || err2 != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	result, err := h.sugerir.Execute(r.Context(), id, itemID, r.URL.Query().Get("q"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// DownloadXML devolve o XML original da nota, como foi importado.
func (h *FiscalEntryHandler) DownloadXML(w http.ResponseWriter, r *http.Request) {
	if !h.auth.CanGetFiscalEntry(r.Context()) {
		security.RespondUseCaseError(w, errorsuc.ErrUnauthorized)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	numero, conteudo, err := h.docs.GetEntryXML(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	if conteudo == "" {
		security.RespondError(w, http.StatusNotFound, "esta nota foi lançada sem XML")
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="nfe-entrada-%d.xml"`, numero))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(conteudo))
}
