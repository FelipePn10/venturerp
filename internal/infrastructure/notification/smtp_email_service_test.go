package notification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

func TestBuildMIMEAlternativeAndInjection(t *testing.T) {
	m := ports.EmailMessage{MessageID: "<stable@example>", To: []string{"interno@example.com"}, Subject: "Alerta ç", Text: "texto", HTML: "<b>html</b>"}
	raw, err := BuildMIME("erp@example.com", m)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{"multipart/alternative", "Message-ID: <stable@example>", "Content-Type: text/plain", "Content-Type: text/html"} {
		if !strings.Contains(s, want) {
			t.Fatalf("MIME sem %q", want)
		}
	}
	m.Subject = "ok\r\nBcc: externo@example.com"
	if _, err = BuildMIME("erp@example.com", m); err == nil {
		t.Fatal("header injection aceita")
	}
}

func TestBuildMIMEAttachmentAndLimits(t *testing.T) {
	content := []byte("conteúdo interno validado")
	sum := sha256.Sum256(content)
	m := ports.EmailMessage{MessageID: "<attachment@example>", To: []string{"interno@example.com"}, Subject: "Relatório", Text: strings.Repeat("texto", 100), HTML: "<b>html</b>", Attachments: []ports.EmailAttachment{{FileName: "relatório.pdf", MIMEType: "application/pdf", Content: content, SHA256: hex.EncodeToString(sum[:])}}}
	raw, err := BuildMIME("erp@example.com", m)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "multipart/mixed") || !strings.Contains(s, "Content-Disposition: attachment") {
		t.Fatal("anexo MIME ausente")
	}
	for _, line := range strings.Split(s, "\r\n") {
		if len(line) > 998 {
			t.Fatalf("linha MIME excede RFC 5322: %d", len(line))
		}
	}
	m.Attachments[0].MIMEType = "application/x-executable"
	if _, err = BuildMIME("erp@example.com", m); err == nil {
		t.Fatal("tipo de anexo perigoso aceito")
	}
	m.Attachments[0].MIMEType = "application/pdf"
	m.Attachments[0].SHA256 = strings.Repeat("0", 64)
	if _, err = BuildMIME("erp@example.com", m); err == nil {
		t.Fatal("hash inválido aceito")
	}
}

func TestSMTPRespectsCancelledContext(t *testing.T) {
	service := NewEmailService(SMTPConfig{Host: "127.0.0.1", Port: "1", User: "user", Password: "secret", From: "erp@example.com"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := service.SendMessage(ctx, ports.EmailMessage{MessageID: "<cancel@example>", To: []string{"interno@example.com"}, Subject: "Teste", Text: "texto"})
	if err == nil {
		t.Fatal("contexto cancelado ignorado")
	}
	if ports.FailureClass(err) != ports.EmailFailureTemporary {
		t.Fatalf("classe inesperada: %s", ports.FailureClass(err))
	}
}
func TestRenderDeliveryEscapesDynamicHTML(t *testing.T) {
	html, _ := RenderDelivery("Empresa <x>", "#112233", "", "Usuário", "Título", "https://erp.example.com", []byte(`{"descricao":"<script>alert(1)</script>","link":"/stock"}`))
	if strings.Contains(html, "<script>") {
		t.Fatal("HTML dinâmico não escapado")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("conteúdo escapado ausente")
	}
	if strings.Contains(html, "https://erp.example.com/stock") {
		t.Fatal("template desktop expôs deep link web")
	}
	if !strings.Contains(html, "Consultar no ERP Desktop") {
		t.Fatal("orientação para aplicativo desktop ausente")
	}
}

func TestRenderDeliveryShowsOperationalData(t *testing.T) {
	payload, _ := json.Marshal(previewPayload("ESTOQUE_CONTAGEM_DIVERGENCIA", "Divergência de contagem"))
	html, text := RenderDelivery("VentureERP", "#000000", "", "Equipe de estoque", "Divergência de contagem", "", payload)
	for _, expected := range []string{"Leitura rápida", "Impacto operacional", "Próxima ação", "Quantidade esperada", "128.500000", "Estoque › Alertas e Contagens", "Decisão de abastecimento e disponibilidade", "max-width:980px"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("HTML sem informação operacional %q", expected)
		}
	}
	if !strings.Contains(text, "ONDE CONSULTAR") || !strings.Contains(text, "-9.250000") {
		t.Fatal("alternativa texto incompleta")
	}
}

func TestRenderDeliveryIdentifiesDailyPendingReminder(t *testing.T) {
	html, _ := RenderDelivery("VentureERP", "", "", "Equipe", "Estoque abaixo do mínimo", "", []byte(`{"_event_key":"ESTOQUE_ABAIXO_MINIMO","_reminder_daily":true,"descricao":"A pendência continua aberta."}`))
	for _, expected := range []string{"Lembrete diário", "Pendência ainda aberta", "A pendência continua aberta."} {
		if !strings.Contains(html, expected) {
			t.Fatalf("lembrete diário sem identificação %q", expected)
		}
	}
}
