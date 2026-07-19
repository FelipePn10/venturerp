package consumer_service_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
	csrepo "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository"
	"github.com/google/uuid"
)

type csAllowAuth struct{ ports.AuthService }

func (csAllowAuth) CanManageTechnicalAssistance(context.Context) bool { return true }
func (csAllowAuth) EnterpriseID(context.Context) (int64, error)       { return 1, nil }

type fakeCSRepo struct {
	callTypes map[int64]*entity.CallType
	consumer  *entity.Consumer
	call      *entity.Call
	tenantID  int64
}

func (f *fakeCSRepo) NextConsumerCode(context.Context) (int64, error)      { return 1, nil }
func (f *fakeCSRepo) NextCallNumber(context.Context, int64) (int64, error) { return 1, nil }
func (f *fakeCSRepo) CreateCallType(_ context.Context, v *entity.CallType) (*entity.CallType, error) {
	v.Code = 1
	return v, nil
}
func (f *fakeCSRepo) ListCallTypes(context.Context, bool) ([]*entity.CallType, error) {
	return nil, nil
}
func (f *fakeCSRepo) GetCallType(_ context.Context, code int64) (*entity.CallType, error) {
	return f.callTypes[code], nil
}
func (f *fakeCSRepo) CreateKnowledgeSource(context.Context, *entity.KnowledgeSource) (*entity.KnowledgeSource, error) {
	return nil, nil
}
func (f *fakeCSRepo) ListKnowledgeSources(context.Context, bool) ([]*entity.KnowledgeSource, error) {
	return nil, nil
}
func (f *fakeCSRepo) CreateConsumer(_ context.Context, _ int64, v *entity.Consumer) (*entity.Consumer, error) {
	f.consumer = v
	return v, nil
}
func (f *fakeCSRepo) UpdateConsumer(_ context.Context, _ int64, v *entity.Consumer) (*entity.Consumer, error) {
	f.consumer = v
	return v, nil
}
func (f *fakeCSRepo) GetConsumer(context.Context, int64, int64) (*entity.Consumer, error) {
	return f.consumer, nil
}
func (f *fakeCSRepo) ListConsumers(context.Context, int64, csrepo.ConsumerFilter) ([]*entity.Consumer, error) {
	return nil, nil
}
func (f *fakeCSRepo) AddConsumerPhone(_ context.Context, _ int64, v *entity.ConsumerPhone) (*entity.ConsumerPhone, error) {
	return v, nil
}
func (f *fakeCSRepo) AddConsumerEmail(_ context.Context, _ int64, v *entity.ConsumerEmail) (*entity.ConsumerEmail, error) {
	return v, nil
}
func (f *fakeCSRepo) AddConsumerContact(_ context.Context, _ int64, v *entity.ConsumerContact) (*entity.ConsumerContact, error) {
	return v, nil
}
func (f *fakeCSRepo) CreateCustomerContact(_ context.Context, _ int64, v *entity.CustomerContactHistory) (*entity.CustomerContactHistory, error) {
	return v, nil
}
func (f *fakeCSRepo) ListCustomerContacts(context.Context, int64, csrepo.CustomerContactFilter) ([]*entity.CustomerContactHistory, error) {
	return nil, nil
}
func (f *fakeCSRepo) CreateCall(_ context.Context, tenantID int64, v *entity.Call) (*entity.Call, error) {
	f.tenantID = tenantID
	v.Code = 1
	v.EnterpriseCode = tenantID
	f.call = v
	return v, nil
}

func TestCreateCallUsesAuthenticatedTenant(t *testing.T) {
	repo := &fakeCSRepo{callTypes: map[int64]*entity.CallType{7: {Code: 7}}}
	uc := &UseCase{Repo: repo, Auth: csAllowAuth{}}
	_, err := uc.CreateCall(context.Background(), request.CreateConsumerServiceCallDTO{
		EnterpriseCode: 999, ConsumerCode: 1, CallTypeCode: 7, Subject: "tenant test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.tenantID != 1 {
		t.Fatalf("repository tenant = %d, want authenticated tenant 1", repo.tenantID)
	}
}
func (f *fakeCSRepo) UpdateCall(_ context.Context, _ int64, v *entity.Call) (*entity.Call, error) {
	f.call = v
	return v, nil
}
func (f *fakeCSRepo) GetCall(context.Context, int64, int64) (*entity.Call, error) { return f.call, nil }
func (f *fakeCSRepo) ListCalls(context.Context, int64, csrepo.CallFilter) ([]*entity.Call, error) {
	return nil, nil
}
func (f *fakeCSRepo) AddCallReturn(_ context.Context, _ int64, v *entity.CallReturn) (*entity.CallReturn, error) {
	return v, nil
}
func (f *fakeCSRepo) AddCallAttachment(_ context.Context, _ int64, v *entity.CallAttachment) (*entity.CallAttachment, error) {
	return v, nil
}
func (f *fakeCSRepo) AddChecklistItem(_ context.Context, _ int64, v *entity.CallChecklistItem) (*entity.CallChecklistItem, error) {
	return v, nil
}
func (f *fakeCSRepo) SetChecklistItemDone(context.Context, int64, int64, bool, *string) (*entity.CallChecklistItem, error) {
	return nil, nil
}
func (f *fakeCSRepo) ReportCalls(context.Context, int64, csrepo.CallFilter) (*csrepo.CallReport, error) {
	return nil, nil
}

func TestCreateConsumerRejectsCNPJForPessoaFisica(t *testing.T) {
	cnpj := "123"
	uc := &UseCase{Repo: &fakeCSRepo{}, Auth: csAllowAuth{}}

	_, err := uc.CreateConsumer(context.Background(), request.CreateConsumerDTO{
		Name: "Consumidor", PersonType: "F", CNPJ: &cnpj, CreatedBy: uuid.Nil,
	})
	if err == nil {
		t.Fatal("expected document validation error")
	}
}

func TestCreateCallRequiresSymptomsForComplaintType(t *testing.T) {
	uc := &UseCase{
		Repo: &fakeCSRepo{callTypes: map[int64]*entity.CallType{7: {Code: 7, IsComplaint: true}}},
		Auth: csAllowAuth{},
	}

	_, err := uc.CreateCall(context.Background(), request.CreateConsumerServiceCallDTO{
		EnterpriseCode: 1, ConsumerCode: 1, CallTypeCode: 7, Subject: "Reclamacao", CreatedBy: uuid.Nil,
	})
	if err == nil {
		t.Fatal("expected symptoms validation error")
	}
}

func TestCreateCallRequiresVisitDateForTechnicalVisit(t *testing.T) {
	uc := &UseCase{
		Repo: &fakeCSRepo{callTypes: map[int64]*entity.CallType{8: {Code: 8}}},
		Auth: csAllowAuth{},
	}

	_, err := uc.CreateCall(context.Background(), request.CreateConsumerServiceCallDTO{
		EnterpriseCode: 1, ConsumerCode: 1, CallTypeCode: 8, Subject: "Vistoria",
		Situation: "TECHNICAL_VISIT", CreatedBy: uuid.Nil,
	})
	if err == nil {
		t.Fatal("expected visit_requested_date validation error")
	}
}

func TestCreateCustomerContactDefaultsScheduleToNow(t *testing.T) {
	repo := &fakeCSRepo{}
	uc := &UseCase{Repo: repo, Auth: csAllowAuth{}}

	got, err := uc.CreateCustomerContact(context.Background(), request.CreateCustomerContactHistoryDTO{
		CustomerCode: 10, ContactType: "email", Description: "Retorno combinado", CreatedBy: uuid.Nil,
	})
	if err != nil {
		t.Fatalf("CreateCustomerContact() error = %v", err)
	}
	if time.Since(got.ScheduledAt) > time.Minute {
		t.Fatalf("scheduled_at = %v, want default near now", got.ScheduledAt)
	}
}
