// Package contabilizacao monta os lançamentos contábeis automáticos do ciclo
// financeiro e de vendas: pagamento e recebimento de títulos (inclusive o
// recolhimento de impostos retidos) e a NF-e de saída (receita, impostos sobre
// vendas e CMV). São regras puras: quem chama resolve as contas e grava.
//
// O livro do sistema é de partidas simples (um débito e um crédito por linha);
// cada função devolve pares que, somados, fecham débito = crédito.
package contabilizacao

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Lancamento é uma partida: débito numa conta, crédito noutra.
type Lancamento struct {
	DebitoID  int64
	CreditoID int64
	DebitoCC  *int64
	CreditoCC *int64
	Valor     decimal.Decimal
	Historico string
}

// Parte é um pedaço do principal do título: a conta (e o centro de custo) e o
// peso. Título vindo de nota já contabilizada tem uma parte só (Fornecedores
// ou Clientes); título avulso é distribuído pelo rateio do plano de contas.
type Parte struct {
	Conta *int64
	CC    *int64
	Peso  decimal.Decimal
	Nome  string
}

// Baixa é o pagamento (ou recebimento) de um título.
type Baixa struct {
	Banco         *int64
	Partes        []Parte
	Principal     decimal.Decimal // valor do título quitado em dinheiro (sem juros)
	Juros         decimal.Decimal // juros + multa
	Desconto      decimal.Decimal
	ContaJuros    *int64 // juros pagos (pagar) / juros recebidos (receber)
	ContaDesconto *int64 // descontos obtidos (pagar) / concedidos (receber)
	Historico     string
}

func pos(v decimal.Decimal) bool { return v.IsPositive() }

// dividir reparte o valor pelos pesos; a última parte leva os centavos.
func dividir(valor decimal.Decimal, partes []Parte) []decimal.Decimal {
	out := make([]decimal.Decimal, len(partes))
	soma := decimal.Zero
	for _, p := range partes {
		soma = soma.Add(p.Peso)
	}
	acumulado := decimal.Zero
	for i, p := range partes {
		if i == len(partes)-1 || !soma.IsPositive() {
			out[i] = valor.Sub(acumulado)
			if !soma.IsPositive() && i < len(partes)-1 {
				out[i] = decimal.Zero
			}
			continue
		}
		out[i] = valor.Mul(p.Peso).Div(soma).Round(2)
		acumulado = acumulado.Add(out[i])
	}
	return out
}

func faltaParte(partes []Parte) []string {
	var f []string
	if len(partes) == 0 {
		f = append(f, "o título não tem plano de contas para a contrapartida")
	}
	for _, p := range partes {
		if p.Conta == nil {
			nome := p.Nome
			if nome == "" {
				nome = "a contrapartida"
			}
			f = append(f, fmt.Sprintf("%s não tem conta contábil", nome))
		}
	}
	return f
}

// Pagamento: débito na contrapartida (Fornecedores, imposto a recolher ou a
// despesa do plano) pelo principal + desconto, débito em juros pagos pelos
// juros e multa, crédito no banco pelo que saiu do caixa (principal + juros) e
// crédito em descontos obtidos pelo desconto.
func Pagamento(b Baixa) ([]Lancamento, []string) {
	var faltando []string
	if b.Banco == nil {
		faltando = append(faltando, "a conta bancária não tem conta contábil (e não há conta de banco padrão)")
	}
	faltando = append(faltando, faltaParte(b.Partes)...)
	if pos(b.Juros) && b.ContaJuros == nil {
		faltando = append(faltando, "falta a conta de juros pagos")
	}
	if pos(b.Desconto) && b.ContaDesconto == nil {
		faltando = append(faltando, "falta a conta de descontos obtidos")
	}
	if len(faltando) > 0 {
		return nil, faltando
	}
	var out []Lancamento
	principal := dividir(b.Principal, b.Partes)
	desconto := dividir(b.Desconto, b.Partes)
	for i, p := range b.Partes {
		if pos(principal[i]) {
			out = append(out, Lancamento{DebitoID: *p.Conta, DebitoCC: p.CC, CreditoID: *b.Banco, Valor: principal[i], Historico: b.Historico})
		}
		if pos(desconto[i]) {
			out = append(out, Lancamento{DebitoID: *p.Conta, DebitoCC: p.CC, CreditoID: *b.ContaDesconto, Valor: desconto[i], Historico: b.Historico + " — desconto obtido"})
		}
	}
	if pos(b.Juros) {
		out = append(out, Lancamento{DebitoID: *b.ContaJuros, CreditoID: *b.Banco, Valor: b.Juros.Round(2), Historico: b.Historico + " — juros e multa"})
	}
	return out, nil
}

// Recebimento: débito no banco pelo que entrou (principal + juros), débito em
// descontos concedidos pelo desconto, crédito na contrapartida (Clientes ou a
// receita do plano) pelo principal + desconto e crédito em juros recebidos.
func Recebimento(b Baixa) ([]Lancamento, []string) {
	var faltando []string
	if b.Banco == nil {
		faltando = append(faltando, "a conta bancária não tem conta contábil (e não há conta de banco padrão)")
	}
	faltando = append(faltando, faltaParte(b.Partes)...)
	if pos(b.Juros) && b.ContaJuros == nil {
		faltando = append(faltando, "falta a conta de juros recebidos")
	}
	if pos(b.Desconto) && b.ContaDesconto == nil {
		faltando = append(faltando, "falta a conta de descontos concedidos")
	}
	if len(faltando) > 0 {
		return nil, faltando
	}
	var out []Lancamento
	principal := dividir(b.Principal, b.Partes)
	desconto := dividir(b.Desconto, b.Partes)
	for i, p := range b.Partes {
		if pos(principal[i]) {
			out = append(out, Lancamento{DebitoID: *b.Banco, CreditoID: *p.Conta, CreditoCC: p.CC, Valor: principal[i], Historico: b.Historico})
		}
		if pos(desconto[i]) {
			out = append(out, Lancamento{DebitoID: *b.ContaDesconto, CreditoID: *p.Conta, CreditoCC: p.CC, Valor: desconto[i], Historico: b.Historico + " — desconto concedido"})
		}
	}
	if pos(b.Juros) {
		out = append(out, Lancamento{DebitoID: *b.Banco, CreditoID: *b.ContaJuros, Valor: b.Juros.Round(2), Historico: b.Historico + " — juros e multa"})
	}
	return out, nil
}

// ContasSaida são as contas da NF-e de saída.
type ContasSaida struct {
	Clientes, Receita                               *int64
	IPIRecolher, ICMSSTRecolher                     *int64
	ICMSVendas, ICMSRecolher                        *int64
	PISVendas, PISRecolher, COFINSVendas, COFINSRec *int64
	CMV, Estoque                                    *int64
}

// Saida é a NF-e de saída autorizada.
type Saida struct {
	Total, IPI, ICMSST     decimal.Decimal
	ICMS, PIS, COFINS, CMV decimal.Decimal
	// GeraReceita: venda (gera contas a receber). Remessa, bonificação e
	// devolução não têm receita, só os impostos destacados e a baixa do custo.
	GeraReceita bool
	Historico   string
}

// NotaSaida: D Clientes × C Receita (total − IPI − ICMS-ST), C IPI e ICMS-ST a
// recolher (o cliente paga, a empresa repassa); D ICMS/PIS/COFINS sobre vendas
// × C a recolher; D CMV × C Estoque pelo custo médio que saiu.
func NotaSaida(s Saida, c ContasSaida) ([]Lancamento, []string) {
	var out []Lancamento
	var faltando []string
	par := func(deb, cred *int64, valor decimal.Decimal, hist, nomeDeb, nomeCred string) {
		if !pos(valor) {
			return
		}
		if deb == nil {
			faltando = append(faltando, "falta a conta de "+nomeDeb)
		}
		if cred == nil {
			faltando = append(faltando, "falta a conta de "+nomeCred)
		}
		if deb != nil && cred != nil {
			out = append(out, Lancamento{DebitoID: *deb, CreditoID: *cred, Valor: valor.Round(2), Historico: s.Historico + " — " + hist})
		}
	}
	if s.GeraReceita {
		receita := s.Total.Sub(s.IPI).Sub(s.ICMSST)
		par(c.Clientes, c.Receita, receita, "receita de vendas", "clientes", "receita de vendas")
		par(c.Clientes, c.IPIRecolher, s.IPI, "IPI destacado", "clientes", "IPI a recolher")
		par(c.Clientes, c.ICMSSTRecolher, s.ICMSST, "ICMS-ST retido", "clientes", "ICMS-ST a recolher")
	}
	par(c.ICMSVendas, c.ICMSRecolher, s.ICMS, "ICMS sobre vendas", "ICMS sobre vendas", "ICMS a recolher")
	par(c.PISVendas, c.PISRecolher, s.PIS, "PIS sobre vendas", "PIS sobre vendas", "PIS a recolher")
	par(c.COFINSVendas, c.COFINSRec, s.COFINS, "COFINS sobre vendas", "COFINS sobre vendas", "COFINS a recolher")
	par(c.CMV, c.Estoque, s.CMV, "custo da mercadoria vendida", "CMV", "estoque")
	return out, unicos(faltando)
}

func unicos(xs []string) []string {
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

// Estorno inverte débito e crédito de cada lançamento.
func Estorno(ls []Lancamento, historico string) []Lancamento {
	out := make([]Lancamento, len(ls))
	for i, l := range ls {
		out[i] = Lancamento{DebitoID: l.CreditoID, DebitoCC: l.CreditoCC, CreditoID: l.DebitoID, CreditoCC: l.DebitoCC, Valor: l.Valor, Historico: historico + " — " + l.Historico}
	}
	return out
}

// Saldo devolve débitos − créditos por conta (para testes e conferência).
func Saldo(ls []Lancamento) map[int64]decimal.Decimal {
	m := map[int64]decimal.Decimal{}
	for _, l := range ls {
		m[l.DebitoID] = m[l.DebitoID].Add(l.Valor)
		m[l.CreditoID] = m[l.CreditoID].Sub(l.Valor)
	}
	return m
}

// ContaRetencao escolhe a conta "a recolher" do imposto retido.
func ContaRetencao(tipo string, irrf, pcc, inss, iss *int64) *int64 {
	switch tipo {
	case "IRRF":
		return irrf
	case "PIS", "COFINS", "CSLL", "PCC":
		return pcc
	case "INSS":
		return inss
	case "ISS":
		return iss
	}
	return nil
}

// Lote é o conjunto de lançamentos de um fato (um pagamento, uma nota) com a
// origem que permite achá-lo e estorná-lo depois.
type Lote struct {
	PlanID      int64
	Data        time.Time
	Prefixo     string // número do lançamento: Prefixo + id + "-" + n
	SourceType  string
	SourceID    int64
	Lancamentos []Lancamento
}

// Origens dos lançamentos automáticos.
const (
	OrigemPagamento    = "PAGAMENTO_CP"
	OrigemRecebimento  = "RECEBIMENTO_CR"
	OrigemSaida        = "NFE_SAIDA"
	OrigemSaidaEstorno = "NFE_SAIDA_ESTORNO"
	OrigemEntrada      = "NFE_ENTRADA"
)

// LinhaDevolucao é o custo que saiu do estoque por um item devolvido.
type LinhaDevolucao struct {
	Conta *int64 // conta do plano de contas do item (estoque/despesa)
	CC    *int64
	Custo decimal.Decimal
	Nome  string
}

// CreditoEstornado é o crédito de imposto da entrada que volta com a devolução.
type CreditoEstornado struct {
	Conta *int64 // imposto a recuperar (nil: o crédito tinha ido ao custo)
	Valor decimal.Decimal
	Nome  string
}

// Devolucao é a NF-e de devolução de compra autorizada.
type Devolucao struct {
	Total        decimal.Decimal // o que o fornecedor deve de volta
	Linhas       []LinhaDevolucao
	Creditos     []CreditoEstornado
	Fornecedores *int64
	Variacao     *int64 // diferença entre o custo médio e o da compra
	Historico    string
}

// NotaDevolucao: D Fornecedores pelo total; C estoque pelo custo médio que
// saiu; C impostos a recuperar pelos créditos estornados; a diferença entre o
// custo da compra e o custo médio de hoje vai para a conta de variação.
func NotaDevolucao(d Devolucao) ([]Lancamento, []string) {
	var faltando []string
	if d.Fornecedores == nil {
		faltando = append(faltando, "falta a conta de fornecedores")
	}
	var out []Lancamento
	soma := decimal.Zero
	for _, l := range d.Linhas {
		if !pos(l.Custo) {
			continue
		}
		if l.Conta == nil {
			faltando = append(faltando, fmt.Sprintf("%s não tem conta contábil (plano de contas ou despesa padrão)", l.Nome))
			continue
		}
		if d.Fornecedores != nil {
			out = append(out, Lancamento{DebitoID: *d.Fornecedores, CreditoID: *l.Conta, CreditoCC: l.CC, Valor: l.Custo.Round(2), Historico: d.Historico + " — " + l.Nome})
		}
		soma = soma.Add(l.Custo.Round(2))
	}
	for _, c := range d.Creditos {
		if !pos(c.Valor) || c.Conta == nil {
			continue // sem conta: o crédito estava no custo e entra na variação
		}
		if d.Fornecedores != nil {
			out = append(out, Lancamento{DebitoID: *d.Fornecedores, CreditoID: *c.Conta, Valor: c.Valor.Round(2), Historico: d.Historico + " — estorno do crédito de " + c.Nome})
		}
		soma = soma.Add(c.Valor.Round(2))
	}
	dif := d.Total.Round(2).Sub(soma)
	if !dif.IsZero() {
		if d.Variacao == nil {
			faltando = append(faltando, "falta a conta de despesa padrão para a diferença entre o custo da compra e o custo médio")
		} else if d.Fornecedores != nil {
			if dif.IsPositive() {
				out = append(out, Lancamento{DebitoID: *d.Fornecedores, CreditoID: *d.Variacao, Valor: dif, Historico: d.Historico + " — diferença de custo"})
			} else {
				out = append(out, Lancamento{DebitoID: *d.Variacao, CreditoID: *d.Fornecedores, Valor: dif.Neg(), Historico: d.Historico + " — diferença de custo"})
			}
		}
	}
	if len(faltando) > 0 {
		return nil, unicos(faltando)
	}
	return out, nil
}
