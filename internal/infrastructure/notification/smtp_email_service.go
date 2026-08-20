package notification

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

type SMTPConfig struct {
	Host, Port, User, Password, From string
	Timeout                          time.Duration
}
type EmailService struct{ cfg SMTPConfig }

func NewEmailService(cfg SMTPConfig) *EmailService {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &EmailService{cfg: cfg}
}
func (s *EmailService) Enabled() bool {
	return s.cfg.Host != "" && s.cfg.User != "" && s.cfg.From != ""
}

// Send mantém a compatibilidade do alerta manual do MRP durante a migração.
func (s *EmailService) Send(to []string, subject, body string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Timeout)
	defer cancel()
	return s.SendMessage(ctx, ports.EmailMessage{MessageID: "<" + time.Now().UTC().Format("20060102150405.000000000") + "@venturerp.local>", To: to, Subject: subject, Text: body})
}
func (s *EmailService) SendMessage(ctx context.Context, m ports.EmailMessage) error {
	if !s.Enabled() {
		return &ports.EmailDeliveryError{Class: ports.EmailFailurePermanent, Code: "PROVEDOR_NAO_CONFIGURADO", Err: errors.New("provedor central de e-mail não configurado")}
	}
	raw, err := BuildMIME(s.cfg.From, m)
	if err != nil {
		return &ports.EmailDeliveryError{Class: ports.EmailFailurePermanent, Code: "MENSAGEM_INVALIDA", Err: err}
	}
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	dialer := net.Dialer{Timeout: s.cfg.Timeout}
	tlsCfg := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	if s.cfg.Port == "465" {
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return &ports.EmailDeliveryError{Class: ports.EmailFailureTemporary, Code: "CONEXAO_SMTP", Err: err}
	}
	defer conn.Close()
	deadline := time.Now().Add(s.cfg.Timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err = conn.SetDeadline(deadline); err != nil {
		return &ports.EmailDeliveryError{Class: ports.EmailFailureTemporary, Code: "TIMEOUT_SMTP", Err: err}
	}
	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return &ports.EmailDeliveryError{Class: ports.EmailFailureTemporary, Code: "CLIENTE_SMTP", Err: err}
	}
	defer c.Close()
	if s.cfg.Port != "465" {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return &ports.EmailDeliveryError{Class: ports.EmailFailurePermanent, Code: "TLS_INDISPONIVEL", Err: errors.New("provedor SMTP não oferece TLS")}
		}
		if err = c.StartTLS(tlsCfg); err != nil {
			return &ports.EmailDeliveryError{Class: ports.EmailFailureTemporary, Code: "TLS_SMTP", Err: err}
		}
	}
	if err = c.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)); err != nil {
		return &ports.EmailDeliveryError{Class: ports.EmailFailurePermanent, Code: "AUTENTICACAO_SMTP", Err: err}
	}
	if err = c.Mail(s.cfg.From); err != nil {
		return &ports.EmailDeliveryError{Class: ports.EmailFailurePermanent, Code: "REMETENTE_SMTP", Err: err}
	}
	for range m.To { /* validation precedes protocol */
	}
	for _, to := range m.To {
		if err = c.Rcpt(to); err != nil {
			return &ports.EmailDeliveryError{Class: ports.EmailFailurePermanent, Code: "DESTINATARIO_RECUSADO", Err: err}
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("dados SMTP: %w", err)
	}
	if _, err = w.Write(raw); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

type CentralEmailProvider struct{ service *EmailService }

func NewCentralEmailProvider(service *EmailService) *CentralEmailProvider {
	return &CentralEmailProvider{service: service}
}
func (p *CentralEmailProvider) Send(ctx context.Context, m ports.EmailMessage) error {
	return p.service.SendMessage(ctx, m)
}

func safeHeader(v string) error {
	if strings.ContainsAny(v, "\r\n") || strings.TrimSpace(v) == "" {
		return errors.New("cabeçalho de e-mail inválido")
	}
	return nil
}
func BuildMIME(from string, m ports.EmailMessage) ([]byte, error) {
	if err := safeHeader(from); err != nil {
		return nil, err
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, errors.New("remetente inválido")
	}
	if err := safeHeader(m.Subject); err != nil {
		return nil, err
	}
	if err := safeHeader(m.MessageID); err != nil {
		return nil, err
	}
	if len(m.To) == 0 {
		return nil, errors.New("destinatário obrigatório")
	}
	for _, to := range m.To {
		if err := safeHeader(to); err != nil {
			return nil, err
		}
		if _, err := mail.ParseAddress(to); err != nil {
			return nil, errors.New("destinatário inválido")
		}
	}
	digest := sha256.Sum256([]byte(m.MessageID))
	boundary := "venturerp-alternative-" + hex.EncodeToString(digest[:12])
	outerBoundary := "venturerp-mixed-" + hex.EncodeToString(digest[12:24])
	for _, attachment := range m.Attachments {
		if err := validateAttachment(attachment); err != nil {
			return nil, err
		}
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: =?UTF-8?B?%s?=\r\nMessage-ID: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\n", from, strings.Join(m.To, ", "), base64.StdEncoding.EncodeToString([]byte(m.Subject)), m.MessageID, time.Now().UTC().Format(time.RFC1123Z))
	if len(m.Attachments) > 0 {
		fmt.Fprintf(&b, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n--%s\r\n", outerBoundary, outerBoundary)
	}
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n", boundary)
	writeBase64(&b, []byte(m.Text))
	fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n", boundary)
	writeBase64(&b, []byte(m.HTML))
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	for _, attachment := range m.Attachments {
		fmt.Fprintf(&b, "\r\n--%s\r\nContent-Type: %s\r\nContent-Disposition: attachment; filename=%q\r\nContent-Transfer-Encoding: base64\r\n\r\n", outerBoundary, attachment.MIMEType, mime.QEncoding.Encode("UTF-8", attachment.FileName))
		writeBase64(&b, attachment.Content)
	}
	if len(m.Attachments) > 0 {
		fmt.Fprintf(&b, "--%s--\r\n", outerBoundary)
	}
	return b.Bytes(), nil
}

func validateAttachment(attachment ports.EmailAttachment) error {
	if err := safeHeader(attachment.FileName); err != nil {
		return errors.New("nome de anexo inválido")
	}
	allowed := map[string]bool{"application/pdf": true, "text/csv": true, "image/png": true, "image/jpeg": true}
	if !allowed[attachment.MIMEType] {
		return errors.New("tipo de anexo não permitido")
	}
	if len(attachment.Content) == 0 || len(attachment.Content) > 25*1024*1024 {
		return errors.New("tamanho de anexo inválido")
	}
	sum := sha256.Sum256(attachment.Content)
	if attachment.SHA256 != "" && attachment.SHA256 != hex.EncodeToString(sum[:]) {
		return errors.New("hash de anexo inválido")
	}
	return nil
}

func writeBase64(b *bytes.Buffer, content []byte) {
	encoded := base64.StdEncoding.EncodeToString(content)
	for len(encoded) > 76 {
		b.WriteString(encoded[:76])
		b.WriteString("\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded)
	b.WriteString("\r\n")
}
