package item_supplier_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository"
	"strings"
	"time"
)

type ItemSupplierUseCase struct {
	repo repository.ItemSupplierRepository
	auth ports.AuthService
}

func NewItemSupplierUseCase(repo repository.ItemSupplierRepository, auth ports.AuthService) *ItemSupplierUseCase {
	return &ItemSupplierUseCase{repo: repo, auth: auth}
}
func parseOptionalDate(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	x, err := time.Parse("2006-01-02", strings.TrimSpace(*s))
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("data %q inválida: use o formato ano-mês-dia", *s))
	}
	return &x, nil
}
func (uc *ItemSupplierUseCase) Upsert(ctx context.Context, d request.UpsertItemPreferredSupplierDTO) (*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	by, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := entity.NewItemPreferredSupplier(e, d.ItemCode, d.SupplierCode, d.Mask, d.Ranking, by)
	if err != nil {
		return nil, err
	}
	// Vazio = não informado: o repositório mantém o que a NF-e aprendeu.
	s.SupplierItemCode = textoOuNil(d.SupplierItemCode)
	s.SupplierDescription = textoOuNil(d.SupplierDescription)
	s.UOM = textoOuNil(d.UOM)
	s.XMLUOM = d.XMLUOM
	s.ConversionFactor = d.ConversionFactor
	s.PackageQuantity = d.PackageQuantity
	// ranking=1 was the legacy preferred-supplier contract; keep it compatible
	// while exposing the explicit flag required by the richer register.
	s.IsPreferred = d.IsPreferred || d.Ranking == 1
	s.ClassificationID = d.ClassificationID
	s.ClassificationGrade = d.ClassificationGrade
	s.DirectBilling = d.DirectBilling
	s.ThirdPartyOrder = d.ThirdPartyOrder
	s.IgnoreAvgCostAddition = d.IgnoreAvgCostAddition
	s.Ecommerce = d.Ecommerce
	s.Barcode = d.Barcode
	s.Notes = d.Notes
	s.LeadTimeDays = d.LeadTimeDays
	s.MinOrderQty = d.MinOrderQty
	s.OrderMultiple = d.OrderMultiple
	s.QuotaPercent = d.QuotaPercent
	if s.ValidFrom, err = parseOptionalDate(d.ValidFrom); err != nil {
		return nil, err
	}
	if s.ClassificationDate, err = parseOptionalDate(d.ClassificationDate); err != nil {
		return nil, err
	}
	if s.ValidUntil, err = parseOptionalDate(d.ValidUntil); err != nil {
		return nil, err
	}
	if s.ConversionFactor != nil {
		ok, err := uc.repo.ItemAllowsConversionFactor(ctx, s.ItemCode)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorsuc.NewValidationError("o fator de conversão só vale para itens genéricos de importação de XML")
		}
	}
	if err = s.Validate(); err != nil {
		return nil, err
	}
	saved, err := uc.repo.Upsert(ctx, s)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponse(saved), nil
}
func (uc *ItemSupplierUseCase) ListByItem(ctx context.Context, item int64) ([]*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListByItem(ctx, e, item)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponses(x), nil
}
func (uc *ItemSupplierUseCase) ListBySupplier(ctx context.Context, supplier int64) ([]*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListBySupplier(ctx, e, supplier)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponses(x), nil
}
func (uc *ItemSupplierUseCase) SearchExternal(ctx context.Context, supplier int64, term string) ([]*response.ItemPreferredSupplierResponse, error) {
	if supplier <= 0 || strings.TrimSpace(term) == "" {
		return nil, errorsuc.NewValidationError("informe o fornecedor e o termo de busca")
	}
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.SearchExternal(ctx, e, supplier, term)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponses(x), nil
}

func (uc *ItemSupplierUseCase) ResolveExternal(ctx context.Context, supplier int64, externalCode, description string) (*ports.SupplierItemResolution, error) {
	term := strings.TrimSpace(externalCode)
	strategy := "CODIGO_EXATO"
	if term == "" {
		term = strings.TrimSpace(description)
		strategy = "DESCRICAO"
	}
	if supplier <= 0 || term == "" {
		return nil, errorsuc.NewValidationError("informe o fornecedor e o código ou a descrição externa")
	}
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	matches, err := uc.repo.SearchExternal(ctx, e, supplier, term)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	if len(matches) > 1 {
		return nil, errorsuc.NewValidationError("identificacao externa ambigua para o fornecedor")
	}
	return &ports.SupplierItemResolution{LinkID: matches[0].ID, ItemCode: matches[0].ItemCode, Strategy: strategy}, nil
}
func (uc *ItemSupplierUseCase) Delete(ctx context.Context, id int64) error {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return err
	}
	return uc.repo.Delete(ctx, e, id)
}
func (uc *ItemSupplierUseCase) GetPreferredSupplier(ctx context.Context, item int64) (int64, bool, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return 0, false, err
	}
	s, err := uc.repo.GetPreferred(ctx, e, item)
	if err != nil {
		// Falha de leitura não é "item sem fornecedor".
		return 0, false, fmt.Errorf("lendo o fornecedor preferencial do item %d: %w", item, err)
	}
	if s == nil {
		return 0, false, nil
	}
	return s.SupplierCode, true, nil
}

// FonteDoItem: ver repository.FonteDoItem. Usado na geração de pedidos a
// partir de requisições, para não mandar ao banco uma linha que a trava recusa.
func (uc *ItemSupplierUseCase) FonteDoItem(ctx context.Context, item int64, mask string, informado *int64) (int64, bool, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return 0, false, err
	}
	return uc.repo.FonteDoItem(ctx, e, item, mask, informado)
}

func textoOuNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// Block / Unblock: bloquear tira o vínculo das fontes (MRP e documentos de
// compra) sem apagá-lo; o motivo fica no histórico.
func (uc *ItemSupplierUseCase) Block(ctx context.Context, id int64, d request.ItemSupplierBlockDTO) (*response.ItemPreferredSupplierResponse, error) {
	return uc.setBlocked(ctx, id, entity.AcaoBloqueio, d.Reason)
}
func (uc *ItemSupplierUseCase) Unblock(ctx context.Context, id int64, d request.ItemSupplierBlockDTO) (*response.ItemPreferredSupplierResponse, error) {
	return uc.setBlocked(ctx, id, entity.AcaoDesbloqueio, d.Reason)
}
func (uc *ItemSupplierUseCase) setBlocked(ctx context.Context, id int64, action, reason string) (*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	by, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	ev, err := entity.NewBlockEvent(e, id, action, reason, by)
	if err != nil {
		return nil, err
	}
	saved, err := uc.repo.SetBlocked(ctx, ev)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponse(saved), nil
}
func (uc *ItemSupplierUseCase) BlockHistory(ctx context.Context, id int64) ([]response.ItemSupplierBlockEventResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListBlockHistory(ctx, e, id)
	if err != nil {
		return nil, err
	}
	return toBlockEventResponses(x), nil
}

// ClearLearned desfaz o código/descrição do fornecedor aprendidos na NF-e.
func (uc *ItemSupplierUseCase) ClearLearned(ctx context.Context, id int64) (*response.ItemPreferredSupplierResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	saved, err := uc.repo.ClearLearned(ctx, e, id)
	if err != nil {
		return nil, err
	}
	return toItemPreferredSupplierResponse(saved), nil
}
func (uc *ItemSupplierUseCase) CreateQualityReport(ctx context.Context, link int64, d request.CreateItemSupplierQualityReportDTO) (*response.ItemSupplierQualityReportResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	by, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	on, err := time.Parse("2006-01-02", d.RegisteredOn)
	if err != nil {
		return nil, errorsuc.NewValidationError("informe a data de registro no formato ano-mês-dia")
	}
	q, err := entity.NewQualityReport(e, link, on, d.Status, by)
	if err != nil {
		return nil, err
	}
	q.FileName = d.FileName
	q.ContentType = d.ContentType
	q.Content = d.Content
	if len(q.Content) > 10*1024*1024 {
		return nil, errorsuc.NewValidationError("o anexo do laudo passa de 10 MiB")
	}
	q.Notes = d.Notes
	saved, err := uc.repo.CreateQualityReport(ctx, q)
	if err != nil {
		return nil, err
	}
	return toQualityResponse(saved), nil
}
func (uc *ItemSupplierUseCase) ListQualityReports(ctx context.Context, link int64) ([]response.ItemSupplierQualityReportResponse, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	x, err := uc.repo.ListQualityReports(ctx, e, link)
	if err != nil {
		return nil, err
	}
	out := make([]response.ItemSupplierQualityReportResponse, 0, len(x))
	for _, q := range x {
		out = append(out, *toQualityResponse(q))
	}
	return out, nil
}

func (uc *ItemSupplierUseCase) GetQualityReport(ctx context.Context, id int64) (*entity.QualityReport, error) {
	e, err := uc.auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.repo.GetQualityReport(ctx, e, id)
}
