package customer_material_uc

// Nota de saída do beneficiamento: fatura o serviço e devolve o material do
// cliente na MESMA nota.
//
// As regras abaixo foram confirmadas pela contadora da Usimac em 29/09/2026, sobre
// a NF-e 5.911 de 02/09/2026 (a única emitida de verdade que temos como referência).
// Antes disso, o levantamento do cliente dizia "ICMS = 5,93%" para o serviço, o que
// não confere com a nota real — a contadora confirmou que a nota está certa e que
// não há hoje situação em que os 5,93% sejam destacados.
//
//	linha            CFOP   CST ICMS              PIS/COFINS
//	serviço          5124   051 (diferimento)     CST 01 — 0,65% e 3,00%
//	material         5902   050 (suspensão)       sem destaque
//	sobra e sucata   5903   050 (suspensão)       sem destaque
//
// IRPJ (1,2%) e CSLL (1,08%) NÃO são indicados na NF-e, por orientação da contadora.
//
// Somente CFOP interno: a Usimac não faz industrialização interestadual hoje. Os
// CFOPs 6124/6902/6903 exigem nova validação fiscal antes do primeiro uso, e por
// isso este construtor recusa destinatário de outra UF em vez de inventar o CFOP.

import (
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	"github.com/shopspring/decimal"
)

// CFOPs internos da operação. Confirmados pela contadora; interestadual fica
// deliberadamente de fora.
const (
	CFOPServico  = "5124" // industrialização efetuada para outra empresa
	CFOPRetorno  = "5902" // retorno de mercadoria utilizada na industrialização
	CFOPSobra    = "5903" // retorno de mercadoria não aplicada no processo
	CFOPDiferido = CFOPServico
)

// CST confirmados pela contadora.
const (
	// ⚠️ CST tem DOIS dígitos. A origem da mercadoria vai em campo SEPARADO
	// (`OrigemMercadoria`), e "051"/"050" é a notação de conversa — origem 0 + CST
	// 51/50 — que não vale no documento. A NF-e 5.956 da Usimac, anexada ao
	// levantamento, traz `<orig>0</orig>` e `<CST>51</CST>` em campos distintos;
	// mandar três dígitos ao provedor causa rejeição na autorização.
	CSTICMSServico = "51" // diferimento — Portaria CAT 22/2007
	CSTICMSRetorno = "50" // suspensão
	// CSTPISCOFINS vale para a linha de SERVIÇO: operação tributável com alíquota
	// básica.
	CSTPISCOFINS = "01"
	// O material que VOLTA não é operação tributável de saída: a nota real usa CST
	// 49 ("outras operações de saída"). Deixar o campo vazio faria o autorizador
	// cair no padrão "01" e declarar o material como tributado.
	CSTPISCOFINSRetorno = "49"
)

// Alíquotas de PIS e COFINS sobre o serviço de industrialização.
var (
	AliquotaPIS    = decimal.RequireFromString("0.0065") // 0,65%
	AliquotaCOFINS = decimal.RequireFromString("0.03")   // 3,00%
)

// UFEmitente é a UF da Usimac. Destinatário em outra UF muda todos os CFOPs e
// exige validação fiscal nova, então o construtor para em vez de adivinhar.
const UFEmitente = "SP"

// NCMServico é o NCM da linha de serviço na nota real: serviço não tem NCM de
// mercadoria, e a nota sai com zeros.
const NCMServico = "00000000"

// LinhaDaNota é uma linha pronta para virar item de saída fiscal.
type LinhaDaNota struct {
	Sequencia   int
	Descricao   string
	NCM         string
	CFOP        string
	CSTICMS     string
	Unidade     string
	Quantidade  decimal.Decimal
	ValorUnit   decimal.Decimal
	ValorTotal  decimal.Decimal
	AliqPIS     decimal.Decimal
	ValorPIS    decimal.Decimal
	AliqCOFINS  decimal.Decimal
	ValorCOFINS decimal.Decimal
	CSTPIS      string
	CSTCOFINS   string
	// CodigoProduto é o `cProd` da linha na NF-e. No serviço é o código do produto
	// beneficiado; no material que volta é o código DO CLIENTE, porque o material de
	// terceiro não existe no nosso cadastro.
	CodigoProduto string
	// ItemDaRemessa é a linha de origem, quando a linha devolve material. Nulo na
	// linha de serviço. É o que liga a nota ao saldo que ela vai baixar.
	ItemDaRemessa *entity.ItemRemessa
	// Movimento diz qual baixa esta linha representa, para o faturamento gravar o
	// movimento certo no razão do material de terceiro.
	Movimento entity.TipoMovimento
}

// ServicoFaturado é o que a Usimac cobra pelo beneficiamento.
type ServicoFaturado struct {
	// CodigoItem é o código do serviço na nota (faixa 25xxxxxx na operação atual).
	CodigoItem string
	Descricao  string
	Unidade    string
	Quantidade string
	ValorUnit  string
}

// DevolucaoDeMaterial é uma linha de material do cliente voltando.
type DevolucaoDeMaterial struct {
	RemittanceItemID int64
	// Quantidade a devolver. Não pode passar do saldo em poder da empresa.
	Quantidade string
	// Tipo distingue retorno com o faturamento (5902) de sobra e sucata (5903).
	Tipo entity.TipoMovimento
}

// NotaDeRetorno é o resultado: as linhas e os totais da nota.
type NotaDeRetorno struct {
	NaturezaOperacao string
	UF               string
	Linhas           []LinhaDaNota
	ValorServico     decimal.Decimal
	ValorMaterial    decimal.Decimal
	ValorTotal       decimal.Decimal
	ValorPIS         decimal.Decimal
	ValorCOFINS      decimal.Decimal
	// Observacao vai nos dados adicionais. O diferimento precisa estar escrito na
	// nota: é o que sustenta o ICMS zero na linha do serviço.
	Observacao string
}

// MontarNotaDeRetorno monta as linhas da nota a partir da remessa, do serviço
// cobrado e do material que volta.
//
// Recusa em vez de adivinhar, em três situações que dariam nota errada: saldo
// insuficiente, remessa bloqueada e destinatário fora de SP.
func MontarNotaDeRetorno(
	remessa *entity.Remessa,
	ufDestinatario string,
	servico ServicoFaturado,
	devolucoes []DevolucaoDeMaterial,
) (*NotaDeRetorno, error) {
	if remessa == nil {
		return nil, errorsuc.NewValidationError("informe a remessa de origem")
	}
	if remessa.Blocked {
		return nil, errorsuc.NewConflictError(
			"a remessa está bloqueada; regularize antes de faturar o beneficiamento")
	}
	if remessa.Status == entity.StatusCancelada {
		return nil, errorsuc.NewConflictError("a remessa está cancelada")
	}

	// Interestadual muda 5124/5902/5903 para 6124/6902/6903 e exige validação
	// fiscal nova. A contadora confirmou que hoje não se aplica; emitir com CFOP
	// interno para outra UF é nota rejeitada ou, pior, aceita e errada.
	uf := strings.ToUpper(strings.TrimSpace(ufDestinatario))
	if uf != "" && uf != UFEmitente {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"destinatário em %s: a industrialização interestadual usa os CFOPs 6124/6902/6903 "+
				"e ainda não foi validada pelo responsável fiscal", uf))
	}

	if len(devolucoes) == 0 {
		return nil, errorsuc.NewValidationError(
			"informe o material que retorna: a nota de beneficiamento devolve o material do cliente junto com o serviço")
	}

	nota := &NotaDeRetorno{
		// A natureza é a que a nota real usa. Manter o texto igual evita divergência
		// entre o que o cliente conhece e o que o sistema emite.
		NaturezaOperacao: "Venda de mercadoria para industrializacao",
		UF:               UFEmitente,
		Observacao:       "DIFERIMENTO DO ICMS - PORTARIA CAT 22/2007 INDUSTRIALIZACAO COM SAIDA",
	}

	sequencia := 1

	// ── Linha 1: o serviço ────────────────────────────────────────────────────
	linhaServico, err := montarLinhaDeServico(servico, sequencia)
	if err != nil {
		return nil, err
	}
	nota.Linhas = append(nota.Linhas, *linhaServico)
	nota.ValorServico = linhaServico.ValorTotal
	nota.ValorPIS = linhaServico.ValorPIS
	nota.ValorCOFINS = linhaServico.ValorCOFINS
	sequencia++

	// ── Linhas seguintes: o material do cliente voltando ──────────────────────
	// Acumula por linha de remessa para conferir o saldo uma vez só: duas
	// devoluções da mesma linha na mesma nota somam contra o mesmo saldo.
	porItem := map[int64]decimal.Decimal{}
	itens := map[int64]*entity.ItemRemessa{}
	for _, item := range remessa.Itens {
		itens[item.ID] = item
	}

	for _, dev := range devolucoes {
		item, achou := itens[dev.RemittanceItemID]
		if !achou {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"o item %d não pertence à remessa %d", dev.RemittanceItemID, remessa.NFeNumber))
		}
		quantidade, err := decimal.NewFromString(strings.TrimSpace(dev.Quantidade))
		if err != nil {
			return nil, errorsuc.NewValidationError(
				"quantidade a devolver inválida no item " + item.CustomerItemCode)
		}
		if !quantidade.IsPositive() {
			return nil, errorsuc.NewValidationError(
				"a quantidade a devolver do item " + item.CustomerItemCode + " deve ser maior que zero")
		}
		porItem[item.ID] = porItem[item.ID].Add(quantidade)
		if porItem[item.ID].GreaterThan(item.BalanceQty) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"a devolução de %s do item %s passa do saldo de %s em poder da empresa",
				porItem[item.ID].String(), item.CustomerItemCode, item.BalanceQty.String()))
		}

		cfop := CFOPRetorno
		movimento := dev.Tipo
		switch dev.Tipo {
		case entity.MovimentoRetorno, "":
			cfop = CFOPRetorno
			movimento = entity.MovimentoRetorno
		case entity.MovimentoSobra, entity.MovimentoSucata:
			// Sobra e sucata saem por 5903, ambas com CST 050.
			cfop = CFOPSobra
		default:
			return nil, errorsuc.NewValidationError(
				"a nota de beneficiamento devolve retorno, sobra ou sucata; ajuste não vai na nota")
		}

		// O valor unitário é o da nota de ENTRADA. Devolver com outro valor
		// mudaria o valor do material do cliente no documento fiscal.
		total := quantidade.Mul(item.UnitValue).Round(2)
		nota.Linhas = append(nota.Linhas, LinhaDaNota{
			Sequencia: sequencia,
			Descricao: item.Description,
			// O NCM é o da nota de entrada: o retorno tem de sair com a mesma
			// classificação que o cliente enviou.
			NCM:           item.NCM,
			CFOP:          cfop,
			CSTICMS:       CSTICMSRetorno,
			Unidade:       item.UOM,
			Quantidade:    quantidade,
			ValorUnit:     item.UnitValue,
			ValorTotal:    total,
			ItemDaRemessa: item,
			Movimento:     movimento,
			// Sem VALOR de PIS/COFINS — a suspensão cobre o material que volta —, mas
			// COM o CST 49: é o que a nota real declara, e o campo vazio cairia no
			// padrão "01" do autorizador, declarando o material como tributado.
			CSTPIS:      CSTPISCOFINSRetorno,
			CSTCOFINS:   CSTPISCOFINSRetorno,
			AliqPIS:     decimal.Zero,
			ValorPIS:    decimal.Zero,
			AliqCOFINS:  decimal.Zero,
			ValorCOFINS: decimal.Zero,
			// CodigoProduto é o código DO CLIENTE: o material de terceiro não existe
			// no nosso cadastro, e a nota tem de sair com o código que o cliente usa
			// (10014485, 10014670 na NF-e 5.956).
			CodigoProduto: item.CustomerItemCode,
		})
		nota.ValorMaterial = nota.ValorMaterial.Add(total)
		sequencia++
	}

	nota.ValorTotal = nota.ValorServico.Add(nota.ValorMaterial)
	return nota, nil
}

func montarLinhaDeServico(servico ServicoFaturado, sequencia int) (*LinhaDaNota, error) {
	if strings.TrimSpace(servico.Descricao) == "" {
		return nil, errorsuc.NewValidationError("informe a descrição do serviço de industrialização")
	}
	quantidade, err := decimal.NewFromString(strings.TrimSpace(servico.Quantidade))
	if err != nil || !quantidade.IsPositive() {
		return nil, errorsuc.NewValidationError("quantidade do serviço inválida")
	}
	valorUnit, err := decimal.NewFromString(strings.TrimSpace(servico.ValorUnit))
	if err != nil || !valorUnit.IsPositive() {
		return nil, errorsuc.NewValidationError("valor do serviço inválido")
	}

	total := quantidade.Mul(valorUnit).Round(2)
	unidade := strings.ToUpper(strings.TrimSpace(servico.Unidade))
	if unidade == "" {
		unidade = "PC"
	}

	// PIS e COFINS sobre o valor do serviço, CST 01. Arredondados a duas casas,
	// como vão na nota.
	valorPIS := total.Mul(AliquotaPIS).Round(2)
	valorCOFINS := total.Mul(AliquotaCOFINS).Round(2)

	return &LinhaDaNota{
		Sequencia: sequencia,
		Descricao: servico.Descricao,
		NCM:       NCMServico,
		CFOP:      CFOPServico,
		CSTICMS:   CSTICMSServico,
		Unidade:   unidade,
		// `CodigoItem` já vinha no pedido e era DESCARTADO: a linha saía sem código
		// de produto, e o autorizador serializava "0" no `cProd`.
		CodigoProduto: strings.TrimSpace(servico.CodigoItem),
		Quantidade:    quantidade,
		ValorUnit:     valorUnit,
		ValorTotal:    total,
		AliqPIS:       AliquotaPIS,
		ValorPIS:      valorPIS,
		AliqCOFINS:    AliquotaCOFINS,
		ValorCOFINS:   valorCOFINS,
		CSTPIS:        CSTPISCOFINS,
		CSTCOFINS:     CSTPISCOFINS,
		Movimento:     "",
	}, nil
}
