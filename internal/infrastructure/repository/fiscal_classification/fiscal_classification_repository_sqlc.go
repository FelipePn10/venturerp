package fiscal_classification

import (
	"context"
	"fmt"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FiscalClassificationRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.FiscalClassificationRepository {
	return &FiscalClassificationRepositorySQLC{q: q, pool: pool}
}

func (r *FiscalClassificationRepositorySQLC) Create(ctx context.Context, c *entity.FiscalClassification) (*entity.FiscalClassification, error) {
	row, err := r.q.CreateFiscalClassification(ctx, sqlc.CreateFiscalClassificationParams{
		Code:                      c.Code,
		Description:               c.Description,
		Ncm:                       pgutil.ToPgTextFromPtr(c.NCM),
		Cest:                      pgutil.ToPgTextFromPtr(c.CEST),
		IpiRate:                   pgutil.ToPgNumericFromFloat64(c.IPIRate),
		IpiIndicator:              string(c.IPIIndicator),
		Apuracao:                  pgutil.ToPgTextFromPtr(c.Apuracao),
		CstIpiEntrada:             pgutil.ToPgTextFromPtr(c.CSTIPIEntrada),
		CstIpiSaida:               pgutil.ToPgTextFromPtr(c.CSTIPISaida),
		PisRate:                   pgutil.ToPgNumericFromFloat64(c.PISRate),
		PisIndicator:              string(c.PISIndicator),
		CstPisEntrada:             pgutil.ToPgTextFromPtr(c.CSTPISEntrada),
		CstPisSaida:               pgutil.ToPgTextFromPtr(c.CSTPISSaida),
		CofinsRate:                pgutil.ToPgNumericFromFloat64(c.COFINSRate),
		CofinsIndicator:           string(c.COFINSIndicator),
		CstCofinsEntrada:          pgutil.ToPgTextFromPtr(c.CSTCOFINSEntrada),
		CstCofinsSaida:            pgutil.ToPgTextFromPtr(c.CSTCOFINSSaida),
		CofinsMajoradoPct:         pgutil.ToPgNumericFromFloat64(c.COFINSMajoradoPct),
		PisStPct:                  pgutil.ToPgNumericFromFloat64(c.PISSTPct),
		CofinsStPct:               pgutil.ToPgNumericFromFloat64(c.COFINSSTPct),
		PisConsumoPct:             pgutil.ToPgNumericFromFloat64(c.PISConsumoPct),
		CstPisConsumoEntrada:      pgutil.ToPgTextFromPtr(c.CSTPISConsumoEntrada),
		CstPisConsumoSaida:        pgutil.ToPgTextFromPtr(c.CSTPISConsumoSaida),
		CofinsConsumoPct:          pgutil.ToPgNumericFromFloat64(c.COFINSConsumoPct),
		CstCofinsConsumoEntrada:   pgutil.ToPgTextFromPtr(c.CSTCOFINSConsumoEntrada),
		CstCofinsConsumoSaida:     pgutil.ToPgTextFromPtr(c.CSTCOFINSConsumoSaida),
		PisRetencaoPct:            pgutil.ToPgNumericFromFloat64(c.PISRetencaoPct),
		CstPisRetencao:            pgutil.ToPgTextFromPtr(c.CSTPISRetencao),
		CofinsRetencaoPct:         pgutil.ToPgNumericFromFloat64(c.COFINSRetencaoPct),
		CstCofinsRetencao:         pgutil.ToPgTextFromPtr(c.CSTCOFINSRetencao),
		PisReducaoPct:             pgutil.ToPgNumericFromFloat64(c.PISReducaoPct),
		CstPisReducao:             pgutil.ToPgTextFromPtr(c.CSTPISReducao),
		CofinsReducaoPct:          pgutil.ToPgNumericFromFloat64(c.COFINSReducaoPct),
		CstCofinsReducao:          pgutil.ToPgTextFromPtr(c.CSTCOFINSReducao),
		DescPisZfPct:              pgutil.ToPgNumericFromFloat64(c.DescPISZFPct),
		DescCofinsZfPct:           pgutil.ToPgNumericFromFloat64(c.DescCOFINSZFPct),
		ExTarifario:               pgutil.ToPgTextFromPtr(c.ExTarifario),
		UnIpi:                     pgutil.ToPgTextFromPtr(c.UNIPI),
		UnTributacao:              pgutil.ToPgTextFromPtr(c.UNTributacao),
		ModBcIcms:                 pgutil.ToPgTextFromPtr(c.ModBCICMS),
		ModBcIcmsSt:               pgutil.ToPgTextFromPtr(c.ModBCICMSST),
		CodClasTrib:               pgutil.ToPgTextFromPtr(c.CodClasTrib),
		CodClasTribTribReg:        pgutil.ToPgTextFromPtr(c.CodClasTribTribReg),
		ObsFiscal:                 pgutil.ToPgTextFromPtr(c.ObsFiscal),
		ItemCode:                  c.ItemCode,
		CreatedBy:                 pgutil.ToPgUUID(c.CreatedBy),
		EnterpriseID:              c.EnterpriseID,
		ValidFrom:                 pgutil.ToPgDateFromPtr(c.ValidFrom),
		ValidUntil:                pgutil.ToPgDateFromPtr(c.ValidUntil),
		DefaultOrigin:             toPgOrigin(c.DefaultOrigin),
		DefaultIcmsRate:           pgutil.ToPgNumericFromFloat64(c.DefaultICMSRate),
		DefaultCalculatePisCofins: pgutil.ToPgBool(c.DefaultCalculatePISCOFINS),
	})
	if err != nil {
		return nil, fmt.Errorf("creating fiscal classification: %w", err)
	}
	return classificationToEntity(row), nil
}

func (r *FiscalClassificationRepositorySQLC) Update(ctx context.Context, c *entity.FiscalClassification) (*entity.FiscalClassification, error) {
	row, err := r.q.UpdateFiscalClassification(ctx, sqlc.UpdateFiscalClassificationParams{
		Code:                      c.Code,
		Description:               c.Description,
		Ncm:                       pgutil.ToPgTextFromPtr(c.NCM),
		Cest:                      pgutil.ToPgTextFromPtr(c.CEST),
		IpiRate:                   pgutil.ToPgNumericFromFloat64(c.IPIRate),
		IpiIndicator:              string(c.IPIIndicator),
		Apuracao:                  pgutil.ToPgTextFromPtr(c.Apuracao),
		CstIpiEntrada:             pgutil.ToPgTextFromPtr(c.CSTIPIEntrada),
		CstIpiSaida:               pgutil.ToPgTextFromPtr(c.CSTIPISaida),
		PisRate:                   pgutil.ToPgNumericFromFloat64(c.PISRate),
		PisIndicator:              string(c.PISIndicator),
		CstPisEntrada:             pgutil.ToPgTextFromPtr(c.CSTPISEntrada),
		CstPisSaida:               pgutil.ToPgTextFromPtr(c.CSTPISSaida),
		CofinsRate:                pgutil.ToPgNumericFromFloat64(c.COFINSRate),
		CofinsIndicator:           string(c.COFINSIndicator),
		CstCofinsEntrada:          pgutil.ToPgTextFromPtr(c.CSTCOFINSEntrada),
		CstCofinsSaida:            pgutil.ToPgTextFromPtr(c.CSTCOFINSSaida),
		CofinsMajoradoPct:         pgutil.ToPgNumericFromFloat64(c.COFINSMajoradoPct),
		PisStPct:                  pgutil.ToPgNumericFromFloat64(c.PISSTPct),
		CofinsStPct:               pgutil.ToPgNumericFromFloat64(c.COFINSSTPct),
		PisConsumoPct:             pgutil.ToPgNumericFromFloat64(c.PISConsumoPct),
		CstPisConsumoEntrada:      pgutil.ToPgTextFromPtr(c.CSTPISConsumoEntrada),
		CstPisConsumoSaida:        pgutil.ToPgTextFromPtr(c.CSTPISConsumoSaida),
		CofinsConsumoPct:          pgutil.ToPgNumericFromFloat64(c.COFINSConsumoPct),
		CstCofinsConsumoEntrada:   pgutil.ToPgTextFromPtr(c.CSTCOFINSConsumoEntrada),
		CstCofinsConsumoSaida:     pgutil.ToPgTextFromPtr(c.CSTCOFINSConsumoSaida),
		PisRetencaoPct:            pgutil.ToPgNumericFromFloat64(c.PISRetencaoPct),
		CstPisRetencao:            pgutil.ToPgTextFromPtr(c.CSTPISRetencao),
		CofinsRetencaoPct:         pgutil.ToPgNumericFromFloat64(c.COFINSRetencaoPct),
		CstCofinsRetencao:         pgutil.ToPgTextFromPtr(c.CSTCOFINSRetencao),
		PisReducaoPct:             pgutil.ToPgNumericFromFloat64(c.PISReducaoPct),
		CstPisReducao:             pgutil.ToPgTextFromPtr(c.CSTPISReducao),
		CofinsReducaoPct:          pgutil.ToPgNumericFromFloat64(c.COFINSReducaoPct),
		CstCofinsReducao:          pgutil.ToPgTextFromPtr(c.CSTCOFINSReducao),
		DescPisZfPct:              pgutil.ToPgNumericFromFloat64(c.DescPISZFPct),
		DescCofinsZfPct:           pgutil.ToPgNumericFromFloat64(c.DescCOFINSZFPct),
		ExTarifario:               pgutil.ToPgTextFromPtr(c.ExTarifario),
		UnIpi:                     pgutil.ToPgTextFromPtr(c.UNIPI),
		UnTributacao:              pgutil.ToPgTextFromPtr(c.UNTributacao),
		ModBcIcms:                 pgutil.ToPgTextFromPtr(c.ModBCICMS),
		ModBcIcmsSt:               pgutil.ToPgTextFromPtr(c.ModBCICMSST),
		CodClasTrib:               pgutil.ToPgTextFromPtr(c.CodClasTrib),
		CodClasTribTribReg:        pgutil.ToPgTextFromPtr(c.CodClasTribTribReg),
		ObsFiscal:                 pgutil.ToPgTextFromPtr(c.ObsFiscal),
		ItemCode:                  c.ItemCode,
		IsActive:                  c.IsActive,
		EnterpriseID:              c.EnterpriseID,
		ValidFrom:                 pgutil.ToPgDateFromPtr(c.ValidFrom),
		ValidUntil:                pgutil.ToPgDateFromPtr(c.ValidUntil),
		DefaultOrigin:             toPgOrigin(c.DefaultOrigin),
		DefaultIcmsRate:           pgutil.ToPgNumericFromFloat64(c.DefaultICMSRate),
		DefaultCalculatePisCofins: pgutil.ToPgBool(c.DefaultCalculatePISCOFINS),
	})
	if err != nil {
		return nil, fmt.Errorf("updating fiscal classification: %w", err)
	}
	return classificationToEntity(row), nil
}

func (r *FiscalClassificationRepositorySQLC) GetByCode(ctx context.Context, enterpriseID, code int64) (*entity.FiscalClassification, error) {
	row, err := r.q.GetFiscalClassificationByCode(ctx, sqlc.GetFiscalClassificationByCodeParams{EnterpriseID: enterpriseID, Code: code})
	if err != nil {
		return nil, fmt.Errorf("fiscal classification %d not found: %w", code, err)
	}
	return classificationToEntity(row), nil
}

func (r *FiscalClassificationRepositorySQLC) List(ctx context.Context, enterpriseID int64, onlyActive bool) ([]*entity.FiscalClassification, error) {
	rows, err := r.q.ListFiscalClassifications(ctx, sqlc.ListFiscalClassificationsParams{EnterpriseID: enterpriseID, Column2: onlyActive})
	if err != nil {
		return nil, err
	}
	out := make([]*entity.FiscalClassification, 0, len(rows))
	for _, row := range rows {
		out = append(out, classificationToEntity(row))
	}
	return out, nil
}

func (r *FiscalClassificationRepositorySQLC) NextCode(ctx context.Context, enterpriseID int64) (int64, error) {
	v, err := r.q.NextFiscalClassificationCode(ctx, enterpriseID)
	return int64(v), err
}

func (r *FiscalClassificationRepositorySQLC) AddLanguage(ctx context.Context, l *entity.FiscalClassificationLanguage) (*entity.FiscalClassificationLanguage, error) {
	row, err := r.q.CreateFiscalClassificationLanguage(ctx, sqlc.CreateFiscalClassificationLanguageParams{
		ClassificationID: l.ClassificationID,
		Language:         l.Language,
		Description:      l.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("adding language: %w", err)
	}
	return &entity.FiscalClassificationLanguage{ID: row.ID, ClassificationID: row.ClassificationID, Language: row.Language, Description: row.Description}, nil
}

func (r *FiscalClassificationRepositorySQLC) ListLanguages(ctx context.Context, classificationID int64) ([]*entity.FiscalClassificationLanguage, error) {
	rows, err := r.q.ListFiscalClassificationLanguages(ctx, classificationID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.FiscalClassificationLanguage, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.FiscalClassificationLanguage{ID: row.ID, ClassificationID: row.ClassificationID, Language: row.Language, Description: row.Description})
	}
	return out, nil
}

func (r *FiscalClassificationRepositorySQLC) DeleteLanguage(ctx context.Context, id int64) error {
	return r.q.DeleteFiscalClassificationLanguage(ctx, id)
}

func (r *FiscalClassificationRepositorySQLC) AddExportAttribute(ctx context.Context, a *entity.FiscalClassificationExportAttribute) (*entity.FiscalClassificationExportAttribute, error) {
	row, err := r.q.CreateFiscalClassificationExportAttribute(ctx, sqlc.CreateFiscalClassificationExportAttributeParams{
		ClassificationID: a.ClassificationID,
		Code:             a.Code,
		Description:      pgutil.ToPgTextFromPtr(a.Description),
		Domain:           pgutil.ToPgTextFromPtr(a.Domain),
		StartDate:        pgutil.ToPgDateFromPtr(a.StartDate),
		EndDate:          pgutil.ToPgDateFromPtr(a.EndDate),
	})
	if err != nil {
		return nil, fmt.Errorf("adding export attribute: %w", err)
	}
	return exportAttributeToEntity(row), nil
}

func (r *FiscalClassificationRepositorySQLC) ListExportAttributes(ctx context.Context, classificationID int64) ([]*entity.FiscalClassificationExportAttribute, error) {
	rows, err := r.q.ListFiscalClassificationExportAttributes(ctx, classificationID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.FiscalClassificationExportAttribute, 0, len(rows))
	for _, row := range rows {
		out = append(out, exportAttributeToEntity(row))
	}
	return out, nil
}

func (r *FiscalClassificationRepositorySQLC) DeleteExportAttribute(ctx context.Context, id int64) error {
	return r.q.DeleteFiscalClassificationExportAttribute(ctx, id)
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func classificationToEntity(row sqlc.FiscalClassification) *entity.FiscalClassification {
	return &entity.FiscalClassification{
		ID:                        row.ID,
		EnterpriseID:              row.EnterpriseID,
		Code:                      row.Code,
		Description:               row.Description,
		ItemCode:                  row.ItemCode,
		NCM:                       pgutil.FromPgTextPtr(row.Ncm),
		CEST:                      pgutil.FromPgTextPtr(row.Cest),
		IPIRate:                   pgutil.FromPgNumericToFloat64(row.IpiRate),
		IPIIndicator:              entity.RateIndicator(row.IpiIndicator),
		Apuracao:                  pgutil.FromPgTextPtr(row.Apuracao),
		CSTIPIEntrada:             pgutil.FromPgTextPtr(row.CstIpiEntrada),
		CSTIPISaida:               pgutil.FromPgTextPtr(row.CstIpiSaida),
		PISRate:                   pgutil.FromPgNumericToFloat64(row.PisRate),
		PISIndicator:              entity.RateIndicator(row.PisIndicator),
		CSTPISEntrada:             pgutil.FromPgTextPtr(row.CstPisEntrada),
		CSTPISSaida:               pgutil.FromPgTextPtr(row.CstPisSaida),
		COFINSRate:                pgutil.FromPgNumericToFloat64(row.CofinsRate),
		COFINSIndicator:           entity.RateIndicator(row.CofinsIndicator),
		CSTCOFINSEntrada:          pgutil.FromPgTextPtr(row.CstCofinsEntrada),
		CSTCOFINSSaida:            pgutil.FromPgTextPtr(row.CstCofinsSaida),
		COFINSMajoradoPct:         pgutil.FromPgNumericToFloat64(row.CofinsMajoradoPct),
		PISSTPct:                  pgutil.FromPgNumericToFloat64(row.PisStPct),
		COFINSSTPct:               pgutil.FromPgNumericToFloat64(row.CofinsStPct),
		PISConsumoPct:             pgutil.FromPgNumericToFloat64(row.PisConsumoPct),
		CSTPISConsumoEntrada:      pgutil.FromPgTextPtr(row.CstPisConsumoEntrada),
		CSTPISConsumoSaida:        pgutil.FromPgTextPtr(row.CstPisConsumoSaida),
		COFINSConsumoPct:          pgutil.FromPgNumericToFloat64(row.CofinsConsumoPct),
		CSTCOFINSConsumoEntrada:   pgutil.FromPgTextPtr(row.CstCofinsConsumoEntrada),
		CSTCOFINSConsumoSaida:     pgutil.FromPgTextPtr(row.CstCofinsConsumoSaida),
		PISRetencaoPct:            pgutil.FromPgNumericToFloat64(row.PisRetencaoPct),
		CSTPISRetencao:            pgutil.FromPgTextPtr(row.CstPisRetencao),
		COFINSRetencaoPct:         pgutil.FromPgNumericToFloat64(row.CofinsRetencaoPct),
		CSTCOFINSRetencao:         pgutil.FromPgTextPtr(row.CstCofinsRetencao),
		PISReducaoPct:             pgutil.FromPgNumericToFloat64(row.PisReducaoPct),
		CSTPISReducao:             pgutil.FromPgTextPtr(row.CstPisReducao),
		COFINSReducaoPct:          pgutil.FromPgNumericToFloat64(row.CofinsReducaoPct),
		CSTCOFINSReducao:          pgutil.FromPgTextPtr(row.CstCofinsReducao),
		DescPISZFPct:              pgutil.FromPgNumericToFloat64(row.DescPisZfPct),
		DescCOFINSZFPct:           pgutil.FromPgNumericToFloat64(row.DescCofinsZfPct),
		ExTarifario:               pgutil.FromPgTextPtr(row.ExTarifario),
		UNIPI:                     pgutil.FromPgTextPtr(row.UnIpi),
		UNTributacao:              pgutil.FromPgTextPtr(row.UnTributacao),
		ModBCICMS:                 pgutil.FromPgTextPtr(row.ModBcIcms),
		ModBCICMSST:               pgutil.FromPgTextPtr(row.ModBcIcmsSt),
		CodClasTrib:               pgutil.FromPgTextPtr(row.CodClasTrib),
		CodClasTribTribReg:        pgutil.FromPgTextPtr(row.CodClasTribTribReg),
		ObsFiscal:                 pgutil.FromPgTextPtr(row.ObsFiscal),
		ValidFrom:                 pgutil.FromPgDateToPtr(row.ValidFrom),
		ValidUntil:                pgutil.FromPgDateToPtr(row.ValidUntil),
		DefaultOrigin:             fromPgOrigin(row.DefaultOrigin),
		DefaultICMSRate:           pgutil.FromPgNumericToFloat64(row.DefaultIcmsRate),
		DefaultCalculatePISCOFINS: row.DefaultCalculatePisCofins.Valid && row.DefaultCalculatePisCofins.Bool,
		IsActive:                  row.IsActive,
		CreatedAt:                 pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:                 pgutil.FromPgUUID(row.CreatedBy),
		UpdatedAt:                 pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func toPgOrigin(value *string) pgtype.Int2 {
	if value == nil {
		return pgtype.Int2{}
	}
	n, err := strconv.ParseInt(*value, 10, 16)
	if err != nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: int16(n), Valid: true}
}

func fromPgOrigin(value pgtype.Int2) *string {
	if !value.Valid {
		return nil
	}
	v := strconv.FormatInt(int64(value.Int16), 10)
	return &v
}

func exportAttributeToEntity(row sqlc.FiscalClassificationExportAttribute) *entity.FiscalClassificationExportAttribute {
	return &entity.FiscalClassificationExportAttribute{
		ID:               row.ID,
		ClassificationID: row.ClassificationID,
		Code:             row.Code,
		Description:      pgutil.FromPgTextPtr(row.Description),
		Domain:           pgutil.FromPgTextPtr(row.Domain),
		StartDate:        pgutil.FromPgDateToPtr(row.StartDate),
		EndDate:          pgutil.FromPgDateToPtr(row.EndDate),
	}
}
