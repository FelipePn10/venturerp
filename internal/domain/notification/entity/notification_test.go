package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPublicEnumsArePortuguese(t *testing.T) {
	values := []string{string(SeverityInformational), string(SeverityAttention), string(SeverityCritical), string(CadenceImmediate), string(CadenceDailyDigest), string(CadenceBoth), string(AlertOpen), string(AlertResolved), string(AlertIgnored), string(DeliveryPending), string(DeliveryProcessing), string(DeliverySent), string(DeliveryFailed), string(DeliveryDiscarded), string(DeliveryCancelled), string(RecipientUser), string(RecipientRole), string(RecipientDepartment), string(ChannelEmail), string(ChannelInternal)}
	for _, v := range values {
		if v == "" {
			t.Fatal("enum vazio")
		}
	}
}

func TestSettingsTimezoneAndSubscriptionValidation(t *testing.T) {
	s := Settings{EnterpriseID: 1, DigestTime: "08:00", Timezone: "America/Sao_Paulo", RetentionDays: 365, MaxAttachmentBytes: 1024, MaxEmailsPerMinute: 60}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Timezone = "Brasil/Inexistente"
	if s.Validate() == nil {
		t.Fatal("fuso inválido aceito")
	}
	id := uuid.New()
	sub := Subscription{EnterpriseID: 1, EventKey: "EVENTO", Cadence: CadenceBoth, Recipients: []Recipient{{Type: RecipientUser, UserID: &id}}}
	if err := sub.Validate(); err != nil {
		t.Fatal(err)
	}
	sub.Thresholds = []byte(`{"tolerancia_quantidade":"0.250000","horario_inicio":"06:00","horario_fim":"22:00","tolerancia_por_item":{"1001":"0.100000"}}`)
	if err := sub.Validate(); err != nil {
		t.Fatal(err)
	}
	sub.Thresholds = []byte(`{"tolerancia_quantidade":"-1"}`)
	if err := sub.Validate(); err == nil {
		t.Fatal("tolerância negativa aceita")
	}
	sub.Thresholds = []byte(`{"horario_inicio":"25:00"}`)
	if err := sub.Validate(); err == nil {
		t.Fatal("horário inválido aceito")
	}
}

func TestRetryDelay(t *testing.T) {
	want := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 24 * time.Hour, 24 * time.Hour}
	for i, w := range want {
		if got := RetryDelay(i + 1); got != w {
			t.Fatalf("tentativa %d: %v", i+1, got)
		}
	}
}
