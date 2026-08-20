package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"
	"time"
)

type emailRow struct{ Label, Value string }
type emailSection struct {
	Title string
	Rows  []emailRow
}
type emailTable struct {
	Title   string
	Headers []string
	Rows    [][]string
}
type emailSectionPair struct {
	Left, Right emailSection
	HasRight    bool
}
type TemplateData struct {
	EnterpriseName, RecipientName, Title, Kicker, Severity, SeverityColor string
	ContextLabel, AccentColor                                             string
	Summary, Impact, Action, DesktopPath, GeneratedAt                     string
	LogoURL                                                               template.URL
	Metrics                                                               []emailRow
	Sections                                                              []emailSection
	SectionPairs                                                          []emailSectionPair
	Tables                                                                []emailTable
}

var emailTemplate = template.Must(template.New("email-enterprise").Parse(`<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><style>@media only screen and (max-width:720px){.email-shell{width:100%!important}.stack,.stack tbody,.stack tr,.stack td{display:block!important;width:auto!important}.metric{display:inline-block!important;width:46%!important;vertical-align:top}.pad{padding-left:18px!important;padding-right:18px!important}}</style></head>
<body style="margin:0;background:#EEF1E4;color:#14201A;font-family:Arial,Helvetica,sans-serif">
<div style="display:none;max-height:0;overflow:hidden">{{.Summary}}</div>
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#EEF1E4"><tr><td align="center" style="padding:28px 12px">
<table role="presentation" width="980" class="email-shell" cellspacing="0" cellpadding="0" style="width:100%;max-width:980px;background:#fff;border:1px solid #DCD6B8;border-radius:16px;overflow:hidden;box-shadow:0 8px 28px rgba(20,32,26,.08)">
<tr><td class="pad" style="padding:22px 30px;background:#14201A;border-bottom:6px solid {{.AccentColor}}"><table role="presentation" width="100%"><tr><td valign="middle">{{if .LogoURL}}<img src="{{.LogoURL}}" alt="" width="128" style="display:block;max-height:45px;object-fit:contain;margin-bottom:10px">{{end}}<div style="color:#BCC885;font-size:11px;letter-spacing:1.5px;text-transform:uppercase;font-weight:bold">{{.Kicker}}</div><div style="color:#fff;font-size:16px;font-weight:bold;margin-top:6px">{{.EnterpriseName}}</div></td><td align="right" valign="middle"><span style="display:inline-block;padding:7px 11px;border-radius:20px;background:{{.SeverityColor}};color:#fff;font-size:11px;letter-spacing:.8px;font-weight:bold">{{.Severity}}</span><div style="color:#BCC885;font-size:11px;margin-top:9px">{{.GeneratedAt}}</div></td></tr></table></td></tr>
<tr><td class="pad" style="padding:24px 30px 12px"><div style="color:{{.AccentColor}};font-size:11px;text-transform:uppercase;letter-spacing:1.2px;font-weight:bold">{{.ContextLabel}}</div><h1 style="margin:7px 0 7px;font-size:25px;line-height:1.2;color:#14201A">{{.Title}}</h1><p style="margin:0;color:#5F6A5C;font-size:13px">Olá, {{.RecipientName}}. Dados consolidados no momento em que o VentureERP identificou a ocorrência.</p></td></tr>
{{if .Metrics}}<tr><td class="pad" style="padding:10px 30px"><table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr>{{range .Metrics}}<td class="metric" valign="top" style="padding:12px 13px;background:#EEF1E4;border-right:4px solid #fff"><div style="font-size:10px;text-transform:uppercase;letter-spacing:.7px;color:#5D7822;font-weight:bold">{{.Label}}</div><div style="font-size:17px;line-height:1.25;color:#14201A;font-weight:bold;margin-top:6px">{{.Value}}</div></td>{{end}}</tr></table></td></tr>{{end}}
<tr><td class="pad" style="padding:10px 30px"><table class="stack" role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr><td valign="top" style="width:61%;padding:18px 20px;background:#F4F1DF;border-left:5px solid {{.AccentColor}}"><div style="font-size:10px;text-transform:uppercase;letter-spacing:1px;color:#5D7822;font-weight:bold;margin-bottom:7px">Leitura rápida</div><div style="font-size:15px;line-height:1.5;color:#14201A">{{.Summary}}</div></td><td style="width:12px"></td><td valign="top" style="padding:16px 18px;background:#F2E8B0"><div style="font-size:10px;text-transform:uppercase;color:#7E670F;font-weight:bold">Próxima ação</div><div style="font-size:13px;line-height:1.45;margin-top:7px">{{.Action}}</div></td></tr></table></td></tr>
{{range .SectionPairs}}<tr><td class="pad" style="padding:9px 30px"><table class="stack" role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr><td valign="top" style="width:50%;padding:15px 17px;border:1px solid #E1E5D4"><h2 style="font-size:14px;color:#324512;margin:0 0 8px;border-bottom:2px solid #BCC885;padding-bottom:7px">{{.Left.Title}}</h2><table role="presentation" width="100%">{{range .Left.Rows}}<tr><td valign="top" style="width:42%;padding:6px 8px 6px 0;color:#687164;font-size:11px;border-bottom:1px solid #EEF1E4">{{.Label}}</td><td valign="top" style="padding:6px 0;color:#14201A;font-size:12px;font-weight:600;border-bottom:1px solid #EEF1E4;word-break:break-word">{{.Value}}</td></tr>{{end}}</table></td>{{if .HasRight}}<td style="width:12px"></td><td valign="top" style="width:50%;padding:15px 17px;border:1px solid #E1E5D4"><h2 style="font-size:14px;color:#324512;margin:0 0 8px;border-bottom:2px solid #BCC885;padding-bottom:7px">{{.Right.Title}}</h2><table role="presentation" width="100%">{{range .Right.Rows}}<tr><td valign="top" style="width:42%;padding:6px 8px 6px 0;color:#687164;font-size:11px;border-bottom:1px solid #EEF1E4">{{.Label}}</td><td valign="top" style="padding:6px 0;color:#14201A;font-size:12px;font-weight:600;border-bottom:1px solid #EEF1E4;word-break:break-word">{{.Value}}</td></tr>{{end}}</table></td>{{end}}</tr></table></td></tr>{{end}}
{{range .Tables}}<tr><td class="pad" style="padding:9px 30px"><h2 style="font-size:14px;color:#324512;margin:0 0 9px">{{.Title}}</h2><div style="overflow-x:auto"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border:1px solid #DCD6B8"><tr>{{range .Headers}}<th align="left" style="background:#324512;color:#fff;padding:9px;font-size:10px">{{.}}</th>{{end}}</tr>{{range .Rows}}<tr>{{range .}}<td valign="top" style="padding:9px;font-size:11px;border-bottom:1px solid #E7E3D0">{{.}}</td>{{end}}</tr>{{end}}</table></div></td></tr>{{end}}
<tr><td class="pad" style="padding:10px 30px 25px"><table class="stack" role="presentation" width="100%"><tr><td valign="top" style="width:50%;padding:14px 16px;background:#EEF1E4"><div style="font-size:10px;text-transform:uppercase;color:#5D7822;font-weight:bold">Impacto operacional</div><p style="font-size:12px;line-height:1.45;margin:7px 0 0">{{.Impact}}</p></td><td style="width:12px"></td><td valign="top" style="padding:14px 16px;border:1px solid #BCC885"><div style="font-size:10px;text-transform:uppercase;color:#5D7822;font-weight:bold">Consultar no ERP Desktop</div><p style="font-size:12px;line-height:1.45;margin:7px 0 0"><strong>{{.DesktopPath}}</strong></p></td></tr></table></td></tr>
<tr><td class="pad" style="padding:16px 30px;background:#14201A;color:#BCC885;font-size:10px;line-height:1.5">Alerta interno automático · Os dados refletem o momento de geração. Confirme a situação atual no VentureERP antes de agir.</td></tr>
</table></td></tr></table></body></html>`))

func RenderDelivery(enterprise, color, logoDataURI, recipient, title, publicURL string, payload json.RawMessage) (string, string) {
	_ = color
	_ = publicURL
	data := buildTemplateData(enterprise, logoDataURI, recipient, title, payload)
	var html bytes.Buffer
	_ = emailTemplate.Execute(&html, data)
	var text strings.Builder
	fmt.Fprintf(&text, "%s\n%s\n\nRESUMO\n%s\n\nIMPACTO OPERACIONAL\n%s\n\nAÇÃO RECOMENDADA\n%s\n\nONDE CONSULTAR\nAbra o VentureERP Desktop e acesse: %s\n", data.EnterpriseName, data.Title, data.Summary, data.Impact, data.Action, data.DesktopPath)
	for _, section := range data.Sections {
		fmt.Fprintf(&text, "\n%s\n", strings.ToUpper(section.Title))
		for _, row := range section.Rows {
			fmt.Fprintf(&text, "%s: %s\n", row.Label, row.Value)
		}
	}
	return html.String(), text.String()
}

func buildTemplateData(enterprise, logoDataURI, recipient, title string, payload json.RawMessage) TemplateData {
	var p map[string]any
	_ = json.Unmarshal(payload, &p)
	eventKey, _ := p["_event_key"].(string)
	if eventKey == "" {
		eventKey, _ = p["event_key"].(string)
	}
	severity, _ := p["_severity"].(string)
	if severity == "" {
		severity = inferSeverity(eventKey, title)
	}
	module, _ := p["_module"].(string)
	if module == "" {
		module = moduleFor(eventKey)
	}
	data := TemplateData{EnterpriseName: enterprise, RecipientName: recipient, Title: title, Kicker: "VentureERP · " + module, Severity: severity, SeverityColor: severityColor(severity), GeneratedAt: time.Now().Format("02/01/2006 às 15:04"), DesktopPath: desktopPath(eventKey), Impact: impactFor(eventKey), Action: actionFor(eventKey), ContextLabel: contextLabel(eventKey), AccentColor: accentColor(eventKey)}
	if reminder, _ := p["_reminder_daily"].(bool); reminder {
		data.Kicker = "VentureERP · Lembrete diário · " + module
		data.ContextLabel = "Pendência ainda aberta · requer acompanhamento"
	}
	if strings.HasPrefix(logoDataURI, "data:image/png;base64,") || strings.HasPrefix(logoDataURI, "data:image/jpeg;base64,") {
		data.LogoURL = template.URL(logoDataURI)
	}
	if description, ok := p["descricao"].(string); ok && strings.TrimSpace(description) != "" {
		data.Summary = description
	} else {
		data.Summary = title + " requer conferência da equipe responsável."
	}
	keys := make([]string, 0, len(p))
	for key := range p {
		if !strings.HasPrefix(key, "_") && key != "descricao" && key != "link" && key != "event_key" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	general := emailSection{Title: "Dados da ocorrência"}
	for _, key := range keys {
		switch value := p[key].(type) {
		case map[string]any:
			data.Sections = append(data.Sections, sectionFromMap(labelFor(key), value))
		case []any:
			if table := tableFromArray(labelFor(key), value); len(table.Rows) > 0 {
				data.Tables = append(data.Tables, table)
			}
		default:
			general.Rows = append(general.Rows, emailRow{labelFor(key), formatValue(key, value)})
		}
	}
	if len(general.Rows) > 0 {
		data.Sections = append([]emailSection{general}, data.Sections...)
	}
	data.Metrics = metricsFor(eventKey, p)
	for i := 0; i < len(data.Sections); i += 2 {
		pair := emailSectionPair{Left: data.Sections[i]}
		if i+1 < len(data.Sections) {
			pair.Right, pair.HasRight = data.Sections[i+1], true
		}
		data.SectionPairs = append(data.SectionPairs, pair)
	}
	return data
}

func metricsFor(eventKey string, p map[string]any) []emailRow {
	var candidates []struct {
		label string
		paths []string
	}
	switch {
	case strings.HasPrefix(eventKey, "ESTOQUE_"):
		candidates = []struct {
			label string
			paths []string
		}{
			{"Disponível", []string{"estoque.disponivel", "quantidade"}},
			{"Unidade", []string{"item.unidade_estoque", "unidade_estoque", "um"}},
			{"Estoque mínimo", []string{"estoque.minimo", "estoque_minimo"}},
			{"Falta para repor", []string{"estoque.necessidade_reposicao", "divergencia"}},
			{"Lead time", []string{"fornecedor_recomendado.lead_time_dias"}},
		}
	case strings.HasPrefix(eventKey, "FISCAL_"):
		candidates = []struct {
			label string
			paths []string
		}{{"Documento", []string{"documento.numero", "numero"}}, {"Situação", []string{"documento.situacao", "status"}}, {"Valor", []string{"documento.total", "valor_total"}}, {"Emissão", []string{"documento.emissao", "emissao"}}}
	case strings.HasPrefix(eventKey, "PRODUCAO_") || strings.HasPrefix(eventKey, "MRP_") || strings.HasPrefix(eventKey, "APS_"):
		candidates = []struct {
			label string
			paths []string
		}{{"Ordem", []string{"ordem_fabricacao.numero", "ordem"}}, {"Quantidade", []string{"ordem_fabricacao.quantidade", "quantidade"}}, {"Progresso", []string{"ordem_fabricacao.progresso", "progresso"}}, {"Entrega", []string{"ordem_fabricacao.entrega", "entrega"}}}
	case strings.HasPrefix(eventKey, "COMERCIAL_"):
		candidates = []struct {
			label string
			paths []string
		}{{"Pedido", []string{"pedido.numero", "numero"}}, {"Cliente", []string{"pedido.cliente", "cliente"}}, {"Valor", []string{"pedido.total_liquido", "valor_total"}}, {"Entrega", []string{"pedido.entrega", "entrega"}}}
	case strings.HasPrefix(eventKey, "COMPRAS_"):
		candidates = []struct {
			label string
			paths []string
		}{{"Pedido", []string{"pedido_compra.numero", "numero"}}, {"Fornecedor", []string{"pedido_compra.fornecedor", "fornecedor"}}, {"Previsão", []string{"pedido_compra.previsao", "previsao"}}, {"Valor", []string{"pedido_compra.total", "valor_total"}}}
	case strings.HasPrefix(eventKey, "FINANCEIRO_"):
		candidates = []struct {
			label string
			paths []string
		}{{"Título", []string{"titulo.numero", "numero"}}, {"Vencimento", []string{"titulo.vencimento", "vencimento"}}, {"Valor", []string{"titulo.valor_total", "valor_total"}}, {"Saldo", []string{"titulo.saldo", "saldo"}}}
	}
	metrics := make([]emailRow, 0, len(candidates))
	for _, candidate := range candidates {
		for _, path := range candidate.paths {
			if value, key, ok := nestedValue(p, path); ok {
				metrics = append(metrics, emailRow{candidate.label, formatValue(key, value)})
				break
			}
		}
	}
	if len(metrics) > 5 {
		return metrics[:5]
	}
	return metrics
}

func nestedValue(values map[string]any, path string) (any, string, bool) {
	parts := strings.Split(path, ".")
	var current any = values
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, part, false
		}
		current, ok = m[part]
		if !ok || current == nil || fmt.Sprint(current) == "" {
			return nil, part, false
		}
	}
	return current, parts[len(parts)-1], true
}

func sectionFromMap(title string, values map[string]any) emailSection {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := emailSection{Title: title}
	for _, k := range keys {
		s.Rows = append(s.Rows, emailRow{labelFor(k), formatValue(k, values[k])})
	}
	return s
}
func tableFromArray(title string, values []any) emailTable {
	table := emailTable{Title: title}
	if len(values) == 0 {
		return table
	}
	first, ok := values[0].(map[string]any)
	if !ok {
		return table
	}
	keys := make([]string, 0, len(first))
	for k := range first {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 8 {
		keys = keys[:8]
	}
	for _, k := range keys {
		table.Headers = append(table.Headers, labelFor(k))
	}
	limit := len(values)
	if limit > 20 {
		limit = 20
	}
	for _, raw := range values[:limit] {
		rowMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		row := make([]string, 0, len(keys))
		for _, k := range keys {
			row = append(row, formatValue(k, rowMap[k]))
		}
		table.Rows = append(table.Rows, row)
	}
	return table
}
func labelFor(key string) string {
	known := map[string]string{"codigo": "Código", "codigo_interno": "Código interno", "numero": "Número", "status": "Situação", "situacao": "Situação", "emissao": "Emissão", "entrega": "Entrega", "cliente_codigo": "Cliente", "representante_codigo": "Representante", "responsavel_usuario_id": "Responsável", "item_codigo": "Item", "almoxarifado_id": "Almoxarifado", "almoxarifado": "Almoxarifado", "quantidade": "Quantidade", "saldo_fisico": "Saldo físico", "reservado": "Reservado", "disponivel": "Disponível", "minimo": "Estoque mínimo", "maximo": "Estoque máximo", "seguranca": "Estoque de segurança", "necessidade_reposicao": "Necessidade de reposição", "consumo_medio_mensal": "Consumo médio mensal", "cobertura_dias": "Cobertura estimada", "unidade_estoque": "Unidade de estoque", "unidade_compra": "Unidade de compra", "supplier_item_code": "Código no fornecedor", "codigo_item_fornecedor": "Código no fornecedor", "lead_time_dias": "Lead time", "quantidade_embalagem": "Embalagem de compra", "homologado": "Fornecedor homologado", "bloqueado": "Fornecedor bloqueado", "quantidade_contada": "Quantidade contada", "quantidade_esperada": "Quantidade esperada", "divergencia": "Divergência", "valor_total": "Valor total", "total_liquido": "Total líquido", "total_bruto": "Total bruto", "preco_unitario": "Preço unitário", "programada_para": "Programada para", "regra": "Regra acionada", "itens": "Itens", "pedido": "Pedido", "orcamento": "Orçamento", "observacoes": "Observações"}
	if v := known[key]; v != "" {
		return v
	}
	parts := strings.Split(strings.ReplaceAll(key, "-", "_"), "_")
	for i := range parts {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, " ")
}
func contextLabel(key string) string {
	switch {
	case strings.HasPrefix(key, "ESTOQUE_"):
		return "Decisão de abastecimento e disponibilidade"
	case strings.HasPrefix(key, "FISCAL_"):
		return "Conformidade e continuidade fiscal"
	case strings.HasPrefix(key, "PRODUCAO_"), strings.HasPrefix(key, "MRP_"), strings.HasPrefix(key, "APS_"):
		return "Continuidade e prazo da fábrica"
	case strings.HasPrefix(key, "COMPRAS_"):
		return "Suprimentos e compromisso do fornecedor"
	case strings.HasPrefix(key, "COMERCIAL_"):
		return "Atendimento ao cliente e compromisso de entrega"
	case strings.HasPrefix(key, "FINANCEIRO_"):
		return "Exposição financeira e necessidade de tratativa"
	default:
		return "Monitoramento operacional"
	}
}
func accentColor(key string) string {
	switch {
	case strings.HasPrefix(key, "ESTOQUE_"), strings.HasPrefix(key, "COMPRAS_"):
		return "#5D7822"
	case strings.HasPrefix(key, "PRODUCAO_"), strings.HasPrefix(key, "MRP_"), strings.HasPrefix(key, "APS_"):
		return "#789334"
	case strings.HasPrefix(key, "FISCAL_"), strings.HasPrefix(key, "FINANCEIRO_"):
		return "#324512"
	default:
		return "#9AAE56"
	}
}
func formatValue(key string, value any) string {
	if value == nil {
		return "—"
	}
	switch v := value.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Format("02/01/2006 15:04")
		}
		return v
	case float64:
		if strings.Contains(key, "valor") || strings.Contains(key, "preco") || strings.Contains(key, "total") {
			return "R$ " + strconv.FormatFloat(v, 'f', 2, 64)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "Sim"
		}
		return "Não"
	default:
		return fmt.Sprint(v)
	}
}
func moduleFor(key string) string {
	if i := strings.Index(key, "_"); i > 0 {
		return strings.ReplaceAll(key[:i], "MRP", "Planejamento")
	}
	return "OPERAÇÃO"
}
func inferSeverity(key, title string) string {
	upper := key + " " + strings.ToUpper(title)
	if strings.Contains(upper, "CRITIC") || strings.Contains(upper, "REJEIT") || strings.Contains(upper, "NEGATIVO") || strings.Contains(upper, "VENCIDA") || strings.Contains(upper, "FALHA") {
		return "CRÍTICO"
	}
	if strings.Contains(upper, "CONCLUID") || strings.Contains(upper, "APROVAD") || strings.Contains(upper, "AUTORIZAD") || strings.Contains(upper, "CONVERTIDO") {
		return "INFORMATIVO"
	}
	return "ATENÇÃO"
}
func severityColor(severity string) string {
	if strings.Contains(severity, "CRÍT") {
		return "#9B2C2C"
	}
	if strings.Contains(severity, "ATEN") {
		return "#7E670F"
	}
	return "#465D18"
}
func desktopPath(key string) string {
	paths := map[string]string{"COMERCIAL": "Comercial › Pedidos e Orçamentos", "ESTOQUE": "Estoque › Alertas e Contagens", "FISCAL": "Fiscal › Documentos Fiscais", "CADASTRO": "Cadastros › Itens", "COMPRAS": "Compras › Pendências", "PRODUCAO": "Produção › Ordens de Fabricação", "MRP": "Planejamento › MRP › Exceções", "APS": "Planejamento › APS", "MANUTENCAO": "Manutenção › Programação", "QUALIDADE": "Qualidade › Pendências", "FINANCEIRO": "Financeiro › Alertas", "SEGURANCA": "Administração › Segurança", "OPERACAO": "Administração › Operação"}
	prefix := key
	if i := strings.Index(key, "_"); i > 0 {
		prefix = key[:i]
	}
	if p := paths[prefix]; p != "" {
		return p
	}
	return "Central de Alertas"
}
func impactFor(key string) string {
	if strings.Contains(key, "FISCAL") {
		return "Pode afetar conformidade fiscal, recebimento, faturamento ou escrituração."
	}
	if strings.Contains(key, "ESTOQUE") || strings.Contains(key, "COMPRAS") {
		return "Pode causar falta de material, parada de produção, divergência física ou atraso de entrega."
	}
	if strings.Contains(key, "PRODUCAO") || strings.Contains(key, "MRP") || strings.Contains(key, "APS") {
		return "Pode comprometer capacidade, prazo prometido, consumo ou continuidade da fábrica."
	}
	if strings.Contains(key, "FINANCEIRO") {
		return "Pode afetar caixa, crédito, cobrança, pagamento ou conciliação."
	}
	return "A ocorrência pode gerar retrabalho, atraso ou perda de controle se não for conferida."
}
func actionFor(key string) string {
	if strings.Contains(key, "APROVAD") || strings.Contains(key, "CONCLUID") || strings.Contains(key, "AUTORIZAD") || strings.Contains(key, "CONVERTIDO") {
		return "Confirme os dados registrados e acompanhe as próximas etapas do processo."
	}
	if strings.Contains(key, "REJEIT") || strings.Contains(key, "DIVERGEN") || strings.Contains(key, "FALHA") || strings.Contains(key, "NEGATIVO") {
		return "Priorize a conferência, identifique a causa e registre a correção no processo correspondente."
	}
	if strings.Contains(key, "VENC") || strings.Contains(key, "ATRAS") || strings.Contains(key, "PROXIM") {
		return "Defina um responsável e uma data de regularização antes que o prazo operacional seja comprometido."
	}
	return "Abra a tela indicada, confira os dados e atribua a tratativa ao responsável da área."
}
