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

type fakeCSRepo struct {
	callTypes map[int64]*entity.CallType
	consumer  *entity.Consumer
	call      *entity.Call
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
func (f *fakeCSRepo) CreateConsumer(_ context.Context, v *entity.Consumer) (*entity.Consumer, error) {
	f.consumer = v
	return v, nil
}
func (f *fakeCSRepo) UpdateConsumer(_ context.Context, v *entity.Consumer) (*entity.Consumer, error) {
	f.consumer = v
	return v, nil
}
func (f *fakeCSRepo) GetConsumer(context.Context, int64) (*entity.Consumer, error) {
	return f.consumer, nil
}
func (f *fakeCSRepo) ListConsumers(context.Context, csrepo.ConsumerFilter) ([]*entity.Consumer, error) {
	return nil, nil
}
func (f *fakeCSRepo) AddConsumerPhone(_ context.Context, v *entity.ConsumerPhone) (*entity.ConsumerPhone, error) {
	return v, nil
}
func (f *fakeCSRepo) AddConsumerEmail(_ context.Context, v *entity.ConsumerEmail) (*entity.ConsumerEmail, error) {
	return v, nil
}
func (f *fakeCSRepo) AddConsumerContact(_ context.Context, v *entity.ConsumerContact) (*entity.ConsumerContact, error) {
	return v, nil
}
func (f *fakeCSRepo) CreateCustomerContact(_ context.Context, v *entity.CustomerContactHistory) (*entity.CustomerContactHistory, error) {
	return v, nil
}
func (f *fakeCSRepo) ListCustomerContacts(context.Context, csrepo.CustomerContactFilter) ([]*entity.CustomerContactHistory, error) {
	return nil, nil
}
func (f *fakeCSRepo) CreateCall(_ context.Context, v *entity.Call) (*entity.Call, error) {
	v.Code = 1
	f.call = v
	return v, nil
}
func (f *fakeCSRepo) UpdateCall(_ context.Context, v *entity.Call) (*entity.Call, error) {
	f.call = v
	return v, nil
}
func (f *fakeCSRepo) GetCall(context.Context, int64) (*entity.Call, error) { return f.call, nil }
func (f *fakeCSRepo) ListCalls(context.Context, csrepo.CallFilter) ([]*entity.Call, error) {
	return nil, nil
}
func (f *fakeCSRepo) AddCallReturn(_ context.Context, v *entity.CallReturn) (*entity.CallReturn, error) {
	return v, nil
}
func (f *fakeCSRepo) AddCallAttachment(_ context.Context, v *entity.CallAttachment) (*entity.CallAttachment, error) {
	return v, nil
}
func (f *fakeCSRepo) AddChecklistItem(_ context.Context, v *entity.CallChecklistItem) (*entity.CallChecklistItem, error) {
	return v, nil
}
func (f *fakeCSRepo) SetChecklistItemDone(context.Context, int64, bool, *string) (*entity.CallChecklistItem, error) {
	return nil, nil
}
func (f *fakeCSRepo) ReportCalls(context.Context, csrepo.CallFilter) (*csrepo.CallReport, error) {
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
