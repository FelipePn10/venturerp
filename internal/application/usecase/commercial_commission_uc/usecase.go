package commercial_commission_uc

import (
	"context"
	"errors"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/commercial_commission/entity"
	commissionrepo "github.com/FelipePn10/panossoerp/internal/domain/commercial_commission/repository"
	"strconv"
	"strings"
	"time"
)

type UseCase struct {
	Repo commissionrepo.Repository
	Auth ports.AuthService
}
type SettingsDTO struct {
	CompetenceEvent string `json:"competence_event"`
	InvoiceSharePct string `json:"invoice_share_pct"`
	ReceiptSharePct string `json:"receipt_share_pct"`
}
type TransitionDTO struct {
	Reason           string  `json:"reason"`
	PaymentReference *string `json:"payment_reference,omitempty"`
}

func (uc *UseCase) identity(ctx context.Context) (int64, error) {
	if uc.Auth == nil {
		return 0, errorsuc.ErrUnauthorized
	}
	tenant, err := uc.Auth.EnterpriseID(ctx)
	if err != nil || tenant <= 0 {
		return 0, errorsuc.ErrUnauthorized
	}
	return tenant, nil
}
func (uc *UseCase) UpsertSettings(ctx context.Context, d SettingsDTO) (*entity.Settings, error) {
	tenant, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	d.CompetenceEvent = strings.ToUpper(strings.TrimSpace(d.CompetenceEvent))
	if d.CompetenceEvent != "FATURAMENTO" && d.CompetenceEvent != "RECEBIMENTO" && d.CompetenceEvent != "RATEIO" {
		return nil, errorsuc.NewValidationError("competence_event deve ser FATURAMENTO, RECEBIMENTO ou RATEIO")
	}
	invoice, e1 := strconv.ParseFloat(d.InvoiceSharePct, 64)
	receipt, e2 := strconv.ParseFloat(d.ReceiptSharePct, 64)
	if e1 != nil || e2 != nil || invoice < 0 || receipt < 0 || invoice+receipt != 100 {
		return nil, errorsuc.NewValidationError("os percentuais de faturamento e recebimento devem ser válidos e totalizar 100")
	}
	return uc.Repo.UpsertSettings(ctx, tenant, &entity.Settings{CompetenceEvent: d.CompetenceEvent, InvoiceSharePct: d.InvoiceSharePct, ReceiptSharePct: d.ReceiptSharePct, UpdatedBy: actor})
}
func (uc *UseCase) GetSettings(ctx context.Context) (*entity.Settings, error) {
	tenant, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Repo.GetSettings(ctx, tenant)
}
func (uc *UseCase) List(ctx context.Context, f entity.Filter) ([]*entity.LedgerEntry, error) {
	tenant, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}
	if f.Limit > 500 {
		f.Limit = 500
	}
	if f.Offset < 0 {
		return nil, errorsuc.NewValidationError("offset não pode ser negativo")
	}
	return uc.Repo.List(ctx, tenant, f)
}
func (uc *UseCase) Transition(ctx context.Context, code int64, action, key string, d TransitionDTO) (*entity.LedgerEntry, error) {
	tenant, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	action = strings.ToUpper(strings.TrimSpace(action))
	key = strings.TrimSpace(key)
	if code <= 0 || key == "" {
		return nil, errorsuc.NewValidationError("comissão e chave de idempotência são obrigatórias")
	}
	if strings.TrimSpace(d.Reason) == "" {
		return nil, errorsuc.NewValidationError("o motivo é obrigatório")
	}
	value, err := uc.Repo.Transition(ctx, tenant, entity.TransitionCommand{Code: code, Action: action, Reason: strings.TrimSpace(d.Reason), PaymentReference: d.PaymentReference, IdempotencyKey: key, ActorID: actor})
	if errors.Is(err, commissionrepo.ErrInvalidState) {
		return nil, errorsuc.NewConflictError("a situação atual da comissão não permite esta operação")
	}
	if errors.Is(err, commissionrepo.ErrIdempotencyConflict) {
		return nil, errorsuc.NewConflictError("a chave de idempotência já foi usada em outra operação")
	}
	return value, err
}
func ParseDate(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	v, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, errorsuc.NewValidationError("as datas devem usar o formato AAAA-MM-DD")
	}
	return &v, nil
}
