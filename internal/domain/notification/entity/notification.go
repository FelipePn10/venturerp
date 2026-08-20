package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Severity string
type Cadence string
type AlertState string
type DeliveryState string
type RecipientType string
type Channel string
type ProducerStatus string

const (
	SeverityInformational Severity       = "INFORMATIVO"
	SeverityAttention     Severity       = "ATENCAO"
	SeverityCritical      Severity       = "CRITICO"
	CadenceImmediate      Cadence        = "IMEDIATO"
	CadenceDailyDigest    Cadence        = "RESUMO_DIARIO"
	CadenceBoth           Cadence        = "IMEDIATO_E_RESUMO_DIARIO"
	AlertOpen             AlertState     = "ABERTO"
	AlertResolved         AlertState     = "RESOLVIDO"
	AlertIgnored          AlertState     = "IGNORADO"
	DeliveryPending       DeliveryState  = "PENDENTE"
	DeliveryProcessing    DeliveryState  = "PROCESSANDO"
	DeliverySent          DeliveryState  = "ENVIADO"
	DeliveryFailed        DeliveryState  = "FALHOU"
	DeliveryDiscarded     DeliveryState  = "DESCARTADO"
	DeliveryCancelled     DeliveryState  = "CANCELADO"
	RecipientUser         RecipientType  = "USUARIO"
	RecipientRole         RecipientType  = "PAPEL"
	RecipientDepartment   RecipientType  = "DEPARTAMENTO"
	ChannelEmail          Channel        = "EMAIL"
	ChannelInternal       Channel        = "NOTIFICACAO_INTERNA"
	ChannelWebhook        Channel        = "WEBHOOK"
	ProducerActive        ProducerStatus = "ATIVO"
	ProducerFuture        ProducerStatus = "FUTURO"
)

var (
	ErrNotFound   = errors.New("recurso de notificação não encontrado")
	ErrConflict   = errors.New("conflito de estado da notificação")
	ErrValidation = errors.New("regra de notificação inválida")
)

type Event struct {
	Key                     string         `json:"event_key"`
	Version                 int            `json:"version"`
	Name                    string         `json:"name"`
	Description             string         `json:"description"`
	Module                  string         `json:"module"`
	EventKind               string         `json:"event_kind"`
	Severity                Severity       `json:"severity"`
	AllowedCadences         []Cadence      `json:"allowed_cadences"`
	TemplateKey             string         `json:"template_key"`
	EnabledByDefault        bool           `json:"enabled_by_default"`
	SuggestedRecipientRoles []string       `json:"suggested_recipient_roles"`
	ProducerStatus          ProducerStatus `json:"producer_status"`
	ProducerDescription     string         `json:"producer_description"`
}

type EligibleUser struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Role   string    `json:"role"`
	Active bool      `json:"active"`
}

type EligibleDepartment struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

type Settings struct {
	EnterpriseID       int64  `json:"enterprise_id"`
	Enabled            bool   `json:"enabled"`
	DigestTime         string `json:"digest_time"`
	Timezone           string `json:"timezone"`
	RetentionDays      int    `json:"retention_days"`
	MaxAttachmentBytes int64  `json:"max_attachment_bytes"`
	MaxEmailsPerMinute int    `json:"max_emails_per_minute"`
	FiscalConfigID     *int64 `json:"fiscal_config_id,omitempty"`
}

func (s Settings) Validate() error {
	if s.EnterpriseID <= 0 {
		return errors.New("empresa inválida")
	}
	if _, err := time.Parse("15:04", s.DigestTime); err != nil {
		return errors.New("horário deve usar HH:MM")
	}
	if _, err := time.LoadLocation(s.Timezone); err != nil {
		return fmt.Errorf("fuso IANA inválido: %w", err)
	}
	if s.RetentionDays < 30 || s.RetentionDays > 3650 {
		return errors.New("retenção deve estar entre 30 e 3650 dias")
	}
	if s.MaxAttachmentBytes < 0 || s.MaxAttachmentBytes > 25*1024*1024 {
		return errors.New("limite de anexos inválido")
	}
	if s.MaxEmailsPerMinute < 1 || s.MaxEmailsPerMinute > 1000 {
		return errors.New("limite de e-mails por minuto inválido")
	}
	return nil
}

type Recipient struct {
	Type   RecipientType `json:"recipient_type"`
	UserID *uuid.UUID    `json:"user_id,omitempty"`
	Key    string        `json:"recipient_key,omitempty"`
}

type Subscription struct {
	ID           uuid.UUID       `json:"id"`
	EnterpriseID int64           `json:"enterprise_id"`
	EventKey     string          `json:"event_key"`
	EventVersion int             `json:"event_version"`
	Enabled      bool            `json:"enabled"`
	Cadence      Cadence         `json:"cadence"`
	Thresholds   json.RawMessage `json:"thresholds"`
	Recipients   []Recipient     `json:"recipients"`
}

func (s Subscription) Validate() error {
	if s.EnterpriseID <= 0 || strings.TrimSpace(s.EventKey) == "" {
		return errors.New("empresa e evento são obrigatórios")
	}
	if s.Cadence != CadenceImmediate && s.Cadence != CadenceDailyDigest && s.Cadence != CadenceBoth {
		return errors.New("cadência inválida")
	}
	if len(s.Thresholds) > 0 {
		var thresholds map[string]json.RawMessage
		if err := json.Unmarshal(s.Thresholds, &thresholds); err != nil {
			return errors.New("limiares devem ser um objeto JSON válido")
		}
		for _, key := range []string{"antecedencia_dias", "quantidade_limite", "valor_limite", "tolerancia_quantidade"} {
			if raw, ok := thresholds[key]; ok && !validNonNegativeDecimal(raw) {
				return fmt.Errorf("limiar %s inválido", key)
			}
		}
		for _, key := range []string{"horario_inicio", "horario_fim"} {
			if raw, ok := thresholds[key]; ok {
				var value string
				if json.Unmarshal(raw, &value) != nil {
					return fmt.Errorf("limiar %s inválido", key)
				}
				if _, err := time.Parse("15:04", value); err != nil {
					return fmt.Errorf("limiar %s inválido", key)
				}
			}
		}
		for _, key := range []string{"tolerancia_por_item", "tolerancia_por_almoxarifado"} {
			if raw, ok := thresholds[key]; ok {
				var overrides map[string]json.RawMessage
				if json.Unmarshal(raw, &overrides) != nil {
					return fmt.Errorf("limiar %s inválido", key)
				}
				for _, value := range overrides {
					if !validNonNegativeDecimal(value) {
						return fmt.Errorf("limiar %s inválido", key)
					}
				}
			}
		}
	}
	if len(s.Recipients) == 0 {
		return errors.New("ao menos um destinatário interno é obrigatório")
	}
	for _, r := range s.Recipients {
		switch r.Type {
		case RecipientUser:
			if r.UserID == nil || *r.UserID == uuid.Nil || r.Key != "" {
				return errors.New("destinatário USUARIO inválido")
			}
		case RecipientRole, RecipientDepartment:
			if r.UserID != nil || strings.TrimSpace(r.Key) == "" {
				return errors.New("papel/departamento inválido")
			}
		default:
			return errors.New("tipo de destinatário inválido")
		}
	}
	return nil
}

func validNonNegativeDecimal(raw json.RawMessage) bool {
	value := strings.Trim(string(raw), `"`)
	d, err := decimal.NewFromString(value)
	return err == nil && !d.IsNegative()
}

type OutboxEvent struct {
	ID                  uuid.UUID
	EnterpriseID        int64
	EventKey            string
	EventVersion        int
	AggregateType       string
	AggregateInternalID string
	AggregatePublicID   string
	Payload             json.RawMessage
	DeduplicationKey    string
	CorrelationID       uuid.UUID
	OriginatorUserID    *uuid.UUID
	OccurredAt          time.Time
	Attempts            int
}

func RetryDelay(attempt int) time.Duration {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 24 * time.Hour}
	if attempt <= 0 {
		return delays[0]
	}
	if attempt > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempt-1]
}
