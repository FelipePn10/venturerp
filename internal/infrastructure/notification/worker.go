package notification

import (
	"context"
	"log/slog"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Worker struct {
	repo            *notificationrepo.Repository
	provider        ports.EmailProvider
	owner           string
	interval, lease time.Duration
}

func NewWorker(repo *notificationrepo.Repository, provider ports.EmailProvider, owner string) *Worker {
	return &Worker{repo: repo, provider: provider, owner: owner, interval: 5 * time.Second, lease: 2 * time.Minute}
}
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		w.drain(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w *Worker) drain(ctx context.Context) {
	ctx, span := otel.Tracer("venturerp/notifications").Start(ctx, "notification.worker.drain")
	defer span.End()
	if err := w.repo.ScheduleDigests(ctx); err != nil {
		span.RecordError(err)
		slog.Error("falha ao agendar resumos de notificações", "erro", err)
	}
	if err := w.repo.SchedulePolicyCycleCounts(ctx); err != nil {
		span.RecordError(err)
		slog.Error("falha ao programar contagens cíclicas por política de item", "erro", err)
	}
	if err := w.repo.ScheduleOperationalAlerts(ctx); err != nil {
		span.RecordError(err)
		slog.Error("falha ao avaliar alertas operacionais", "erro", err)
	}
	if err := w.repo.SchedulePendingReminders(ctx); err != nil {
		span.RecordError(err)
		slog.Error("falha ao agendar lembretes diários de pendências", "erro", err)
	}
	events, err := w.repo.ClaimOutbox(ctx, w.owner, 50, w.lease)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "outbox claim failed")
		slog.Error("falha ao obter outbox de notificações", "erro", err)
		return
	}
	for _, e := range events {
		if err = w.repo.ProcessOutbox(ctx, e); err != nil {
			attempt := e.Attempts + 1
			_ = w.repo.FailOutbox(ctx, e.ID.String(), attempt, time.Now().Add(notificationentity.RetryDelay(attempt)), "PROCESSAMENTO", err.Error(), attempt >= 6)
		}
	}
	deliveries, err := w.repo.ClaimDeliveries(ctx, w.owner, 50, w.lease)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delivery claim failed")
		slog.Error("falha ao obter entregas de notificações", "erro", err)
		return
	}
	for _, d := range deliveries {
		permit, permitErr := w.repo.AcquireProviderPermit(ctx, d.EnterpriseID, w.owner)
		if permitErr != nil {
			slog.Error("falha ao aplicar proteção do provedor", "erro", permitErr)
			continue
		}
		if !permit.Allowed {
			_ = w.repo.DeferDelivery(ctx, d.EnterpriseID, d.ID, permit.RetryAt, permit.Reason)
			continue
		}
		html, text := RenderDelivery(d.EnterpriseName, d.BrandColor, d.LogoDataURI, d.RecipientName, d.Subject, "", d.Payload)
		sendErr := w.provider.Send(ctx, ports.EmailMessage{EnterpriseID: d.EnterpriseID, MessageID: d.MessageID, To: []string{d.Recipient}, Subject: d.Subject, HTML: html, Text: text, Attachments: d.Attachments})
		_ = w.repo.RecordProviderResult(ctx, sendErr == nil)
		if err = w.repo.CompleteDelivery(ctx, d, sendErr); err != nil {
			slog.Error("falha ao registrar entrega de notificação", "entrega_id", d.ID, "erro", err)
		}
	}
	if err := w.repo.Cleanup(ctx); err != nil {
		slog.Error("falha na retenção de notificações", "erro", err)
	}
}
