package notification_uc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/google/uuid"
)

var ErrUnauthorized = errors.New("operação restrita ao administrador")

type Repository interface {
	ListEvents(context.Context) ([]notificationentity.Event, error)
	ListEligibleUsers(context.Context, int64) ([]notificationentity.EligibleUser, error)
	ListEligibleDepartments(context.Context, int64) ([]notificationentity.EligibleDepartment, error)
	GetSettings(context.Context, int64) (notificationentity.Settings, error)
	SaveSettings(context.Context, notificationentity.Settings, uuid.UUID) error
	ListSubscriptions(context.Context, int64) ([]notificationentity.Subscription, error)
	SaveSubscription(context.Context, notificationentity.Subscription, uuid.UUID) (uuid.UUID, error)
	DeleteSubscription(context.Context, int64, uuid.UUID) error
	ListRecords(context.Context, int64, string, int, int) ([]map[string]any, error)
	RetryDelivery(context.Context, int64, uuid.UUID, uuid.UUID) error
	EnqueueTest(context.Context, int64, uuid.UUID) error
	GetAlert(context.Context, int64, uuid.UUID) (map[string]any, error)
	CreateCycleCount(context.Context, notificationentity.CycleCount, uuid.UUID) (notificationentity.CycleCount, error)
	ListCycleCounts(context.Context, int64, int, int) ([]notificationentity.CycleCount, error)
	GetCycleCount(context.Context, int64, uuid.UUID) (notificationentity.CycleCount, error)
	TransitionCycleCount(context.Context, int64, uuid.UUID, uuid.UUID, notificationentity.CycleCountState, *string, string) (notificationentity.CycleCount, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func actor(ctx context.Context) (*security.AuthUser, uuid.UUID, error) {
	u, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok || u == nil || u.EnterpriseID <= 0 {
		return nil, uuid.Nil, ErrUnauthorized
	}
	id, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, uuid.Nil, ErrUnauthorized
	}
	return u, id, nil
}

func admin(ctx context.Context) (*security.AuthUser, uuid.UUID, error) {
	u, id, err := actor(ctx)
	if err != nil || u.Role != "ADMIN" {
		return nil, uuid.Nil, ErrUnauthorized
	}
	return u, id, nil
}

func (s *Service) ListEvents(ctx context.Context) ([]notificationentity.Event, error) {
	if _, _, err := actor(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx)
}
func (s *Service) ListEligibleUsers(ctx context.Context) ([]notificationentity.EligibleUser, error) {
	u, _, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListEligibleUsers(ctx, u.EnterpriseID)
}
func (s *Service) ListEligibleDepartments(ctx context.Context) ([]notificationentity.EligibleDepartment, error) {
	u, _, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListEligibleDepartments(ctx, u.EnterpriseID)
}
func (s *Service) GetSettings(ctx context.Context) (notificationentity.Settings, error) {
	u, _, err := admin(ctx)
	if err != nil {
		return notificationentity.Settings{}, err
	}
	return s.repo.GetSettings(ctx, u.EnterpriseID)
}
func (s *Service) SaveSettings(ctx context.Context, in notificationentity.Settings) error {
	u, id, err := admin(ctx)
	if err != nil {
		return err
	}
	in.EnterpriseID = u.EnterpriseID
	if err = in.Validate(); err != nil {
		return errors.Join(notificationentity.ErrValidation, err)
	}
	return s.repo.SaveSettings(ctx, in, id)
}
func (s *Service) ListSubscriptions(ctx context.Context) ([]notificationentity.Subscription, error) {
	u, _, err := admin(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSubscriptions(ctx, u.EnterpriseID)
}
func (s *Service) SaveSubscription(ctx context.Context, in notificationentity.Subscription) (uuid.UUID, error) {
	u, id, err := admin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	in.EnterpriseID = u.EnterpriseID
	if in.EventVersion == 0 {
		in.EventVersion = 1
	}
	if len(in.Thresholds) == 0 {
		in.Thresholds = json.RawMessage(`{}`)
	}
	if err = in.Validate(); err != nil {
		return uuid.Nil, errors.Join(notificationentity.ErrValidation, err)
	}
	return s.repo.SaveSubscription(ctx, in, id)
}
func (s *Service) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	u, _, err := admin(ctx)
	if err != nil {
		return err
	}
	return s.repo.DeleteSubscription(ctx, u.EnterpriseID, id)
}
func (s *Service) ListRecords(ctx context.Context, kind string, limit, offset int) ([]map[string]any, error) {
	u, _, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	if kind != "alerts" && u.Role != "ADMIN" {
		return nil, ErrUnauthorized
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListRecords(ctx, u.EnterpriseID, kind, limit, offset)
}
func (s *Service) Retry(ctx context.Context, deliveryID uuid.UUID) error {
	u, id, err := admin(ctx)
	if err != nil {
		return err
	}
	return s.repo.RetryDelivery(ctx, u.EnterpriseID, deliveryID, id)
}
func (s *Service) TestEmail(ctx context.Context) error {
	u, id, err := admin(ctx)
	if err != nil {
		return err
	}
	return s.repo.EnqueueTest(ctx, u.EnterpriseID, id)
}
func (s *Service) GetAlert(ctx context.Context, id uuid.UUID) (map[string]any, error) {
	u, _, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAlert(ctx, u.EnterpriseID, id)
}
func (s *Service) CreateCycleCount(ctx context.Context, in notificationentity.CycleCount) (notificationentity.CycleCount, error) {
	u, id, err := actor(ctx)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	in.EnterpriseID = u.EnterpriseID
	if err = in.ValidateSchedule(); err != nil {
		return notificationentity.CycleCount{}, errors.Join(notificationentity.ErrValidation, err)
	}
	return s.repo.CreateCycleCount(ctx, in, id)
}
func (s *Service) ListCycleCounts(ctx context.Context, limit, offset int) ([]notificationentity.CycleCount, error) {
	u, _, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListCycleCounts(ctx, u.EnterpriseID, limit, offset)
}
func (s *Service) GetCycleCount(ctx context.Context, id uuid.UUID) (notificationentity.CycleCount, error) {
	u, _, err := actor(ctx)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	return s.repo.GetCycleCount(ctx, u.EnterpriseID, id)
}
func (s *Service) TransitionCycleCount(ctx context.Context, id uuid.UUID, target notificationentity.CycleCountState, counted *string, reason string) (notificationentity.CycleCount, error) {
	u, actorID, err := actor(ctx)
	if err != nil {
		return notificationentity.CycleCount{}, err
	}
	if target == notificationentity.CycleApproved && u.Role != "ADMIN" {
		return notificationentity.CycleCount{}, ErrUnauthorized
	}
	return s.repo.TransitionCycleCount(ctx, u.EnterpriseID, id, actorID, target, counted, reason)
}

func LocalDate(now time.Time, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc), nil
}

func NextDigestAfter(now time.Time, digestTime, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}
	clock, err := time.Parse("15:04", digestTime)
	if err != nil {
		return time.Time{}, err
	}
	local := now.In(loc)
	candidate := time.Date(local.Year(), local.Month(), local.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
	if !candidate.After(local) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate.UTC(), nil
}
