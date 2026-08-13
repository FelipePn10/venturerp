// Command demo-manufacturing-documents regenerates the fictitious presentation
// documents with the same PDF engine used by the ERP.
package main

import (
	"log"
	"os"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/manufacturing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

func main() {
	generated := time.Date(2026, 8, 13, 9, 30, 0, 0, time.FixedZone("BRT", -3*60*60))
	validUntil := time.Date(2026, 12, 31, 23, 59, 59, 0, generated.Location())
	branding := manufacturing.Branding{
		BrandColorHex: "#1B5E36",
		Company: pdfkit.Company{
			Name: "Indústria Metalúrgica Venture Demonstração Ltda.", CNPJ: "12.345.678/0001-90",
			IE: "123.456.789.112", Address: "Av. das Indústrias, 1500 - Distrito Industrial - Joinville/SC - CEP 89219-000",
			Phone: "(47) 3333-4500", Email: "pcp@empresa-demonstracao.invalid",
		},
	}
	order := manufacturing.ManufacturingOrder{
		Branding: branding, Title: "ORDEM INTEGRADA DE FABRICAÇÃO E COMPRAS", Number: "OF 2026-000184",
		Status: "LIBERADA", Subtitle: "Documento de demonstração - revisão C", Generated: generated,
		// Token fictício com o mesmo formato opaco retornado pelo endpoint scanner/tokens.
		BarcodeValue: "OF1_DEMO-ORDER-2026-000184-NOT-A-REAL-TOKEN", BarcodeValidUntil: &validUntil,
		Summary: []manufacturing.Field{
			{Label: "Item produzido", Value: "TEA452-0 - Tampa de Acionamento"}, {Label: "Quantidade planejada", Value: "40 UN"},
			{Label: "Período industrial", Value: "13/08 a 28/08/2026"}, {Label: "Entrega prometida", Value: "31/08/2026"},
			{Label: "Origem da demanda", Value: "PV 2026-00491"}, {Label: "Prioridade", Value: "ALTA"},
			{Label: "Roteiro", Value: "RTE-TEA452-C"}, {Label: "Custo previsto", Value: "R$ 15.124,00 | R$ 378,10/un"},
		},
		Sections: []manufacturing.OrderSection{
			{Title: "Roteiro de fabricação", Columns: columns("Seq.", "Operação", "Centro / Recurso", "Início", "Término", "Tempo", "Situação", "Controle"), Rows: [][]string{
				{"010", "Corte laser conforme desenho DWG-452-C", "LASER-01", "14/08 07:30", "14/08 10:18", "2,80 h", "PENDENTE", "Operador habilitado NR-12"},
				{"020", "Dobra CNC - sequência de 4 dobras", "DOBRA-02", "17/08 08:00", "17/08 11:12", "3,20 h", "PENDENTE", "Gabarito GAB-452"},
				{"030", "Soldagem MIG dos reforços e suportes", "SOLDA-03", "18/08 07:30", "18/08 14:00", "6,50 h", "PENDENTE", "EPS-SOL-017"},
				{"040", "Usinagem de alojamentos e furações críticas", "CNC-04", "20/08 07:30", "20/08 11:30", "4,00 h", "PENDENTE", "Tolerância Ø25 H7"},
				{"050", "Tratamento superficial e pintura RAL 7016", "TERCEIRO", "21/08", "24/08", "40 un", "AGUARDA ENVIO", "OS 2026-00119"},
				{"060", "Montagem final, torque e identificação", "MONT-01", "26/08 07:30", "26/08 13:00", "5,50 h", "PENDENTE", "Torque 28 N·m"},
				{"070", "Inspeção dimensional e funcional final", "QUAL-01", "27/08 08:00", "27/08 16:00", "40 un", "PENDENTE", "Plano PI-452-C"},
			}},
			{Title: "Materiais e abastecimento", Columns: columns("Código", "Descrição", "Necessidade", "UM", "Disponível", "Comprar", "Data necessária", "Situação", "Rastreabilidade"), Rows: [][]string{
				{"MP-CH-1020-3.00", "Chapa SAE 1020 esp. 3,00 mm", "188,000", "KG", "68,000", "120,000", "14/08/2026", "PARCIAL", "Lote + certificado 3.1"},
				{"MP-BAR-1045-25", "Barra SAE 1045 Ø25 mm", "12,800", "KG", "12,800", "0,000", "19/08/2026", "DISPONÍVEL", "Separar lote L240811"},
				{"COMP-ROL-6205", "Rolamento 6205-2RS", "80,000", "UN", "0,000", "80,000", "21/08/2026", "COMPRAR", "Fornecedor homologado"},
				{"COMP-PAR-M8X25", "Parafuso sextavado 8.8 M8 × 25", "320,000", "UN", "500,000", "0,000", "25/08/2026", "DISPONÍVEL", "Inspeção visual"},
				{"INS-TINTA-7016", "Tinta poliéster pó RAL 7016", "18,000", "KG", "0,000", "20,000", "20/08/2026", "COMPRAR", "Validade mínima 9 meses"},
			}},
			{Title: "Ordens de compra vinculadas", Columns: columns("Pedido", "Fornecedor", "Item / Serviço", "Quantidade", "Entrega", "Valor unit.", "Valor total", "Situação"), Rows: [][]string{
				{"PC 2026-00372", "Rolamentos Sul Ltda.", "COMP-ROL-6205 - Rolamento 6205-2RS", "80 UN", "21/08/2026", "R$ 38,50", "R$ 3.080,00", "APROVADO"},
				{"PC 2026-00373", "Cores Técnicas Ltda.", "INS-TINTA-7016 - Tinta pó RAL 7016", "20 KG", "20/08/2026", "R$ 42,80", "R$ 856,00", "APROVADO"},
				{"PC 2026-00374", "Aços Joinville S.A.", "MP-CH-1020-3.00 - Chapa 3,00 mm", "120 KG", "14/08/2026", "R$ 8,90", "R$ 1.068,00", "URGENTE"},
			}},
			{Title: "Qualidade, custos e liberação", Columns: columns("Tipo", "Referência", "Descrição / Critério", "Amostra / Base", "Valor", "Situação", "Evidência requerida"), Rows: [][]string{
				{"QUALIDADE", "IQ-REC-6205", "Inspeção de recebimento dos rolamentos", "13 UN", "-", "PROGRAMADA", "Laudo dimensional + certificado"},
				{"QUALIDADE", "IQ-REC-CHAPA", "Composição química e espessura da chapa", "3 chapas", "-", "PROGRAMADA", "Certificado do fornecedor"},
				{"QUALIDADE", "PI-452-C", "Cotas críticas, torque e acabamento final", "100%", "-", "OBRIGATÓRIO", "Registro aprovado"},
				{"CUSTO", "MATERIAIS", "Materiais diretos previstos", "40 UN", "R$ 8.764,00", "ORÇADO", "Variação máxima ±5%"},
				{"CUSTO", "MOD + CIF", "Mão de obra e custos indiretos", "22,00 h", "R$ 4.560,00", "ORÇADO", "Apontamento por operação"},
				{"CUSTO", "SERVIÇOS", "Pintura eletrostática terceirizada", "40 UN", "R$ 1.800,00", "COTADO", "NF-e + certificado de pintura"},
			}},
		},
		Approvals:  []manufacturing.Field{{Label: "PCP", Value: "Marina Costa - aprovado em 13/08/2026 09:10"}, {Label: "Qualidade", Value: "Rafael Lima - aprovado em 13/08/2026 09:18"}, {Label: "Produção", Value: "Carlos Mendes - aprovado em 13/08/2026 09:24"}},
		Disclaimer: "DOCUMENTO FICTÍCIO PARA DEMONSTRAÇÃO. A execução real exige apontamentos, rastreabilidade e liberações registradas no ERP.",
	}
	write("docs/apresentacao/ordem-fabricacao-compras-demonstracao.pdf", manufacturing.RenderManufacturingOrder(order))
	structure := manufacturing.ItemStructure{
		Branding: branding, Title: "ESTRUTURA MULTINÍVEL DO ITEM", RootCode: "TEA452-0", Revision: "C", BaseQty: "1 UN", Generated: generated,
		Summary: []manufacturing.Field{{Label: "Descrição", Value: "Conjunto Tampa de Acionamento Industrial"}},
		Nodes: []manufacturing.StructureNode{
			node(0, "000", "TEA452-0", "Conjunto Tampa de Acionamento Industrial", "1,0000", "UN", "0,00%", "FABRICADO", "9 dias", "R$ 378,10"),
			node(1, "010", "SUB-CAR-452", "Carcaça soldada e usinada", "1,0000", "UN", "1,00%", "FABRICADO", "5 dias", "R$ 168,40"),
			node(2, "010.010", "PÇ-LAT-452-A", "Lateral esquerda cortada", "1,0000", "UN", "3,00%", "FABRICADO", "1 dia", "R$ 24,80"),
			node(3, "010.010.010", "MP-CH-1020-3.00", "Chapa SAE 1020 esp. 3,00 mm", "2,1000", "KG", "8,00%", "COMPRADO", "7 dias", "R$ 8,90"),
			node(2, "010.020", "PÇ-LAT-452-B", "Lateral direita cortada", "1,0000", "UN", "3,00%", "FABRICADO", "1 dia", "R$ 24,80"),
			node(3, "010.020.010", "MP-CH-1020-3.00", "Chapa SAE 1020 esp. 3,00 mm", "2,1000", "KG", "8,00%", "COMPRADO", "7 dias", "R$ 8,90"),
			node(2, "010.030", "PÇ-TAM-452", "Tampa superior dobrada", "1,0000", "UN", "3,00%", "FABRICADO", "2 dias", "R$ 39,20"),
			node(3, "010.030.010", "MP-CH-1020-3.00", "Chapa SAE 1020 esp. 3,00 mm", "3,2500", "KG", "8,00%", "COMPRADO", "7 dias", "R$ 8,90"),
			node(2, "010.040", "REF-452-01", "Reforço interno tipo A", "2,0000", "UN", "2,00%", "FABRICADO", "1 dia", "R$ 11,70"),
			node(3, "010.040.010", "MP-CH-1020-3.00", "Chapa SAE 1020 esp. 3,00 mm", "0,6500", "KG", "8,00%", "COMPRADO", "7 dias", "R$ 8,90"),
			node(1, "020", "SUB-EIX-452", "Conjunto do eixo", "1,0000", "UN", "0,50%", "FABRICADO", "4 dias", "R$ 96,70"),
			node(2, "020.010", "EIX-452-C", "Eixo usinado revisão C", "1,0000", "UN", "2,00%", "FABRICADO", "3 dias", "R$ 54,90"),
			node(3, "020.010.010", "MP-BAR-1045-25", "Barra SAE 1045 Ø25 mm", "0,3200", "KG", "12,00%", "COMPRADO", "10 dias", "R$ 14,20"),
			node(2, "020.020", "COMP-ROL-6205", "Rolamento 6205-2RS", "2,0000", "UN", "0,00%", "COMPRADO", "12 dias", "R$ 38,50"),
			node(1, "030", "KIT-FIX-452", "Kit de fixação e montagem", "1,0000", "KT", "0,00%", "SEPARADO", "1 dia", "R$ 28,60"),
			node(2, "030.010", "COMP-PAR-M8X25", "Parafuso sextavado 8.8 M8 × 25", "8,0000", "UN", "2,00%", "COMPRADO", "4 dias", "R$ 0,78"),
			node(2, "030.020", "COMP-ARR-M8", "Arruela lisa M8 zincada", "8,0000", "UN", "2,00%", "COMPRADO", "4 dias", "R$ 0,19"),
			node(2, "030.030", "COMP-POR-M8", "Porca autotravante M8", "8,0000", "UN", "2,00%", "COMPRADO", "5 dias", "R$ 0,62"),
			node(1, "040", "SERV-PINT-7016", "Pintura eletrostática RAL 7016", "1,0000", "SV", "0,00%", "TERCEIRIZADO", "4 dias", "R$ 45,00"),
			node(2, "040.010", "INS-TINTA-7016", "Tinta poliéster pó RAL 7016", "0,4500", "KG", "10,00%", "COMPRADO", "8 dias", "R$ 42,80"),
			node(1, "050", "ETQ-TEA452", "Etiqueta técnica serializada com QR Code", "1,0000", "UN", "1,00%", "COMPRADO", "3 dias", "R$ 3,90"),
		},
	}
	write("docs/apresentacao/estrutura-item-tea452-demonstracao.pdf", manufacturing.RenderItemStructure(structure))
}

func node(level int, position, code, description, quantity, unit, loss, origin, leadTime, unitCost string) manufacturing.StructureNode {
	return manufacturing.StructureNode{Level: level, Position: position, Code: code, Description: description, Quantity: quantity, Unit: unit, Loss: loss, Origin: origin, LeadTime: leadTime, UnitCost: unitCost}
}

func columns(titles ...string) []manufacturing.Column {
	result := make([]manufacturing.Column, len(titles))
	for i, title := range titles {
		result[i] = manufacturing.Column{Title: title, Weight: 1}
	}
	return result
}

func write(path string, data []byte) {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Fatal(err)
	}
}
