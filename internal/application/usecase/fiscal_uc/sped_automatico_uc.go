package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
)

// SpedAutomaticoUseCase gera a EFD ICMS/IPI do mês a partir das próprias
// notas do sistema: entradas aprovadas, NF-e emitidas (autorizadas e
// canceladas) e CT-e de frete lançados, com a apuração do ICMS e do IPI.
type SpedAutomaticoUseCase struct {
	Periodo repository.SpedRepository
	Config  interface {
		GetFiscalConfig(ctx context.Context) (*fiscalEntity.FiscalConfig, error)
	}
}

type SpedAutomaticoRequest struct {
	Ano, Mes int
	// Finalidade: 0 original, 1 substituta (retificadora).
	Finalidade string
	// Perfil de apresentação definido pela SEFAZ (A, B ou C).
	Perfil string
	// IndAtividade: 0 industrial ou equiparado, 1 outros.
	IndAtividade                                    string
	ContribuinteIPI                                 *bool
	ContabilistaNome, ContabilistaCPF               string
	ContabilistaCRC, ContabilistaCNPJ               string
	SaldoCredorAnteriorICMS, SaldoCredorAnteriorIPI decimal.Decimal
	CodReceitaICMS                                  string
	VencimentoICMS                                  *time.Time
	// Inventario (bloco H): data do inventário (em geral 31/12, entregue na
	// EFD de fevereiro) e motivo (01 final do período, 02 mudança de forma de
	// tributação, 03 baixa cadastral, 04 regime de pagamento, 05 determinação
	// dos fiscos, 06 controle das mercadorias sujeitas à ST). Nil = bloco H sem movimento.
	InventarioData   *time.Time
	InventarioMotivo string
}

type SpedResumo struct {
	Entradas, Saidas, Canceladas, Fretes int
	Participantes, Itens                 int
	ItensInventario                      int
	ValorInventario                      decimal.Decimal
	ICMSDebitos, ICMSCreditos            decimal.Decimal
	ICMSRecolher, ICMSSaldoCredor        decimal.Decimal
	IPIDebitos, IPICreditos              decimal.Decimal
	Linhas                               int
}

type SpedAutomaticoResultado struct {
	Arquivo     string
	NomeArquivo string
	Resumo      SpedResumo
	// Avisos: o que o PVA vai cobrar e só o cadastro resolve (município do
	// participante, NCM do item...). O arquivo é gerado mesmo assim.
	Avisos []string
}

func (uc *SpedAutomaticoUseCase) Gerar(ctx context.Context, req SpedAutomaticoRequest) (*SpedAutomaticoResultado, error) {
	if req.Mes < 1 || req.Mes > 12 || req.Ano < 2009 || req.Ano > 2100 {
		return nil, errorsuc.NewValidationError("informe o mês e o ano da escrituração")
	}
	inicio := time.Date(req.Ano, time.Month(req.Mes), 1, 0, 0, 0, 0, time.UTC)
	fim := inicio.AddDate(0, 1, -1)
	if req.Finalidade == "" {
		req.Finalidade = "0"
	}
	if req.Finalidade != "0" && req.Finalidade != "1" {
		return nil, errorsuc.NewValidationError("finalidade deve ser 0 (original) ou 1 (substituta)")
	}
	req.Perfil = strings.ToUpper(strings.TrimSpace(req.Perfil))
	if req.Perfil == "" {
		req.Perfil = "A"
	}
	if req.Perfil != "A" && req.Perfil != "B" && req.Perfil != "C" {
		return nil, errorsuc.NewValidationError("perfil deve ser A, B ou C")
	}
	if req.IndAtividade == "" {
		req.IndAtividade = "0"
	}
	if req.IndAtividade != "0" && req.IndAtividade != "1" {
		return nil, errorsuc.NewValidationError("indicador de atividade deve ser 0 (industrial) ou 1 (outros)")
	}
	cpf := soDigitos(req.ContabilistaCPF)
	if strings.TrimSpace(req.ContabilistaNome) == "" || len(cpf) != 11 {
		return nil, errorsuc.NewValidationError("informe o nome e o CPF do contabilista (registro 0100)")
	}
	if req.SaldoCredorAnteriorICMS.IsNegative() || req.SaldoCredorAnteriorIPI.IsNegative() {
		return nil, errorsuc.NewValidationError("saldo credor anterior não pode ser negativo")
	}

	d, err := uc.Periodo.CarregarPeriodoEFD(ctx, inicio, fim)
	if err != nil {
		return nil, err
	}
	var faltando []string
	if len(d.Empresa.CNPJ) != 14 {
		faltando = append(faltando, "CNPJ")
	}
	if d.Empresa.IE == "" {
		faltando = append(faltando, "inscrição estadual")
	}
	if len(d.Empresa.UF) != 2 {
		faltando = append(faltando, "UF")
	}
	if len(d.Empresa.CodigoMunicipio) != 7 {
		faltando = append(faltando, "código do município (IBGE)")
	}
	if len(faltando) > 0 {
		return nil, errorsuc.NewValidationError("complete a configuração fiscal da empresa: " + strings.Join(faltando, ", "))
	}

	d.Empresa.RegimeTributario = req.Perfil
	d.Empresa.IndAtividade = req.IndAtividade
	d.Empresa.ContabilistaNome = strings.TrimSpace(req.ContabilistaNome)
	d.Empresa.ContabilistaCPF = cpf
	d.Empresa.ContabilistaCRC = strings.TrimSpace(req.ContabilistaCRC)
	d.Empresa.ContabilistaCNPJ = soDigitos(req.ContabilistaCNPJ)
	d.Periodo = sped.EFDPeriodo{DataInicial: inicio, DataFinal: fim, IndicadorSituacaoEspecial: req.Finalidade}
	d.SaldoCredorAnteriorICMS = req.SaldoCredorAnteriorICMS
	d.SaldoCredorAnteriorIPI = req.SaldoCredorAnteriorIPI
	d.CodReceitaICMS = strings.TrimSpace(req.CodReceitaICMS)
	d.ContribuinteIPI = req.IndAtividade == "0"
	if req.ContribuinteIPI != nil {
		d.ContribuinteIPI = *req.ContribuinteIPI
	}
	if req.VencimentoICMS != nil {
		d.VencimentoICMS = *req.VencimentoICMS
	} else {
		d.VencimentoICMS = uc.vencimentoPadrao(ctx, fim)
	}

	var avisosInventario []string
	if req.InventarioData != nil {
		if req.InventarioData.After(fim) {
			return nil, errorsuc.NewValidationError("a data do inventário não pode ser posterior ao fim do período da escrituração")
		}
		motivo := firstNonEmpty(strings.TrimSpace(req.InventarioMotivo), "01")
		if len(motivo) != 2 || motivo < "01" || motivo > "06" {
			return nil, errorsuc.NewValidationError("motivo do inventário deve ser de 01 a 06")
		}
		itens, conta, err := uc.Periodo.InventarioEFD(ctx, *req.InventarioData)
		if err != nil {
			return nil, err
		}
		d.InventarioItens, d.DataInventario, d.ContaEstoque, d.MotivoInventario = itens, *req.InventarioData, conta, motivo
		if conta == "" && len(itens) > 0 {
			avisosInventario = append(avisosInventario, "Inventário sem a conta contábil de estoque (H010/COD_CTA): informe a conta de Estoque nos parâmetros de contabilização (VCTB0200).")
		}
		for _, it := range itens {
			if !it.Valor.IsPositive() {
				avisosInventario = append(avisosInventario, fmt.Sprintf("Item %s (%s) no inventário sem valor: confira o custo do item.", it.Item.Cod, it.Item.Desc))
			}
		}
	}

	params := sped.Montar(*d)
	arquivo := sped.Generate(params)

	res := &SpedAutomaticoResultado{Arquivo: arquivo,
		NomeArquivo: fmt.Sprintf("EFD_ICMS_IPI_%s_%04d%02d.txt", d.Empresa.CNPJ, req.Ano, req.Mes)}
	r := &res.Resumo
	r.Entradas, r.Fretes = len(d.Entradas), len(d.Fretes)
	for _, s := range d.Saidas {
		if s.Cancelada {
			r.Canceladas++
		} else {
			r.Saidas++
		}
	}
	r.Participantes, r.Itens = len(params.Participantes), len(params.Itens)
	if a := params.ApuracaoICMS; a != nil {
		r.ICMSDebitos, r.ICMSCreditos = decimal.NewFromFloat(a.VlTotDebitos), decimal.NewFromFloat(a.VlTotCreditos)
		r.ICMSRecolher, r.ICMSSaldoCredor = decimal.NewFromFloat(a.VlIcmsRecolher), decimal.NewFromFloat(a.VlSaldoCredorTransp)
	}
	if a := params.ApuracaoIPI; a != nil {
		r.IPIDebitos, r.IPICreditos = decimal.NewFromFloat(a.Deb), decimal.NewFromFloat(a.Cred)
	}
	r.Linhas = strings.Count(arquivo, "\n")
	r.ItensInventario = len(params.Inventario)
	for _, inv := range params.Inventario {
		r.ValorInventario = r.ValorInventario.Add(decimal.NewFromFloat(inv.VlItem))
	}
	res.Avisos = append(avisosEFD(params, d.CodReceitaICMS), avisosInventario...)
	for _, aj := range d.AjustesApuracao {
		if _, ok := sped.TipoAjuste(aj.Codigo); !ok {
			res.Avisos = append(res.Avisos, fmt.Sprintf("Nota de ajuste \"%s\" (%s) ficou fora do E111: o código de ajuste %q não é da apuração do ICMS próprio (tabela 5.1.1, 8 caracteres).",
				aj.Descricao, aj.Valor.StringFixed(2), aj.Codigo))
		}
	}
	return res, nil
}

// vencimentoPadrao: o dia de vencimento do ICMS da configuração fiscal, no mês
// seguinte ao da apuração (sem configuração, dia 10).
func (uc *SpedAutomaticoUseCase) vencimentoPadrao(ctx context.Context, fim time.Time) time.Time {
	dia := 10
	if uc.Config != nil {
		if cfg, err := uc.Config.GetFiscalConfig(ctx); err == nil && cfg != nil && cfg.VencimentoIcmsDia >= 1 && cfg.VencimentoIcmsDia <= 31 {
			dia = cfg.VencimentoIcmsDia
		}
	}
	prox := time.Date(fim.Year(), fim.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	ultimo := prox.AddDate(0, 1, -1).Day()
	if dia > ultimo {
		dia = ultimo
	}
	return time.Date(prox.Year(), prox.Month(), dia, 0, 0, 0, 0, time.UTC)
}

func avisosEFD(p sped.EFDParams, codReceita string) []string {
	var avisos []string
	for _, pt := range p.Participantes {
		if len(pt.CodigoMunicipio) != 7 {
			avisos = append(avisos, fmt.Sprintf("Participante %s (%s) sem código de município (0150/COD_MUN).", pt.Nome, pt.CodPart))
		}
	}
	for _, it := range p.Itens {
		if it.TipoItem != "09" && it.TipoItem != "07" && it.TipoItem != "08" && len(it.CodNCM) != 8 {
			avisos = append(avisos, fmt.Sprintf("Item %s (%s) sem NCM (0200/COD_NCM).", it.CodItem, it.DescItem))
		}
	}
	for _, c := range p.Conhecimentos {
		if len(c.CodMunOrig) != 7 || len(c.CodMunDest) != 7 {
			avisos = append(avisos, fmt.Sprintf("CT-e %s sem os municípios de origem/destino (D100) — anexe o XML do CT-e.", c.NumDoc))
		}
	}
	if a := p.ApuracaoICMS; a != nil && len(a.Obrigacoes) > 0 && codReceita == "" {
		avisos = append(avisos, "Há ICMS a recolher: informe o código de receita da UF (E116/COD_REC).")
	}
	return avisos
}
