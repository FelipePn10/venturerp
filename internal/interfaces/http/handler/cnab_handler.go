package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/cnab"
)

// CNABHandler exposes bank exchange file generation (CNAB 240 remessa).
type CNABHandler struct{}

func NewCNABHandler() *CNABHandler { return &CNABHandler{} }

type cnabTituloRequest struct {
	NossoNumero  string  `json:"nosso_numero"`
	NumeroDoc    string  `json:"numero_documento"`
	Vencimento   string  `json:"vencimento"` // YYYY-MM-DD
	Valor        float64 `json:"valor"`
	Emissao      string  `json:"emissao"` // YYYY-MM-DD
	SacadoNome   string  `json:"sacado_nome"`
	SacadoTipo   int     `json:"sacado_tipo"` // 1=CPF 2=CNPJ
	SacadoDoc    string  `json:"sacado_documento"`
	SacadoEnd    string  `json:"sacado_endereco"`
	SacadoBairro string  `json:"sacado_bairro"`
	SacadoCidade string  `json:"sacado_cidade"`
	SacadoUF     string  `json:"sacado_uf"`
	SacadoCEP    string  `json:"sacado_cep"`
}

type cnabRemessaRequest struct {
	Config  cnab.RemessaConfig  `json:"config"`
	Titulos []cnabTituloRequest `json:"titulos"`
}

// GenerateRemessa240 returns a CNAB 240 remessa file (text/plain) for the given
// configuration and titles.
func (h *CNABHandler) GenerateRemessa240(w http.ResponseWriter, r *http.Request) {
	var req cnabRemessaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}

	titulos := make([]cnab.Titulo, 0, len(req.Titulos))
	for _, t := range req.Titulos {
		venc, _ := time.Parse("2006-01-02", t.Vencimento)
		emis, _ := time.Parse("2006-01-02", t.Emissao)
		titulos = append(titulos, cnab.Titulo{
			NossoNumero:  t.NossoNumero,
			NumeroDoc:    t.NumeroDoc,
			Vencimento:   venc,
			Valor:        t.Valor,
			Emissao:      emis,
			SacadoNome:   t.SacadoNome,
			SacadoTipo:   t.SacadoTipo,
			SacadoDoc:    t.SacadoDoc,
			SacadoEnd:    t.SacadoEnd,
			SacadoBairro: t.SacadoBairro,
			SacadoCidade: t.SacadoCidade,
			SacadoUF:     t.SacadoUF,
			SacadoCEP:    t.SacadoCEP,
		})
	}

	content, err := cnab.GenerateRemessa240(req.Config, titulos)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"remessa.rem\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}
