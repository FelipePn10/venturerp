package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
)

// Catálogo rota → módulo de permissão. Vale o prefixo mais longo. Toda rota
// autenticada precisa cair em um módulo (ou ser livre): o teste
// TestToRotaTemModuloDePermissao percorre o roteador e falha na rota nova que
// ninguém classificou — assim nenhum endpoint nasce fora do controle de acesso.
var rotasPorModulo = []struct {
	prefixo string
	modulo  acesso.Modulo
}{
	// Comercial
	{"/api/sales-order", acesso.Comercial},
	{"/api/sales-quotation", acesso.Comercial},
	{"/api/sales-division", acesso.Comercial},
	{"/api/sales-goals", acesso.Comercial},
	{"/api/representatives", acesso.Comercial},
	{"/api/commercial-commissions", acesso.Comercial},
	{"/api/recurring-sales", acesso.Comercial},
	{"/api/delivery-promise", acesso.Comercial},
	{"/api/delivery-promise-params", acesso.Comercial},
	{"/api/delivery-reschedule", acesso.Comercial},
	{"/api/customers", acesso.Comercial},
	{"/api/restriction", acesso.Comercial},
	{"/api/restriction-reason", acesso.Comercial},
	// Suprimentos
	{"/api/purchase-order", acesso.Suprimentos},
	{"/api/purchase-order-tolerances", acesso.Suprimentos},
	{"/api/purchase-price-tables", acesso.Suprimentos},
	{"/api/purchase-quotations", acesso.Suprimentos},
	{"/api/purchase-requisitions", acesso.Suprimentos},
	{"/api/suppliers", acesso.Suprimentos},
	{"/api/item-suppliers", acesso.Suprimentos},
	{"/api/item-conversions", acesso.Suprimentos},
	{"/api/third-party-services", acesso.Suprimentos},
	{"/api/shipping-carriers", acesso.Suprimentos},
	{"/api/procurement", acesso.Suprimentos},
	// Qualidade (inspeção de recebimento, homologação e IQF de fornecedor)
	{"/api/quality", acesso.Qualidade},
	{"/api/procurement/receiving-inspection-orders", acesso.Qualidade},
	{"/api/procurement/receiving-inspection-routes", acesso.Qualidade},
	{"/api/procurement/receiving-inspections", acesso.Qualidade},
	{"/api/procurement/receiving-divergences", acesso.Qualidade},
	{"/api/procurement/supplier-homologations", acesso.Qualidade},
	{"/api/procurement/supplier-scorecards", acesso.Qualidade},
	// Estoque, almoxarifado e expedição
	{"/api/stock", acesso.Estoque},
	{"/api/stock-remnants", acesso.Estoque},
	{"/api/inventory", acesso.Estoque},
	{"/api/warehouse", acesso.Estoque},
	{"/api/warehouse-addresses", acesso.Estoque},
	{"/api/estoque", acesso.Estoque},
	{"/api/lot-masks", acesso.Estoque},
	{"/api/shipments", acesso.Estoque},
	{"/api/customer-material", acesso.Estoque},
	// Engenharia
	{"/api/items", acesso.Engenharia},
	{"/api/pdm", acesso.Engenharia},
	{"/api/configurator", acesso.Engenharia},
	{"/api/drawings", acesso.Engenharia},
	{"/api/routing", acesso.Engenharia},
	{"/api/bom-headers", acesso.Engenharia},
	{"/api/machine", acesso.Engenharia},
	// Planejamento
	{"/api/mrp-calculation", acesso.Planejamento},
	{"/api/mrp-reports", acesso.Planejamento},
	{"/api/planning", acesso.Planejamento},
	{"/api/planning-params", acesso.Planejamento},
	{"/api/planned-order", acesso.Planejamento},
	{"/api/independent-demand", acesso.Planejamento},
	{"/api/forecast", acesso.Planejamento},
	{"/api/sales-forecast", acesso.Planejamento},
	{"/api/production-plan", acesso.Planejamento},
	{"/api/crp", acesso.Planejamento},
	{"/api/order-priority", acesso.Planejamento},
	{"/api/item-calendar-promise", acesso.Planejamento},
	{"/api/industrial-calendar", acesso.Planejamento},
	// Produção
	{"/api/production-order", acesso.Producao},
	{"/api/cutting-plans", acesso.Producao},
	{"/api/cutting-settings", acesso.Producao},
	{"/api/aps", acesso.Producao},
	{"/api/tool-production-sheet", acesso.Producao},
	// Manutenção
	{"/api/maintenance", acesso.Manutencao},
	// Assistência
	{"/api/technical-assistance", acesso.Assistencia},
	{"/api/pos-venda", acesso.Assistencia},
	{"/api/devolucoes-venda", acesso.Comercial},
	{"/api/contratos-venda", acesso.Comercial},
	{"/api/parametros-comerciais", acesso.Comercial},
	{"/api/consumer-service", acesso.Assistencia},
	// Fiscal
	{"/api/fiscal", acesso.Fiscal},
	{"/api/fiscal-classifications", acesso.Fiscal},
	{"/api/entry-operations", acesso.Fiscal},
	// Financeiro
	{"/api/financial", acesso.Financeiro},
	// Contabilidade
	{"/api/accounting", acesso.Contabilidade},
	{"/api/cost-center", acesso.Contabilidade},
	// Custos
	{"/api/standard-cost", acesso.Custos},
	{"/api/margin", acesso.Custos},
	{"/api/allocations", acesso.Custos},
	{"/api/overhead-allocation", acesso.Custos},
	// Cadastros gerais
	{"/api/enterprise", acesso.Cadastros},
	{"/api/employee", acesso.Cadastros},
	{"/api/location", acesso.Cadastros},
	{"/api/notifications", acesso.Cadastros},
	{"/api/cnpj", acesso.Cadastros},
	{"/api/reports", acesso.Cadastros},
	// Administração
	{"/api/audit-log", acesso.Administracao},
	{"/api/system", acesso.Administracao},
	{"/api/password-change-requests", acesso.Administracao},
	{"/api/acesso", acesso.Administracao},
	{"/api/aprovacoes", acesso.Administracao},
}

// leituraDeReferencia: consultas (GET) de cadastros que as outras áreas usam
// como pesquisa. Quem tem perfil comercial precisa achar o item no pedido sem
// ganhar acesso à engenharia — ler o cadastro não é poder alterá-lo.
var leituraDeReferencia = []string{
	"/api/items", "/api/pdm", "/api/customers", "/api/suppliers", "/api/item-suppliers", "/api/item-conversions",
	"/api/warehouse", "/api/warehouse-addresses", "/api/location", "/api/employee", "/api/enterprise",
	"/api/fiscal-classifications", "/api/fiscal/support", "/api/fiscal/reforma/classificacoes", "/api/shipping-carriers",
	"/api/industrial-calendar", "/api/order-priority", "/api/estoque/tipos-movimento", "/api/lot-masks",
	"/api/cost-center", "/api/financial/condicoes-pagamento", "/api/financial/centros-custo", "/api/financial/plano-contas",
	"/api/notifications/alerts", "/api/cnpj", "/api/stock/balances", "/api/routing", "/api/machine",
}

// rotasLivres valem para qualquer usuário autenticado, com ou sem perfil: o
// próprio pedido de troca de senha e a consulta das próprias permissões.
var rotasLivres = []struct {
	metodo string
	padrao *regexp.Regexp
}{
	{http.MethodPost, regexp.MustCompile(`^/api/password-change-requests/?$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/password-change-requests/[^/]+/complete$`)},
	{http.MethodGet, regexp.MustCompile(`^/api/acesso/minhas-permissoes$`)},
	{http.MethodGet, regexp.MustCompile(`^/api/version$`)},
	// Aprovações: quem pode aprovar é decidido pela alçada da política, não
	// pelo módulo — o aprovador de crédito pode ter perfil só financeiro.
	{http.MethodGet, regexp.MustCompile(`^/api/aprovacoes/solicitacoes(/[^/]+)?$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/aprovacoes/solicitacoes/[^/]+/(aprovar|reprovar)$`)},
	// Governança do item: cada área libera a sua visão — a permissão é a do
	// módulo da área, conferida no caso de uso, não a da engenharia.
	{http.MethodGet, regexp.MustCompile(`^/api/items/governanca/(pendentes|itens/[^/]+)$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/items/governanca/itens/[^/]+/(liberacoes|estorno)$`)},
}

// acoesEspeciais: rotas que não são inclusão comum — aprovar, autorizar na
// SEFAZ ou executar cálculo. Valem antes da regra pelo método HTTP.
var acoesEspeciais = []struct {
	metodo string
	padrao *regexp.Regexp
	acao   acesso.Acao
}{
	// Aprovação e liberação
	{http.MethodPost, regexp.MustCompile(`^/api/purchase-order/[^/]+/(approve|authorize)$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/purchase-order/suggestions/[^/]+/approve$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/financial/contas-pagar/[^/]+/approve$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/fiscal/entries/[^/]+/approve$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/sales-order/[^/]+/release$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/sales-quotation/[^/]+/release$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/amostragem/estados/liberar$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/lotes/[^/]+/decisao$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/lotes/[^/]+/cancelar$`), acesso.Excluir},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/capas/[^/]+/encerrar$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/capas/[^/]+/cancelar$`), acesso.Excluir},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/instrumentos/[^/]+/baixar$`), acesso.Excluir},
	{http.MethodPost, regexp.MustCompile(`^/api/pos-venda/series/[^/]+/baixa$`), acesso.Excluir},
	{http.MethodPost, regexp.MustCompile(`^/api/pos-venda/os/[^/]+/cancelamento$`), acesso.Excluir},
	{http.MethodPost, regexp.MustCompile(`^/api/items/governanca/itens/[^/]+/status$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/devolucoes-venda/[^/]+/efetivar$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/contratos-venda/[^/]+/situacao$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/devolucoes-venda/[^/]+/cancelamento$`), acesso.Excluir},
	{http.MethodPost, regexp.MustCompile(`^/api/pos-venda/os/[^/]+/faturamento$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/certificados/[^/]+/cancelar$`), acesso.Excluir},
	{http.MethodPost, regexp.MustCompile(`^/api/quality/documentos/[^/]+/aprovar$`), acesso.Aprovar},
	{http.MethodPost, regexp.MustCompile(`^/api/password-change-requests/[^/]+/(approve|reject)$`), acesso.Aprovar},
	// Documentos fiscais na SEFAZ / prefeitura
	{http.MethodPost, regexp.MustCompile(`^/api/fiscal/(exits|cte|nfse)/[^/]+/(authorize|cancel)$`), acesso.Autorizar},
	{http.MethodPost, regexp.MustCompile(`^/api/fiscal/exits/[^/]+/carta-correcao$`), acesso.Autorizar},
	{http.MethodPost, regexp.MustCompile(`^/api/fiscal/(inutilizacao|manifestacao)$`), acesso.Autorizar},
	{http.MethodPost, regexp.MustCompile(`^/api/fiscal/recebidas/[^/]+/manifestar$`), acesso.Autorizar},
	{http.MethodPost, regexp.MustCompile(`^/api/fiscal/devolucoes/[^/]+/efetivar$`), acesso.Autorizar},
	// Cálculos do planejamento
	{http.MethodPost, regexp.MustCompile(`^/api/mrp-calculation/run$`), acesso.Executar},
	{http.MethodPost, regexp.MustCompile(`^/api/mrp-calculation/suggestions/[^/]+/firm$`), acesso.Executar},
	{http.MethodPost, regexp.MustCompile(`^/api/planning/run-pipeline$`), acesso.Executar},
	{"", regexp.MustCompile(`^/api/planning/auto-run(/run-now)?$`), acesso.Executar},
	{http.MethodPost, regexp.MustCompile(`^/api/planning/scenarios/[^/]+/(calculate|promote)$`), acesso.Executar},
	{http.MethodPost, regexp.MustCompile(`^/api/crp/calculate$`), acesso.Executar},
}

// ClassificacaoDeRota é o que a rota exige.
type ClassificacaoDeRota struct {
	Livre             bool
	LeituraReferencia bool
	Modulo            acesso.Modulo
	Acao              acesso.Acao
	Conhecida         bool
}

// ClassificarRota decide módulo e ação de uma requisição.
func ClassificarRota(metodo, caminho string) ClassificacaoDeRota {
	caminho = strings.TrimRight(caminho, "/")
	if caminho == "" {
		caminho = "/"
	}
	for _, l := range rotasLivres {
		if l.metodo == metodo && l.padrao.MatchString(caminho) {
			return ClassificacaoDeRota{Livre: true, Conhecida: true}
		}
	}
	var c ClassificacaoDeRota
	melhor := -1
	for _, r := range rotasPorModulo {
		if temPrefixo(caminho, r.prefixo) && len(r.prefixo) > melhor {
			melhor, c.Modulo, c.Conhecida = len(r.prefixo), r.modulo, true
		}
	}
	if !c.Conhecida {
		return c
	}
	for _, e := range acoesEspeciais {
		if (e.metodo == "" || e.metodo == metodo) && e.padrao.MatchString(caminho) {
			c.Acao = e.acao
			return c
		}
	}
	switch metodo {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		c.Acao = acesso.Ler
		for _, p := range leituraDeReferencia {
			if temPrefixo(caminho, p) {
				c.LeituraReferencia = true
				break
			}
		}
	case http.MethodDelete:
		c.Acao = acesso.Excluir
	default:
		c.Acao = acesso.IncluirAlterar
	}
	return c
}

// temPrefixo casa o prefixo por segmento: /api/stock não pega /api/stock-remnants.
func temPrefixo(caminho, prefixo string) bool {
	return caminho == prefixo || strings.HasPrefix(caminho, prefixo+"/")
}
