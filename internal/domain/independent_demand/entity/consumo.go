package entity

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// Consumo pela venda (SAP, estratégia 40): a previsão e a demanda de estoque
// são venda esperada; quando o pedido chega, ele toma o lugar delas em vez de
// somar. Planeja-se o maior dos dois, nunca a soma.

// Consumivel é uma necessidade que a venda pode consumir: previsão ou demanda
// de expectativa de venda. Chave agrupa item e máscara; Item escolhe a
// política de consumo (janela) do item.
type Consumivel struct {
	Chave      string
	Item       int64
	Data       time.Time
	Quantidade decimal.Decimal
}

// Venda é a quantidade vendida de uma linha de pedido (a atendida também
// conta: o que já saiu consumiu a expectativa do mesmo jeito), sem a
// cancelada.
type Venda struct {
	Chave      string
	Data       time.Time
	Quantidade decimal.Decimal
	// Linha é o código da linha do pedido (o cenário altera pedido por ela).
	Linha *int64
	// Para o rastreio do consumo (preenchidos pela leitura): número do
	// pedido, sequência da linha e cliente.
	Pedido    int64
	Sequencia int
	Cliente   string
}

// Consumo é a política de consumo de um item: a janela (dias antes e depois da
// venda) e se o item é planejado sob encomenda — aí só o pedido gera
// necessidade, e a previsão e a demanda de expectativa de venda não entram.
type Consumo struct {
	Antes, Depois int
	SobEncomenda  bool
}

// PoliticaDeConsumo devolve a política de consumo do item.
type PoliticaDeConsumo func(item int64) Consumo

// JanelaFixa é a mesma janela para todos os itens, sem sob encomenda.
func JanelaFixa(antes, depois int) PoliticaDeConsumo {
	return func(int64) Consumo { return Consumo{Antes: antes, Depois: depois} }
}

// Alocacao é quanto da venda v consumiu do consumível c.
type Alocacao struct {
	Consumivel int
	Venda      int
	Quantidade decimal.Decimal
}

// ResultadoConsumo: o consumido de cada consumível, as alocações venda →
// consumível (o rastreio) e o que sobrou de cada venda (venda acima da
// expectativa).
type ResultadoConsumo struct {
	Consumido  []decimal.Decimal
	Alocacoes  []Alocacao
	SobraVenda []decimal.Decimal
}

// Consumir devolve, para cada consumível, quanto as vendas consumiram dele.
// Mesma janela para todos os itens; ver ConsumirComPolitica.
func Consumir(cons []Consumivel, vendas []Venda, diasAntes, diasDepois int) []decimal.Decimal {
	return ConsumirComPolitica(cons, vendas, JanelaFixa(diasAntes, diasDepois)).Consumido
}

// ConsumirComPolitica: cada venda, em ordem de data, consome primeiro para
// trás — do consumível mais próximo para o mais antigo, até os dias antes do
// item — e depois para a frente, até os dias depois. É a ordem do consumo do
// SAP (modo 1 com período para a frente). A janela é a do item da chave.
func ConsumirComPolitica(cons []Consumivel, vendas []Venda, pol PoliticaDeConsumo) ResultadoConsumo {
	res := ResultadoConsumo{
		Consumido:  make([]decimal.Decimal, len(cons)),
		SobraVenda: make([]decimal.Decimal, len(vendas)),
	}
	saldo := make([]decimal.Decimal, len(cons))
	porChave := map[string][]int{}
	itemDaChave := map[string]int64{}
	for i, c := range cons {
		saldo[i] = decimal.Max(decimal.Zero, c.Quantidade)
		porChave[c.Chave] = append(porChave[c.Chave], i)
		if _, ok := itemDaChave[c.Chave]; !ok {
			itemDaChave[c.Chave] = c.Item
		}
	}
	// Na mesma data, o consumível que veio antes na lista é consumido primeiro
	// (a demanda antes da previsão), nos dois sentidos.
	paraTras := map[string][]int{}
	for chave, idx := range porChave {
		sort.SliceStable(idx, func(a, b int) bool { return cons[idx[a]].Data.Before(cons[idx[b]].Data) })
		tras := append([]int(nil), idx...)
		sort.SliceStable(tras, func(a, b int) bool { return cons[tras[a]].Data.After(cons[tras[b]].Data) })
		paraTras[chave] = tras
	}
	ordem := make([]int, len(vendas))
	for i := range vendas {
		ordem[i] = i
	}
	sort.SliceStable(ordem, func(a, b int) bool { return vendas[ordem[a]].Data.Before(vendas[ordem[b]].Data) })

	tomar := func(c, v int, resta decimal.Decimal) decimal.Decimal {
		q := decimal.Min(resta, saldo[c])
		if !q.IsPositive() {
			return resta
		}
		saldo[c] = saldo[c].Sub(q)
		res.Consumido[c] = res.Consumido[c].Add(q)
		res.Alocacoes = append(res.Alocacoes, Alocacao{Consumivel: c, Venda: v, Quantidade: q})
		return resta.Sub(q)
	}

	for _, vi := range ordem {
		v := vendas[vi]
		resta := decimal.Max(decimal.Zero, v.Quantidade)
		item, conhecida := itemDaChave[v.Chave]
		if !conhecida || !resta.IsPositive() {
			res.SobraVenda[vi] = resta
			continue
		}
		janela := pol(item)
		if janela.SobEncomenda {
			res.SobraVenda[vi] = resta
			continue
		}
		inicio := dia(v.Data).AddDate(0, 0, -janela.Antes)
		fim := dia(v.Data).AddDate(0, 0, janela.Depois)
		// Para trás: do mais próximo da venda para o mais antigo.
		for _, i := range paraTras[v.Chave] {
			if !resta.IsPositive() {
				break
			}
			d := dia(cons[i].Data)
			if d.After(dia(v.Data)) || d.Before(inicio) {
				continue
			}
			resta = tomar(i, vi, resta)
		}
		// Para a frente: do mais próximo para o mais distante.
		for _, i := range porChave[v.Chave] {
			if !resta.IsPositive() {
				break
			}
			d := dia(cons[i].Data)
			if !d.After(dia(v.Data)) || d.After(fim) {
				continue
			}
			resta = tomar(i, vi, resta)
		}
		res.SobraVenda[vi] = resta
	}
	return res
}

// Vencida: nenhuma venda futura pode mais consumir a demanda — a mais tardia
// que a alcança é a de data + diasAntes. Sai do cálculo.
func Vencida(data, hoje time.Time, diasAntes int) bool {
	return dia(data).AddDate(0, 0, diasAntes).Before(dia(hoje))
}

func dia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ChaveItem agrupa item e máscara para o consumo.
func ChaveItem(item int64, mask string) string {
	return fmt.Sprintf("%d|%s", item, mask)
}

// Situação calculada da demanda aberta.
const (
	PosicaoAberta    = "ABERTA"
	PosicaoParcial   = "PARCIAL"
	PosicaoAtendida  = "ATENDIDA"
	PosicaoConsumida = "CONSUMIDA"
	PosicaoVencida   = "VENCIDA"
	// PosicaoSobEncomenda: expectativa de venda de item planejado sob
	// encomenda — só o pedido gera necessidade; a demanda não entra no MRP.
	PosicaoSobEncomenda = "SOB_ENCOMENDA"
)

// ConsumoDaVenda é uma linha de pedido que consumiu parte de uma demanda ou
// previsão (o rastreio).
type ConsumoDaVenda struct {
	Venda      Venda
	Quantidade decimal.Decimal
}

// Posicao é o que a demanda significa hoje para o planejamento.
type Posicao struct {
	Consumida decimal.Decimal
	// APlanejar é o que entra no MRP: o saldo não consumido (expectativa de
	// venda que não venceu) ou o saldo não atendido (finalidade própria).
	APlanejar decimal.Decimal
	Vencida   bool
	// VenceEm: a última data em que uma venda ainda consome a demanda.
	VenceEm  *time.Time
	Situacao string
	// Consumo: as vendas que consumiram a demanda.
	Consumo []ConsumoDaVenda
}

// PosicaoPrevisao é a previsão depois do consumo: o que sobra para o MRP, o
// consumido e por quais vendas.
type PosicaoPrevisao struct {
	Resto        decimal.Decimal
	Consumida    decimal.Decimal
	Consumo      []ConsumoDaVenda
	SobEncomenda bool
}

// Posicionar calcula a posição com a mesma janela para todos os itens e
// devolve o que sobra de cada previsão. Ver PosicionarComPolitica.
func Posicionar(ds []*IndependentDemand, previsoes []Consumivel, vendas []Venda, antes, depois int, hoje time.Time) ([]Posicao, []decimal.Decimal) {
	pos, prev, _ := PosicionarComPolitica(ds, previsoes, vendas, JanelaFixa(antes, depois), hoje)
	resto := make([]decimal.Decimal, len(prev))
	for i, p := range prev {
		resto[i] = p.Resto
	}
	return pos, resto
}

// PosicionarComPolitica calcula a posição de cada demanda e de cada previsão
// depois do consumo pelas vendas, com a política de consumo de cada item.
// Demandas de expectativa de venda e previsões são consumidas juntas, na
// ordem de data (a demanda antes da previsão do mesmo dia); as de finalidade
// própria seguram o saldo não atendido. Item sob encomenda: a expectativa de
// venda e a previsão não entram no cálculo. Devolve também a sobra de cada
// venda (o que ela vendeu além da expectativa).
func PosicionarComPolitica(ds []*IndependentDemand, previsoes []Consumivel, vendas []Venda, pol PoliticaDeConsumo, hoje time.Time) ([]Posicao, []PosicaoPrevisao, []decimal.Decimal) {
	pos := make([]Posicao, len(ds))
	var cons []Consumivel
	var deQuem []int // índice da demanda; -1 - i para a previsão i
	for i, d := range ds {
		if d.Situacao != SituacaoAberta || !ConsumidaPelaVenda(d.Motivo) || pol(d.ItemCode).SobEncomenda {
			continue
		}
		mask := ""
		if d.Mask != nil {
			mask = *d.Mask
		}
		cons = append(cons, Consumivel{Chave: ChaveItem(d.ItemCode, mask), Item: d.ItemCode, Data: d.DemandDate, Quantidade: d.Quantity})
		deQuem = append(deQuem, i)
	}
	prev := make([]PosicaoPrevisao, len(previsoes))
	for i, p := range previsoes {
		if pol(p.Item).SobEncomenda {
			prev[i] = PosicaoPrevisao{Resto: decimal.Zero, SobEncomenda: true}
			continue
		}
		prev[i].Resto = decimal.Max(decimal.Zero, p.Quantidade)
		cons = append(cons, p)
		deQuem = append(deQuem, -1-i)
	}
	res := ConsumirComPolitica(cons, vendas, pol)
	for _, a := range res.Alocacoes {
		consumo := ConsumoDaVenda{Venda: vendas[a.Venda], Quantidade: a.Quantidade}
		if quem := deQuem[a.Consumivel]; quem < 0 {
			prev[-1-quem].Consumo = append(prev[-1-quem].Consumo, consumo)
		} else {
			pos[quem].Consumo = append(pos[quem].Consumo, consumo)
		}
	}
	for k, quem := range deQuem {
		if quem < 0 {
			i := -1 - quem
			prev[i].Consumida = res.Consumido[k]
			prev[i].Resto = decimal.Max(decimal.Zero, previsoes[i].Quantidade.Sub(res.Consumido[k]))
			continue
		}
		pos[quem].Consumida = res.Consumido[k]
	}

	for i, d := range ds {
		p := &pos[i]
		if d.Situacao != SituacaoAberta {
			p.Situacao = d.Situacao
			p.APlanejar = decimal.Zero
			continue
		}
		if !ConsumidaPelaVenda(d.Motivo) {
			p.APlanejar = d.Saldo()
			switch {
			case !p.APlanejar.IsPositive():
				p.Situacao = PosicaoAtendida
			case d.Atendida.IsPositive():
				p.Situacao = PosicaoParcial
			default:
				p.Situacao = PosicaoAberta
			}
			continue
		}
		janela := pol(d.ItemCode)
		if janela.SobEncomenda {
			p.APlanejar = decimal.Zero
			p.Situacao = PosicaoSobEncomenda
			continue
		}
		vence := dia(d.DemandDate).AddDate(0, 0, janela.Antes)
		p.VenceEm = &vence
		resto := decimal.Max(decimal.Zero, d.Quantity.Sub(p.Consumida))
		switch {
		case !resto.IsPositive():
			p.Situacao = PosicaoConsumida
		case Vencida(d.DemandDate, hoje, janela.Antes):
			p.Vencida = true
			p.Situacao = PosicaoVencida
		case p.Consumida.IsPositive():
			p.APlanejar = resto
			p.Situacao = PosicaoParcial
		default:
			p.APlanejar = resto
			p.Situacao = PosicaoAberta
		}
	}
	return pos, prev, res.SobraVenda
}
