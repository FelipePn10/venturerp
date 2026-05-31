package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
)

type SPEDHandler struct {
	uc *fiscal_uc.SPEDUseCase
}

func NewSPEDHandler(uc *fiscal_uc.SPEDUseCase) *SPEDHandler {
	return &SPEDHandler{uc: uc}
}

type generateEFDRequest struct {
	CNPJ              string    `json:"cnpj"`
	Nome              string    `json:"nome"`
	UF                string    `json:"uf"`
	IE                string    `json:"ie"`
	IM                string    `json:"im"`
	SUFRAMA           string    `json:"suframa"`
	CodigoMunicipio   string    `json:"codigo_municipio"`
	RegimeTributario  string    `json:"regime_tributario"`
	DataInicial       time.Time `json:"data_inicial"`
	DataFinal         time.Time `json:"data_final"`
	IndicadorSituacao string    `json:"indicador_situacao"`
	// Contabilista
	ContabilistaNome string `json:"contabilista_nome"`
	ContabilistaCPF  string `json:"contabilista_cpf"`
	ContabilistaCRC  string `json:"contabilista_crc"`
	ContabilistaCNPJ string `json:"contabilista_cnpj"`
	// Optional pre-populated data
	Participantes     []sped.EFDParticipante    `json:"participantes"`
	Unidades          []sped.EFDUnidade         `json:"unidades"`
	Itens             []sped.EFDItem            `json:"itens"`
	DocumentosFiscais []sped.EFDDocumentoFiscal `json:"documentos_fiscais"`
	Inventario        []sped.EFDInventarioItem  `json:"inventario"`
}

// GenerateEFD handles POST /api/fiscal/sped/efd — generates the EFD ICMS/IPI text file.
func (h *SPEDHandler) GenerateEFD(w http.ResponseWriter, r *http.Request) {
	var req generateEFDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}

	spedReq := fiscal_uc.SpedRequest{
		Empresa: sped.EFDEmpresa{
			CNPJ:             req.CNPJ,
			Nome:             req.Nome,
			UF:               req.UF,
			IE:               req.IE,
			IM:               req.IM,
			SUFRAMA:          req.SUFRAMA,
			CodigoMunicipio:  req.CodigoMunicipio,
			RegimeTributario: req.RegimeTributario,
			ContabilistaNome: req.ContabilistaNome,
			ContabilistaCPF:  req.ContabilistaCPF,
			ContabilistaCRC:  req.ContabilistaCRC,
			ContabilistaCNPJ: req.ContabilistaCNPJ,
		},
		DataInicial:       req.DataInicial,
		DataFinal:         req.DataFinal,
		IndicadorSituacao: req.IndicadorSituacao,
		Participantes:     req.Participantes,
		Unidades:          req.Unidades,
		Itens:             req.Itens,
		DocumentosFiscais: req.DocumentosFiscais,
		Inventario:        req.Inventario,
	}

	content, err := h.uc.GenerateEFD(r.Context(), spedReq)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="SPED_EFD_ICMS_IPI.txt"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}
