package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// SpedAutomaticoHandler gera a EFD ICMS/IPI do mês a partir das notas do sistema.
type SpedAutomaticoHandler struct {
	uc *fiscal_uc.SpedAutomaticoUseCase
}

func NewSpedAutomaticoHandler(uc *fiscal_uc.SpedAutomaticoUseCase) *SpedAutomaticoHandler {
	return &SpedAutomaticoHandler{uc: uc}
}

type spedAutomaticoRequest struct {
	Ano                     int             `json:"ano"`
	Mes                     int             `json:"mes"`
	Finalidade              string          `json:"finalidade"`
	Perfil                  string          `json:"perfil"`
	IndAtividade            string          `json:"ind_atividade"`
	ContribuinteIPI         *bool           `json:"contribuinte_ipi"`
	ContabilistaNome        string          `json:"contabilista_nome"`
	ContabilistaCPF         string          `json:"contabilista_cpf"`
	ContabilistaCRC         string          `json:"contabilista_crc"`
	ContabilistaCNPJ        string          `json:"contabilista_cnpj"`
	SaldoCredorAnteriorICMS decimal.Decimal `json:"saldo_credor_anterior_icms"`
	SaldoCredorAnteriorIPI  decimal.Decimal `json:"saldo_credor_anterior_ipi"`
	CodReceitaICMS          string          `json:"cod_receita_icms"`
	VencimentoICMS          string          `json:"vencimento_icms"` // AAAA-MM-DD, opcional
	// Bloco H: data do inventário (AAAA-MM-DD) e motivo (01 a 06); vazio = sem inventário.
	InventarioData   string `json:"inventario_data"`
	InventarioMotivo string `json:"inventario_motivo"`
}

type spedResumoResponse struct {
	Entradas        int             `json:"entradas"`
	Saidas          int             `json:"saidas"`
	Canceladas      int             `json:"canceladas"`
	Fretes          int             `json:"fretes"`
	Participantes   int             `json:"participantes"`
	Itens           int             `json:"itens"`
	ICMSDebitos     decimal.Decimal `json:"icms_debitos"`
	ICMSCreditos    decimal.Decimal `json:"icms_creditos"`
	ICMSRecolher    decimal.Decimal `json:"icms_recolher"`
	ICMSSaldoCredor decimal.Decimal `json:"icms_saldo_credor"`
	IPIDebitos      decimal.Decimal `json:"ipi_debitos"`
	IPICreditos     decimal.Decimal `json:"ipi_creditos"`
	Linhas          int             `json:"linhas"`
	ItensInventario int             `json:"itens_inventario"`
	ValorInventario decimal.Decimal `json:"valor_inventario"`
}

type spedAutomaticoResponse struct {
	Arquivo     string             `json:"arquivo"`
	NomeArquivo string             `json:"nome_arquivo"`
	Resumo      spedResumoResponse `json:"resumo"`
	Avisos      []string           `json:"avisos"`
}

// Gerar — POST /api/fiscal/sped/efd/automatico.
func (h *SpedAutomaticoHandler) Gerar(w http.ResponseWriter, r *http.Request) {
	var req spedAutomaticoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	in := fiscal_uc.SpedAutomaticoRequest{Ano: req.Ano, Mes: req.Mes, Finalidade: req.Finalidade, Perfil: req.Perfil,
		IndAtividade: req.IndAtividade, ContribuinteIPI: req.ContribuinteIPI, ContabilistaNome: req.ContabilistaNome,
		ContabilistaCPF: req.ContabilistaCPF, ContabilistaCRC: req.ContabilistaCRC, ContabilistaCNPJ: req.ContabilistaCNPJ,
		SaldoCredorAnteriorICMS: req.SaldoCredorAnteriorICMS, SaldoCredorAnteriorIPI: req.SaldoCredorAnteriorIPI,
		CodReceitaICMS: req.CodReceitaICMS, InventarioMotivo: req.InventarioMotivo}
	if req.InventarioData != "" {
		v, err := time.Parse("2006-01-02", req.InventarioData)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "data do inventário inválida (use AAAA-MM-DD)")
			return
		}
		in.InventarioData = &v
	}
	if req.VencimentoICMS != "" {
		v, err := time.Parse("2006-01-02", req.VencimentoICMS)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "vencimento do ICMS inválido (use AAAA-MM-DD)")
			return
		}
		in.VencimentoICMS = &v
	}
	out, err := h.uc.Gerar(r.Context(), in)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	s := out.Resumo
	avisos := out.Avisos
	if avisos == nil {
		avisos = []string{}
	}
	security.RespondJSON(w, http.StatusOK, spedAutomaticoResponse{Arquivo: out.Arquivo, NomeArquivo: out.NomeArquivo, Avisos: avisos,
		Resumo: spedResumoResponse{Entradas: s.Entradas, Saidas: s.Saidas, Canceladas: s.Canceladas, Fretes: s.Fretes,
			Participantes: s.Participantes, Itens: s.Itens, ICMSDebitos: s.ICMSDebitos, ICMSCreditos: s.ICMSCreditos,
			ICMSRecolher: s.ICMSRecolher, ICMSSaldoCredor: s.ICMSSaldoCredor, IPIDebitos: s.IPIDebitos, IPICreditos: s.IPICreditos,
			Linhas: s.Linhas, ItensInventario: s.ItensInventario, ValorInventario: s.ValorInventario}})
}
