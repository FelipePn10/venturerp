package ports

import (
	"context"
	"errors"
	"fmt"
	"time"

	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
)

type EmailMessage struct {
	EnterpriseID int64
	MessageID    string
	FromName     string
	To           []string
	Subject      string
	HTML         string
	Text         string
	Attachments  []EmailAttachment
}

type EmailAttachment struct {
	FileName string
	MIMEType string
	Content  []byte
	SHA256   string
}

type EmailProvider interface {
	Send(ctx context.Context, message EmailMessage) error
}

type EmailFailureClass string

const (
	EmailFailureTemporary EmailFailureClass = "TRANSITORIA"
	EmailFailurePermanent EmailFailureClass = "PERMANENTE"
)

type EmailDeliveryError struct {
	Class EmailFailureClass
	Code  string
	Err   error
}

func (e *EmailDeliveryError) Error() string { return fmt.Sprintf("%s: %v", e.Code, e.Err) }
func (e *EmailDeliveryError) Unwrap() error { return e.Err }
func FailureClass(err error) EmailFailureClass {
	var deliveryErr *EmailDeliveryError
	if errors.As(err, &deliveryErr) {
		return deliveryErr.Class
	}
	return EmailFailureTemporary
}

type NotificationOutboxWriter interface {
	Enqueue(ctx context.Context, event notificationentity.OutboxEvent) error
}

type NotificationWorkerRepository interface {
	ClaimOutbox(ctx context.Context, owner string, limit int, lease time.Duration) ([]notificationentity.OutboxEvent, error)
	ProcessOutbox(ctx context.Context, event notificationentity.OutboxEvent) error
	FailOutbox(ctx context.Context, id string, attempt int, next time.Time, code, sanitizedMessage string, discard bool) error
}
