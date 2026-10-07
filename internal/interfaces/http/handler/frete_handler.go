package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// FreteHandler expõe o frete sobre compras (CT-e da transportadora).
type FreteHandler struct {
	uc *fiscal_uc.FreteUseCase
}

func NewFreteHandler(uc *fiscal_uc.FreteUseCase) *FreteHandler { return &FreteHandler{uc: uc} }

const maxXMLCTeBytes = 2 << 20

func (h *FreteHandler) Listar(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Listar(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FreteHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	out, err := h.uc.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FreteHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.FreteDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Criar(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

func (h *FreteHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var dto fiscal_uc.FreteDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Atualizar(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// UploadXML recebe o ARQUIVO do CT-e (campo "file"); opcionais
// "data_vencimento" (AAAA-MM-DD) e "tipo_rateio".
func (h *FreteHandler) UploadXML(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxXMLCTeBytes+(1<<20))
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		security.RespondError(w, http.StatusBadRequest, "envie o arquivo XML do CT-e (multipart, campo file)")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()
	arq, _, err := r.FormFile("file")
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "envie o arquivo XML do CT-e no campo file")
		return
	}
	defer arq.Close()
	conteudo, err := io.ReadAll(io.LimitReader(arq, maxXMLCTeBytes+1))
	if err != nil || len(conteudo) > maxXMLCTeBytes {
		security.RespondError(w, http.StatusBadRequest, "arquivo do CT-e inválido ou maior que 2 MB")
		return
	}
	out, err := h.uc.ImportarXML(r.Context(), conteudo, r.FormValue("data_vencimento"), r.FormValue("tipo_rateio"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

func (h *FreteHandler) Lancar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	out, err := h.uc.Lancar(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *FreteHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	var body struct {
		Motivo string `json:"motivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Cancelar(r.Context(), id, body.Motivo)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
