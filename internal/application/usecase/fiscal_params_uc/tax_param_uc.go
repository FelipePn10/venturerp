package fiscal_params_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type TaxParamUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *TaxParamUseCase) Create(ctx context.Context, dto request.CreateTaxParamDTO) (*response.TaxParamResponse, error) {
	if dto.UF == "" {
		return nil, errors.New("uf is required")
	}
	if dto.NCMCode == nil && dto.ItemCode == nil {
		return nil, errors.New("either ncm_code or item_code must be provided")
	}
	if dto.NCMCode != nil && dto.ItemCode != nil {
		return nil, errors.New("only one of ncm_code or item_code may be provided")
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
		return nil, errors.New("uf is required")
	}
	createDTO := request.CreateTaxParamDTO(dto.CreateTaxParamDTO)
	p := dtoToTaxParamEntity(createDTO)
	p.ID = dto.ID
	p.IsActive = dto.IsActive
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
		NCMCode:                          dto.NCMCode,
		ItemCode:                         dto.ItemCode,
		ItemConfigMask:                   dto.ItemConfigMask,
		UF:                               dto.UF,
		OperationType:                    entity.TaxParamOperation(dto.OperationType),
		CustomerCode:                     dto.CustomerCode,
		CustomerEstablishmentCode:        dto.CustomerEstablishmentCode,
		MarketSegmentID:                  dto.MarketSegmentID,
		InvoiceTypeExitID:                dto.InvoiceTypeExitID,
		InvoiceTypeEntryID:               dto.InvoiceTypeEntryID,
		TaxTypeID:                        dto.TaxTypeID,
		IsPreferred:                      dto.IsPreferred,
		IsSimpleOptante:                  dto.IsSimpleOptante,
		ICMSPctContrib:                   dto.ICMSPctContrib,
		LegalDeviceICMSContribID:         dto.LegalDeviceICMSContribID,
		ICMSPctNonContrib:                dto.ICMSPctNonContrib,
		LegalDeviceICMSNonContribID:      dto.LegalDeviceICMSNonContribID,
		ICMSRedPctContrib:                dto.ICMSRedPctContrib,
		ICMSRedTargetContrib:             toReductionTarget(dto.ICMSRedTargetContrib),
		LegalDeviceICMSRedContribID:      dto.LegalDeviceICMSRedContribID,
		ICMSRedPctNonContrib:             dto.ICMSRedPctNonContrib,
		ICMSRedTargetNonContrib:          toReductionTarget(dto.ICMSRedTargetNonContrib),
		LegalDeviceICMSRedNonContribID:   dto.LegalDeviceICMSRedNonContribID,
		ICMSDeferralPct:                  dto.ICMSDeferralPct,
		ICMSDeferralTarget:               toReductionTarget(dto.ICMSDeferralTarget),
		LegalDeviceICMSDeferralID:        dto.LegalDeviceICMSDeferralID,
		CodBenefRBC:                      dto.CodBenefRBC,
		ICMSSubstPctContrib:              dto.ICMSSubstPctContrib,
		LegalDeviceICMSSubstContribID:    dto.LegalDeviceICMSSubstContribID,
		ICMSSubstPctNonContrib:           dto.ICMSSubstPctNonContrib,
		LegalDeviceICMSSubstNonContribID: dto.LegalDeviceICMSSubstNonContribID,
		ICMSSubstPctContribUC:            dto.ICMSSubstPctContribUC,
		ICMSSubstRedPct:                  dto.ICMSSubstRedPct,
		LegalDeviceICMSSubstRedID:        dto.LegalDeviceICMSSubstRedID,
		ICMSInternalPct:                  dto.ICMSInternalPct,
		BCICMSSTModality:                 dto.BCICMSSTModality,
		ICMSPctForSTContrib:              dto.ICMSPctForSTContrib,
		ICMSPctForSTNonContrib:           dto.ICMSPctForSTNonContrib,
		CSTSituationB:                    dto.CSTSituationB,
		CSOSNIGMS:                        dto.CSOSNIGMS,
		CSTICMSContrib:                   dto.CSTICMSContrib,
		CSTICMSNonContrib:                dto.CSTICMSNonContrib,
		CodBeneficioFiscal:               dto.CodBeneficioFiscal,
		CSTICMSContribDev:                dto.CSTICMSContribDev,
		CSTICMSNonContribDev:             dto.CSTICMSNonContribDev,
		IPIRedPctContrib:                 dto.IPIRedPctContrib,
		IPIRedTargetContrib:              toReductionTarget(dto.IPIRedTargetContrib),
		LegalDeviceIPIContribID:          dto.LegalDeviceIPIContribID,
		IPIRedPctNonContrib:              dto.IPIRedPctNonContrib,
		IPIRedTargetNonContrib:           toReductionTarget(dto.IPIRedTargetNonContrib),
		LegalDeviceIPINonContribID:       dto.LegalDeviceIPINonContribID,
		CSTIPIExit:                       dto.CSTIPIExit,
		CSTIPIEntry:                      dto.CSTIPIEntry,
		ICMSPctOrigins1238:               dto.ICMSPctOrigins1238,
		CalcBaseRedFCI:                   dto.CalcBaseRedFCI,
		ICMSSubstPctOrigins1238:          dto.ICMSSubstPctOrigins1238,
		CSTICMSFci:                       dto.CSTICMSFci,
		UsesICMSZonaFranca:               dto.UsesICMSZonaFranca,
		DifAliqSTContribUC:               dto.DifAliqSTContribUC,
		CodBenefContrib:                  dto.CodBenefContrib,
		CodBenefNonContrib:               dto.CodBenefNonContrib,
		ICMSAcresPctContrib:              dto.ICMSAcresPctContrib,
		ICMSAcresTypeContrib:             toAcresType(dto.ICMSAcresTypeContrib),
		ICMSAcresSumContrib:              dto.ICMSAcresSumContrib,
		ICMSAcresPctNonContrib:           dto.ICMSAcresPctNonContrib,
		ICMSAcresTypeNonContrib:          toAcresType(dto.ICMSAcresTypeNonContrib),
		ICMSAcresSumNonContrib:           dto.ICMSAcresSumNonContrib,
		ICMSSTAcresPctContrib:            dto.ICMSSTAcresPctContrib,
		ICMSSTAcresTypeContrib:           toAcresType(dto.ICMSSTAcresTypeContrib),
		ICMSSTAcresSumContrib:            dto.ICMSSTAcresSumContrib,
		ICMSSTAcresPctNonContrib:         dto.ICMSSTAcresPctNonContrib,
		ICMSSTAcresTypeNonContrib:        toAcresType(dto.ICMSSTAcresTypeNonContrib),
		ICMSSTAcresSumNonContrib:         dto.ICMSSTAcresSumNonContrib,
		FCPSTPartilhaPct:                 dto.FCPSTPartilhaPct,
		ICMSDifalRedPct:                  dto.ICMSDifalRedPct,
		ICMSDifalType:                    toDifalType(dto.ICMSDifalType),
		DifalPurchaseRedPct:              dto.DifalPurchaseRedPct,
		DifalPurchaseRedTarget:           toReductionTarget(dto.DifalPurchaseRedTarget),
	}
}
