package entity

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// De onde vem a reposição de um almoxarifado.
const (
	OrigemCompra        = "COMPRA"
	OrigemTransferencia = "TRANSFERENCIA"
	OrigemProducao      = "PRODUCAO"
)

// Tipos do item (items.engineering_type) que a reposição distingue.
const (
	itemFabricado  = 0
	itemComprado   = 1
	itemDeTerceiro = 2
	itemServico    = 3
)

// PoliticaReposicao é o mínimo e o máximo do item em um almoxarifado. Campo nil
// herda o valor do cadastro do item.
type PoliticaReposicao struct {
	ID          int64            `json:"id"`
	ItemCode    int64            `json:"item_code"`
	Mask        string           `json:"mask"`
	WarehouseID int64            `json:"warehouse_id"`
	Minimo      *decimal.Decimal `json:"minimum_qty,omitempty"`
	Maximo      *decimal.Decimal `json:"maximum_qty,omitempty"`
	Seguranca   *decimal.Decimal `json:"safety_qty,omitempty"`
	Origem      string           `json:"source_type"`
	AlmoxOrigem *int64           `json:"source_warehouse_id,omitempty"`
	Ativa       bool             `json:"is_active"`
	CreatedBy   uuid.UUID        `json:"-"`
}

// Validar confere a política sozinha. Item e almoxarifado são conferidos ao gravar.
func (p PoliticaReposicao) Validar() error {
	if p.ItemCode <= 0 {
		return domainerr.NewValidation("escolha o item da política")
	}
	if len(p.Mask) > 200 {
		return domainerr.NewValidation("a máscara passa de 200 caracteres")
	}
	if p.WarehouseID <= 0 {
		return domainerr.NewValidation("escolha o almoxarifado da política")
	}
	for _, v := range []*decimal.Decimal{p.Minimo, p.Maximo, p.Seguranca} {
		if v != nil && v.IsNegative() {
			return domainerr.NewValidation("mínimo, máximo e segurança não podem ser negativos")
		}
	}
	if p.Minimo != nil && p.Maximo != nil && p.Maximo.IsPositive() && p.Maximo.LessThan(*p.Minimo) {
		return domainerr.NewValidation("o máximo não pode ser menor que o mínimo")
	}
	switch p.Origem {
	case OrigemCompra, OrigemProducao:
		if p.AlmoxOrigem != nil {
			return domainerr.NewValidation("almoxarifado de origem só vale para reposição por transferência")
		}
	case OrigemTransferencia:
		if p.AlmoxOrigem == nil || *p.AlmoxOrigem <= 0 {
			return domainerr.NewValidation("informe de qual almoxarifado vem a transferência")
		}
		if *p.AlmoxOrigem == p.WarehouseID {
			return domainerr.NewValidation("a origem da transferência não pode ser o próprio almoxarifado")
		}
	default:
		return domainerr.NewValidation("escolha a origem da reposição: compra, transferência ou produção")
	}
	return nil
}

// OrigemAceitaParaItem recusa o que o tipo do item não permite: item de terceiro
// e serviço não têm reposição, e item comprado não se produz.
func OrigemAceitaParaItem(origem string, tipoItem int) error {
	switch tipoItem {
	case itemDeTerceiro, itemServico:
		return domainerr.NewValidation("item de terceiro e serviço não têm reposição de estoque")
	case itemComprado:
		if origem == OrigemProducao {
			return domainerr.NewValidation("item comprado não pode ser reposto por produção")
		}
	}
	return nil
}

// origemPadrao é a origem do item sem política. Vazia: o item não se repõe.
func origemPadrao(tipoItem int) string {
	switch tipoItem {
	case itemComprado:
		return OrigemCompra
	case itemFabricado:
		return OrigemProducao
	}
	return ""
}

// SaldoReposicao é a situação do item em um almoxarifado, ou no conjunto deles.
type SaldoReposicao struct {
	Fisico      decimal.Decimal
	Reservado   decimal.Decimal
	Retido      decimal.Decimal // lote bloqueado ou vencido: está lá, mas não sai
	Requisitado decimal.Decimal // requisição aberta além do que conseguiu reservar
	Chegando    decimal.Decimal // pedido, ordem e transferência com este destino
}

// Somar junta o saldo de outra máscara do mesmo item no almoxarifado.
func (s SaldoReposicao) Somar(o SaldoReposicao) SaldoReposicao {
	s.Fisico = s.Fisico.Add(o.Fisico)
	s.Reservado = s.Reservado.Add(o.Reservado)
	s.Retido = s.Retido.Add(o.Retido)
	s.Requisitado = s.Requisitado.Add(o.Requisitado)
	s.Chegando = s.Chegando.Add(o.Chegando)
	return s
}

// Livre é o que dá para usar agora.
func (s SaldoReposicao) Livre() decimal.Decimal {
	return s.Fisico.Sub(s.Reservado).Sub(s.Retido).Sub(s.Requisitado)
}

// ItemReposicao reúne o que o cálculo precisa de um item.
type ItemReposicao struct {
	ItemCode int64
	// Mask: vazia é o item com todas as máscaras sem política própria.
	Mask         string
	Codigo       string
	Nome         string
	Unidade      string
	TipoItem     int
	Ponto        decimal.Decimal // ponto de pedido do cadastro, já com a segurança
	Motivo       string
	Maximo       decimal.Decimal
	Seguranca    decimal.Decimal
	LoteMinimo   decimal.Decimal
	LoteMultiplo decimal.Decimal
	ConsumoMedio decimal.Decimal
	Politicas    []PoliticaReposicao
	// Total soma os almoxarifados que contam para promessa, para o item sem política.
	Total      SaldoReposicao
	PorAlmox   map[int64]SaldoReposicao
	SemDestino decimal.Decimal // pedido e requisição de compra sem almoxarifado
}

// DaMascara devolve o item só com a máscara: herda o cadastro e começa sem saldo.
func (it ItemReposicao) DaMascara(mask string) *ItemReposicao {
	c := it
	c.Mask, c.Politicas, c.Total, c.SemDestino = mask, nil, SaldoReposicao{}, decimal.Zero
	c.PorAlmox = map[int64]SaldoReposicao{}
	return &c
}

// QuantidadeDeReposicao leva a projeção até o alvo, no mínimo o lote mínimo,
// arredondada para cima ao múltiplo. Com múltiplo o resultado é múltiplo exato
// (barra de 6,35 M sugere 12,7 M, não 13); sem múltiplo sobe para o inteiro.
// Projeção que já alcança o alvo não pede nada.
func QuantidadeDeReposicao(alvo, projetado, loteMin, multiplo decimal.Decimal) decimal.Decimal {
	q := alvo.Sub(projetado)
	if !q.IsPositive() {
		return decimal.Zero
	}
	if loteMin.IsPositive() && q.LessThan(loteMin) {
		q = loteMin
	}
	if multiplo.IsPositive() {
		return q.Div(multiplo).Ceil().Mul(multiplo)
	}
	return q.Ceil()
}

// alvoDeReposicao: com máximo acima do ponto, repõe até o máximo (min-max do
// Oracle e do Focco); sem máximo, até o próprio ponto, que já tem a segurança.
func alvoDeReposicao(ponto, maximo decimal.Decimal) decimal.Decimal {
	if maximo.GreaterThan(ponto) {
		return maximo
	}
	return ponto
}

// Sugestoes calcula a reposição do item. Com política, só os almoxarifados da
// política são planejados; sem ela, vale o cadastro sobre o saldo somado.
func (it ItemReposicao) Sugestoes() []SugestaoReposicao {
	if len(it.Politicas) == 0 {
		return it.sugestaoSemPolitica()
	}
	politicas := append([]PoliticaReposicao(nil), it.Politicas...)
	sort.Slice(politicas, func(i, j int) bool { return politicas[i].WarehouseID < politicas[j].WarehouseID })

	semDestino := it.SemDestino
	// O que sai de uma origem deixa de estar livre para a próxima sugestão.
	usado := map[int64]decimal.Decimal{}
	var out []SugestaoReposicao
	for _, p := range politicas {
		ponto, maximo, seguranca := it.parametros(p)
		if !ponto.IsPositive() {
			continue
		}
		saldo := it.PorAlmox[p.WarehouseID]
		alvo := alvoDeReposicao(ponto, maximo)
		projetado := saldo.Livre().Add(saldo.Chegando)

		// A compra sem almoxarifado cobre o primeiro destino que precisar dela.
		compraCoberta := decimal.Zero
		if p.Origem == OrigemCompra && semDestino.IsPositive() {
			compraCoberta = decimal.Min(semDestino, decimal.Max(alvo.Sub(projetado), decimal.Zero))
			semDestino = semDestino.Sub(compraCoberta)
			projetado = projetado.Add(compraCoberta)
		}
		// No ponto sem nada a repor (ponto igual ao alvo) não há o que sugerir.
		if projetado.GreaterThan(ponto) || !alvo.Sub(projetado).IsPositive() {
			continue
		}
		s := it.base(saldo, projetado, ponto, seguranca, maximo)
		s.EmPedido = saldo.Chegando.Add(compraCoberta)
		s.WarehouseID = &p.WarehouseID
		s.Origem = p.Origem
		s.Motivo = "política do almoxarifado"
		if p.Origem == OrigemTransferencia {
			s.AlmoxOrigem = p.AlmoxOrigem
			s.Sugerido, s.Aviso = it.transferencia(p, alvo.Sub(projetado), usado)
			if s.Sugerido.IsPositive() {
				usado[*p.AlmoxOrigem] = usado[*p.AlmoxOrigem].Add(s.Sugerido)
			}
		} else {
			s.Sugerido = QuantidadeDeReposicao(alvo, projetado, it.LoteMinimo, it.LoteMultiplo)
			if !s.Sugerido.IsPositive() {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

func (it ItemReposicao) sugestaoSemPolitica() []SugestaoReposicao {
	origem := origemPadrao(it.TipoItem)
	if origem == "" || !it.Ponto.IsPositive() {
		return nil
	}
	projetado := it.Total.Livre().Add(it.Total.Chegando).Add(it.SemDestino)
	if projetado.GreaterThan(it.Ponto) {
		return nil
	}
	q := QuantidadeDeReposicao(alvoDeReposicao(it.Ponto, it.Maximo), projetado, it.LoteMinimo, it.LoteMultiplo)
	if !q.IsPositive() {
		return nil
	}
	s := it.base(it.Total, projetado, it.Ponto, it.Seguranca, it.Maximo)
	s.EmPedido = it.Total.Chegando.Add(it.SemDestino)
	s.Origem, s.Motivo, s.Sugerido = origem, it.Motivo, q
	return []SugestaoReposicao{s}
}

// parametros aplica a herança: o que a política não diz vem do cadastro. A
// segurança sobe o ponto quando é maior que ele; não soma, porque o ponto de
// pedido já a contém.
func (it ItemReposicao) parametros(p PoliticaReposicao) (ponto, maximo, seguranca decimal.Decimal) {
	ponto, maximo, seguranca = it.Ponto, it.Maximo, it.Seguranca
	if p.Minimo != nil {
		ponto = *p.Minimo
	}
	if p.Maximo != nil {
		maximo = *p.Maximo
	}
	if p.Seguranca != nil {
		seguranca = *p.Seguranca
	}
	return decimal.Max(ponto, seguranca), maximo, seguranca
}

// transferencia limita o pedido ao que a origem pode ceder sem cair abaixo do
// próprio ponto. Lote de compra não se aplica: move-se só o que falta.
func (it ItemReposicao) transferencia(p PoliticaReposicao, falta decimal.Decimal, usado map[int64]decimal.Decimal) (decimal.Decimal, string) {
	origem := *p.AlmoxOrigem
	livre := it.PorAlmox[origem].Livre().Sub(usado[origem])
	for _, o := range it.Politicas {
		if o.WarehouseID == origem {
			ponto, _, _ := it.parametros(o)
			livre = livre.Sub(ponto)
		}
	}
	livre = decimal.Max(livre, decimal.Zero)
	if livre.GreaterThanOrEqual(falta) {
		return falta, ""
	}
	if !livre.IsPositive() {
		return decimal.Zero, "A origem não tem saldo livre para transferir."
	}
	return livre, fmt.Sprintf("A origem só pode ceder %s dos %s que faltam.", QuantidadeBR(livre), QuantidadeBR(falta))
}

func (it ItemReposicao) base(saldo SaldoReposicao, projetado, ponto, seguranca, maximo decimal.Decimal) SugestaoReposicao {
	return SugestaoReposicao{
		ItemCode: it.ItemCode, Mask: it.Mask, Codigo: it.Codigo, Nome: it.Nome, Unidade: it.Unidade,
		Disponivel: saldo.Fisico, Reservado: saldo.Reservado, Retido: saldo.Retido, Requisitado: saldo.Requisitado,
		Projetado: projetado, PontoPedido: ponto, EstoqueSeguranca: seguranca, Maximo: maximo,
		LoteMinimo: it.LoteMinimo, LoteMultiplo: it.LoteMultiplo, ConsumoMedio: it.ConsumoMedio,
	}
}

// LinhaGeracaoReposicao é uma sugestão aceita pelo usuário.
type LinhaGeracaoReposicao struct {
	ItemCode    int64
	Mask        string
	WarehouseID *int64
	Origem      string
	AlmoxOrigem *int64
	Quantidade  decimal.Decimal
}

// ResultadoGeracaoReposicao diz quais documentos a reposição gerou.
type ResultadoGeracaoReposicao struct {
	RequisicaoCompra *int64  `json:"purchase_requisition_code,omitempty"`
	Transferencias   []int64 `json:"transfer_requisition_codes"`
	OrdensPlanejadas []int64 `json:"planned_order_codes"`
}
