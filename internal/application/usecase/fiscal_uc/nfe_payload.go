package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// PlanoDaNota é a condição de pagamento da nota já resolvida em parcelas.
//
// É o mesmo cálculo que a proposta mostra ao cliente no orçamento; aqui ele vira
// duplicata na NF-e e título no contas a receber, para que os três números —
// proposta, nota e cobrança — sejam sempre o mesmo.
type PlanoDaNota struct {
	CondicaoCode      *int64
	CondicaoDescricao string
	Parcelas          []customerentity.ParcelaCalculada
	// Origem diz de onde a condição veio (pedido de venda, cadastro do cliente
	// ou nenhuma), para a prévia poder explicar o que o usuário está vendo.
	Origem string
	// Aviso traz o problema de cadastro que impediu o cálculo, quando houver.
	Aviso string
}

// resolverPlanoDaNota descobre a condição de pagamento que vale para a nota e a
// transforma em parcelas sobre o valor total do documento.
//
// A condição vem do pedido de venda que originou a nota. Quando a nota foi
// digitada solta, cai na condição padrão do cadastro do cliente — é o que o
// vendedor combinaria de novo. Sem nenhuma das duas, a nota é à vista.
func resolverPlanoDaNota(
	ctx context.Context,
	exit *entity.FiscalExit,
	customers customerrepo.CustomerRepository,
	orders salesrepo.SalesOrderRepository,
) PlanoDaNota {
	plano := PlanoDaNota{Origem: "SEM_CONDICAO"}
	if exit == nil {
		return plano
	}

	total := decimal.NewFromFloat(exit.ValorTotal)
	agora := time.Now()
	datas := customerentity.DatasBase{Emissao: exit.DataEmissao, Faturamento: &agora}
	if exit.DataSaida != nil {
		datas.Entrega = exit.DataSaida
	}

	var cond *customerentity.PaymentCondition
	if customers != nil {
		if exit.SalesOrderCode != nil && orders != nil {
			if order, err := orders.GetByCode(ctx, *exit.SalesOrderCode); err == nil && order != nil && order.PaymentTermCode != nil {
				if c, err := customers.GetPaymentConditionByCode(ctx, *order.PaymentTermCode); err == nil && c != nil {
					cond, plano.Origem = c, "PEDIDO_DE_VENDA"
				}
			}
		}
		if cond == nil && exit.CustomerCode != nil {
			if cliente, err := customers.GetCustomerByCode(ctx, *exit.CustomerCode); err == nil && cliente != nil && cliente.PaymentConditionID != nil {
				if c, err := customers.GetPaymentConditionByID(ctx, *cliente.PaymentConditionID); err == nil && c != nil {
					cond, plano.Origem = c, "CADASTRO_DO_CLIENTE"
				}
			}
		}
	}

	if cond == nil {
		// À vista: uma parcela na emissão. É o comportamento que a nota já tinha
		// antes de existir condição de pagamento, então nada muda para quem não
		// usa condição.
		plano.Parcelas = []customerentity.ParcelaCalculada{{
			Numero: 1, Percentual: decimal.NewFromInt(100), Valor: total.Round(2),
			Vencimento: exit.DataEmissao, Evento: customerentity.BaseEmissao, Descricao: "à vista",
		}}
		return plano
	}

	plano.CondicaoCode = &cond.Code
	plano.CondicaoDescricao = cond.Description
	parcelas, err := customerentity.CalcularPlano(cond, total, datas)
	if err != nil {
		// Percentual que não fecha 100 é problema do CADASTRO da condição. A nota
		// não pode inventar a divisão, então cai em parcela única e avisa.
		plano.Aviso = err.Error()
		plano.Parcelas = []customerentity.ParcelaCalculada{{
			Numero: 1, Percentual: decimal.NewFromInt(100), Valor: total.Round(2),
			Vencimento: exit.DataEmissao, Evento: customerentity.BaseEmissao, Descricao: "à vista",
		}}
		return plano
	}
	plano.Parcelas = parcelas
	return plano
}

// formaPagamentoNFe traduz a parcela para o código da SEFAZ (tabela tPag).
//
// O código antigo mandava sempre "01" (dinheiro) com o valor inteiro da nota:
// uma venda em 28/56/84 dias saía da SEFAZ declarada como paga em dinheiro no
// ato, o que é informação fiscal errada.
func formaPagamentoNFe(p customerentity.ParcelaCalculada) string {
	if p.DocumentType != nil {
		switch strings.ToUpper(strings.TrimSpace(*p.DocumentType)) {
		case "PIX":
			return "17"
		case "BOLETO", "DUPLICATA", "DUP":
			return "15"
		case "CARTAO_CREDITO", "CARTAO":
			return "03"
		case "CARTAO_DEBITO":
			return "04"
		case "CHEQUE":
			return "02"
		case "DINHEIRO", "ESPECIE":
			return "01"
		case "TRANSFERENCIA", "TED", "DOC":
			return "16"
		}
	}
	if !parcelaAPrazo(p) {
		return "01"
	}
	return "15"
}

// parcelaAPrazo diz se a parcela é cobrança futura, e por isso vira duplicata.
//
// "30% de entrada" é paga no ato: sem prazo e no próprio evento. O que tem dias
// de prazo, ou conta da ENTREGA/FATURAMENTO, é cobrado depois.
func parcelaAPrazo(p customerentity.ParcelaCalculada) bool {
	if p.DiasPrazo > 0 {
		return true
	}
	return p.Evento == customerentity.BaseEntrega || p.Evento == customerentity.BaseFaturamento
}

// montarPayloadNFe constrói o documento que vai para a SEFAZ.
//
// Prévia e autorização chamam esta MESMA função: é a única forma de garantir que
// o que o time confere na tela é exatamente o que será transmitido. Enquanto a
// prévia montava o seu próprio resumo, ela podia mostrar um número e a nota sair
// com outro.
func montarPayloadNFe(
	exit *entity.FiscalExit,
	items []*entity.FiscalExitItem,
	cfg *entity.FiscalConfig,
	plano PlanoDaNota,
) focusnfe.NFEPayload {
	ufDest := "PR"
	if exit.UFDestinatario != nil && strings.TrimSpace(*exit.UFDestinatario) != "" {
		ufDest = *exit.UFDestinatario
	}
	razaoDest := "Destinatário"
	if exit.RazaoSocialDestinatario != nil && strings.TrimSpace(*exit.RazaoSocialDestinatario) != "" {
		razaoDest = *exit.RazaoSocialDestinatario
	}
	cnpjDest := ""
	if exit.CnpjDestinatario != nil {
		cnpjDest = *exit.CnpjDestinatario
	}

	localDestino := 1
	if ufDest != cfg.UFEmpresa {
		localDestino = 2
	}
	consumidorFinal, indicadorIE := 0, 1
	if exit.IEDestinatario == nil || *exit.IEDestinatario == "" || *exit.IEDestinatario == "ISENTO" {
		consumidorFinal, indicadorIE = 1, 9
	}

	formas := make([]focusnfe.NFEFormaPagamento, 0, len(plano.Parcelas))
	duplicatas := make([]focusnfe.NFEDuplicata, 0, len(plano.Parcelas))
	aPrazo := false
	for _, p := range plano.Parcelas {
		valor, _ := p.Valor.Float64()
		formas = append(formas, focusnfe.NFEFormaPagamento{FormaPagamento: formaPagamentoNFe(p), Valor: valor})
		if parcelaAPrazo(p) {
			aPrazo = true
			duplicatas = append(duplicatas, focusnfe.NFEDuplicata{
				Numero:         fmt.Sprintf("%d/%d-%d", exit.NumeroNF, len(plano.Parcelas), p.Numero),
				DataVencimento: p.Vencimento.Format("2006-01-02"),
				Valor:          valor,
			})
		}
	}
	if len(formas) == 0 {
		formas = append(formas, focusnfe.NFEFormaPagamento{FormaPagamento: "01", Valor: exit.ValorTotal})
	}
	// Duplicata só existe em venda a prazo; à vista o grupo não vai.
	if !aPrazo {
		duplicatas = nil
	}

	return focusnfe.NFEPayload{
		NaturezaOperacao:  exit.NaturezaOperacao,
		DataEmissao:       exit.DataEmissao.Format("2006-01-02T15:04:05-03:00"),
		TipoDocumento:     1,
		LocalDestino:      localDestino,
		FinalidadeEmissao: 1,
		ConsumidorFinal:   consumidorFinal,
		PresencaComprador: 4,
		Emitente: focusnfe.NFEEmitente{
			CNPJ:             cfg.CnpjEmpresa,
			Nome:             cfg.RazaoSocial,
			Logradouro:       cfg.Logradouro,
			Numero:           cfg.Numero,
			Bairro:           cfg.Bairro,
			Municipio:        cfg.Municipio,
			UF:               cfg.UFEmpresa,
			CEP:              cfg.CEP,
			Telefone:         derefStr(cfg.Telefone),
			RegimeTributario: 3,
		},
		Destinatario: focusnfe.NFEDestinatario{
			CNPJCPF:     cnpjDest,
			Nome:        razaoDest,
			Logradouro:  derefStr(exit.DestLogradouro),
			Numero:      derefStr(exit.DestNumero),
			Bairro:      derefStr(exit.DestBairro),
			Municipio:   derefStr(exit.DestMunicipio),
			UF:          ufDest,
			CEP:         derefStr(exit.DestCEP),
			Email:       derefStr(exit.DestEmail),
			IndicadorIE: indicadorIE,
			IE:          exit.IEDestinatario,
		},
		Items:          buildFocusItems(items, cfg),
		FormaPagamento: formas,
		Duplicatas:     duplicatas,
	}
}
