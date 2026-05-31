package fiscal_classification_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/repository"
)

type FiscalClassificationUseCase struct {
	repo repository.FiscalClassificationRepository
}

func NewFiscalClassificationUseCase(repo repository.FiscalClassificationRepository) *FiscalClassificationUseCase {
	return &FiscalClassificationUseCase{repo: repo}
}

func applyFields(c *entity.FiscalClassification, f request.FiscalClassificationFields) {
	c.Description = f.Description
	c.NCM = f.NCM
	c.CEST = f.CEST
	c.IPIRate = f.IPIRate
	c.Apuracao = f.Apuracao
	c.CSTIPIEntrada = f.CSTIPIEntrada
	c.CSTIPISaida = f.CSTIPISaida
	c.PISRate = f.PISRate
	c.CSTPISEntrada = f.CSTPISEntrada
	c.CSTPISSaida = f.CSTPISSaida
	c.COFINSRate = f.COFINSRate
	c.CSTCOFINSEntrada = f.CSTCOFINSEntrada
	c.CSTCOFINSSaida = f.CSTCOFINSSaida
	c.COFINSMajoradoPct = f.COFINSMajoradoPct
	c.PISSTPct = f.PISSTPct
	c.COFINSSTPct = f.COFINSSTPct
	c.PISConsumoPct = f.PISConsumoPct
	c.CSTPISConsumoEntrada = f.CSTPISConsumoEntrada
	c.CSTPISConsumoSaida = f.CSTPISConsumoSaida
	c.COFINSConsumoPct = f.COFINSConsumoPct
	c.CSTCOFINSConsumoEntrada = f.CSTCOFINSConsumoEntrada
	c.CSTCOFINSConsumoSaida = f.CSTCOFINSConsumoSaida
	c.PISRetencaoPct = f.PISRetencaoPct
	c.CSTPISRetencao = f.CSTPISRetencao
	c.COFINSRetencaoPct = f.COFINSRetencaoPct
	c.CSTCOFINSRetencao = f.CSTCOFINSRetencao
	c.PISReducaoPct = f.PISReducaoPct
	c.CSTPISReducao = f.CSTPISReducao
	c.COFINSReducaoPct = f.COFINSReducaoPct
	c.CSTCOFINSReducao = f.CSTCOFINSReducao
	c.DescPISZFPct = f.DescPISZFPct
	c.DescCOFINSZFPct = f.DescCOFINSZFPct
	c.ExTarifario = f.ExTarifario
	c.UNIPI = f.UNIPI
	c.UNTributacao = f.UNTributacao
	c.ModBCICMS = f.ModBCICMS
	c.ModBCICMSST = f.ModBCICMSST
	c.CodClasTrib = f.CodClasTrib
	c.CodClasTribTribReg = f.CodClasTribTribReg
	c.ObsFiscal = f.ObsFiscal
	c.IPIIndicator = indicatorOrDefault(f.IPIIndicator)
	c.PISIndicator = indicatorOrDefault(f.PISIndicator)
	c.COFINSIndicator = indicatorOrDefault(f.COFINSIndicator)
}

func indicatorOrDefault(v string) entity.RateIndicator {
	if v == string(entity.IndicatorValor) {
		return entity.IndicatorValor
	}
	return entity.IndicatorPercentual
}

func (uc *FiscalClassificationUseCase) Create(ctx context.Context, dto request.CreateFiscalClassificationDTO) (*entity.FiscalClassification, error) {
	code, err := uc.repo.NextCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	c, err := entity.NewFiscalClassification(code, dto.Description, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	applyFields(c, dto.FiscalClassificationFields)
	return uc.repo.Create(ctx, c)
}

func (uc *FiscalClassificationUseCase) Update(ctx context.Context, dto request.UpdateFiscalClassificationDTO) (*entity.FiscalClassification, error) {
	c, err := uc.repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	applyFields(c, dto.FiscalClassificationFields)
	c.IsActive = dto.IsActive
	return uc.repo.Update(ctx, c)
}

func (uc *FiscalClassificationUseCase) Get(ctx context.Context, code int64) (*entity.FiscalClassification, error) {
	c, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if c.Languages, err = uc.repo.ListLanguages(ctx, c.ID); err != nil {
		return nil, err
	}
	if c.ExportAttributes, err = uc.repo.ListExportAttributes(ctx, c.ID); err != nil {
		return nil, err
	}
	return c, nil
}

func (uc *FiscalClassificationUseCase) List(ctx context.Context, onlyActive bool) ([]*entity.FiscalClassification, error) {
	return uc.repo.List(ctx, onlyActive)
}

// GetIPIRate implements ports.FiscalClassificationProvider.
func (uc *FiscalClassificationUseCase) GetIPIRate(ctx context.Context, classificationCode int64) (float64, bool, error) {
	c, err := uc.repo.GetByCode(ctx, classificationCode)
	if err != nil || c == nil {
		return 0, false, nil
	}
	return c.IPIRate, true, nil
}

func (uc *FiscalClassificationUseCase) AddLanguage(ctx context.Context, dto request.AddFiscalClassificationLanguageDTO) (*entity.FiscalClassificationLanguage, error) {
	c, err := uc.repo.GetByCode(ctx, dto.ClassificationCode)
	if err != nil {
		return nil, err
	}
	return uc.repo.AddLanguage(ctx, &entity.FiscalClassificationLanguage{
		ClassificationID: c.ID,
		Language:         dto.Language,
		Description:      dto.Description,
	})
}

func (uc *FiscalClassificationUseCase) AddExportAttribute(ctx context.Context, dto request.AddFiscalClassificationExportAttributeDTO) (*entity.FiscalClassificationExportAttribute, error) {
	c, err := uc.repo.GetByCode(ctx, dto.ClassificationCode)
	if err != nil {
		return nil, err
	}
	attr := &entity.FiscalClassificationExportAttribute{
		ClassificationID: c.ID,
		Code:             dto.Code,
		Description:      dto.Description,
		Domain:           dto.Domain,
	}
	if dto.StartDate != nil {
		if t, perr := time.Parse("2006-01-02", *dto.StartDate); perr == nil {
			attr.StartDate = &t
		}
	}
	if dto.EndDate != nil {
		if t, perr := time.Parse("2006-01-02", *dto.EndDate); perr == nil {
			attr.EndDate = &t
		}
	}
	return uc.repo.AddExportAttribute(ctx, attr)
}
