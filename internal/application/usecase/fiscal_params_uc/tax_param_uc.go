package fiscal_params_uc

import (
	"context"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"
)

type TaxParamUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *TaxParamUseCase) Create(ctx context.Context, dto request.CreateTaxParamDTO) (*response.TaxParamResponse, error) {
	if dto.UF == "" {
		return nil, errorsuc.NewValidationError("informe a UF")
	}
	if dto.NCMCode == nil && dto.ItemCode == nil {
		return nil, errorsuc.NewValidationError("informe o NCM ou o item")
	}
	if dto.NCMCode != nil && dto.ItemCode != nil {
		return nil, errorsuc.NewValidationError("informe o NCM ou o item, não os dois")
	}
	if err := conferirCenario(dto); err != nil {
		return nil, err
	}
	p := dtoToTaxParamEntity(dto)
	p.IsActive = true
	created, err := uc.Repo.CreateTaxParam(ctx, p)
	if err != nil {
		return nil, err
	}
	return toTaxParamResponse(created), nil
}

func (uc *TaxParamUseCase) Update(ctx context.Context, dto request.UpdateTaxParamDTO) (*response.TaxParamResponse, error) {
	if dto.UF == "" {
		return nil, errorsuc.NewValidationError("informe a UF")
	}
	createDTO := request.CreateTaxParamDTO(dto.CreateTaxParamDTO)
	if err := conferirCenario(createDTO); err != nil {
		return nil, err
	}
	p := dtoToTaxParamEntity(createDTO)
	p.ID = dto.ID
	p.IsActive = ptrutil.BoolOrTrue(dto.IsActive)
	updated, err := uc.Repo.UpdateTaxParam(ctx, p)
	if err != nil {
		return nil, err
	}
	return toTaxParamResponse(updated), nil
}

func (uc *TaxParamUseCase) GetByID(ctx context.Context, id int64) (*response.TaxParamResponse, error) {
	p, err := uc.Repo.GetTaxParamByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toTaxParamResponse(p), nil
}

func (uc *TaxParamUseCase) List(ctx context.Context, onlyActive bool) ([]*response.TaxParamResponse, error) {
	list, err := uc.Repo.ListTaxParams(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toTaxParamResponses(list), nil
}

func (uc *TaxParamUseCase) ListByUF(ctx context.Context, uf string, onlyActive bool) ([]*response.TaxParamResponse, error) {
	list, err := uc.Repo.ListTaxParamsByUF(ctx, uf, onlyActive)
	if err != nil {
		return nil, err
	}
	return toTaxParamResponses(list), nil
}

func (uc *TaxParamUseCase) ListByItem(ctx context.Context, itemCode int64, onlyActive bool) ([]*response.TaxParamResponse, error) {
	list, err := uc.Repo.ListTaxParamsByItem(ctx, itemCode, onlyActive)
	if err != nil {
		return nil, err
	}
	return toTaxParamResponses(list), nil
}

func (uc *TaxParamUseCase) ListByNCM(ctx context.Context, ncmCode string, onlyActive bool) ([]*response.TaxParamResponse, error) {
	list, err := uc.Repo.ListTaxParamsByNCM(ctx, ncmCode, onlyActive)
	if err != nil {
		return nil, err
	}
	return toTaxParamResponses(list), nil
}

func toReductionTarget(s *string) *entity.IcmsReductionTarget {
	if s == nil {
		return nil
	}
	t := entity.IcmsReductionTarget(*s)
	return &t
}

func toDifalType(s *string) *entity.IcmsDifalType {
	if s == nil {
		return nil
	}
	t := entity.IcmsDifalType(*s)
	return &t
}

func toAcresType(s *string) *entity.IcmsAcresType {
	if s == nil {
		return nil
	}
	t := entity.IcmsAcresType(*s)
	return &t
}

func dtoToTaxParamEntity(dto request.CreateTaxParamDTO) *entity.ICMSIPITaxParam {
	return &entity.ICMSIPITaxParam{
		NCMCode:                        dto.NCMCode,
		ItemCode:                       dto.ItemCode,
		ItemConfigMask:                 dto.ItemConfigMask,
		UF:                             dto.UF,
		OperationType:                  entity.TaxParamOperation(dto.OperationType),
		CustomerCode:                   dto.CustomerCode,
		CustomerEstablishmentCode:      dto.CustomerEstablishmentCode,
		MarketSegmentID:                dto.MarketSegmentID,
		InvoiceTypeExitID:              dto.InvoiceTypeExitID,
		InvoiceTypeEntryID:             dto.InvoiceTypeEntryID,
		TaxTypeID:                      dto.TaxTypeID,
		IsPreferred:                    dto.IsPreferred,
		IsSimpleOptante:                dto.IsSimpleOptante,
		ICMSPctContrib:                 dto.ICMSPctContrib,
		LegalDeviceICMSContribID:       dto.LegalDeviceICMSContribID,
		ICMSPctNonContrib:              dto.ICMSPctNonContrib,
		LegalDeviceICMSNonContribID:    dto.LegalDeviceICMSNonContribID,
		ICMSRedPctContrib:              dto.ICMSRedPctContrib,
		ICMSRedTargetContrib:           toReductionTarget(dto.ICMSRedTargetContrib),
		LegalDeviceICMSRedContribID:    dto.LegalDeviceICMSRedContribID,
		ICMSRedPctNonContrib:           dto.ICMSRedPctNonContrib,
		ICMSRedTargetNonContrib:        toReductionTarget(dto.ICMSRedTargetNonContrib),
		LegalDeviceICMSRedNonContribID: dto.LegalDeviceICMSRedNonContribID,
		ICMSDeferralPct:                dto.ICMSDeferralPct,
		LegalDeviceICMSDeferralID:      dto.LegalDeviceICMSDeferralID,
		CodBenefRBC:                    dto.CodBenefRBC,
		ICMSSubstRedPct:                dto.ICMSSubstRedPct,
		ICMSPctForSTContrib:            dto.ICMSPctForSTContrib,
		ICMSPctForSTNonContrib:         dto.ICMSPctForSTNonContrib,
		CSOSNIGMS:                      dto.CSOSNIGMS,
		CSTICMSContrib:                 dto.CSTICMSContrib,
		CSTICMSNonContrib:              dto.CSTICMSNonContrib,
		CodBeneficioFiscal:             dto.CodBeneficioFiscal,
		IPIRedPctContrib:               dto.IPIRedPctContrib,
		IPIRedTargetContrib:            toReductionTarget(dto.IPIRedTargetContrib),
		LegalDeviceIPIContribID:        dto.LegalDeviceIPIContribID,
		IPIRedPctNonContrib:            dto.IPIRedPctNonContrib,
		IPIRedTargetNonContrib:         toReductionTarget(dto.IPIRedTargetNonContrib),
		LegalDeviceIPINonContribID:     dto.LegalDeviceIPINonContribID,
		CSTIPIExit:                     dto.CSTIPIExit,
		ICMSPctOrigins1238:             dto.ICMSPctOrigins1238,
		UsesICMSZonaFranca:             dto.UsesICMSZonaFranca,
		IPICodigoEnquadramento:         enquadramentoIPI(dto.IPICodigoEnquadramento),
		CodBenefContrib:                dto.CodBenefContrib,
		CodBenefNonContrib:             dto.CodBenefNonContrib,
		ICMSAcresPctNonContrib:         dto.ICMSAcresPctNonContrib,
		ICMSAcresTypeNonContrib:        toAcresType(dto.ICMSAcresTypeNonContrib),
		ICMSDifalRedPct:                dto.ICMSDifalRedPct,
		ICMSDifalType:                  toDifalType(dto.ICMSDifalType),
		FinalidadeCompra:               finalidadeOuNil(dto.FinalidadeCompra),
		MvaPct:                         dto.MvaPct,
		MvaAjustarInterestadual:        ptrutil.BoolOrTrue(dto.MvaAjustarInterestadual),
		SupplierCode:                   dto.SupplierCode,
	}
}

func finalidadeOuNil(f *string) *string {
	if f == nil || strings.TrimSpace(*f) == "" {
		return nil
	}
	v := strings.ToUpper(strings.TrimSpace(*f))
	return &v
}

// conferirCenario: finalidade conhecida, MVA não negativa e CSOSN do Simples
// com três dígitos válidos.
func conferirCenario(dto request.CreateTaxParamDTO) error {
	if f := finalidadeOuNil(dto.FinalidadeCompra); f != nil && !engine.FinalidadeValida(*f) {
		return errorsuc.NewValidationError("finalidade da compra inválida: use REVENDA, INDUSTRIALIZACAO ou USO_CONSUMO")
	}
	if dto.MvaPct.IsNegative() {
		return errorsuc.NewValidationError("a MVA não pode ser negativa")
	}
	if c := dto.CSOSNIGMS; c != nil && strings.TrimSpace(*c) != "" {
		switch strings.TrimSpace(*c) {
		case "101", "102", "103", "201", "202", "203", "300", "400", "500", "900":
		default:
			return errorsuc.NewValidationError("CSOSN inválido: use 101, 102, 103, 201, 202, 203, 300, 400, 500 ou 900")
		}
	}
	return nil
}

// enquadramentoIPI: o cEnq tem 3 dígitos (tabela da Receita); vazio = 999.
func enquadramentoIPI(v *string) *string {
	if v == nil {
		return nil
	}
	d := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, *v)
	if d == "" {
		return nil
	}
	return &d
}
