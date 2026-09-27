package entity

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// Carga é o que se quer despachar. Volume é opcional: só entra na escolha do
// veículo, nunca no preço.
type Carga struct {
	PesoKg          decimal.Decimal
	VolumeM3        decimal.Decimal
	ValorMercadoria decimal.Decimal
	UF              string
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
	Regiao          *RegiaoAtendida
	ValorPorPeso    decimal.Decimal
	ValorAdValorem  decimal.Decimal
	ValorGris       decimal.Decimal
	ValorPedagio    decimal.Decimal
	PisoAplicado    bool
	Total           decimal.Decimal
	PrazoDias       int16
	PrevisaoEntrega time.Time
	Alertas         []string
}

var ErrRegiaoNaoAtendida = errors.New("a transportadora não atende o destino informado")

// Cotar aplica a tabela da região atendida e, na falta dela, a tabela da capa.
// Ordem dos componentes: peso + ad valorem + GRIS + pedágio, e o piso só levanta
// o total — nunca substitui os componentes, senão a conferência não fecha.
func (t *Transportadora) Cotar(c Carga) (*Cotacao, error) {
	if !t.IsActive {
		return nil, errors.New("a transportadora está inativa")
	}
	if c.PesoKg.IsNegative() || c.ValorMercadoria.IsNegative() {
		return nil, errors.New("peso e valor da mercadoria não podem ser negativos")
	}
	cem := decimal.NewFromInt(100)

	var regiao *RegiaoAtendida
	for _, a := range t.ServiceAreas {
		if a.Atende(c.UF, c.CEP) {
			// A faixa de CEP é mais específica que a UF: se as duas casam, a
			// faixa manda.
			if regiao == nil || (a.PostalCodeFrom != nil && regiao.PostalCodeFrom == nil) {
				regiao = a
			}
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
	cot.ValorPorPeso = kgRate.Mul(c.PesoKg).Round(2)
	cot.ValorAdValorem = c.ValorMercadoria.Mul(pct).Div(cem).Round(2)
	cot.ValorGris = c.ValorMercadoria.Mul(t.GrisPct).Div(cem).Round(2)
	// Pedágio é cobrado por fração de 100 kg: 150 kg paga duas frações.
	if t.TollPer100Kg.IsPositive() && c.PesoKg.IsPositive() {
		fracoes := c.PesoKg.Div(decimal.NewFromInt(100)).Ceil()
		cot.ValorPedagio = t.TollPer100Kg.Mul(fracoes).Round(2)
	}
	total := cot.ValorPorPeso.Add(cot.ValorAdValorem).Add(cot.ValorGris).Add(cot.ValorPedagio)
	if piso.GreaterThan(total) {
		total = piso
		cot.PisoAplicado = true
	}
	cot.Total = total

	base := c.Base
	if base.IsZero() {
		base = time.Now()
	}
	cot.PrevisaoEntrega = base.AddDate(0, 0, int(prazo))

	cot.Alertas = t.Alertas(base)
	if t.InsuranceCoverage.IsPositive() && c.ValorMercadoria.GreaterThan(t.InsuranceCoverage) {
		cot.Alertas = append(cot.Alertas,
			"o valor da carga passa da cobertura do seguro da transportadora")
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
			cot.Alertas = append(cot.Alertas, "nenhum veículo ativo da frota tem capacidade para esta carga")
		}
	}
	return cot, nil
}
