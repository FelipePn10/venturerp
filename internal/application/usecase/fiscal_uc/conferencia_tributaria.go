package fiscal_uc

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

// conferirTributacao lista o que, na tributação gravada nas linhas, impede ou
// pede atenção antes da transmissão. A prévia mostra; a autorização recusa o
// que é IMPEDE — a mesma regra nos dois, para a prévia nunca dizer "pode
// transmitir" e a autorização recusar.
func conferirTributacao(exit *entity.FiscalExit, items []*entity.FiscalExitItem, cfg *entity.FiscalConfig) []response.PreviaPendenciaNFe {
	var p []response.PreviaPendenciaNFe
	add := func(nivel, campo, msg, como string) {
		p = append(p, response.PreviaPendenciaNFe{Nivel: nivel, Campo: campo, Mensagem: msg, ComoResolver: como})
	}
	if exit == nil || cfg == nil || (exit.Finalidade != 0 && exit.Finalidade != 1) {
		return nil
	}
	tributacaoPropria := exit.SourceType != nil && (*exit.SourceType == OrigemBeneficiamento || *exit.SourceType == SourceDevolucaoCompra)

	// ── Regra de ICMS/IPI por linha
	if !tributacaoPropria {
		var semRegra []string
		for _, it := range items {
			if it.RegraICMSIPIID == nil {
				semRegra = append(semRegra, fmt.Sprint(it.Sequence))
			}
		}
		if len(semRegra) > 0 {
			linhas := linhasPorExtenso(semRegra)
			if exigeRegraNaNota(cfg, exit) {
				add(NivelImpede, "itens.regra_icms_ipi",
					fmt.Sprintf("a empresa exige regra de ICMS/IPI e não há regra para %s neste cenário (item/NCM, UF %s, cliente, finalidade)", linhas, derefStr(exit.UFDestinatario)),
					"cadastre a regra em VFIS0320 — Parâmetros de ICMS/IPI (confira no simulador de tributação) e gere a nota novamente")
			} else {
				add(NivelAtencao, "itens.regra_icms_ipi",
					fmt.Sprintf("sem regra de ICMS/IPI no cadastro para %s: tributação pela regra geral da empresa (alíquota interna única)", linhas),
					"confira o ICMS/IPI dessas linhas; para tributar por produto e cenário, cadastre a regra em VFIS0320 — Parâmetros de ICMS/IPI")
			}
		}
	}

	// ── CFOP x destino: 5 dentro do estado, 6 para outro, 7 para o exterior.
	// A SEFAZ rejeita a nota com CFOP de outra localização (idDest).
	ufDest := strings.ToUpper(strings.TrimSpace(derefStr(exit.UFDestinatario)))
	if ufDest != "" {
		quer := byte('6')
		switch {
		case ufDest == "EX":
			quer = '7'
		case strings.EqualFold(ufDest, cfg.UFEmpresa):
			quer = '5'
		}
		for _, it := range items {
			c := strings.TrimSpace(it.Cfop)
			if len(c) == 4 && (c[0] == '5' || c[0] == '6' || c[0] == '7') && c[0] != quer {
				onde := map[byte]string{'5': "dentro do estado", '6': "para outro estado", '7': "para o exterior"}[quer]
				add(NivelImpede, fmt.Sprintf("itens[%d].cfop", it.Sequence),
					fmt.Sprintf("a linha %d tem CFOP %s, mas a venda é %s (destino %s): a SEFAZ recusa", it.Sequence, c, onde, ufDest),
					fmt.Sprintf("use um CFOP iniciado por %c (por exemplo %c101 ou %c102) e gere a nota novamente", quer, quer, quer))
			}
			if it.ValorICMSST > 0 && len(c) == 4 && !cfopDeST[c[1:]] {
				add(NivelAtencao, fmt.Sprintf("itens[%d].cfop_st", it.Sequence),
					fmt.Sprintf("a linha %d tem ICMS-ST, mas o CFOP %s não é de venda com substituição tributária", it.Sequence, c),
					"para venda com ST use 5401/6401 (produção) ou 5403/6403 (revenda)")
			}
		}
	}

	// ── CST do regime: CSOSN (3 dígitos) no Simples, CST (2 dígitos) no normal.
	// A nota guarda o que foi calculado; se o regime mudou depois, ela está velha.
	simples := regimeSimples(cfg)
	for _, it := range items {
		cst := strings.TrimSpace(derefStr(it.CstICMS))
		if cst == "" {
			continue
		}
		switch {
		case simples && len(cst) == 2:
			add(NivelImpede, fmt.Sprintf("itens[%d].cst_icms", it.Sequence),
				fmt.Sprintf("a empresa é do Simples Nacional, mas a linha %d está com CST %s do regime normal", it.Sequence, cst),
				"gere a nota novamente para o sistema calcular o CSOSN")
		case !simples && len(cst) == 3:
			add(NivelImpede, fmt.Sprintf("itens[%d].cst_icms", it.Sequence),
				fmt.Sprintf("a empresa é do regime normal, mas a linha %d está com CSOSN %s do Simples Nacional", it.Sequence, cst),
				"confira o regime em VFIS0100 — Configuração Fiscal e gere a nota novamente")
		case simples && it.ValorICMSST > 0 && !csosnComST[cst]:
			add(NivelImpede, fmt.Sprintf("itens[%d].cst_icms", it.Sequence),
				fmt.Sprintf("a linha %d calcula ICMS-ST, mas o CSOSN %s é de operação sem substituição", it.Sequence, cst),
				"use CSOSN 201, 202, 203 ou 900 na regra de ICMS/IPI (VFIS0320), ou tire a MVA, e gere a nota novamente")
		case simples && (cst == "101" || cst == "201") && it.AliqCreditoSimples <= 0:
			add(NivelImpede, fmt.Sprintf("itens[%d].credito_simples", it.Sequence),
				fmt.Sprintf("a linha %d tem CSOSN %s (tributada com crédito) sem a alíquota de crédito do Simples (pCredSN), que a SEFAZ exige", it.Sequence, cst),
				"informe a alíquota de crédito do Simples em VFIS0100 — Configuração Fiscal, ou troque o CSOSN da regra para 102, e gere a nota novamente")
		}
	}

	// ── DIFAL: venda interestadual de empresa do regime normal a não
	// contribuinte leva o grupo ICMSUFDest; sem ele a SEFAZ recusa (694).
	interestadual := exit.UFDestinatario != nil && strings.TrimSpace(*exit.UFDestinatario) != "" &&
		!strings.EqualFold(strings.TrimSpace(*exit.UFDestinatario), cfg.UFEmpresa)
	if _, indIE := indicadoresDestinatario(exit); interestadual && indIE == 9 && !simples && !tributacaoPropria {
		for _, it := range items {
			cst := strings.TrimSpace(derefStr(it.CstICMS))
			if it.ICMSUFDestBase > 0 || cstICMSSemImpostoProprio[cst] {
				continue
			}
			add(NivelImpede, fmt.Sprintf("itens[%d].difal", it.Sequence),
				fmt.Sprintf("a linha %d é venda interestadual a não contribuinte e está sem o grupo do DIFAL (ICMS da UF de destino)", it.Sequence),
				fmt.Sprintf("confira a alíquota interna de %s na tabela de ICMS por UF e gere a nota novamente (nota gerada antes do cálculo do DIFAL também precisa ser refeita)", derefStr(exit.UFDestinatario)))
		}
	}
	// O DIFAL usa a alíquota interna e o FCP da tabela de ICMS por UF, que
	// mudam por lei estadual: quem transmite confere o que foi usado.
	for _, it := range items {
		if it.ICMSUFDestBase <= 0 {
			continue
		}
		msg := fmt.Sprintf("a linha %d leva DIFAL para %s com alíquota interna de %s%%", it.Sequence, derefStr(exit.UFDestinatario), pctTexto(it.ICMSUFDestAliqInterna))
		if it.FCPUFDestPct > 0 {
			msg += fmt.Sprintf(" e FCP de %s%%", pctTexto(it.FCPUFDestPct))
		}
		add(NivelAtencao, fmt.Sprintf("itens[%d].difal_aliquotas", it.Sequence), msg,
			"confira na tabela de ICMS por UF se a alíquota e o FCP valem para esta mercadoria no estado de destino (o FCP só incide sobre o que a lei do estado define)")
		break
	}
	return p
}

// cfopDeST: os três últimos dígitos dos CFOPs de venda com ST.
var cfopDeST = map[string]bool{"401": true, "402": true, "403": true, "404": true, "405": true}

// csosnComST: CSOSN que levam o grupo da substituição tributária.
var csosnComST = map[string]bool{"201": true, "202": true, "203": true, "900": true}

// cstICMSSemImpostoProprio: sem ICMS próprio, sem DIFAL na nota.
var cstICMSSemImpostoProprio = map[string]bool{"40": true, "41": true, "50": true, "60": true}

// recusarPendenciasTributarias: a autorização não transmite com IMPEDE.
func recusarPendenciasTributarias(exit *entity.FiscalExit, items []*entity.FiscalExitItem, cfg *entity.FiscalConfig) error {
	var msgs []string
	for _, p := range conferirTributacao(exit, items, cfg) {
		if p.Nivel == NivelImpede {
			msgs = append(msgs, p.Mensagem+" — "+p.ComoResolver)
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return errorsuc.NewValidationError("a tributação da nota tem pendências: " + strings.Join(msgs, "; "))
}

// divergenciasDoPedido compara o IPI e o ICMS da nota com os do pedido de
// venda (que o vendedor pode ter ajustado à mão): o cliente comprou por um
// número e a nota vai sair com outro. Só avisa — a nota é a que vale.
func divergenciasDoPedido(ctx context.Context, orders salesrepo.SalesOrderRepository, exit *entity.FiscalExit, items []*entity.FiscalExitItem) ([]response.PreviaPendenciaNFe, error) {
	if orders == nil || exit == nil || exit.SalesOrderCode == nil {
		return nil, nil
	}
	linhas, err := orders.ListItems(ctx, *exit.SalesOrderCode)
	if err != nil {
		if errorsuc.EhNaoEncontrado(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lendo as linhas do pedido para conferir a tributação: %w", err)
	}
	porCodigo := make(map[int64]float64, len(linhas))
	icms := make(map[int64]float64, len(linhas))
	for _, l := range linhas {
		porCodigo[l.Code], icms[l.Code] = l.IPIPct, l.ICMSPct
	}
	var out []response.PreviaPendenciaNFe
	for _, it := range items {
		if it.SalesOrderItemCode == nil {
			continue
		}
		ipiPedido, ok := porCodigo[*it.SalesOrderItemCode]
		if !ok {
			continue
		}
		ipiNota := it.AliqIPI * 100
		if math.Abs(ipiNota-ipiPedido) > 0.005 {
			out = append(out, response.PreviaPendenciaNFe{Nivel: NivelAtencao, Campo: fmt.Sprintf("itens[%d].aliq_ipi", it.Sequence),
				Mensagem:     fmt.Sprintf("a linha %d sai com IPI de %s%% e o pedido foi negociado com %s%%", it.Sequence, pctTexto(ipiNota), pctTexto(ipiPedido)),
				ComoResolver: "confira com o vendedor: a nota segue a regra de ICMS/IPI; se o pedido estava certo, ajuste a regra em VFIS0320"})
		}
		icmsPedido := icms[*it.SalesOrderItemCode]
		icmsNota := it.AliqICMS * 100
		if icmsPedido > 0 && len(strings.TrimSpace(derefStr(it.CstICMS))) == 2 && math.Abs(icmsNota-icmsPedido) > 0.005 {
			out = append(out, response.PreviaPendenciaNFe{Nivel: NivelAtencao, Campo: fmt.Sprintf("itens[%d].aliq_icms", it.Sequence),
				Mensagem:     fmt.Sprintf("a linha %d sai com ICMS de %s%% e o pedido foi negociado com %s%%", it.Sequence, pctTexto(icmsNota), pctTexto(icmsPedido)),
				ComoResolver: "confira com o vendedor: a nota segue a regra de ICMS/IPI; se o pedido estava certo, ajuste a regra em VFIS0320"})
		}
	}
	return out, nil
}

// pctTexto: percentual com vírgula decimal, como a tela mostra (19,50).
func pctTexto(v float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", v), ".", ",", 1)
}
