package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// Tributador calcula os impostos de uma operação de saída do jeito que a nota
// calcula. Nota fiscal, simulador de tributação, orçamento e pedido passam por
// aqui: é o que garante que o número mostrado antes de faturar é o número que
// a nota vai ter.
//
// Para cada linha ele resolve a regra do cadastro de parâmetros de ICMS/IPI
// (VFIS0320) pelo item/NCM, UF, cliente, finalidade e tipo de nota; junta o
// regime da empresa (Simples, Presumido, Real), a finalidade da compra e os
// indicadores do tipo de nota; e chama o motor (engine.CalcularImpostos).
type Tributador struct {
	Repo repository.FiscalRepository
	// Parametros acha a regra de cada linha. Opcional: sem ele vale a regra geral.
	Parametros ResolvedorParametroICMSIPI
	// Cadastro lê finalidade (cliente, pedido) e tipo de nota. Opcional: sem
	// ele a finalidade é só a informada e o tipo de nota é ignorado.
	Cadastro CadastroTributario
}

// ResolvedorParametroICMSIPI acha a regra de ICMS/IPI de uma linha; nil
// quando nenhuma regra do cadastro se aplica.
type ResolvedorParametroICMSIPI interface {
	ParametroICMSIPI(ctx context.Context, q engine.ConsultaParametroICMSIPI) (*engine.ParametroICMSIPI, error)
}

// CadastroTributario lê o que o cadastro diz sobre a operação.
type CadastroTributario interface {
	// FinalidadeDoCliente: a finalidade do cadastro do cliente, ou USO_CONSUMO
	// quando a tributação do cliente é de consumidor; nil = não informada.
	FinalidadeDoCliente(ctx context.Context, customerCode int64) (*string, error)
	// FinalidadeDoPedido: a do pedido de venda, ou a da sua tributação; nil = não informada.
	FinalidadeDoPedido(ctx context.Context, salesOrderCode int64) (*string, error)
	// TipoNota: os indicadores do tipo de nota (VFIS0110) da empresa.
	TipoNota(ctx context.Context, id int64) (*entity.TipoNotaFiscal, error)
}

// OperacaoTributavel descreve a venda a tributar.
type OperacaoTributavel struct {
	UFDestino  string
	IE         *string
	TipoPessoa *string
	Cliente    *int64
	Pedido     *int64
	// Finalidade informada na operação (vence a do pedido e a do cliente).
	Finalidade string
	TipoNotaID *int64
	Cfop       string
	// Frete e desconto da nota inteira; o motor rateia pelo valor das linhas.
	ValorFrete, ValorDesconto float64
	Itens                     []LinhaTributavel
}

// LinhaTributavel é uma linha da operação.
type LinhaTributavel struct {
	ItemCode      *int64
	Ncm           *string
	Quantidade    float64
	ValorUnitario float64
	Origem        string
	// ST informada à mão (fração); vence a MVA da regra.
	MvaPct, AliqInternaDestinoST, RedBaseSTPct float64
}

// TributacaoCalculada é o resultado com o contexto que o produziu.
type TributacaoCalculada struct {
	Config      *entity.FiscalConfig
	Regime      engine.Regime
	DestinoTipo string
	// Finalidade que valeu (vazio = deduzida do destinatário) e de onde veio.
	Finalidade       string
	OrigemFinalidade string
	ConsumidorFinal  bool
	TipoNota         *entity.TipoNotaFiscal
	// Regras[i] é a regra da linha i (nil = regra geral da empresa).
	Regras    []*engine.ParametroICMSIPI
	Resultado *engine.TaxCalculationResult
}

// LinhasSemRegra devolve as posições (0-based) das linhas sem regra.
func (t *TributacaoCalculada) LinhasSemRegra() []int {
	var out []int
	for i, r := range t.Regras {
		if r == nil {
			out = append(out, i)
		}
	}
	return out
}

// tipoDestinatario: quem tem inscrição estadual é contribuinte do ICMS — inclusive
// a pessoa física (produtor rural), como o XML declara (indIEDest 1). Sem IE,
// ou com "ISENTO" (contribuinte isento), é não contribuinte; a pessoa física
// sem IE é registrada como tal. Antes a pessoa física com IE era tratada como
// não contribuinte: DIFAL e IPI na base indevidos, divergindo do XML.
func tipoDestinatario(ie, tipoPessoa *string) string {
	if ie == nil || strings.TrimSpace(*ie) == "" || strings.EqualFold(strings.TrimSpace(*ie), "ISENTO") {
		if tipoPessoa != nil && *tipoPessoa == "F" {
			return "pessoa_fisica"
		}
		return "nao_contribuinte"
	}
	return "contribuinte"
}

// resolverFinalidade: a informada, a do pedido, a do cliente; nenhuma = vazio
// (deduzida do destinatário).
func (t *Tributador) resolverFinalidade(ctx context.Context, op OperacaoTributavel) (string, string, error) {
	if f := strings.ToUpper(strings.TrimSpace(op.Finalidade)); f != "" {
		if !engine.FinalidadeValida(f) {
			return "", "", errorsuc.NewValidationError(fmt.Sprintf("finalidade da compra %q inválida: use REVENDA, INDUSTRIALIZACAO ou USO_CONSUMO", op.Finalidade))
		}
		return f, "INFORMADA", nil
	}
	if t.Cadastro == nil {
		return "", "DEDUZIDA", nil
	}
	if op.Pedido != nil {
		f, err := t.Cadastro.FinalidadeDoPedido(ctx, *op.Pedido)
		if err != nil {
			return "", "", fmt.Errorf("lendo a finalidade do pedido %d: %w", *op.Pedido, err)
		}
		if f != nil && *f != "" {
			return *f, "PEDIDO", nil
		}
	}
	if op.Cliente != nil {
		f, err := t.Cadastro.FinalidadeDoCliente(ctx, *op.Cliente)
		if err != nil {
			return "", "", fmt.Errorf("lendo a finalidade do cliente %d: %w", *op.Cliente, err)
		}
		if f != nil && *f != "" {
			return *f, "CLIENTE", nil
		}
	}
	return "", "DEDUZIDA", nil
}

// flagsDoTipoNota traduz os indicadores do tipo de nota para o motor. Sem tipo
// de nota, nada muda.
func flagsDoTipoNota(tn *entity.TipoNotaFiscal) engine.InvoiceTypeFlags {
	if tn == nil {
		return engine.InvoiceTypeFlags{}
	}
	f := engine.InvoiceTypeFlags{
		SkipPISCOFINS:  !tn.CalculaPISCOFINS,
		SemDiferimento: !tn.CalculaDiferimento,
		SemST:          !tn.CalculaST,
		SemDIFAL:       !tn.CalculaDIFAL,
	}
	switch strings.ToUpper(strings.TrimSpace(tn.TipoICMS)) {
	case "ISENTO":
		f.CSTICMSOperacao = "40"
	case "OUTROS":
		f.CSTICMSOperacao = "41"
	}
	if tn.ReducaoICMSPct > 0 {
		f.CalcReducao, f.ICMSReductionPct = true, tn.ReducaoICMSPct/100
	}
	return f
}

// Calcular resolve o contexto e calcula a operação.
func (t *Tributador) Calcular(ctx context.Context, op OperacaoTributavel) (*TributacaoCalculada, error) {
	config, err := t.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	ncmTaxes, err := t.Repo.ListNcmTaxes(ctx)
	if err != nil {
		return nil, err
	}
	interstateTable, err := t.Repo.ListICMSInterstate(ctx)
	if err != nil {
		return nil, err
	}
	internalTable, err := t.Repo.ListICMSInternal(ctx)
	if err != nil {
		return nil, err
	}
	ncmMap := make(map[string]*engine.NcmTaxConfig, len(ncmTaxes))
	for _, n := range ncmTaxes {
		ncmMap[n.Ncm] = &engine.NcmTaxConfig{
			AliqIPI: n.AliqIPI, AliqPis: n.AliqPis, AliqCofins: n.AliqCofins,
			CstPis: n.CstPis, CstCofins: n.CstCofins, CstIPI: n.CstIPI,
		}
	}
	internalEngineMap := make(map[string]engine.ICMSInternalConfig, len(internalTable))
	for uf, v := range internalTable {
		internalEngineMap[uf] = engine.ICMSInternalConfig{ICMS: v.ICMS, FCP: v.FCP}
	}

	out := &TributacaoCalculada{
		Config:      config,
		Regime:      engine.RegimeDoCadastro(config.RegimeTributario),
		DestinoTipo: tipoDestinatario(op.IE, op.TipoPessoa),
		Regras:      make([]*engine.ParametroICMSIPI, len(op.Itens)),
	}
	if out.Finalidade, out.OrigemFinalidade, err = t.resolverFinalidade(ctx, op); err != nil {
		return nil, err
	}
	if op.TipoNotaID != nil && t.Cadastro != nil {
		tn, err := t.Cadastro.TipoNota(ctx, *op.TipoNotaID)
		if err != nil {
			return nil, err
		}
		out.TipoNota = tn
	}

	if t.Parametros != nil {
		for i, it := range op.Itens {
			p, err := t.Parametros.ParametroICMSIPI(ctx, engine.ConsultaParametroICMSIPI{
				ItemCode: it.ItemCode, NCM: it.Ncm, UF: op.UFDestino, Cliente: op.Cliente,
				Simples: out.Regime == engine.RegimeSimples, Finalidade: out.Finalidade, TipoNotaID: op.TipoNotaID,
			})
			if err != nil {
				return nil, fmt.Errorf("lendo o parâmetro de ICMS/IPI da linha %d: %w", i+1, err)
			}
			out.Regras[i] = p
		}
	}

	taxItems := make([]engine.TaxItem, 0, len(op.Itens))
	for i, it := range op.Itens {
		ncm := ""
		if it.Ncm != nil {
			ncm = *it.Ncm
		}
		taxItems = append(taxItems, engine.TaxItem{
			Ncm: ncm, ValorUnitario: it.ValorUnitario, Quantidade: it.Quantidade,
			ValorFrete: op.ValorFrete, ValorDesconto: op.ValorDesconto, OrigemMercadoria: it.Origem,
			MvaPct: it.MvaPct, AliqInternaDestinoST: it.AliqInternaDestinoST, RedBaseSTPct: it.RedBaseSTPct,
			Parametro: out.Regras[i],
		})
	}
	params := engine.TaxCalculationParams{
		Itens: taxItems, EmitenteUF: config.UFEmpresa, DestinoUF: op.UFDestino, DestinoTipo: out.DestinoTipo,
		Cfop: op.Cfop, Finalidade: out.Finalidade, Flags: flagsDoTipoNota(out.TipoNota),
	}
	out.ConsumidorFinal = params.ConsumidorFinal()
	fiscalCfg := engine.FiscalConfig{
		UFEmpresa: config.UFEmpresa, IcmsInternoAliquota: config.IcmsInternoAliquota,
		IcmsDiferimentoPercentual: config.IcmsDiferimentoPercentual,
		Regime:                    out.Regime, CreditoSimplesPct: config.AliquotaCreditoSimples,
	}
	scenario := engine.TaxScenarioConfig{
		AliqICMS: config.IcmsInternoAliquota, DifICMSPct: config.IcmsDiferimentoPercentual, CstICMS: "00", CalcDifal: true,
	}
	if out.Resultado, err = engine.CalcularImpostos(params, ncmMap, interstateTable, internalEngineMap, scenario, fiscalCfg); err != nil {
		return nil, err
	}
	return out, nil
}

// exigeRegraNaNota: com a exigência ligada, toda linha de nota normal de venda
// calculada pelo motor precisa de regra. Devolução, complementar, ajuste e as
// notas com tributação própria (beneficiamento, devolução de compra) ficam fora.
func exigeRegraNaNota(cfg *entity.FiscalConfig, exit *entity.FiscalExit) bool {
	if cfg == nil || !cfg.ExigirRegraICMSIPI || exit == nil {
		return false
	}
	if exit.Finalidade != 0 && exit.Finalidade != 1 {
		return false
	}
	if exit.SourceType != nil && (*exit.SourceType == OrigemBeneficiamento || *exit.SourceType == SourceDevolucaoCompra) {
		return false
	}
	return true
}

// semRepetir mantém a ordem e tira os textos repetidos.
func semRepetir(xs []string) []string {
	visto := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !visto[x] {
			visto[x] = true
			out = append(out, x)
		}
	}
	return out
}

// informacoesComplementares monta o infCpl: o texto do usuário, a base legal
// dos benefícios aplicados (dispositivos das regras de ICMS/IPI) e, no Simples
// Nacional, a identificação de optante (Resolução CGSN 140/2018, art. 59) e o
// crédito de ICMS que o comprador pode aproveitar (LC 123/2006, art. 23).
func informacoesComplementares(usuario *string, simples bool, credito decimal.Decimal, creditoPct float64, legais []string) *string {
	var partes []string
	if usuario != nil && strings.TrimSpace(*usuario) != "" {
		partes = append(partes, strings.TrimSpace(*usuario))
	}
	if simples {
		partes = append(partes, "DOCUMENTO EMITIDO POR ME OU EPP OPTANTE PELO SIMPLES NACIONAL. NAO GERA DIREITO A CREDITO FISCAL DE IPI.")
		if credito.IsPositive() {
			partes = append(partes, fmt.Sprintf("PERMITE O APROVEITAMENTO DO CREDITO DE ICMS NO VALOR DE R$ %s, CORRESPONDENTE A ALIQUOTA DE %s%%, NOS TERMOS DO ART. 23 DA LEI COMPLEMENTAR N. 123, DE 2006.",
				strings.Replace(credito.StringFixed(2), ".", ",", 1), strings.Replace(decimal.NewFromFloat(creditoPct).StringFixed(2), ".", ",", 1)))
		}
	}
	partes = append(partes, legais...)
	if len(partes) == 0 {
		return nil
	}
	texto := strings.Join(partes, " ")
	// infCpl aceita até 5.000 caracteres.
	if r := []rune(texto); len(r) > 5000 {
		texto = string(r[:5000])
	}
	return &texto
}
