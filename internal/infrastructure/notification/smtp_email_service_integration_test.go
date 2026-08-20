package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

func TestSMTPHostingerIntegration(t *testing.T) {
	if os.Getenv("RUN_SMTP_INTEGRATION") != "1" {
		t.Skip("integração SMTP real desabilitada")
	}
	to := os.Getenv("SMTP_TEST_TO")
	if to == "" {
		to = os.Getenv("SMTP_FROM")
	}
	service := NewEmailService(SMTPConfig{
		Host: os.Getenv("SMTP_HOST"), Port: os.Getenv("SMTP_PORT"),
		User: os.Getenv("SMTP_USER"), Password: os.Getenv("SMTP_PASSWORD"),
		From: os.Getenv("SMTP_FROM"), Timeout: 20 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	err := service.SendMessage(ctx, ports.EmailMessage{
		MessageID: "<integration-hostinger-" + time.Now().UTC().Format("20060102150405") + "@venturerp.com>",
		To:        []string{to}, Subject: "[VentureERP] Validação do serviço de alertas",
		Text: "Validação interna do serviço enterprise de alertas do VentureERP.",
		HTML: "<p><strong>Validação concluída:</strong> serviço enterprise de alertas do VentureERP.</p>",
	})
	if err != nil {
		t.Fatalf("envio SMTP Hostinger falhou: %v", err)
	}
}

func TestSMTPHostingerAllCatalogPreviews(t *testing.T) {
	mode := os.Getenv("RUN_SMTP_CATALOG_PREVIEWS")
	if mode != "1" && mode != "validate" {
		t.Skip("prévia integral do catálogo desabilitada")
	}
	_, currentFile, _, _ := runtime.Caller(0)
	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "000308_enterprise_notifications.up.sql")
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`\('([A-Z0-9_]+)','([^']+)'`)
	matches := re.FindAllStringSubmatch(string(migration), -1)
	seen := map[string]bool{}
	type preview struct{ key, name string }
	previews := make([]preview, 0, 80)
	for _, match := range matches {
		if !strings.Contains(match[1], "_") || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		previews = append(previews, preview{key: match[1], name: match[2]})
	}
	if len(previews) != 75 {
		t.Fatalf("catálogo inesperado: obtidos %d eventos, esperados 75", len(previews))
	}
	if mode == "validate" {
		return
	}
	if selected := strings.TrimSpace(os.Getenv("SMTP_PREVIEW_KEYS")); selected != "" {
		allowed := map[string]bool{}
		for _, key := range strings.Split(selected, ",") {
			allowed[strings.TrimSpace(key)] = true
		}
		filtered := previews[:0]
		for _, item := range previews {
			if allowed[item.key] {
				filtered = append(filtered, item)
			}
		}
		previews = filtered
		if len(previews) != len(allowed) {
			t.Fatalf("um ou mais eventos selecionados não existem no catálogo")
		}
	}
	to := os.Getenv("SMTP_TEST_TO")
	if to == "" {
		to = os.Getenv("SMTP_FROM")
	}
	service := NewEmailService(SMTPConfig{Host: os.Getenv("SMTP_HOST"), Port: os.Getenv("SMTP_PORT"), User: os.Getenv("SMTP_USER"), Password: os.Getenv("SMTP_PASSWORD"), From: os.Getenv("SMTP_FROM"), Timeout: 20 * time.Second})
	previewEnterprise := strings.TrimSpace(os.Getenv("SMTP_PREVIEW_ENTERPRISE"))
	if previewEnterprise == "" {
		previewEnterprise = "VentureERP"
	}
	previewRecipient := strings.TrimSpace(os.Getenv("SMTP_PREVIEW_RECIPIENT"))
	if previewRecipient == "" {
		previewRecipient = "Equipe VentureERP"
	}
	publicURL := os.Getenv("ERP_PUBLIC_URL")
	if publicURL == "" {
		publicURL = "https://erp.venturerp.com"
	}
	for index, item := range previews {
		payload, _ := json.Marshal(previewPayload(item.key, item.name))
		html, text := RenderDelivery(previewEnterprise, "#1F4E78", "", previewRecipient, item.name, publicURL, payload)
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		err = service.SendMessage(ctx, ports.EmailMessage{MessageID: "<preview-" + strings.ToLower(item.key) + "@venturerp.com>", To: []string{to}, Subject: "[PRÉVIA " + fmtPreviewIndex(index+1, len(previews)) + "] " + item.name, Text: text, HTML: html})
		cancel()
		if err != nil {
			t.Fatalf("prévia %d/%d (%s): %v", index+1, len(previews), item.key, err)
		}
		t.Logf("enviada %d/%d: %s", index+1, len(previews), item.key)
		time.Sleep(1100 * time.Millisecond)
	}
}

func previewPayload(key, name string) map[string]any {
	p := map[string]any{"descricao": "O VentureERP identificou “" + name + "” e reuniu abaixo os dados necessários para a equipe avaliar e agir.", "event_key": key, "empresa": "Indústria Demonstração Ltda.", "responsavel": "Equipe responsável da área", "identificado_em": time.Now().UTC().Format(time.RFC3339), "situacao": "Requer conferência"}
	dailyReminderPreviews := map[string]bool{
		"ESTOQUE_ABAIXO_MINIMO":                 true,
		"ESTOQUE_NEGATIVO":                      true,
		"FISCAL_NFE_SAIDA_REJEITADA":            true,
		"FISCAL_NFE_ENTRADA_DIVERGENCIA_FISCAL": true,
		"PRODUCAO_OF_ATRASADA_PARADA":           true,
		"PRODUCAO_FALTA_MATERIAL_OF_LIBERADA":   true,
		"COMPRAS_PEDIDO_ATRASADO":               true,
		"FINANCEIRO_TITULO_VENCIDO":             true,
		"QUALIDADE_NAO_CONFORMIDADE_CRITICA":    true,
	}
	if dailyReminderPreviews[key] {
		p["_reminder_daily"] = true
	}
	switch {
	case strings.HasPrefix(key, "COMERCIAL_"):
		p["pedido"] = map[string]any{"numero": "PV-001284", "status": "LIBERADO", "cliente": "Metalúrgica Exemplo Ltda.", "emissao": "14/08/2026", "entrega": "21/08/2026", "total_liquido": 48750.90}
		p["itens"] = []any{map[string]any{"codigo": "000184", "descricao": "Conjunto estrutural sob medida", "um": "UN", "quantidade": 12, "preco_unitario": 3250.50, "total_liquido": 39006.00}, map[string]any{"codigo": "000219", "descricao": "Kit de fixação", "um": "KIT", "quantidade": 12, "preco_unitario": 812.075, "total_liquido": 9744.90}}
	case strings.HasPrefix(key, "ESTOQUE_"):
		p["item"] = map[string]any{"codigo": "MP-000184", "codigo_interno": 184, "descricao": "Chapa aço carbono 3,00 mm", "mascara": "3000x1200", "unidade_estoque": "KG", "unidade_compra": "KG", "classe_abc": "A", "critico": true}
		p["estoque"] = map[string]any{"almoxarifado": "MP — Matéria-prima", "saldo_fisico": 128.5, "reservado": 84.25, "disponivel": 44.25, "minimo": 180, "seguranca": 120, "maximo": 600, "necessidade_reposicao": 135.75, "consumo_medio_mensal": 480, "cobertura_dias": 2.8, "custo_medio": 8.72, "valor_saldo": 1120.52, "ultima_movimentacao": time.Now().UTC().Format(time.RFC3339), "quantidade_esperada": "128.500000", "quantidade_contada": "119.250000", "divergencia": "-9.250000"}
		p["fornecedor_recomendado"] = map[string]any{"codigo": 204, "nome": "Aços Paraná Ltda.", "codigo_item_fornecedor": "CH-AC-3MM", "unidade_compra": "KG", "lead_time_dias": 7, "quantidade_embalagem": 50, "homologado": true, "bloqueado": false, "validade_cadastro": "31/12/2026"}
		p["compras_em_aberto"] = []any{map[string]any{"pedido": "PC-00491", "fornecedor": "Aços Paraná Ltda.", "quantidade_pendente": 100, "unidade": "KG", "previsao": "18/08/2026"}, map[string]any{"pedido": "PC-00504", "fornecedor": "Siderúrgica Exemplo", "quantidade_pendente": 75, "unidade": "KG", "previsao": "21/08/2026"}}
	case strings.HasPrefix(key, "FISCAL_"):
		p["documento"] = map[string]any{"numero": "12894", "serie": "1", "chave_final": "90123456", "situacao": "AGUARDANDO TRATATIVA", "emissao": "14/08/2026 09:42", "total": 48750.90}
		p["parte"] = "Metalúrgica Exemplo Ltda."
		p["natureza_operacao"] = "Venda de produção do estabelecimento"
		p["motivo"] = "Validação fiscal necessária antes de prosseguir"
	case strings.HasPrefix(key, "CADASTRO_"):
		p["item_codigo"] = "000184"
		p["descricao_item"] = "Conjunto estrutural configurável"
		p["tipo"] = "FABRICADO"
		p["criado_por"] = "Ana Souza"
		p["pendencias"] = []any{map[string]any{"campo": "Classificação fiscal", "situacao": "Não informada"}, map[string]any{"campo": "Parâmetros MRP", "situacao": "Incompletos"}}
	case strings.HasPrefix(key, "COMPRAS_"):
		p["pedido_compra"] = map[string]any{"numero": "PC-00491", "fornecedor": "Aços Paraná Ltda.", "emissao": "05/08/2026", "previsao": "12/08/2026", "total": 63200.00}
		p["dias_atraso"] = 2
		p["material_critico_para"] = "OF-008124"
	case strings.HasPrefix(key, "PRODUCAO_") || strings.HasPrefix(key, "MRP_") || strings.HasPrefix(key, "APS_"):
		p["ordem_fabricacao"] = map[string]any{"numero": "OF-008124", "item": "000184 — Conjunto estrutural", "quantidade": "12.000000", "entrega": "19/08/2026", "progresso": "42%"}
		p["operacao_atual"] = "Solda"
		p["centro_trabalho"] = "CT-SOLDA-02"
		p["horas_impacto"] = "6.5"
	case strings.HasPrefix(key, "FINANCEIRO_"):
		p["titulo"] = map[string]any{"numero": "REC-2026-01842", "cliente": "Metalúrgica Exemplo Ltda.", "vencimento": "13/08/2026", "valor_total": 48750.90, "saldo": 48750.90}
		p["dias_em_atraso"] = 1
		p["condicao_credito"] = "Limite comprometido"
	default:
		p["processo"] = "Rotina operacional monitorada"
		p["ultima_execucao"] = "14/08/2026 08:00"
		p["tentativas"] = 3
		p["erro_sanitizado"] = "A rotina não concluiu dentro do período esperado"
	}
	return p
}

func fmtPreviewIndex(current, total int) string {
	return fmt.Sprintf("%02d/%02d", current, total)
}
