package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"

	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

// SimularTributacaoUseCase calcula a tributação de uma venda sem gravar nada:
// o contador confere a parametrização antes de faturar, e o orçamento e o
// pedido sugerem IPI, ICMS e ST com o mesmo cálculo que a nota vai fazer.
type SimularTributacaoUseCase struct {
	Tributador *Tributador
	Customers  customerrepo.CustomerRepository
	Orders     salesrepo.SalesOrderRepository
	Itens      interface {
		ItemsByCode(ctx context.Context, codes []int64) (map[int64]repository.ItemCandidate, error)
	}
	// ItemsRepo resolve o código do item que a pessoa conhece (comercial) na
	// chave interna, como o pedido faz (itemresolution).
	ItemsRepo any
	Auth      ports.AuthService
}

// SimularTributacaoDTO: o cliente (ou UF/IE à mão), a finalidade, o tipo de
// nota e as linhas. NCM e origem vêm do cadastro do item quando não informados.
type SimularTributacaoDTO struct {
	CustomerCode     *int64                     `json:"customer_code,omitempty"`
	SalesOrderCode   *int64                     `json:"sales_order_code,omitempty"`
	UFDestino        *string                    `json:"uf_destino,omitempty"`
	IEDestinatario   *string                    `json:"ie_destinatario,omitempty"`
	TipoPessoa       *string                    `json:"tipo_pessoa,omitempty"`
	FinalidadeCompra *string                    `json:"finalidade_compra,omitempty"`
	InvoiceTypeID    *int64                     `json:"invoice_type_id,omitempty"`
	Cfop             string                     `json:"cfop,omitempty"`
	ValorFrete       float64                    `json:"valor_frete,omitempty"`
	ValorDesconto    float64                    `json:"valor_desconto,omitempty"`
	Itens            []SimularTributacaoItemDTO `json:"itens"`
}

type SimularTributacaoItemDTO struct {
	// ItemCode é o código do item (o comercial, como na tela); NCM e origem vêm
	// do cadastro dele quando não informados.
	ItemCode      *request.TextCode `json:"item_code,omitempty"`
	Ncm           *string           `json:"ncm,omitempty"`
	Origem        *string           `json:"origem_mercadoria,omitempty"`
	Quantidade    float64           `json:"quantidade"`
	ValorUnitario float64           `json:"valor_unitario"`
}

// SimulacaoTributacao é o resultado: o contexto que valeu, cada linha com a
// regra e a explicação, os totais e as pendências que impediriam a nota.
type SimulacaoTributacao struct {
	Regime           string               `json:"regime"`
	UFOrigem         string               `json:"uf_origem"`
	UFDestino        string               `json:"uf_destino"`
	TipoDestinatario string               `json:"tipo_destinatario"`
	FinalidadeCompra string               `json:"finalidade_compra,omitempty"`
	OrigemFinalidade string               `json:"origem_finalidade"`
	ConsumidorFinal  bool                 `json:"consumidor_final"`
	TipoNota         *string              `json:"tipo_nota,omitempty"`
	ExigeRegra       bool                 `json:"exige_regra"`
	Itens            []SimulacaoLinha     `json:"itens"`
	Totais           SimulacaoTotais      `json:"totais"`
	ValorTotalNota   float64              `json:"valor_total_nota"`
	Pendencias       []SimulacaoPendencia `json:"pendencias"`
}

type SimulacaoLinha struct {
	Sequencia     int      `json:"sequencia"`
	ItemCode      *int64   `json:"item_code,omitempty"`
	Ncm           string   `json:"ncm"`
	Origem        string   `json:"origem_mercadoria"`
	ValorProdutos float64  `json:"valor_produtos"`
	RegraICMSIPI  *int64   `json:"regra_icms_ipi_id,omitempty"`
	CSTICMS       string   `json:"cst_icms"`
	BaseICMS      float64  `json:"base_icms"`
	AliqICMSPct   float64  `json:"aliq_icms_pct"`
	ValorICMS     float64  `json:"valor_icms"`
	ReducaoBase   float64  `json:"icms_reducao_base_pct"`
	DiferidoPct   float64  `json:"icms_diferimento_pct"`
	ValorDiferido float64  `json:"valor_icms_diferido"`
	MVAPct        float64  `json:"mva_pct"`
	BaseICMSST    float64  `json:"base_icms_st"`
	AliqICMSSTPct float64  `json:"aliq_icms_st_pct"`
	ValorICMSST   float64  `json:"valor_icms_st"`
	CSTIPI        string   `json:"cst_ipi"`
	AliqIPIPct    float64  `json:"aliq_ipi_pct"`
	ValorIPI      float64  `json:"valor_ipi"`
	CSTPIS        string   `json:"cst_pis"`
	AliqPISPct    float64  `json:"aliq_pis_pct"`
	ValorPIS      float64  `json:"valor_pis"`
	CSTCOFINS     string   `json:"cst_cofins"`
	AliqCOFINSPct float64  `json:"aliq_cofins_pct"`
	ValorCOFINS   float64  `json:"valor_cofins"`
	ValorDIFAL    float64  `json:"valor_difal"`
	ValorFCP      float64  `json:"valor_fcp_uf_dest"`
	CreditoSNPct  float64  `json:"aliq_credito_simples_pct"`
	CreditoSN     float64  `json:"valor_credito_simples"`
	Notas         []string `json:"notas"`
}

// SimulacaoTotais: os totais da operação simulada.
type SimulacaoTotais struct {
	BaseICMS      float64 `json:"base_icms"`
	ValorICMS     float64 `json:"valor_icms"`
	ValorDiferido float64 `json:"valor_icms_diferido"`
	BaseICMSST    float64 `json:"base_icms_st"`
	ValorICMSST   float64 `json:"valor_icms_st"`
	ValorIPI      float64 `json:"valor_ipi"`
	ValorPIS      float64 `json:"valor_pis"`
	ValorCOFINS   float64 `json:"valor_cofins"`
	ValorDIFAL    float64 `json:"valor_difal"`
	ValorFCP      float64 `json:"valor_fcp_uf_dest"`
}

type SimulacaoPendencia struct {
	Nivel    string `json:"nivel"`
	Mensagem string `json:"mensagem"`
}

// pct: fração do motor (0,12) em percentual (12), com quatro casas.
func pct(f float64) float64 { return decimal.NewFromFloat(f).Mul(cemDec).Round(4).InexactFloat64() }

func (uc *SimularTributacaoUseCase) Execute(ctx context.Context, dto SimularTributacaoDTO) (*SimulacaoTributacao, error) {
	if uc.Auth != nil && !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(dto.Itens) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos um item para simular")
	}

	// Destinatário: o informado vence; o que faltar vem do cliente (ou do
	// pedido), como na nota.
	dest := request.CreateFiscalExitDTO{
		CustomerCode: dto.CustomerCode, SalesOrderCode: dto.SalesOrderCode,
		UFDestinatario: dto.UFDestino, IEDestinatario: dto.IEDestinatario, TipoPessoa: dto.TipoPessoa,
	}
	if err := resolverDestinatario(ctx, &dest, uc.Customers, uc.Orders); err != nil {
		return nil, err
	}
	uf := strings.ToUpper(strings.TrimSpace(derefStr(dest.UFDestinatario)))
	if uf == "" {
		return nil, errorsuc.NewValidationError("informe a UF de destino ou um cliente com endereço completo")
	}

	// Código do item → chave interna; NCM e origem do cadastro do item.
	internos := make([]*int64, len(dto.Itens))
	codigos := []int64{}
	for i, it := range dto.Itens {
		if it.ItemCode == nil || strings.TrimSpace(it.ItemCode.String()) == "" {
			continue
		}
		item, err := itemresolution.Resolve(ctx, uc.ItemsRepo, *it.ItemCode)
		if err != nil {
			return nil, err
		}
		c := int64(item.Code)
		internos[i] = &c
		codigos = append(codigos, c)
	}
	cadastro := map[int64]repository.ItemCandidate{}
	if len(codigos) > 0 && uc.Itens != nil {
		var err error
		if cadastro, err = uc.Itens.ItemsByCode(ctx, codigos); err != nil {
			return nil, err
		}
	}

	op := OperacaoTributavel{
		UFDestino: uf, IE: dest.IEDestinatario, TipoPessoa: dest.TipoPessoa, Cliente: dest.CustomerCode,
		Pedido: dto.SalesOrderCode, TipoNotaID: dto.InvoiceTypeID, Cfop: dto.Cfop,
		ValorFrete: dto.ValorFrete, ValorDesconto: dto.ValorDesconto,
	}
	if dto.FinalidadeCompra != nil {
		op.Finalidade = *dto.FinalidadeCompra
	}
	linhas := make([]SimulacaoLinha, len(dto.Itens))
	var pend []SimulacaoPendencia
	for i, it := range dto.Itens {
		if it.Quantidade <= 0 {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: a quantidade deve ser maior que zero", i+1))
		}
		ncm, origem := "", "0"
		if internos[i] != nil {
			c, ok := cadastro[*internos[i]]
			if !ok {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: o item %s não existe nesta empresa", i+1, it.ItemCode.String()))
			}
			ncm = c.NCM
			if strings.TrimSpace(c.Origem) != "" {
				origem = c.Origem
			}
		}
		if it.Ncm != nil && strings.TrimSpace(*it.Ncm) != "" {
			ncm = *it.Ncm
		}
		if it.Origem != nil && strings.TrimSpace(*it.Origem) != "" {
			origem = *it.Origem
		}
		if engine.NormalizarNCM(ncm) == "" {
			pend = append(pend, SimulacaoPendencia{Nivel: NivelImpede, Mensagem: fmt.Sprintf("linha %d sem NCM: a nota não sai sem ele", i+1)})
		}
		n := ncm
		op.Itens = append(op.Itens, LinhaTributavel{ItemCode: internos[i], Ncm: &n, Quantidade: it.Quantidade, ValorUnitario: it.ValorUnitario, Origem: origem})
		linhas[i] = SimulacaoLinha{Sequencia: i + 1, ItemCode: internos[i], Ncm: engine.NormalizarNCM(ncm), Origem: origem,
			ValorProdutos: decimal.NewFromFloat(it.Quantidade).Mul(decimal.NewFromFloat(it.ValorUnitario)).Round(2).InexactFloat64()}
	}

	t, err := uc.Tributador.Calcular(ctx, op)
	if err != nil {
		return nil, err
	}
	out := &SimulacaoTributacao{
		Regime: string(t.Regime), UFOrigem: t.Config.UFEmpresa, UFDestino: uf, TipoDestinatario: t.DestinoTipo,
		FinalidadeCompra: t.Finalidade, OrigemFinalidade: t.OrigemFinalidade, ConsumidorFinal: t.ConsumidorFinal,
		ExigeRegra: t.Config.ExigirRegraICMSIPI,
	}
	// Sem a UF da empresa toda operação parece interestadual (alíquota, DIFAL e
	// MVA ajustada saem errados); a prévia da nota já impede, o simulador avisa.
	if strings.TrimSpace(t.Config.UFEmpresa) == "" {
		pend = append(pend, SimulacaoPendencia{Nivel: NivelImpede, Mensagem: "a UF da empresa não está preenchida na Configuração Fiscal (VFIS0100): sem ela o cálculo trata toda operação como interestadual"})
	}
	if t.TipoNota != nil {
		d := fmt.Sprintf("%d — %s", t.TipoNota.Code, t.TipoNota.Descricao)
		out.TipoNota = &d
	}
	produtos := decimal.Zero
	for i, r := range t.Resultado.Itens {
		l := &linhas[i]
		if t.Regras[i] != nil {
			id := t.Regras[i].ID
			l.RegraICMSIPI = &id
		}
		l.CSTICMS, l.BaseICMS, l.AliqICMSPct, l.ValorICMS = r.CSTICMS, r.BaseICMS, pct(r.AliquotaICMS), r.ValorICMS
		l.ReducaoBase, l.DiferidoPct, l.ValorDiferido = r.ReducaoBaseICMSPct, r.DiferimentoICMSPct, r.ValorICMSDiferido
		l.MVAPct, l.BaseICMSST, l.AliqICMSSTPct, l.ValorICMSST = pct(r.MVA), r.BaseICMSST, pct(r.AliquotaICMSST), r.ValorICMSST
		l.CSTIPI, l.AliqIPIPct, l.ValorIPI = r.CSTIPI, pct(r.AliquotaIPI), r.ValorIPI
		l.CSTPIS, l.AliqPISPct, l.ValorPIS = r.CSTPIS, pct(r.AliquotaPIS), r.ValorPIS
		l.CSTCOFINS, l.AliqCOFINSPct, l.ValorCOFINS = r.CSTCOFINS, pct(r.AliquotaCOFINS), r.ValorCOFINS
		l.ValorDIFAL, l.ValorFCP, l.CreditoSNPct, l.CreditoSN = r.ValorDIFAL, r.ValorFCP, r.AliqCreditoSimplesPct, r.ValorCreditoSimples
		l.Notas = r.Notas
		produtos = produtos.Add(decimal.NewFromFloat(l.ValorProdutos))
	}
	if sem := t.LinhasSemRegra(); len(sem) > 0 {
		nums := make([]string, 0, len(sem))
		for _, i := range sem {
			nums = append(nums, fmt.Sprint(i+1))
		}
		quais := linhasPorExtenso(nums)
		nivel, msg := NivelAtencao, strings.ToUpper(quais[:1])+quais[1:]+" não tem regra de ICMS/IPI no cadastro: seria tributada pela regra geral da empresa (alíquota interna única)"
		if len(nums) > 1 {
			msg = strings.ToUpper(quais[:1]) + quais[1:] + " não têm regra de ICMS/IPI no cadastro: seriam tributadas pela regra geral da empresa (alíquota interna única)"
		}
		if t.Config.ExigirRegraICMSIPI {
			nivel, msg = NivelImpede, "a empresa exige regra de ICMS/IPI e não há regra para "+quais+" neste cenário: a nota não seria transmitida"
		}
		pend = append(pend, SimulacaoPendencia{Nivel: nivel, Mensagem: msg})
	}
	for i, r := range t.Resultado.Itens {
		if (r.CSTICMS == "101" || r.CSTICMS == "201") && r.AliqCreditoSimplesPct <= 0 {
			pend = append(pend, SimulacaoPendencia{Nivel: NivelImpede, Mensagem: fmt.Sprintf("linha %d: CSOSN %s sem alíquota de crédito do Simples configurada", i+1, r.CSTICMS)})
		}
	}
	out.Itens, out.Pendencias = linhas, pend
	if out.Pendencias == nil {
		out.Pendencias = []SimulacaoPendencia{}
	}
	tot := t.Resultado.Totais
	out.Totais = SimulacaoTotais{BaseICMS: tot.BaseICMS, ValorICMS: tot.ValorICMS, ValorDiferido: tot.ValorDiferido,
		BaseICMSST: tot.BaseICMSST, ValorICMSST: tot.ValorICMSST, ValorIPI: tot.ValorIPI, ValorPIS: tot.ValorPIS,
		ValorCOFINS: tot.ValorCOFINS, ValorDIFAL: tot.ValorDIFAL, ValorFCP: tot.ValorFCP}
	out.ValorTotalNota = produtos.Add(decimal.NewFromFloat(tot.ValorIPI)).Add(decimal.NewFromFloat(tot.ValorICMSST)).
		Add(decimal.NewFromFloat(dto.ValorFrete)).Sub(decimal.NewFromFloat(dto.ValorDesconto)).Round(2).InexactFloat64()
	return out, nil
}

// linhasPorExtenso: "a linha 2" ou "as linhas 1, 2 e 3".
func linhasPorExtenso(nums []string) string {
	switch len(nums) {
	case 0:
		return ""
	case 1:
		return "a linha " + nums[0]
	}
	return "as linhas " + strings.Join(nums[:len(nums)-1], ", ") + " e " + nums[len(nums)-1]
}
