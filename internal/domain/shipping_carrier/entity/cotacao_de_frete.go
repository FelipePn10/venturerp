package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"time"

	"github.com/shopspring/decimal"
)

// Carga é o que se quer despachar. O volume entra na escolha do veículo e, se a
// transportadora tem fator de cubagem, no peso cobrado.
type Carga struct {
	PesoKg          decimal.Decimal
	VolumeM3        decimal.Decimal
	ValorMercadoria decimal.Decimal
	// ProdutoPerigoso tira da cotação quem não aceita carga perigosa.
	ProdutoPerigoso bool
	UF              string
	Cidade          string
	CEP             string
	// Base é a data de saída; o prazo é contado a partir dela.
	Base time.Time
}

// Cotacao é o frete aberto em componentes. Mostrar o total só esconde o motivo
// de o frete ter dado aquilo — e é o componente que o usuário questiona.
type Cotacao struct {
	CarrierID    int64
	SupplierCode int64
	SupplierName string
	Modal        Modal
	// Regiao é a região atendida usada; vazia quando a tabela da capa foi usada.
	Regiao *RegiaoAtendida
	// PesoTaxado é o maior entre o peso real e o cubado: é sobre ele que o
	// valor por quilo e o pedágio incidem.
	PesoTaxado      decimal.Decimal
	PesoCubado      bool
	ValorDespacho   decimal.Decimal
	ValorPorPeso    decimal.Decimal
	ValorAdValorem  decimal.Decimal
	ValorGris       decimal.Decimal
	ValorPedagio    decimal.Decimal
	PisoAplicado    bool
	Total           decimal.Decimal
	PrazoDias       int16
	PrevisaoEntrega time.Time
	Alertas         []string
	// Impedimentos tiram a transportadora de "mais barata"/"mais rápida".
	Impedimentos []string
}

var ErrRegiaoNaoAtendida = domainerr.NewValidation("a transportadora não atende o destino informado")

// Cotar aplica a tabela da região atendida e, na falta dela, a tabela da capa.
// Ordem dos componentes: peso + ad valorem + GRIS + pedágio, e o piso só levanta
// o total — nunca substitui os componentes, senão a conferência não fecha.
func (t *Transportadora) Cotar(c Carga) (*Cotacao, error) {
	if !t.IsActive {
		return nil, domainerr.NewValidation("a transportadora está inativa")
	}
	if c.PesoKg.IsNegative() || c.ValorMercadoria.IsNegative() || c.VolumeM3.IsNegative() {
		return nil, domainerr.NewValidation("peso, volume e valor da mercadoria não podem ser negativos")
	}
	if c.ProdutoPerigoso && !t.AcceptsDangerousGoods {
		return nil, domainerr.NewValidation("não aceita produto perigoso")
	}
	cem := decimal.NewFromInt(100)

	var regiao *RegiaoAtendida
	for _, a := range t.ServiceAreas {
		// Mais de uma região pode atender o destino: a mais específica manda
		// (faixa de CEP, depois cidade, depois a UF).
		if a.Atende(c.UF, c.Cidade, c.CEP) && (regiao == nil || a.Especificidade() > regiao.Especificidade()) {
			regiao = a
		}
	}
	if regiao == nil && len(t.ServiceAreas) > 0 {
		return nil, ErrRegiaoNaoAtendida
	}

	kgRate := t.FreightKgRate
	pct := t.FreightPctValue
	piso := t.FreightMinValue
	prazo := int16(0)
	if t.AverageLeadDays != nil {
		prazo = *t.AverageLeadDays
	}
	if regiao != nil {
		if regiao.KgRate.IsPositive() {
			kgRate = regiao.KgRate
		}
		if regiao.PctValue.IsPositive() {
			pct = regiao.PctValue
		}
		if regiao.MinValue.IsPositive() {
			piso = regiao.MinValue
		}
		if regiao.LeadDays > 0 {
			prazo = regiao.LeadDays
		}
	}

	cot := &Cotacao{
		CarrierID:    t.ID,
		SupplierCode: t.SupplierCode,
		SupplierName: t.SupplierName,
		Modal:        t.Modal,
		Regiao:       regiao,
		PrazoDias:    prazo,
	}
	cot.PesoTaxado = c.PesoKg
	if t.CubageFactor.IsPositive() {
		if cubado := c.VolumeM3.Mul(t.CubageFactor); cubado.GreaterThan(c.PesoKg) {
			cot.PesoTaxado = cubado.Round(3)
			cot.PesoCubado = true
		}
	}
	cot.ValorPorPeso = kgRate.Mul(cot.PesoTaxado).Round(2)
	cot.ValorAdValorem = c.ValorMercadoria.Mul(pct).Div(cem).Round(2)
	cot.ValorGris = c.ValorMercadoria.Mul(t.GrisPct).Div(cem).Round(2)
	// Pedágio é cobrado por fração de 100 kg: 150 kg paga duas frações.
	if t.TollPer100Kg.IsPositive() && cot.PesoTaxado.IsPositive() {
		fracoes := cot.PesoTaxado.Div(decimal.NewFromInt(100)).Ceil()
		cot.ValorPedagio = t.TollPer100Kg.Mul(fracoes).Round(2)
	}
	total := cot.ValorPorPeso.Add(cot.ValorAdValorem).Add(cot.ValorGris).Add(cot.ValorPedagio)
	if piso.GreaterThan(total) {
		total = piso
		cot.PisoAplicado = true
	}
	// A taxa de despacho é cobrada por embarque, por cima do frete (inclusive
	// do piso): é assim que vem destacada no CT-e.
	cot.ValorDespacho = t.DispatchFee
	cot.Total = total.Add(t.DispatchFee)
	// Sem tabela (nem na capa nem na região) o total dá zero: não é frete
	// grátis, é cadastro incompleto — e zero ganharia de todo mundo.
	if !cot.Total.IsPositive() {
		return nil, domainerr.NewValidation("sem tabela de frete cadastrada: preencha a tabela na VSUP0140 para cotar")
	}

	base := c.Base
	if base.IsZero() {
		base = time.Now()
	}
	cot.PrevisaoEntrega = SomarDiasUteis(base, int(prazo))

	// Impedimento é o que impede esta transportadora de levar ESTA carga hoje:
	// ela continua na lista (o usuário vê o preço), mas não pode ser indicada
	// como a mais barata ou a mais rápida. O resto é aviso.
	data := func(d *time.Time) string { return d.Format("02/01/2006") }
	if t.ANTTExpiry != nil && t.ANTTExpiry.Before(base) {
		cot.Impedimentos = append(cot.Impedimentos, "RNTRC vencido em "+data(t.ANTTExpiry))
	}
	if t.InsuranceExpiry != nil && t.InsuranceExpiry.Before(base) {
		cot.Impedimentos = append(cot.Impedimentos, "seguro vencido em "+data(t.InsuranceExpiry))
	}
	if t.InsuranceCoverage.IsPositive() && c.ValorMercadoria.GreaterThan(t.InsuranceCoverage) {
		cot.Impedimentos = append(cot.Impedimentos, "o valor da carga passa da cobertura do seguro da transportadora")
	}
	if t.Modal == ModalRodoviario {
		cabe := false
		for _, v := range t.Vehicles {
			if v.IsActive && v.Cabe(c.PesoKg, c.VolumeM3) {
				cabe = true
				break
			}
		}
		if !cabe && len(t.Vehicles) > 0 {
			cot.Impedimentos = append(cot.Impedimentos, "nenhum veículo ativo da frota tem capacidade para esta carga")
		}
	}
	impede := map[string]bool{}
	for _, i := range cot.Impedimentos {
		impede[i] = true
	}
	for _, a := range t.Alertas(base) {
		if !impede[a] {
			cot.Alertas = append(cot.Alertas, a)
		}
	}
	// Sem fator de cubagem, carga leve e volumosa sai pelo peso real: o preço
	// fica abaixo do que a transportadora vai cobrar no CT-e e ela "ganha" a
	// comparação. 300 kg/m³ é a referência do rodoviário.
	if t.Modal == ModalRodoviario && t.CubageFactor.IsZero() && c.VolumeM3.Mul(decimal.NewFromInt(300)).GreaterThan(c.PesoKg) {
		cot.Alertas = append(cot.Alertas, "sem fator de cubagem cadastrado: para carga volumosa o frete real tende a ser maior que o cotado")
	}
	return cot, nil
}
