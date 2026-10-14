package fiscal_classification_uc

import (
	"context"
	"fmt"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"
)

type FiscalClassificationUseCase struct {
	repo repository.FiscalClassificationRepository
	auth ports.AuthService
}

func NewFiscalClassificationUseCase(repo repository.FiscalClassificationRepository, auth ports.AuthService) *FiscalClassificationUseCase {
	return &FiscalClassificationUseCase{repo: repo, auth: auth}
}

func applyFields(c *entity.FiscalClassification, f request.FiscalClassificationFields) {
	c.Description = f.Description
	c.ItemCode = f.ItemCode
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
	c.ValidFrom = f.ValidFrom.Ponteiro()
	c.ValidUntil = f.ValidUntil.Ponteiro()
	c.DefaultOrigin = f.DefaultOrigin
	c.DefaultICMSRate = f.DefaultICMSRate
	c.DefaultCalculatePISCOFINS = f.DefaultCalculatePISCOFINS
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

func (uc *FiscalClassificationUseCase) Create(ctx context.Context, dto request.CreateFiscalClassificationDTO) (*response.FiscalClassificationResponse, error) {
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, fmt.Errorf("empresa autenticada obrigatoria: %w", err)
	}
	code, err := uc.repo.NextCode(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	c, err := entity.NewFiscalClassification(code, dto.Description, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	applyFields(c, dto.FiscalClassificationFields)
	c.EnterpriseID = enterpriseID
	created, err := uc.repo.Create(ctx, c)
	if err != nil {
		return nil, err
	}
	return toFiscalClassificationResponse(created), nil
}

func (uc *FiscalClassificationUseCase) Update(ctx context.Context, dto request.UpdateFiscalClassificationDTO) (*response.FiscalClassificationResponse, error) {
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, fmt.Errorf("empresa autenticada obrigatoria: %w", err)
	}
	c, err := uc.repo.GetByCode(ctx, enterpriseID, dto.Code)
	if err != nil {
		return nil, err
	}
	applyFields(c, dto.FiscalClassificationFields)
	c.IsActive = ptrutil.BoolOr(dto.IsActive, c.IsActive)
	updated, err := uc.repo.Update(ctx, c)
	if err != nil {
		return nil, err
	}
	return toFiscalClassificationResponse(updated), nil
}

func (uc *FiscalClassificationUseCase) Get(ctx context.Context, code int64) (*response.FiscalClassificationResponse, error) {
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, fmt.Errorf("empresa autenticada obrigatoria: %w", err)
	}
	c, err := uc.repo.GetByCode(ctx, enterpriseID, code)
	if err != nil {
		return nil, err
	}
	if c.Languages, err = uc.repo.ListLanguages(ctx, c.ID); err != nil {
		return nil, err
	}
	if c.ExportAttributes, err = uc.repo.ListExportAttributes(ctx, c.ID); err != nil {
		return nil, err
	}
	return toFiscalClassificationResponse(c), nil
}

func (uc *FiscalClassificationUseCase) List(ctx context.Context, onlyActive bool) ([]*response.FiscalClassificationResponse, error) {
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, fmt.Errorf("empresa autenticada obrigatoria: %w", err)
	}
	list, err := uc.repo.List(ctx, enterpriseID, onlyActive)
	if err != nil {
		return nil, err
	}
	return toFiscalClassificationResponses(list), nil
}

// GetIPIRate implements ports.FiscalClassificationProvider.
func (uc *FiscalClassificationUseCase) GetIPIRate(ctx context.Context, classificationCode int64) (float64, bool, error) {
	enterpriseID, authErr := uc.auth.EnterpriseID(ctx)
	if authErr != nil {
		return 0, false, authErr
	}
	c, err := uc.repo.GetByCode(ctx, enterpriseID, classificationCode)
	if err != nil || c == nil {
		return 0, false, nil
	}
	return c.IPIRate, true, nil
}

func (uc *FiscalClassificationUseCase) AddLanguage(ctx context.Context, dto request.AddFiscalClassificationLanguageDTO) (*response.FiscalClassificationLanguageResponse, error) {
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := uc.repo.GetByCode(ctx, enterpriseID, dto.ClassificationCode)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.AddLanguage(ctx, &entity.FiscalClassificationLanguage{
		ClassificationID: c.ID,
		Language:         dto.Language,
		Description:      dto.Description,
	})
	if err != nil {
		return nil, err
	}
	return toFiscalClassificationLanguageResponse(created), nil
}

func (uc *FiscalClassificationUseCase) AddExportAttribute(ctx context.Context, dto request.AddFiscalClassificationExportAttributeDTO) (*response.FiscalClassificationExportAttributeResponse, error) {
	enterpriseID, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := uc.repo.GetByCode(ctx, enterpriseID, dto.ClassificationCode)
	if err != nil {
		return nil, err
	}
	attr := &entity.FiscalClassificationExportAttribute{
		ClassificationID: c.ID,
		Code:             dto.Code,
		Description:      dto.Description,
		Domain:           dto.Domain,
	}
	// Data mal digitada era descartada em silêncio: o atributo ficava sem
	// vigência e a pessoa achava que tinha gravado.
	var err2 error
	if attr.StartDate, err2 = dataOpcional(dto.StartDate, "início da vigência"); err2 != nil {
		return nil, err2
	}
	if attr.EndDate, err2 = dataOpcional(dto.EndDate, "fim da vigência"); err2 != nil {
		return nil, err2
	}
	if attr.StartDate != nil && attr.EndDate != nil && attr.EndDate.Before(*attr.StartDate) {
		return nil, errorsuc.NewValidationError("o fim da vigência não pode ser antes do início")
	}
	created, err := uc.repo.AddExportAttribute(ctx, attr)
	if err != nil {
		return nil, err
	}
	return toFiscalClassificationExportAttributeResponse(created), nil
}

// dataOpcional lê uma data AAAA-MM-DD opcional; vazia é "sem data".
func dataOpcional(v *string, campo string) (*time.Time, error) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*v))
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválida: use o formato AAAA-MM-DD", campo))
	}
	return &t, nil
}
