package service

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Assistente de conversão (melhoria 3): pelas medidas e pelo material, o peso da
// peça e as conversões que o cadastro precisa — "1 UN = X KG", "1 UN = 6000 MM",
// "1 UN = X M2" — e o lote múltiplo 1 de quem compra peça inteira. Era a dúvida
// da Tecnofer e da Usimac na implantação: a chapa vem em kg na nota e é
// consumida em unidade.

// Formatos de peça.
const (
	FormatoChapa          = "CHAPA"
	FormatoBarraRedonda   = "BARRA_REDONDA"
	FormatoBarraQuadrada  = "BARRA_QUADRADA"
	FormatoBarraSextavada = "BARRA_SEXTAVADA"
	FormatoTuboRedondo    = "TUBO_REDONDO"
	FormatoTuboRetangular = "TUBO_RETANGULAR"
	FormatoKgPorMetro     = "KG_POR_METRO"
)

// Densidades em g/cm³ (= kg/dm³). Aço carbono 7,85 é a da norma de chapas.
var Densidades = map[string]decimal.Decimal{
	"ACO":           decimal.RequireFromString("7.85"),
	"INOX":          decimal.RequireFromString("7.93"),
	"ALUMINIO":      decimal.RequireFromString("2.70"),
	"LATAO":         decimal.RequireFromString("8.50"),
	"COBRE":         decimal.RequireFromString("8.96"),
	"FERRO_FUNDIDO": decimal.RequireFromString("7.20"),
}

var (
	pi       = decimal.RequireFromString("3.14159265358979")
	raiz3Sob = decimal.RequireFromString("0.86602540378444") // √3/2
	milhao   = decimal.NewFromInt(1_000_000)
)

// MedidasDaPeca são as medidas em milímetro.
type MedidasDaPeca struct {
	Formato       string          `json:"shape"`
	Material      string          `json:"material,omitempty"`
	Densidade     decimal.Decimal `json:"density,omitempty"` // g/cm³; sobrepõe o material
	ComprimentoMM decimal.Decimal `json:"length_mm"`
	LarguraMM     decimal.Decimal `json:"width_mm,omitempty"`
	EspessuraMM   decimal.Decimal `json:"thickness_mm,omitempty"` // chapa e parede do tubo
	DiametroMM    decimal.Decimal `json:"diameter_mm,omitempty"`
	LadoMM        decimal.Decimal `json:"side_mm,omitempty"` // barra quadrada/sextavada (entre faces), tubo retangular
	KgPorMetro    decimal.Decimal `json:"kg_per_meter,omitempty"`
}

// SugestaoDeConversao: 1 De = Fator × Para.
type SugestaoDeConversao struct {
	De         string          `json:"from_uom"`
	Para       string          `json:"to_uom"`
	Fator      decimal.Decimal `json:"factor"`
	Explicacao string          `json:"explanation"`
}

// ResultadoAssistente é o que o assistente sugere.
type ResultadoAssistente struct {
	PesoKg       decimal.Decimal       `json:"weight_kg"`
	Formula      string                `json:"formula"`
	Sugestoes    []SugestaoDeConversao `json:"suggestions"`
	LoteMultiplo decimal.Decimal       `json:"multiple_lot"`
	Explicacao   string                `json:"multiple_lot_explanation"`
}

// CalcularPeca calcula o peso de uma unidade e sugere as conversões.
func CalcularPeca(m MedidasDaPeca) (ResultadoAssistente, error) {
	var r ResultadoAssistente
	formato := strings.ToUpper(strings.TrimSpace(m.Formato))
	densidade := m.Densidade
	if !densidade.IsPositive() && formato != FormatoKgPorMetro {
		d, ok := Densidades[strings.ToUpper(strings.TrimSpace(m.Material))]
		if !ok {
			return r, fmt.Errorf("informe o material (ACO, INOX, ALUMINIO, LATAO, COBRE, FERRO_FUNDIDO) ou a densidade em g/cm³")
		}
		densidade = d
	}
	c := m.ComprimentoMM
	if !c.IsPositive() {
		return r, fmt.Errorf("informe o comprimento da peça em mm")
	}
	exigir := func(nome string, v decimal.Decimal) error {
		if !v.IsPositive() {
			return fmt.Errorf("informe %s em mm", nome)
		}
		return nil
	}
	var volume decimal.Decimal // mm³
	switch formato {
	case FormatoChapa:
		if err := firstErr(exigir("a largura", m.LarguraMM), exigir("a espessura", m.EspessuraMM)); err != nil {
			return r, err
		}
		volume = c.Mul(m.LarguraMM).Mul(m.EspessuraMM)
		r.Formula = "C × L × E × densidade"
	case FormatoBarraRedonda:
		if err := exigir("o diâmetro", m.DiametroMM); err != nil {
			return r, err
		}
		volume = pi.Div(decimal.NewFromInt(4)).Mul(m.DiametroMM).Mul(m.DiametroMM).Mul(c)
		r.Formula = "π/4 × D² × C × densidade"
	case FormatoBarraQuadrada:
		if err := exigir("o lado", m.LadoMM); err != nil {
			return r, err
		}
		volume = m.LadoMM.Mul(m.LadoMM).Mul(c)
		r.Formula = "lado² × C × densidade"
	case FormatoBarraSextavada:
		if err := exigir("a medida entre faces (lado)", m.LadoMM); err != nil {
			return r, err
		}
		volume = raiz3Sob.Mul(m.LadoMM).Mul(m.LadoMM).Mul(c)
		r.Formula = "(√3/2) × S² × C × densidade"
	case FormatoTuboRedondo:
		if err := firstErr(exigir("o diâmetro externo", m.DiametroMM), exigir("a parede (espessura)", m.EspessuraMM)); err != nil {
			return r, err
		}
		if m.EspessuraMM.Mul(decimal.NewFromInt(2)).GreaterThanOrEqual(m.DiametroMM) {
			return r, fmt.Errorf("a parede do tubo precisa ser menor que o raio")
		}
		volume = pi.Mul(m.DiametroMM.Sub(m.EspessuraMM)).Mul(m.EspessuraMM).Mul(c)
		r.Formula = "π × (D − e) × e × C × densidade"
	case FormatoTuboRetangular:
		if err := firstErr(exigir("o lado", m.LadoMM), exigir("a parede (espessura)", m.EspessuraMM)); err != nil {
			return r, err
		}
		b := m.LarguraMM
		if !b.IsPositive() {
			b = m.LadoMM // tubo quadrado
		}
		e := m.EspessuraMM
		if e.Mul(decimal.NewFromInt(2)).GreaterThanOrEqual(decimal.Min(b, m.LadoMM)) {
			return r, fmt.Errorf("a parede do tubo precisa ser menor que a metade do lado")
		}
		volume = decimal.NewFromInt(2).Mul(m.LadoMM.Add(b)).Sub(e.Mul(decimal.NewFromInt(4))).Mul(e).Mul(c)
		r.Formula = "(2 × (A + B) − 4e) × e × C × densidade"
	case FormatoKgPorMetro:
		if !m.KgPorMetro.IsPositive() {
			return r, fmt.Errorf("informe o peso por metro (kg/m) do perfil")
		}
		r.PesoKg = m.KgPorMetro.Mul(c).Div(decimal.NewFromInt(1000)).Round(6)
		r.Formula = "kg/m × C"
	default:
		return r, fmt.Errorf("formato inválido: use CHAPA, BARRA_REDONDA, BARRA_QUADRADA, BARRA_SEXTAVADA, TUBO_REDONDO, TUBO_RETANGULAR ou KG_POR_METRO")
	}
	if formato != FormatoKgPorMetro {
		r.PesoKg = volume.Mul(densidade).Div(milhao).Round(6)
	}
	if !r.PesoKg.IsPositive() {
		return r, fmt.Errorf("as medidas dão peso zero: confira as unidades (mm)")
	}
	r.Sugestoes = append(r.Sugestoes, SugestaoDeConversao{De: "UN", Para: "KG", Fator: r.PesoKg,
		Explicacao: fmt.Sprintf("1 peça pesa %s kg (%s): a nota em kg vira unidades no estoque", Numero(r.PesoKg), r.Formula)})
	if formato == FormatoChapa {
		area := c.Mul(m.LarguraMM).Div(milhao).Round(6)
		r.Sugestoes = append(r.Sugestoes, SugestaoDeConversao{De: "UN", Para: "M2", Fator: area,
			Explicacao: fmt.Sprintf("1 chapa tem %s m² (C × L)", Numero(area))})
	} else {
		r.Sugestoes = append(r.Sugestoes, SugestaoDeConversao{De: "UN", Para: "MM", Fator: c,
			Explicacao: fmt.Sprintf("1 barra tem %s mm: a estrutura consome em mm e o estoque fica em barras", Numero(c))})
	}
	r.LoteMultiplo = decimal.NewFromInt(1)
	r.Explicacao = "compra de peça inteira: lote múltiplo 1 faz o MRP sugerir peças inteiras (2 barras, não 1,37)"
	return r, nil
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
