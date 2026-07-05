package representative_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type UseCase struct {
	Repo repository.RepresentativeRepository
	Auth ports.AuthService
}

func (uc *UseCase) CreateType(ctx context.Context, dto request.CreateRepresentativeTypeDTO) (*response.RepresentativeTypeResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("description is required")
	}
	ignores := true
	if dto.IgnoresDirectBilling != nil {
		ignores = *dto.IgnoresDirectBilling
	}
	t, err := uc.Repo.CreateType(ctx, &entity.RepresentativeType{
		Description:          strings.TrimSpace(dto.Description),
		IsFree:               dto.IsFree,
		IgnoresDirectBilling: ignores,
		IsActive:             true,
	})
	if err != nil {
		return nil, err
	}
	return toTypeResponse(t), nil
}

func (uc *UseCase) UpdateType(ctx context.Context, dto request.UpdateRepresentativeTypeDTO) (*response.RepresentativeTypeResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Code == 0 {
		return nil, errorsuc.NewValidationError("code is required")
	}
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("description is required")
	}
	t, err := uc.Repo.UpdateType(ctx, &entity.RepresentativeType{
		Code:                 dto.Code,
		Description:          strings.TrimSpace(dto.Description),
		IsFree:               dto.IsFree,
		IgnoresDirectBilling: dto.IgnoresDirectBilling,
		IsActive:             dto.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return toTypeResponse(t), nil
}

func (uc *UseCase) GetType(ctx context.Context, code int64) (*response.RepresentativeTypeResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	t, err := uc.Repo.GetType(ctx, code)
	if err != nil {
		return nil, err
	}
	return toTypeResponse(t), nil
}

func (uc *UseCase) ListTypes(ctx context.Context, onlyActive bool) ([]*response.RepresentativeTypeResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.ListTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.RepresentativeTypeResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTypeResponse(row))
	}
	return out, nil
}

func (uc *UseCase) Create(ctx context.Context, dto request.CreateRepresentativeDTO) (*response.RepresentativeResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rep, err := representativeFromCreate(dto)
	if err != nil {
		return nil, err
	}
	created, err := uc.Repo.Create(ctx, rep)
	if err != nil {
		return nil, err
	}
	return toResponse(created), nil
}

func (uc *UseCase) Update(ctx context.Context, dto request.UpdateRepresentativeDTO) (*response.RepresentativeResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Code == 0 {
		return nil, errorsuc.NewValidationError("code is required")
	}
	rep, err := representativeFromUpdate(dto)
	if err != nil {
		return nil, err
	}
	updated, err := uc.Repo.Update(ctx, rep)
	if err != nil {
		return nil, err
	}
	return toResponse(updated), nil
}

func (uc *UseCase) Get(ctx context.Context, code int64) (*response.RepresentativeResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rep, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	return toResponse(rep), nil
}

func (uc *UseCase) List(ctx context.Context, filter repository.RepresentativeFilter) ([]*response.RepresentativeResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.List(ctx, normalizeFilter(filter))
	if err != nil {
		return nil, err
	}
	out := make([]*response.RepresentativeResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toResponse(row))
	}
	return out, nil
}

func (uc *UseCase) Block(ctx context.Context, code int64, dto request.BlockRepresentativeDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return errorsuc.NewValidationError("reason is required")
	}
	return uc.Repo.Block(ctx, code, strings.TrimSpace(dto.Reason))
}

func (uc *UseCase) Unblock(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Unblock(ctx, code)
}

func (uc *UseCase) AddEnterprise(ctx context.Context, dto request.RepresentativeEnterpriseDTO) (*response.RepresentativeEnterpriseResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || dto.EnterpriseCode == 0 {
		return nil, errorsuc.NewValidationError("representative_code and enterprise_code are required")
	}
	row, err := uc.Repo.AddEnterprise(ctx, &entity.RepresentativeEnterprise{
		RepresentativeCode:    dto.RepresentativeCode,
		EnterpriseCode:        dto.EnterpriseCode,
		EnterpriseName:        dto.EnterpriseName,
		CommissionPatternCode: dto.CommissionPatternCode,
		CommissionPct:         dto.CommissionPct,
		IsDefault:             dto.IsDefault,
		IsActive:              dto.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return toEnterpriseResponse(row), nil
}

func (uc *UseCase) AddAccounting(ctx context.Context, dto request.RepresentativeAccountingDTO) (*response.RepresentativeAccountingResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	event := strings.ToUpper(strings.TrimSpace(dto.EventType))
	if dto.RepresentativeCode == 0 || (event != "GENERATED" && event != "REVERSED") {
		return nil, errorsuc.NewValidationError("representative_code and valid event_type are required")
	}
	row, err := uc.Repo.AddAccounting(ctx, &entity.RepresentativeAccounting{
		RepresentativeCode:   dto.RepresentativeCode,
		EnterpriseCode:       dto.EnterpriseCode,
		EventType:            event,
		DebitAccountCode:     dto.DebitAccountCode,
		DebitCostCenterCode:  dto.DebitCostCenterCode,
		CreditAccountCode:    dto.CreditAccountCode,
		CreditCostCenterCode: dto.CreditCostCenterCode,
		HistoryCode:          dto.HistoryCode,
	})
	if err != nil {
		return nil, err
	}
	return toAccountingResponse(row), nil
}

func (uc *UseCase) AddRegion(ctx context.Context, dto request.RepresentativeRegionDTO) (*response.RepresentativeRegionResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || dto.RegionCode == 0 {
		return nil, errorsuc.NewValidationError("representative_code and region_code are required")
	}
	row, err := uc.Repo.AddRegion(ctx, &entity.RepresentativeRegion{RepresentativeCode: dto.RepresentativeCode, EnterpriseCode: dto.EnterpriseCode, RegionCode: dto.RegionCode, MicroregionCode: dto.MicroregionCode, IsActive: dto.IsActive})
	if err != nil {
		return nil, err
	}
	return toRegionResponse(row), nil
}

func (uc *UseCase) AddSegment(ctx context.Context, dto request.RepresentativeSegmentDTO) (*response.RepresentativeSegmentResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || dto.MarketSegmentCode == 0 {
		return nil, errorsuc.NewValidationError("representative_code and market_segment_code are required")
	}
	row, err := uc.Repo.AddSegment(ctx, &entity.RepresentativeSegment{RepresentativeCode: dto.RepresentativeCode, EnterpriseCode: dto.EnterpriseCode, MicroregionCode: dto.MicroregionCode, MarketSegmentCode: dto.MarketSegmentCode, IsActive: dto.IsActive})
	if err != nil {
		return nil, err
	}
	return toSegmentResponse(row), nil
}

func (uc *UseCase) AddSalesPlan(ctx context.Context, dto request.RepresentativeSalesPlanDTO) (*response.RepresentativeSalesPlanResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || dto.SalesPlanCode == 0 {
		return nil, errorsuc.NewValidationError("representative_code and sales_plan_code are required")
	}
	row, err := uc.Repo.AddSalesPlan(ctx, &entity.RepresentativeSalesPlan{RepresentativeCode: dto.RepresentativeCode, EnterpriseCode: dto.EnterpriseCode, MicroregionCode: dto.MicroregionCode, SalesPlanCode: dto.SalesPlanCode, IsActive: dto.IsActive})
	if err != nil {
		return nil, err
	}
	return toSalesPlanResponse(row), nil
}

func (uc *UseCase) AddInterest(ctx context.Context, dto request.RepresentativeInterestDTO) (*response.RepresentativeInterestResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || dto.ItemClassificationCode == 0 {
		return nil, errorsuc.NewValidationError("representative_code and item_classification_code are required")
	}
	row, err := uc.Repo.AddInterest(ctx, &entity.RepresentativeInterest{RepresentativeCode: dto.RepresentativeCode, ItemClassificationCode: dto.ItemClassificationCode, IsActive: dto.IsActive})
	if err != nil {
		return nil, err
	}
	return toInterestResponse(row), nil
}

func (uc *UseCase) AddPhone(ctx context.Context, dto request.RepresentativePhoneDTO) (*response.RepresentativePhoneResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || strings.TrimSpace(dto.Phone) == "" {
		return nil, errorsuc.NewValidationError("representative_code and phone are required")
	}
	if dto.Ranking <= 0 {
		dto.Ranking = 1
	}
	if strings.TrimSpace(dto.PhoneType) == "" {
		dto.PhoneType = "COMERCIAL"
	}
	row, err := uc.Repo.AddPhone(ctx, &entity.RepresentativePhone{RepresentativeCode: dto.RepresentativeCode, DDI: dto.DDI, DDD: dto.DDD, Phone: strings.TrimSpace(dto.Phone), PhoneType: strings.ToUpper(strings.TrimSpace(dto.PhoneType)), Ranking: dto.Ranking})
	if err != nil {
		return nil, err
	}
	return toPhoneResponse(row), nil
}

func (uc *UseCase) AddEmail(ctx context.Context, dto request.RepresentativeEmailDTO) (*response.RepresentativeEmailResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || !strings.Contains(dto.Email, "@") {
		return nil, errorsuc.NewValidationError("representative_code and valid email are required")
	}
	if dto.Ranking <= 0 {
		dto.Ranking = 1
	}
	row, err := uc.Repo.AddEmail(ctx, &entity.RepresentativeEmail{RepresentativeCode: dto.RepresentativeCode, Email: strings.TrimSpace(dto.Email), Ranking: dto.Ranking})
	if err != nil {
		return nil, err
	}
	return toEmailResponse(row), nil
}

func (uc *UseCase) AddCorrespondenceAddress(ctx context.Context, dto request.RepresentativeCorrespondenceAddressDTO) (*response.RepresentativeCorrespondenceAddressResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 {
		return nil, errorsuc.NewValidationError("representative_code is required")
	}
	normalizeState(dto.State)
	row, err := uc.Repo.AddCorrespondenceAddress(ctx, &entity.RepresentativeCorrespondenceAddress{RepresentativeCode: dto.RepresentativeCode, PostalCode: dto.PostalCode, City: dto.City, State: dto.State, FullAddress: fullAddress(dto.FullAddress, dto.Street, dto.StreetNumber), Street: dto.Street, StreetNumber: dto.StreetNumber, Complement: dto.Complement, District: dto.District, IsDefault: dto.IsDefault})
	if err != nil {
		return nil, err
	}
	return toCorrespondenceResponse(row), nil
}

func (uc *UseCase) AddContact(ctx context.Context, dto request.RepresentativeContactDTO) (*response.RepresentativeContactResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RepresentativeCode == 0 || strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("representative_code and name are required")
	}
	row, err := uc.Repo.AddContact(ctx, &entity.RepresentativeContact{RepresentativeCode: dto.RepresentativeCode, ContactTypeCode: dto.ContactTypeCode, Name: strings.TrimSpace(dto.Name), Role: dto.Role, Phone: dto.Phone, Email: dto.Email, Notes: dto.Notes, IsActive: dto.IsActive})
	if err != nil {
		return nil, err
	}
	return toContactResponse(row), nil
}

func (uc *UseCase) Report(ctx context.Context, filter repository.RepresentativeFilter) ([]response.RepresentativeReportRowResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.Report(ctx, normalizeFilter(filter))
	if err != nil {
		return nil, err
	}
	out := make([]response.RepresentativeReportRowResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toReportRowResponse(row))
	}
	return out, nil
}

func (uc *UseCase) FollowUp(ctx context.Context, filter repository.FollowUpFilter) ([]response.RepresentativeFollowUpResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.FollowUp(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]response.RepresentativeFollowUpResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toFollowUpResponse(row))
	}
	return out, nil
}

func representativeFromCreate(dto request.CreateRepresentativeDTO) (*entity.Representative, error) {
	if strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("name is required")
	}
	if strings.TrimSpace(dto.DocumentNumber) == "" {
		return nil, errorsuc.NewValidationError("document_number is required")
	}
	if dto.DeviceQuantity < 0 {
		return nil, errorsuc.NewValidationError("device_quantity cannot be negative")
	}
	normalizeState(dto.State)
	registerDate := datetime.ParseDateOrDefault(dto.RegisterDate, time.Now())
	return &entity.Representative{
		IsCustomer:     dto.IsCustomer,
		CustomerCode:   dto.CustomerCode,
		IsSupplier:     dto.IsSupplier,
		SupplierCode:   dto.SupplierCode,
		Name:           strings.TrimSpace(dto.Name),
		TradeName:      dto.TradeName,
		TypeCode:       dto.TypeCode,
		CategoryCode:   dto.CategoryCode,
		RegisterDate:   registerDate,
		CoreNumber:     dto.CoreNumber,
		DocumentNumber: strings.TrimSpace(dto.DocumentNumber),
		PostalCode:     dto.PostalCode,
		City:           dto.City,
		State:          dto.State,
		FullAddress:    fullAddress(dto.FullAddress, dto.Street, dto.StreetNumber),
		Street:         dto.Street,
		StreetNumber:   dto.StreetNumber,
		Complement:     dto.Complement,
		District:       dto.District,
		DeviceQuantity: dto.DeviceQuantity,
		IsActive:       true,
	}, nil
}

func representativeFromUpdate(dto request.UpdateRepresentativeDTO) (*entity.Representative, error) {
	base, err := representativeFromCreate(request.CreateRepresentativeDTO{
		IsCustomer:     dto.IsCustomer,
		CustomerCode:   dto.CustomerCode,
		IsSupplier:     dto.IsSupplier,
		SupplierCode:   dto.SupplierCode,
		Name:           dto.Name,
		TradeName:      dto.TradeName,
		TypeCode:       dto.TypeCode,
		CategoryCode:   dto.CategoryCode,
		RegisterDate:   dto.RegisterDate,
		CoreNumber:     dto.CoreNumber,
		DocumentNumber: dto.DocumentNumber,
		PostalCode:     dto.PostalCode,
		City:           dto.City,
		State:          dto.State,
		FullAddress:    dto.FullAddress,
		Street:         dto.Street,
		StreetNumber:   dto.StreetNumber,
		Complement:     dto.Complement,
		District:       dto.District,
		DeviceQuantity: dto.DeviceQuantity,
	})
	if err != nil {
		return nil, err
	}
	base.Code = dto.Code
	base.IsActive = dto.IsActive
	return base, nil
}

func normalizeState(state *string) {
	if state == nil {
		return
	}
	v := strings.ToUpper(strings.TrimSpace(*state))
	*state = v
}

func fullAddress(explicit, street, number *string) *string {
	if explicit != nil && strings.TrimSpace(*explicit) != "" {
		return explicit
	}
	if street == nil || strings.TrimSpace(*street) == "" {
		return nil
	}
	value := strings.TrimSpace(*street)
	if number != nil && strings.TrimSpace(*number) != "" {
		value += ", " + strings.TrimSpace(*number)
	}
	return &value
}

func normalizeFilter(filter repository.RepresentativeFilter) repository.RepresentativeFilter {
	filter.SortBy = strings.ToUpper(strings.TrimSpace(filter.SortBy))
	switch filter.SortBy {
	case "CODE", "NAME", "STATE", "REGION":
	default:
		filter.SortBy = "CODE"
	}
	filter.ActiveStatus = strings.ToUpper(strings.TrimSpace(filter.ActiveStatus))
	switch filter.ActiveStatus {
	case "ACTIVE", "INACTIVE", "ALL":
	default:
		filter.ActiveStatus = "ACTIVE"
	}
	return filter
}
