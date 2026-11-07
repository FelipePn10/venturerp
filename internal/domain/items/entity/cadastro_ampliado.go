package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/shopspring/decimal"
)

// Roteiros de recebimento do item.
const (
	RecebimentoDireto   = "DIRETO"
	RecebimentoPadrao   = "PADRAO"
	RecebimentoInspecao = "INSPECAO"
)

// Sentido do material no corte.
const (
	VeioNenhum      = "NONE"
	VeioComprimento = "LENGTH"
	VeioLargura     = "WIDTH"
)

// Momentos da baixa de material e formas de apontamento da OF.
const (
	BaixaNaLiberacao  = "REGISTRATION_RELEASE"
	BaixaNaProducao   = "PRODUCTION"
	ApontaPorOrdem    = "ORDER"
	ApontaPorOperacao = "OPERATION"
)

// TiposItemEFD é a tabela do campo TIPO_ITEM do Registro 0200 (Guia Prático
// EFD-ICMS/IPI).
var TiposItemEFD = map[string]string{
	"00": "Mercadoria para revenda", "01": "Matéria-prima", "02": "Embalagem", "03": "Produto em processo",
	"04": "Produto acabado", "05": "Subproduto", "06": "Produto intermediário", "07": "Material de uso e consumo",
	"08": "Ativo imobilizado", "09": "Serviços", "10": "Outros insumos", "99": "Outras",
}

// AplicarPadroes preenche o que a tela antiga não envia: um cliente que omite
// esses campos não pode gravar vazio numa coluna que tem regra.
func (i *Item) AplicarPadroes() {
	if i.Supplies.ReceiptRouting == "" {
		i.Supplies.ReceiptRouting = RecebimentoPadrao
	}
	if i.Planning.MaterialIssueTiming == "" {
		i.Planning.MaterialIssueTiming = BaixaNaProducao
	}
	if i.Planning.ProductionReportingType == "" {
		i.Planning.ProductionReportingType = ApontaPorOperacao
	}
	if i.Engineering.CuttingGrain == "" {
		i.Engineering.CuttingGrain = VeioNenhum
	}
}

// ForaDoMRP diz se o planejamento ignora o item: desligado no MRP, protótipo (de
// controle manual), sucata, que nasce do processo, ou item de processo em
// terceiro, que a operação externa gera.
func (i *Item) ForaDoMRP() bool {
	return !i.Planning.Active || i.IsPrototype || i.IsScrap || i.IsProcessItem
}

// EntregaPadrao é a data de entrega da linha de venda sem data: hoje mais o prazo
// estimado do cadastro. Nil quando o item não informa prazo.
func (i *Item) EntregaPadrao(hoje time.Time) *time.Time {
	if i.Commercial.EstimatedDeliveryDays == nil || *i.Commercial.EstimatedDeliveryDays <= 0 {
		return nil
	}
	d := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, *i.Commercial.EstimatedDeliveryDays)
	return &d
}

// ConferirVenda aplica as regras do cadastro à linha de orçamento e de pedido.
func (i *Item) ConferirVenda(quantidade decimal.Decimal, cliente *int64) error {
	if c := i.Commercial.ExclusiveCustomerCode; c != nil && (cliente == nil || *cliente != *c) {
		return domainerr.NewValidation(fmt.Sprintf("o item %s é exclusivo de outro cliente e não pode ser vendido neste documento", i.BusinessCode))
	}
	if !i.Warehouse.AcceptsFractionalQuantity && !quantidade.Equal(quantidade.Truncate(0)) {
		return domainerr.NewValidation(fmt.Sprintf("o item %s não aceita quantidade fracionada", i.BusinessCode))
	}
	if m := i.Commercial.MinimumSaleQuantity; m != nil && quantidade.LessThan(*m) {
		return domainerr.NewValidation(fmt.Sprintf("a venda mínima do item %s é %s", i.BusinessCode, m.String()))
	}
	if m := i.Commercial.SaleMultiple; m != nil && m.IsPositive() && !quantidade.Mod(*m).IsZero() {
		return domainerr.NewValidation(fmt.Sprintf("o item %s é vendido em múltiplos de %s", i.BusinessCode, m.String()))
	}
	return nil
}

func soDigitos(v string, min, max int) bool {
	if len(v) < min || len(v) > max {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// validarCadastroAmpliado confere os campos fiscais, logísticos e de suprimentos
// que vão para nota, EFD e recebimento. A mensagem é para quem preenche a tela.
// conferirMarcadores recusa marcações que se contradizem. Sem isto a tela deixava
// marcar tudo de uma vez: sucata que é item-base, fantasma comprado, serviço
// perigoso — e cada processo lia uma parte e ignorava o resto.
func (i *Item) conferirMarcadores() error {
	nomes := func(ms map[string]bool) []string {
		out := []string{}
		for _, n := range []string{"item base", "configurado", "protótipo", "ferramenta", "item de processo"} {
			if ms[n] {
				out = append(out, n)
			}
		}
		return out
	}
	marcados := map[string]bool{"item base": i.IsBase, "configurado": i.IsConfigured, "protótipo": i.IsPrototype, "ferramenta": i.IsTool, "item de processo": i.IsProcessItem}
	if i.IsScrap {
		if n := nomes(marcados); len(n) > 0 {
			return domainerr.NewValidation("sucata/retalho nasce do processo e não pode ser também " + strings.Join(n, ", "))
		}
	}
	if i.IsProcessItem {
		delete(marcados, "item de processo")
		delete(marcados, "protótipo")
		if n := nomes(marcados); len(n) > 0 {
			return domainerr.NewValidation("item de processo representa uma operação em terceiros e não pode ser também " + strings.Join(n, ", "))
		}
	}
	if i.Planning.Ghost {
		if i.Engineering.Type != types.FABRICADO {
			return domainerr.NewValidation("fantasma é um conjunto fabricado que não vai para o estoque: o tipo do item precisa ser Fabricado")
		}
		if i.Planning.Critical {
			return domainerr.NewValidation("fantasma não tem estoque, então não pode ser crítico: o alerta de falta nunca dispararia")
		}
		if i.Warehouse.CyclicalCountConfig != nil && i.Warehouse.CyclicalCountConfig.IsValid() {
			return domainerr.NewValidation("fantasma não tem estoque para contar: desmarque a contagem cíclica")
		}
	}
	if i.Commercial.ExportPackaging && i.Commercial.RequiresSpecialPackaging {
		return domainerr.NewValidation("marque embalagem de exportação ou embalagem diferenciada, não as duas: o romaneio destaca uma só")
	}
	if i.Commercial.SaleType != nil && strings.EqualFold(string(*i.Commercial.SaleType), "REVENDA") && i.Engineering.Type != types.COMPRADO {
		return domainerr.NewValidation("revenda é item comprado pronto: mude o tipo do item para Comprado ou a venda para Venda")
	}
	if i.Commercial.IsPackaging && i.Accounting.EFDItemType != nil && *i.Accounting.EFDItemType != "" && *i.Accounting.EFDItemType != "02" {
		return domainerr.NewValidation("item que é embalagem é do tipo 02 na EFD")
	}
	if m, s := i.Commercial.MinimumSaleQuantity, i.Commercial.SaleMultiple; m != nil && s != nil && m.IsPositive() && s.IsPositive() && !m.Mod(*s).IsZero() {
		return domainerr.NewValidation("o mínimo de venda precisa ser múltiplo do múltiplo de venda: o pedido no mínimo seria recusado")
	}
	if i.Engineering.Type == types.SERVICO {
		if i.Warehouse.Hazardous {
			return domainerr.NewValidation("serviço não é produto perigoso: o risco é do material, que deve ser outro item")
		}
		if i.Warehouse.CyclicalCountConfig != nil && i.Warehouse.CyclicalCountConfig.IsValid() {
			return domainerr.NewValidation("serviço não tem estoque para contar: desmarque a contagem cíclica")
		}
	}
	return nil
}

func (i *Item) validarCadastroAmpliado() error {
	i.AplicarPadroes()
	if err := i.conferirMarcadores(); err != nil {
		return err
	}
	for rotulo, gtin := range map[string]*string{"GTIN comercial": i.Accounting.GTIN, "GTIN tributável": i.Accounting.GTINTax} {
		if gtin != nil && !valueobject.GTINValido(*gtin) {
			return domainerr.NewValidation(fmt.Sprintf("o %s %s é inválido: confira os dígitos (8, 12, 13 ou 14) e o dígito verificador", rotulo, *gtin))
		}
	}
	if t := i.Accounting.EFDItemType; t != nil {
		if _, ok := TiposItemEFD[*t]; !ok {
			return domainerr.NewValidation("tipo do item na EFD inválido: use um código da tabela do Registro 0200")
		}
	}
	if i.Accounting.TaxUnit != nil && len(*i.Accounting.TaxUnit) > 6 {
		return domainerr.NewValidation("a unidade tributável tem no máximo 6 caracteres")
	}
	if f := i.Accounting.TaxUnitFactor; f != nil {
		if !f.IsPositive() {
			return domainerr.NewValidation("o fator da unidade tributável deve ser maior que zero")
		}
		if i.Accounting.TaxUnit == nil {
			return domainerr.NewValidation("informe a unidade tributável do fator de conversão")
		}
	}
	if i.Accounting.TaxUnit != nil && i.Accounting.TaxUnitFactor == nil {
		return domainerr.NewValidation("informe o fator da unidade tributável: quantas dela há em uma unidade comercial")
	}
	w := i.Warehouse
	if w.ShelfLifeDays != nil && *w.ShelfLifeDays <= 0 {
		return domainerr.NewValidation("a validade do item deve ser maior que zero dias")
	}
	if w.MinRemainingLifeDays != nil {
		if *w.MinRemainingLifeDays < 0 {
			return domainerr.NewValidation("a vida útil mínima no recebimento não pode ser negativa")
		}
		if w.ShelfLifeDays != nil && *w.MinRemainingLifeDays > *w.ShelfLifeDays {
			return domainerr.NewValidation("a vida útil mínima no recebimento não pode passar da validade do item")
		}
	}
	if w.UNNumber != nil {
		ok := len(*w.UNNumber) == 4
		for _, c := range *w.UNNumber {
			ok = ok && c >= '0' && c <= '9'
		}
		if !ok {
			return domainerr.NewValidation("o número ONU tem exatamente 4 dígitos")
		}
	}
	if w.Hazardous && w.UNNumber == nil {
		return domainerr.NewValidation("produto perigoso exige o número ONU")
	}
	if !w.Hazardous && (w.UNNumber != nil || w.HazardClass != nil) {
		return domainerr.NewValidation("número ONU e classe de risco são de produto perigoso: marque o item como perigoso")
	}
	if v := i.Engineering.Volume; v != nil && v.IsNegative() {
		return domainerr.NewValidation("o volume do item não pode ser negativo")
	}
	p := i.Planning
	for rotulo, dias := range map[string]*int{"prazo de compra": p.PurchaseLeadTimeDays, "prazo de fabricação": p.ManufacturingLeadTimeDays} {
		if dias != nil && *dias < 0 {
			return domainerr.NewValidation(fmt.Sprintf("o %s não pode ser negativo", rotulo))
		}
	}
	if p.MaterialIssueTiming != BaixaNaLiberacao && p.MaterialIssueTiming != BaixaNaProducao {
		return domainerr.NewValidation("momento da baixa de material inválido")
	}
	if p.ProductionReportingType != ApontaPorOrdem && p.ProductionReportingType != ApontaPorOperacao {
		return domainerr.NewValidation("forma de apontamento inválida")
	}
	e := p.Extras
	if e.MaximumStock.IsNegative() || e.Coverage < 0 {
		return domainerr.NewValidation("estoque máximo e cobertura não podem ser negativos")
	}
	if e.SafetyTimeDays != nil && (*e.SafetyTimeDays < 0 || *e.SafetyTimeDays > 365) {
		return domainerr.NewValidation("o tempo de segurança vai de 0 a 365 dias")
	}
	if e.MaximumLot != nil {
		if !e.MaximumLot.IsPositive() {
			return domainerr.NewValidation("o lote máximo precisa ser maior que zero")
		}
		if p.MinimumLot.IsPositive() && e.MaximumLot.LessThan(p.MinimumLot) {
			return domainerr.NewValidation("o lote máximo não pode ser menor que o lote mínimo")
		}
	}
	switch i.Engineering.CuttingGrain {
	case VeioNenhum, VeioComprimento, VeioLargura:
	default:
		return domainerr.NewValidation("sentido do material inválido: use sem sentido, comprimento ou largura")
	}
	if v := i.Commercial.VolumesPerUnit; v != nil && *v <= 0 {
		return domainerr.NewValidation("os volumes por unidade devem ser maiores que zero")
	}
	if (w.SecondaryUOM == nil) != (w.SecondaryFactor == nil) {
		return domainerr.NewValidation("a segunda unidade do estoque exige a unidade e o fator")
	}
	if f := w.SecondaryFactor; f != nil && !f.IsPositive() {
		return domainerr.NewValidation("o fator da segunda unidade deve ser maior que zero")
	}
	if w.SecondaryUOM != nil && *w.SecondaryUOM == i.Warehouse.UnitOfMeasurement.String() {
		return domainerr.NewValidation("a segunda unidade deve ser diferente da unidade do estoque")
	}
	if v := i.Accounting.ExTIPI; v != nil && !soDigitos(*v, 2, 3) {
		return domainerr.NewValidation("o EX TIPI tem 2 ou 3 dígitos")
	}
	if v := i.Accounting.ManufacturerCNPJ; v != nil && !soDigitos(*v, 14, 14) {
		return domainerr.NewValidation("o CNPJ do fabricante tem 14 dígitos")
	}
	// Rejeição 879 da SEFAZ: escala não relevante exige o CNPJ do fabricante.
	if v := i.Accounting.RelevantScale; v != nil && !*v && i.Accounting.ManufacturerCNPJ == nil {
		return domainerr.NewValidation("produção em escala não relevante exige o CNPJ do fabricante")
	}
	if v := i.Accounting.FCINumber; v != nil && len(*v) != 36 {
		return domainerr.NewValidation("o número da FCI tem 36 caracteres")
	}
	if i.Commercial.ExclusiveCustomerCode != nil && *i.Commercial.ExclusiveCustomerCode <= 0 {
		return domainerr.NewValidation("cliente exclusivo inválido")
	}
	s := i.Supplies
	cem := decimal.NewFromInt(100)
	if v := s.UnderDeliveryTolerancePct; v != nil && (v.IsNegative() || v.GreaterThan(cem)) {
		return domainerr.NewValidation("a tolerância de falta no recebimento vai de 0 a 100%")
	}
	if v := s.OverDeliveryTolerancePct; v != nil && v.IsNegative() {
		return domainerr.NewValidation("a tolerância de excesso no recebimento não pode ser negativa")
	}
	switch s.ReceiptRouting {
	case RecebimentoDireto, RecebimentoPadrao, RecebimentoInspecao:
	default:
		return domainerr.NewValidation("roteiro de recebimento inválido: use direto, padrão ou inspeção")
	}
	for campo, f := range map[string]struct {
		v   *string
		max int
	}{
		"observações gerais": {i.GeneralNotes, 1000}, "revisão": {i.Revision, 20}, "código anterior": {i.PreviousCode, 60},
		"fabricante": {i.ManufacturerName, 120}, "referência do fabricante": {i.ManufacturerPartNumber, 60},
		"condições de armazenagem": {w.StorageConditions, 120}, "classe de risco": {w.HazardClass, 10},
		"observações de planejamento": {p.Notes, 1000}, "classificação de suprimentos": {s.ClassificationCode, 40},
		"agrupamento": {e.GroupingKey, 50},
	} {
		if f.v != nil && len([]rune(*f.v)) > f.max {
			return domainerr.NewValidation(fmt.Sprintf("o campo %s passou do tamanho máximo de %d caracteres", campo, f.max))
		}
	}
	return nil
}
