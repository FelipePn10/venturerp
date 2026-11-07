package governanca

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

// Origem do tipo do item na EFD.
const (
	TipoEFDInformado = "INFORMADO"
	TipoEFDDeduzido  = "DEDUZIDO"
)

// CheckFiscal é uma conferência da prévia.
type CheckFiscal struct {
	Nivel        string `json:"nivel"`
	Mensagem     string `json:"mensagem"`
	ComoResolver string `json:"como_resolver"`
}

// PreviaFiscal é a identificação fiscal que o item vai levar à nota e à EFD.
type PreviaFiscal struct {
	NCM               string        `json:"ncm"`
	CEST              string        `json:"cest"`
	Origem            *int16        `json:"origem,omitempty"`
	ClassTrib         string        `json:"cclass_trib"`
	UnidadeComercial  string        `json:"unidade_comercial"`
	UnidadeTributavel string        `json:"unidade_tributavel"`
	FatorTributavel   string        `json:"fator_tributavel"`
	GTIN              string        `json:"gtin"`
	GTINTributavel    string        `json:"gtin_tributavel"`
	AliquotaIPI       *string       `json:"aliquota_ipi,omitempty"`
	AliquotaICMS      *string       `json:"aliquota_icms,omitempty"`
	AliquotaPIS       *string       `json:"aliquota_pis,omitempty"`
	AliquotaCOFINS    *string       `json:"aliquota_cofins,omitempty"`
	TipoEFD           string        `json:"tipo_efd"`
	TipoEFDOrigem     string        `json:"tipo_efd_origem"`
	Checks            []CheckFiscal `json:"checks"`
	PodeFaturar       bool          `json:"pode_faturar"`
}

// AvaliarFiscal monta a prévia e confere o que viraria rejeição ou multa. Os
// limites vêm dos leiautes: xProd com até 120 caracteres, NCM com 8 dígitos, GTIN
// com dígito verificador, cClassTrib com 6 dígitos (NT 2025.002).
func AvaliarFiscal(f Ficha, uso Uso) PreviaFiscal {
	unidade := f.texto("accounting_sale_unit_of_measurement")
	if unidade == "" {
		unidade = f.texto("warehouse_unit_of_measurement")
	}
	p := PreviaFiscal{
		NCM: f.Fiscal.NCM, CEST: f.texto("accounting_cest"), ClassTrib: f.Fiscal.ClassTrib, UnidadeComercial: unidade,
		UnidadeTributavel: f.texto("accounting_tax_unit"), FatorTributavel: f.texto("accounting_tax_unit_factor"),
		GTIN: f.texto("accounting_gtin"), GTINTributavel: f.texto("accounting_gtin_tax"), Origem: f.Fiscal.Origem,
		AliquotaIPI: f.Fiscal.IPI, AliquotaICMS: f.Fiscal.ICMS, AliquotaPIS: f.Fiscal.PIS, AliquotaCOFINS: f.Fiscal.COFINS,
		Checks: []CheckFiscal{},
	}
	if p.CEST == "" {
		p.CEST = f.Fiscal.CEST
	}
	if v := f.texto("accounting_origin"); v != "" {
		if o, err := strconv.ParseInt(v, 10, 16); err == nil {
			origem := int16(o)
			p.Origem = &origem
		}
	}
	if p.UnidadeTributavel == "" {
		p.UnidadeTributavel, p.FatorTributavel = p.UnidadeComercial, "1"
	}
	if p.GTINTributavel == "" {
		p.GTINTributavel = p.GTIN
	}
	impede := func(m, c string) { p.Checks = append(p.Checks, CheckFiscal{NivelImpede, m, c}) }
	atencao := func(m, c string) { p.Checks = append(p.Checks, CheckFiscal{NivelAtencao, m, c}) }

	fisico := f.Tipo != Servico
	if len(soDigitos(p.NCM)) != 8 {
		impede("NCM ausente ou sem 8 dígitos", "Informe a classificação fiscal de venda na pasta Contábil")
	}
	if fisico && p.Origem == nil {
		impede("Origem da mercadoria não informada", "Informe a origem na pasta Contábil ou na classificação fiscal")
	}
	for rotulo, gtin := range map[string]string{"comercial": f.texto("accounting_gtin"), "tributável": f.texto("accounting_gtin_tax")} {
		if gtin != "" && !valueobject.GTINValido(gtin) {
			impede("GTIN "+rotulo+" com dígito verificador errado: a SEFAZ rejeita", "Corrija o GTIN na pasta Contábil")
		}
	}
	if g := soDigitos(f.texto("accounting_gtin")); g != "" && valueobject.GTINValido(g) && !gtinBrasileiro(g) {
		atencao("GTIN sem prefixo 789/790: na homologação a nota saiu com SEM GTIN", "Confira no XML da primeira nota. A nota é autorizada mesmo assim")
	}
	if p.CEST != "" && len(soDigitos(p.CEST)) != 7 {
		impede("CEST sem 7 dígitos", "Corrija o CEST na pasta Contábil")
	}
	if p.CEST == "" && (f.texto("accounting_relevant_scale") != "" || f.texto("accounting_manufacturer_cnpj") != "") {
		atencao("Escala relevante e CNPJ do fabricante só vão à nota junto do CEST: sem ele ficam de fora", "Informe o CEST na pasta Contábil ou na classificação fiscal")
	}
	switch {
	case p.ClassTrib == "":
		atencao("Sem cClassTrib na classificação fiscal: a nota dependerá do padrão do CFOP ou da empresa", "Informe o cClassTrib na classificação fiscal")
	case len(soDigitos(p.ClassTrib)) != 6:
		impede("cClassTrib sem 6 dígitos", "Corrija o cClassTrib na classificação fiscal")
	}
	descricao := f.texto("commercial_description")
	if descricao == "" {
		descricao = f.texto("pdm_description_technique")
	}
	if utf8.RuneCountInString(descricao) > 120 {
		atencao("Descrição acima dos 120 caracteres da NF-e: será cortada", "Informe uma descrição comercial mais curta")
	}

	p.TipoEFD, p.TipoEFDOrigem = f.texto("accounting_efd_item_type"), TipoEFDInformado
	if p.TipoEFD == "" {
		p.TipoEFD, p.TipoEFDOrigem = sped.TipoItem(f.cadastroEFD(), ""), TipoEFDDeduzido
	}
	if p.TipoEFD == "04" && len(uso.EstruturasPai) > 0 {
		atencao("Produto acabado que é componente de outra estrutura: confira se não é 03 ou 06", "Escolha o tipo na pasta Contábil. O 0210 só existe para o tipo 03")
	}
	if p.TipoEFD == "99" {
		atencao("Tipo do item na EFD ficou em 99 - Outras", "Escolha o tipo na pasta Contábil")
	}
	p.PodeFaturar = true
	for _, c := range p.Checks {
		if c.Nivel == NivelImpede {
			p.PodeFaturar = false
		}
	}
	return p
}

func (f Ficha) cadastroEFD() sped.CadastroDoItem {
	tipo := f.Tipo
	c := sped.CadastroDoItem{TipoEngenharia: &tipo, Revenda: f.texto("commercial_sale_type") == "REVENDA", Embalagem: f.texto("commercial_is_packaging") == "true"}
	if v, err := strconv.Atoi(f.texto("supplies_type_of_use")); err == nil {
		c.TipoUso = &v
	}
	if f.texto("is_scrap") == "true" {
		c.Informado = "05"
	}
	return c
}

// gtinBrasileiro: prefixo GS1 Brasil. No GTIN-14 o primeiro dígito é o indicador
// da embalagem e o prefixo vem depois dele.
func gtinBrasileiro(gtin string) bool {
	if len(gtin) == 14 {
		gtin = gtin[1:]
	}
	return strings.HasPrefix(gtin, "789") || strings.HasPrefix(gtin, "790")
}
