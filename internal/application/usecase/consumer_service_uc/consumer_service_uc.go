package consumer_service_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
	csrepo "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type UseCase struct {
	Repo csrepo.Repository
	Auth ports.AuthService
}

func (uc *UseCase) ensureAllowed(ctx context.Context) error {
	if uc.Auth == nil || !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

func (uc *UseCase) CreateCallType(ctx context.Context, dto request.CreateConsumerServiceCallTypeDTO) (*response.ConsumerServiceCallTypeResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("description is required")
	}
	created, err := uc.Repo.CreateCallType(ctx, &entity.CallType{Description: strings.TrimSpace(dto.Description), IsComplaint: dto.IsComplaint, IsActive: true, CreatedBy: dto.CreatedBy})
	if err != nil {
		return nil, err
	}
	return toCallTypeResponse(created), nil
}

func (uc *UseCase) ListCallTypes(ctx context.Context, onlyActive bool) ([]*response.ConsumerServiceCallTypeResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	rows, err := uc.Repo.ListCallTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ConsumerServiceCallTypeResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCallTypeResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateKnowledgeSource(ctx context.Context, dto request.CreateConsumerServiceKnowledgeSourceDTO) (*response.ConsumerServiceKnowledgeSourceResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("description is required")
	}
	created, err := uc.Repo.CreateKnowledgeSource(ctx, &entity.KnowledgeSource{Description: strings.TrimSpace(dto.Description), IsActive: true, CreatedBy: dto.CreatedBy})
	if err != nil {
		return nil, err
	}
	return toKnowledgeSourceResponse(created), nil
}

func (uc *UseCase) ListKnowledgeSources(ctx context.Context, onlyActive bool) ([]*response.ConsumerServiceKnowledgeSourceResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	rows, err := uc.Repo.ListKnowledgeSources(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ConsumerServiceKnowledgeSourceResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toKnowledgeSourceResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateConsumer(ctx context.Context, dto request.CreateConsumerDTO) (*response.ConsumerResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("name is required")
	}
	personType, err := normalizePersonType(dto.PersonType)
	if err != nil {
		return nil, err
	}
	if err := validateConsumerDocument(personType, dto.CPF, dto.CNPJ); err != nil {
		return nil, err
	}
	code := dto.Code
	if code == 0 {
		code, err = uc.Repo.NextConsumerCode(ctx)
		if err != nil {
			return nil, err
		}
	}
	consumer := &entity.Consumer{
		Code: code, Name: strings.TrimSpace(dto.Name), IsActive: true, PersonType: personType, CPF: dto.CPF, RG: dto.RG,
		CNPJ: dto.CNPJ, StateRegistration: dto.StateRegistration, ZipCode: dto.ZipCode, City: dto.City, State: dto.State,
		Address: dto.Address, AddressNumber: dto.AddressNumber, Complement: dto.Complement, District: dto.District,
		MarketSegmentCode: dto.MarketSegmentCode, KnowledgeCode: dto.KnowledgeCode, Notes: dto.Notes, CreatedBy: dto.CreatedBy,
	}
	created, err := uc.Repo.CreateConsumer(ctx, consumer)
	if err != nil {
		return nil, err
	}
	for _, phone := range dto.Phones {
		phone.ConsumerCode = created.Code
		if _, err := uc.AddConsumerPhone(ctx, phone); err != nil {
			return nil, err
		}
	}
	for _, email := range dto.Emails {
		email.ConsumerCode = created.Code
		if _, err := uc.AddConsumerEmail(ctx, email); err != nil {
			return nil, err
		}
	}
	for _, contact := range dto.Contacts {
		contact.ConsumerCode = created.Code
		if _, err := uc.AddConsumerContact(ctx, contact); err != nil {
			return nil, err
		}
	}
	return uc.GetConsumer(ctx, created.Code)
}

func (uc *UseCase) UpdateConsumer(ctx context.Context, code int64, dto request.UpdateConsumerDTO) (*response.ConsumerResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	current, err := uc.Repo.GetConsumer(ctx, code)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Name) != "" {
		current.Name = strings.TrimSpace(dto.Name)
	}
	if dto.IsActive != nil {
		current.IsActive = *dto.IsActive
	}
	if dto.PersonType != "" {
		current.PersonType, err = normalizePersonType(dto.PersonType)
		if err != nil {
			return nil, err
		}
	}
	current.CPF, current.RG, current.CNPJ, current.StateRegistration = dto.CPF, dto.RG, dto.CNPJ, dto.StateRegistration
	current.ZipCode, current.City, current.State = dto.ZipCode, dto.City, dto.State
	current.Address, current.AddressNumber, current.Complement, current.District = dto.Address, dto.AddressNumber, dto.Complement, dto.District
	current.MarketSegmentCode, current.KnowledgeCode, current.Notes = dto.MarketSegmentCode, dto.KnowledgeCode, dto.Notes
	if err := validateConsumerDocument(current.PersonType, current.CPF, current.CNPJ); err != nil {
		return nil, err
	}
	updated, err := uc.Repo.UpdateConsumer(ctx, current)
	if err != nil {
		return nil, err
	}
	return toConsumerResponse(updated), nil
}

func (uc *UseCase) GetConsumer(ctx context.Context, code int64) (*response.ConsumerResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	row, err := uc.Repo.GetConsumer(ctx, code)
	if err != nil {
		return nil, err
	}
	return toConsumerResponse(row), nil
}

func (uc *UseCase) ListConsumers(ctx context.Context, filter csrepo.ConsumerFilter) ([]*response.ConsumerResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	rows, err := uc.Repo.ListConsumers(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ConsumerResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toConsumerResponse(row))
	}
	return out, nil
}

func (uc *UseCase) AddConsumerPhone(ctx context.Context, dto request.CreateConsumerPhoneDTO) (*response.ConsumerPhoneResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.ConsumerCode == 0 || strings.TrimSpace(dto.Number) == "" {
		return nil, errorsuc.NewValidationError("consumer_code and number are required")
	}
	phoneType := strings.ToUpper(strings.TrimSpace(dto.PhoneType))
	if phoneType == "" {
		phoneType = "PHONE"
	}
	created, err := uc.Repo.AddConsumerPhone(ctx, &entity.ConsumerPhone{ConsumerCode: dto.ConsumerCode, ContactCode: dto.ContactCode, PhoneType: phoneType, Number: strings.TrimSpace(dto.Number), IsPrimary: dto.IsPrimary})
	if err != nil {
		return nil, err
	}
	return &response.ConsumerPhoneResponse{Code: created.Code, PhoneType: created.PhoneType, Number: created.Number, IsPrimary: created.IsPrimary, ContactCode: created.ContactCode}, nil
}

func (uc *UseCase) AddConsumerEmail(ctx context.Context, dto request.CreateConsumerEmailDTO) (*response.ConsumerEmailResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.ConsumerCode == 0 || !strings.Contains(dto.Email, "@") {
		return nil, errorsuc.NewValidationError("consumer_code and valid email are required")
	}
	created, err := uc.Repo.AddConsumerEmail(ctx, &entity.ConsumerEmail{ConsumerCode: dto.ConsumerCode, ContactCode: dto.ContactCode, Email: strings.TrimSpace(dto.Email), IsPrimary: dto.IsPrimary})
	if err != nil {
		return nil, err
	}
	return &response.ConsumerEmailResponse{Code: created.Code, Email: created.Email, IsPrimary: created.IsPrimary, ContactCode: created.ContactCode}, nil
}

func (uc *UseCase) AddConsumerContact(ctx context.Context, dto request.CreateConsumerContactDTO) (*response.ConsumerContactResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.ConsumerCode == 0 || strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("consumer_code and name are required")
	}
	created, err := uc.Repo.AddConsumerContact(ctx, &entity.ConsumerContact{ConsumerCode: dto.ConsumerCode, Name: strings.TrimSpace(dto.Name), Role: dto.Role, ContactType: dto.ContactType, Notes: dto.Notes})
	if err != nil {
		return nil, err
	}
	return &response.ConsumerContactResponse{Code: created.Code, Name: created.Name, Role: created.Role, ContactType: created.ContactType, Notes: created.Notes}, nil
}

func (uc *UseCase) CreateCustomerContact(ctx context.Context, dto request.CreateCustomerContactHistoryDTO) (*response.CustomerContactHistoryResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.CustomerCode == 0 || strings.TrimSpace(dto.ContactType) == "" || strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("customer_code, contact_type and description are required")
	}
	openedAt := parseDateTimeOrNow(dto.OpenedAt)
	scheduledAt := parseDateTimeOrNow(dto.ScheduledAt)
	created, err := uc.Repo.CreateCustomerContact(ctx, &entity.CustomerContactHistory{
		CustomerCode: dto.CustomerCode, OpenedAt: openedAt, ScheduledAt: scheduledAt, UserCode: dto.UserCode,
		ContactType: strings.ToUpper(strings.TrimSpace(dto.ContactType)), Description: strings.TrimSpace(dto.Description), CreatedBy: dto.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return toCustomerContactResponse(created), nil
}

func (uc *UseCase) ListCustomerContacts(ctx context.Context, filter csrepo.CustomerContactFilter) ([]*response.CustomerContactHistoryResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	rows, err := uc.Repo.ListCustomerContacts(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CustomerContactHistoryResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCustomerContactResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateCall(ctx context.Context, dto request.CreateConsumerServiceCallDTO) (*response.ConsumerServiceCallResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.EnterpriseCode == 0 || dto.ConsumerCode == 0 || dto.CallTypeCode == 0 || strings.TrimSpace(dto.Subject) == "" {
		return nil, errorsuc.NewValidationError("enterprise_code, consumer_code, call_type_code and subject are required")
	}
	callType, err := uc.Repo.GetCallType(ctx, dto.CallTypeCode)
	if err != nil {
		return nil, err
	}
	position := normalizePosition(dto.Position)
	situation := normalizeSituation(dto.Situation)
	if err := validateVisitDates(situation, dto.VisitRequestedDate, dto.VisitReturnedDate); err != nil {
		return nil, err
	}
	if callType.IsComplaint && (dto.Symptoms == nil || strings.TrimSpace(*dto.Symptoms) == "") {
		return nil, errorsuc.NewValidationError("symptoms is required for complaint call types")
	}
	callNumber, err := uc.Repo.NextCallNumber(ctx, dto.EnterpriseCode)
	if err != nil {
		return nil, err
	}
	call := &entity.Call{
		CallNumber: callNumber, EnterpriseCode: dto.EnterpriseCode, ConsumerCode: dto.ConsumerCode, CustomerCode: dto.CustomerCode,
		CallTypeCode: dto.CallTypeCode, Direction: normalizeDirection(dto.Direction), InWarranty: dto.InWarranty,
		DefectGroupCode: dto.DefectGroupCode, DefectReasonCode: dto.DefectReasonCode, ResponsibleUserCode: dto.ResponsibleUserCode,
		Position: position, Situation: situation, OpenedAt: parseDateTimeOrNow(dto.OpenedAt), ReturnDate: parseDatePtr(dto.ReturnDate),
		VisitRequestedDate: parseDatePtr(dto.VisitRequestedDate), VisitReturnedDate: parseDatePtr(dto.VisitReturnedDate),
		SaleStoreCode: dto.SaleStoreCode, EstablishmentCode: dto.EstablishmentCode, TechnicianDescription: dto.TechnicianDescription,
		Symptoms: dto.Symptoms, ForwardedStoreCode: dto.ForwardedStoreCode, Subject: strings.TrimSpace(dto.Subject),
		Description: dto.Description, ChecklistCode: dto.ChecklistCode, IsActive: true, CreatedBy: dto.CreatedBy,
	}
	created, err := uc.Repo.CreateCall(ctx, call)
	if err != nil {
		return nil, err
	}
	return uc.GetCall(ctx, created.Code)
}

func (uc *UseCase) UpdateCall(ctx context.Context, code int64, dto request.UpdateConsumerServiceCallDTO) (*response.ConsumerServiceCallResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	current, err := uc.Repo.GetCall(ctx, code)
	if err != nil {
		return nil, err
	}
	if dto.CallTypeCode != 0 {
		current.CallTypeCode = dto.CallTypeCode
	}
	callType, err := uc.Repo.GetCallType(ctx, current.CallTypeCode)
	if err != nil {
		return nil, err
	}
	current.Direction = normalizeDirection(dto.Direction)
	current.InWarranty = dto.InWarranty
	current.DefectGroupCode, current.DefectReasonCode, current.ResponsibleUserCode = dto.DefectGroupCode, dto.DefectReasonCode, dto.ResponsibleUserCode
	current.Position = normalizePosition(dto.Position)
	current.Situation = normalizeSituation(dto.Situation)
	if err := validateVisitDates(current.Situation, dto.VisitRequestedDate, dto.VisitReturnedDate); err != nil {
		return nil, err
	}
	current.ReturnDate = parseDatePtr(dto.ReturnDate)
	current.VisitRequestedDate = parseDatePtr(dto.VisitRequestedDate)
	current.VisitReturnedDate = parseDatePtr(dto.VisitReturnedDate)
	current.SaleStoreCode, current.EstablishmentCode = dto.SaleStoreCode, dto.EstablishmentCode
	current.TechnicianDescription, current.Symptoms, current.ForwardedStoreCode = dto.TechnicianDescription, dto.Symptoms, dto.ForwardedStoreCode
	if strings.TrimSpace(dto.Subject) != "" {
		current.Subject = strings.TrimSpace(dto.Subject)
	}
	current.Description, current.Solution, current.ChecklistCode = dto.Description, dto.Solution, dto.ChecklistCode
	if callType.IsComplaint && (current.Symptoms == nil || strings.TrimSpace(*current.Symptoms) == "") {
		return nil, errorsuc.NewValidationError("symptoms is required for complaint call types")
	}
	updated, err := uc.Repo.UpdateCall(ctx, current)
	if err != nil {
		return nil, err
	}
	return toCallResponse(updated), nil
}

func (uc *UseCase) GetCall(ctx context.Context, code int64) (*response.ConsumerServiceCallResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	row, err := uc.Repo.GetCall(ctx, code)
	if err != nil {
		return nil, err
	}
	return toCallResponse(row), nil
}

func (uc *UseCase) ListCalls(ctx context.Context, filter csrepo.CallFilter) ([]*response.ConsumerServiceCallResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	rows, err := uc.Repo.ListCalls(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ConsumerServiceCallResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCallResponse(row))
	}
	return out, nil
}

func (uc *UseCase) AddCallReturn(ctx context.Context, dto request.AddConsumerServiceCallReturnDTO) (*response.ConsumerServiceCallReturnResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.CallCode == 0 || strings.TrimSpace(dto.ContactType) == "" || strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("call_code, contact_type and description are required")
	}
	created, err := uc.Repo.AddCallReturn(ctx, &entity.CallReturn{
		CallCode: dto.CallCode, ContactedAt: parseDateTimeOrNow(dto.ContactedAt),
		ContactType: strings.ToUpper(strings.TrimSpace(dto.ContactType)), Description: strings.TrimSpace(dto.Description),
		NextReturnAt: parseDatePtr(dto.NextReturnAt), UserCode: dto.UserCode, CreatedBy: dto.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return toCallReturnResponse(created), nil
}

func (uc *UseCase) AddCallAttachment(ctx context.Context, dto request.AddConsumerServiceCallAttachmentDTO) (*response.ConsumerServiceCallAttachmentResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.CallCode == 0 || strings.TrimSpace(dto.FileName) == "" || strings.TrimSpace(dto.FilePath) == "" {
		return nil, errorsuc.NewValidationError("call_code, file_name and file_path are required")
	}
	created, err := uc.Repo.AddCallAttachment(ctx, &entity.CallAttachment{
		CallCode: dto.CallCode, FileName: strings.TrimSpace(dto.FileName), FilePath: strings.TrimSpace(dto.FilePath),
		ContentType: dto.ContentType, Notes: dto.Notes, CreatedBy: dto.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return toCallAttachmentResponse(created), nil
}

func (uc *UseCase) AddChecklistItem(ctx context.Context, dto request.AddConsumerServiceChecklistItemDTO) (*response.ConsumerServiceChecklistItemResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.CallCode == 0 || strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("call_code and description are required")
	}
	created, err := uc.Repo.AddChecklistItem(ctx, &entity.CallChecklistItem{CallCode: dto.CallCode, Sequence: dto.Sequence, Description: strings.TrimSpace(dto.Description), Notes: dto.Notes})
	if err != nil {
		return nil, err
	}
	return toChecklistItemResponse(created), nil
}

func (uc *UseCase) SetChecklistItemDone(ctx context.Context, code int64, dto request.SetConsumerServiceChecklistItemDoneDTO) (*response.ConsumerServiceChecklistItemResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	updated, err := uc.Repo.SetChecklistItemDone(ctx, code, dto.Done, dto.Notes)
	if err != nil {
		return nil, err
	}
	return toChecklistItemResponse(updated), nil
}

func (uc *UseCase) ReportCalls(ctx context.Context, filter csrepo.CallFilter) (*response.ConsumerServiceCallReportResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	return uc.Repo.ReportCalls(ctx, filter)
}

func normalizePersonType(value string) (string, error) {
	v := strings.ToUpper(strings.TrimSpace(value))
	if v == "" {
		v = "F"
	}
	switch v {
	case "F", "PF", "PERSON", "PESSOA_FISICA":
		return "F", nil
	case "J", "PJ", "COMPANY", "PESSOA_JURIDICA":
		return "J", nil
	default:
		return "", errorsuc.NewValidationError("person_type must be F or J")
	}
}

func validateConsumerDocument(personType string, cpf, cnpj *string) error {
	if personType == "F" && cnpj != nil && strings.TrimSpace(*cnpj) != "" {
		return errorsuc.NewValidationError("cnpj is not allowed for pessoa fisica")
	}
	if personType == "J" && cpf != nil && strings.TrimSpace(*cpf) != "" {
		return errorsuc.NewValidationError("cpf is not allowed for pessoa juridica")
	}
	return nil
}

func normalizeDirection(value string) entity.CallDirection {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MADE", "EFETUADA":
		return entity.CallDirectionMade
	case "WARRANTY", "GARANTIA":
		return entity.CallDirectionWarranty
	default:
		return entity.CallDirectionReceived
	}
}

func normalizePosition(value string) entity.CallPosition {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SCHEDULED", "AGENDAMENTO":
		return entity.CallPositionScheduled
	case "RESOLVED", "RESOLVIDO":
		return entity.CallPositionResolved
	default:
		return entity.CallPositionPending
	}
}

func normalizeSituation(value string) entity.CallSituation {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ORDER", "PEDIDO":
		return entity.CallSituationOrder
	case "DISCONTINUED_ORDER", "PEDIDO_FORA_DE_LINHA":
		return entity.CallSituationDiscontinued
	case "TECHNICAL_VISIT", "VISTORIA":
		return entity.CallSituationTechnicalVisit
	default:
		return entity.CallSituationOther
	}
}

func validateVisitDates(situation entity.CallSituation, requested, returned string) error {
	if situation != entity.CallSituationTechnicalVisit {
		return nil
	}
	if parseDatePtr(requested) == nil {
		return errorsuc.NewValidationError("visit_requested_date is required when situation is TECHNICAL_VISIT")
	}
	if returned != "" {
		if ret := parseDatePtr(returned); ret == nil {
			return errorsuc.NewValidationError("visit_returned_date must be a valid date")
		}
	}
	return nil
}

func parseDatePtr(value string) *time.Time {
	return datetime.ParseDatePtr(&value)
}

func parseDateTimeOrNow(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Now()
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed := parseDatePtr(value); parsed != nil {
		return *parsed
	}
	return time.Now()
}
