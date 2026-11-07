package governanca

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Processos que a política por status controla: cada um é um tipo de documento.
const (
	ProcOrcamento   = "ORCAMENTO"
	ProcVenda       = "VENDA"
	ProcCompra      = "COMPRA"
	ProcRequisicao  = "REQUISICAO"
	ProcProducao    = "PRODUCAO"
	ProcFaturamento = "FATURAMENTO"
)

// Políticas de um status num processo.
const (
	PoliticaLiberado  = "LIBERADO"
	PoliticaAviso     = "AVISO"
	PoliticaBloqueado = "BLOQUEADO"
)

// Processos, na ordem da matriz da tela.
var Processos = []string{ProcOrcamento, ProcVenda, ProcCompra, ProcRequisicao, ProcProducao, ProcFaturamento}

// StatusComPolitica: LIBERADO libera tudo por definição, os demais são ajustáveis.
var StatusComPolitica = []string{EmCadastro, Bloqueado, Descontinuado}

// RegraPolitica é uma célula da matriz status × processo.
type RegraPolitica struct {
	Status   string `json:"status"`
	Processo string `json:"processo"`
	Politica string `json:"politica"`
}

// Conferir valida a célula antes de gravar.
func (r RegraPolitica) Conferir() error {
	okStatus, okProcesso := false, false
	for _, s := range StatusComPolitica {
		okStatus = okStatus || s == r.Status
	}
	for _, p := range Processos {
		okProcesso = okProcesso || p == r.Processo
	}
	if !okStatus || !okProcesso {
		return domainerr.NewValidation("status ou processo inválido na política do cadastro")
	}
	if r.Politica != PoliticaLiberado && r.Politica != PoliticaAviso && r.Politica != PoliticaBloqueado {
		return domainerr.NewValidation("política inválida: use liberado, aviso ou bloqueado")
	}
	return nil
}

// MatrizDePolitica completa o que a empresa gravou com o padrão, que é bloquear.
func MatrizDePolitica(gravadas []RegraPolitica) []RegraPolitica {
	idx := map[string]string{}
	for _, g := range gravadas {
		idx[g.Status+"|"+g.Processo] = g.Politica
	}
	out := make([]RegraPolitica, 0, len(StatusComPolitica)*len(Processos))
	for _, s := range StatusComPolitica {
		for _, p := range Processos {
			pol := idx[s+"|"+p]
			if pol == "" {
				pol = PoliticaBloqueado
			}
			out = append(out, RegraPolitica{Status: s, Processo: p, Politica: pol})
		}
	}
	return out
}

// PoliticaDe diz o que o status permite no processo.
func PoliticaDe(status, processo string, matriz []RegraPolitica) string {
	if status == Liberado || status == "" {
		return PoliticaLiberado
	}
	for _, r := range matriz {
		if r.Status == status && r.Processo == processo {
			return r.Politica
		}
	}
	return PoliticaBloqueado
}

// Níveis de uma pendência.
const (
	NivelImpede  = "IMPEDE"
	NivelAtencao = "ATENCAO"
)

// PendenciaProcesso é o que falta, com o lugar de resolver.
type PendenciaProcesso struct {
	Nivel    string `json:"nivel"`
	Mensagem string `json:"mensagem"`
	Onde     string `json:"onde"`
}

// Prontidao é o estado do item em um processo do dia a dia.
type Prontidao struct {
	Processo   string              `json:"processo"`
	Rotulo     string              `json:"rotulo"`
	Aplica     bool                `json:"aplica"`
	Pronto     bool                `json:"pronto"`
	Politica   string              `json:"politica"`
	Pendencias []PendenciaProcesso `json:"pendencias"`
}

// FichaFiscal é o que a classificação de venda resolve para o item.
type FichaFiscal struct {
	NCM, CEST, ClassTrib, Unidade string
	Origem                        *int16
	IPI, ICMS, PIS, COFINS        *string
}

// AvaliarProntidao responde se o item está pronto para cada processo. Vai além do
// campo preenchido: olha estrutura, roteiro, fornecedor, conversão e o que a
// classificação fiscal resolve, e soma a política do status do cadastro.
func AvaliarProntidao(f Ficha, status string, matriz []RegraPolitica) []Prontidao {
	impede := func(m, onde string) PendenciaProcesso { return PendenciaProcesso{NivelImpede, m, onde} }
	atencao := func(m, onde string) PendenciaProcesso { return PendenciaProcesso{NivelAtencao, m, onde} }
	fisico := f.Tipo != Servico
	inativo := f.texto("health") == "INATIVO"

	estoqueFerramenta := false
	foraDoMRP := f.texto("planning_active") == "false" || f.texto("is_prototype") == "true" || f.texto("is_scrap") == "true" || f.texto("is_process_item") == "true"
	comprar := []PendenciaProcesso{}
	if !f.TemFornecedor {
		comprar = append(comprar, impede("Sem fornecedor preferencial", "VSUP0130 — Fornecedor Preferencial por Item"))
	}
	if !f.preenchido("supplies_purchase_uom") {
		comprar = append(comprar, impede("Sem unidade de compra", "VENT0200 — Suprimentos"))
	}
	if f.CompraEmOutraUM && !f.TemConversao {
		comprar = append(comprar, impede("Compra em outra unidade sem conversão para a do estoque", "VSUP0110 — Conversão de UM por Item"))
	}
	if !f.preenchido("supplies_warehouse_code") {
		comprar = append(comprar, atencao("Sem almoxarifado de recebimento: o pedido vai pedir um a cada linha", "VENT0200 — Suprimentos"))
	}
	if !f.preenchido("planning_purchase_lead_time_days") && !f.TemFornecedor {
		comprar = append(comprar, atencao("Sem prazo de compra: o MRP planejará com prazo zero", "VENT0200 — Planejamento"))
	}
	if f.texto("supplies_receipt_routing") == "INSPECAO" && !f.TemRotaInspecao {
		comprar = append(comprar, impede("Exige inspeção, mas não há rota de inspeção ativa para o item", "VINS0200 — Roteiro de Inspeção"))
	}

	produzir := []PendenciaProcesso{}
	if !f.TemEstrutura {
		produzir = append(produzir, impede("Sem estrutura (lista de materiais)", "VENT0210 — Estrutura de Produtos"))
	}
	if !f.TemRoteiro {
		produzir = append(produzir, impede("Sem roteiro de fabricação", "VENT0202 — Roteiro de Fabricação"))
	}
	if !f.TemCustoPadrao {
		produzir = append(produzir, atencao("Sem custo padrão calculado", "VPRO0300 — Custo Padrão"))
	}
	if f.texto("engineering_requires_drawing") == "true" && !f.TemDesenho {
		produzir = append(produzir, impede("Exige desenho e nenhum está vinculado", "VDES0100 — Desenhos"))
	}
	if f.texto("is_tool") == "true" && !f.TemFichaFerramenta {
		estoqueFerramenta = true
	}

	estocar := []PendenciaProcesso{}
	if !f.preenchido("warehouse_code") {
		estocar = append(estocar, impede("Sem almoxarifado padrão", "VENT0200 — Estoque"))
	}
	if f.preenchido("warehouse_shelf_life_days") && !f.ControlaLote {
		estocar = append(estocar, atencao("Tem validade, mas não controla lote: a validade só vale por lote", "VENT0200 — Estoque"))
	}
	if f.texto("warehouse_hazardous") == "true" && !f.preenchido("warehouse_storage_conditions") {
		estocar = append(estocar, atencao("Produto perigoso sem condição de armazenagem", "VENT0200 — Estoque"))
	}
	if estoqueFerramenta {
		estocar = append(estocar, atencao("Ferramenta sem ficha de vida útil vinculada", "VPRO1000 — Ferramentas"))
	}

	vender := []PendenciaProcesso{}
	if inativo {
		vender = append(vender, impede("Item inativo", "VENT0200 — Capa"))
	}
	if f.Tipo == Fabricado && !f.preenchido("commercial_sale_type") {
		vender = append(vender, impede("Sem tipo de venda (venda ou revenda)", "VENT0200 — Comercial"))
	}
	if !f.preenchido("commercial_description") {
		vender = append(vender, atencao("Sem descrição comercial: a nota usará a descrição técnica", "VENT0200 — Comercial"))
	}
	if f.texto("is_prototype") == "true" {
		vender = append(vender, atencao("Protótipo: confirme antes de vender", "VENT0200 — Capa"))
	}
	if (f.texto("engineering_oem") == "true" || f.texto("planning_exclusive") == "true") && !f.preenchido("commercial_exclusive_customer_code") {
		vender = append(vender, atencao("Item OEM ou exclusivo sem o cliente dono: qualquer cliente pode comprar", "VENT0200 — Comercial"))
	}

	faturar := []PendenciaProcesso{}
	if ncm := soDigitos(f.Fiscal.NCM); len(ncm) != 8 {
		faturar = append(faturar, impede("Sem NCM de 8 dígitos na classificação fiscal de venda", "VENT0200 — Contábil"))
	}
	if fisico && f.texto("accounting_origin") == "" {
		faturar = append(faturar, impede("Sem origem da mercadoria", "VENT0200 — Contábil"))
	}
	if f.Fiscal.ClassTrib == "" {
		faturar = append(faturar, atencao("Classificação fiscal sem cClassTrib: a nota usará o padrão do CFOP ou da empresa", "VFIS0350 — Classificações Fiscais"))
	}
	if fisico && !f.preenchido("accounting_gtin") {
		faturar = append(faturar, atencao("Sem GTIN: a nota sai sem código de barras", "VENT0200 — Contábil"))
	}

	if foraDoMRP {
		aviso := atencao("Fora do MRP: nenhuma ordem será sugerida", "VENT0200 — Planejamento")
		comprar = append(comprar, aviso)
		produzir = append(produzir, aviso)
	}

	monta := func(processo, rotulo string, aplica bool, politica string, p []PendenciaProcesso) Prontidao {
		pronto := aplica
		for _, x := range p {
			if x.Nivel == NivelImpede {
				pronto = false
			}
		}
		if politica == PoliticaBloqueado {
			pronto = false
		}
		if !aplica {
			p = []PendenciaProcesso{}
		}
		return Prontidao{Processo: processo, Rotulo: rotulo, Aplica: aplica, Pronto: pronto, Politica: politica, Pendencias: p}
	}
	return []Prontidao{
		monta("COMPRAR", "Comprar", f.Tipo == Comprado, PoliticaDe(status, ProcCompra, matriz), comprar),
		monta("PRODUZIR", "Produzir", f.Tipo == Fabricado, PoliticaDe(status, ProcProducao, matriz), produzir),
		monta("ESTOCAR", "Estocar", fisico, PoliticaLiberado, estocar),
		monta("VENDER", "Vender", true, PoliticaDe(status, ProcVenda, matriz), vender),
		monta("FATURAR", "Faturar", true, PoliticaDe(status, ProcFaturamento, matriz), faturar),
	}
}

func soDigitos(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// Severidades do impacto de uma alteração.
const (
	ImpactoBloqueia = "BLOQUEIA"
	ImpactoAtencao  = "ATENCAO"
)

// Retrato são os campos do cadastro cuja mudança tem consequência fora dele.
type Retrato struct {
	Unidade, UnidadeCompra, Tipo, Saude, GTIN, TipoEFD, UnidadeTributavel, Fator, Validade, Roteiro string
}

// Impacto é uma consequência de alterar um campo crítico.
type Impacto struct {
	Campo      string `json:"campo"`
	Rotulo     string `json:"rotulo"`
	Antes      string `json:"antes"`
	Depois     string `json:"depois"`
	Severidade string `json:"severidade"`
	Mensagem   string `json:"mensagem"`
}

// AvaliarImpacto compara o cadastro antes e depois e diz o que a mudança quebra.
// BLOQUEIA é o que deixaria saldo ou documento aberto incoerente.
func AvaliarImpacto(antes, depois Retrato, uso Uso) []Impacto {
	out := []Impacto{}
	add := func(campo, rotulo, a, d, sev, msg string) {
		out = append(out, Impacto{campo, rotulo, a, d, sev, msg})
	}
	saldo, err := decimal.NewFromString(strings.TrimSpace(uso.Saldo))
	temSaldo := err == nil && !saldo.IsZero()
	documentos := uso.PedidosVenda + uso.PedidosCompra + uso.OrdensFabricacao
	if antes.Unidade != depois.Unidade {
		switch {
		case temSaldo:
			add("unit_of_measurement", "Unidade de estoque", antes.Unidade, depois.Unidade, ImpactoBloqueia,
				fmt.Sprintf("Há saldo de %s %s: mudar a unidade reinterpretaria o estoque. Zere o saldo antes.", uso.Saldo, antes.Unidade))
		case documentos > 0:
			add("unit_of_measurement", "Unidade de estoque", antes.Unidade, depois.Unidade, ImpactoBloqueia,
				fmt.Sprintf("Há %d documento(s) aberto(s) na unidade %s. Encerre-os antes de mudar.", documentos, antes.Unidade))
		case len(uso.EstruturasPai) > 0:
			add("unit_of_measurement", "Unidade de estoque", antes.Unidade, depois.Unidade, ImpactoAtencao,
				fmt.Sprintf("O item é componente de %d estrutura(s): as quantidades delas seguirão com o número atual, agora em %s.", len(uso.EstruturasPai), depois.Unidade))
		}
	}
	if antes.Tipo != depois.Tipo {
		if uso.OrdensFabricacao > 0 || uso.PedidosCompra > 0 {
			add("type", "Tipo do item", antes.Tipo, depois.Tipo, ImpactoBloqueia,
				fmt.Sprintf("Há %d ordem(ns) de fabricação e %d pedido(s) de compra abertos para o tipo atual.", uso.OrdensFabricacao, uso.PedidosCompra))
		} else {
			add("type", "Tipo do item", antes.Tipo, depois.Tipo, ImpactoAtencao, "O MRP passa a tratar o item pelo novo tipo: confira estrutura, roteiro e fornecedor.")
		}
	}
	if antes.UnidadeCompra != depois.UnidadeCompra && uso.PedidosCompra > 0 {
		add("purchase_uom", "Unidade de compra", antes.UnidadeCompra, depois.UnidadeCompra, ImpactoAtencao,
			fmt.Sprintf("%d pedido(s) de compra aberto(s) seguem na unidade em que foram feitos.", uso.PedidosCompra))
	}
	if antes.Saude != depois.Saude && depois.Saude == "INATIVO" && (documentos > 0 || len(uso.EstruturasPai) > 0) {
		add("health", "Situação", antes.Saude, depois.Saude, ImpactoAtencao,
			fmt.Sprintf("O item está em %d documento(s) aberto(s) e em %d estrutura(s).", documentos, len(uso.EstruturasPai)))
	}
	if antes.GTIN != depois.GTIN && antes.GTIN != "" {
		add("gtin", "GTIN", antes.GTIN, depois.GTIN, ImpactoAtencao, "Etiquetas já impressas continuam com o código antigo; notas novas usam o novo.")
	}
	if antes.TipoEFD != depois.TipoEFD {
		add("efd_item_type", "Tipo do item na EFD", antes.TipoEFD, depois.TipoEFD, ImpactoAtencao, "Vale para as próximas EFD. O tipo 03 exige a composição no Registro 0210.")
	}
	if antes.UnidadeTributavel != depois.UnidadeTributavel || antes.Fator != depois.Fator {
		add("tax_unit", "Unidade tributável", antes.UnidadeTributavel+" × "+antes.Fator, depois.UnidadeTributavel+" × "+depois.Fator, ImpactoAtencao, "Notas novas sairão com a nova quantidade tributável; as autorizadas não mudam.")
	}
	if antes.Validade != depois.Validade {
		add("shelf_life_days", "Validade", antes.Validade, depois.Validade, ImpactoAtencao, "Lotes já recebidos mantêm a data que têm; só as próximas entradas usam a nova validade.")
	}
	if antes.Roteiro != depois.Roteiro && uso.PedidosCompra > 0 {
		add("receipt_routing", "Roteiro de recebimento", antes.Roteiro, depois.Roteiro, ImpactoAtencao, "Vale para os próximos recebimentos, inclusive dos pedidos já abertos.")
	}
	return out
}

// TemBloqueio diz se a alteração não pode ser gravada, e por quê.
func TemBloqueio(impactos []Impacto) error {
	msgs := []string{}
	for _, i := range impactos {
		if i.Severidade == ImpactoBloqueia {
			msgs = append(msgs, i.Rotulo+": "+i.Mensagem)
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return domainerr.NewValidation(strings.Join(msgs, " "))
}
