package fiscal

import (
	"context"
	"fmt"

	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── FiscalParamsRepositorySQLC ───────────────────────────────────────────────

type FiscalParamsRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

var _ domainrepo.FiscalParamsRepository = (*FiscalParamsRepositorySQLC)(nil)

func NewFiscalParamsRepository(q *sqlc.Queries, pool *pgxpool.Pool) *FiscalParamsRepositorySQLC {
	return &FiscalParamsRepositorySQLC{q: q, pool: pool}
}

// ─── Legal Devices ────────────────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateLegalDevice(ctx context.Context, d *fiscalEntity.LegalDevice) (*fiscalEntity.LegalDevice, error) {
	code, err := r.q.NextLegalDeviceCode(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateLegalDevice(ctx, sqlc.CreateLegalDeviceParams{
		Code:        int64(code),
		Type:        string(d.Type),
		Description: d.Description,
	})
	if err != nil {
		return nil, err
	}
	return legalDeviceToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) UpdateLegalDevice(ctx context.Context, d *fiscalEntity.LegalDevice) (*fiscalEntity.LegalDevice, error) {
	row, err := r.q.UpdateLegalDevice(ctx, sqlc.UpdateLegalDeviceParams{
		ID:          d.ID,
		Type:        string(d.Type),
		Description: d.Description,
		IsActive:    d.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return legalDeviceToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) GetLegalDeviceByCode(ctx context.Context, code int64) (*fiscalEntity.LegalDevice, error) {
	row, err := r.q.GetLegalDeviceByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return legalDeviceToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) ListLegalDevices(ctx context.Context, onlyActive bool) ([]*fiscalEntity.LegalDevice, error) {
	rows, err := r.q.ListLegalDevices(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	result := make([]*fiscalEntity.LegalDevice, len(rows))
	for i, row := range rows {
		result[i] = legalDeviceToEntity(row)
	}
	return result, nil
}

func (r *FiscalParamsRepositorySQLC) ListLegalDevicesByType(ctx context.Context, devType fiscalEntity.LegalDeviceType, onlyActive bool) ([]*fiscalEntity.LegalDevice, error) {
	rows, err := r.q.ListLegalDevicesByType(ctx, sqlc.ListLegalDevicesByTypeParams{
		Type:    string(devType),
		Column2: onlyActive,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*fiscalEntity.LegalDevice, len(rows))
	for i, row := range rows {
		result[i] = legalDeviceToEntity(row)
	}
	return result, nil
}

func (r *FiscalParamsRepositorySQLC) NextLegalDeviceCode(ctx context.Context) (int64, error) {
	code, err := r.q.NextLegalDeviceCode(ctx)
	if err != nil {
		return 0, err
	}
	return int64(code), nil
}

func legalDeviceToEntity(row sqlc.LegalDevice) *fiscalEntity.LegalDevice {
	return &fiscalEntity.LegalDevice{
		ID:          row.ID,
		Code:        row.Code,
		Type:        fiscalEntity.LegalDeviceType(row.Type),
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── CFOP ──────────────────────────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateCFOP(ctx context.Context, c *fiscalEntity.CFOP) (*fiscalEntity.CFOP, error) {
	row, err := r.q.CreateCFOP(ctx, sqlc.CreateCFOPParams{
		Code:            c.Code,
		Description:     c.Description,
		DescriptionFull: pgutil.ToPgTextFromPtr(c.DescriptionFull),
		Utilization:     sqltypes.CfopUtilizationEnum(c.Utilization),
		OrigemClasIpi:   pgutil.ToPgTextFromPtr(c.OrigemClasIPI),
		IndOperacao:     sqltypes.CfopIndOperacaoEnum(c.IndOperacao),
		TipoUtilizacao:  sqltypes.CfopTipoUtilizacaoEnum(c.TipoUtilizacao),
		CodigoAnexoSn:   pgutil.ToPgTextFromPtr(c.CodigoAnexoSN),
		Difal:           c.DIFAL,
		Doacao:          c.Doacao,
	})
	if err != nil {
		return nil, err
	}
	return cfopToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) UpdateCFOP(ctx context.Context, c *fiscalEntity.CFOP) (*fiscalEntity.CFOP, error) {
	row, err := r.q.UpdateCFOP(ctx, sqlc.UpdateCFOPParams{
		ID:              c.ID,
		Description:     c.Description,
		DescriptionFull: pgutil.ToPgTextFromPtr(c.DescriptionFull),
		Utilization:     sqltypes.CfopUtilizationEnum(c.Utilization),
		OrigemClasIpi:   pgutil.ToPgTextFromPtr(c.OrigemClasIPI),
		IndOperacao:     sqltypes.CfopIndOperacaoEnum(c.IndOperacao),
		TipoUtilizacao:  sqltypes.CfopTipoUtilizacaoEnum(c.TipoUtilizacao),
		CodigoAnexoSn:   pgutil.ToPgTextFromPtr(c.CodigoAnexoSN),
		Difal:           c.DIFAL,
		Doacao:          c.Doacao,
		IsActive:        c.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return cfopToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) GetCFOPByCode(ctx context.Context, code int32) (*fiscalEntity.CFOP, error) {
	row, err := r.q.GetCFOPByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return cfopToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) ListCFOPs(ctx context.Context, onlyActive bool) ([]*fiscalEntity.CFOP, error) {
	rows, err := r.q.ListCFOPs(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	result := make([]*fiscalEntity.CFOP, len(rows))
	for i, row := range rows {
		result[i] = cfopToEntity(row)
	}
	return result, nil
}

func (r *FiscalParamsRepositorySQLC) ListCFOPsByDirection(ctx context.Context, direction string, onlyActive bool) ([]*fiscalEntity.CFOP, error) {
	rows, err := r.q.ListCFOPsByDirection(ctx, sqlc.ListCFOPsByDirectionParams{
		Column1: direction,
		Column2: onlyActive,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*fiscalEntity.CFOP, len(rows))
	for i, row := range rows {
		result[i] = cfopToEntity(row)
	}
	return result, nil
}

func cfopToEntity(row sqlc.Cfop) *fiscalEntity.CFOP {
	return &fiscalEntity.CFOP{
		ID:              row.ID,
		Code:            row.Code,
		Description:     row.Description,
		DescriptionFull: pgutil.FromPgTextPtr(row.DescriptionFull),
		Utilization:     fiscalEntity.CfopUtilization(row.Utilization),
		OrigemClasIPI:   pgutil.FromPgTextPtr(row.OrigemClasIpi),
		IndOperacao:     fiscalEntity.CfopIndOperacao(row.IndOperacao),
		TipoUtilizacao:  fiscalEntity.CfopTipoUtilizacao(row.TipoUtilizacao),
		CodigoAnexoSN:   pgutil.FromPgTextPtr(row.CodigoAnexoSn),
		DIFAL:           row.Difal,
		Doacao:          row.Doacao,
		IsActive:        row.IsActive,
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── ICMS/IPI Tax Params ──────────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateTaxParam(ctx context.Context, p *fiscalEntity.ICMSIPITaxParam) (*fiscalEntity.ICMSIPITaxParam, error) {
	row, err := r.q.CreateTaxParam(ctx, taxParamToCreateParams(p))
	if err != nil {
		return nil, err
	}
	return taxParamToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) UpdateTaxParam(ctx context.Context, p *fiscalEntity.ICMSIPITaxParam) (*fiscalEntity.ICMSIPITaxParam, error) {
	row, err := r.q.UpdateTaxParam(ctx, taxParamToUpdateParams(p))
	if err != nil {
		return nil, err
	}
	return taxParamToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) GetTaxParamByID(ctx context.Context, id int64) (*fiscalEntity.ICMSIPITaxParam, error) {
	row, err := r.q.GetTaxParamByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return taxParamToEntity(row), nil
}

func (r *FiscalParamsRepositorySQLC) ListTaxParams(ctx context.Context, onlyActive bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	rows, err := r.q.ListTaxParams(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return taxParamSliceToEntity(rows), nil
}

func (r *FiscalParamsRepositorySQLC) ListTaxParamsByUF(ctx context.Context, uf string, onlyActive bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	rows, err := r.q.ListTaxParamsByUF(ctx, sqlc.ListTaxParamsByUFParams{Uf: uf, Column2: onlyActive})
	if err != nil {
		return nil, err
	}
	return taxParamSliceToEntity(rows), nil
}

func (r *FiscalParamsRepositorySQLC) ListTaxParamsByItem(ctx context.Context, itemCode int64, onlyActive bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	rows, err := r.q.ListTaxParamsByItem(ctx, sqlc.ListTaxParamsByItemParams{ItemCode: &itemCode, Column2: onlyActive})
	if err != nil {
		return nil, err
	}
	return taxParamSliceToEntity(rows), nil
}

func (r *FiscalParamsRepositorySQLC) ListTaxParamsByNCM(ctx context.Context, ncmCode string, onlyActive bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	rows, err := r.q.ListTaxParamsByNCM(ctx, sqlc.ListTaxParamsByNCMParams{
		NcmCode: pgutil.ToPgTextFromString(ncmCode),
		Column2: onlyActive,
	})
	if err != nil {
		return nil, err
	}
	return taxParamSliceToEntity(rows), nil
}

func taxParamSliceToEntity(rows []sqlc.IcmsIpiTaxParam) []*fiscalEntity.ICMSIPITaxParam {
	result := make([]*fiscalEntity.ICMSIPITaxParam, len(rows))
	for i, row := range rows {
		result[i] = taxParamToEntity(row)
	}
	return result
}

func nullReductionTargetFromEntity(v *fiscalEntity.IcmsReductionTarget) sqlc.NullIcmsReductionTargetEnum {
	if v == nil {
		return sqlc.NullIcmsReductionTargetEnum{Valid: false}
	}
	return sqlc.NullIcmsReductionTargetEnum{
		IcmsReductionTargetEnum: sqlc.IcmsReductionTargetEnum(*v),
		Valid:                   true,
	}
}

func nullReductionTargetToEntity(v sqlc.NullIcmsReductionTargetEnum) *fiscalEntity.IcmsReductionTarget {
	if !v.Valid {
		return nil
	}
	t := fiscalEntity.IcmsReductionTarget(v.IcmsReductionTargetEnum)
	return &t
}

func nullDifalTypeFromEntity(v *fiscalEntity.IcmsDifalType) sqlc.NullIcmsDifalTypeEnum {
	if v == nil {
		return sqlc.NullIcmsDifalTypeEnum{Valid: false}
	}
	return sqlc.NullIcmsDifalTypeEnum{
		IcmsDifalTypeEnum: sqlc.IcmsDifalTypeEnum(*v),
		Valid:             true,
	}
}

func nullDifalTypeToEntity(v sqlc.NullIcmsDifalTypeEnum) *fiscalEntity.IcmsDifalType {
	if !v.Valid {
		return nil
	}
	t := fiscalEntity.IcmsDifalType(v.IcmsDifalTypeEnum)
	return &t
}

func nullAcresTypeFromEntity(v *fiscalEntity.IcmsAcresType) sqlc.NullIcmsAcresTypeEnum {
	if v == nil {
		return sqlc.NullIcmsAcresTypeEnum{Valid: false}
	}
	return sqlc.NullIcmsAcresTypeEnum{
		IcmsAcresTypeEnum: sqlc.IcmsAcresTypeEnum(*v),
		Valid:             true,
	}
}

func nullAcresTypeToEntity(v sqlc.NullIcmsAcresTypeEnum) *fiscalEntity.IcmsAcresType {
	if !v.Valid {
		return nil
	}
	t := fiscalEntity.IcmsAcresType(v.IcmsAcresTypeEnum)
	return &t
}

func taxParamToCreateParams(p *fiscalEntity.ICMSIPITaxParam) sqlc.CreateTaxParamParams {
	return sqlc.CreateTaxParamParams{
		NcmCode:                          pgutil.ToPgTextFromPtr(p.NCMCode),
		ItemCode:                         p.ItemCode,
		ItemConfigMask:                   pgutil.ToPgTextFromPtr(p.ItemConfigMask),
		Uf:                               p.UF,
		OperationType:                    sqltypes.TaxParamOperationEnum(p.OperationType),
		CustomerCode:                     p.CustomerCode,
		CustomerEstablishmentCode:        p.CustomerEstablishmentCode,
		MarketSegmentID:                  p.MarketSegmentID,
		InvoiceTypeExitID:                p.InvoiceTypeExitID,
		InvoiceTypeEntryID:               p.InvoiceTypeEntryID,
		TaxTypeID:                        p.TaxTypeID,
		IsPreferred:                      p.IsPreferred,
		IsSimplesOptante:                 p.IsSimpleOptante,
		IcmsPctContrib:                   pgutil.ToPgNumericFromFloat64(p.ICMSPctContrib),
		LegalDeviceIcmsContribID:         p.LegalDeviceICMSContribID,
		IcmsPctNonContrib:                pgutil.ToPgNumericFromFloat64(p.ICMSPctNonContrib),
		LegalDeviceIcmsNonContribID:      p.LegalDeviceICMSNonContribID,
		IcmsRedPctContrib:                pgutil.ToPgNumericFromFloat64(p.ICMSRedPctContrib),
		IcmsRedTargetContrib:             nullReductionTargetFromEntity(p.ICMSRedTargetContrib),
		LegalDeviceIcmsRedContribID:      p.LegalDeviceICMSRedContribID,
		IcmsRedPctNonContrib:             pgutil.ToPgNumericFromFloat64(p.ICMSRedPctNonContrib),
		IcmsRedTargetNonContrib:          nullReductionTargetFromEntity(p.ICMSRedTargetNonContrib),
		LegalDeviceIcmsRedNonContribID:   p.LegalDeviceICMSRedNonContribID,
		IcmsDeferralPct:                  pgutil.ToPgNumericFromFloat64(p.ICMSDeferralPct),
		IcmsDeferralTarget:               nullReductionTargetFromEntity(p.ICMSDeferralTarget),
		LegalDeviceIcmsDeferralID:        p.LegalDeviceICMSDeferralID,
		CodBenefRbc:                      pgutil.ToPgTextFromPtr(p.CodBenefRBC),
		IcmsSubstPctContrib:              pgutil.ToPgNumericFromFloat64(p.ICMSSubstPctContrib),
		LegalDeviceIcmsSubstContribID:    p.LegalDeviceICMSSubstContribID,
		IcmsSubstPctNonContrib:           pgutil.ToPgNumericFromFloat64(p.ICMSSubstPctNonContrib),
		LegalDeviceIcmsSubstNonContribID: p.LegalDeviceICMSSubstNonContribID,
		IcmsSubstPctContribUc:            pgutil.ToPgNumericFromFloat64(p.ICMSSubstPctContribUC),
		IcmsSubstRedPct:                  pgutil.ToPgNumericFromFloat64(p.ICMSSubstRedPct),
		LegalDeviceIcmsSubstRedID:        p.LegalDeviceICMSSubstRedID,
		IcmsInternalPct:                  pgutil.ToPgNumericFromFloat64(p.ICMSInternalPct),
		BcIcmsStModality:                 pgutil.ToPgTextFromPtr(p.BCICMSSTModality),
		IcmsPctForStContrib:              pgutil.ToPgNumericFromFloat64(p.ICMSPctForSTContrib),
		IcmsPctForStNonContrib:           pgutil.ToPgNumericFromFloat64(p.ICMSPctForSTNonContrib),
		CstSituationB:                    pgutil.ToPgTextFromPtr(p.CSTSituationB),
		CsosnIcms:                        pgutil.ToPgTextFromPtr(p.CSOSNIGMS),
		CstIcmsContrib:                   pgutil.ToPgTextFromPtr(p.CSTICMSContrib),
		CstIcmsNonContrib:                pgutil.ToPgTextFromPtr(p.CSTICMSNonContrib),
		CodBeneficioFiscal:               pgutil.ToPgTextFromPtr(p.CodBeneficioFiscal),
		CstIcmsContribDev:                pgutil.ToPgTextFromPtr(p.CSTICMSContribDev),
		CstIcmsNonContribDev:             pgutil.ToPgTextFromPtr(p.CSTICMSNonContribDev),
		IpiRedPctContrib:                 pgutil.ToPgNumericFromFloat64(p.IPIRedPctContrib),
		IpiRedTargetContrib:              nullReductionTargetFromEntity(p.IPIRedTargetContrib),
		LegalDeviceIpiContribID:          p.LegalDeviceIPIContribID,
		IpiRedPctNonContrib:              pgutil.ToPgNumericFromFloat64(p.IPIRedPctNonContrib),
		IpiRedTargetNonContrib:           nullReductionTargetFromEntity(p.IPIRedTargetNonContrib),
		LegalDeviceIpiNonContribID:       p.LegalDeviceIPINonContribID,
		CstIpiExit:                       pgutil.ToPgTextFromPtr(p.CSTIPIExit),
		CstIpiEntry:                      pgutil.ToPgTextFromPtr(p.CSTIPIEntry),
		IcmsPctOrigins1238:               pgutil.ToPgNumericFromFloat64(p.ICMSPctOrigins1238),
		CalcBaseRedFci:                   p.CalcBaseRedFCI,
		IcmsSubstPctOrigins1238:          pgutil.ToPgNumericFromFloat64(p.ICMSSubstPctOrigins1238),
		CstIcmsFci:                       pgutil.ToPgTextFromPtr(p.CSTICMSFci),
		UsesIcmsZonaFranca:               p.UsesICMSZonaFranca,
		DifAliqStContribUc:               pgutil.ToPgNumericFromFloat64(p.DifAliqSTContribUC),
		CodBenefContrib:                  pgutil.ToPgTextFromPtr(p.CodBenefContrib),
		CodBenefNonContrib:               pgutil.ToPgTextFromPtr(p.CodBenefNonContrib),
		IcmsAcresPctContrib:              pgutil.ToPgNumericFromFloat64(p.ICMSAcresPctContrib),
		IcmsAcresTypeContrib:             nullAcresTypeFromEntity(p.ICMSAcresTypeContrib),
		IcmsAcresSumContrib:              p.ICMSAcresSumContrib,
		IcmsAcresPctNonContrib:           pgutil.ToPgNumericFromFloat64(p.ICMSAcresPctNonContrib),
		IcmsAcresTypeNonContrib:          nullAcresTypeFromEntity(p.ICMSAcresTypeNonContrib),
		IcmsAcresSumNonContrib:           p.ICMSAcresSumNonContrib,
		IcmsStAcresPctContrib:            pgutil.ToPgNumericFromFloat64(p.ICMSSTAcresPctContrib),
		IcmsStAcresTypeContrib:           nullAcresTypeFromEntity(p.ICMSSTAcresTypeContrib),
		IcmsStAcresSumContrib:            p.ICMSSTAcresSumContrib,
		IcmsStAcresPctNonContrib:         pgutil.ToPgNumericFromFloat64(p.ICMSSTAcresPctNonContrib),
		IcmsStAcresTypeNonContrib:        nullAcresTypeFromEntity(p.ICMSSTAcresTypeNonContrib),
		IcmsStAcresSumNonContrib:         p.ICMSSTAcresSumNonContrib,
		FcpStPartilhaPct:                 pgutil.ToPgNumericFromFloat64(p.FCPSTPartilhaPct),
		IcmsDifalRedPct:                  pgutil.ToPgNumericFromFloat64(p.ICMSDifalRedPct),
		IcmsDifalType:                    nullDifalTypeFromEntity(p.ICMSDifalType),
		DifalPurchaseRedPct:              pgutil.ToPgNumericFromFloat64(p.DifalPurchaseRedPct),
		DifalPurchaseRedTarget:           nullReductionTargetFromEntity(p.DifalPurchaseRedTarget),
	}
}

func taxParamToUpdateParams(p *fiscalEntity.ICMSIPITaxParam) sqlc.UpdateTaxParamParams {
	cp := taxParamToCreateParams(p)
	return sqlc.UpdateTaxParamParams{
		ID:                               p.ID,
		NcmCode:                          cp.NcmCode,
		ItemCode:                         cp.ItemCode,
		ItemConfigMask:                   cp.ItemConfigMask,
		Uf:                               cp.Uf,
		OperationType:                    cp.OperationType,
		CustomerCode:                     cp.CustomerCode,
		CustomerEstablishmentCode:        cp.CustomerEstablishmentCode,
		MarketSegmentID:                  cp.MarketSegmentID,
		InvoiceTypeExitID:                cp.InvoiceTypeExitID,
		InvoiceTypeEntryID:               cp.InvoiceTypeEntryID,
		TaxTypeID:                        cp.TaxTypeID,
		IsPreferred:                      cp.IsPreferred,
		IsSimplesOptante:                 cp.IsSimplesOptante,
		IcmsPctContrib:                   cp.IcmsPctContrib,
		LegalDeviceIcmsContribID:         cp.LegalDeviceIcmsContribID,
		IcmsPctNonContrib:                cp.IcmsPctNonContrib,
		LegalDeviceIcmsNonContribID:      cp.LegalDeviceIcmsNonContribID,
		IcmsRedPctContrib:                cp.IcmsRedPctContrib,
		IcmsRedTargetContrib:             cp.IcmsRedTargetContrib,
		LegalDeviceIcmsRedContribID:      cp.LegalDeviceIcmsRedContribID,
		IcmsRedPctNonContrib:             cp.IcmsRedPctNonContrib,
		IcmsRedTargetNonContrib:          cp.IcmsRedTargetNonContrib,
		LegalDeviceIcmsRedNonContribID:   cp.LegalDeviceIcmsRedNonContribID,
		IcmsDeferralPct:                  cp.IcmsDeferralPct,
		IcmsDeferralTarget:               cp.IcmsDeferralTarget,
		LegalDeviceIcmsDeferralID:        cp.LegalDeviceIcmsDeferralID,
		CodBenefRbc:                      cp.CodBenefRbc,
		IcmsSubstPctContrib:              cp.IcmsSubstPctContrib,
		LegalDeviceIcmsSubstContribID:    cp.LegalDeviceIcmsSubstContribID,
		IcmsSubstPctNonContrib:           cp.IcmsSubstPctNonContrib,
		LegalDeviceIcmsSubstNonContribID: cp.LegalDeviceIcmsSubstNonContribID,
		IcmsSubstPctContribUc:            cp.IcmsSubstPctContribUc,
		IcmsSubstRedPct:                  cp.IcmsSubstRedPct,
		LegalDeviceIcmsSubstRedID:        cp.LegalDeviceIcmsSubstRedID,
		IcmsInternalPct:                  cp.IcmsInternalPct,
		BcIcmsStModality:                 cp.BcIcmsStModality,
		IcmsPctForStContrib:              cp.IcmsPctForStContrib,
		IcmsPctForStNonContrib:           cp.IcmsPctForStNonContrib,
		CstSituationB:                    cp.CstSituationB,
		CsosnIcms:                        cp.CsosnIcms,
		CstIcmsContrib:                   cp.CstIcmsContrib,
		CstIcmsNonContrib:                cp.CstIcmsNonContrib,
		CodBeneficioFiscal:               cp.CodBeneficioFiscal,
		CstIcmsContribDev:                cp.CstIcmsContribDev,
		CstIcmsNonContribDev:             cp.CstIcmsNonContribDev,
		IpiRedPctContrib:                 cp.IpiRedPctContrib,
		IpiRedTargetContrib:              cp.IpiRedTargetContrib,
		LegalDeviceIpiContribID:          cp.LegalDeviceIpiContribID,
		IpiRedPctNonContrib:              cp.IpiRedPctNonContrib,
		IpiRedTargetNonContrib:           cp.IpiRedTargetNonContrib,
		LegalDeviceIpiNonContribID:       cp.LegalDeviceIpiNonContribID,
		CstIpiExit:                       cp.CstIpiExit,
		CstIpiEntry:                      cp.CstIpiEntry,
		IcmsPctOrigins1238:               cp.IcmsPctOrigins1238,
		CalcBaseRedFci:                   cp.CalcBaseRedFci,
		IcmsSubstPctOrigins1238:          cp.IcmsSubstPctOrigins1238,
		CstIcmsFci:                       cp.CstIcmsFci,
		UsesIcmsZonaFranca:               cp.UsesIcmsZonaFranca,
		DifAliqStContribUc:               cp.DifAliqStContribUc,
		CodBenefContrib:                  cp.CodBenefContrib,
		CodBenefNonContrib:               cp.CodBenefNonContrib,
		IcmsAcresPctContrib:              cp.IcmsAcresPctContrib,
		IcmsAcresTypeContrib:             cp.IcmsAcresTypeContrib,
		IcmsAcresSumContrib:              cp.IcmsAcresSumContrib,
		IcmsAcresPctNonContrib:           cp.IcmsAcresPctNonContrib,
		IcmsAcresTypeNonContrib:          cp.IcmsAcresTypeNonContrib,
		IcmsAcresSumNonContrib:           cp.IcmsAcresSumNonContrib,
		IcmsStAcresPctContrib:            cp.IcmsStAcresPctContrib,
		IcmsStAcresTypeContrib:           cp.IcmsStAcresTypeContrib,
		IcmsStAcresSumContrib:            cp.IcmsStAcresSumContrib,
		IcmsStAcresPctNonContrib:         cp.IcmsStAcresPctNonContrib,
		IcmsStAcresTypeNonContrib:        cp.IcmsStAcresTypeNonContrib,
		IcmsStAcresSumNonContrib:         cp.IcmsStAcresSumNonContrib,
		FcpStPartilhaPct:                 cp.FcpStPartilhaPct,
		IcmsDifalRedPct:                  cp.IcmsDifalRedPct,
		IcmsDifalType:                    cp.IcmsDifalType,
		DifalPurchaseRedPct:              cp.DifalPurchaseRedPct,
		DifalPurchaseRedTarget:           cp.DifalPurchaseRedTarget,
		IsActive:                         p.IsActive,
	}
}

func taxParamToEntity(row sqlc.IcmsIpiTaxParam) *fiscalEntity.ICMSIPITaxParam {
	return &fiscalEntity.ICMSIPITaxParam{
		ID:                               row.ID,
		NCMCode:                          pgutil.FromPgTextPtr(row.NcmCode),
		ItemCode:                         row.ItemCode,
		ItemConfigMask:                   pgutil.FromPgTextPtr(row.ItemConfigMask),
		UF:                               row.Uf,
		OperationType:                    fiscalEntity.TaxParamOperation(row.OperationType),
		CustomerCode:                     row.CustomerCode,
		CustomerEstablishmentCode:        row.CustomerEstablishmentCode,
		MarketSegmentID:                  row.MarketSegmentID,
		InvoiceTypeExitID:                row.InvoiceTypeExitID,
		InvoiceTypeEntryID:               row.InvoiceTypeEntryID,
		TaxTypeID:                        row.TaxTypeID,
		IsPreferred:                      row.IsPreferred,
		IsSimpleOptante:                  row.IsSimplesOptante,
		ICMSPctContrib:                   pgutil.FromPgNumericToFloat64(row.IcmsPctContrib),
		LegalDeviceICMSContribID:         row.LegalDeviceIcmsContribID,
		ICMSPctNonContrib:                pgutil.FromPgNumericToFloat64(row.IcmsPctNonContrib),
		LegalDeviceICMSNonContribID:      row.LegalDeviceIcmsNonContribID,
		ICMSRedPctContrib:                pgutil.FromPgNumericToFloat64(row.IcmsRedPctContrib),
		ICMSRedTargetContrib:             nullReductionTargetToEntity(row.IcmsRedTargetContrib),
		LegalDeviceICMSRedContribID:      row.LegalDeviceIcmsRedContribID,
		ICMSRedPctNonContrib:             pgutil.FromPgNumericToFloat64(row.IcmsRedPctNonContrib),
		ICMSRedTargetNonContrib:          nullReductionTargetToEntity(row.IcmsRedTargetNonContrib),
		LegalDeviceICMSRedNonContribID:   row.LegalDeviceIcmsRedNonContribID,
		ICMSDeferralPct:                  pgutil.FromPgNumericToFloat64(row.IcmsDeferralPct),
		ICMSDeferralTarget:               nullReductionTargetToEntity(row.IcmsDeferralTarget),
		LegalDeviceICMSDeferralID:        row.LegalDeviceIcmsDeferralID,
		CodBenefRBC:                      pgutil.FromPgTextPtr(row.CodBenefRbc),
		ICMSSubstPctContrib:              pgutil.FromPgNumericToFloat64(row.IcmsSubstPctContrib),
		LegalDeviceICMSSubstContribID:    row.LegalDeviceIcmsSubstContribID,
		ICMSSubstPctNonContrib:           pgutil.FromPgNumericToFloat64(row.IcmsSubstPctNonContrib),
		LegalDeviceICMSSubstNonContribID: row.LegalDeviceIcmsSubstNonContribID,
		ICMSSubstPctContribUC:            pgutil.FromPgNumericToFloat64(row.IcmsSubstPctContribUc),
		ICMSSubstRedPct:                  pgutil.FromPgNumericToFloat64(row.IcmsSubstRedPct),
		LegalDeviceICMSSubstRedID:        row.LegalDeviceIcmsSubstRedID,
		ICMSInternalPct:                  pgutil.FromPgNumericToFloat64(row.IcmsInternalPct),
		BCICMSSTModality:                 pgutil.FromPgTextPtr(row.BcIcmsStModality),
		ICMSPctForSTContrib:              pgutil.FromPgNumericToFloat64(row.IcmsPctForStContrib),
		ICMSPctForSTNonContrib:           pgutil.FromPgNumericToFloat64(row.IcmsPctForStNonContrib),
		CSTSituationB:                    pgutil.FromPgTextPtr(row.CstSituationB),
		CSOSNIGMS:                        pgutil.FromPgTextPtr(row.CsosnIcms),
		CSTICMSContrib:                   pgutil.FromPgTextPtr(row.CstIcmsContrib),
		CSTICMSNonContrib:                pgutil.FromPgTextPtr(row.CstIcmsNonContrib),
		CodBeneficioFiscal:               pgutil.FromPgTextPtr(row.CodBeneficioFiscal),
		CSTICMSContribDev:                pgutil.FromPgTextPtr(row.CstIcmsContribDev),
		CSTICMSNonContribDev:             pgutil.FromPgTextPtr(row.CstIcmsNonContribDev),
		IPIRedPctContrib:                 pgutil.FromPgNumericToFloat64(row.IpiRedPctContrib),
		IPIRedTargetContrib:              nullReductionTargetToEntity(row.IpiRedTargetContrib),
		LegalDeviceIPIContribID:          row.LegalDeviceIpiContribID,
		IPIRedPctNonContrib:              pgutil.FromPgNumericToFloat64(row.IpiRedPctNonContrib),
		IPIRedTargetNonContrib:           nullReductionTargetToEntity(row.IpiRedTargetNonContrib),
		LegalDeviceIPINonContribID:       row.LegalDeviceIpiNonContribID,
		CSTIPIExit:                       pgutil.FromPgTextPtr(row.CstIpiExit),
		CSTIPIEntry:                      pgutil.FromPgTextPtr(row.CstIpiEntry),
		ICMSPctOrigins1238:               pgutil.FromPgNumericToFloat64(row.IcmsPctOrigins1238),
		CalcBaseRedFCI:                   row.CalcBaseRedFci,
		ICMSSubstPctOrigins1238:          pgutil.FromPgNumericToFloat64(row.IcmsSubstPctOrigins1238),
		CSTICMSFci:                       pgutil.FromPgTextPtr(row.CstIcmsFci),
		UsesICMSZonaFranca:               row.UsesIcmsZonaFranca,
		DifAliqSTContribUC:               pgutil.FromPgNumericToFloat64(row.DifAliqStContribUc),
		CodBenefContrib:                  pgutil.FromPgTextPtr(row.CodBenefContrib),
		CodBenefNonContrib:               pgutil.FromPgTextPtr(row.CodBenefNonContrib),
		ICMSAcresPctContrib:              pgutil.FromPgNumericToFloat64(row.IcmsAcresPctContrib),
		ICMSAcresTypeContrib:             nullAcresTypeToEntity(row.IcmsAcresTypeContrib),
		ICMSAcresSumContrib:              row.IcmsAcresSumContrib,
		ICMSAcresPctNonContrib:           pgutil.FromPgNumericToFloat64(row.IcmsAcresPctNonContrib),
		ICMSAcresTypeNonContrib:          nullAcresTypeToEntity(row.IcmsAcresTypeNonContrib),
		ICMSAcresSumNonContrib:           row.IcmsAcresSumNonContrib,
		ICMSSTAcresPctContrib:            pgutil.FromPgNumericToFloat64(row.IcmsStAcresPctContrib),
		ICMSSTAcresTypeContrib:           nullAcresTypeToEntity(row.IcmsStAcresTypeContrib),
		ICMSSTAcresSumContrib:            row.IcmsStAcresSumContrib,
		ICMSSTAcresPctNonContrib:         pgutil.FromPgNumericToFloat64(row.IcmsStAcresPctNonContrib),
		ICMSSTAcresTypeNonContrib:        nullAcresTypeToEntity(row.IcmsStAcresTypeNonContrib),
		ICMSSTAcresSumNonContrib:         row.IcmsStAcresSumNonContrib,
		FCPSTPartilhaPct:                 pgutil.FromPgNumericToFloat64(row.FcpStPartilhaPct),
		ICMSDifalRedPct:                  pgutil.FromPgNumericToFloat64(row.IcmsDifalRedPct),
		ICMSDifalType:                    nullDifalTypeToEntity(row.IcmsDifalType),
		DifalPurchaseRedPct:              pgutil.FromPgNumericToFloat64(row.DifalPurchaseRedPct),
		DifalPurchaseRedTarget:           nullReductionTargetToEntity(row.DifalPurchaseRedTarget),
		IsActive:                         row.IsActive,
		CreatedAt:                        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:                        pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

// ─── DAPI Transfer Reasons ────────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateDAPITransferReason(ctx context.Context, d *fiscalEntity.DAPITransferReason) (*fiscalEntity.DAPITransferReason, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO dapi_transfer_reasons (code, reason, destination, valid_from, valid_to, is_active)
		 VALUES ($1,$2,$3,$4,$5,TRUE) RETURNING id, created_at`,
		d.Code, d.Reason, d.Destination, d.ValidFrom, d.ValidTo,
	).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating dapi transfer reason: %w", err)
	}
	return d, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateDAPITransferReason(ctx context.Context, d *fiscalEntity.DAPITransferReason) (*fiscalEntity.DAPITransferReason, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE dapi_transfer_reasons SET reason=$1, destination=$2, valid_from=$3, valid_to=$4, is_active=$5 WHERE id=$6`,
		d.Reason, d.Destination, d.ValidFrom, d.ValidTo, d.IsActive, d.ID)
	if err != nil {
		return nil, fmt.Errorf("updating dapi transfer reason %d: %w", d.ID, err)
	}
	return d, nil
}

func (r *FiscalParamsRepositorySQLC) GetDAPITransferReasonByCode(ctx context.Context, code string) (*fiscalEntity.DAPITransferReason, error) {
	var d fiscalEntity.DAPITransferReason
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, reason, destination, valid_from, valid_to, is_active, created_at FROM dapi_transfer_reasons WHERE code=$1`,
		code).Scan(&d.ID, &d.Code, &d.Reason, &d.Destination, &d.ValidFrom, &d.ValidTo, &d.IsActive, &d.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("dapi transfer reason %s not found", code)
		}
		return nil, fmt.Errorf("getting dapi transfer reason: %w", err)
	}
	return &d, nil
}

func (r *FiscalParamsRepositorySQLC) ListDAPITransferReasons(ctx context.Context, onlyActive bool) ([]*fiscalEntity.DAPITransferReason, error) {
	q := `SELECT id, code, reason, destination, valid_from, valid_to, is_active, created_at FROM dapi_transfer_reasons`
	if onlyActive {
		q += ` WHERE is_active = TRUE`
	}
	q += ` ORDER BY code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("listing dapi transfer reasons: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.DAPITransferReason
	for rows.Next() {
		var d fiscalEntity.DAPITransferReason
		if err := rows.Scan(&d.ID, &d.Code, &d.Reason, &d.Destination, &d.ValidFrom, &d.ValidTo, &d.IsActive, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

// ─── ICMS Apuração Adjustment Codes (tabela 5.1.1) ───────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateICMSApuracaoAdjCode(ctx context.Context, c *fiscalEntity.ICMSApuracaoAdjustmentCode) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_apuracao_adjustment_codes (code, uf, description, valid_from, valid_to, is_active)
		 VALUES ($1,$2,$3,$4,$5,TRUE) RETURNING id, created_at`,
		c.Code, c.UF, c.Description, c.ValidFrom, c.ValidTo,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating icms apuracao adj code: %w", err)
	}
	return c, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateICMSApuracaoAdjCode(ctx context.Context, c *fiscalEntity.ICMSApuracaoAdjustmentCode) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE icms_apuracao_adjustment_codes SET description=$1, valid_from=$2, valid_to=$3, is_active=$4 WHERE id=$5`,
		c.Description, c.ValidFrom, c.ValidTo, c.IsActive, c.ID)
	if err != nil {
		return nil, fmt.Errorf("updating icms apuracao adj code %d: %w", c.ID, err)
	}
	return c, nil
}

func (r *FiscalParamsRepositorySQLC) GetICMSApuracaoAdjCode(ctx context.Context, id int64) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	var c fiscalEntity.ICMSApuracaoAdjustmentCode
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, uf, description, valid_from, valid_to, is_active, created_at FROM icms_apuracao_adjustment_codes WHERE id=$1`, id).
		Scan(&c.ID, &c.Code, &c.UF, &c.Description, &c.ValidFrom, &c.ValidTo, &c.IsActive, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("icms apuracao adj code %d not found", id)
		}
		return nil, fmt.Errorf("getting icms apuracao adj code: %w", err)
	}
	return &c, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSApuracaoAdjCodes(ctx context.Context, uf string, onlyActive bool) ([]*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	q := `SELECT id, code, uf, description, valid_from, valid_to, is_active, created_at FROM icms_apuracao_adjustment_codes WHERE 1=1`
	args := []any{}
	if uf != "" {
		args = append(args, uf)
		q += fmt.Sprintf(" AND uf=$%d", len(args))
	}
	if onlyActive {
		q += ` AND is_active = TRUE`
	}
	q += ` ORDER BY uf, code`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing icms apuracao adj codes: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSApuracaoAdjustmentCode
	for rows.Next() {
		var c fiscalEntity.ICMSApuracaoAdjustmentCode
		if err := rows.Scan(&c.ID, &c.Code, &c.UF, &c.Description, &c.ValidFrom, &c.ValidTo, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ─── ICMS Adjustment Codes (tabelas 5.2/5.3/5.6/5.7) ────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateICMSAdjustmentCode(ctx context.Context, c *fiscalEntity.ICMSAdjustmentCode) (*fiscalEntity.ICMSAdjustmentCode, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_adjustment_codes (uf, code, description, table_ref, valid_from, valid_to, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,TRUE) RETURNING id, created_at`,
		c.UF, c.Code, c.Description, string(c.TableRef), c.ValidFrom, c.ValidTo,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating icms adjustment code: %w", err)
	}
	return c, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateICMSAdjustmentCode(ctx context.Context, c *fiscalEntity.ICMSAdjustmentCode) (*fiscalEntity.ICMSAdjustmentCode, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE icms_adjustment_codes SET description=$1, valid_from=$2, valid_to=$3, is_active=$4 WHERE id=$5`,
		c.Description, c.ValidFrom, c.ValidTo, c.IsActive, c.ID)
	if err != nil {
		return nil, fmt.Errorf("updating icms adjustment code %d: %w", c.ID, err)
	}
	return c, nil
}

func (r *FiscalParamsRepositorySQLC) GetICMSAdjustmentCode(ctx context.Context, id int64) (*fiscalEntity.ICMSAdjustmentCode, error) {
	var c fiscalEntity.ICMSAdjustmentCode
	var tr string
	err := r.pool.QueryRow(ctx,
		`SELECT id, uf, code, description, table_ref, valid_from, valid_to, is_active, created_at FROM icms_adjustment_codes WHERE id=$1`, id).
		Scan(&c.ID, &c.UF, &c.Code, &c.Description, &tr, &c.ValidFrom, &c.ValidTo, &c.IsActive, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("icms adjustment code %d not found", id)
		}
		return nil, fmt.Errorf("getting icms adjustment code: %w", err)
	}
	c.TableRef = fiscalEntity.ICMSAdjustmentTableRef(tr)
	return &c, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSAdjustmentCodes(ctx context.Context, uf string, tableRef string, onlyActive bool) ([]*fiscalEntity.ICMSAdjustmentCode, error) {
	q := `SELECT id, uf, code, description, table_ref, valid_from, valid_to, is_active, created_at FROM icms_adjustment_codes WHERE 1=1`
	args := []any{}
	if uf != "" {
		args = append(args, uf)
		q += fmt.Sprintf(" AND uf=$%d", len(args))
	}
	if tableRef != "" {
		args = append(args, tableRef)
		q += fmt.Sprintf(" AND table_ref=$%d", len(args))
	}
	if onlyActive {
		q += ` AND is_active = TRUE`
	}
	q += ` ORDER BY uf, code`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing icms adjustment codes: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSAdjustmentCode
	for rows.Next() {
		var c fiscalEntity.ICMSAdjustmentCode
		var tr string
		if err := rows.Scan(&c.ID, &c.UF, &c.Code, &c.Description, &tr, &c.ValidFrom, &c.ValidTo, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.TableRef = fiscalEntity.ICMSAdjustmentTableRef(tr)
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ─── ICMS Apuração Lines ──────────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateICMSApuracaoLine(ctx context.Context, l *fiscalEntity.ICMSApuracaoLine) (*fiscalEntity.ICMSApuracaoLine, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_apuracao_lines (code, description, line_type, accepts_entries, nature, apuracao_adjustment_code_id, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,TRUE) RETURNING id, created_at`,
		l.Code, l.Description, string(l.LineType), l.AcceptsEntries, l.Nature, l.ApuracaoAdjCodeID,
	).Scan(&l.ID, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating icms apuracao line: %w", err)
	}
	return l, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateICMSApuracaoLine(ctx context.Context, l *fiscalEntity.ICMSApuracaoLine) (*fiscalEntity.ICMSApuracaoLine, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE icms_apuracao_lines SET description=$1, line_type=$2, accepts_entries=$3, nature=$4, apuracao_adjustment_code_id=$5, is_active=$6 WHERE id=$7`,
		l.Description, string(l.LineType), l.AcceptsEntries, l.Nature, l.ApuracaoAdjCodeID, l.IsActive, l.ID)
	if err != nil {
		return nil, fmt.Errorf("updating icms apuracao line %d: %w", l.ID, err)
	}
	return l, nil
}

func (r *FiscalParamsRepositorySQLC) GetICMSApuracaoLine(ctx context.Context, code string) (*fiscalEntity.ICMSApuracaoLine, error) {
	var l fiscalEntity.ICMSApuracaoLine
	var lt string
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, description, line_type, accepts_entries, nature, apuracao_adjustment_code_id, is_active, created_at FROM icms_apuracao_lines WHERE code=$1`,
		code).Scan(&l.ID, &l.Code, &l.Description, &lt, &l.AcceptsEntries, &l.Nature, &l.ApuracaoAdjCodeID, &l.IsActive, &l.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("icms apuracao line %s not found", code)
		}
		return nil, fmt.Errorf("getting icms apuracao line: %w", err)
	}
	l.LineType = fiscalEntity.ApuracaoLineType(lt)
	return &l, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSApuracaoLines(ctx context.Context, onlyActive bool) ([]*fiscalEntity.ICMSApuracaoLine, error) {
	q := `SELECT id, code, description, line_type, accepts_entries, nature, apuracao_adjustment_code_id, is_active, created_at FROM icms_apuracao_lines`
	if onlyActive {
		q += ` WHERE is_active = TRUE`
	}
	q += ` ORDER BY code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("listing icms apuracao lines: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSApuracaoLine
	for rows.Next() {
		var l fiscalEntity.ICMSApuracaoLine
		var lt string
		if err := rows.Scan(&l.ID, &l.Code, &l.Description, &lt, &l.AcceptsEntries, &l.Nature, &l.ApuracaoAdjCodeID, &l.IsActive, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.LineType = fiscalEntity.ApuracaoLineType(lt)
		out = append(out, &l)
	}
	return out, rows.Err()
}

// ─── ICMS Summary Entries ─────────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateICMSSummaryEntry(ctx context.Context, e *fiscalEntity.ICMSSummaryEntry) (*fiscalEntity.ICMSSummaryEntry, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_summary_entries (period, uf, cfop_id, icms_base, icms_value, icms_base_other, icms_value_other, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE) RETURNING id, created_at`,
		e.Period, e.UF, e.CFOPID, e.ICMSBase, e.ICMSValue, e.ICMSBaseOther, e.ICMSValueOther,
	).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating icms summary entry: %w", err)
	}
	return e, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateICMSSummaryEntry(ctx context.Context, e *fiscalEntity.ICMSSummaryEntry) (*fiscalEntity.ICMSSummaryEntry, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE icms_summary_entries SET icms_base=$1, icms_value=$2, icms_base_other=$3, icms_value_other=$4, is_active=$5 WHERE id=$6`,
		e.ICMSBase, e.ICMSValue, e.ICMSBaseOther, e.ICMSValueOther, e.IsActive, e.ID)
	if err != nil {
		return nil, fmt.Errorf("updating icms summary entry %d: %w", e.ID, err)
	}
	return e, nil
}

func (r *FiscalParamsRepositorySQLC) GetICMSSummaryEntry(ctx context.Context, id int64) (*fiscalEntity.ICMSSummaryEntry, error) {
	var e fiscalEntity.ICMSSummaryEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, period, uf, cfop_id, icms_base, icms_value, icms_base_other, icms_value_other, is_active, created_at FROM icms_summary_entries WHERE id=$1`, id).
		Scan(&e.ID, &e.Period, &e.UF, &e.CFOPID, &e.ICMSBase, &e.ICMSValue, &e.ICMSBaseOther, &e.ICMSValueOther, &e.IsActive, &e.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("icms summary entry %d not found", id)
		}
		return nil, fmt.Errorf("getting icms summary entry: %w", err)
	}
	return &e, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSSummaryEntries(ctx context.Context, period string, uf string) ([]*fiscalEntity.ICMSSummaryEntry, error) {
	q := `SELECT id, period, uf, cfop_id, icms_base, icms_value, icms_base_other, icms_value_other, is_active, created_at FROM icms_summary_entries WHERE 1=1`
	args := []any{}
	if period != "" {
		args = append(args, period)
		q += fmt.Sprintf(" AND period=$%d", len(args))
	}
	if uf != "" {
		args = append(args, uf)
		q += fmt.Sprintf(" AND uf=$%d", len(args))
	}
	q += ` ORDER BY period, uf`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing icms summary entries: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSSummaryEntry
	for rows.Next() {
		var e fiscalEntity.ICMSSummaryEntry
		if err := rows.Scan(&e.ID, &e.Period, &e.UF, &e.CFOPID, &e.ICMSBase, &e.ICMSValue, &e.ICMSBaseOther, &e.ICMSValueOther, &e.IsActive, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *FiscalParamsRepositorySQLC) AddICMSSummaryEntryNote(ctx context.Context, n *fiscalEntity.ICMSSummaryEntryNote) (*fiscalEntity.ICMSSummaryEntryNote, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_summary_entry_notes (summary_entry_id, note_number, note_series, emitter_cnpj, issue_date, item_value, icms_base, icms_value, observation)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at`,
		n.SummaryEntryID, n.NoteNumber, n.NoteSeries, n.EmitterCNPJ, n.IssueDate, n.ItemValue, n.ICMSBase, n.ICMSValue, n.Observation,
	).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding icms summary entry note: %w", err)
	}
	return n, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSSummaryEntryNotes(ctx context.Context, summaryEntryID int64) ([]*fiscalEntity.ICMSSummaryEntryNote, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, summary_entry_id, note_number, note_series, emitter_cnpj, issue_date, item_value, icms_base, icms_value, observation, created_at
		 FROM icms_summary_entry_notes WHERE summary_entry_id=$1 ORDER BY issue_date`, summaryEntryID)
	if err != nil {
		return nil, fmt.Errorf("listing icms summary entry notes: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSSummaryEntryNote
	for rows.Next() {
		var n fiscalEntity.ICMSSummaryEntryNote
		if err := rows.Scan(&n.ID, &n.SummaryEntryID, &n.NoteNumber, &n.NoteSeries, &n.EmitterCNPJ, &n.IssueDate, &n.ItemValue, &n.ICMSBase, &n.ICMSValue, &n.Observation, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

// ─── Simples Nacional Apuração ────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateSimplesNacionalApuracao(ctx context.Context, s *fiscalEntity.SimplesNacionalApuracao) (*fiscalEntity.SimplesNacionalApuracao, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO simples_nacional_apuracoes
		 (period, annex, receita_interna, receita_externa, folha_pagamento, receita_bruta_12m, simples_recolhido, aliquota_nominal, aliquota_efetiva, aliquota_efetiva_icms, parcela_deduzir, observation, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,TRUE) RETURNING id, created_at`,
		s.Period, string(s.Annex), s.ReceitaInterna, s.ReceitaExterna, s.FolhaPagamento,
		s.ReceitaBruta12M, s.SimplesRecolhido, s.AliquotaNominal, s.AliquotaEfetiva,
		s.AliquotaEfetivaICMS, s.ParcelaDeduzir, s.Observation,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating simples nacional apuracao: %w", err)
	}
	return s, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateSimplesNacionalApuracao(ctx context.Context, s *fiscalEntity.SimplesNacionalApuracao) (*fiscalEntity.SimplesNacionalApuracao, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE simples_nacional_apuracoes
		 SET receita_interna=$1, receita_externa=$2, folha_pagamento=$3, receita_bruta_12m=$4, simples_recolhido=$5,
		     aliquota_nominal=$6, aliquota_efetiva=$7, aliquota_efetiva_icms=$8, parcela_deduzir=$9, observation=$10, is_active=$11
		 WHERE id=$12`,
		s.ReceitaInterna, s.ReceitaExterna, s.FolhaPagamento, s.ReceitaBruta12M, s.SimplesRecolhido,
		s.AliquotaNominal, s.AliquotaEfetiva, s.AliquotaEfetivaICMS, s.ParcelaDeduzir, s.Observation, s.IsActive, s.ID)
	if err != nil {
		return nil, fmt.Errorf("updating simples nacional apuracao %d: %w", s.ID, err)
	}
	return s, nil
}

func (r *FiscalParamsRepositorySQLC) GetSimplesNacionalApuracao(ctx context.Context, period string, annex fiscalEntity.SimplesNacionalAnnex) (*fiscalEntity.SimplesNacionalApuracao, error) {
	var s fiscalEntity.SimplesNacionalApuracao
	var ann string
	err := r.pool.QueryRow(ctx,
		`SELECT id, period, annex, receita_interna, receita_externa, folha_pagamento, receita_bruta_12m, simples_recolhido, aliquota_nominal, aliquota_efetiva, aliquota_efetiva_icms, parcela_deduzir, observation, is_active, created_at
		 FROM simples_nacional_apuracoes WHERE period=$1 AND annex=$2`, period, string(annex)).
		Scan(&s.ID, &s.Period, &ann, &s.ReceitaInterna, &s.ReceitaExterna, &s.FolhaPagamento,
			&s.ReceitaBruta12M, &s.SimplesRecolhido, &s.AliquotaNominal, &s.AliquotaEfetiva,
			&s.AliquotaEfetivaICMS, &s.ParcelaDeduzir, &s.Observation, &s.IsActive, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("simples nacional apuracao %s/%s not found", period, annex)
		}
		return nil, fmt.Errorf("getting simples nacional apuracao: %w", err)
	}
	s.Annex = fiscalEntity.SimplesNacionalAnnex(ann)
	return &s, nil
}

func (r *FiscalParamsRepositorySQLC) ListSimplesNacionalApuracoes(ctx context.Context, period string) ([]*fiscalEntity.SimplesNacionalApuracao, error) {
	q := `SELECT id, period, annex, receita_interna, receita_externa, folha_pagamento, receita_bruta_12m, simples_recolhido, aliquota_nominal, aliquota_efetiva, aliquota_efetiva_icms, parcela_deduzir, observation, is_active, created_at FROM simples_nacional_apuracoes`
	args := []any{}
	if period != "" {
		args = append(args, period)
		q += fmt.Sprintf(" WHERE period=$%d", len(args))
	}
	q += ` ORDER BY period, annex`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing simples nacional apuracoes: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.SimplesNacionalApuracao
	for rows.Next() {
		var s fiscalEntity.SimplesNacionalApuracao
		var ann string
		if err := rows.Scan(&s.ID, &s.Period, &ann, &s.ReceitaInterna, &s.ReceitaExterna, &s.FolhaPagamento,
			&s.ReceitaBruta12M, &s.SimplesRecolhido, &s.AliquotaNominal, &s.AliquotaEfetiva,
			&s.AliquotaEfetivaICMS, &s.ParcelaDeduzir, &s.Observation, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.Annex = fiscalEntity.SimplesNacionalAnnex(ann)
		out = append(out, &s)
	}
	return out, rows.Err()
}

// ─── ICMS Reduction / Substitution / Deferral ────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateICMSReductionSubstitution(ctx context.Context, rs *fiscalEntity.ICMSReductionSubstitution) (*fiscalEntity.ICMSReductionSubstitution, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_reduction_substitutions
		 (item_id, item_mask, ncm_code, uf, operation_type, customer_id, establishment_id, supplier_id,
		  invoice_type_out_id, invoice_type_in_id, market_segment_id, is_preferential,
		  icms_pct_contrib, icms_pct_non_contrib, legal_device_icms_contrib_id, legal_device_icms_non_contrib_id,
		  icms_red_pct_contrib, icms_red_target_contrib, icms_red_pct_non_contrib, icms_red_target_non_contrib,
		  legal_device_icms_red_contrib_id, legal_device_icms_red_non_contrib_id,
		  icms_deferral_pct, icms_deferral_target, legal_device_icms_deferral_id, icms_deferral_benefit_code,
		  icms_subst_pct_contrib, icms_subst_pct_non_contrib, icms_subst_pct_contrib_uc, icms_subst_red_pct,
		  icms_internal_pct, legal_device_icms_subst_contrib_id, legal_device_icms_subst_non_contrib_id,
		  legal_device_icms_subst_red_id, mod_bc_icms_st, icms_pct_for_st_contrib, icms_pct_for_st_non_contrib,
		  cst_icms_contrib, cst_icms_non_contrib, csosn_icms, cst_icms_contrib_dev, cst_icms_non_contrib_dev,
		  fiscal_benefit_code_contrib, fiscal_benefit_code_non_contrib, fiscal_benefit_code,
		  ipi_red_pct_contrib, ipi_red_target_contrib, ipi_red_pct_non_contrib, ipi_red_target_non_contrib,
		  legal_device_ipi_contrib_id, legal_device_ipi_non_contrib_id, cst_ipi_out, cst_ipi_in,
		  fci_icms_pct, fci_reduce_base, fci_icms_subst_pct, fci_cst_icms, fci_use_icms_zf, fci_difal_st_contrib_uc_pct,
		  icms_add_pct_contrib, icms_add_type_contrib, icms_add_sum_aliq_contrib,
		  icms_add_pct_non_contrib, icms_add_type_non_contrib, icms_add_sum_aliq_non_contrib,
		  icms_st_add_pct_contrib, icms_st_add_type_contrib, icms_st_add_pct_non_contrib, icms_st_add_type_non_contrib,
		  fcp_partition_pct, difal_icms_red_pct, difal_icms_type, difal_purchase_red_pct,
		  is_simples_optante, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		         $21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,
		         $38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,
		         $55,$56,$57,$58,$59,$60,$61,$62,$63,$64,$65,$66,$67,$68,$69,$70,$71,$72,TRUE)
		 RETURNING id, created_at`,
		pgutil.ToPgInt8Ptr(rs.ItemID), pgutil.ToPgTextFromPtr(rs.ItemMask), pgutil.ToPgTextFromPtr(rs.NCMCode),
		rs.UF, string(rs.OperationType), pgutil.ToPgInt8Ptr(rs.CustomerID), pgutil.ToPgInt8Ptr(rs.EstablishmentID),
		pgutil.ToPgInt8Ptr(rs.SupplierID), pgutil.ToPgInt8Ptr(rs.InvoiceTypeOutID), pgutil.ToPgInt8Ptr(rs.InvoiceTypeInID),
		pgutil.ToPgInt8Ptr(rs.MarketSegmentID), rs.IsPreferential,
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSPctContrib), pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSPctNonContrib),
		pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSContribID), pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSNonContribID),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSRedPctContrib), string(rs.ICMSRedTargetContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSRedPctNonContrib), string(rs.ICMSRedTargetNonContrib),
		pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSRedContribID), pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSRedNonContribID),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSDeferralPct), string(rs.ICMSDeferralTarget),
		pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSDeferralID), pgutil.ToPgTextFromPtr(rs.ICMSDeferralBenefitCode),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSubstPctContrib), pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSubstPctNonContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSubstPctContribUC), pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSubstRedPct),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSInternalPct),
		pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSSubstContribID), pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSSubstNonContribID),
		pgutil.ToPgInt8Ptr(rs.LegalDeviceICMSSubstRedID), pgutil.ToPgTextFromPtr(rs.ModBCICMSST),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSPctForSTContrib), pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSPctForSTNonContrib),
		pgutil.ToPgTextFromPtr(rs.CSTICMSContrib), pgutil.ToPgTextFromPtr(rs.CSTICMSNonContrib),
		pgutil.ToPgTextFromPtr(rs.CSOSNTICMS), pgutil.ToPgTextFromPtr(rs.CSTICMSContribDev), pgutil.ToPgTextFromPtr(rs.CSTICMSNonContribDev),
		pgutil.ToPgTextFromPtr(rs.FiscalBenefitCodeContrib), pgutil.ToPgTextFromPtr(rs.FiscalBenefitCodeNonContrib), pgutil.ToPgTextFromPtr(rs.FiscalBenefitCode),
		pgutil.ToPgNumericFromFloat64Ptr(rs.IPIRedPctContrib), string(rs.IPIRedTargetContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.IPIRedPctNonContrib), string(rs.IPIRedTargetNonContrib),
		pgutil.ToPgInt8Ptr(rs.LegalDeviceIPIContribID), pgutil.ToPgInt8Ptr(rs.LegalDeviceIPINonContribID),
		pgutil.ToPgTextFromPtr(rs.CSTIPIOut), pgutil.ToPgTextFromPtr(rs.CSTIPIIn),
		pgutil.ToPgNumericFromFloat64Ptr(rs.FCIICMSPct), rs.FCIReduceBase,
		pgutil.ToPgNumericFromFloat64Ptr(rs.FCIICMSSubstPct), pgutil.ToPgTextFromPtr(rs.FCICSTICMs),
		rs.FCIUseICMSZF, pgutil.ToPgNumericFromFloat64Ptr(rs.FCIDIFALSTContribUCPct),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSAddPctContrib), pgutil.ToPgTextFromPtr(rs.ICMSAddTypeContrib), rs.ICMSAddSumAliqContrib,
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSAddPctNonContrib), pgutil.ToPgTextFromPtr(rs.ICMSAddTypeNonContrib), rs.ICMSAddSumAliqNonContrib,
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSTAddPctContrib), pgutil.ToPgTextFromPtr(rs.ICMSSTAddTypeContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSTAddPctNonContrib), pgutil.ToPgTextFromPtr(rs.ICMSSTAddTypeNonContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.FCPPartitionPct),
		pgutil.ToPgNumericFromFloat64Ptr(rs.DIFALICMSRedPct), pgutil.ToPgTextFromPtr(rs.DIFALICMSType),
		pgutil.ToPgNumericFromFloat64Ptr(rs.DIFALPurchaseRedPct), rs.IsSimplesOptante,
	).Scan(&rs.ID, &rs.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating icms reduction substitution: %w", err)
	}
	return rs, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateICMSReductionSubstitution(ctx context.Context, rs *fiscalEntity.ICMSReductionSubstitution) (*fiscalEntity.ICMSReductionSubstitution, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE icms_reduction_substitutions SET
		  uf=$1, operation_type=$2, is_preferential=$3,
		  icms_pct_contrib=$4, icms_pct_non_contrib=$5,
		  icms_red_pct_contrib=$6, icms_red_target_contrib=$7, icms_red_pct_non_contrib=$8, icms_red_target_non_contrib=$9,
		  icms_deferral_pct=$10, icms_deferral_target=$11, icms_deferral_benefit_code=$12,
		  icms_subst_pct_contrib=$13, icms_subst_pct_non_contrib=$14,
		  cst_icms_contrib=$15, cst_icms_non_contrib=$16, csosn_icms=$17,
		  ipi_red_pct_contrib=$18, ipi_red_target_contrib=$19, ipi_red_pct_non_contrib=$20, ipi_red_target_non_contrib=$21,
		  cst_ipi_out=$22, cst_ipi_in=$23, fiscal_benefit_code=$24,
		  difal_icms_red_pct=$25, difal_icms_type=$26, difal_purchase_red_pct=$27,
		  is_simples_optante=$28, is_active=$29
		 WHERE id=$30`,
		rs.UF, string(rs.OperationType), rs.IsPreferential,
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSPctContrib), pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSPctNonContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSRedPctContrib), string(rs.ICMSRedTargetContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSRedPctNonContrib), string(rs.ICMSRedTargetNonContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSDeferralPct), string(rs.ICMSDeferralTarget),
		pgutil.ToPgTextFromPtr(rs.ICMSDeferralBenefitCode),
		pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSubstPctContrib), pgutil.ToPgNumericFromFloat64Ptr(rs.ICMSSubstPctNonContrib),
		pgutil.ToPgTextFromPtr(rs.CSTICMSContrib), pgutil.ToPgTextFromPtr(rs.CSTICMSNonContrib), pgutil.ToPgTextFromPtr(rs.CSOSNTICMS),
		pgutil.ToPgNumericFromFloat64Ptr(rs.IPIRedPctContrib), string(rs.IPIRedTargetContrib),
		pgutil.ToPgNumericFromFloat64Ptr(rs.IPIRedPctNonContrib), string(rs.IPIRedTargetNonContrib),
		pgutil.ToPgTextFromPtr(rs.CSTIPIOut), pgutil.ToPgTextFromPtr(rs.CSTIPIIn),
		pgutil.ToPgTextFromPtr(rs.FiscalBenefitCode),
		pgutil.ToPgNumericFromFloat64Ptr(rs.DIFALICMSRedPct), pgutil.ToPgTextFromPtr(rs.DIFALICMSType),
		pgutil.ToPgNumericFromFloat64Ptr(rs.DIFALPurchaseRedPct), rs.IsSimplesOptante, rs.IsActive, rs.ID)
	if err != nil {
		return nil, fmt.Errorf("updating icms reduction substitution %d: %w", rs.ID, err)
	}
	return rs, nil
}

func (r *FiscalParamsRepositorySQLC) GetICMSReductionSubstitution(ctx context.Context, id int64) (*fiscalEntity.ICMSReductionSubstitution, error) {
	var rs fiscalEntity.ICMSReductionSubstitution
	var opType, redTargCtrib, redTargNonCtrib, defTarget, ipiRedTargCtrib, ipiRedTargNonCtrib string
	err := r.pool.QueryRow(ctx,
		`SELECT id, item_id, item_mask, ncm_code, uf, operation_type, customer_id, establishment_id, supplier_id,
		 invoice_type_out_id, invoice_type_in_id, market_segment_id, is_preferential,
		 icms_pct_contrib, icms_pct_non_contrib, icms_red_pct_contrib, icms_red_target_contrib,
		 icms_red_pct_non_contrib, icms_red_target_non_contrib, icms_deferral_pct, icms_deferral_target,
		 icms_deferral_benefit_code, icms_subst_pct_contrib, icms_subst_pct_non_contrib,
		 cst_icms_contrib, cst_icms_non_contrib, csosn_icms, cst_icms_contrib_dev, cst_icms_non_contrib_dev,
		 fiscal_benefit_code_contrib, fiscal_benefit_code_non_contrib, fiscal_benefit_code,
		 ipi_red_pct_contrib, ipi_red_target_contrib, ipi_red_pct_non_contrib, ipi_red_target_non_contrib,
		 cst_ipi_out, cst_ipi_in, fci_icms_pct, fci_reduce_base, fci_icms_subst_pct, fci_cst_icms, fci_use_icms_zf,
		 icms_add_pct_contrib, icms_add_type_contrib, icms_add_sum_aliq_contrib,
		 icms_add_pct_non_contrib, icms_add_type_non_contrib, icms_add_sum_aliq_non_contrib,
		 fcp_partition_pct, difal_icms_red_pct, difal_icms_type, difal_purchase_red_pct,
		 is_simples_optante, is_active, created_at
		 FROM icms_reduction_substitutions WHERE id=$1`, id).
		Scan(&rs.ID, pgutil.ScanPgInt8Ptr(&rs.ItemID), pgutil.ScanPgTextPtr(&rs.ItemMask),
			pgutil.ScanPgTextPtr(&rs.NCMCode), &rs.UF, &opType,
			pgutil.ScanPgInt8Ptr(&rs.CustomerID), pgutil.ScanPgInt8Ptr(&rs.EstablishmentID),
			pgutil.ScanPgInt8Ptr(&rs.SupplierID), pgutil.ScanPgInt8Ptr(&rs.InvoiceTypeOutID),
			pgutil.ScanPgInt8Ptr(&rs.InvoiceTypeInID), pgutil.ScanPgInt8Ptr(&rs.MarketSegmentID), &rs.IsPreferential,
			pgutil.ScanPgNumericPtr(&rs.ICMSPctContrib), pgutil.ScanPgNumericPtr(&rs.ICMSPctNonContrib),
			pgutil.ScanPgNumericPtr(&rs.ICMSRedPctContrib), &redTargCtrib,
			pgutil.ScanPgNumericPtr(&rs.ICMSRedPctNonContrib), &redTargNonCtrib,
			pgutil.ScanPgNumericPtr(&rs.ICMSDeferralPct), &defTarget,
			pgutil.ScanPgTextPtr(&rs.ICMSDeferralBenefitCode),
			pgutil.ScanPgNumericPtr(&rs.ICMSSubstPctContrib), pgutil.ScanPgNumericPtr(&rs.ICMSSubstPctNonContrib),
			pgutil.ScanPgTextPtr(&rs.CSTICMSContrib), pgutil.ScanPgTextPtr(&rs.CSTICMSNonContrib),
			pgutil.ScanPgTextPtr(&rs.CSOSNTICMS), pgutil.ScanPgTextPtr(&rs.CSTICMSContribDev), pgutil.ScanPgTextPtr(&rs.CSTICMSNonContribDev),
			pgutil.ScanPgTextPtr(&rs.FiscalBenefitCodeContrib), pgutil.ScanPgTextPtr(&rs.FiscalBenefitCodeNonContrib),
			pgutil.ScanPgTextPtr(&rs.FiscalBenefitCode),
			pgutil.ScanPgNumericPtr(&rs.IPIRedPctContrib), &ipiRedTargCtrib,
			pgutil.ScanPgNumericPtr(&rs.IPIRedPctNonContrib), &ipiRedTargNonCtrib,
			pgutil.ScanPgTextPtr(&rs.CSTIPIOut), pgutil.ScanPgTextPtr(&rs.CSTIPIIn),
			pgutil.ScanPgNumericPtr(&rs.FCIICMSPct), &rs.FCIReduceBase,
			pgutil.ScanPgNumericPtr(&rs.FCIICMSSubstPct), pgutil.ScanPgTextPtr(&rs.FCICSTICMs), &rs.FCIUseICMSZF,
			pgutil.ScanPgNumericPtr(&rs.ICMSAddPctContrib), pgutil.ScanPgTextPtr(&rs.ICMSAddTypeContrib), &rs.ICMSAddSumAliqContrib,
			pgutil.ScanPgNumericPtr(&rs.ICMSAddPctNonContrib), pgutil.ScanPgTextPtr(&rs.ICMSAddTypeNonContrib), &rs.ICMSAddSumAliqNonContrib,
			pgutil.ScanPgNumericPtr(&rs.FCPPartitionPct),
			pgutil.ScanPgNumericPtr(&rs.DIFALICMSRedPct), pgutil.ScanPgTextPtr(&rs.DIFALICMSType),
			pgutil.ScanPgNumericPtr(&rs.DIFALPurchaseRedPct), &rs.IsSimplesOptante, &rs.IsActive, &rs.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("icms reduction substitution %d not found", id)
		}
		return nil, fmt.Errorf("getting icms reduction substitution: %w", err)
	}
	rs.OperationType = fiscalEntity.ICMSOperationType(opType)
	rs.ICMSRedTargetContrib = fiscalEntity.ICMSReductionTarget(redTargCtrib)
	rs.ICMSRedTargetNonContrib = fiscalEntity.ICMSReductionTarget(redTargNonCtrib)
	rs.ICMSDeferralTarget = fiscalEntity.ICMSReductionTarget(defTarget)
	rs.IPIRedTargetContrib = fiscalEntity.ICMSReductionTarget(ipiRedTargCtrib)
	rs.IPIRedTargetNonContrib = fiscalEntity.ICMSReductionTarget(ipiRedTargNonCtrib)
	return &rs, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSReductionSubstitutions(ctx context.Context, uf string, itemID *int64, onlyActive bool) ([]*fiscalEntity.ICMSReductionSubstitution, error) {
	q := `SELECT id, uf, operation_type, item_id, item_mask, ncm_code, customer_id, is_preferential,
		  icms_pct_contrib, icms_pct_non_contrib, icms_red_pct_contrib, icms_red_target_contrib,
		  icms_red_pct_non_contrib, icms_red_target_non_contrib, icms_deferral_pct, icms_deferral_target,
		  cst_icms_contrib, cst_icms_non_contrib, csosn_icms, cst_ipi_out, cst_ipi_in,
		  fiscal_benefit_code, difal_icms_type, is_simples_optante, is_active, created_at
		  FROM icms_reduction_substitutions WHERE uf=$1`
	args := []any{uf}
	if itemID != nil {
		args = append(args, *itemID)
		q += fmt.Sprintf(" AND item_id=$%d", len(args))
	}
	if onlyActive {
		q += " AND is_active=TRUE"
	}
	q += " ORDER BY is_preferential DESC, id"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing icms reduction substitutions: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSReductionSubstitution
	for rows.Next() {
		var rs fiscalEntity.ICMSReductionSubstitution
		var opType, redTC, redTNC, defT, ipiTC, ipiTNC string
		if err := rows.Scan(&rs.ID, &rs.UF, &opType, pgutil.ScanPgInt8Ptr(&rs.ItemID),
			pgutil.ScanPgTextPtr(&rs.ItemMask), pgutil.ScanPgTextPtr(&rs.NCMCode),
			pgutil.ScanPgInt8Ptr(&rs.CustomerID), &rs.IsPreferential,
			pgutil.ScanPgNumericPtr(&rs.ICMSPctContrib), pgutil.ScanPgNumericPtr(&rs.ICMSPctNonContrib),
			pgutil.ScanPgNumericPtr(&rs.ICMSRedPctContrib), &redTC,
			pgutil.ScanPgNumericPtr(&rs.ICMSRedPctNonContrib), &redTNC,
			pgutil.ScanPgNumericPtr(&rs.ICMSDeferralPct), &defT,
			pgutil.ScanPgTextPtr(&rs.CSTICMSContrib), pgutil.ScanPgTextPtr(&rs.CSTICMSNonContrib),
			pgutil.ScanPgTextPtr(&rs.CSOSNTICMS), pgutil.ScanPgTextPtr(&rs.CSTIPIOut), pgutil.ScanPgTextPtr(&rs.CSTIPIIn),
			pgutil.ScanPgTextPtr(&rs.FiscalBenefitCode), pgutil.ScanPgTextPtr(&rs.DIFALICMSType),
			&rs.IsSimplesOptante, &rs.IsActive, &rs.CreatedAt); err != nil {
			return nil, err
		}
		rs.OperationType = fiscalEntity.ICMSOperationType(opType)
		rs.ICMSRedTargetContrib = fiscalEntity.ICMSReductionTarget(redTC)
		rs.ICMSRedTargetNonContrib = fiscalEntity.ICMSReductionTarget(redTNC)
		rs.ICMSDeferralTarget = fiscalEntity.ICMSReductionTarget(defT)
		rs.IPIRedTargetContrib = fiscalEntity.ICMSReductionTarget(ipiTC)
		rs.IPIRedTargetNonContrib = fiscalEntity.ICMSReductionTarget(ipiTNC)
		out = append(out, &rs)
	}
	return out, rows.Err()
}

func (r *FiscalParamsRepositorySQLC) FindICMSReductionSubstitution(ctx context.Context, uf string, itemID *int64, customerID *int64, opType fiscalEntity.ICMSOperationType) (*fiscalEntity.ICMSReductionSubstitution, error) {
	// Priority: preferential > item+customer > item only > classification
	q := `SELECT id, uf, operation_type, item_id, item_mask, ncm_code, customer_id, is_preferential,
		  icms_pct_contrib, icms_pct_non_contrib, icms_red_pct_contrib, icms_red_target_contrib,
		  icms_red_pct_non_contrib, icms_red_target_non_contrib, icms_deferral_pct, icms_deferral_target,
		  cst_icms_contrib, cst_icms_non_contrib, csosn_icms, cst_ipi_out, cst_ipi_in,
		  fiscal_benefit_code, difal_icms_type, is_simples_optante, is_active, created_at
		  FROM icms_reduction_substitutions
		  WHERE uf=$1 AND (operation_type=$2 OR operation_type='AMBAS') AND is_active=TRUE`
	args := []any{uf, string(opType)}
	if itemID != nil {
		args = append(args, *itemID)
		q += fmt.Sprintf(" AND (item_id=$%d OR item_id IS NULL)", len(args))
	}
	if customerID != nil {
		args = append(args, *customerID)
		q += fmt.Sprintf(" AND (customer_id=$%d OR customer_id IS NULL)", len(args))
	}
	q += " ORDER BY is_preferential DESC, (item_id IS NOT NULL) DESC, (customer_id IS NOT NULL) DESC LIMIT 1"
	var rs fiscalEntity.ICMSReductionSubstitution
	var opT, redTC, redTNC, defT, ipiTC, ipiTNC string
	err := r.pool.QueryRow(ctx, q, args...).Scan(
		&rs.ID, &rs.UF, &opT, pgutil.ScanPgInt8Ptr(&rs.ItemID),
		pgutil.ScanPgTextPtr(&rs.ItemMask), pgutil.ScanPgTextPtr(&rs.NCMCode),
		pgutil.ScanPgInt8Ptr(&rs.CustomerID), &rs.IsPreferential,
		pgutil.ScanPgNumericPtr(&rs.ICMSPctContrib), pgutil.ScanPgNumericPtr(&rs.ICMSPctNonContrib),
		pgutil.ScanPgNumericPtr(&rs.ICMSRedPctContrib), &redTC,
		pgutil.ScanPgNumericPtr(&rs.ICMSRedPctNonContrib), &redTNC,
		pgutil.ScanPgNumericPtr(&rs.ICMSDeferralPct), &defT,
		pgutil.ScanPgTextPtr(&rs.CSTICMSContrib), pgutil.ScanPgTextPtr(&rs.CSTICMSNonContrib),
		pgutil.ScanPgTextPtr(&rs.CSOSNTICMS), pgutil.ScanPgTextPtr(&rs.CSTIPIOut), pgutil.ScanPgTextPtr(&rs.CSTIPIIn),
		pgutil.ScanPgTextPtr(&rs.FiscalBenefitCode), pgutil.ScanPgTextPtr(&rs.DIFALICMSType),
		&rs.IsSimplesOptante, &rs.IsActive, &rs.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("finding icms reduction substitution: %w", err)
	}
	rs.OperationType = fiscalEntity.ICMSOperationType(opT)
	rs.ICMSRedTargetContrib = fiscalEntity.ICMSReductionTarget(redTC)
	rs.ICMSRedTargetNonContrib = fiscalEntity.ICMSReductionTarget(redTNC)
	rs.ICMSDeferralTarget = fiscalEntity.ICMSReductionTarget(defT)
	rs.IPIRedTargetContrib = fiscalEntity.ICMSReductionTarget(ipiTC)
	rs.IPIRedTargetNonContrib = fiscalEntity.ICMSReductionTarget(ipiTNC)
	return &rs, nil
}

// ─── ICMS Summary Entry Additionals ──────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) AddICMSSummaryEntryAdditional(ctx context.Context, a *fiscalEntity.ICMSSummaryEntryAdditional) (*fiscalEntity.ICMSSummaryEntryAdditional, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_summary_entry_additionals
		 (summary_entry_id, sequence, arrecadacao_indicator, processo, arrecadacao, description, dief_table, dief_code)
		 VALUES ($1, COALESCE((SELECT MAX(sequence)+1 FROM icms_summary_entry_additionals WHERE summary_entry_id=$1), 1), $2,$3,$4,$5,$6,$7)
		 RETURNING id, sequence, created_at`,
		a.SummaryEntryID, string(a.ArrecadacaoIndicator),
		pgutil.ToPgTextFromPtr(a.Processo), pgutil.ToPgTextFromPtr(a.Arrecadacao),
		pgutil.ToPgTextFromPtr(a.Description), pgutil.ToPgTextFromPtr(a.DIEFTable), pgutil.ToPgTextFromPtr(a.DIEFCode),
	).Scan(&a.ID, &a.Sequence, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding icms summary entry additional: %w", err)
	}
	return a, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSSummaryEntryAdditionals(ctx context.Context, summaryEntryID int64) ([]*fiscalEntity.ICMSSummaryEntryAdditional, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, summary_entry_id, sequence, arrecadacao_indicator, processo, arrecadacao, description, dief_table, dief_code, created_at
		 FROM icms_summary_entry_additionals WHERE summary_entry_id=$1 ORDER BY sequence`, summaryEntryID)
	if err != nil {
		return nil, fmt.Errorf("listing icms summary entry additionals: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSSummaryEntryAdditional
	for rows.Next() {
		var a fiscalEntity.ICMSSummaryEntryAdditional
		var indicator string
		if err := rows.Scan(&a.ID, &a.SummaryEntryID, &a.Sequence, &indicator,
			pgutil.ScanPgTextPtr(&a.Processo), pgutil.ScanPgTextPtr(&a.Arrecadacao),
			pgutil.ScanPgTextPtr(&a.Description), pgutil.ScanPgTextPtr(&a.DIEFTable),
			pgutil.ScanPgTextPtr(&a.DIEFCode), &a.CreatedAt); err != nil {
			return nil, err
		}
		a.ArrecadacaoIndicator = fiscalEntity.ArrecadacaoIndicator(indicator)
		out = append(out, &a)
	}
	return out, rows.Err()
}

// ─── ICMS ST Restitution ──────────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateICMSSTRestitution(ctx context.Context, rs *fiscalEntity.ICMSSTRestitution) (*fiscalEntity.ICMSSTRestitution, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO icms_st_restitutions
		 (empresa_id, period, restitution_type, uf, orig_doc_model, orig_doc_series, orig_doc_number, orig_doc_date,
		  orig_emitter_cnpj, orig_emitter_ie, item_id, item_code, cfop, motivo_code, cst_icms,
		  icms_st_base, icms_st_aliq, icms_st_value, icms_st_base_restitution, icms_st_value_restitution,
		  icms_st_consolidated_base, icms_st_consolidated_value, h030_ind_estoque, sped_block, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,TRUE)
		 RETURNING id, created_at`,
		rs.EmpresaID, rs.Period, string(rs.RestitutionType), rs.UF,
		pgutil.ToPgTextFromPtr(rs.OrigDocModel), pgutil.ToPgTextFromPtr(rs.OrigDocSeries),
		pgutil.ToPgTextFromPtr(rs.OrigDocNumber), rs.OrigDocDate,
		pgutil.ToPgTextFromPtr(rs.OrigEmitterCNPJ), pgutil.ToPgTextFromPtr(rs.OrigEmitterIE),
		pgutil.ToPgInt8Ptr(rs.ItemID), pgutil.ToPgTextFromPtr(rs.ItemCode),
		pgutil.ToPgTextFromPtr(rs.CFOP), pgutil.ToPgTextFromPtr(rs.MotivoCode), pgutil.ToPgTextFromPtr(rs.CSTICMS),
		rs.ICMSSTBase, rs.ICMSSTAliq, rs.ICMSSTValue,
		rs.ICMSSTBaseRestitution, rs.ICMSSTValueRestitution,
		rs.ICMSSTConsolidatedBase, rs.ICMSSTConsolidatedValue,
		pgutil.ToPgTextFromPtr(rs.H030IndEstoque), pgutil.ToPgTextFromPtr(rs.SpedBlock),
	).Scan(&rs.ID, &rs.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating icms st restitution: %w", err)
	}
	return rs, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateICMSSTRestitution(ctx context.Context, rs *fiscalEntity.ICMSSTRestitution) (*fiscalEntity.ICMSSTRestitution, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE icms_st_restitutions SET
		 restitution_type=$1, uf=$2, motivo_code=$3, cst_icms=$4,
		 icms_st_base=$5, icms_st_aliq=$6, icms_st_value=$7,
		 icms_st_base_restitution=$8, icms_st_value_restitution=$9,
		 icms_st_consolidated_base=$10, icms_st_consolidated_value=$11,
		 sped_block=$12, is_active=$13
		 WHERE id=$14`,
		string(rs.RestitutionType), rs.UF, pgutil.ToPgTextFromPtr(rs.MotivoCode), pgutil.ToPgTextFromPtr(rs.CSTICMS),
		rs.ICMSSTBase, rs.ICMSSTAliq, rs.ICMSSTValue,
		rs.ICMSSTBaseRestitution, rs.ICMSSTValueRestitution,
		rs.ICMSSTConsolidatedBase, rs.ICMSSTConsolidatedValue,
		pgutil.ToPgTextFromPtr(rs.SpedBlock), rs.IsActive, rs.ID)
	if err != nil {
		return nil, fmt.Errorf("updating icms st restitution %d: %w", rs.ID, err)
	}
	return rs, nil
}

func (r *FiscalParamsRepositorySQLC) GetICMSSTRestitution(ctx context.Context, id int64) (*fiscalEntity.ICMSSTRestitution, error) {
	var rs fiscalEntity.ICMSSTRestitution
	var restType string
	err := r.pool.QueryRow(ctx,
		`SELECT id, empresa_id, period, restitution_type, uf, orig_doc_model, orig_doc_series, orig_doc_number,
		 orig_doc_date, orig_emitter_cnpj, orig_emitter_ie, item_id, item_code, cfop, motivo_code, cst_icms,
		 icms_st_base, icms_st_aliq, icms_st_value, icms_st_base_restitution, icms_st_value_restitution,
		 icms_st_consolidated_base, icms_st_consolidated_value, h030_ind_estoque, sped_block, is_active, created_at
		 FROM icms_st_restitutions WHERE id=$1`, id).
		Scan(&rs.ID, &rs.EmpresaID, &rs.Period, &restType, &rs.UF,
			pgutil.ScanPgTextPtr(&rs.OrigDocModel), pgutil.ScanPgTextPtr(&rs.OrigDocSeries),
			pgutil.ScanPgTextPtr(&rs.OrigDocNumber), &rs.OrigDocDate,
			pgutil.ScanPgTextPtr(&rs.OrigEmitterCNPJ), pgutil.ScanPgTextPtr(&rs.OrigEmitterIE),
			pgutil.ScanPgInt8Ptr(&rs.ItemID), pgutil.ScanPgTextPtr(&rs.ItemCode),
			pgutil.ScanPgTextPtr(&rs.CFOP), pgutil.ScanPgTextPtr(&rs.MotivoCode), pgutil.ScanPgTextPtr(&rs.CSTICMS),
			&rs.ICMSSTBase, &rs.ICMSSTAliq, &rs.ICMSSTValue,
			&rs.ICMSSTBaseRestitution, &rs.ICMSSTValueRestitution,
			&rs.ICMSSTConsolidatedBase, &rs.ICMSSTConsolidatedValue,
			pgutil.ScanPgTextPtr(&rs.H030IndEstoque), pgutil.ScanPgTextPtr(&rs.SpedBlock),
			&rs.IsActive, &rs.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("icms st restitution %d not found", id)
		}
		return nil, fmt.Errorf("getting icms st restitution: %w", err)
	}
	rs.RestitutionType = fiscalEntity.ICMSSTRestitutionType(restType)
	return &rs, nil
}

func (r *FiscalParamsRepositorySQLC) ListICMSSTRestitutions(ctx context.Context, empresaID int, period string, uf string) ([]*fiscalEntity.ICMSSTRestitution, error) {
	q := `SELECT id, empresa_id, period, restitution_type, uf, orig_doc_number, item_code, cfop, motivo_code, cst_icms,
		  icms_st_base, icms_st_aliq, icms_st_value, icms_st_base_restitution, icms_st_value_restitution,
		  icms_st_consolidated_base, icms_st_consolidated_value, sped_block, is_active, created_at
		  FROM icms_st_restitutions WHERE empresa_id=$1 AND is_active=TRUE`
	args := []any{empresaID}
	if period != "" {
		args = append(args, period)
		q += fmt.Sprintf(" AND period=$%d", len(args))
	}
	if uf != "" {
		args = append(args, uf)
		q += fmt.Sprintf(" AND uf=$%d", len(args))
	}
	q += " ORDER BY period DESC, id"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing icms st restitutions: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.ICMSSTRestitution
	for rows.Next() {
		var rs fiscalEntity.ICMSSTRestitution
		var restType string
		if err := rows.Scan(&rs.ID, &rs.EmpresaID, &rs.Period, &restType, &rs.UF,
			pgutil.ScanPgTextPtr(&rs.OrigDocNumber), pgutil.ScanPgTextPtr(&rs.ItemCode),
			pgutil.ScanPgTextPtr(&rs.CFOP), pgutil.ScanPgTextPtr(&rs.MotivoCode), pgutil.ScanPgTextPtr(&rs.CSTICMS),
			&rs.ICMSSTBase, &rs.ICMSSTAliq, &rs.ICMSSTValue,
			&rs.ICMSSTBaseRestitution, &rs.ICMSSTValueRestitution,
			&rs.ICMSSTConsolidatedBase, &rs.ICMSSTConsolidatedValue,
			pgutil.ScanPgTextPtr(&rs.SpedBlock), &rs.IsActive, &rs.CreatedAt); err != nil {
			return nil, err
		}
		rs.RestitutionType = fiscalEntity.ICMSSTRestitutionType(restType)
		out = append(out, &rs)
	}
	return out, rows.Err()
}

// ─── Special Adjustment Notes ─────────────────────────────────────────────────

func (r *FiscalParamsRepositorySQLC) CreateSpecialAdjustmentNote(ctx context.Context, n *fiscalEntity.SpecialAdjustmentNote) (*fiscalEntity.SpecialAdjustmentNote, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO special_adjustment_notes
		 (empresa_id, purpose, status, number, series, issue_date, period, invoice_type_id, cfop_id,
		  icms_apuracao_line_id, adjustment_code_id, adjustment_doc_code_id, history, auto_generate_summary,
		  total_value, total_icms, total_ipi, observation)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		 RETURNING id, created_at`,
		n.EmpresaID, string(n.Purpose), string(n.Status),
		pgutil.ToPgTextFromPtr(n.Number), pgutil.ToPgTextFromPtr(n.Series),
		n.IssueDate, n.Period, pgutil.ToPgInt8Ptr(n.InvoiceTypeID),
		pgutil.ToPgInt8Ptr(n.CFOPID), pgutil.ToPgInt8Ptr(n.ICMSApuracaoLineID),
		pgutil.ToPgInt8Ptr(n.AdjustmentCodeID), pgutil.ToPgInt8Ptr(n.AdjustmentDocCodeID),
		pgutil.ToPgTextFromPtr(n.History), n.AutoGenerateSummary,
		n.TotalValue, n.TotalICMS, n.TotalIPI, pgutil.ToPgTextFromPtr(n.Observation),
	).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating special adjustment note: %w", err)
	}
	return n, nil
}

func (r *FiscalParamsRepositorySQLC) UpdateSpecialAdjustmentNote(ctx context.Context, n *fiscalEntity.SpecialAdjustmentNote) (*fiscalEntity.SpecialAdjustmentNote, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE special_adjustment_notes SET
		 status=$1, number=$2, series=$3, history=$4, total_value=$5, total_icms=$6, total_ipi=$7,
		 generated_summary_entry_id=$8, observation=$9
		 WHERE id=$10`,
		string(n.Status), pgutil.ToPgTextFromPtr(n.Number), pgutil.ToPgTextFromPtr(n.Series),
		pgutil.ToPgTextFromPtr(n.History), n.TotalValue, n.TotalICMS, n.TotalIPI,
		pgutil.ToPgInt8Ptr(n.GeneratedSummaryEntryID), pgutil.ToPgTextFromPtr(n.Observation), n.ID)
	if err != nil {
		return nil, fmt.Errorf("updating special adjustment note %d: %w", n.ID, err)
	}
	return n, nil
}

func (r *FiscalParamsRepositorySQLC) GetSpecialAdjustmentNote(ctx context.Context, id int64) (*fiscalEntity.SpecialAdjustmentNote, error) {
	var n fiscalEntity.SpecialAdjustmentNote
	var purpose, status string
	err := r.pool.QueryRow(ctx,
		`SELECT id, empresa_id, purpose, status, number, series, issue_date, period, invoice_type_id, cfop_id,
		 icms_apuracao_line_id, adjustment_code_id, adjustment_doc_code_id, history, auto_generate_summary,
		 generated_summary_entry_id, total_value, total_icms, total_ipi, observation, created_at
		 FROM special_adjustment_notes WHERE id=$1`, id).
		Scan(&n.ID, &n.EmpresaID, &purpose, &status,
			pgutil.ScanPgTextPtr(&n.Number), pgutil.ScanPgTextPtr(&n.Series),
			&n.IssueDate, &n.Period,
			pgutil.ScanPgInt8Ptr(&n.InvoiceTypeID), pgutil.ScanPgInt8Ptr(&n.CFOPID),
			pgutil.ScanPgInt8Ptr(&n.ICMSApuracaoLineID), pgutil.ScanPgInt8Ptr(&n.AdjustmentCodeID),
			pgutil.ScanPgInt8Ptr(&n.AdjustmentDocCodeID), pgutil.ScanPgTextPtr(&n.History),
			&n.AutoGenerateSummary, pgutil.ScanPgInt8Ptr(&n.GeneratedSummaryEntryID),
			&n.TotalValue, &n.TotalICMS, &n.TotalIPI, pgutil.ScanPgTextPtr(&n.Observation), &n.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("special adjustment note %d not found", id)
		}
		return nil, fmt.Errorf("getting special adjustment note: %w", err)
	}
	n.Purpose = fiscalEntity.SpecialNotePurpose(purpose)
	n.Status = fiscalEntity.SpecialNoteStatus(status)
	return &n, nil
}

func (r *FiscalParamsRepositorySQLC) ListSpecialAdjustmentNotes(ctx context.Context, empresaID int, period string) ([]*fiscalEntity.SpecialAdjustmentNote, error) {
	q := `SELECT id, empresa_id, purpose, status, number, series, issue_date, period,
		  total_value, total_icms, total_ipi, observation, created_at
		  FROM special_adjustment_notes WHERE empresa_id=$1`
	args := []any{empresaID}
	if period != "" {
		args = append(args, period)
		q += fmt.Sprintf(" AND period=$%d", len(args))
	}
	q += " ORDER BY issue_date DESC, id"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing special adjustment notes: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.SpecialAdjustmentNote
	for rows.Next() {
		var n fiscalEntity.SpecialAdjustmentNote
		var purpose, status string
		if err := rows.Scan(&n.ID, &n.EmpresaID, &purpose, &status,
			pgutil.ScanPgTextPtr(&n.Number), pgutil.ScanPgTextPtr(&n.Series),
			&n.IssueDate, &n.Period, &n.TotalValue, &n.TotalICMS, &n.TotalIPI,
			pgutil.ScanPgTextPtr(&n.Observation), &n.CreatedAt); err != nil {
			return nil, err
		}
		n.Purpose = fiscalEntity.SpecialNotePurpose(purpose)
		n.Status = fiscalEntity.SpecialNoteStatus(status)
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *FiscalParamsRepositorySQLC) AddSpecialAdjustmentNoteItem(ctx context.Context, item *fiscalEntity.SpecialAdjustmentNoteItem) (*fiscalEntity.SpecialAdjustmentNoteItem, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO special_adjustment_note_items
		 (note_id, sequence, item_id, item_code, description, quantity, unit, unit_value, total_value,
		  icms_base, icms_pct, icms_deferral_pct, icms_value, icms_deferred_value,
		  ipi_base, ipi_pct, ipi_value, cst_icms, cst_ipi, cfop_id)
		 VALUES ($1, COALESCE((SELECT MAX(sequence)+1 FROM special_adjustment_note_items WHERE note_id=$1), 1),
		 $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		 RETURNING id, sequence, created_at`,
		item.NoteID, pgutil.ToPgInt8Ptr(item.ItemID), pgutil.ToPgTextFromPtr(item.ItemCode),
		pgutil.ToPgTextFromPtr(item.Description), item.Quantity, pgutil.ToPgTextFromPtr(item.Unit),
		item.UnitValue, item.TotalValue, item.ICMSBase, item.ICMSPct, item.ICMSDeferralPct,
		item.ICMSValue, item.ICMSDeferredValue, item.IPIBase, item.IPIPct, item.IPIValue,
		pgutil.ToPgTextFromPtr(item.CSTICMS), pgutil.ToPgTextFromPtr(item.CSTIPI),
		pgutil.ToPgInt8Ptr(item.CFOPID),
	).Scan(&item.ID, &item.Sequence, &item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding special adjustment note item: %w", err)
	}
	return item, nil
}

func (r *FiscalParamsRepositorySQLC) ListSpecialAdjustmentNoteItems(ctx context.Context, noteID int64) ([]*fiscalEntity.SpecialAdjustmentNoteItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, note_id, sequence, item_id, item_code, description, quantity, unit, unit_value, total_value,
		 icms_base, icms_pct, icms_deferral_pct, icms_value, icms_deferred_value,
		 ipi_base, ipi_pct, ipi_value, cst_icms, cst_ipi, cfop_id, created_at
		 FROM special_adjustment_note_items WHERE note_id=$1 ORDER BY sequence`, noteID)
	if err != nil {
		return nil, fmt.Errorf("listing special adjustment note items: %w", err)
	}
	defer rows.Close()
	var out []*fiscalEntity.SpecialAdjustmentNoteItem
	for rows.Next() {
		var item fiscalEntity.SpecialAdjustmentNoteItem
		if err := rows.Scan(&item.ID, &item.NoteID, &item.Sequence,
			pgutil.ScanPgInt8Ptr(&item.ItemID), pgutil.ScanPgTextPtr(&item.ItemCode),
			pgutil.ScanPgTextPtr(&item.Description), &item.Quantity,
			pgutil.ScanPgTextPtr(&item.Unit), &item.UnitValue, &item.TotalValue,
			&item.ICMSBase, &item.ICMSPct, &item.ICMSDeferralPct, &item.ICMSValue, &item.ICMSDeferredValue,
			&item.IPIBase, &item.IPIPct, &item.IPIValue,
			pgutil.ScanPgTextPtr(&item.CSTICMS), pgutil.ScanPgTextPtr(&item.CSTIPI),
			pgutil.ScanPgInt8Ptr(&item.CFOPID), &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}
